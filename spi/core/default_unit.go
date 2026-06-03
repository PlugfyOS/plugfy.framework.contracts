package core

import (
	"context"

	commonspi "github.com/PlugfyOS/plugfy.framework.contracts/spi"
)

// Handler is one named method's implementation.
type Handler func(ctx UnitContext, in map[string]any) (map[string]any, error)

// DefaultUnit is the embeddable zero-value brick (the heir of spi.DefaultLifecycle).
// Embedding it makes a struct a Unit; the author overrides only Describe (or sets
// Desc) and registers one Handler per method via Method(). This is the proof that
// "the author writes ~nothing": there is no lifecycle plumbing to override,
// because the plumbing is the wrapper's.
//
// The zero value is usable: an empty DefaultUnit is a Unit whose Invoke returns
// ErrMethodNotFound for every method. Method() lazily initializes the registry.
type DefaultUnit struct {
	Desc     UnitDescriptor
	handlers map[string]Handler // method name -> impl, registered via Method()
}

// Provider surface — satisfied directly from the descriptor. Capabilities is the
// generic Provider feature-flag map; the pure core declares none, so it returns
// an empty map. Capability NEGOTIATION (provides/requires/baseplate
// requirements) is a PLATFORM concern layered on by reading Describe(), not a
// core descriptor field — override this if a host wants to surface flags.
func (b *DefaultUnit) Name() string                      { return b.Desc.ID }
func (b *DefaultUnit) Kind() commonspi.Kind              { return commonspi.Kind(b.Desc.Kind) }
func (b *DefaultUnit) Capabilities() map[string]any      { return map[string]any{} }
func (b *DefaultUnit) HealthCheck(context.Context) error { return nil } // override if needed

// Describe returns the embedded descriptor.
func (b *DefaultUnit) Describe() UnitDescriptor { return b.Desc }

// Method registers a Handler for a named method and returns the receiver so
// registrations chain. Re-registering a name overwrites the prior handler.
func (b *DefaultUnit) Method(name string, h Handler) *DefaultUnit {
	if b.handlers == nil {
		b.handlers = make(map[string]Handler)
	}
	b.handlers[name] = h
	return b
}

// Invoke dispatches to the registered Handler for method, returning a typed,
// classifiable ErrMethodNotFound when no handler is registered.
func (b *DefaultUnit) Invoke(ctx UnitContext, method string, in map[string]any) (map[string]any, error) {
	h, ok := b.handlers[method]
	if !ok {
		return nil, ErrMethodNotFound(method)
	}
	return h(ctx, in)
}

// Compile-time assertion that DefaultUnit satisfies the Unit contract.
var _ Unit = (*DefaultUnit)(nil)
