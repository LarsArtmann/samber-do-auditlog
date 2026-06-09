# Status Report — samber-do-auditlog

**Date:** 2026-06-09 23:18  
**Branch:** master (up to date with origin/master)  
**Build:** passing | **Tests:** 17/17 PASS | **Lint:** 0 issues | **Coverage:** not measured

---

## A) FULLY DONE

| Item            | Detail                                                                                                                          |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Core recorder   | Stack-based dependency inference, sequence numbers, scope tree, shutdown duration                                               |
| Type model      | `Event` (with ContainerID), `ServiceInfo` (with FirstBuildDurationMs, ShutdownDurationMs), `Report` (with Version), `ScopeNode` |
| Plugin API      | `New()`, `Opts()`, `Report()`, `Events()`, 3x writer methods, 3x file export methods                                            |
| Env var toggle  | `DO_AUDITLOG_ENABLED` with 7 value tests + explicit override                                                                    |
| HTML visualizer | Self-contained templ-based dark page with stats, services table, dependency graph, timeline, events table                       |
| Lint config     | `.golangci.yml` with 90+ linters, 0 issues on production code                                                                   |
| Example app     | 5-service demo with `Enabled: true`, JSON/NDJSON/HTML export                                                                    |
| Git hygiene     | Generated files in `.gitignore`, clean history, 22 commits total                                                                |

---

## B) PARTIALLY DONE

| Item          | What's Done                                                                                   | What's Missing                                                                                                                                                                                   |
| ------------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| README        | Good structure, alpha banner, API table                                                       | **Stale data model** — still shows `service_type`, `build_duration_ms` (renamed to `first_build_duration_ms`), missing `version`, `container_id` on Event, `shutdown_duration_ms` on ServiceInfo |
| Test coverage | 17 tests covering registration, invocation, deps, shutdown, export, env var, transient, value | No test for **shutdown duration** being recorded in ServiceInfo, no test for **ContainerID on Event**, no test for **Version on Report**, no test for **non-root scope dependencies**            |

---

## C) NOT STARTED

| Item                                                                      | Impact                              | Effort |
| ------------------------------------------------------------------------- | ----------------------------------- | ------ |
| Update README data model to match current types                           | High — docs are lying               | S      |
| Add missing test assertions (shutdown duration, ContainerID, Version)     | High — untested code is broken code | S      |
| Test coverage report (`go test -cover`)                                   | Medium — blind spot                 | S      |
| Add `go test -cover` to CI or justfile                                    | Medium                              | S      |
| Sort dependencies/dependents in ServiceInfo for deterministic output      | Medium                              | S      |
| DependencyRef should include ScopeID for uniqueness                       | Medium — ScopeName can be ambiguous | S      |
| ExportToHTML is missing from README API table                             | Medium                              | S      |
| Add `context.Context` support to WriteHTML                                | Low — advanced use                  | M      |
| Thread-safe benchmark for concurrent invocations                          | Low                                 | M      |
| Add `ProvideOverride` test                                                | Low                                 | S      |
| Test multi-scope dependency tracking (child invoking parent service)      | Medium                              | S      |
| Add `.golangci.yml` line-length exemptions for generated templ strings    | Low                                 | S      |
| Investigate `html/template` import in html.go (LSP reports broken import) | Medium — might be dead import       | S      |

---

## D) TOTALLY FUCKED UP

| Item                                                                    | Severity | Detail                                                                                                                                                                                              |
| ----------------------------------------------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **README data model is stale**                                          | HIGH     | Still documents `service_type` (removed), `build_duration_ms` (renamed), missing `version`, `container_id` on Event, `shutdown_duration_ms`. Anyone reading the README will get wrong expectations. |
| **README JSON example shows `service_type: "lazy"`**                    | HIGH     | That field doesn't exist anymore. Copy-paste from README will mislead.                                                                                                                              |
| **README claims "Zero extra deps"**                                     | MEDIUM   | We now depend on `github.com/a-h/templ` (indirect). Not zero anymore.                                                                                                                               |
| **README still says "Go 1.22+"**                                        | LOW      | `go.mod` says `1.26.3`, and code uses `slices.Backward` (1.24+).                                                                                                                                    |
| **example/main.go Connect()/Start() return errors that are always nil** | LOW      | Linter noise (unparam), but misleading API in the example.                                                                                                                                          |
| **`html.go` has dead import `html/template`**                           | LOW      | The LSP reports a broken import — likely leftover from the templ migration.                                                                                                                         |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **3 event constructors are near-identical** — `newRegistrationEvent`, `newInvocationEvent`, `newShutdownEvent` differ only in EventType and whether DurationMs/Error are set. A single `newEvent` with functional options or a builder would eliminate ~60 lines of duplication.
2. **Key construction repeated** — `scope.ID() + "/" + serviceName` appears 6 times. Extract to a helper.
3. **`buildScopeTreeLocked` handles empty scopes poorly** — if no scopes recorded, returns a zero-value `ScopeNode` with empty fields. Should return a sensible empty state or document the contract.
4. **Mutex nesting** — `OnBeforeInvocation` takes `stackMu` then `mu` (lines 232-242). `recordInvocationResult` takes `mu` then `invocationMu` (lines 298-315). This is technically safe (consistent order within each path) but fragile. Document the lock ordering.

### Type Model

5. **`DependencyRef` has ScopeName but not ScopeID** — two scopes can have the same name. For machine consumers, ScopeID is unambiguous. Add `ScopeID string` to `DependencyRef`.
6. **`ServiceInfo.RegisteredAt` is set on `HookAfterRegistration`** — but "registered" in samber/do means `Provide` was called. The timing is when the hook fires, not when the user called `Provide`. Document this subtlety.
7. **`Event` has `ContainerID` but `ServiceInfo` doesn't** — if you only look at `ServiceInfo`, you can't tell which container it came from. For multi-container NDJSON this is fine (Event has it), but for the services array in the Report it's ambiguous.
8. **`Report.Services` is `[]ServiceInfo` (not `omitempty`)** — empty container returns `"services": []` which is correct, but inconsistent with `Events` which uses `omitempty`.

### Testing

9. **No test for shutdown duration in ServiceInfo** — `TestPlugin_ShutdownTracking` only checks `ShutdownAt != nil` and event count. Should assert `ShutdownDurationMs != nil`.
10. **No test for Event.ContainerID** — was added but never asserted in tests.
11. **No test for Report.Version** — was added but never asserted.
12. **No parallel test coverage** — `TestPlugin_ProvideTransient` uses `t.Parallel()` but most tests don't. Should add it to all safe tests.
13. **Benchmarks use `for range b.N`** — Go 1.26 has `b.Loop()`, but this is cosmetic.

### DX / Ecosystem

14. **No `go test -cover` baseline** — we're flying blind on coverage.
15. **No CI/CD** — no GitHub Actions, no PR checks.
16. **LICENSE mismatch** — file says "PROPRIETARY", README says "MIT". Pick one.

---

## F) TOP 25 NEXT ACTIONS (sorted by impact/effort)

| #   | Action                                                                                                                               | Impact | Effort | Category     |
| --- | ------------------------------------------------------------------------------------------------------------------------------------ | ------ | ------ | ------------ |
| 1   | **Fix README data model** — remove `service_type`, rename `build_duration_ms`, add `version`, `container_id`, `shutdown_duration_ms` | HIGH   | S      | Docs         |
| 2   | **Fix README JSON example** — remove `service_type: "lazy"`, update field names                                                      | HIGH   | S      | Docs         |
| 3   | **Fix README "Zero extra deps" claim** — we now depend on `a-h/templ`                                                                | MEDIUM | S      | Docs         |
| 4   | **Fix README Go version** — 1.24+ (for `slices.Backward`), not 1.22+                                                                 | MEDIUM | S      | Docs         |
| 5   | **Add ScopeID to DependencyRef** — machine consumers need unambiguous IDs                                                            | MEDIUM | S      | Type Model   |
| 6   | **Extract scopeKey helper** — `scope.ID() + "/" + serviceName` x6                                                                    | MEDIUM | S      | Architecture |
| 7   | **Add test: shutdown duration in ServiceInfo**                                                                                       | MEDIUM | S      | Testing      |
| 8   | **Add test: Event.ContainerID is set**                                                                                               | MEDIUM | S      | Testing      |
| 9   | **Add test: Report.Version matches SchemaVersion**                                                                                   | MEDIUM | S      | Testing      |
| 10  | **Consolidate 3 event constructors into 1** — eliminate ~60 lines duplication                                                        | MEDIUM | M      | Architecture |
| 11  | **Sort dependencies/dependents in ServiceInfo** — deterministic output                                                               | MEDIUM | S      | Quality      |
| 12  | **Fix dead `html/template` import in html.go**                                                                                       | LOW    | S      | Cleanup      |
| 13  | **Document lock ordering in Recorder**                                                                                               | LOW    | S      | Quality      |
| 14  | **Add `go test -cover` and set baseline**                                                                                            | MEDIUM | S      | Testing      |
| 15  | **Add `t.Parallel()` to all safe tests**                                                                                             | LOW    | S      | Testing      |
| 16  | **Test multi-scope dependency tracking**                                                                                             | MEDIUM | M      | Testing      |
| 17  | **Fix example Connect()/Start() to not return errors**                                                                               | LOW    | S      | Example      |
| 18  | **Update README HTML section** — remove "type tag" and "color-coded nodes" text                                                      | MEDIUM | S      | Docs         |
| 19  | **Add README note about `DO_AUDITLOG_ENABLED` env var**                                                                              | LOW    | S      | Docs         |
| 20  | **Add CI GitHub Action** — `go test`, `golangci-lint run`                                                                            | MEDIUM | M      | DX           |
| 21  | **Fix LICENSE vs README mismatch**                                                                                                   | MEDIUM | S      | Legal        |
| 22  | **Add ExportToHTML to README API table** — it's listed but not in the feature description                                            | LOW    | S      | Docs         |
| 23  | **Resolve `containerID` on ServiceInfo** — either add it or document why it's only on Event                                          | LOW    | S      | Type Model   |
| 24  | **Handle empty ScopeNode in BuildReport** — document or return sensible zero                                                         | LOW    | S      | Robustness   |
| 25  | **Update benchmark results in README** — they may have changed after ServiceType removal                                             | LOW    | S      | Docs         |

---

## G) Top #1 Question

**Should `DependencyRef` include `ScopeID` in addition to `ScopeName`?**

Currently `DependencyRef` only has `ScopeName` and `ServiceName`. Two scopes can have the same name (e.g., two child scopes both named "request"). This makes `DependencyRef` ambiguous for machine consumers. Adding `ScopeID` would make it unique, but breaks the JSON shape for existing consumers.

I recommend adding it as `scope_id` (omitempty) — it's backward-compatible since new fields are ignored by lenient parsers, and the data is already available in `serviceRecord.scopeID`. This affects `types.go`, `recorder.go` (2 places), and `html.go` (JS consumers).
