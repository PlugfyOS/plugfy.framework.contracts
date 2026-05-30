package idempotency

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemStore_GetMissing(t *testing.T) {
	s := NewMemStore()
	defer s.Close()

	resp, ok, err := s.Get(context.Background(), "absent")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if ok || resp != nil {
		t.Errorf("Get on missing key: ok=%v resp=%v", ok, resp)
	}
}

func TestMemStore_SetGetRoundtrip(t *testing.T) {
	s := NewMemStore()
	defer s.Close()
	ctx := context.Background()

	want := &CachedResponse{
		Status:  201,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"id":"x"}`),
	}
	if err := s.Set(ctx, "k1", want, time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, err := s.Get(ctx, "k1")
	if err != nil || !ok {
		t.Fatalf("Get k1: ok=%v err=%v", ok, err)
	}
	if got.Status != want.Status || string(got.Body) != string(want.Body) {
		t.Errorf("round-trip mismatch: got=%+v want=%+v", got, want)
	}
}

func TestMemStore_Overwrite(t *testing.T) {
	s := NewMemStore()
	defer s.Close()
	ctx := context.Background()

	_ = s.Set(ctx, "k", &CachedResponse{Status: 200, Body: []byte("first")}, time.Hour)
	_ = s.Set(ctx, "k", &CachedResponse{Status: 201, Body: []byte("second")}, time.Hour)
	got, _, _ := s.Get(ctx, "k")
	if string(got.Body) != "second" {
		t.Errorf("Overwrite failed: got %q", got.Body)
	}
}

func TestMemStore_Expiry(t *testing.T) {
	s := NewMemStore()
	defer s.Close()
	// Inject a clock so we can fast-forward.
	t0 := time.Now()
	s.now = func() time.Time { return t0 }

	ctx := context.Background()
	_ = s.Set(ctx, "expires", &CachedResponse{Status: 200}, 100*time.Millisecond)

	// Before expiry
	if _, ok, _ := s.Get(ctx, "expires"); !ok {
		t.Fatalf("entry should be present before expiry")
	}

	// Fast-forward past expiry
	s.now = func() time.Time { return t0.Add(time.Second) }
	if _, ok, _ := s.Get(ctx, "expires"); ok {
		t.Errorf("entry should have expired")
	}
}

func TestMemStore_Delete(t *testing.T) {
	s := NewMemStore()
	defer s.Close()
	ctx := context.Background()

	_ = s.Set(ctx, "k", &CachedResponse{Status: 200}, time.Hour)
	if err := s.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Errorf("entry should be gone after Delete")
	}
	// Idempotent: deleting again is a no-op
	if err := s.Delete(ctx, "k"); err != nil {
		t.Errorf("second Delete returned %v, want nil", err)
	}
}

func TestMemStore_DefaultTTL(t *testing.T) {
	s := NewMemStore()
	defer s.Close()
	t0 := time.Now()
	s.now = func() time.Time { return t0 }

	// ttl <= 0 should fall back to 24h
	_ = s.Set(context.Background(), "k", &CachedResponse{Status: 200}, 0)

	s.now = func() time.Time { return t0.Add(23 * time.Hour) }
	if _, ok, _ := s.Get(context.Background(), "k"); !ok {
		t.Errorf("entry with default TTL should survive 23h")
	}
	s.now = func() time.Time { return t0.Add(25 * time.Hour) }
	if _, ok, _ := s.Get(context.Background(), "k"); ok {
		t.Errorf("entry with default TTL should expire after 24h")
	}
}

func TestMemStore_ConcurrentAccess(t *testing.T) {
	s := NewMemStore()
	defer s.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(50)
	for i := 0; i < 50; i++ {
		i := i
		go func() {
			defer wg.Done()
			key := "concurrent"
			_ = s.Set(ctx, key, &CachedResponse{Status: 200 + i, Body: []byte("payload")}, time.Hour)
			_, _, _ = s.Get(ctx, key)
		}()
	}
	wg.Wait()
	if _, ok, _ := s.Get(ctx, "concurrent"); !ok {
		t.Errorf("entry should be present after concurrent writers")
	}
}

func TestMemStore_PurgeExpired(t *testing.T) {
	s := NewMemStore()
	defer s.Close()
	t0 := time.Now()
	s.now = func() time.Time { return t0 }
	ctx := context.Background()

	_ = s.Set(ctx, "alive", &CachedResponse{Status: 200}, time.Hour)
	_ = s.Set(ctx, "dead", &CachedResponse{Status: 200}, time.Millisecond)

	s.now = func() time.Time { return t0.Add(time.Hour / 2) }
	s.purgeExpired()

	if got := s.Len(); got != 1 {
		t.Errorf("after purge: Len=%d want 1", got)
	}
}
