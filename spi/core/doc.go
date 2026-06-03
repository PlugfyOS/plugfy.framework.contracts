// Package core declares the ONE LEGO brick of the Plugfy Framework: a Unit.
//
// A Unit is everything executable in the platform — a tool, connector, agent,
// app, component, plugin, layer or whole solution. There is no second contract
// per role; richer roles are the SAME brick composed. The brick is exactly two
// methods:
//
//	Describe() UnitDescriptor                                  // pure, ctx-free self-description
//	Invoke(ctx UnitContext, method string, in map[string]any)  // one named, async method
//
// Everything execution-cross-cutting — progress, events, errors, finalization,
// retries, timeouts, validation — lives in the UnitContext the brick is handed
// and in a runtime policy wrapper (the future core.Runner) that sits OUTSIDE the
// brick, driven by what the brick DECLARES in its UnitDescriptor. The author of a
// Unit writes a Describe() literal and one method body; they write nothing for
// retries, finalization, progress plumbing, or event fan-out.
//
// # Typed IO: scalars, objects, and arrays
//
// A method DECLARES its input and output shape as []ParamDef. ParamType is a
// closed set: the scalars (boolean/string/integer/float/enum), two structured
// tags — object (an unordered keyed map) and array (an ordered list) — and secret.
// A structured param carries an OPTIONAL, recursive element schema on ParamDef:
// an array's element shape in Items, an object's named fields in Fields. These are
// pure declarations — the runtime validator reads them to check a value's shape
// before Invoke and the UI reads them to render a typed editor; the core itself
// never executes them. Recursion composes arbitrarily (array-of-objects,
// object-of-arrays), so a brick fully describes structured IO without a separate
// schema language or a platform type.
//
// # What the core does NOT know
//
// The core is PURE: it imports NO platform/foundation type and carries ONLY what
// is intrinsic to running a brick. Everything else is a PLATFORM/FOUNDATION
// concern layered ON the core by reading Describe() and wrapping the Unit —
// never a field on this contract:
//
//   - capability negotiation (provides/requires, spi.CapabilityRequirement) —
//     the resolver/host concern;
//   - signing / provenance (supply-chain, verify-before-install) — the
//     installer/foundation concern;
//   - data / state (CQRS, apps-own-data) — the persistence foundation concern;
//   - settings and themes contributions — the settings/ui foundation concern;
//   - auth-scope, visibility (CEL gating), output masking — the access-control /
//     compliance foundation concern.
//
// The platform reads the pure Describe() and ADDS its own governance descriptor
// plus Unit -> Unit wrappers (signature gating, capability checks, state binding,
// settings/theme registration, visibility/mask enforcement). The core stays a
// self-contained, execution-only contract; the richness lives in the layers
// composed on top.
//
// # The brick (this package, Phase 0)
//
// This package is layer 1 of the three-layer stack (DEFINITION -> EXECUTION ->
// OPERATION) and its contract HOME is commons. It is PURE ADDITIVE: nothing in
// the platform consumes it yet (Phase 1 promotes the wrapper; Phase 2+ migrate
// callers). It REUSES, not reinvents, the existing commons surfaces:
//
//   - spi.Provider          — embedded by Unit (Name/Kind/Capabilities/HealthCheck),
//     so the existing registry/discovery/health/capability machinery works
//     unchanged: a Unit is already a first-class Provider.
//   - spi.LifecycleContext  — embedded by UnitContext (identity/tenant/logger/
//     tracer/state/credentials), extended with only Report/Emit/Deadline/Method.
//   - events.CloudEvent     — aliased as Event; the brick never duplicates the shape.
//   - resilience.{Guard,Breaker,Bulkhead,RetryPolicy} — the wrapper reuses these
//     verbatim; this package declares a DECLARATIVE RetryPolicy the wrapper maps
//     onto them.
//   - spi.Evaluator         — the CEL port the future core.Runner uses to evaluate
//     ParamDef.Validate before Invoke (platform-layer visibility/access predicates
//     reuse the same port outside the core).
//
// # Recursive composition
//
// A higher layer is a Unit whose Invoke fans out to child Units: a pipeline IS a
// Unit, a node IS a Unit reference, a solution IS a Unit of Unit-nodes. Wrapping
// a Unit yields a Unit, so policy wrappers (trust-tier isolation, signature
// gating, CEL visibility, the retry/finalize Runner) STACK — every safety and
// observability concern is a Unit -> Unit decorator, none of which touches the
// brick body. That is how "richness lives in composition" is literally true.
package core
