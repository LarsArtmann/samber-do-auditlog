# Release Process

Step-by-step guide for cutting a release of samber-do-auditlog.
This is a **single-module repo**: every release produces one tag.

---

## Tag Convention

| Module | Import path                                        | Tag format | Example  |
| ------ | -------------------------------------------------- | ---------- | -------- |
| Core   | `github.com/larsartmann/samber-do-auditlog`        | `vX.Y.Z`   | `v0.8.1` |

### SemVer guidance

- **PATCH** (`v0.8.1`): bug fixes, dependency bumps, additive changes on
  Evolving API surfaces, toolchain/security fixes.
- **MINOR** (`v0.9.0`): new features, new Stable/Evolving API additions.
- **MAJOR** (`v1.0.0`): breaking changes (post-1.0 stability guarantee).

Consult [`STABILITY.md`](STABILITY.md) for which API surfaces permit
breaking changes in 0.x minor releases.

---

## Pre-release Checklist

1. **Run the canonical check suite:**

   ```bash
   GOEXPERIMENT=jsonv2 go vet ./...
   GOEXPERIMENT=jsonv2 go test -race ./...
   golangci-lint run --timeout=10m
   govulncheck ./...
   ```

2. **Verify coverage** is at or above the 94% gate:

   ```bash
   GOEXPERIMENT=jsonv2 go test -race -count=1 -coverprofile=cover.out -covermode=atomic ./...
   grep -v -e '/example/' -e '/cmd/' -e '/live/demo/' -e '/testhelpers/' cover.out > cover-filtered.out
   go tool cover -func=cover-filtered.out | tail -1
   ```

3. **Update [`CHANGELOG.md`](CHANGELOG.md):**
   - Move `[Unreleased]` entries to a new `[X.Y.Z] - YYYY-MM-DD` section.
   - Use Keep a Changelog categories: Added / Changed / Fixed / Removed.

4. **Regenerate schema** if types changed:

   ```bash
   GOEXPERIMENT=jsonv2 go generate ./...
   ```

5. **Verify goreleaser config:**

   ```bash
   goreleaser check
   ```

---

## Cutting the Release

1. **Commit and push all changes.**

2. **Tag the release:**

   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```

3. **Goreleaser runs automatically via CI** (or manually):

   ```bash
   GORELEASER_CURRENT_TAG=vX.Y.Z GOEXPERIMENT=jsonv2 goreleaser release --clean
   ```

4. **Verify the GitHub Release:**
   - Changelog is auto-generated and grouped.
   - Binary archives (linux/darwin amd64/arm64) are attached.
   - Checksums file is present.

5. **Update pkg.go.dev** (usually automatic within a few minutes):
   - Visit https://pkg.go.dev/github.com/larsartmann/samber-do-auditlog
   - If not updated, request via the "Request" button.

---

## Post-release

1. **Add a new `[Unreleased]` section** to `CHANGELOG.md`.
2. **Bump version** in `cmd/auditlog/main.go` (`CLIVersion`).
3. **Announce** in relevant channels if notable.
