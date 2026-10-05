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
LINTER_COUNT="$(sed -n "$(grep -n '^ *enable:' .golangci.yml | head -1 | cut -d: -f1),$(grep -n '^ *settings:' .golangci.yml | head -1 | cut -d: -f1)p" .golangci.yml | grep -c '^ *- ' || true)"
FUZZ_COUNT="$(grep -rh -o -E '^func (Fuzz[A-Za-z]+)' --include='*_test.go' . | wc -l | tr -d ' ')"
BENCH_COUNT="$(grep -rh -c -E '^func Benchmark[A-Za-z]+' --include='*_test.go' . | awk -F: '{s+=$1} END {print s+0}')"
CI_JOB_COUNT="$(sed -n '/^jobs:/,$p' .github/workflows/ci.yml | grep -cE '^  [a-z][a-z-]*:$')"
ENV_KEY="$(grep -oE 'EnvKeyEnabled = "[A-Z_]+"' plugin.go | grep -oE '"[A-Z_]+"' | tr -d '"' | head -n 1)"
DIAGRAM_COUNT="$(ls mermaid.go plantuml.go dot.go d2.go 2>/dev/null | wc -l | tr -d ' ')"

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

# 6. Env-var toggle claims must quote the exact EnvKeyEnabled constant.
if [ -n "$ENV_KEY" ]; then
	for doc in README.md AGENTS.md; do
		if grep -qE 'DO_AUDITLOG_[A-Z_]+' "$doc" && ! grep -q "$ENV_KEY" "$doc"; then
			claim_fail "$doc env var name" "(no exact $ENV_KEY mention)" "plugin.go: $ENV_KEY"
		fi
	done
	# Any env-var spelling that differs from the constant is a stale claim.
	for doc in README.md AGENTS.md; do
		bad_env="$(grep -oE 'DO_AUDITLOG_[A-Z_]+' "$doc" | sort -u | grep -v "^$ENV_KEY$" || true)"
		if [ -n "$bad_env" ]; then
			claim_fail "$doc env var spelling" "$bad_env" "plugin.go: $ENV_KEY"
		fi
	done
fi

# 7. Benchmark count claims ("Benchmarks (N)" / "N benchmarks") must match the test grep.
for doc in AGENTS.md BENCHMARKS.md; do
	DOC_BENCH="$(grep -oE '[0-9]+ benchmarks?|Benchmarks \([0-9]+\)' "$doc" | grep -oE '[0-9]+' | head -n 1 || true)"
	if [ -n "$DOC_BENCH" ] && [ "$DOC_BENCH" != "$BENCH_COUNT" ]; then
		claim_fail "$doc benchmark count" "$DOC_BENCH" "$BENCH_COUNT (grep '^func Benchmark')"
	fi
done

# 8. CI job count claims ("N parallel jobs") must match ci.yml's jobs block.
for doc in AGENTS.md FEATURES.md; do
	DOC_JOBS="$(grep -oE '[0-9]+ parallel jobs' "$doc" | grep -oE '[0-9]+' | head -n 1 || true)"
	if [ -n "$DOC_JOBS" ] && [ "$DOC_JOBS" != "$CI_JOB_COUNT" ]; then
		claim_fail "$doc CI job count" "$DOC_JOBS parallel jobs" "$CI_JOB_COUNT jobs (ci.yml)"
	fi
done

# 9. Fuzz-target count claims ("N targets") in AGENTS.md and FEATURES.md.
for doc in AGENTS.md FEATURES.md; do
	DOC_FUZZ2="$(grep -oE '\([0-9]+ targets?\)|[0-9]+ fuzz targets?' "$doc" | grep -oE '[0-9]+' | head -n 1 || true)"
	if [ -n "$DOC_FUZZ2" ] && [ "$DOC_FUZZ2" != "$FUZZ_COUNT" ]; then
		claim_fail "$doc fuzz target count" "$DOC_FUZZ2 targets" "$FUZZ_COUNT targets"
	fi
done

# 10. Diagram-format count claims ("N diagram exports/formats") must match the renderer files.
for doc in AGENTS.md FEATURES.md README.md; do
	DOC_DIAG="$(grep -oE '[0-9]+ diagram (exports|formats)' "$doc" | grep -oE '[0-9]+' | head -n 1 || true)"
	if [ -n "$DOC_DIAG" ] && [ "$DOC_DIAG" != "$DIAGRAM_COUNT" ]; then
		claim_fail "$doc diagram format count" "$DOC_DIAG diagram formats" "$DIAGRAM_COUNT renderer files (mermaid/plantuml/dot/d2.go)"
	fi
done

# 11. README Go-version badge must match go.mod's major.minor.
GO_MOD_MINOR="$(printf '%s' "$GO_MOD_VERSION" | cut -d. -f1,2)"
DOC_BADGE_GO="$(grep -oE 'Go-[0-9]+\.[0-9]+\+' README.md | grep -oE '[0-9]+\.[0-9]+' | head -n 1 || true)"
if [ -n "$DOC_BADGE_GO" ] && [ "$DOC_BADGE_GO" != "$GO_MOD_MINOR" ]; then
	claim_fail "README Go version badge" "Go-$DOC_BADGE_GO+" "go.mod: $GO_MOD_VERSION"
fi

# --- Result ---------------------------------------------------------------

if [ "$fail" -ne 0 ]; then
	echo ""
	echo "Doc claims are stale. Update the docs (or the check's source extraction)."
	exit 1
fi

echo "doc claims OK: go=$GO_MOD_VERSION schema=$SCHEMA_VERSION coverage-gate=$COVERAGE_GATE% linters=$LINTER_COUNT fuzz=$FUZZ_COUNT benchmarks=$BENCH_COUNT ci-jobs=$CI_JOB_COUNT env-key=$ENV_KEY diagram-formats=$DIAGRAM_COUNT"
