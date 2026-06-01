package installed

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// PlatformSpine is the system-layout.v1 description of the running platform
// itself: the fixed coordinates ops-packaging records at install time and
// platform-api reports as the system identity every installed module's
// [Compatibility] is checked against.
type PlatformSpine struct {
	// Schema is the spine schema identifier (e.g. "system-layout.v1").
	Schema string `json:"schema"`
	// Release is the platform release/version (SemVer, no leading "v").
	Release string `json:"release"`
	// Engine is the runtime engine version powering pipelines.
	Engine string `json:"engine"`
	// UISchema is the UI-schema contract version the front-end speaks.
	UISchema string `json:"uiSchema"`
	// ABI is the plugfy-common ABI version the platform was built against
	// (e.g. "1.1.0").
	ABI string `json:"abi"`
	// HostOS is the host operating system family ("linux", "windows",
	// "darwin").
	HostOS string `json:"hostOS"`
	// HostOSVersion is the host OS version the install runs on (e.g. "11" for
	// Windows 11, "13" for macOS 13). Empty means the host does not report a
	// version; a versioned hostOS constraint (e.g. "windows>=10") then fails
	// while a bare-OS constraint ("linux") still matches. Used by [Admissible]
	// to evaluate versioned hostOS constraints.
	HostOSVersion string `json:"hostOSVersion,omitempty"`
	// Edition is the active PlugfyOS edition ("local", "cloud",
	// "enterprise").
	Edition string `json:"edition"`
	// EventBus is the active event-bus backend ("inproc", "nats", …).
	EventBus string `json:"eventBus"`
	// Database is the active database backend ("postgres", "sqlite").
	Database string `json:"database"`
	// Channel is the release channel the platform tracks ("stable", …).
	Channel string `json:"channel"`
}

// Validate checks the required identity fields of the spine.
func (s PlatformSpine) Validate() error {
	if strings.TrimSpace(s.Schema) == "" {
		return fmt.Errorf("installed: spine schema is required")
	}
	if strings.TrimSpace(s.Release) == "" {
		return fmt.Errorf("installed: spine release is required")
	}
	if strings.HasPrefix(s.Release, "v") || strings.HasPrefix(s.Release, "V") {
		return fmt.Errorf("installed: spine release %q must not carry a leading 'v'", s.Release)
	}
	return nil
}

// Area names a logical directory of a PlugfyOS install rooted at
// [SystemLayout.Root]. The set of areas is fixed: ops-packaging materializes
// exactly these directories and platform-api resolves paths through them.
type Area string

const (
	// System holds the platform spine and core binaries.
	System Area = "system"
	// Capabilities holds installed capability modules.
	Capabilities Area = "capabilities"
	// Drivers holds installed L2 provider drivers.
	Drivers Area = "drivers"
	// Themes holds installed UI themes.
	Themes Area = "themes"
	// Apps holds installed applications.
	Apps Area = "apps"
	// Data holds persistent application data.
	Data Area = "data"
	// Logs holds platform and module logs.
	Logs Area = "logs"
	// Var holds mutable runtime state (pids, sockets, caches).
	Var Area = "var"
	// SourcesD holds drop-in source/registry definitions (sources.d).
	SourcesD Area = "sources.d"
)

// KnownAreas is the canonical, ordered set of layout areas. Iteration order is
// stable so callers materializing or auditing the layout get deterministic
// output.
var KnownAreas = []Area{
	System, Capabilities, Drivers, Themes, Apps, Data, Logs, Var, SourcesD,
}

// SystemLayout is the on-disk layout of an install: a root directory plus the
// concrete path of each [Area] beneath it. ops-packaging writes it; platform-api
// reads it to resolve where installed artifacts live.
type SystemLayout struct {
	// Root is the absolute install root directory.
	Root string `json:"root"`
	// Areas maps each layout Area to its concrete path. Paths MAY be
	// absolute or relative to Root; resolve through [SystemLayout.Path].
	Areas map[Area]string `json:"areas"`
}

// Validate checks the root is set and every known area is mapped.
func (l SystemLayout) Validate() error {
	if strings.TrimSpace(l.Root) == "" {
		return fmt.Errorf("installed: layout root is required")
	}
	for _, a := range KnownAreas {
		if strings.TrimSpace(l.Areas[a]) == "" {
			return fmt.Errorf("installed: layout area %q is unmapped", a)
		}
	}
	return nil
}

// Path returns the configured path for an area and whether it is mapped.
func (l SystemLayout) Path(a Area) (string, bool) {
	p, ok := l.Areas[a]
	return p, ok
}

// ParseLayout decodes a SystemLayout from JSON and validates it, rejecting
// unknown fields.
func ParseLayout(r io.Reader) (SystemLayout, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var l SystemLayout
	if err := dec.Decode(&l); err != nil {
		return SystemLayout{}, fmt.Errorf("installed: decode layout: %w", err)
	}
	if err := l.Validate(); err != nil {
		return SystemLayout{}, err
	}
	return l, nil
}

// ParseSpine decodes a PlatformSpine from JSON and validates it, rejecting
// unknown fields.
func ParseSpine(r io.Reader) (PlatformSpine, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var s PlatformSpine
	if err := dec.Decode(&s); err != nil {
		return PlatformSpine{}, fmt.Errorf("installed: decode spine: %w", err)
	}
	if err := s.Validate(); err != nil {
		return PlatformSpine{}, err
	}
	return s, nil
}
