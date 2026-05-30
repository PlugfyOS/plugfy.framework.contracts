package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBreakerOpensAndHalfOpens(t *testing.T) {
	now := time.Unix(0, 0)
	b := NewBreaker(3, 2, time.Minute)
	b.now = func() time.Time { return now }

	fail := func() error { return errors.New("boom") }
	// 3 consecutive failures → open.
	for i := 0; i < 3; i++ {
		_ = b.Do(fail)
	}
	if b.State() != StateOpen {
		t.Fatalf("expected open, got %s", b.State())
	}
	if err := b.Do(func() error { return nil }); err != ErrOpen {
		t.Fatalf("open breaker should reject with ErrOpen, got %v", err)
	}

	// After reset timeout → half-open; a success doesn't fully close yet (needs 2).
	now = now.Add(time.Minute)
	if b.State() != StateHalfOpen {
		t.Fatalf("expected half-open after reset, got %s", b.State())
	}
	_ = b.Do(func() error { return nil })
	if b.State() != StateHalfOpen {
		t.Fatalf("one success should keep half-open, got %s", b.State())
	}
	_ = b.Do(func() error { return nil })
	if b.State() != StateClosed {
		t.Fatalf("two successes should close, got %s", b.State())
	}
}

func TestBreakerHalfOpenFailureReopens(t *testing.T) {
	now := time.Unix(0, 0)
	b := NewBreaker(1, 1, time.Minute)
	b.now = func() time.Time { return now }
	_ = b.Do(func() error { return errors.New("x") }) // open
	now = now.Add(time.Minute)                        // half-open
	_ = b.Do(func() error { return errors.New("x") }) // fail in half-open → reopen
	if b.State() != StateOpen {
		t.Fatalf("half-open failure should reopen, got %s", b.State())
	}
}

func TestRetryEventuallySucceeds(t *testing.T) {
	calls := 0
	p := RetryPolicy{MaxAttempts: 3, sleep: func(context.Context, time.Duration) error { return nil }}
	err := p.Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("err=%v calls=%d, want nil/3", err, calls)
	}
}

func TestRetryRespectsRetryablePredicate(t *testing.T) {
	calls := 0
	fatal := errors.New("fatal")
	p := RetryPolicy{MaxAttempts: 5, Retryable: func(e error) bool { return !errors.Is(e, fatal) },
		sleep: func(context.Context, time.Duration) error { return nil }}
	err := p.Do(context.Background(), func() error { calls++; return fatal })
	if calls != 1 || !errors.Is(err, fatal) {
		t.Fatalf("non-retryable should stop at 1 call, got calls=%d err=%v", calls, err)
	}
}

func TestBulkheadLimitsConcurrency(t *testing.T) {
	bh := NewBulkhead(1)
	// First call occupies the only slot; a second with a cancelled context fails.
	release := make(chan struct{})
	go func() { _ = bh.Do(context.Background(), func() error { <-release; return nil }) }()
	time.Sleep(10 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := bh.Do(ctx, func() error { return nil }); err == nil {
		t.Fatal("expected bulkhead to reject when full and ctx cancelled")
	}
	close(release)
}

func TestGuardComposes(t *testing.T) {
	b := NewBreaker(2, 1, time.Minute)
	g := Guard{
		Bulkhead: NewBulkhead(4),
		Retry:    &RetryPolicy{MaxAttempts: 2, sleep: func(context.Context, time.Duration) error { return nil }},
		Breaker:  b,
	}
	calls := 0
	err := g.Do(context.Background(), func() error {
		calls++
		if calls < 2 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("guard should retry to success: err=%v calls=%d", err, calls)
	}
}
