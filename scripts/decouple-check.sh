#!/usr/bin/env sh
# DC-1 — L1 dependency-direction check (current documented decision, CI-verified).
#
# The documented design decision today is that the L1 Framework contracts depend
# on NOTHING from L2 Foundation or L3 Platform: the dependency arrow points up TO
# L1, never down FROM it. This script machine-checks that decision over the whole
# module. It is a documented, adjustable decision — if the layering needs to
# change, revise the decision and this guard accordingly; it is not a permanent
# law, just the invariant we currently hold and verify in CI.
#
# It FAILS (non-zero exit) if any package in this module imports a path under
#   github.com/PlugfyOS/plugfy.foundation.   (L2 Foundation)
#   github.com/PlugfyOS/plugfy.platform.     (L3 Platform)
#
# Run locally with the workspace disabled so it resolves like a fresh clone:
#   GOWORK=off sh scripts/decouple-check.sh
set -eu

FORBIDDEN='github\.com/PlugfyOS/plugfy\.(foundation|platform)\.'

# Resolve the full transitive import set per package so a forbidden import is
# caught even when it is reached indirectly. `{{.ImportPath}}: {{.Deps}}` lets us
# report both the offending dependency AND the package that pulls it in.
deps="$(GOWORK=off go list -deps -f '{{.ImportPath}}: {{join .Deps " "}}' ./... 2>/dev/null \
  || go list -deps -f '{{.ImportPath}}: {{join .Deps " "}}' ./...)"

# An importer line is offending if any of its deps matches a forbidden prefix.
offending="$(printf '%s\n' "$deps" | grep -E ": .*${FORBIDDEN}" || true)"

if [ -n "$offending" ]; then
  echo "FAIL: L1 contracts must not import L2 Foundation or L3 Platform."
  echo "The current documented dependency-direction decision is violated by:"
  printf '%s\n' "$offending" | while IFS= read -r line; do
    importer="${line%%:*}"
    bad="$(printf '%s\n' "$line" | tr ' ' '\n' | grep -E "$FORBIDDEN" | sort -u | tr '\n' ' ')"
    echo "  - $importer  imports  $bad"
  done
  exit 1
fi

echo "OK: no L2 Foundation / L3 Platform imports — L1 dependency-direction decision holds."
