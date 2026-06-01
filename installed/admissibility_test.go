package installed

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/PlugfyOS/plugfy-common/spi"
)

// baseSpine is a representative installed platform spine used across the
// admissibility golden tests: a Desktop edition on Windows 11 with embedded
// infra routes.
func baseSpine() PlatformSpine {
	return PlatformSpine{
		Schema:        "system-layout.v1",
		Release:       "2026.06.0",
		Engine:        "1.0.0",
		UISchema:      "v1",
		ABI:           "1.0.0",
		HostOS:        "windows",
		HostOSVersion: "11",
		Edition:       "desktop",
		EventBus:      "inproc",
		Database:      "embedded",
	}
}

// installedCaps is a representative installed-capability set (storage 1.2.0,
// identity 1.0.0) so `requires` checks have something to resolve against.
func installedCaps() []InstalledModule {
	return []InstalledModule{
		{ID: "com.plugfy.system.storage", Capability: "storage", Version: "1.2.0"},
		{ID: "com.plugfy.system.identity", Capability: "identity", Version: "1.0.0"},
	}
}

func TestAdmissible_GoldenRows(t *testing.T) {
	spine := baseSpine()
	caps := NewCapabilityIndex(installedCaps())

	type row struct {
		name       string
		spec       CompatSpec
		admissible bool
		axis       string // expected failing axis when blocked
	}
	rows := []row{
		{
			name: "fully compatible candidate",
			spec: CompatSpec{
				Platform: ">=2026.06.0 <2027.0.0",
				Engine:   ">=1.0.0",
				UISchema: "v1",
				HostOS:   []string{"windows>=10", "macos>=13", "linux"},
				Edition:  []string{"desktop", "cloud"},
				Requires: []spi.CapabilityRequirement{{Capability: "storage", Version: ">=1.0.0"}},
				Channel:  ChannelStable,
			},
			admissible: true,
		},
		{
			name:       "empty spec admits (no constraints)",
			spec:       CompatSpec{},
			admissible: true,
		},
		{
			name:       "platform below floor -> blocked on platform",
			spec:       CompatSpec{Platform: ">=2026.07.0"},
			admissible: false,
			axis:       AxisPlatform,
		},
		{
			name:       "platform above ceiling -> blocked on platform",
			spec:       CompatSpec{Platform: ">=2025.0.0 <2026.06.0"},
			admissible: false,
			axis:       AxisPlatform,
		},
		{
			name:       "malformed platform range -> blocked on platform",
			spec:       CompatSpec{Platform: ">=garbage"},
			admissible: false,
			axis:       AxisPlatform,
		},
		{
			name:       "engine range excludes installed -> blocked on engine",
			spec:       CompatSpec{Engine: ">=2.0.0"},
			admissible: false,
			axis:       AxisEngine,
		},
		{
			name:       "uischema mismatch -> blocked on uischema",
			spec:       CompatSpec{UISchema: "v2"},
			admissible: false,
			axis:       AxisUISchema,
		},
		{
			name:       "abi range excludes installed -> blocked on abi",
			spec:       CompatSpec{ABI: ">=2.0.0"},
			admissible: false,
			axis:       AxisABI,
		},
		{
			name:       "hostOS not supported -> blocked on hostOS",
			spec:       CompatSpec{HostOS: []string{"macos>=13", "linux"}},
			admissible: false,
			axis:       AxisHostOS,
		},
		{
			name:       "hostOS version too low -> blocked on hostOS",
			spec:       CompatSpec{HostOS: []string{"windows>=12"}},
			admissible: false,
			axis:       AxisHostOS,
		},
		{
			name:       "edition not supported -> blocked on edition",
			spec:       CompatSpec{Edition: []string{"cloud", "enterprise"}},
			admissible: false,
			axis:       AxisEdition,
		},
		{
			name:       "infra requires nats on inproc bus -> blocked on infra",
			spec:       CompatSpec{Infra: InfraSupport{EventBus: []string{"nats"}}},
			admissible: false,
			axis:       AxisInfra,
		},
		{
			name:       "infra requires postgres on embedded db -> blocked on infra",
			spec:       CompatSpec{Infra: InfraSupport{Database: []string{"postgres"}}},
			admissible: false,
			axis:       AxisInfra,
		},
		{
			name:       "missing required capability -> blocked on requires",
			spec:       CompatSpec{Requires: []spi.CapabilityRequirement{{Capability: "billing", Version: ">=1.0.0"}}},
			admissible: false,
			axis:       AxisRequires,
		},
		{
			name:       "required capability too old -> blocked on requires",
			spec:       CompatSpec{Requires: []spi.CapabilityRequirement{{Capability: "storage", Version: ">=2.0.0"}}},
			admissible: false,
			axis:       AxisRequires,
		},
		{
			name:       "beta candidate invisible on stable -> blocked on channel",
			spec:       CompatSpec{Channel: ChannelBeta},
			admissible: false,
			axis:       AxisChannel,
		},
	}

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			ok, axis, reason := AdmissibleAxis(r.spec, spine, caps, ChannelStable)
			if ok != r.admissible {
				t.Fatalf("Admissible=%v want %v (reason: %s)", ok, r.admissible, reason)
			}
			// The two-value Admissible must agree with the axis form.
			ok2, reason2 := Admissible(r.spec, spine, caps, ChannelStable)
			if ok2 != ok || reason2 != reason {
				t.Fatalf("Admissible disagrees with AdmissibleAxis: (%v,%q) vs (%v,%q)", ok2, reason2, ok, reason)
			}
			if !r.admissible {
				if axis != r.axis {
					t.Fatalf("Axis=%q want %q (reason: %s)", axis, r.axis, reason)
				}
				if reason == "" {
					t.Fatal("blocked verdict must carry a reason")
				}
			} else {
				if axis != "" || reason != "" {
					t.Fatalf("admissible verdict must carry no axis/reason, got axis=%q reason=%q", axis, reason)
				}
			}
		})
	}
}

func TestAdmissible_ChannelLadder(t *testing.T) {
	spine := baseSpine()

	// On beta, a stable and a beta candidate are visible; nightly is not.
	if ok, _ := Admissible(CompatSpec{Channel: ChannelStable}, spine, nil, ChannelBeta); !ok {
		t.Error("stable must be visible under beta")
	}
	if ok, _ := Admissible(CompatSpec{Channel: ChannelBeta}, spine, nil, ChannelBeta); !ok {
		t.Error("beta must be visible under beta")
	}
	if ok, _ := Admissible(CompatSpec{Channel: ChannelNightly}, spine, nil, ChannelBeta); ok {
		t.Error("nightly must NOT be visible under beta")
	}
	if ok, _ := Admissible(CompatSpec{Channel: ChannelNightly}, spine, nil, ChannelNightly); !ok {
		t.Error("nightly must be visible under nightly")
	}
}

func TestAdmissible_EmptyChannelNormalizesToStable(t *testing.T) {
	spine := baseSpine()
	// An empty selected channel normalizes to stable: a beta candidate is then
	// invisible, blocked on channel.
	if ok, _ := Admissible(CompatSpec{Channel: ChannelBeta}, spine, nil, ""); ok {
		t.Error("beta must be invisible under empty (==stable) channel")
	}
	if ok, _ := Admissible(CompatSpec{Channel: ChannelStable}, spine, nil, ""); !ok {
		t.Error("stable must be visible under empty (==stable) channel")
	}
}

func TestAdmissible_EmptySpineAxisSkips(t *testing.T) {
	// When the spine does not pin an axis, a concrete range on that axis is
	// accepted (no installed value to violate).
	spine := baseSpine()
	spine.Engine = ""
	if ok, reason := Admissible(CompatSpec{Engine: ">=1.0.0"}, spine, nil, ChannelStable); !ok {
		t.Errorf("engine range should pass when spine pins no engine, got reason %q", reason)
	}
}

func TestCapabilityIndex_KeepsHighest(t *testing.T) {
	idx := NewCapabilityIndex([]InstalledModule{
		{Capability: "storage", Version: "1.0.0"},
		{Capability: "storage", Version: "1.5.0"},
		{Capability: "storage", Version: "1.2.0"},
		{ID: "no-cap"}, // skipped (no capability)
	})
	if idx["storage"] != "1.5.0" {
		t.Errorf("index should keep highest storage = 1.5.0, got %q", idx["storage"])
	}
	if _, ok := idx[""]; ok {
		t.Error("modules with no capability must be skipped")
	}
}

func TestAdmissibleVersions_SortedNewestFirst(t *testing.T) {
	spine := baseSpine()
	versions := map[string]CompatSpec{
		"1.0.0": {},
		"1.2.0": {},
		"1.1.0": {Edition: []string{"cloud"}}, // blocked on edition
	}
	got := AdmissibleVersions(versions, spine, nil, ChannelStable)
	if len(got) != 2 {
		t.Fatalf("expected 2 admissible, got %d (%v)", len(got), got)
	}
	if got[0] != "1.2.0" || got[1] != "1.0.0" {
		t.Errorf("expected newest-first [1.2.0 1.0.0], got %v", got)
	}
}

func TestRangeContains(t *testing.T) {
	cases := []struct {
		rng, ver string
		want     bool
	}{
		{"", "9.9.9", true},                // empty range admits everything
		{">=1.0.0", "1.0.0", true},         // inclusive lower bound
		{">=1.0.0 <2.0.0", "1.5.0", true},  // within band
		{">=1.0.0 <2.0.0", "2.0.0", false}, // exclusive upper bound
		{">=garbage", "1.0.0", false},      // malformed range admits nothing
		{"1.2", "1.2.0", true},             // trailing zero equivalence, exact
	}
	for _, c := range cases {
		if got := RangeContains(c.rng, c.ver); got != c.want {
			t.Errorf("RangeContains(%q, %q) = %v, want %v", c.rng, c.ver, got, c.want)
		}
	}
}

func TestCompatSpecJSONRoundTrip(t *testing.T) {
	in := CompatSpec{
		Platform: ">=2026.06.0 <2027.0.0",
		Engine:   ">=1.0.0",
		UISchema: "v1",
		ABI:      ">=1.0.0",
		HostOS:   []string{"windows>=10", "linux"},
		Edition:  []string{"desktop"},
		Infra:    InfraSupport{EventBus: []string{"inproc", "nats"}, Database: []string{"embedded"}},
		Requires: []spi.CapabilityRequirement{{Capability: "storage", Version: ">=1.0.0"}},
		Channel:  ChannelStable,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out CompatSpec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("CompatSpec round trip mismatch:\n in = %+v\nout = %+v", in, out)
	}
}
