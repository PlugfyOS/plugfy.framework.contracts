package core

import "time"

// RetryPolicy is the DECLARED retry policy a MethodDef (or the unit-wide
// DefaultPolicy) surfaces. core.Runner maps it onto the canonical
// resilience.Guard (bulkhead -> retry -> breaker) and resilience.RetryPolicy —
// it is a declarative mirror, not a second engine. Unlike the imperative
// resilience.RetryPolicy (whose Retryable is a func(error) bool), this one is
// pure data: Retryable is a list of error CLASSES the wrapper matches against
// UnitError.Class.
type RetryPolicy struct {
	MaxAttempts int
	Base, Max   time.Duration
	Multiplier  float64         // backoff growth factor; 0 => resilience default *2, else honored verbatim
	Jitter      float64         // 0..1 fraction of the delay randomized (maps to resilience.RetryPolicy.Jitter)
	Retryable   []string        // error CLASSES to retry: transient|timeout|… (empty + Idempotent => default set)
	Breaker     *BreakerPolicy  // optional circuit breaker (maps to resilience.Breaker)
	Bulkhead    *BulkheadPolicy // optional concurrency cap (maps to resilience.Bulkhead)
}

// BreakerPolicy declares a circuit breaker (maps to resilience.Breaker).
type BreakerPolicy struct {
	FailureThreshold, SuccessThreshold int
	Reset                              time.Duration
}

// BulkheadPolicy declares a concurrency cap (maps to resilience.Bulkhead).
type BulkheadPolicy struct{ Max int }

// Result is the uniform outcome the wrapper materializes for the engine from a
// brick's map[string]any return, threading attempts/progress into the StepFrame.
type Result struct {
	Out      map[string]any // validated against MethodDef.Returns by the wrapper
	Attempts int            // threaded from the retry loop into StepFrame.Attempt
	Progress Progress       // last reported
}

// UnitError is the classifiable error so DECLARED Retryable + the onError/
// onTimeout/onCancel typed edges work UNIFORMLY on the in-proc and the spawned
// path. ErrorClass() satisfies the engine's existing classer interface verbatim;
// Code unifies supervisor.v1's function_not_found vs generic error split.
type UnitError struct {
	Code    string // "method_not_found" | "invalid_param" | "unauthorized" | domain code
	Class   string // "transient" | "timeout" | "cancel" | "permanent" — feeds RetryPolicy.Retryable
	Message string
	Cause   error
}

func (e *UnitError) Error() string      { return e.Message }
func (e *UnitError) Unwrap() error      { return e.Cause }
func (e *UnitError) ErrorClass() string { return e.Class } // == the engine's errclass classer

// ErrMethodNotFound builds the typed, classifiable error returned when a Unit is
// invoked with a method name it does not declare.
func ErrMethodNotFound(m string) *UnitError {
	return &UnitError{Code: "method_not_found", Class: "permanent", Message: "no such method: " + m}
}
