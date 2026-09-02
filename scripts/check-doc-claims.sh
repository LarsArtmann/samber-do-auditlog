#!/bin/sh
# check-doc-claims.sh — claims linter for README/docs.
#
# Catches stale hard-coded numbers in documentation by comparing doc claims
# against machine-verifiable sources (go.mod, ci.yml, .golangci.yml, types.go,
# the fuzz-target grep). Born from the 2026-09-01 launch audit, which found
# 5 stale claims by hand (linter counts, fuzz counts, coverage numbers,
# schema versions, Go versions).
#
# Each check: extract the claim from the doc, extract the truth from source,
# fail loudly on mismatch. Add new checks as new hard-coded numbers appear.

set -eu

ROOT="${DOC_CLAIMS_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
cd "$ROOT"

fail=0

claim_fail() {
	echo "FAIL: $1"
	echo "      doc says:     $2"
	echo "      source truth: $3"
	fail=1
}

# --- Ground truth ---------------------------------------------------------

GO_MOD_VERSION="$(sed -n 's/^go \([0-9][0-9.]*\)$/\1/p' go.mod | head -n 1)"
SCHEMA_VERSION="$(grep -oE 'SchemaVersion +ProviderVersion = "[0-9.]+"' types.go | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n 1)"
if [ -z "$SCHEMA_VERSION" ]; then
	SCHEMA_VERSION="$(grep -oE 'SchemaVersion *= *"[0-9.]+"' types.go | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n 1)"
fi
COVERAGE_GATE="$(grep -oE 'below [0-9]+(\.[0-9]+)?%' .github/workflows/ci.yml | grep -oE '[0-9]+(\.[0-9]+)?' | head -n 1)"
LINTER_COUNT="$(sed -n "$(grep -n '^  enable:' .golangci.yml | head -1 | cut -d: -f1),$(grep -n '^  settings:' .golangci.yml | head -1 | cut -d: -f1)p" .golangci.yml | grep -c '^    - ' || true)"
FUZZ_COUNT="$(grep -rh -o -E '^func (Fuzz[A-Za-z]+)' --include='*_test.go' . | wc -l | tr -d ' ')"

# --- Checks ---------------------------------------------------------------

# 1. CONTRIBUTING.md must state the exact Go version.
if ! grep -q "Go $GO_MOD_VERSION" CONTRIBUTING.md; then
	claim_fail "CONTRIBUTING.md Go version claim" "(missing Go $GO_MOD_VERSION)" "go.mod: go $GO_MOD_VERSION"
fi

# 2. STABILITY.md must state the current schema version.
if ! grep -q "$SCHEMA_VERSION" STABILITY.md; then
	claim_fail "STABILITY.md schema version claim" "(missing $SCHEMA_VERSION)" "types.go: $SCHEMA_VERSION"
fi

# 3. README fuzz-target count claim (README.md line: "N fuzz targets").
DOC_FUZZ="$(grep -oE '[0-9]+ fuzz targets?' README.md | grep -oE '[0-9]+' | head -n 1 || true)"
if [ -n "$DOC_FUZZ" ] && [ "$DOC_FUZZ" != "$FUZZ_COUNT" ]; then
	claim_fail "README fuzz target count" "$DOC_FUZZ fuzz targets" "$FUZZ_COUNT fuzz targets"
fi

# 4. README linter count claim ("N linters").
DOC_LINT="$(grep -oE '[0-9]+ linters' README.md | grep -oE '[0-9]+' | head -n 1 || true)"
if [ -n "$DOC_LINT" ] && [ "$DOC_LINT" != "$LINTER_COUNT" ]; then
	claim_fail "README linter count" "$DOC_LINT linters" "$LINTER_COUNT linters"
fi

# 5. README coverage-gate claim ("94% coverage gate" style) must match ci.yml.
DOC_COV="$(grep -oE '[0-9]+(\.[0-9]+)?% coverage gate' README.md | grep -oE '[0-9]+(\.[0-9]+)?' | head -n 1 || true)"
if [ -n "$DOC_COV" ] && [ "$DOC_COV" != "$COVERAGE_GATE" ]; then
	claim_fail "README coverage gate" "$DOC_COV% coverage gate" "$COVERAGE_GATE% (ci.yml)"
fi

# --- Result ---------------------------------------------------------------

if [ "$fail" -ne 0 ]; then
	echo ""
	echo "Doc claims are stale. Update the docs (or the check's source extraction)."
	exit 1
fi

echo "doc claims OK: go=$GO_MOD_VERSION schema=$SCHEMA_VERSION coverage-gate=$COVERAGE_GATE% linters=$LINTER_COUNT fuzz=$FUZZ_COUNT"
