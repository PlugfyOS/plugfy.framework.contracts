package installed

import (
	"sort"
	"strconv"
	"strings"

	"github.com/PlugfyOS/plugfy-common/spi"
)

// This file LIFTS the pure, I/O-free version-compatibility admissibility matrix
// into the L1 baseplate. The matrix answers one question — "is this candidate
// admissible on this platform?" — across nine independent axes (platform,
// engine, uischema, abi, hostOS, edition, infra, requires, channel), returning
// a verdict and, when blocked, the first failing axis + a human-readable reason.
//
// It operates ONLY on a candidate compatibility block ([CompatSpec]), the
// running [PlatformSpine] and capability requirements ([spi.CapabilityRequirement]
// resolved through a [CapabilityIndex]) — all of which live on this baseplate —
// so the check has no business in any single consumer. The logic is identical
// (axis order, semantics, reasons) to the engine that previously lived only in
// the system-update unit; system-update SHOULD later re-export these from
// plugfy-common to dedupe (it currently keeps a private copy — see the note on
// [Admissible]). The engine imports no backend and no other unit, preserving the
// stdlib-only decoupling gate.

// ───────────────────────────── candidate inputs ─────────────────────────────

// Channel is a release-quality channel forming the nested visibility ladder
// stable ⊂ beta ⊂ nightly: a candidate published on a narrower channel is
// visible to a scope subscribed to a wider one, never the reverse.
type Channel string

// The release channels, narrowest (most stable) first.
const (
	// ChannelStable is the default production channel.
	ChannelStable Channel = "stable"
	// ChannelBeta is the pre-release channel; sees stable + beta candidates.
	ChannelBeta Channel = "beta"
	// ChannelNightly is the bleeding-edge channel; sees everything.
	ChannelNightly Channel = "nightly"
)

// rank returns the channel's width (stable=0 ⊂ beta=1 ⊂ nightly=2). An unknown
// channel is treated as stable so a malformed value never widens visibility.
func (c Channel) rank() int {
	switch c {
	case ChannelBeta:
		return 1
	case ChannelNightly:
		return 2
	default:
		return 0
	}
}

// Valid reports whether the channel is one of the three known channels.
func (c Channel) Valid() bool {
	switch c {
	case ChannelStable, ChannelBeta, ChannelNightly:
		return true
	default:
		return false
	}
}

// VisibleUnder reports whether a candidate published on channel c is visible to
// a scope subscribed to the selected channel sel, honoring the nested ladder
// stable ⊂ beta ⊂ nightly: a stable candidate is visible on every channel; a
// nightly candidate only on nightly. An empty candidate channel is treated as
// stable (always visible). An empty selected channel normalizes to stable.
func (c Channel) VisibleUnder(sel Channel) bool {
	cand := c
	if cand == "" {
		cand = ChannelStable
	}
	return cand.rank() <= sel.rank()
}

// InfraSupport declares which infrastructure routes a candidate needs/supports.
// The matrix rejects a candidate on a platform whose infra is outside these
// sets (e.g. a unit that requires "nats" is inadmissible on an "inproc"-only
// bus). An empty set on an axis means "any route" and passes.
type InfraSupport struct {
	// EventBus lists the supported event-bus routes (e.g. "inproc", "nats").
	// Empty means "any bus".
	EventBus []string `json:"eventbus,omitempty"`
	// Database lists the supported database routes (e.g. "embedded",
	// "postgres"). Empty means "any database".
	Database []string `json:"database,omitempty"`
}

// CompatSpec is the candidate compatibility{} block evaluated by [Admissible]:
// the per-candidate, RANGE-typed input to the version-compatibility matrix
// (distinct from [Compatibility], which is the single-valued install RECORD of
// what an already-installed module was built against). Every field is optional;
// an empty field means "no constraint on that axis", so a minimal candidate
// that only pins a platform range is fully expressible.
type CompatSpec struct {
	// Platform is the platform RELEASE range this candidate works with
	// (e.g. ">=2026.06.0 <2027.0.0"), validated against the spine release.
	Platform string `json:"platform,omitempty"`
	// Engine is the runtime-engine SemVer range (UI/engine-bound candidates).
	Engine string `json:"engine,omitempty"`
	// UISchema is the UI-schema contract version this candidate renders
	// against, matched for exact equality (e.g. "v1").
	UISchema string `json:"uischema,omitempty"`
	// ABI is the plugfy-common ABI SemVer range (units/drivers).
	ABI string `json:"abi,omitempty"`
	// HostOS lists the host-OS constraints (e.g. "windows>=10", "macos>=13",
	// "linux"). Empty means "any host OS". Satisfied when ANY entry matches.
	HostOS []string `json:"hostOS,omitempty"`
	// Edition lists the editions this candidate supports (e.g. "desktop",
	// "cloud", "enterprise"). Empty means "any edition".
	Edition []string `json:"edition,omitempty"`
	// Infra declares the infrastructure routes the candidate needs/supports.
	Infra InfraSupport `json:"infra,omitzero"`
	// Requires lists the capability dependencies (MVS), resolved against the
	// installed [CapabilityIndex]. Reuses [spi.CapabilityRequirement] verbatim.
	Requires []spi.CapabilityRequirement `json:"requires,omitempty"`
	// Channel is the release channel this version is published on
	// (stable ⊂ beta ⊂ nightly). Empty normalizes to stable.
	Channel Channel `json:"channel,omitempty"`
}

// ───────────────────────────── capability index ─────────────────────────────

// CapabilityIndex is the installed-capability lookup the matrix uses to resolve
// a candidate's `requires`: capability name → highest installed version that
// provides it. Built once from the installed set, it answers the MVS check
// "∃ installed capability ⊇ req".
type CapabilityIndex map[string]string

// NewCapabilityIndex builds the index from the installed modules, keeping the
// highest version per capability (a capability provided by more than one module
// resolves to the newest). Modules that provide no capability are skipped.
func NewCapabilityIndex(modules []InstalledModule) CapabilityIndex {
	idx := CapabilityIndex{}
	for i := range modules {
		m := modules[i]
		if m.Capability == "" {
			continue
		}
		if cur, ok := idx[m.Capability]; !ok || compareVersions(m.Version, cur) > 0 {
			idx[m.Capability] = m.Version
		}
	}
	return idx
}

// ───────────────────────────── the admissibility ────────────────────────────

// The nine admissibility axes, in evaluation order. They are the values
// [Admissible] reports as the first failing axis.
const (
	AxisPlatform = "platform"
	AxisEngine   = "engine"
	AxisUISchema = "uischema"
	AxisABI      = "abi"
	AxisHostOS   = "hostOS"
	AxisEdition  = "edition"
	AxisInfra    = "infra"
	AxisRequires = "requires"
	AxisChannel  = "channel"
)

// Admissible evaluates a candidate's compatibility against the running platform
// spine, the selected channel and the installed-capability index, returning
// whether it is admissible and, when not, the first failing axis + a
// human-readable reason. It is PURE and deterministic: the same inputs always
// yield the same verdict.
//
// It checks the nine axes in canonical order, short-circuiting on the first
// failure so the reported axis is the most fundamental violated constraint
// (platform first, channel last). An empty constraint on any axis means "no
// constraint" and passes; an empty installed value on a range axis means the
// spine does not pin that axis, so a concrete range cannot be violated and
// passes; a malformed (unsatisfiable) range fails on its axis.
//
// The boolean is the verdict; reason is empty when admissible and otherwise
// names the failing axis (one of the Axis* constants) in its text. To obtain
// the axis programmatically use [AdmissibleAxis].
//
// NOTE (dedupe): the system-update unit currently keeps a private copy of this
// engine (system-update/domain/matrix.go + range/version/hostos). Now that the
// pure logic lives on the baseplate over types that already live here, that
// copy SHOULD be retired in a later wave by re-exporting these symbols from
// plugfy-common. This wave does not touch system-update.
func Admissible(spec CompatSpec, spine PlatformSpine, caps CapabilityIndex, channel Channel) (bool, string) {
	ok, _, reason := evaluate(spec, spine, caps, channel)
	return ok, reason
}

// AdmissibleAxis is the axis-returning form of [Admissible]: on a blocked
// verdict it also returns the first failing axis (one of the Axis* constants),
// empty when admissible. Use it when the caller needs to branch on the axis
// rather than only present the reason.
func AdmissibleAxis(spec CompatSpec, spine PlatformSpine, caps CapabilityIndex, channel Channel) (ok bool, axis, reason string) {
	return evaluate(spec, spine, caps, channel)
}

// evaluate is the shared nine-axis core. It returns (admissible, axis, reason)
// where axis/reason are empty on admission.
func evaluate(spec CompatSpec, spine PlatformSpine, caps CapabilityIndex, channel Channel) (bool, string, string) {
	if channel == "" {
		channel = ChannelStable
	}
	if caps == nil {
		caps = CapabilityIndex{}
	}

	// 1. platform release ⊇ installed.platform.release.
	if ok, reason := checkRange(AxisPlatform, spec.Platform, spine.Release); !ok {
		return false, AxisPlatform, reason
	}
	// 2. engine range ⊇ installed.engine.version (if applicable).
	if spec.Engine != "" {
		if ok, reason := checkRange(AxisEngine, spec.Engine, spine.Engine); !ok {
			return false, AxisEngine, reason
		}
	}
	// 3. uischema = installed.uischema.version (exact, if applicable).
	if spec.UISchema != "" && spine.UISchema != "" && spec.UISchema != spine.UISchema {
		return false, AxisUISchema, "uischema " + quote(spec.UISchema) + " != installed " + quote(spine.UISchema)
	}
	// 4. abi range ⊇ installed.common.abi (if applicable).
	if spec.ABI != "" {
		if ok, reason := checkRange(AxisABI, spec.ABI, spine.ABI); !ok {
			return false, AxisABI, reason
		}
	}
	// 5. hostOS satisfied.
	if !hostOSSatisfied(spec.HostOS, spine.HostOS, spine.HostOSVersion) {
		return false, AxisHostOS, "host " + spine.HostOS + " " + spine.HostOSVersion + " not supported by " + joinSet(spec.HostOS)
	}
	// 6. edition ∈ candidate.edition.
	if len(spec.Edition) > 0 && !contains(spec.Edition, spine.Edition) {
		return false, AxisEdition, "edition " + quote(spine.Edition) + " not in " + joinSet(spec.Edition)
	}
	// 7. infra(installed) ∈ candidate.infra.
	if ok, reason := checkInfra(spec.Infra, spine); !ok {
		return false, AxisInfra, reason
	}
	// 8. ∀ req ∈ requires : ∃ installed capability ⊇ req.
	if ok, reason := checkRequires(spec.Requires, caps); !ok {
		return false, AxisRequires, reason
	}
	// 9. channel(candidate) visible under the selected channel.
	if !spec.Channel.VisibleUnder(channel) {
		ch := spec.Channel
		if ch == "" {
			ch = ChannelStable
		}
		return false, AxisChannel, "channel " + quote(string(ch)) + " not visible under " + quote(string(channel))
	}
	return true, "", ""
}

// checkRange evaluates a single SemVer-range axis: the candidate's range must
// CONTAIN the installed spine value. An empty installed value means the spine
// does not pin that axis, so any non-unsatisfiable range passes. An
// unsatisfiable (malformed) range fails.
func checkRange(axis, rangeStr, installed string) (bool, string) {
	r := parseRange(rangeStr)
	if r.unsatisfiable() {
		return false, axis + " range " + quote(rangeStr) + " is unsatisfiable"
	}
	if r.isAny() {
		return true, ""
	}
	if installed == "" {
		return true, ""
	}
	if !r.contains(parseVersion(installed)) {
		return false, "installed " + axis + " " + quote(installed) + " not in " + quote(rangeStr)
	}
	return true, ""
}

// checkInfra verifies the installed event-bus and database routes are supported
// by the candidate. An empty support set on an axis means "any route" and
// passes; a non-empty set must contain the installed route.
func checkInfra(infra InfraSupport, spine PlatformSpine) (bool, string) {
	if len(infra.EventBus) > 0 && spine.EventBus != "" && !contains(infra.EventBus, spine.EventBus) {
		return false, "eventbus " + quote(spine.EventBus) + " not in " + joinSet(infra.EventBus)
	}
	if len(infra.Database) > 0 && spine.Database != "" && !contains(infra.Database, spine.Database) {
		return false, "database " + quote(spine.Database) + " not in " + joinSet(infra.Database)
	}
	return true, ""
}

// checkRequires verifies every declared capability dependency resolves against
// an installed capability whose version satisfies the requirement's range (MVS).
func checkRequires(reqs []spi.CapabilityRequirement, caps CapabilityIndex) (bool, string) {
	for _, req := range reqs {
		have, ok := caps[req.Capability]
		if !ok {
			return false, "capability " + quote(req.Capability) + " not installed"
		}
		if req.Version != "" && !rangeContains(req.Version, have) {
			return false, "capability " + quote(req.Capability) + " installed " + quote(have) + " not in " + quote(req.Version)
		}
	}
	return true, ""
}

// AdmissibleVersions filters candidate compatibility specs to the admissible
// subset, keyed by the candidate version, and returns those versions sorted
// newest-first. It is the per-module FILTER+sort a planner's SELECT consumes:
// the resolver hands a module's offered versions and their specs, and gets back
// the admissible versions in preference order.
func AdmissibleVersions(versions map[string]CompatSpec, spine PlatformSpine, caps CapabilityIndex, channel Channel) []string {
	out := make([]string, 0, len(versions))
	for v, spec := range versions {
		if ok, _ := Admissible(spec, spine, caps, channel); ok {
			out = append(out, v)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return compareVersions(out[i], out[j]) > 0
	})
	return out
}

// ───────────────────────── version / range / hostOS ─────────────────────────
// Pure SemVer/calendar version + range + hostOS-constraint engine, lifted
// verbatim from the matrix's helpers. Kept package-private (the public surface
// is the Admissible* funcs + RangeContains); they have no I/O.

// version is a parsed dotted-numeric version with an optional pre-release tag.
// It models both SemVer (major.minor.patch[-pre]) and the platform's calendar
// release (year.month.patch) uniformly: both are compared component-by-component
// as integers, with a pre-release sorting below the release sharing its core.
type version struct {
	nums []int
	pre  string
	ok   bool
}

// parseVersion parses a dotted-numeric version, tolerating a leading "v" and an
// ignored "+build" suffix. It never errors: an unparsable input yields a version
// whose valid() is false, treated conservatively by the range engine.
func parseVersion(s string) version {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return version{}
	}
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	core := s
	pre := ""
	if i := strings.IndexByte(s, '-'); i >= 0 {
		core, pre = s[:i], s[i+1:]
	}
	parts := strings.Split(core, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return version{}
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return version{}
	}
	return version{nums: nums, pre: pre, ok: true}
}

// valid reports whether the version parsed into at least one numeric component.
func (v version) valid() bool { return v.ok }

// compare orders two versions, returning -1 if v < o, 0 if equal, +1 if v > o.
// Numeric components are compared left-to-right (missing trailing components
// count as 0); on an equal core a release outranks a pre-release, and two
// pre-releases of the same core are ordered lexically. An invalid version sorts
// below any valid one.
func (v version) compare(o version) int {
	switch {
	case !v.ok && !o.ok:
		return 0
	case !v.ok:
		return -1
	case !o.ok:
		return 1
	}
	n := len(v.nums)
	if len(o.nums) > n {
		n = len(o.nums)
	}
	for i := 0; i < n; i++ {
		a, b := 0, 0
		if i < len(v.nums) {
			a = v.nums[i]
		}
		if i < len(o.nums) {
			b = o.nums[i]
		}
		if a != b {
			if a < b {
				return -1
			}
			return 1
		}
	}
	switch {
	case v.pre == "" && o.pre != "":
		return 1
	case v.pre != "" && o.pre == "":
		return -1
	case v.pre == o.pre:
		return 0
	case v.pre < o.pre:
		return -1
	default:
		return 1
	}
}

// compareVersions orders two version strings by version.compare.
func compareVersions(a, b string) int { return parseVersion(a).compare(parseVersion(b)) }

// op is a comparison operator in a version-range comparator.
type op int

const (
	opGE op = iota // >=
	opGT           // >
	opLE           // <=
	opLT           // <
	opEQ           // = or bare version (exact)
)

// comparator is a single operator+version constraint (e.g. ">=2026.06.0").
type comparator struct {
	op op
	v  version
}

// satisfied reports whether candidate meets this comparator.
func (c comparator) satisfied(candidate version) bool {
	cmp := candidate.compare(c.v)
	switch c.op {
	case opGE:
		return cmp >= 0
	case opGT:
		return cmp > 0
	case opLE:
		return cmp <= 0
	case opLT:
		return cmp < 0
	default:
		return cmp == 0
	}
}

// versionRange is a conjunction of comparators (space-separated). An empty range
// imposes no constraint and is satisfied by every valid version. A range that
// fails to parse any comparator is unsatisfiable (admits nothing) so a malformed
// constraint never silently admits a candidate.
type versionRange struct {
	comps []comparator
	any   bool
	bad   bool
}

// parseRange parses a space-separated comparator range. Recognized prefixes:
// ">=", ">", "<=", "<", "=" and a bare version (exact match). A leading "v" on
// any version token is tolerated. An empty string yields the unconstrained
// range. A token that cannot be parsed marks the whole range unsatisfiable.
func parseRange(s string) versionRange {
	s = strings.TrimSpace(s)
	if s == "" {
		return versionRange{any: true}
	}
	tokens := strings.Fields(s)
	comps := make([]comparator, 0, len(tokens))
	for _, tok := range tokens {
		c, ok := parseComparator(tok)
		if !ok {
			return versionRange{bad: true}
		}
		comps = append(comps, c)
	}
	if len(comps) == 0 {
		return versionRange{any: true}
	}
	return versionRange{comps: comps}
}

// parseComparator parses one operator+version token.
func parseComparator(tok string) (comparator, bool) {
	o := opEQ
	rest := tok
	switch {
	case strings.HasPrefix(tok, ">="):
		o, rest = opGE, tok[2:]
	case strings.HasPrefix(tok, "<="):
		o, rest = opLE, tok[2:]
	case strings.HasPrefix(tok, ">"):
		o, rest = opGT, tok[1:]
	case strings.HasPrefix(tok, "<"):
		o, rest = opLT, tok[1:]
	case strings.HasPrefix(tok, "="):
		o, rest = opEQ, tok[1:]
	}
	v := parseVersion(strings.TrimSpace(rest))
	if !v.valid() {
		return comparator{}, false
	}
	return comparator{op: o, v: v}, true
}

// isAny reports whether the range imposes no constraint.
func (r versionRange) isAny() bool { return r.any }

// unsatisfiable reports whether the range admits no version (a malformed
// constraint). Distinct from isAny: an unsatisfiable range rejects everything.
func (r versionRange) unsatisfiable() bool { return r.bad }

// contains reports whether candidate satisfies every comparator in the range.
func (r versionRange) contains(candidate version) bool {
	if r.any {
		return true
	}
	if r.bad || !candidate.valid() {
		return false
	}
	for _, c := range r.comps {
		if !c.satisfied(candidate) {
			return false
		}
	}
	return true
}

// RangeContains parses rng and reports whether the version string satisfies it.
// An empty range admits everything (no constraint); a malformed range admits
// nothing. It is the string convenience the resolver uses to test a single
// version against a SemVer range without constructing intermediate values.
func RangeContains(rng, version string) bool {
	return parseRange(rng).contains(parseVersion(version))
}

// hostOSConstraint is one parsed hostOS entry in the grammar
// "<os>[<op><version>]" — e.g. "windows>=10", "macos>=13", "linux". The bare-OS
// form matches the OS on any version.
type hostOSConstraint struct {
	os   string
	comp *comparator
}

// parseHostOSConstraint parses a single hostOS entry.
func parseHostOSConstraint(s string) hostOSConstraint {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return hostOSConstraint{}
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '>' || ch == '<' || ch == '=' {
			osTok := strings.TrimSpace(s[:i])
			if c, ok := parseComparator(s[i:]); ok {
				return hostOSConstraint{os: osTok, comp: &c}
			}
			return hostOSConstraint{os: osTok} // malformed comparator → OS-only match
		}
	}
	return hostOSConstraint{os: s}
}

// matches reports whether the host (its OS token + numeric version) satisfies
// this constraint. OS tokens match case-insensitively; a versioned constraint is
// evaluated against hostVersion (an empty/unparsable hostVersion fails a
// versioned constraint but passes a bare-OS one).
func (c hostOSConstraint) matches(hostOS, hostVersion string) bool {
	if c.os == "" || !strings.EqualFold(c.os, hostOS) {
		return false
	}
	if c.comp == nil {
		return true
	}
	hv := parseVersion(hostVersion)
	if !hv.valid() {
		return false
	}
	return c.comp.satisfied(hv)
}

// hostOSSatisfied reports whether a host (hostOS + hostVersion) satisfies a
// candidate's hostOS constraint list. An empty list means "any host OS". A
// non-empty list is satisfied when ANY entry matches.
func hostOSSatisfied(constraints []string, hostOS, hostVersion string) bool {
	if len(constraints) == 0 {
		return true
	}
	for _, raw := range constraints {
		if parseHostOSConstraint(raw).matches(hostOS, hostVersion) {
			return true
		}
	}
	return false
}

// rangeContains is the lower-case package-internal alias used by checkRequires.
func rangeContains(rng, ver string) bool { return RangeContains(rng, ver) }

// ───────────────────────────────── helpers ──────────────────────────────────

// contains reports whether set holds v.
func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// quote wraps s in double quotes for reason strings (matches the previous
// fmt %q rendering for the common ASCII case without importing fmt here).
func quote(s string) string { return "\"" + s + "\"" }

// joinSet renders a set as ["a" "b"] for reason strings.
func joinSet(set []string) string {
	parts := make([]string, len(set))
	for i, s := range set {
		parts[i] = quote(s)
	}
	return "[" + strings.Join(parts, " ") + "]"
}
