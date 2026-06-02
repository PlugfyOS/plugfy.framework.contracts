// Package spi defines the Service Provider Interfaces (SPIs) of the Plugfy
// platform. Every pluggable external dependency (models, embeddings, vector
// store, object storage, identity, connectors, notifications, secrets, event
// bus) is abstracted behind one of these interfaces so that the concrete
// implementation can be selected per edition (Local / Cloud / Enterprise).
//
// The interfaces are the normative L1 contracts of the platform; the layer
// model and edition matrix live in PlugfyOS/plugfy-platform.
package spi

import "context"

// Kind categorizes an SPI provider.
type Kind string

const (
	KindModel        Kind = "model"
	KindEmbedding    Kind = "embedding"
	KindVectorStore  Kind = "vectorstore"
	KindStorage      Kind = "storage"
	KindIdentity     Kind = "identity"
	KindConnector    Kind = "connector"
	KindNotification Kind = "notification"
	KindSecret       Kind = "secret"
	KindEventBus     Kind = "eventbus"
	KindDatabase     Kind = "database"
	KindRAG          Kind = "rag"
	KindAuthorizer   Kind = "authorizer"
	// KindRegistry categorizes a control-plane registry provider that
	// persists namespaced key/value records (the installed-module index,
	// route contributions, capability bindings) behind the persistence
	// RegistryStore contract. The concrete backend (Postgres, SQLite) lives
	// in a provider repo; only the SPI category is named here.
	KindRegistry Kind = "registry"
	// KindAPI categorizes a route-provider: a unit that contributes HTTP
	// route declarations (api.RouteContribution) to the platform API host
	// for mounting. The provider returns pure data — method, path, auth
	// scope, schema refs — and never imports net/http.
	KindAPI Kind = "api"
)

// Provider is the base interface implemented by every pluggable provider.
type Provider interface {
	// Name is the unique identifier of the provider implementation.
	Name() string
	// Kind reports the SPI category this provider satisfies.
	Kind() Kind
	// Capabilities declares feature flags/limits for capability negotiation.
	Capabilities() map[string]any
	// HealthCheck verifies the provider is reachable and ready.
	HealthCheck(ctx context.Context) error
}

// CapabilityRequirement is one declared capability dependency: a capability
// name plus the SemVer range the dependent admits. It is the cross-cutting
// shape a host or a module uses to say "I need capability X within version
// range R", which the version-compatibility matrix resolves by
// Minimal-Version-Selection against the installed set.
//
// It lives on the L1 baseplate (rather than in a single consumer) because both
// the per-host dependency manifest (installed.HostManifest) and every unit's
// compatibility{} block speak it, and the pure admissibility check
// (installed.Admissible) operates on it — so the contract belongs where every
// side can reuse it verbatim without importing another unit.
type CapabilityRequirement struct {
	// Capability is the required capability name (e.g. "storage", "identity").
	Capability string `json:"capability"`
	// Version is the SemVer range the requirement admits (e.g. ">=1.0.0").
	// Empty means "any version of that capability".
	Version string `json:"version,omitempty"`
}
