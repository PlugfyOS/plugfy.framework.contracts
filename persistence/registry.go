package persistence

import "context"

// Record is one namespaced key/value entry in a [RegistryStore]: an opaque
// JSON value addressed by (Namespace, Key) with an optimistic-concurrency
// Version. The platform stores its own registries (the installed-module index,
// route contributions, capability bindings) as records; the value bytes are
// opaque to the store.
type Record struct {
	// Namespace groups related keys (e.g. "installed", "routes",
	// "capabilities"). It scopes List/Scan and isolates registries.
	Namespace string `json:"namespace"`
	// Key is the unique key within the namespace.
	Key string `json:"key"`
	// Value is the opaque payload (typically marshaled JSON). The store
	// does not interpret it.
	Value []byte `json:"value"`
	// Version is the monotonically increasing revision of this record,
	// assigned by the store on each Set. Callers MAY use it for
	// optimistic concurrency; 0 means "the record did not exist".
	Version int64 `json:"version"`
}

// RegistryTx is a control-plane transaction: the same record operations as
// [RegistryStore] applied atomically until Commit or Rollback.
type RegistryTx interface {
	Get(ctx context.Context, namespace, key string) (Record, error)
	Set(ctx context.Context, rec Record) (Record, error)
	Delete(ctx context.Context, namespace, key string) error
	List(ctx context.Context, namespace string) ([]Record, error)
	Commit() error
	Rollback() error
}

// RegistryStore is the control-plane key/value contract the platform programs
// against for its own registries. It is namespaced, versioned, and
// transactional. Implementations MUST be safe for concurrent use and return
// [ErrNoRows] from Get/Delete when the addressed record is absent.
type RegistryStore interface {
	// Get returns the record for (namespace, key) or [ErrNoRows] if it is
	// absent.
	Get(ctx context.Context, namespace, key string) (Record, error)

	// Set writes the record (insert or overwrite) and returns it with its
	// store-assigned Version. The Namespace and Key of rec address the
	// row; the Version of rec is ignored on write (the store assigns the
	// new one).
	Set(ctx context.Context, rec Record) (Record, error)

	// Delete removes (namespace, key). Returns [ErrNoRows] when the record
	// is absent so callers can distinguish a no-op from a deletion.
	Delete(ctx context.Context, namespace, key string) error

	// List returns every record in the namespace, ordered by Key.
	List(ctx context.Context, namespace string) ([]Record, error)

	// Scan returns every record in the namespace whose Key begins with
	// prefix, ordered by Key. A prefix of "" is equivalent to List.
	Scan(ctx context.Context, namespace, prefix string) ([]Record, error)

	// Tx runs fn inside a control-plane transaction, committing on a nil
	// return and rolling back on a non-nil return or panic.
	Tx(ctx context.Context, fn func(tx RegistryTx) error) error
}
