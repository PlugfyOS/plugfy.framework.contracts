package ids

import (
	"regexp"
	"testing"
)

var ulidRe = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

func TestNewFormat(t *testing.T) {
	id := New()
	if len(id) != 26 {
		t.Fatalf("expected 26 chars, got %d (%q)", len(id), id)
	}
	if !ulidRe.MatchString(id) {
		t.Fatalf("ulid %q does not match Crockford base32 pattern", id)
	}
}

func TestUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		id := New()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ulid generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestPrefixed(t *testing.T) {
	id := Prefixed("proj")
	if id[:5] != "proj_" {
		t.Fatalf("expected proj_ prefix, got %q", id)
	}
	if !ulidRe.MatchString(id[5:]) {
		t.Fatalf("suffix not a valid ulid: %q", id)
	}
}
