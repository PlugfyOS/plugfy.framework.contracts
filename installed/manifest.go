// Package installed defines installed-manifest.v1 and system-layout.v1: the
// single shape ops-packaging WRITES when it materializes a PlugfyOS install and
// platform-api READS to serve the running system's composition.
//
// It is also the home of the render-path and compatibility contract surfaced to
// the UX (Wave 0 / S2): each installed module declares whether its UI is
// rendered declaratively from a UI schema or by custom code (see [RenderPath],
// whose tokens match the ui-engine Dart enum), plus the platform build / host
// OS / edition / infrastructure it was built against ([Compatibility]).
//
// This package is pure data plus parse/validate helpers; it imports no backend
// and no HTTP, so the manifest can be written by the installer, read by the
// API host, and validated by either side from the same frozen shapes.
package installed

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RenderPath declares how a module's UI is produced. The tokens are identical
// to the ui-engine Dart enum so a manifest written by ops-packaging and a
// front-end reading it agree on the same vocabulary without translation.
type RenderPath string

const (
	// RenderDeclarative — the UI is rendered from a declarative UI schema
	// (the micro-front-end registry) with no module-supplied code.
	RenderDeclarative RenderPath = "declarative"
	// RenderCustom — the module supplies custom render code; the platform
	// loads it rather than driving a declarative schema.
	RenderCustom RenderPath = "custom"
)

// Valid reports whether the render path is one of the known tokens.
func (r RenderPath) Valid() bool {
	switch r {
	case RenderDeclarative, RenderCustom:
		return true
	default:
		return false
	}
}

// Compatibility records the environment a module was built and validated
// against. The API host compares it to the running [PlatformSpine] to flag
// modules that may not be safe to run after a platform upgrade or host change.
type Compatibility struct {
	// PlatformBuild is the platform build the module targets
	// (matches PlatformSpine.Release / .Build the installer recorded).
	PlatformBuild string `json:"platformBuild"`
	// HostOS is the host operating system family the module supports
	// (e.g. "linux", "windows", "darwin"). Empty means OS-agnostic.
	HostOS string `json:"hostOS,omitempty"`
	// Edition is the PlugfyOS edition the module targets
	// (e.g. "local", "cloud", "enterprise").
	Edition string `json:"edition,omitempty"`
	// Infra is the infrastructure profile the module requires
	// (e.g. "postgres", "sqlite", "nats"). Empty means infra-agnostic.
	Infra string `json:"infra,omitempty"`
}

// ModuleRef is a dependency reference: a module ID with a SemVer constraint.
type ModuleRef struct {
	// ID is the referenced module's reverse-DNS identifier.
	ID string `json:"id"`
	// Version is the SemVer constraint the dependant requires
	// (e.g. "1.2.0" or "^1.2"). No leading "v".
	Version string `json:"version"`
}

// Sig is the detached signature over a module's content hash, recorded by the
// installer so the API host can verify provenance without re-hashing payloads.
type Sig struct {
	// Algorithm is the signature algorithm identifier (e.g. "ed25519").
	Algorithm string `json:"algorithm"`
	// KeyID identifies the signing key (e.g. a key fingerprint).
	KeyID string `json:"keyId"`
	// Value is the base64-encoded signature over the module Hash.
	Value string `json:"value"`
}

// InstalledModule is the single record describing one installed module, written
// by ops-packaging and read by platform-api. It is the row of an
// [InstalledIndex].
type InstalledModule struct {
	// ID is the module's reverse-DNS identifier
	// (e.g. "com.plugfy.installed").
	ID string `json:"id"`
	// Name is the human-readable display name.
	Name string `json:"name"`
	// Layer is the architecture layer the module occupies
	// (e.g. "L2", "L7").
	Layer string `json:"layer"`
	// Capability is the capability the module provides
	// (e.g. "persistence", "api", "registry").
	Capability string `json:"capability"`
	// Version is the module's SemVer with NO leading "v"
	// (e.g. "1.4.2").
	Version string `json:"version"`
	// Hash is the content hash of the installed payload
	// (e.g. "sha256:...").
	Hash string `json:"hash"`
	// Source is where the module was installed from
	// (a registry URL, a local path, an OCI ref).
	Source string `json:"source"`
	// Channel is the release channel the module came from
	// (e.g. "stable", "beta", "edge").
	Channel string `json:"channel"`
	// RenderPath declares how the module's UI is rendered.
	RenderPath RenderPath `json:"renderPath"`
	// Compatibility records the environment the module was built against.
	Compatibility Compatibility `json:"compatibility"`
	// Pinned, when true, freezes the module at this version: the updater
	// MUST NOT auto-upgrade it.
	Pinned bool `json:"pinned"`
	// Deps are the module's declared dependencies.
	Deps []ModuleRef `json:"deps,omitempty"`
	// Signature is the detached provenance signature, when present.
	Signature *Sig `json:"signature,omitempty"`
}

// Validate checks the required fields and enumerations of a single module
// record. It returns the first problem found, identifying the offending field.
func (m InstalledModule) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("installed: module id is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("installed: module %q version is required", m.ID)
	}
	if strings.HasPrefix(m.Version, "v") || strings.HasPrefix(m.Version, "V") {
		return fmt.Errorf("installed: module %q version %q must not carry a leading 'v'", m.ID, m.Version)
	}
	if !m.RenderPath.Valid() {
		return fmt.Errorf("installed: module %q has unknown renderPath %q", m.ID, m.RenderPath)
	}
	for i, d := range m.Deps {
		if strings.TrimSpace(d.ID) == "" {
			return fmt.Errorf("installed: module %q dep[%d] id is required", m.ID, i)
		}
	}
	return nil
}

// InstalledIndex is the ordered set of installed modules — the document
// ops-packaging writes and platform-api reads to describe the whole install.
type InstalledIndex []InstalledModule

// Validate checks every module and rejects duplicate IDs.
func (ix InstalledIndex) Validate() error {
	seen := make(map[string]struct{}, len(ix))
	for i := range ix {
		if err := ix[i].Validate(); err != nil {
			return err
		}
		id := ix[i].ID
		if _, dup := seen[id]; dup {
			return fmt.Errorf("installed: duplicate module id %q", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// Find returns the module with the given ID and whether it was present.
func (ix InstalledIndex) Find(id string) (InstalledModule, bool) {
	for i := range ix {
		if ix[i].ID == id {
			return ix[i], true
		}
	}
	return InstalledModule{}, false
}

// ParseIndex decodes an InstalledIndex from JSON and validates it. It rejects
// unknown fields so a manifest written by a newer writer than this reader
// understands surfaces as an error rather than silent data loss.
func ParseIndex(r io.Reader) (InstalledIndex, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var ix InstalledIndex
	if err := dec.Decode(&ix); err != nil {
		return nil, fmt.Errorf("installed: decode index: %w", err)
	}
	if err := ix.Validate(); err != nil {
		return nil, err
	}
	return ix, nil
}
