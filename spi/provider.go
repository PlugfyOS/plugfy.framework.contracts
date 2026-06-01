// Package spi defines the Service Provider Interfaces (SPIs) of the PlugfyOS
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
