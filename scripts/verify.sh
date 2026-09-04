#!/usr/bin/env sh
# verify.sh — one-command local parity with the pre-commit hook AND the CI
# gates, so "works on my machine" means the same thing locally as in CI.
#
# Runs, in order (stop on first failure):
#   1. go-version drift guard          (CI: test job step + pre-commit)
#   2. doc claims linter               (pre-commit)
#   3. go build ./...                  (CI: test job)
#   4. go generate drift check         (CI: stale-generation job + pre-commit)
#   5. go mod tidy drift check         (CI: mod-tidy job; needs Go >= 1.23)
#   6. go vet ./...                    (CI + pre-commit)
#   7. golangci-lint config verify     (CI: lint job)
#      golangci-lint run --timeout=10m
#   8. go test -race -count=1 ./...    (CI + pre-commit)
#   9. coverage gate >= 94%            (CI: test job)
#  10. fuzz seed corpus                (go test runs Fuzz targets' seeds)
#  11. short live fuzzing              (VERIFY_FUZZ_SECS per target, 0 = skip)
#
# Usage (from the repo root, in the devShell):
#   scripts/verify.sh
#   VERIFY_FUZZ_SECS=0 scripts/verify.sh      # skip live fuzzing
#   VERIFY_FUZZ_SECS=30 scripts/verify.sh     # longer fuzz budget per target
#
# On the go1.23-compat branch the ambient environment must NOT carry
# GOEXPERIMENT=jsonv2 (master-only); the devShell clears it.

set -eu
cd "$(dirname "$0")/.."

: "${VERIFY_FUZZ_SECS:=10}"

echo "=== 1/11 go-version drift guard"
./scripts/check-go-version.sh

echo "=== 2/11 doc claims linter"
./scripts/check-doc-claims.sh

echo "=== 3/11 go build"
go build ./...

echo "=== 4/11 go generate drift check"
generated_files="schema/report.schema.json"
snapshot() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum $generated_files 2>/dev/null || true
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 $generated_files 2>/dev/null || true
	else
		cksum $generated_files 2>/dev/null || true
	fi
}
before=$(snapshot)
go generate ./...
after=$(snapshot)
if [ "$before" != "$after" ]; then
	echo "❌ Generated code is stale or changed by 'go generate'." >&2
	echo "   Run 'go generate ./...' and commit: $generated_files" >&2
	exit 1
fi

echo "=== 5/11 go mod tidy drift check"
tidy_diff="$(go mod tidy -diff || true)"
if [ -n "$tidy_diff" ]; then
	echo "❌ go.mod/go.sum are stale. Run 'go mod tidy' and commit." >&2
	printf '%s\n' "$tidy_diff" >&2
	exit 1
fi

echo "=== 6/11 go vet"
go vet ./...

echo "=== 7/11 golangci-lint (config verify + run)"
# config verify downloads the remote JSON schema; if the network is
# unavailable, warn instead of failing — CI enforces this check, and
# `golangci-lint run` below compiles the config independently.
if ! golangci-lint config verify; then
	echo "WARN: golangci-lint config verify failed (offline?); CI enforces it." >&2
fi
golangci-lint run --timeout=10m ./...

echo "=== 8/11 go test -race"
go test -race -count=1 ./...

echo "=== 9/11 coverage gate"
./scripts/coverage-gate.sh

echo "=== 10/11 fuzz seed corpus"
go test -run 'Fuzz' -count=1 ./...

if [ "$VERIFY_FUZZ_SECS" -gt 0 ]; then
	echo "=== 11/11 live fuzzing (${VERIFY_FUZZ_SECS}s per target)"
	for target in $(
		grep -rh -o -E '^func (Fuzz[A-Za-z]+)' --include='*_test.go' . |
			sed 's/^func //' | sort -u
	); do
		pkg_dir="$(grep -rl --include='*_test.go' "func $target(" . | head -n 1 | xargs dirname)"
		echo "--- fuzz $target (in $pkg_dir)"
		go test -fuzz "^$target\$" -fuzztime "${VERIFY_FUZZ_SECS}s" -run '^$' "./$pkg_dir"
	done
else
	echo "=== 11/11 live fuzzing skipped (VERIFY_FUZZ_SECS=0)"
fi

echo "✓ verify: all checks passed"
