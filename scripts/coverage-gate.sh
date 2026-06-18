#!/usr/bin/env sh
# Coverage gate for samber-do-auditlog — mirrors the CI test job.
#
# Runs the race-enabled test suite with a coverage profile, excludes the
# example/ demo package, and fails if non-example coverage drops below 95%.
#
# Usage (from the repo root, in the devShell):
#   scripts/coverage-gate.sh

set -e

go test -race -count=1 -coverprofile=cover.out -covermode=atomic ./...

# Exclude the example/ (demo) and cmd/ (tooling) packages from the gate.
# Their logic is exercised by integration/golden tests that run a built binary.
grep -v -e '/example/' -e '/cmd/' cover.out > cover-filtered.out

coverage=$(go tool cover -func=cover-filtered.out | grep '^total:' | awk '{print $3}' | tr -d '%')
echo "Total coverage (non-example): ${coverage}%"

threshold=95
if awk "BEGIN {exit !($coverage < $threshold)}"; then
  echo "❌ Coverage ${coverage}% is below ${threshold}%" >&2
  exit 1
fi

echo "✓ Coverage ${coverage}% meets the ${threshold}% gate"
