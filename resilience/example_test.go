package resilience_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PlugfyOS/plugfy-common/resilience"
)

// ExampleGuard shows how a connector composes a bulkhead, a retry policy and a
// circuit breaker into a single Guard and wraps a remote call with it. A flaky
// dependency that fails twice and then succeeds is recovered by the retry, while
// the breaker and bulkhead bound the blast radius of a sustained outage.
func ExampleGuard() {
	g := resilience.Guard{
		Bulkhead: resilience.NewBulkhead(8), // at most 8 concurrent calls
		Retry: &resilience.RetryPolicy{
			MaxAttempts: 3,
			Base:        time.Millisecond,
			Max:         10 * time.Millisecond,
		},
		Breaker: resilience.NewBreaker(5, 1, time.Second), // open after 5 failures
	}

	attempts := 0
	err := g.Do(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("upstream temporarily unavailable")
		}
		return nil // third attempt succeeds
	})

	fmt.Printf("attempts=%d err=%v\n", attempts, err)
	// Output: attempts=3 err=<nil>
}
