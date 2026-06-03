package core

import (
	"time"

	"github.com/PlugfyOS/plugfy-common/events"
	commonspi "github.com/PlugfyOS/plugfy-common/spi"
)

// Unit is the LEGO brick. SUPER SIMPLE by design.
//
// It embeds spi.Provider (Name/Kind/Capabilities/HealthCheck) so the EXISTING
// registry, discovery, health-probe and capability-negotiation machinery work
// UNCHANGED — a Unit is already a first-class Provider. Over Provider it adds
// exactly two methods.
type Unit interface {
	commonspi.Provider

	// Describe returns the brick's complete static self-description: identity,
	// version, typed UI-hinted parameters, the NAMED method set, and declared
	// capabilities + cross-cutting policy.
	//
	// Describe MUST be pure and ctx-free. The kernel reads it BEFORE any run — to
	// verify the signature, negotiate capabilities, mount EAPI routes, validate a
	// composition at design time, and build the policy wrapper. Same instance ->
	// same descriptor; cacheable per (id, version).
	Describe() UnitDescriptor

	// Invoke runs ONE named method asynchronously. It is the single entry verb.
	//
	//   - ctx UnitContext   — the brick RECEIVES the context. It carries
	//     cancel+deadline, progress, events, logger, tracer, tenant, state and
	//     brokered credentials. ASYNC lives here: ctx.Context() is
	//     cancellation-aware and the engine honors it at every boundary.
	//   - method string     — NAMED METHODS. method MUST be one the descriptor
	//     declared; the wrapper validates this and the inputs BEFORE calling, so
	//     the body may assume both are valid.
	//   - in map[string]any — the typed-validated input (the ModuleDispatcher /
	//     supervisor.v1 shape, so no re-marshalling at the native call site).
	//
	// Returns the method's typed output (or a *UnitError the wrapper classifies).
	Invoke(ctx UnitContext, method string, in map[string]any) (map[string]any, error)
}

// UnitContext is the rich context Invoke receives. It EXTENDS the real
// spi.LifecycleContext with exactly the missing emitter surfaces (Report, Emit),
// an explicit Deadline, the currently-executing Method, and the optional
// idempotency key. Everything else — identity, tenant, logger, tracer, state,
// brokered credentials — is inherited UNCHANGED. The brick is handed this; it
// codes none of it.
type UnitContext interface {
	commonspi.LifecycleContext

	// Method returns the named method currently executing.
	Method() string

	// Deadline is the effective deadline (from MethodDef.Timeout or the run).
	Deadline() (time.Time, bool)

	// IdempotencyKey is the OPTIONAL caller-supplied de-dup key for this
	// invocation. It lives on the CONTEXT, not the brick signature, so the
	// two-method brick is untouched. When present AND MethodDef.Idempotent is
	// set, the Runner dedups on it via plugfy-common/idempotency.Store (replay
	// the recorded Result instead of re-invoking). Empty => at-least-once with no
	// dedup. The brick body never reads this; the Runner does.
	IdempotencyKey() string

	// Report emits INTRA-execution PROGRESS. The author calls it inside a loop;
	// the wrapper relays it to a StepFrame delta (in-proc) or a supervisor.v1
	// InvokeEvent{event:"progress"} frame (spawned).
	Report(p Progress)

	// Emit raises a domain EVENT from inside Invoke THROUGH the context (not via a
	// bus reference captured at construction). The wrapper routes it to the
	// commonspi.EventBus (in-proc) or the supervisor.v1 EventChannel (spawned).
	Emit(e Event) error
}

// Progress is one intra-execution progress report relayed by the wrapper.
type Progress struct {
	Done, Total int64   // 0/0 = indeterminate
	Percent     float64 // optional precomputed
	Message     string
	Stage       string // free-form sub-phase label
	Data        map[string]any
}

// Event is the plugfy-common/events.CloudEvent envelope, reused verbatim so the
// brick never duplicates the shape (the wire carries it as cloudevent_json).
type Event = events.CloudEvent

// Resourceful units acquire long-lived resources ONCE per ACTIVATION. Acquire is
// the heir of OnInit / a loader's Open; the returned Finalizer is the ACTIVATION
// teardown the wrapper defers to DEACTIVATION (load/unload lifetime), NOT to the
// next Invoke. A spawned subprocess / wasm instance is opened here and torn down
// via this teardown at deactivation (architect decision 3).
type Resourceful interface {
	Acquire(ctx UnitContext) (ActivationFinalizer, error)
}

// ActivationFinalizer is the activation-level teardown returned by
// Resourceful.Acquire; the wrapper calls it ONCE, at deactivation, after the last
// Invoke. SPLIT from the per-Invoke OnFinalizer below — different lifetime,
// different hook (architect decision 3). Named distinctly from the per-Invoke
// finalize so the two levels can never be conflated.
type ActivationFinalizer interface {
	// Finalize releases the activation-level resources acquired in Acquire.
	Finalize(ctx UnitContext) error
}

// OnFinalizer is the PER-INVOKE finalize the Runner calls on EVERY exit path of
// EACH Invoke (success, error, panic-recover, timeout, cancel). It applies
// MethodDef.Mask redaction + per-call cleanup. It is the heir of
// Lifecycle.OnFinalize(out, runErr) and is DISTINCT from the activation teardown
// above. The brick body writes none of this; the Runner guarantees it.
type OnFinalizer interface {
	OnFinalize(ctx UnitContext, out map[string]any, runErr error) map[string]any
}

// ParamProcessor is an OPTIONAL normalize/resolve hook (the heir of
// OnProcessParameters). Most bricks omit it: the wrapper already validates and
// coerces `in` against MethodDef.Params (with CEL Validate) BEFORE Invoke.
type ParamProcessor interface {
	OnProcessParameters(ctx UnitContext, in map[string]any) (map[string]any, error)
}
