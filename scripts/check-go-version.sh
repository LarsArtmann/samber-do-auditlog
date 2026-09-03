#!/bin/sh
# check-go-version.sh — Go-version drift guard.
#
# The Aug-2026 outage class: go.mod bumped to a newer Go while ci.yml still
# pinned the old version (or vice versa) -> 33 days of red master. This script
# asserts all four pinned sources agree:
#
#   1. go.mod            `go` directive           (canonical source)
#   2. .github/workflows/ci.yml   every `go-version:` value
#   3. flake.nix         every `GOTOOLCHAIN = "goX.Y.Z"` value
#   4. .golangci.yml     `run.go` setting
#
# Exit 0 = all agree. Exit 1 = drift detected (message says exactly where).

set -u

ROOT="${CHECK_GO_VERSION_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"

fail=0

version_from_gomod() {
	sed -n 's/^go \([0-9][0-9.]*\)$/\1/p' "$ROOT/go.mod" | head -n 1
}

EXPECTED="$(version_from_gomod)"
if [ -z "$EXPECTED" ]; then
	echo "FAIL: could not read 'go <version>' directive from go.mod"
	exit 1
fi

check() {
	# check <description> <expected> <actual>
	if [ "$2" != "$3" ]; then
		echo "FAIL: $1"
		echo "      expected: $2"
		echo "      actual:   $3"
		fail=1
	fi
}

# 2. ci.yml — every go-version value must match.
CI_VERSIONS="$(sed -n 's/.*go-version: *"\([^"]*\)".*/\1/p' "$ROOT/.github/workflows/ci.yml" | sort -u)"
if [ -z "$CI_VERSIONS" ]; then
	echo "FAIL: no go-version found in .github/workflows/ci.yml"
	fail=1
fi
for v in $CI_VERSIONS; do
	check "ci.yml go-version ($v) vs go.mod ($EXPECTED)" "$EXPECTED" "$v"
done

# 3. flake.nix — every GOTOOLCHAIN assignment must be go<expected>.
FLAKE_VERSIONS="$(sed -n 's/^.*GOTOOLCHAIN *= *"go\([0-9][0-9.]*\)".*$/\1/p' "$ROOT/flake.nix" | sort -u)"
if [ -z "$FLAKE_VERSIONS" ]; then
	echo "FAIL: no GOTOOLCHAIN pin found in flake.nix"
	fail=1
fi
for v in $FLAKE_VERSIONS; do
	# The flake may pin a full toolchain version (go1.23.12) while go.mod
	# declares the language line (go 1.23) — accept a patch-level extension.
	case "$v" in
	"$EXPECTED"|"$EXPECTED".*)
		;;
	*)
		echo "FAIL: flake.nix GOTOOLCHAIN (go$v) vs go.mod ($EXPECTED)"
		echo "      expected: $EXPECTED or $EXPECTED.x"
		fail=1
		;;
	esac
done

# 4. .golangci.yml — run.go must match.
LINT_VERSION="$(sed -n 's/^  go: *\([0-9][0-9.]*\)$/\1/p' "$ROOT/.golangci.yml" | head -n 1)"
if [ -z "$LINT_VERSION" ]; then
	echo "FAIL: could not read run.go from .golangci.yml"
	fail=1
else
	check ".golangci.yml run.go vs go.mod ($EXPECTED)" "$EXPECTED" "$LINT_VERSION"
fi

if [ "$fail" -ne 0 ]; then
	echo ""
	echo "Go-version drift detected. Canonical source: go.mod (go $EXPECTED)."
	echo "Fix: update the listed files to go $EXPECTED (see AGENTS.md 'Go toolchain pin')."
	exit 1
fi

echo "go-version check OK: go.mod, ci.yml, flake.nix, .golangci.yml all at $EXPECTED"
exit 0
