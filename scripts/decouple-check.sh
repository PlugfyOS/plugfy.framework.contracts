#!/usr/bin/env bash
# L1 baseplate gate: plugfy-common must be stdlib-only and import no other unit.
set -euo pipefail
if grep -qE '^\s*require ' go.mod; then
  echo "FAIL: plugfy-common must be stdlib-only (found 'require' in go.mod)"; exit 1
fi
if go list -deps ./... 2>/dev/null | grep -E 'github\.com/PlugfyOS/' | grep -v 'plugfy-common'; then
  echo "FAIL: baseplate imports another unit"; exit 1
fi
echo "OK: stdlib-only baseplate, imports no unit."
