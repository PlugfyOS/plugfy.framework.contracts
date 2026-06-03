package core

import (
	"time"

	commonspi "github.com/PlugfyOS/plugfy-common/spi"
)

// Kind is the composition ROLE tag. Nothing in the contract branches on Kind —
// it is metadata the resolver/marketplace/host read to decide replaceability and
// the mount surface. A plugin is NOT a different interface; it is a Unit with
// Kind=plugin.
type Kind string

const (
	KindTool      Kind = "tool"
	KindAgent     Kind = "agent"
	KindApp       Kind = "app"
	KindService   Kind = "service"
	KindModule    Kind = "module"
	KindPlugin    Kind = "plugin"
	KindComponent Kind = "component"
	KindExtension Kind = "extension"
	KindSolution  Kind = "solution" // a vertical / pipeline-of-units exposed AS a unit
)

// ParamType is the closed parameter type set, reused VERBATIM from
// FieldDescriptor's Settings type set so the existing UI renderer works for
// invocation params, not just settings — extended with object and secret for
// invocation IO.
type ParamType string

const (
	ParamBoolean   ParamType = "boolean"
	ParamString    ParamType = "string"
	ParamInteger   ParamType = "integer"
	ParamFloat     ParamType = "float"
	ParamEnum      ParamType = "enum"
	ParamColor     ParamType = "color"
	ParamMultiline ParamType = "multiline"
	ParamSDUI      ParamType = "sdui"
	// object and secret extend the FieldDescriptor set for invocation IO:
	ParamObject ParamType = "object"
	ParamSecret ParamType = "secret"
)

// Reserved control-plane method names (Architect decision 1). Control-plane ops
// the HOST drives live under a reserved "sys." prefix; a domain MethodDef.Name
// MUST NOT begin with "sys.". This makes it structurally impossible for a domain
// method to collide with, shadow, or impersonate a host op. An op is just a
// reserved "sys."-prefixed method on the same Invoke seam.
const (
	SysPrefix   = "sys."
	SysSeedDemo = "sys.seedDemo"
	SysDescribe = "sys.describe"
	SysMigrate  = "sys.migrate"
	SysHealth   = "sys.health"
)

// UnitDescriptor is the one self-description: identity, version, typed UI-hinted
// parameters, the NAMED method set, declared capabilities and cross-cutting
// policy. It REFERENCES existing types verbatim (it does not re-invent them):
// Capabilities is the commons spi.CapabilityRequirement; the supply-chain and
// composition fields use the commons-home twins of platform-runtime/manifest
// (see descriptor_manifest.go for why commons cannot import that module).
type UnitDescriptor struct {
	ID          string            // reverse-DNS identity (== LifecycleContext.UnitID())
	Version     string            // SemVer               (== LifecycleContext.UnitVersion())
	Kind        Kind              // tool|agent|app|service|module|plugin|component|extension|solution
	Title       string            //
	Description string            //
	Metadata    map[string]string // labels/annotations carried verbatim into audit/observability

	Methods []MethodDef // NAMED METHODS — the headline gap, closed; >= 1

	// Capability negotiation — KEPT verbatim, no regress to a flat string:
	Provides     []Capability                      // OSGi-style provides
	Requires     []Requirement                     // OSGi version ranges
	Capabilities []commonspi.CapabilityRequirement // commons baseplate requirement shape

	// Supply-chain + state references — the descriptor carries them, the
	// installer/host enforce them (verify-before-install stays):
	Signing *Signing   // signature/provenance policy (no regress)
	State   *UnitState // CQRS/apps-own-data: THIS unit's own state fields

	// Composition contributions (Settings/Themes) — these are composition, not
	// invocation:
	Settings []SettingsContribution
	Themes   []ThemeContribution

	// DefaultPolicy is the unit-wide default a method's policy overrides; nil =
	// run-once.
	DefaultPolicy *RetryPolicy
	// Visibility is a CEL predicate gating whether the unit is offered to a
	// caller (sandboxed).
	Visibility string
}

// capabilitiesMap flattens the descriptor's declared capabilities into the
// map[string]any shape spi.Provider.Capabilities returns, so a DefaultUnit can
// satisfy Provider without the author duplicating the data.
func (d UnitDescriptor) capabilitiesMap() map[string]any {
	out := make(map[string]any, len(d.Provides)+len(d.Requires)+len(d.Capabilities))
	for _, c := range d.Provides {
		out["provides:"+c.Name] = c.Version
	}
	for _, r := range d.Requires {
		out["requires:"+r.Capability] = r.VersionRange
	}
	for _, cr := range d.Capabilities {
		out["capability:"+cr.Capability] = cr.Version
	}
	return out
}

// MethodDef is a named operation with typed UI-hinted IO and declared policy.
// This is where the brick stays simple: every cross-cutting concern is a FIELD
// here, executed by the wrapper, never a method on Unit.
type MethodDef struct {
	Name    string     // the `method` Invoke receives; the wire `function`; the Node `function`
	Title   string     //
	Summary string     //
	Params  []ParamDef // typed UI-hinted INPUT  (closes the opaque UnitConfig.Schema $ref gap)
	Returns []ParamDef // typed OUTPUT schema     (enables compose-time typed edges)

	// Cross-cutting policy this method DECLARES; the wrapper executes it. The
	// author writes none of the imperative logic.
	Retry      *RetryPolicy  // DECLARED retries  — wrapper runs the backoff loop
	Timeout    time.Duration // DECLARED deadline — wrapper sets ctx deadline
	Mask       []string      // output keys redacted on finalize
	Idempotent bool          // safe-to-retry hint feeding the wrapper's default Retryable

	// Transport + mount hints (data, not behavior):
	Streaming  bool   // method emits a frame stream -> supervisor.v1 InvokeStream / api.Route.Streaming
	AuthScope  string // EAPI auth scope when this method is mounted as a route (trust tiers)
	Visibility string // CEL predicate gating availability of THIS method (sandboxed)
}

// ParamDef is a typed, UI-hinted parameter for a runnable method. It supersedes
// FieldDescriptor and reuses its validated type set verbatim (ParamType),
// promoting it from Settings-only to per-method invocation IO (input and output).
type ParamDef struct {
	Key         string
	Label       string
	Description string
	Type        ParamType // reuses the FieldDescriptor closed set + object/secret
	Default     any
	Options     []string // enum
	Required    bool
	Secret      bool   // routed to the secret store, never plaintext, masked in frames
	Validate    string // OPTIONAL CEL predicate over the value (sandboxed) — closes the validation gap
}
