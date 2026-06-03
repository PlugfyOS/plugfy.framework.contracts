package core

// This file declares the commons-HOME twins of the platform-runtime/manifest
// supply-chain and composition types the UnitDescriptor carries. commons cannot
// import platform-runtime (that module depends on commons; importing it would be
// a module cycle and would violate the DEFINITION-layer "no upward dependency"
// invariant). Each type below mirrors its platform-runtime/manifest counterpart
// VERBATIM — identical field names and JSON tags — so the host projects between
// them losslessly at the composition root. They are carried, never interpreted,
// in Phase 0.

// Capability is a contract a Unit provides (OSGi-style). Twin of
// platform-runtime/manifest.Capability.
type Capability struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Requirement is a capability a Unit consumes, with an OSGi-style version range.
// Twin of platform-runtime/manifest.Requirement.
type Requirement struct {
	Capability   string `json:"capability"`
	VersionRange string `json:"versionRange"`
	Filter       string `json:"filter,omitempty"`
	Optional     bool   `json:"optional,omitempty"`
}

// Signing is the supply-chain policy (verify-before-install). Twin of
// platform-runtime/manifest.Signing.
type Signing struct {
	Required    bool   `json:"required"`
	Mode        string `json:"mode,omitempty"` // keyless | key
	Issuer      string `json:"issuer,omitempty"`
	Identity    string `json:"identity,omitempty"`
	Certificate string `json:"certificate,omitempty"`
	Provenance  string `json:"provenance,omitempty"`
}

// UnitState declares the persistent state fields a Unit owns between executions
// (CQRS / apps-own-data). Twin of platform-runtime/manifest.UnitState.
type UnitState struct {
	// Persistence chooses the scope under which the state is keyed: "org",
	// "org+project" (default), "org+project+instance", "org+project+user".
	Persistence string `json:"persistence,omitempty"`
	// Fields lists the state attributes the Unit owns.
	Fields []StateField `json:"fields,omitempty"`
}

// StateField is a single declarative state attribute. Twin of
// platform-runtime/manifest.StateField.
type StateField struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string|integer|float|boolean|timestamp|json
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
	// ResetOn names a lifecycle event that resets the field to its default:
	// "success" | "failure" | "" (manual).
	ResetOn string `json:"resetOn,omitempty"`
}

// SettingsContribution is a Settings section a Unit contributes. Twin of
// platform-runtime/manifest.SettingsContribution.
type SettingsContribution struct {
	Section  string            `json:"section"`
	Title    string            `json:"title,omitempty"`
	Icon     string            `json:"icon,omitempty"`
	Priority int               `json:"priority,omitempty"`
	Fields   []FieldDescriptor `json:"fields,omitempty"`
}

// FieldDescriptor describes a single settings field a Unit contributes. The Type
// drives the Settings renderer. Twin of platform-runtime/manifest.FieldDescriptor.
// Its closed Type set is the source of the ParamType values reused by ParamDef.
type FieldDescriptor struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Default     any      `json:"default,omitempty"`
	Label       string   `json:"label,omitempty"`
	Description string   `json:"description,omitempty"`
	Options     []string `json:"options,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
}

// ThemeContribution is a theme a Unit registers platform-wide (or scoped). Twin
// of platform-runtime/manifest.ThemeContribution.
type ThemeContribution struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Scope      string            `json:"scope,omitempty"` // default "platform"
	Version    string            `json:"version"`
	Tokens     map[string]string `json:"tokens,omitempty"`
	Components map[string]any    `json:"components,omitempty"`
}
