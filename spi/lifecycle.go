package spi

import "context"

// Lifecycle hooks define the canonical phases a Unit goes through on every
// execution. They elevate the template-method pattern into a contract for the
// platform-pipeline engine, which invokes the four hooks in order per node.
//
// All hooks receive a [LifecycleContext]. They MUST be safe to call
// concurrently across distinct execution IDs but NOT for the same
// execution ID. Implementations MAY return errors at any phase; the engine
// stops the execution and emits the appropriate StepFrame transition.
//
// The four phases run in this order for each Unit execution:
//
//  1. OnInit                - resource acquisition: open connections,
//     fetch credentials, prepare buffers.
//  2. OnProcessParameters    - validate, normalize and resolve template
//     expressions in the input map. Returns the
//     processed inputs for the next phase.
//  3. OnExecute              - the actual work. Receives processed inputs,
//     returns outputs (or error).
//  4. OnFinalize             - always-runs cleanup: close connections,
//     report metrics, scrub sensitive data from
//     the output. Receives the outcome (outputs
//     and possibly an error) and MAY mutate the
//     outputs (e.g. apply IO.Mask field redaction).
//
// Units that need only a subset of hooks SHOULD embed [DefaultLifecycle]
// (re-exported by plugfy-sdk) and override the relevant methods.
type Lifecycle interface {
	OnInit(ctx LifecycleContext) error
	OnProcessParameters(ctx LifecycleContext, in map[string]any) (map[string]any, error)
	OnExecute(ctx LifecycleContext, in map[string]any) (map[string]any, error)
	OnFinalize(ctx LifecycleContext, out map[string]any, runErr error)
}

// LifecycleContext is the rich execution context handed to each hook.
// It carries identity (which unit, which run), tracing, cancellation,
// access to credentials and state, and structured logging.
//
// Re-exported through plugfy-sdk as a thin facade so third-party units never
// import platform internals; the platform-pipeline engine supplies the concrete
// implementation at execution time.
type LifecycleContext interface {
	// Context returns the cancellation-aware Go context. Hook
	// implementations SHOULD use it for downstream calls.
	Context() context.Context

	// RunID is the unique identifier of this execution (ULID).
	RunID() string

	// NodeID is the identifier of the current node within the pipeline.
	// Empty for the top-level pipeline lifecycle.
	NodeID() string

	// UnitID is the reverse-DNS identifier of the executing unit
	// (e.g. "com.acme.weather.lookup").
	UnitID() string

	// UnitVersion is the SemVer of the executing unit.
	UnitVersion() string

	// Tenant returns the organization and project owning this run.
	// Used by hooks to scope credential/state lookups.
	Tenant() TenantRef

	// Logger returns a structured logger pre-populated with run/unit
	// identification. Hooks SHOULD use it rather than the standard
	// library log to ensure observability correlation.
	Logger() Logger

	// Tracer returns the tracer for nested span creation. The current
	// lifecycle phase already has its own span; downstream code can
	// build children from it through Tracer().Start(ctx, name).
	Tracer() Tracer

	// State returns a scoped accessor for unit state declared in the
	// manifest's spec.state block (Sprint 5 T5.1).
	State() StateAccessor

	// Credentials returns the credential accessor scoped to this unit.
	// Access is gated by the unit's manifest requires/capabilities
	// (Sprint 4 T4.x).
	Credentials() CredentialAccessor
}

// TenantRef identifies the tenant scope of an execution: an organization
// and a project within it. These two levels scope every credential, state,
// and audit lookup the hooks perform.
type TenantRef struct {
	OrgID     string
	ProjectID string
}

// Logger is the minimal structured-logger contract surfaced through the
// LifecycleContext. The real implementation wraps slog. Kept here to
// avoid pulling log/slog into the contracts module (stdlib is fine, but
// the dependency surface stays tight).
type Logger interface {
	Debug(msg string, fields ...LogField)
	Info(msg string, fields ...LogField)
	Warn(msg string, fields ...LogField)
	Error(msg string, fields ...LogField)
}

// LogField is a structured key/value attached to a log line.
type LogField struct {
	Key   string
	Value any
}

// Tracer is the minimal OpenTelemetry-shaped tracer interface surfaced
// to hooks. Real impl wraps go.opentelemetry.io/otel.
type Tracer interface {
	Start(ctx context.Context, name string, attrs ...TraceAttr) (context.Context, Span)
}

// Span represents an active trace span. End MUST be called once.
type Span interface {
	End(opts ...SpanEndOption)
	SetAttributes(attrs ...TraceAttr)
	RecordError(err error)
}

// TraceAttr is a span attribute (key/value).
type TraceAttr struct {
	Key   string
	Value any
}

// SpanEndOption is a marker interface for span-end behavioral options
// (e.g. set explicit end timestamp). Reserved for future expansion.
type SpanEndOption interface{ isSpanEndOption() }

// StateAccessor reads and writes the unit's declared state fields
// (cursor, lastTimestamp, errorCount, …). Scope is enforced server-side
// per the manifest spec.state.persistence rule (org+project+instance).
type StateAccessor interface {
	Get(key string) (any, bool)
	Set(key string, value any) error
	Delete(key string) error
}

// CredentialAccessor resolves credentials declared as Required in the
// unit's manifest. Returns ErrCredentialNotFound when the credential is
// not configured or ErrCredentialForbidden when the unit's capability set
// doesn't permit access.
type CredentialAccessor interface {
	// Get returns the materialized credential payload (e.g. an OAuth2
	// access token, an API key). The payload is opaque to the unit
	// except for the documented fields per credential schema.
	Get(name string) (CredentialPayload, error)
}

// CredentialPayload is the resolved credential. Refresh is handled
// transparently by the platform; units re-call Get to obtain a current
// payload rather than caching it across executions.
type CredentialPayload struct {
	// Type is the credential schema identifier (e.g. "oauth2",
	// "api_key", "mtls").
	Type string
	// Data carries schema-specific fields (e.g. {"access_token": "..."}).
	Data map[string]string
	// ExpiresAt is the zero value if the payload doesn't expire.
	// Otherwise units MUST NOT use it after that instant; the platform
	// will have refreshed by then if the credential supports it.
	ExpiresAt int64 // unix seconds; 0 = never
}

// DefaultLifecycle is the zero-cost default implementation suitable for
// embedding in concrete Unit structs. It treats OnInit, OnProcessParameters
// and OnFinalize as no-ops, requiring concrete units to override only
// OnExecute. The Plugfy SDK uses this internally; consumers of the SPI
// directly (rare) MAY also embed it.
//
// Embedded usage:
//
//	type MyTool struct { spi.DefaultLifecycle }
//	func (t *MyTool) OnExecute(ctx spi.LifecycleContext, in map[string]any) (map[string]any, error) {
//	    // ...
//	}
type DefaultLifecycle struct{}

func (DefaultLifecycle) OnInit(LifecycleContext) error { return nil }
func (DefaultLifecycle) OnProcessParameters(_ LifecycleContext, in map[string]any) (map[string]any, error) {
	return in, nil
}
func (DefaultLifecycle) OnExecute(LifecycleContext, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (DefaultLifecycle) OnFinalize(LifecycleContext, map[string]any, error) {}
