#!/usr/bin/env sh
# Coverage gate for samber-do-auditlog — mirrors the CI test job.
#
# Runs the race-enabled test suite with a coverage profile, excludes the
# example/ demo package, and fails if non-example coverage drops below 94%.
#
# Usage (from the repo root, in the devShell):
#   scripts/coverage-gate.sh

set -e

export GOEXPERIMENT=jsonv2

go test -race -count=1 -coverprofile=cover.out -covermode=atomic ./...

# Exclude the packages listed in scripts/coverage-exclusions.txt (single
# source shared with the CI test job): example/ (demo), cmd/ (tooling),
# live/demo/, internal/testhelpers/ (test infrastructure), and generated templ
# code (*_templ.go) from the gate.
while IFS= read -r line; do
	[ -n "$line" ] || continue
	set -- "$@" -e "$line"
done <"$(dirname "$0")/coverage-exclusions.txt"
grep -v "$@" cover.out >cover-filtered.out

coverage=$(go tool cover -func=cover-filtered.out | grep '^total:' | awk '{print $3}' | tr -d '%')
echo "Total coverage (non-example): ${coverage}%"

threshold=94
if awk "BEGIN {exit !($coverage < $threshold)}"; then
	echo "❌ Coverage ${coverage}% is below ${threshold}%" >&2
	exit 1
fi

echo "✓ Coverage ${coverage}% meets the ${threshold}% gate"
