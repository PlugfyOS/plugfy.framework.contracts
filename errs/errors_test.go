package errs

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestClassHTTPStatus(t *testing.T) {
	cases := map[Class]int{
		ClassValidation:   http.StatusBadRequest,
		ClassUnauthorized: http.StatusUnauthorized,
		ClassForbidden:    http.StatusForbidden,
		ClassNotFound:     http.StatusNotFound,
		ClassConflict:     http.StatusConflict,
		ClassRateLimit:    http.StatusTooManyRequests,
		ClassUpstream:     http.StatusBadGateway,
		ClassTimeout:      http.StatusGatewayTimeout,
		ClassInternal:     http.StatusInternalServerError,
		Class("garbage"):  http.StatusInternalServerError,
	}
	for c, want := range cases {
		if got := c.HTTPStatus(); got != want {
			t.Errorf("Class(%q).HTTPStatus() = %d, want %d", c, got, want)
		}
	}
}

func TestNew(t *testing.T) {
	e := New(ClassNotFound, CodeNotFoundEntity, "user not found")
	if e == nil {
		t.Fatal("New returned nil")
	}
	if e.Class != ClassNotFound {
		t.Errorf("Class = %q, want %q", e.Class, ClassNotFound)
	}
	if e.Code != CodeNotFoundEntity {
		t.Errorf("Code = %q, want %q", e.Code, CodeNotFoundEntity)
	}
	if e.HTTPStatus() != http.StatusNotFound {
		t.Errorf("HTTPStatus = %d, want %d", e.HTTPStatus(), http.StatusNotFound)
	}
}

func TestErrorFormat(t *testing.T) {
	e := New(ClassValidation, CodeValidationFieldRequired, "email is required")
	want := "validation [validation.field_required]: email is required"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	wrapped := Wrap(fmt.Errorf("dial tcp: connection refused"),
		ClassUpstream, CodeUpstreamUnavailable, "openai unreachable")
	if !contains(wrapped.Error(), "connection refused") {
		t.Errorf("wrapped error should include cause; got %q", wrapped.Error())
	}
	if !contains(wrapped.Error(), "openai unreachable") {
		t.Errorf("wrapped error should include message; got %q", wrapped.Error())
	}
}

func TestWrapNil(t *testing.T) {
	if got := Wrap(nil, ClassInternal, "x", "y"); got != nil {
		t.Errorf("Wrap(nil) = %v, want nil", got)
	}
}

func TestWrapAlreadyError(t *testing.T) {
	original := New(ClassNotFound, "x.y", "original")
	again := Wrap(original, ClassInternal, "z.w", "should not re-classify")
	if again != original {
		t.Errorf("Wrap of an *Error should return it unchanged; got %v", again)
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	e := Wrap(cause, ClassUpstream, CodeUpstreamUnavailable, "wrapping")
	if got := errors.Unwrap(e); got != cause {
		t.Errorf("Unwrap returned %v, want %v", got, cause)
	}
}

func TestWithDetail(t *testing.T) {
	e := New(ClassValidation, CodeValidationFieldFormat, "bad email").
		WithDetail("field", "email").
		WithDetail("rule", "rfc5322")
	if e.Details["field"] != "email" || e.Details["rule"] != "rfc5322" {
		t.Errorf("WithDetail did not set keys correctly: %v", e.Details)
	}
}

func TestWithDetails(t *testing.T) {
	e := New(ClassConflict, CodeConflictDuplicate, "name taken").
		WithDetails(map[string]any{"name": "alice", "tenant": "acme"})
	if e.Details["name"] != "alice" || e.Details["tenant"] != "acme" {
		t.Errorf("WithDetails did not merge: %v", e.Details)
	}
}

func TestWithDetailNil(t *testing.T) {
	var e *Error
	if got := e.WithDetail("k", "v"); got != nil {
		t.Errorf("WithDetail on nil should return nil; got %v", got)
	}
}

func TestClassify(t *testing.T) {
	if got := Classify(nil); got != ClassInternal {
		t.Errorf("Classify(nil) = %q, want %q", got, ClassInternal)
	}
	if got := Classify(errors.New("plain")); got != ClassInternal {
		t.Errorf("Classify(plain) = %q, want %q", got, ClassInternal)
	}
	e := New(ClassForbidden, CodeForbiddenScope, "no")
	if got := Classify(e); got != ClassForbidden {
		t.Errorf("Classify(*Error) = %q, want %q", got, ClassForbidden)
	}
	// wrapped chain
	chain := fmt.Errorf("outer: %w", e)
	if got := Classify(chain); got != ClassForbidden {
		t.Errorf("Classify(chain) = %q, want %q", got, ClassForbidden)
	}
}

func TestIsClass(t *testing.T) {
	e := New(ClassRateLimit, CodeRateLimitExceeded, "slow down")
	if !IsClass(e, ClassRateLimit) {
		t.Errorf("IsClass should match")
	}
	if IsClass(e, ClassValidation) {
		t.Errorf("IsClass should not match wrong class")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
