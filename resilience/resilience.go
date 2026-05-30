// Package resilience provides dependency-free resilience primitives for outbound
// integrations (ROADMAP F3.2, docs/modular/06): a circuit breaker, a retry
// policy with exponential backoff, and a bulkhead (bounded concurrency). They
// compose into a single Guard that connectors wrap around remote calls so a
// failing dependency degrades gracefully instead of cascading.
package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned when the circuit breaker is open.
var ErrOpen = errors.New("resilience: circuit breaker open")

// State is a circuit-breaker state.
type State int

const (
	StateClosed   State = iota // calls pass; failures counted
	StateOpen                  // calls rejected until resetTimeout elapses
	StateHalfOpen              // a limited number of trial calls allowed
)

func (s State) String() string {
	switch s {
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

// Breaker is a circuit breaker. It opens after failureThreshold consecutive
// failures, rejects calls for resetTimeout, then half-opens to trial calls;
// successThreshold consecutive successes in half-open close it again.
type Breaker struct {
	mu               sync.Mutex
	failureThreshold int
	successThreshold int
	resetTimeout     time.Duration
	now              func() time.Time

	state     State
	failures  int
	successes int
	openedAt  time.Time
}

// NewBreaker builds a breaker. Zero thresholds default to 5 failures / 1 success.
func NewBreaker(failureThreshold, successThreshold int, resetTimeout time.Duration) *Breaker {
	if failureThreshold <= 0 {
		failureThreshold = 5
	}
	if successThreshold <= 0 {
		successThreshold = 1
	}
	if resetTimeout <= 0 {
		resetTimeout = 30 * time.Second
	}
	return &Breaker{failureThreshold: failureThreshold, successThreshold: successThreshold, resetTimeout: resetTimeout, now: time.Now}
}

// State returns the current state, transitioning Open→HalfOpen when the reset
// timeout has elapsed.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maybeHalfOpenLocked()
	return b.state
}

func (b *Breaker) maybeHalfOpenLocked() {
	if b.state == StateOpen && b.now().Sub(b.openedAt) >= b.resetTimeout {
		b.state = StateHalfOpen
		b.successes = 0
	}
}

// Allow reports whether a call may proceed now.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maybeHalfOpenLocked()
	return b.state != StateOpen
}

// Record feeds a call outcome back into the breaker.
func (b *Breaker) Record(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if success {
		b.failures = 0
		if b.state == StateHalfOpen {
			b.successes++
			if b.successes >= b.successThreshold {
				b.state = StateClosed
				b.successes = 0
			}
		}
		return
	}
	// failure
	b.successes = 0
	if b.state == StateHalfOpen {
		b.trip()
		return
	}
	b.failures++
	if b.failures >= b.failureThreshold {
		b.trip()
	}
}

func (b *Breaker) trip() {
	b.state = StateOpen
	b.openedAt = b.now()
	b.failures = 0
}

// Do runs fn if the breaker allows it, recording the outcome. A nil error is a
// success; any error (and the rejection ErrOpen) counts as appropriate.
func (b *Breaker) Do(fn func() error) error {
	if !b.Allow() {
		return ErrOpen
	}
	err := fn()
	b.Record(err == nil)
	return err
}

// RetryPolicy retries a call with capped exponential backoff. Attempts<=1 means
// no retry. Backoff respects context cancellation.
type RetryPolicy struct {
	MaxAttempts int
	Base        time.Duration
	Max         time.Duration
	// Retryable decides whether an error is worth retrying; nil retries all.
	Retryable func(error) bool
	sleep     func(context.Context, time.Duration) error // injectable for tests
}

// Do executes fn up to MaxAttempts times, backing off between tries.
func (p RetryPolicy) Do(ctx context.Context, fn func() error) error {
	attempts := p.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	sleep := p.sleep
	if sleep == nil {
		sleep = sleepCtx
	}
	var err error
	delay := p.Base
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if p.Retryable != nil && !p.Retryable(err) {
			return err
		}
		if i == attempts-1 {
			break
		}
		if delay > 0 {
			if serr := sleep(ctx, delay); serr != nil {
				return serr
			}
		}
		delay *= 2
		if p.Max > 0 && delay > p.Max {
			delay = p.Max
		}
	}
	return err
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Bulkhead bounds concurrent in-flight calls to isolate a slow dependency from
// exhausting the whole process.
type Bulkhead struct{ sem chan struct{} }

// NewBulkhead builds a bulkhead allowing max concurrent calls (<=0 → 1).
func NewBulkhead(max int) *Bulkhead {
	if max <= 0 {
		max = 1
	}
	return &Bulkhead{sem: make(chan struct{}, max)}
}

// Do acquires a slot (or returns ctx.Err() if it cannot before cancellation),
// runs fn, and releases the slot.
func (b *Bulkhead) Do(ctx context.Context, fn func() error) error {
	select {
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Guard composes a bulkhead, retry policy and breaker around a call: bulkhead
// (admission) → retry (each attempt through the breaker). A nil component is
// skipped, so a Guard can use any subset.
type Guard struct {
	Bulkhead *Bulkhead
	Retry    *RetryPolicy
	Breaker  *Breaker
}

// Do runs fn under the configured protections.
func (g Guard) Do(ctx context.Context, fn func() error) error {
	call := fn
	if g.Breaker != nil {
		inner := fn
		call = func() error { return g.Breaker.Do(inner) }
	}
	retried := func() error {
		if g.Retry != nil {
			return g.Retry.Do(ctx, call)
		}
		return call()
	}
	if g.Bulkhead != nil {
		return g.Bulkhead.Do(ctx, retried)
	}
	return retried()
}
