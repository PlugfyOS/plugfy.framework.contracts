package installed

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/PlugfyOS/plugfy.framework.contracts/spi"
)

func sampleHostManifest() HostManifest {
	return HostManifest{
		Schema: HostManifestSchema,
		Host:   "io.plugfy.host.lab-01",
		Requires: HostRequires{
			Platform: []spi.CapabilityRequirement{
				{Capability: "storage", Version: ">=1.2.0"},
				{Capability: "identity"}, // any version
			},
			Specific: []ModuleRef{
				{ID: "com.plugfy.system.update", Version: "^1.0"},
			},
		},
	}
}

func TestHostManifestJSONRoundTrip(t *testing.T) {
	in := sampleHostManifest()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out HostManifest
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip mismatch:\n in = %+v\nout = %+v", in, out)
	}
}

func TestHostManifestWireFieldNames(t *testing.T) {
	b, err := json.Marshal(sampleHostManifest())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"schema", "host", "requires"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing wire field %q in %s", key, b)
		}
	}
	if got := m["schema"]; got != HostManifestSchema {
		t.Errorf("schema token = %v, want %q", got, HostManifestSchema)
	}
	req, _ := m["requires"].(map[string]any)
	if _, ok := req["platform"]; !ok {
		t.Errorf("missing requires.platform in %s", b)
	}
	if _, ok := req["specific"]; !ok {
		t.Errorf("missing requires.specific in %s", b)
	}
}

func TestHostManifestValidate(t *testing.T) {
	if err := sampleHostManifest().Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}

	bad := sampleHostManifest()
	bad.Schema = ""
	if err := bad.Validate(); err == nil {
		t.Error("expected empty schema to be rejected")
	}

	bad = sampleHostManifest()
	bad.Schema = "host-manifest.v2"
	if err := bad.Validate(); err == nil {
		t.Error("expected schema mismatch to be rejected")
	}

	bad = sampleHostManifest()
	bad.Host = ""
	if err := bad.Validate(); err == nil {
		t.Error("expected empty host id to be rejected")
	}

	bad = sampleHostManifest()
	bad.Requires.Platform = []spi.CapabilityRequirement{{Version: ">=1.0.0"}} // no capability
	if err := bad.Validate(); err == nil {
		t.Error("expected requires.platform with empty capability to be rejected")
	}

	bad = sampleHostManifest()
	bad.Requires.Specific = []ModuleRef{{Version: "^1.0"}} // no id
	if err := bad.Validate(); err == nil {
		t.Error("expected requires.specific with empty id to be rejected")
	}
}

func TestParseHostManifest(t *testing.T) {
	b, err := json.Marshal(sampleHostManifest())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := ParseHostManifest(strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("ParseHostManifest: %v", err)
	}
	if !reflect.DeepEqual(sampleHostManifest(), got) {
		t.Fatalf("ParseHostManifest mismatch:\n in = %+v\nout = %+v", sampleHostManifest(), got)
	}

	// Unknown fields must be rejected.
	if _, err := ParseHostManifest(strings.NewReader(
		`{"schema":"host-manifest.v1","host":"x","requires":{},"extra":true}`)); err == nil {
		t.Error("expected unknown field to be rejected")
	}

	// A wrong schema must be rejected by validation through the parser.
	if _, err := ParseHostManifest(strings.NewReader(
		`{"schema":"host-manifest.v9","host":"x","requires":{}}`)); err == nil {
		t.Error("expected wrong schema to be rejected by ParseHostManifest")
	}
}
