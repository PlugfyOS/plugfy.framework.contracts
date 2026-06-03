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

func TestRetryDefaultBackoffUnchangedByJitterField(t *testing.T) {
	// Jitter defaults to 0 -> the backoff sequence is EXACTLY the historical
	// deterministic delay*=2 (capped by Max), proving the new field is additive
	// and does not alter existing callers.
	var slept []time.Duration
	p := RetryPolicy{
		MaxAttempts: 5,
		Base:        10 * time.Millisecond,
		Max:         50 * time.Millisecond,
		sleep:       func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil },
	}
	_ = p.Do(context.Background(), func() error { return errors.New("always") })
	// 5 attempts -> 4 sleeps: 10, 20, 40, then capped at 50.
	want := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond, 50 * time.Millisecond}
	if len(slept) != len(want) {
		t.Fatalf("sleep count = %d, want %d (%v)", len(slept), len(want), slept)
	}
	for i := range want {
		if slept[i] != want[i] {
			t.Fatalf("sleep[%d] = %v, want %v (full sequence %v)", i, slept[i], want[i], slept)
		}
	}
}

func TestRetryMultiplier(t *testing.T) {
	// Multiplier is additive: 0 reproduces the historical *2 sequence; >0 replaces
	// the doubling (3 triples each delay), both still capped by Max.
	collect := func(p RetryPolicy) []time.Duration {
		var slept []time.Duration
		p.sleep = func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil }
		_ = p.Do(context.Background(), func() error { return errors.New("always") })
		return slept
	}

	// Multiplier == 0 -> EXACTLY today's *2 backoff (no Max), proving default is unchanged.
	got := collect(RetryPolicy{MaxAttempts: 4, Base: 10 * time.Millisecond})
	wantDouble := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond}
	if len(got) != len(wantDouble) {
		t.Fatalf("Multiplier=0 sleep count = %d, want %d (%v)", len(got), len(wantDouble), got)
	}
	for i := range wantDouble {
		if got[i] != wantDouble[i] {
			t.Fatalf("Multiplier=0 sleep[%d] = %v, want %v (full %v)", i, got[i], wantDouble[i], got)
		}
	}

	// Multiplier == 3 -> each delay triples (10, 30, 90), still capped by Max=80.
	got = collect(RetryPolicy{MaxAttempts: 4, Base: 10 * time.Millisecond, Max: 80 * time.Millisecond, Multiplier: 3})
	wantTriple := []time.Duration{10 * time.Millisecond, 30 * time.Millisecond, 80 * time.Millisecond}
	if len(got) != len(wantTriple) {
		t.Fatalf("Multiplier=3 sleep count = %d, want %d (%v)", len(got), len(wantTriple), got)
	}
	for i := range wantTriple {
		if got[i] != wantTriple[i] {
			t.Fatalf("Multiplier=3 sleep[%d] = %v, want %v (full %v)", i, got[i], wantTriple[i], got)
		}
	}

	// Jitter still composes with a non-2 Multiplier: each sleep stays in [d*(1-Jitter), d]
	// for the tripled deterministic delays. rng injected for determinism.
	var slept []time.Duration
	idx := 0
	rngVals := []float64{0.0, 1.0, 0.5}
	p := RetryPolicy{
		MaxAttempts: 4,
		Base:        100 * time.Millisecond,
		Multiplier:  3,
		Jitter:      0.5,
		sleep:       func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil },
		rng:         func() float64 { v := rngVals[idx%len(rngVals)]; idx++; return v },
	}
	_ = p.Do(context.Background(), func() error { return errors.New("always") })
	delays := []time.Duration{100 * time.Millisecond, 300 * time.Millisecond, 900 * time.Millisecond}
	if len(slept) != len(delays) {
		t.Fatalf("Multiplier+Jitter sleep count = %d, want %d (%v)", len(slept), len(delays), slept)
	}
	for i, d := range delays {
		lo := time.Duration(float64(d) * 0.5) // d*(1-Jitter)
		if slept[i] < lo || slept[i] > d {
			t.Fatalf("Multiplier+Jitter sleep[%d]=%v outside band [%v,%v]", i, slept[i], lo, d)
		}
	}
}

func TestRetryJitterStaysWithinBand(t *testing.T) {
	// With Jitter>0 every sleep must fall in [d*(1-Jitter), d] for the
	// corresponding deterministic delay d. rng is injected for determinism.
	rngVals := []float64{0.0, 1.0, 0.5} // min of band, top of band, midpoint
	idx := 0
	var slept []time.Duration
	p := RetryPolicy{
		MaxAttempts: 4,
		Base:        100 * time.Millisecond,
		Jitter:      0.5,
		sleep:       func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil },
		rng:         func() float64 { v := rngVals[idx%len(rngVals)]; idx++; return v },
	}
	_ = p.Do(context.Background(), func() error { return errors.New("always") })
	delays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	if len(slept) != len(delays) {
		t.Fatalf("sleep count = %d, want %d", len(slept), len(delays))
	}
	for i, d := range delays {
		lo := time.Duration(float64(d) * 0.5) // d*(1-Jitter)
		if slept[i] < lo || slept[i] > d {
			t.Fatalf("sleep[%d]=%v outside band [%v,%v]", i, slept[i], lo, d)
		}
	}
	// rng=0.0 -> floor of band; rng=1.0 -> exactly d.
	if slept[0] != 50*time.Millisecond {
		t.Fatalf("rng=0 should yield band floor 50ms, got %v", slept[0])
	}
	if slept[1] != 200*time.Millisecond {
		t.Fatalf("rng=1 should yield full delay 200ms, got %v", slept[1])
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
