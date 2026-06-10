# Full Code Review & Architecture Review

**Date**: 2026-06-10 · **Reviewer**: Senior Staff Architect · **Project**: samber-do-auditlog

---

## Executive Summary

This is a well-crafted, single-purpose Go library with clean architecture, honest naming, and solid test coverage. The codebase is **1757 lines** across **7 source files** (plus 1 generated). Build, lint, and all 17 tests pass cleanly. The project is in ALPHA status and already demonstrates high engineering quality.

**Overall Grade: A-** — Excellent for an alpha project. The issues below are refinements, not flaws.

---

## File-by-File Review

### `types.go` (83 lines) — Grade: A

**Strengths:**

- Clean domain types with no behavior leakage
- Proper use of pointer types for optional fields (`*float64`, `*string`, `*time.Time`)
- Snake_case JSON tags (consistent with tagliatelle config)
- `SchemaVersion` for forward compatibility
- `DependencyRef` is a proper value type for cross-references

**Observations:**

- `ScopeNode` uses recursive structure — well-suited for tree serialization
- `Event` is the core observation unit — clean, immutable-friendly design
- `Report` is the aggregation root — appropriate structure

**Minor Notes:**

- No `Validate()` method on `Config` — validation is ad-hoc in `New()`. Consider a `Validate() error` method for cleaner API surface.
- `ServiceInfo` has both `InvocationError` and `ShutdownError` — honest, tracks both lifecycle phases correctly.

### `plugin.go` (142 lines) — Grade: A

**Strengths:**

- Clean public API surface: `New()`, `Opts()`, `Report()`, export methods
- Disabled mode returns empty `InjectorOpts` — truly zero-cost
- Environment variable fallback is well-documented and handles all common truthy values
- `writeToFile` helper eliminates duplication across 3 export methods
- Error wrapping with `%w` throughout

**Observations:**

- `envIsEnabled()` uses `os.Getenv` — not testable directly, but tests use `t.Setenv()` which works fine
- `writeToFile` defers close and ignores close error — acceptable for write-only files
- `Config.Enabled` has env var fallback but no `Config.Validate()` — the zero-value means "check env"

**Potential improvements:**

- `Plugin` could implement an `io.WriterTo` interface for streaming scenarios
- No `WriteReportJSONToBuilder` for `strings.Builder` use cases (but `WriteReportJSON` takes `io.Writer` so this is covered)

### `recorder.go` (551 lines) — Grade: A-

**Strengths:**

- Per-recorder atomic sequence counter — no global state, test-safe
- `initialEventCapacity = 1024` — avoids repeated allocations for typical workloads
- Triple-lock design (`mu`, `stackMu`, `invocationMu`) — correctly separates concerns:
  - `mu` protects events/services/scopes (read-heavy)
  - `stackMu` protects invocation stack (write-heavy)
  - `invocationMu` protects invocation ordering counter
- `shutdownMu` + `shutdownStart` map — cleanly tracks shutdown timing
- `buildServicesLocked()` extracted to satisfy funlen limits — good
- `buildDependentsMapLocked()` is a pure function of its input — good testability

**Observations:**

- Line 208: `parentKey := parent.scopeID + "/" + parent.serviceName` duplicates the logic of `scopeKey()` — should use `scopeKey()` for consistency. Actually, it constructs the key manually from the stack entry fields, while `scopeKey` takes a `*do.Scope`. This is a minor inconsistency — the key format is the same but computed differently.
- `buildScopeTreeLocked()` — the root scope lookup iterates all scopes to find `parentID == ""`. If there's no root, it picks an arbitrary scope. This is a potential edge case but acceptable since samber/do always has a root scope.
- `recordInvocationResult` — correctly handles the case where a service is invoked before its registration event arrives (unlikely but defensive)

**Potential improvements:**

- `recorder.go` at 551 lines is the largest file but still well under 350 lines of actual logic (many are short functions with doc comments). The `buildScopeTreeLocked` method is the most complex — could potentially extract scope tree building to a separate file.
- The `stackEntry` struct could benefit from a `key() string` method to centralize the scope+service key format.

### `html.go` (24 lines) — Grade: A

**Strengths:**

- Thin wrapper around templ-generated code
- Proper error wrapping
- Uses `writeToFile` pattern consistently

### `html.templ` (334 lines) — Grade: A-

**Strengths:**

- Self-contained HTML — no external JS/CSS dependencies
- Dark theme with CSS variables — consistent, maintainable
- Force-directed graph simulation — pure JS, no D3 dependency
- Uses `templ.JSONScript` for safe JSON embedding (XSS protection)
- Tab-based UI with services, graph, timeline, events views
- `esc()` function for HTML entity encoding

**Observations:**

- Line 157: `document.querySelectorAll('.tab').classList &&` — this is a dead condition. `querySelectorAll` always returns a NodeList (which has `classList` as undefined, but `&&` doesn't prevent the forEach). This is harmless but unnecessary code.
- The force simulation (lines 241-270) uses fixed 300 iterations — could be made configurable but fine for current scope.
- Inline styles in JS template strings (`style="width:..."`) — not ideal for CSP but acceptable for self-contained HTML.
- No `lang` attribute consideration beyond `en` — fine for current scope.

### `doc.go` (7 lines) — Grade: A

**Strengths:**

- Concise, accurate package documentation
- Lists all major features

**Note:** Both `doc.go` and `plugin.go` have package-level doc comments, causing a `godoclint` warning (documented in AGENTS.md).

### `auditlog_test.go` (714 lines) — Grade: A-

**Strengths:**

- External test package (`auditlog_test`) — tests the public API, not internals
- Table-driven tests for env var values
- Tests cover: disabled/enabled, env var, explicit override, registration, invocation, dependencies, shutdown, scope tree, export formats, error paths, transient/value providers, benchmarks
- Uses `t.TempDir()` for file tests
- Uses `t.Setenv()` for env var tests
- `findServiceByName` and `findServiceBySuffix` helpers
- `contains`/`searchString` helpers for string matching (avoids importing `strings` for simple checks)

**Observations:**

- `contains()` and `searchString()` reimplement `strings.Contains` — minor duplication but avoids an import for a single use
- Test for `ProvideTransient` and `ProvideValue` verify the library works with all samber/do registration patterns
- No test for concurrent access to `Recorder` — the lock design should be verified under concurrent load
- No test for empty container (no services registered) — edge case

### `example/main.go` (176 lines) — Grade: A

**Strengths:**

- Complete working example showing all features
- Demonstrates: plugin creation, service registration, invocation, shutdown, all 3 export formats
- Clear numbered steps in comments

**Observations:**

- Example types (`Config`, `Database`, `Cache`, etc.) differ from test types — not a problem since they're in different packages
- Uses `fmt.Println` — exempt from `forbidigo` via golangci.yml exclusion

---

## Architecture Analysis

### Scalability & Modularity

**Current state**: Single package, clean separation of concerns via file organization:

- `plugin.go` — Public API facade
- `recorder.go` — Core state machine + event capture
- `types.go` — Domain types (pure data, no behavior)
- `html.go` + `html.templ` — Visualization concern

**Modularity Score: 8/10** — Excellent for a single-package library. The internal separation is clean even without package-level boundaries.

### Service Orientation

The library is inherently a **plugin** (extends samber/do) rather than a standalone service. It composes well:

- `Plugin` → wraps `Recorder`
- `Recorder` → captures events independently
- Export methods → format-agnostic (`io.Writer`)

### Composability

**Strengths:**

- All export methods accept `io.Writer` — composable with any destination
- `Config` is a simple struct — easy to serialize/deserialize
- `Report()` returns a value type — safe to cache, serialize, compare

**Potential:**

- Could add `ReportOption` functional options for filtering (e.g., `OnlyService(name)`, `TimeRange(from, to)`)
- Could add `EventHandler` callback in `Config` for real-time event streaming

### Type Safety

**Strengths:**

- `EventType` and `Phase` are string-based enums — type-safe at the Go level
- Optional fields use pointer types correctly (`*float64`, `*string`, `*time.Time`)
- No `interface{}` / `any` in the public API
- No reflection usage

**Potential improvement:**

- `EventType` and `Phase` could use Go 1.22+ iterator patterns or be unexported string types with public constants (already done correctly)
- Could add `IsRegistration()`, `IsInvocation()`, `IsShutdown()` methods on `Event` for convenience

### Data Flow

```
User → New(Config) → Plugin
  ↓
Plugin.Opts() → do.InjectorOpts with hooks
  ↓
Hooks fire → Recorder.OnBefore/After* → Event captured
  ↓
Plugin.Report() → Recorder.BuildReport() → Report
  ↓
Export methods → io.Writer / file
```

This is a clean, unidirectional data flow. No circular dependencies, no callbacks to user code.

---

## Pareto Analysis

### 1% → 51% Impact

| #   | Task                                              | Impact | Effort |
| --- | ------------------------------------------------- | ------ | ------ |
| 1   | Fill DOMAIN_LANGUAGE.md with actual domain terms  | High   | 15min  |
| 2   | Update CHANGELOG.md to reflect actual development | Medium | 10min  |

### 4% → 64% Impact

| #   | Task                                                                              | Impact | Effort |
| --- | --------------------------------------------------------------------------------- | ------ | ------ |
| 3   | Add concurrent access test for Recorder                                           | High   | 30min  |
| 4   | Add test for empty container edge case                                            | Medium | 15min  |
| 5   | Fix `parentKey` construction in `OnBeforeInvocation` to use consistent key format | Medium | 10min  |
| 6   | Remove dead `classList &&` check in html.templ JS                                 | Low    | 5min   |

### 20% → 80% Impact

| #   | Task                                                                       | Impact | Effort |
| --- | -------------------------------------------------------------------------- | ------ | ------ |
| 7   | Add `ReportOption` functional options for filtering                        | High   | 1h     |
| 8   | Add `EventHandler` callback for real-time streaming                        | High   | 1h     |
| 9   | Extract `stackEntry.key()` method                                          | Low    | 15min  |
| 10  | Add `IsRegistration()`/`IsInvocation()`/`IsShutdown()` convenience methods | Medium | 15min  |
| 11  | Add `Config.Validate() error` method                                       | Medium | 20min  |
| 12  | Consider scope tree builder extraction                                     | Low    | 20min  |

---

## Critical Observations

### Split Brains

- **None detected.** All terms are used consistently throughout the codebase.

### Duplications

- **`parentKey` construction** in `recorder.go:208` vs `scopeKey()` — same key format, different construction path. Minor but should be unified.
- **`contains()`/`searchString()`** in test file reimplements `strings.Contains`. Very minor.

### What We Forgot

- **No LICENSE file content was reviewed** — should verify MIT license text is correct
- **No CONTRIBUTING.md review** — should verify contribution guidelines exist
- **DOMAIN_LANGUAGE.md is a template** — needs to be filled with actual domain terms

### What Could Be Removed

- Dead `classList &&` check in `html.templ:157`
- Template placeholder content in `docs/DOMAIN_LANGUAGE.md`

### What Should Be Extracted

- Nothing urgent — the current file organization is clean for this project size

### Long-term Thinking

- As the library grows, consider splitting `Recorder` into `EventCollector` + `ServiceAggregator`
- The HTML visualization could become its own sub-package if more formats are added (e.g., Mermaid, PlantUML)
- `Report` could evolve into a versioned schema with migration support

---

## Go Modularization Assessment

**Current state**: Single `go.mod`, single package — **this is correct and should NOT be modularized.**

| Signal                        | Weight | Present?             |
| ----------------------------- | ------ | -------------------- |
| Small project (< 10 packages) | High   | YES (1 package)      |
| No external consumers yet     | Medium | YES (ALPHA)          |
| All packages change together  | High   | YES (single package) |

**Score: 3 High signals → Do NOT modularize.** The project is too small and too early in its lifecycle for multi-module splitting. Revisit when:

- The library has 5+ packages
- There are external consumers with different dependency needs
- The HTML visualization grows to warrant its own module

---

## Recommendation Summary

| Priority | Task                                      | Category      |
| -------- | ----------------------------------------- | ------------- |
| P0       | Fill DOMAIN_LANGUAGE.md with actual terms | Documentation |
| P0       | Update CHANGELOG.md                       | Documentation |
| P1       | Add concurrent access test                | Testing       |
| P1       | Fix parentKey consistency                 | Code Quality  |
| P2       | Add Report filtering options              | Feature       |
| P2       | Add EventHandler callback                 | Feature       |
| P3       | Remove dead JS code                       | Cleanup       |
| P3       | Add convenience methods on Event          | API Polish    |
