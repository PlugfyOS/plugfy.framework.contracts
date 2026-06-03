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
// Everything cross-cutting — progress, events, errors, finalization, retries,
// timeouts, masking, validation, isolation, signing, CEL gating — lives in the
// UnitContext the brick is handed and in a runtime policy wrapper (the future
// core.Runner) that sits OUTSIDE the brick, driven by what the brick DECLARES in
// its UnitDescriptor. The author of a Unit writes a Describe() literal and one
// method body; they write nothing for retries, finalization, progress plumbing,
// or event fan-out.
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
//   - spi.CapabilityRequirement — carried verbatim on the descriptor.
//   - events.CloudEvent     — aliased as Event; the brick never duplicates the shape.
//   - resilience.{Guard,Breaker,Bulkhead,RetryPolicy} — the wrapper reuses these
//     verbatim; this package declares a DECLARATIVE RetryPolicy the wrapper maps
//     onto them.
//   - spi.Evaluator         — the CEL port the future core.Runner uses to evaluate
//     ParamDef.Validate and Visibility before Invoke.
//
// # Commons-resident descriptor mirror types
//
// The authoritative contract types the UnitDescriptor's supply-chain and
// composition fields (Provides/Requires/Signing/State/Settings/Themes) against
// platform-runtime/manifest. commons CANNOT import platform-runtime — that
// module already depends on commons, so the import would be a module cycle and
// would violate the contract's own invariant that the DEFINITION layer carries
// NO upward dependency on EXECUTION/OPERATION. These declarations therefore live
// here as the commons-HOME twins of the manifest types, carried verbatim onto
// the descriptor (identical field names and JSON tags). The host projects
// between them at the composition root; Phase 3 reconciles the wire/manifest
// projection. Nothing in Phase 0 reads their internals, so the carry is exact.
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
