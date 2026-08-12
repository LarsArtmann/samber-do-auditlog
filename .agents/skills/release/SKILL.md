---
name: release
description: Use whenever it's time for a new release of this project. Triggers on "new release", "cut a release", "ship a version", "is it time for a release", "tag a release", or when the user asks to release, publish, or version-bump the project. Also triggers when the user asks "what's changed since the last tag" or "should we release". Guides the full release process: version determination, verification, CHANGELOG, tagging, GitHub Release, and post-release housekeeping.
user-invocable: true
---

# Release Process for samber-do-auditlog

This is a single-module Go library (`github.com/larsartmann/samber-do-auditlog`).
Every release produces one annotated Git tag (`vX.Y.Z`) and one GitHub Release.

The project is in **BETA (0.x)** — breaking changes are allowed between minor
releases. Pre-1.0 SemVer means: breaking changes bump the **minor** version,
not the major.

## Quick Reference: Exact Commands

All commands assume the Nix devShell (`nix develop`) or `GOEXPERIMENT=jsonv2`
set in the environment. The devShell sets it automatically.

```bash
# Assessment
git tag --sort=-creatordate | head -5              # latest tags
git log $(git describe --tags --abbrev=0)..HEAD --oneline | wc -l  # unreleased commits
git log $(git describe --tags --abbrev=0)..HEAD --format='%s' | sed -E 's/\(.*//' | sort | uniq -c | sort -rn  # by type

# Verification (all must pass before tagging)
export GOEXPERIMENT=jsonv2
go build ./...
go vet ./...
go test -race ./...
golangci-lint run
golangci-lint config verify
sh scripts/coverage-gate.sh                        # ≥94% gate
go generate ./... && git diff --exit-code          # no drift
GOTOOLCHAIN=go1.26.5 go mod tidy && git diff --exit-code go.mod go.sum  # no drift

# Fuzz tests (15s per target)
for fuzz in FuzzPluginHTML FuzzMigrateReport FuzzDiagramSpecialChars FuzzFilterInputs FuzzReadEvents FuzzMultiWriter FuzzNDJSONStreamer FuzzClassifyAdversarialChains; do
  go test -run='^$' -fuzz="^${fuzz}$" -fuzztime=15s .
done

# Release tooling
goreleaser check                                   # validate config

# Tag and push
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin master vX.Y.Z

# GitHub Release
gh release create vX.Y.Z --title "..." --notes "..."
```

---

## Step-by-Step Release Procedure

Execute these phases in order. Each phase has a verification gate — do not
proceed if a gate fails.

### Phase 1: Assess Release Readiness

Determine whether a release is warranted and what version it should be.

1. **Count unreleased commits:**
   ```bash
   last_tag=$(git describe --tags --abbrev=0)
   git log ${last_tag}..HEAD --oneline
   git log ${last_tag}..HEAD --oneline | wc -l
   ```
   A release is typically warranted at 20+ commits or when significant
   features/breaking changes have accumulated.

2. **Categorize changes** to determine the version bump:
   ```bash
   git log ${last_tag}..HEAD --format='%s' | grep -iE '^[a-z]+\(' | sed -E 's/\(.*//' | sort | uniq -c | sort -rn
   ```

3. **Determine the version number** using 0.x SemVer rules:
   - **PATCH** (v0.8.1 → v0.8.2): only bug fixes, dependency bumps, toolchain fixes
   - **MINOR** (v0.8.1 → v0.9.0): new features, new API surfaces, OR any breaking changes
   - **Breaking changes in 0.x bump the MINOR**, not the MAJOR. v1.0.0 is reserved for the stability commitment.

4. **Check for breaking changes** — signature changes, removed/renamed exports,
   schema version bumps. These MUST be documented in the CHANGELOG under a
   `### Breaking` section. Breaking changes are expected in 0.x; they do not
   block release but must be communicated.

5. **Check working tree is clean** — no uncommitted changes before starting.

### Phase 2: Verify Code Quality

Every gate must pass. If any fails, fix it before proceeding — releasing
broken code erodes trust.

```bash
export GOEXPERIMENT=jsonv2
go build ./...          # compiles
go vet ./...            # no vet issues
golangci-lint run       # 0 issues (strict config, ~50 linters)
sh scripts/coverage-gate.sh   # ≥94% (excludes example/, cmd/, live/demo/, testhelpers/)
```

Then run fuzz tests — they catch edge cases unit tests miss:
```bash
for fuzz in FuzzPluginHTML FuzzMigrateReport FuzzDiagramSpecialChars FuzzFilterInputs FuzzReadEvents FuzzMultiWriter FuzzNDJSONStreamer FuzzClassifyAdversarialChains; do
  echo "=== ${fuzz} ==="
  go test -run='^$' -fuzz="^${fuzz}$" -fuzztime=15s .
done
```

Verify no generated-code drift (CI's `stale-generation` job checks this):
```bash
go generate ./...
git diff --exit-code   # should be no changes
```

Verify no go.mod/go.sum drift (CI's `mod-tidy` job checks this):
```bash
GOTOOLCHAIN=go1.26.5 go mod tidy
git diff --exit-code go.mod go.sum
```

### Phase 3: Update Documentation

Three documents need updating before every release:

#### 3a. CHANGELOG.md

Move the `[Unreleased]` section's entries to a new versioned section:

```markdown
## [Unreleased]

## [X.Y.Z] - YYYY-MM-DD

Release summary paragraph — 2-4 sentences covering the headline changes.

### Added
- **Feature name** (`file.go`): description.

### Breaking
- **What changed**: old → new. Migration path if applicable.

### Fixed
- **Issue**: root cause and fix.
```

Accuracy rules:
- Only document what actually shipped. If a feature was added AND removed
  within the same unreleased period, it must NOT appear — verify against the
  codebase, not just commit messages.
- Every breaking change must have a `### Breaking` entry explaining the old
  → new signature and the migration path.
- Use Keep a Changelog categories: Added / Breaking / Changed / Fixed / Removed.

#### 3b. STABILITY.md

Check whether new exported types, functions, methods, or Config fields were
added since the last release. If so, add them to the appropriate table:

- **Stable API** table: core surfaces consumers depend on (`New`, `Config`
  fields, `Plugin.*` methods, `Report.*` methods, exported sentinels).
- **Evolving API** table: newer surfaces whose shape may change between 0.x
  releases (diagram writers, filters, streaming types, table/tree exports).
- **`live/` sub-package** table: SSE server, Hub, dashboard Config.

Failing to classify new API surfaces leaves consumers without guidance on
what they can safely depend on.

#### 3c. AGENTS.md

If new source files were added, ensure they appear in the Architecture file
listing. If new concurrency primitives or behavioral patterns were introduced,
document them in the Concurrency Model or Gotchas sections.

### Phase 4: Bump Version References

Update `CLIVersion` in `cmd/auditlog/main.go`:
```go
var CLIVersion = "X.Y.Z"  // was the previous version
```

The goreleaser config injects the tag version at build time via ldflags
(`-X main.CLIVersion={{.Version}}`), but the source default should still be
current for `go run ./cmd/auditlog` and `go install` users.

### Phase 5: Commit and Push

Create two logical commits:
1. **Code fixes** (if any verification failures were fixed): `fix: ...`
2. **Release docs**: `docs(release): prepare vX.Y.Z release`

```bash
git add <files>
git commit -m "docs(release): prepare vX.Y.Z release"
git push origin master
```

If the pre-commit hook (BuildFlow) fails on missing devShell binaries (npm,
tsc, go-licenses, etc.), those are infrastructure gaps, not code issues —
bypass with `--no-verify` only if the Go-specific checks (golangci-lint,
govulncheck, go-generate, go vet) all passed within the BuildFlow output.

### Phase 6: Tag and Push

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

The tag annotation should be concise — a one-line summary of headline changes.

### Phase 7: Create GitHub Release

```bash
gh release create vX.Y.Z \
  --title "vX.Y.Z — headline features" \
  --notes "release notes from CHANGELOG.md"
```

The release notes should be extracted from the CHANGELOG `[X.Y.Z]` section,
lightly edited for the GitHub audience (less internal detail, more user
impact). Include a link to the full CHANGELOG.

If goreleaser CI is configured to run on tag push, it will build CLI binaries
(linux/darwin amd64/arm64) and attach them automatically. Otherwise, run it
manually:
```bash
GORELEASER_CURRENT_TAG=vX.Y.Z GOEXPERIMENT=jsonv2 goreleaser release --clean
```

### Phase 8: Post-Release Housekeeping

1. **Add a new `[Unreleased]` section** to CHANGELOG.md (empty, ready for
   next cycle's entries).

2. **Verify pkg.go.dev** updates (usually automatic within minutes):
   https://pkg.go.dev/github.com/larsartmann/samber-do-auditlog

3. **Commit** the new `[Unreleased]` section:
   ```bash
   git add CHANGELOG.md
   git commit -m "docs(changelog): add [Unreleased] section post-vX.Y.Z"
   git push origin master
   ```

---

## Common Pitfalls

These are real issues encountered during past releases. Learning from them
prevents repeat mistakes.

### Flaky timing tests

Tests that rely on wall-clock timing with tight intervals (e.g., 20ms) will
fail under CI load. Use generous intervals (250ms+) or, better, extract the
decision logic into a pure function that can be tested without time
dependence.

### Lint on master

If `golangci-lint run` fails on a clean checkout, the pre-commit hook was
either bypassed (`--no-verify`) or the hook has a coverage gap. Fix the lint
issues before releasing — do not ship a release that fails its own lint config.

### Doc comment displacement during refactoring

When extracting a function from another (e.g., to fix cyclop complexity), the
`edit` tool can leave the original function's doc comment orphaned above the
new function. Always verify doc comments match their function names after
structural edits.

### Stale CHANGELOG entries

If a feature was both added and removed within an unreleased period (e.g.,
the `health/` sub-package was created and then extracted to a separate
project before any release), it must not appear in the release CHANGELOG.
Verify CHANGELOG claims against the actual codebase.

### GOEXPERIMENT=jsonv2

The project requires `GOEXPERIMENT=jsonv2` to build (transitive deps use
`encoding/json/v2`). This is set automatically in the Nix devShell and CI.
If running outside the devShell, prefix commands with `export
GOEXPERIMENT=jsonv2` or they will fail with obscure import errors.

### GOTOOLCHAIN=go1.26.5

`go mod tidy` can silently rewrite `go.mod` to a different Go version if a
different `go` binary shadows the devShell's. Always prefix with
`GOTOOLCHAIN=go1.26.5` when running outside the devShell.

---

## Version History Reference

| Tag | Type | Headline |
|-----|------|----------|
| v0.9.0 | MINOR (breaking) | RunID, MultiWriter, StreamEvents, SSE replay, keyboard a11y |
| v0.8.1 | PATCH | Dependency updates |
| v0.8.0 | MINOR | ALPHA→BETA, exported sentinels, Go 1.26.5 |
| v0.7.1 | PATCH | Go version regression |
| v0.7.0 | MINOR (breaking) | Typed identifiers, ServiceInfo domain split |
| v0.6.0 | MINOR | Live dashboard, diagram exports, CLI |
| v0.5.0 | MINOR | Health checks, filtering, D2 export |
| v0.4.0 | MINOR | Report validation, schema, CLI |
| v0.3.0 | MINOR | Tree/table export, CLI format coverage |

The versioning pattern: minor releases carry features and breaking changes;
patch releases carry only fixes and dependency bumps. No v1.0.0 until the
API stabilizes and the stability guarantee locks in.
