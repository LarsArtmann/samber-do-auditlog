# Contributing

Thanks for your interest in making do-auditlog better.

This project uses the standard Go toolchain. No Makefile, no Nix shell, no Docker — just Go, `golangci-lint`, and `templ` if you touch the HTML template.

---

## Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [golangci-lint](https://golangci-lint.run/usage/install/) (latest)
- [templ](https://templ.guide/) (only if you edit `html.templ`)

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

# 4. Full lint (strict config — this is the gatekeeper)
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
    plugin := auditlog.New(auditlog.Config{Enabled: true})
    injector := do.NewWithOpts(plugin.Opts())

    // ... exercise the feature ...

    report := plugin.Report()
    if report.ServiceCount != 1 {
        t.Fatalf("expected 1 service, got %d", report.ServiceCount)
    }
}
```

## Generated Code

`html_templ.go` is generated from `html.templ` by `templ generate`.

- **Never edit `html_templ.go` by hand.**
- If you modify `html.templ`, run `go generate ./...` and include the regenerated file in your commit.

## Documentation

If you add or change user-facing behavior, update:

- `README.md` — for users
- `FEATURES.md` — add the feature to the inventory
- `CHANGELOG.md` — under `[Unreleased]`
- `docs/DOMAIN_LANGUAGE.md` — if you introduce a new domain concept

## Commit Messages

Write clear, imperative commit messages that explain *why*, not just *what*:

```
Add health check event support for samber/do v2

samber/do v2 does not expose HookBeforeHealthCheck, so we wrap
injector.HealthCheckWithContext() instead. This records EventTypeHealthCheck
events and updates ServiceInfo health fields without modifying the core hook flow.
```

## Questions?

Open a [GitHub Issue](https://github.com/larsartmann/samber-do-auditlog/issues) or start a [Discussion](https://github.com/larsartmann/samber-do-auditlog/discussions).
