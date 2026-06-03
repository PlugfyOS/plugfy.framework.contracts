package installed

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/PlugfyOS/plugfy.framework.contracts/spi"
)

// HostManifestSchema is the schema identifier of the per-host dependency
// manifest. It is the contract a host (a desktop, a server, a cloud tenant)
// publishes to declare what the platform MUST provide for that host to be
// admissible — the capabilities it needs (by range) and the specific modules it
// pins (by id + range). The resolver/updater reads it to seed a plan; the
// admissibility matrix evaluates it against the running [PlatformSpine].
const HostManifestSchema = "host-manifest.v1"

// HostRequires is the dependency set a [HostManifest] declares: capability
// requirements (capability + SemVer range) and specific module pins (id +
// version range). Both lists fold into the same admissibility evaluation: the
// capability requirements are resolved against the installed capability index
// (Minimal-Version-Selection), and the specific pins name exact modules the
// host requires present at a compatible version.
type HostRequires struct {
	// Platform lists the capability dependencies the host needs from the
	// platform, each a capability name plus the SemVer range it admits. Reuses
	// the cross-cutting [spi.CapabilityRequirement] verbatim.
	Platform []spi.CapabilityRequirement `json:"platform,omitempty"`
	// Specific lists modules the host pins by id and version range. Reuses the
	// installed [ModuleRef] verbatim.
	Specific []ModuleRef `json:"specific,omitempty"`
}

// HostManifest is the host-manifest.v1 shape: a per-host dependency manifest a
// host publishes so the platform can resolve and verify the modules that host
// requires. It is pure data plus parse/validate helpers — no backend, no HTTP —
// so the host can WRITE it and the resolver/updater can READ and evaluate it
// from the same frozen shape.
type HostManifest struct {
	// Schema is the manifest schema identifier; MUST equal [HostManifestSchema].
	Schema string `json:"schema"`
	// Host is the host's reverse-DNS identifier (e.g. "io.plugfy.host.lab-01").
	Host string `json:"host"`
	// Requires is the dependency set the host declares.
	Requires HostRequires `json:"requires"`
}

// Validate checks the required fields and the well-formedness of the declared
// dependencies. It returns the first problem found, identifying the offending
// field. An empty Schema defaults to [HostManifestSchema] for the caller's
// convenience is NOT applied here: a manifest read off the wire must declare its
// schema explicitly so a mismatched producer surfaces as an error.
func (h HostManifest) Validate() error {
	if strings.TrimSpace(h.Schema) == "" {
		return fmt.Errorf("installed: host manifest schema is required")
	}
	if h.Schema != HostManifestSchema {
		return fmt.Errorf("installed: host manifest schema %q must equal %q", h.Schema, HostManifestSchema)
	}
	if strings.TrimSpace(h.Host) == "" {
		return fmt.Errorf("installed: host manifest host id is required")
	}
	for i, req := range h.Requires.Platform {
		if strings.TrimSpace(req.Capability) == "" {
			return fmt.Errorf("installed: host %q requires.platform[%d] capability is required", h.Host, i)
		}
	}
	for i, ref := range h.Requires.Specific {
		if strings.TrimSpace(ref.ID) == "" {
			return fmt.Errorf("installed: host %q requires.specific[%d] id is required", h.Host, i)
		}
	}
	return nil
}

// ParseHostManifest decodes a HostManifest from JSON and validates it. It
// rejects unknown fields so a manifest written by a newer producer than this
// reader understands surfaces as an error rather than silent data loss.
func ParseHostManifest(r io.Reader) (HostManifest, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var h HostManifest
	if err := dec.Decode(&h); err != nil {
		return HostManifest{}, fmt.Errorf("installed: decode host manifest: %w", err)
	}
	if err := h.Validate(); err != nil {
		return HostManifest{}, err
	}
	return h, nil
}
