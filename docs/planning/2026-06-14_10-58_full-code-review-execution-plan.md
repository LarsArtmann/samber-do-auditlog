# Full Code Review — Execution Plan

Generated: 2026-06-14 10:58

## Context

Repo: `github.com/larsartmann/samber-do-auditlog` — Go plugin for samber/do v2 that records DI lifecycle events and exports JSON/NDJSON/HTML reports.

Current state:

- `go test ./...` passes.
- `golangci-lint run ./...` fails with 6 `thelper` warnings in `helpers_test.go`.
- Example `main.go` has a confirmed bug: the `OnEvent` callback captures a local slice that is returned by value, so `printSummary` sees an empty event log.

## Pareto Breakdown

### 1% → 51% impact (do first)

These few fixes eliminate real bugs, security holes, and lint failure.

| #   | Task                                                                              | Impact                                 | Effort | Files                         |
| --- | --------------------------------------------------------------------------------- | -------------------------------------- | ------ | ----------------------------- |
| 1   | Fix example `main.go` eventLog slice capture bug                                  | Critical — demo is broken              | 10 min | `example/main.go`             |
| 2   | Fix `TestReport_AllHealthChecksPassed_AllHealthy` (service lacks `HealthCheck()`) | Critical — test asserts wrong behavior | 10 min | `healthcheck_export_test.go`  |
| 3   | Fix `thelper` warnings so lint passes                                             | Major — CI gate                        | 15 min | `helpers_test.go`             |
| 4   | Fix HTML XSS vectors (`status`, `event_type` injected unescaped)                  | Critical — security                    | 30 min | `html.templ`, `html_templ.go` |

### 4% → 64% impact (do next)

| #   | Task                                                                                  | Impact                 | Effort | Files                                    |
| --- | ------------------------------------------------------------------------------------- | ---------------------- | ------ | ---------------------------------------- |
| 5   | Deduplicate `failWriter` in `diagram_test.go` and use shared `failingWriter`          | Major — split brain    | 10 min | `diagram_test.go`                        |
| 6   | Deduplicate mermaid/plantuml rendering logic                                          | Major — DRY/complexity | 30 min | `mermaid.go`, `plantuml.go`, `export.go` |
| 7   | Remove/rename duplicate/wasted tests in `plugin_export_test.go`                       | Major — test quality   | 15 min | `plugin_export_test.go`                  |
| 8   | Fix misnamed test `TestPlugin_ShutdownError` and strengthen provider-error assertions | Major — test honesty   | 15 min | `plugin_errors_test.go`                  |
| 9   | Rename `Example_validate` to `ExampleConfig_Validate` for godoc                       | Minor — docs           | 5 min  | `example_test.go`                        |

### 20% → 80% impact (do if time permits)

| #   | Task                                                                  | Impact                  | Effort | Files                                         |
| --- | --------------------------------------------------------------------- | ----------------------- | ------ | --------------------------------------------- |
| 10  | Strengthen weak/vacuous assertions across test suite                  | Minor — test confidence | 30 min | multiple `*_test.go`                          |
| 11  | Remove or consolidate `time.Sleep` in test helpers                    | Minor — speed/flakiness | 20 min | `helpers_test.go`, `plugin_lifecycle_test.go` |
| 12  | Clean up example `register.go` unused scopes and demo sleeps          | Minor — demo quality    | 15 min | `example/register.go`, `example/main.go`      |
| 13  | Improve HTML accessibility (ARIA, labels, keyboard)                   | Minor — a11y            | 30 min | `html.templ`, `html_templ.go`                 |
| 14  | Address `auditlog_test.go` being an empty file with only package docs | Minor — housekeeping    | 10 min | `auditlog_test.go`                            |

## Execution D2 Graph

```d2
direction: down

bugfixes: Bug Fixes {
  main_eventlog: Fix example eventLog bug
  healthcheck_test: Fix all-healthy health-check test
  html_xss: Fix HTML XSS vectors
}

quality_gates: Quality Gates {
  thelper: Fix thelper warnings
  lint: golangci-lint passes
  tests: go test ./... passes
}

refactoring: Refactoring {
  dedup_diagram_writer: Deduplicate failWriter
  dedup_diagram_render: Deduplicate mermaid/plantuml
  clean_tests: Clean misnamed/duplicate tests
}

polish: Polish {
  assertions: Strengthen weak assertions
  sleeps: Remove test sleeps
  example: Clean example demo
  a11y: HTML accessibility
  auditlog_test: Fix empty auditlog_test.go
}

bugfixes.main_eventlog -> quality_gates.tests
bugfixes.healthcheck_test -> quality_gates.tests
bugfixes.html_xss -> quality_gates.tests
quality_gates.thelper -> quality_gates.lint
refactoring.dedup_diagram_writer -> quality_gates.tests
refactoring.dedup_diagram_render -> quality_gates.tests
refactoring.clean_tests -> quality_gates.tests
polish.assertions -> quality_gates.tests
polish.sleeps -> quality_gates.tests
polish.example -> quality_gates.tests
polish.a11y -> quality_gates.tests
polish.auditlog_test -> quality_gates.tests
```

## Verification Command

```bash
go test ./... && golangci-lint run ./...
```

## Definition of Done

- [ ] All critical/major items in 1% and 4% sections are fixed.
- [ ] `go test ./...` passes.
- [ ] `golangci-lint run ./...` passes.
- [ ] Example runs and prints the OnEvent invocation list.
- [ ] Regenerated `html_templ.go` matches `html.templ`.
- [ ] No new warnings or test failures introduced.
