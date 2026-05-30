// Package idempotency provides server-side replay protection keyed by the
// client-supplied Idempotency-Key request header (Stripe / RFC draft style).
//
// The semantics:
//
//   - First request with key K records the produced response (status,
//     headers, body) and returns it normally.
//   - Subsequent requests with the same K within the TTL receive the
//     cached response verbatim, without re-invoking the handler.
//   - Keys are scoped per authenticated principal + request path to
//     prevent cross-tenant collisions and accidental key reuse across
//     unrelated endpoints.
//
// The store contract is intentionally narrow ([Store]) so that an
// in-memory implementation (this package) and a Postgres-backed
// implementation (added in Sprint 1+ when multi-instance deployments
// arrive) can be swapped at the composition root.
//
// Sprint 1 T1.6: introduces the Store contract, the in-memory backing
// implementation and the HTTP middleware wiring under internal/httpapi.
package idempotency

import (
	"context"
	"time"
)

// CachedResponse is the result of a previously-served request that the
// store will replay on duplicate keys.
type CachedResponse struct {
	// Status is the HTTP status code originally returned.
	Status int
	// Headers contains the subset of response headers the store decides
	// to preserve. By default we keep Content-Type and any header
	// matching the "X-*" prefix family (custom domain headers); other
	// headers (e.g. Date, X-Request-Id) are regenerated per request.
	Headers map[string]string
	// Body is the response body bytes. The store does not interpret
	// them.
	Body []byte
	// StoredAt is the wall-clock instant the entry was recorded.
	// Used by clients of the store to compute remaining TTL.
	StoredAt time.Time
}

// Store persists [CachedResponse] entries keyed by a composite
// (subject, path, idempotency_key) tuple. The composite key is opaque to
// the store; the middleware builds it from the authenticated request.
//
// Implementations MUST be safe for concurrent use.
type Store interface {
	// Get returns the cached response for the composite key, if any.
	// The boolean reports presence; the *CachedResponse is nil when
	// false.
	Get(ctx context.Context, key string) (*CachedResponse, bool, error)

	// Set records the response under the composite key with the given
	// TTL. If an entry already exists, Set MUST overwrite it (the
	// middleware uses Get-then-Set semantics; concurrent writers are
	// expected to be rare and last-write-wins is acceptable).
	Set(ctx context.Context, key string, resp *CachedResponse, ttl time.Duration) error

	// Delete removes the entry for the composite key. Idempotent: a
	// missing key is not an error.
	Delete(ctx context.Context, key string) error
}
