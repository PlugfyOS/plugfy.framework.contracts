// Package errs defines the canonical error model for the Plugfy
// platform: a small, stable set of error classes mapped to HTTP statuses,
// with structured details and trace correlation. All API handlers and
// platform services SHOULD return errors that can be classified through
// [As] or constructed directly via [New] / [Wrap].
//
// Design constraints
//
//   - Classes are tied to HTTP status families (RFC 7807-compatible).
//   - Codes are stable, reverse-DNS-flavored strings ("auth.token_expired").
//   - Errors carry an optional Details map for machine consumption and a
//     wrapped underlying cause for log/trace purposes.
//   - Construction is allocation-friendly: most paths take a sentinel
//     class + code + message, no fmt.Errorf chains required.
//
// The platform API host (platform-api) renders this model into the wire error
// envelope so every handler responds uniformly; the platform-pipeline engine
// uses the [Class] to classify a failed step's StepFrame status.
package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// Class is the broad category of an error. Maps deterministically to an
// HTTP status code via [Class.HTTPStatus]. Classes are deliberately
// coarse-grained: code-level discrimination is via [Error.Code].
type Class string

const (
	// ClassValidation — the request was malformed or violates a schema.
	// 400 Bad Request.
	ClassValidation Class = "validation"

	// ClassUnauthorized — the request lacks valid authentication.
	// 401 Unauthorized.
	ClassUnauthorized Class = "unauthorized"

	// ClassForbidden — authenticated but lacks authorization for the
	// requested resource. 403 Forbidden.
	ClassForbidden Class = "forbidden"

	// ClassNotFound — the requested resource does not exist.
	// 404 Not Found.
	ClassNotFound Class = "not_found"

	// ClassConflict — the request would create a conflicting state
	// (duplicate key, version mismatch). 409 Conflict.
	ClassConflict Class = "conflict"

	// ClassRateLimit — too many requests. 429 Too Many Requests.
	ClassRateLimit Class = "rate_limit"

	// ClassUpstream — a downstream system (LLM provider, vendor API,
	// DB) returned an error. 502 Bad Gateway.
	ClassUpstream Class = "upstream"

	// ClassTimeout — the request took too long; a deadline was hit.
	// 504 Gateway Timeout.
	ClassTimeout Class = "timeout"

	// ClassInternal — unexpected failure, no client-actionable detail.
	// 500 Internal Server Error.
	ClassInternal Class = "internal"
)

// HTTPStatus returns the canonical HTTP status code for the class.
func (c Class) HTTPStatus() int {
	switch c {
	case ClassValidation:
		return http.StatusBadRequest
	case ClassUnauthorized:
		return http.StatusUnauthorized
	case ClassForbidden:
		return http.StatusForbidden
	case ClassNotFound:
		return http.StatusNotFound
	case ClassConflict:
		return http.StatusConflict
	case ClassRateLimit:
		return http.StatusTooManyRequests
	case ClassUpstream:
		return http.StatusBadGateway
	case ClassTimeout:
		return http.StatusGatewayTimeout
	case ClassInternal:
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}

// Error is the canonical error type. It implements the error interface,
// supports unwrap chains and JSON-friendly fields. Compare via errors.As:
//
//	var e *errs.Error
//	if errors.As(err, &e) && e.Class == errs.ClassNotFound { ... }
type Error struct {
	// Class is the broad category. Drives the HTTP status code on
	// serialization and the log level used by the gateway middleware.
	Class Class

	// Code is the stable machine-readable identifier
	// ("auth.token_expired", "validation.field_required"). Conventions:
	//   - reverse-DNS-flavored prefix
	//   - lower_snake_case suffix
	//   - never localized
	Code string

	// Message is the human-readable description in English. Localized
	// messages are produced downstream from Code (Sprint 12+ i18n).
	Message string

	// Details carries structured, machine-readable context
	// (e.g. {"field": "email", "rule": "format"}). Implementations
	// MUST NOT include secrets or PII in Details.
	Details map[string]any

	// wrapped is the underlying cause, preserved for log/trace.
	// Not exported: serialization MUST NOT leak its message verbatim
	// to clients of API surfaces that classify as Internal.
	wrapped error
}

// Error implements the error interface. When wrapped is present, the
// formatted output includes the wrapped cause for log/debug purposes.
// Note: this is for developer eyes; production responses come from the
// httpapi.respond layer, which decides whether to expose Message,
// Details and (for non-Internal classes) the wrapped cause string.
func (e *Error) Error() string {
	if e == nil {
		return "<nil errs.Error>"
	}
	if e.wrapped == nil {
		if e.Code != "" {
			return fmt.Sprintf("%s [%s]: %s", e.Class, e.Code, e.Message)
		}
		return fmt.Sprintf("%s: %s", e.Class, e.Message)
	}
	if e.Code != "" {
		return fmt.Sprintf("%s [%s]: %s: %v", e.Class, e.Code, e.Message, e.wrapped)
	}
	return fmt.Sprintf("%s: %s: %v", e.Class, e.Message, e.wrapped)
}

// Unwrap returns the wrapped cause for use with errors.Is / errors.As.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.wrapped
}

// HTTPStatus returns the canonical HTTP status for this error.
func (e *Error) HTTPStatus() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	return e.Class.HTTPStatus()
}

// WithDetail adds or overwrites a single key in Details and returns the
// same *Error for chaining. Nil-safe.
func (e *Error) WithDetail(key string, value any) *Error {
	if e == nil {
		return nil
	}
	if e.Details == nil {
		e.Details = make(map[string]any, 4)
	}
	e.Details[key] = value
	return e
}

// WithDetails merges the given map into Details and returns the same
// *Error for chaining. Nil-safe; nil-map-safe.
func (e *Error) WithDetails(details map[string]any) *Error {
	if e == nil || len(details) == 0 {
		return e
	}
	if e.Details == nil {
		e.Details = make(map[string]any, len(details))
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// New builds a fresh *Error without a wrapped cause. The most common
// constructor for first-party validation/auth errors.
func New(class Class, code, message string) *Error {
	return &Error{Class: class, Code: code, Message: message}
}

// Wrap promotes an arbitrary error into a classified *Error while
// preserving the original cause. If err is already an *Error, Wrap
// returns it unchanged (no double-classification). Passing nil returns
// nil — safe for use as `return errs.Wrap(...)`.
func Wrap(err error, class Class, code, message string) *Error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return existing
	}
	return &Error{
		Class:   class,
		Code:    code,
		Message: message,
		wrapped: err,
	}
}

// Classify extracts the Class from an error, returning ClassInternal as
// the safe default when err is nil or not an *Error. Useful for log
// dispatch and metric labeling.
func Classify(err error) Class {
	if err == nil {
		return ClassInternal
	}
	var e *Error
	if errors.As(err, &e) && e != nil {
		return e.Class
	}
	return ClassInternal
}

// IsClass reports whether err (possibly wrapped) carries the given Class.
// Equivalent to Classify(err) == c but slightly clearer at the callsite.
func IsClass(err error, c Class) bool {
	return Classify(err) == c
}

// ─── Common sentinel codes ──────────────────────────────────────────────
//
// These are not exhaustive — handlers are free to invent codes following
// the conventions documented on Error.Code. The sentinels below are the
// most frequently re-used across the platform and live here to avoid
// drift between handlers.

const (
	CodeValidationFieldRequired = "validation.field_required"
	CodeValidationFieldFormat   = "validation.field_format"
	CodeValidationSchema        = "validation.schema"

	CodeAuthTokenMissing  = "auth.token_missing"
	CodeAuthTokenInvalid  = "auth.token_invalid"
	CodeAuthTokenExpired  = "auth.token_expired"
	CodeAuthScopeRequired = "auth.scope_required"

	CodeForbiddenScope  = "forbidden.scope"
	CodeForbiddenTenant = "forbidden.tenant"
	CodeForbiddenPolicy = "forbidden.policy"

	CodeNotFoundEntity = "not_found.entity"

	CodeConflictDuplicate   = "conflict.duplicate"
	CodeConflictVersion     = "conflict.version_mismatch"
	CodeConflictIdempotency = "conflict.idempotency_mismatch"

	CodeRateLimitExceeded = "rate_limit.exceeded"

	CodeUpstreamUnavailable = "upstream.unavailable"
	CodeUpstreamProtocol    = "upstream.protocol_error"
	CodeUpstreamAuth        = "upstream.auth_failed"

	CodeTimeoutDeadline = "timeout.deadline_exceeded"
	CodeTimeoutGuard    = "timeout.guard"

	CodeInternalUnexpected = "internal.unexpected"
	CodeInternalNotImpl    = "internal.not_implemented"
)
