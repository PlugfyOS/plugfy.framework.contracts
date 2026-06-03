package core

import "time"

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

// ParamType is the closed parameter type set for a method's typed IO. The base
// set (boolean/string/integer/float/enum) plus object/array/secret is intrinsic
// to a method's input/output. The presentation-oriented values (color/multiline/
// sdui) are a harmless declaration on the param: the core never interprets them;
// the UI FOUNDATION reads them when it renders the param. The tag itself stays
// here because it is part of how a method DECLARES its IO; the meaning is layered
// on. The two structured tags carry an OPTIONAL element shape on ParamDef:
// ParamObject reads ParamDef.Fields (named field schemas), ParamArray reads
// ParamDef.Items (the element schema) — both pure, recursive, read by the runtime
// validator and the UI, never executed by the core.
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
	// object, array, and secret extend the set for invocation IO. object is an
	// unordered keyed map (field shape via Fields); array is an ordered list
	// (element shape via Items):
	ParamObject ParamType = "object"
	ParamArray  ParamType = "array"
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

// UnitDescriptor is the ONE self-description of a Unit: identity, version, the
// composition ROLE tag, human-facing text, free-form metadata, the NAMED method
// set, and the unit-wide default execution policy. It is PURE: it carries ONLY
// what is intrinsic to a runnable brick and references NO platform/foundation
// type. Capability negotiation, supply-chain/signing, data/state, settings,
// themes, access-control and visibility are PLATFORM/FOUNDATION concerns layered
// ON the core by reading Describe() — see doc.go.
type UnitDescriptor struct {
	ID          string            // reverse-DNS identity (== LifecycleContext.UnitID())
	Version     string            // SemVer               (== LifecycleContext.UnitVersion())
	Kind        Kind              // tool|agent|app|service|module|plugin|component|extension|solution
	Title       string            //
	Description string            //
	Metadata    map[string]string // labels/annotations carried verbatim into audit/observability

	Methods []MethodDef // NAMED METHODS — the headline gap, closed; >= 1

	// DefaultPolicy is the unit-wide default a method's policy overrides; nil =
	// run-once.
	DefaultPolicy *RetryPolicy
}

// MethodDef is a named operation with typed UI-hinted IO and declared execution
// policy. This is where the brick stays simple: every execution-intrinsic
// concern is a FIELD here, executed by the wrapper, never a method on Unit.
// Cross-cutting governance that is NOT execution-intrinsic — output masking,
// auth scope, visibility — is NOT here; the platform layers it on by reading the
// descriptor (see doc.go).
type MethodDef struct {
	Name    string     // the `method` Invoke receives; the wire `function`; the Node `function`
	Title   string     //
	Summary string     //
	Params  []ParamDef // typed UI-hinted INPUT  (closes the opaque UnitConfig.Schema $ref gap)
	Returns []ParamDef // typed OUTPUT schema     (enables compose-time typed edges)

	// Execution-intrinsic policy this method DECLARES; the wrapper executes it.
	// The author writes none of the imperative logic.
	Retry      *RetryPolicy  // DECLARED retries  — wrapper runs the backoff loop
	Timeout    time.Duration // DECLARED deadline — wrapper sets ctx deadline
	Idempotent bool          // safe-to-retry hint feeding the wrapper's default Retryable
	Streaming  bool          // method emits a frame stream -> supervisor.v1 InvokeStream
}

// ParamDef is a typed, UI-hinted parameter for a runnable method. It is
// intrinsic to a method's IO: the typing and validation belong to the method
// itself. The closed Type set (ParamType) tags the value; presentation-oriented
// tags are interpreted by the ui foundation, never by the core.
type ParamDef struct {
	Key         string
	Label       string
	Description string
	Type        ParamType // closed type set + object/array/secret
	Default     any
	Options     []string // enum
	Required    bool
	Secret      bool   // routed to the secret store, never plaintext, masked in frames
	Validate    string // OPTIONAL CEL predicate over the value (sandboxed) — closes the validation gap

	// Items is the OPTIONAL element schema for Type==ParamArray (nil = elements of
	// any shape). Fields is the OPTIONAL named-field schema for Type==ParamObject
	// (empty = open object; undeclared keys are tolerated). Both are pure,
	// recursive declarations the runtime validator and the UI read — the core never
	// executes them. Recursion gives nested array-of-objects, object-of-arrays, etc.
	Items  *ParamDef  // element schema for arrays
	Fields []ParamDef // field schemas for objects
}
