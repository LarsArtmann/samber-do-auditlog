# Contributing

Thanks for your interest in making do-auditlog better.

This project uses the standard Go toolchain. A `flake.nix` devShell is available for contributors using Nix; otherwise you just need Go and `golangci-lint`. The templ CLI is managed automatically via Go's `tool` directive — no manual install needed.

---

## Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [golangci-lint](https://golangci-lint.run/usage/install/) (latest v2.x)
- [templ](https://templ.guide/) (only if you edit `html.templ`)

**Nix users:** Run `nix develop` to get Go 1.26.3, golangci-lint, govulncheck, and golines pinned in `flake.nix`.

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
- `depguard` — only stdlib, `samber/do`, `a-h/templ`, and this module are allowed
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

- **Release tags** follow `v0.0.x` (e.g. `v0.0.3`). These mark GitHub releases.
- **Schema version** (currently `0.2.0`, in `types.go`) versions the JSON report format. It is upgraded via `MigrateReport` and has no relation to release tags.

### Release Procedure

1. **Update `CHANGELOG.md`** — move `[Unreleased]` items under a new `[0.0.x]` heading with today's date.
2. **Commit** the changelog update.
3. **Tag** the release (signed):
   ```bash
   git tag -s v0.0.x -m "v0.0.x — short description"
   ```
4. **Push** the tag and master:
   ```bash
   git push origin master --tags
   ```
5. **Create a GitHub Release** using `gh release create` with the changelog body as notes. Attach the example HTML artifact:
   ```bash
   DO_AUDITLOG_ENABLED=true go run ./example
   gh release create v0.0.x --notes-file <notes> /tmp/.../audit-report.html
   ```
6. **Verify** the CI badge is green and the release appears on the releases page.

## Questions?

Open a [GitHub Issue](https://github.com/larsartmann/samber-do-auditlog/issues) or start a [Discussion](https://github.com/larsartmann/samber-do-auditlog/discussions).
