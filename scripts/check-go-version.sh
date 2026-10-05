#!/bin/sh
# check-go-version.sh — Go-version drift guard.
#
# The Aug-2026 outage class: go.mod bumped to a newer Go while ci.yml still
# pinned the old version (or vice versa) -> 33 days of red master. This script
# asserts all four pinned sources agree with go.mod:
#
#   1. go.mod                          `go` directive (canonical source,
#                                      minor-only per fleet policy, e.g. `go 1.27`)
#   2. .github/workflows/ci.yml        every `go-version:` value
#   3. flake.nix                       GOTOOLCHAIN pins and Go package refs
#   4. .golangci.yml                   `run.go` setting
#
# go.mod carries the minor-only language line while the toolchain pins may
# carry a patch level (1.27 vs 1.27.1): patch extensions of the expected
# version are accepted everywhere. flake.nix may either pin
# `GOTOOLCHAIN = "goX.Y.Z"` explicitly or use `GOTOOLCHAIN = "local"` — in
# that case the nixpkgs Go package (go_1_MM / buildGo<MM>Module) supplies the
# exact toolchain and its minor version is checked against go.mod instead.
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

# matches_policy <version>: OK when identical to go.mod's directive or a
# patch-level extension of it (1.27 matches `go 1.27`; so does 1.27.1).
matches_policy() {
	case "$1" in
	"$EXPECTED" | "$EXPECTED".*)
		return 0
		;;
	*)
		return 1
		;;
	esac
}

check() {
	# check <description> <version>
	if ! matches_policy "$2"; then
		echo "FAIL: $1"
		echo "      expected: $EXPECTED (or $EXPECTED.x)"
		echo "      actual:   $2"
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
	check "ci.yml go-version ($v) vs go.mod ($EXPECTED)" "$v"
done

# 3. flake.nix — explicit GOTOOLCHAIN pins must match; a `GOTOOLCHAIN =
# "local"` pin is allowed only when the nixpkgs Go packages/builders it
# relies on match go.mod's minor version.
FLAKE_VERSIONS="$(sed -n 's/^.*GOTOOLCHAIN *= *"go\([0-9][0-9.]*\)".*$/\1/p' "$ROOT/flake.nix" | sort -u)"
FLAKE_LOCAL="$(sed -n 's/^.*GOTOOLCHAIN *= *"local".*$/local/p' "$ROOT/flake.nix" | sort -u)"
if [ -z "$FLAKE_VERSIONS" ] && [ -z "$FLAKE_LOCAL" ]; then
	echo "FAIL: no GOTOOLCHAIN pin found in flake.nix"
	fail=1
fi
for v in $FLAKE_VERSIONS; do
	check "flake.nix GOTOOLCHAIN (go$v) vs go.mod ($EXPECTED)" "$v"
done

# go_1_MM package refs: minor version must equal go.mod's minor.
FLAKE_GO_PKGS="$(sed -n 's/^.*\(go_[0-9]*_[0-9]*\).*/\1/p' "$ROOT/flake.nix" | sort -u)"
# buildGo<MM>Module builder refs: digits are major+minor without separators.
FLAKE_GO_BUILDERS="$(sed -n 's/^.*\(buildGo[0-9]*Module\).*/\1/p' "$ROOT/flake.nix" | sort -u)"
if [ -n "$FLAKE_LOCAL" ] && [ -z "$FLAKE_GO_PKGS" ] && [ -z "$FLAKE_GO_BUILDERS" ]; then
	echo "FAIL: flake.nix sets GOTOOLCHAIN=local but pins no nixpkgs Go package/builder (go_1_MM / buildGo<MM>Module)"
	fail=1
fi
for pkg in $FLAKE_GO_PKGS; do
	minor="$(printf '%s' "$pkg" | sed 's/^go_\([0-9]*\)_\([0-9]*\)$/\1.\2/')"
	check "flake.nix Go package ($pkg = go $minor) vs go.mod ($EXPECTED)" "$minor"
done
EXPECTED_NODOTS="$(printf '%s' "$EXPECTED" | tr -d '.')"
for builder in $FLAKE_GO_BUILDERS; do
	digits="$(printf '%s' "$builder" | sed 's/^buildGo\([0-9]*\)Module$/\1/')"
	if [ "$digits" != "$EXPECTED_NODOTS" ]; then
		echo "FAIL: flake.nix Go builder ($builder, digits $digits) vs go.mod ($EXPECTED, digits $EXPECTED_NODOTS)"
		fail=1
	fi
done

# 4. .golangci.yml — run.go must match (quoted or bare).
LINT_VERSION="$(sed -n 's/^ *go: *["]*\([0-9][0-9.]*\)["]*$/\1/p' "$ROOT/.golangci.yml" | head -n 1)"
if [ -z "$LINT_VERSION" ]; then
	echo "FAIL: could not read run.go from .golangci.yml"
	fail=1
else
	check ".golangci.yml run.go ($LINT_VERSION) vs go.mod ($EXPECTED)" "$LINT_VERSION"
fi

if [ "$fail" -ne 0 ]; then
	echo ""
	echo "Go-version drift detected. Canonical source: go.mod (go $EXPECTED)."
	echo "Fix: update the listed files to go $EXPECTED (see AGENTS.md 'Go toolchain pin')."
	exit 1
fi

echo "go-version check OK: go.mod, ci.yml, flake.nix, .golangci.yml all at $EXPECTED"
exit 0
