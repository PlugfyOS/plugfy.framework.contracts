package idempotency

import (
	"context"
	"sync"
	"time"
)

// MemStore is the in-memory [Store] implementation. Suitable for the
// Local edition (single binary) and as a fast default in tests.
//
// Eviction strategy:
//   - Every operation that touches the map opportunistically removes
//     a small number of expired entries (amortized O(1)).
//   - A background sweeper running every minute purges the rest.
//   - The store has no hard size cap; high-volume deployments should
//     plug in the Postgres-backed Store instead.
type MemStore struct {
	mu      sync.RWMutex
	now     func() time.Time // injectable for tests
	entries map[string]memEntry
	stop    chan struct{}
}

type memEntry struct {
	resp      *CachedResponse
	expiresAt time.Time
}

// NewMemStore returns a ready-to-use in-memory store. Call Close (or
// rely on the parent context cancellation in your composition root) to
// stop the background sweeper.
func NewMemStore() *MemStore {
	s := &MemStore{
		now:     time.Now,
		entries: make(map[string]memEntry, 256),
		stop:    make(chan struct{}),
	}
	go s.sweep()
	return s
}

// Close stops the background sweeper. Safe to call multiple times.
func (s *MemStore) Close() {
	select {
	case <-s.stop:
		// already closed
	default:
		close(s.stop)
	}
}

// Get implements [Store].
func (s *MemStore) Get(_ context.Context, key string) (*CachedResponse, bool, error) {
	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if s.now().After(e.expiresAt) {
		// Lazy eviction: writer-removes on race.
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return nil, false, nil
	}
	return e.resp, true, nil
}

// Set implements [Store].
func (s *MemStore) Set(_ context.Context, key string, resp *CachedResponse, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	s.mu.Lock()
	s.entries[key] = memEntry{resp: resp, expiresAt: s.now().Add(ttl)}
	s.mu.Unlock()
	return nil
}

// Delete implements [Store].
func (s *MemStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.entries, key)
	s.mu.Unlock()
	return nil
}

// Len returns the current entry count. Exposed for metrics/tests.
func (s *MemStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// sweep removes expired entries every minute. A goroutine started by
// [NewMemStore].
func (s *MemStore) sweep() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.purgeExpired()
		}
	}
}

func (s *MemStore) purgeExpired() {
	now := s.now()
	s.mu.Lock()
	for k, e := range s.entries {
		if now.After(e.expiresAt) {
			delete(s.entries, k)
		}
	}
	s.mu.Unlock()
}
