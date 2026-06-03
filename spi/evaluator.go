package spi

// Evaluator is the sandboxed expression-engine PORT the platform's dynamic
// predicates depend on: edge guards, If/Switch conditions, ForEach collection
// expressions, ${...} template interpolation, and — for the core.Unit spine —
// ParamDef.Validate and method/unit Visibility predicates.
//
// It is declared here, on the L1 baseplate (commons), so that consumers in the
// DEFINITION layer (notably the future core.Runner under spi/core) can validate
// inputs and gate visibility by depending on this ABSTRACTION alone, without
// importing any EXECUTION/OPERATION engine. The CEL-backed IMPLEMENTATION is
// injected at the composition root (the same sandboxed engine the pipeline
// already uses), so there is one CEL behind one port — never two. This is
// dependency inversion: commons declares the port; the platform supplies the
// adapter; nothing in commons reaches sideways or upward to evaluate an
// expression.
//
// The method set mirrors the engine-side evaluator port verbatim so the single
// CEL adapter satisfies both without a fork.
type Evaluator interface {
	// Eval compiles and evaluates a single expression against the scope,
	// returning the unwrapped Go value.
	Eval(source string, scope map[string]any) (any, error)
	// EvalBool evaluates an expression expected to yield a boolean.
	EvalBool(source string, scope map[string]any) (bool, error)
	// Interpolate replaces every ${...} placeholder in tmpl with the textual
	// rendering of the contained expression's result.
	Interpolate(tmpl string, scope map[string]any) (string, error)
}
