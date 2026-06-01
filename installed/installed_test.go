package installed

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func sampleModule() InstalledModule {
	return InstalledModule{
		ID:         "com.plugfy.installed",
		Name:       "Installed Registry",
		Layer:      "L7",
		Capability: "registry",
		Version:    "1.4.2",
		Hash:       "sha256:abcd",
		Source:     "oci://registry.plugfy.io/installed:1.4.2",
		Channel:    "stable",
		RenderPath: RenderDeclarative,
		Compatibility: Compatibility{
			PlatformBuild: "1.1.0",
			HostOS:        "linux",
			Edition:       "enterprise",
			Infra:         "postgres",
		},
		Pinned: true,
		Deps: []ModuleRef{
			{ID: "com.plugfy.persistence", Version: "^1.1"},
		},
		Signature: &Sig{Algorithm: "ed25519", KeyID: "key-1", Value: "c2ln"},
	}
}

func TestInstalledModuleJSONRoundTrip(t *testing.T) {
	in := sampleModule()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out InstalledModule
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip mismatch:\n in = %+v\nout = %+v", in, out)
	}
}

func TestInstalledModuleWireFieldNames(t *testing.T) {
	b, err := json.Marshal(sampleModule())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{
		"id", "name", "layer", "capability", "version", "hash", "source",
		"channel", "renderPath", "compatibility", "pinned",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing wire field %q in %s", key, b)
		}
	}
	if got := m["renderPath"]; got != string(RenderDeclarative) {
		t.Errorf("renderPath token = %v, want %q", got, RenderDeclarative)
	}
}

func TestRenderPathTokens(t *testing.T) {
	if string(RenderDeclarative) != "declarative" {
		t.Errorf("RenderDeclarative = %q, want %q", RenderDeclarative, "declarative")
	}
	if string(RenderCustom) != "custom" {
		t.Errorf("RenderCustom = %q, want %q", RenderCustom, "custom")
	}
	if RenderPath("bogus").Valid() {
		t.Error("unknown render path reported valid")
	}
}

func TestInstalledModuleValidate(t *testing.T) {
	if err := sampleModule().Validate(); err != nil {
		t.Fatalf("valid module rejected: %v", err)
	}

	bad := sampleModule()
	bad.Version = "v1.4.2"
	if err := bad.Validate(); err == nil {
		t.Error("expected leading-v version to be rejected")
	}

	bad = sampleModule()
	bad.RenderPath = "weird"
	if err := bad.Validate(); err == nil {
		t.Error("expected unknown renderPath to be rejected")
	}

	bad = sampleModule()
	bad.ID = ""
	if err := bad.Validate(); err == nil {
		t.Error("expected empty id to be rejected")
	}
}

func TestInstalledIndexParseAndValidate(t *testing.T) {
	ix := InstalledIndex{sampleModule()}
	b, err := json.Marshal(ix)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := ParseIndex(strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("ParseIndex: %v", err)
	}
	if !reflect.DeepEqual(ix, got) {
		t.Fatalf("ParseIndex mismatch:\n in = %+v\nout = %+v", ix, got)
	}
	if m, ok := got.Find("com.plugfy.installed"); !ok || m.Version != "1.4.2" {
		t.Errorf("Find returned (%+v, %v)", m, ok)
	}

	// Duplicate IDs must be rejected.
	dup := InstalledIndex{sampleModule(), sampleModule()}
	if err := dup.Validate(); err == nil {
		t.Error("expected duplicate id to be rejected")
	}

	// Unknown fields must be rejected by the parser.
	if _, err := ParseIndex(strings.NewReader(`[{"id":"x","version":"1.0.0","renderPath":"custom","extra":true}]`)); err == nil {
		t.Error("expected unknown field to be rejected")
	}
}

func TestSystemLayoutValidateAndParse(t *testing.T) {
	l := SystemLayout{
		Root:  "/opt/plugfy",
		Areas: map[Area]string{},
	}
	for _, a := range KnownAreas {
		l.Areas[a] = "/opt/plugfy/" + string(a)
	}
	if err := l.Validate(); err != nil {
		t.Fatalf("valid layout rejected: %v", err)
	}
	if p, ok := l.Path(System); !ok || p != "/opt/plugfy/system" {
		t.Errorf("Path(System) = (%q, %v)", p, ok)
	}

	b, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := ParseLayout(strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("ParseLayout: %v", err)
	}
	if !reflect.DeepEqual(l, got) {
		t.Fatalf("ParseLayout mismatch")
	}

	// Missing an area must fail validation.
	missing := SystemLayout{Root: "/opt/plugfy", Areas: map[Area]string{System: "/x"}}
	if err := missing.Validate(); err == nil {
		t.Error("expected unmapped area to be rejected")
	}
}

func TestPlatformSpineValidateAndParse(t *testing.T) {
	s := PlatformSpine{
		Schema:   "system-layout.v1",
		Release:  "1.1.0",
		Engine:   "0.9.0",
		UISchema: "1.0.0",
		ABI:      "1.1.0",
		HostOS:   "linux",
		Edition:  "enterprise",
		EventBus: "nats",
		Database: "postgres",
		Channel:  "stable",
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("valid spine rejected: %v", err)
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := ParseSpine(strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("ParseSpine: %v", err)
	}
	if !reflect.DeepEqual(s, got) {
		t.Fatalf("ParseSpine mismatch")
	}

	bad := s
	bad.Release = "v1.1.0"
	if err := bad.Validate(); err == nil {
		t.Error("expected leading-v release to be rejected")
	}
}
