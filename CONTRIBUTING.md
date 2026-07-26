# Contributing

Thanks for your interest in making do-auditlog better.

This project uses the standard Go toolchain. A `flake.nix` devShell is available for contributors using Nix; otherwise you just need Go and `golangci-lint`. The templ CLI is managed automatically via Go's `tool` directive — no manual install needed.

---

## Prerequisites

- [Go 1.26.5](https://go.dev/dl/) (exact version — see `go.mod`)
- [golangci-lint](https://golangci-lint.run/usage/install/) (latest v2.x)
- [templ](https://templ.guide/) (only if you edit `html.templ`)

**Nix users:** Run `nix develop` to get Go 1.26.5, golangci-lint, govulncheck, and golines pinned in `flake.nix`.

### GOEXPERIMENT=jsonv2 (required)

This project must be built with `GOEXPERIMENT=jsonv2`. A transitive dependency (`larsartmann/go-output`) imports `encoding/json/v2`, which requires the build experiment flag in Go 1.26. Without it, compilation fails.

This is **temporary** — `encoding/json/v2` is expected to stabilize in Go 1.27, at which point this requirement will be removed.

- **Nix devShell**: sets `GOEXPERIMENT=jsonv2` automatically
- **Manual**: `export GOEXPERIMENT=jsonv2` in every terminal session

Verify your setup:

```bash
go version
golangci-lint version
```

## Development Workflow

1. **Fork** the repository
2. **Create a branch** from `master`
3. **Make your changes**
4. **Run checks** (see below)
5. **Submit a pull request**

## Running Checks

Run these before every commit. They must all pass.

```bash
# 1. Regenerate generated code (only if you changed html.templ)
go generate ./...

# 2. Run all tests, including race detection
go test ./... -race

# 3. Static analysis
go vet ./...

# 4. Verify lint config schema (catches silent config issues)
golangci-lint config verify

# 5. Full lint (strict config — this is the gatekeeper)
golangci-lint run
```

If `golangci-lint` fails, fix the issues. Do not bypass linters.

## Code Style

Follow the existing code. The project enforces style through `.golangci.yml`, but here are the principles behind it:

- **Early returns** over nested conditionals
- **Explicit over implicit** — no magic, clear signatures
- **Small, focused functions** — single responsibility
- **Composition over inheritance** — behavior injection, not deep hierarchies
- **Strong types** — make impossible states unrepresentable
- **Descriptive names** — if you need a comment to explain what a function does, the name is wrong

### Lint Highlights

- `exhaustruct` — every struct field must be explicitly initialized (tests are exempt)
- `depguard` — only stdlib, `samber/do`, `a-h/templ`, `larsartmann/go-output`, and this module are allowed
- `noinlineerr` — declare `err` on its own line, then check it
- `forbidigo` — no `fmt.Print*` in production code
- `tagliatelle` — JSON tags use `snake_case`
- Maximum line length: 120 characters (`golines`)

## Testing

- Use the **external test package**: `package auditlog_test` (imports `auditlog` explicitly)
- **Table-driven tests** preferred
- No external assertion libraries — standard `testing.T` only
- Every test creates its own `Plugin` + `do.Injector` — no shared state
- Use `t.Setenv()` for env var tests, `t.TempDir()` for file tests

Example pattern:

```go
func TestPlugin_SomeFeature(t *testing.T) {
    plugin, err := auditlog.New(auditlog.Config{Enabled: true})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    injector := do.NewWithOpts(plugin.Opts())

    // ... exercise the feature ...

    report := plugin.Report()
    if report.ServiceCount != 1 {
        t.Fatalf("expected 1 service, got %d", report.ServiceCount)
    }
}
```

## Generated Code

`html_templ.go` is generated from `html.templ` by `go tool templ generate` (pinned via Go's `tool` directive in go.mod).

- **Never edit `html_templ.go` by hand.**
- If you modify `html.templ`, run `go generate ./...` and include the regenerated file in your commit.

## Documentation

If you add or change user-facing behavior, update:

- `README.md` — for users
- `FEATURES.md` — add the feature to the inventory
- `CHANGELOG.md` — under `[Unreleased]`
- `docs/DOMAIN_LANGUAGE.md` — if you introduce a new domain concept

## Commit Messages

Write clear, imperative commit messages that explain _why_, not just _what_:

```
Add health check event support for samber/do v2

samber/do v2 does not expose HookBeforeHealthCheck, so we wrap
injector.HealthCheckWithContext() instead. This records EventTypeHealthCheck
events and updates ServiceInfo health fields without modifying the core hook flow.
```

## Releasing

Release tags and the report schema version are **independent**:

- **Release tags** follow `v0.x.y` (e.g. `v0.5.0`). These mark GitHub releases.
- **Schema version** (currently `0.2.0`, in `types.go`) versions the JSON report format. It is upgraded via `MigrateReport` and has no relation to release tags.

### Release Procedure

1. **Update `CHANGELOG.md`** — move `[Unreleased]` items under a new `[0.x.y]` heading with today's date.
2. **Verify changelog claims against reality.** Open every file the changelog cites (README, STABILITY, FEATURES, etc.) and confirm the claims are true. Inherited `[Unreleased]` text is untrusted until verified.
3. **Commit** the changelog update yourself with a descriptive `release: v0.x.y — …` message (do not let the auto-git daemon commit release artifacts with a generic message).
4. **Re-run every gate from scratch.** Do not trust a prior session's "green" — re-run `go generate ./...`, `go vet ./...`, `go test -race ./...`, `golangci-lint config verify`, and `golangci-lint run`. **Capture each exit code directly from the command, never from a downstream pipe element** (`cmd | tail; echo $?` reports `tail`'s exit, not `cmd`'s). Use `cmd >/tmp/out 2>&1; ec=$?; cat /tmp/out; echo "exit $ec"` or `set -o pipefail`.
5. **Tag** the release (signed):
   ```bash
   git tag -s v0.x.y -m "v0.x.y — short description"
   ```
6. **Verify the tag signature:**
   ```bash
   git tag -v v0.x.y
   ```
7. **Push** the tag and master:
   ```bash
   git push origin master --tags
   ```
8. **Create a GitHub Release** using `gh release create` with the changelog body as notes. Attach the example HTML artifact:
   ```bash
   DO_AUDITLOG_ENABLED=true go run ./example
   gh release create v0.x.y --notes-file <notes> /tmp/.../audit-report.html
   ```
9. **Verify** the CI badge is green, the release appears on the releases page, and (if you sign with SSH) the tag shows as "Verified" on GitHub. GitHub-side verification requires the signing key to be registered as a **Signing Key** (not an Authentication Key) under Settings → SSH and GPG keys.

### Release integrity checklist

Before tagging, confirm:

- [ ] `go.mod` `go` directive == `flake.nix` `GOTOOLCHAIN` == `ci.yml` `go-version` (no version split-brain)
- [ ] `[Unreleased]` is empty and the new version section is dated
- [ ] Every file the changelog names has been opened and the claim verified
- [ ] `golangci-lint run` exits 0 (exit code captured directly, not via pipe)
- [ ] `go test -race ./...` exits 0
- [ ] `go generate ./...` produces no diff (generated code in sync)
- [ ] Dependabot/security alerts triaged or explicitly deferred

## Questions?

Open a [GitHub Issue](https://github.com/larsartmann/samber-do-auditlog/issues) or start a [Discussion](https://github.com/larsartmann/samber-do-auditlog/discussions).
