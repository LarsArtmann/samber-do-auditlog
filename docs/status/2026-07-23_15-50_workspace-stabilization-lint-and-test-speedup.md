# Status Report: auditlog-core Workspace Stabilization, Lint Hygiene & Test Speedup

> Session: 2026-07-23 15:50
> Scope: Fixed broken builds, created go.work workspace, added lint config, sped up tests, added integration/benchmarks/docs

---

## a) FULLY DONE

- [x] **go-workflow-auditlog/live build FIXED** — `live/go.mod` was missing `replace github.com/larsartmann/auditlog-core => ../../auditlog-core` (build broke silently; the root module had it but the live sub-module didn't). Added the directive + `go mod tidy`. Root cause: multi-module repo, `go test ./...` at root doesn't cover `live/`.
- [x] **`SetIndent` API call FIXED** — `go-workflow-auditlog/live/server.go:144` called `encoder.SetIndent("", "  ")` which doesn't exist on `jsontext.Encoder`. Replaced with `jsontext.NewEncoder(&buf, jsontext.WithIndent("  "))`. This was a broken build that the previous session's summary missed.
- [x] **Import ordering FIXED** — `gci` lint failure: `encoding/json/jsontext` was in a separate group below `encoding/json/v2`. Merged into the stdlib block.
- [x] **go.work workspace CREATED** — `/home/lars/projects/go.work` links all 5 modules (`auditlog-core`, `samber-do-auditlog`, `go-workflow-auditlog`, `go-workflow-auditlog/live`, `go-workflow-auditlog/viz`). Eliminates fragile `replace` directives for local development. Wrote `WORKSPACE.md` documenting setup.
- [x] **`.golangci.yml` for auditlog-core CREATED** — Full lint config matching sibling standards (samber-do/go-workflow). Surfaces the same linters: `bodyclose`, `exhaustruct`, `depguard`, `wrapcheck`, `nlreturn`, `wsl_v5`, `varnamelen`, etc. Previously auditlog-core had no config (ran defaults only → false "0 issues").
- [x] **40 lint issues FIXED in auditlog-core** — The new config surfaced 40 real issues. Fixed via `golangci-lint --fix` + manual: `exhaustruct` (healthResponse missing fields), unused `nolint:exhaustruct` directive, `gci` formatting, `wsl_v5` whitespace, `nlreturn` blank lines before returns, `varnamelen` ignore list (added `s`, `w`, `r`), `depguard` test rule (allow self-import).
- [x] **SSE test deadlock FIXED — 835x speedup** — Root cause: `sseConnect` used `t.Cleanup` to close `resp.Body`, but `t.Cleanup` runs AFTER `ts.Close()`, creating a deadlock (httptest server waits for active connections, body close waits for server). Replaced with a returned `cleanup func()` + `defer closeSSE()`. **samber-do: 5.011s → 0.006s. go-workflow: 5.018s → 0.006s.** Applied to both consumer projects.
- [x] **`bodyclose` nolint added** — The returned-cleanup pattern triggers a false positive on `http.DefaultClient.Do(req)`. Added `//nolint:bodyclose // closed via returned cleanup` with explanation.
- [x] **`context.Context` added to `WriteToFile`** — `auditlog-core/helpers.go`: signature changed to `WriteToFile(ctx context.Context, path string, fn func(io.Writer) error)`. Context checked before write and before atomic rename. Temp files cleaned up on cancellation. Updated all callers in test file.
- [x] **Integration test ADDED** — `auditlog-core/live/integration_test.go`: Full E2E lifecycle — connect SSE, verify snapshot, push 3 events, verify each received, signal complete, verify complete event, verify report endpoint, verify health endpoint. Passes in 0.004s.
- [x] **Benchmarks ADDED** — `auditlog-core/live/benchmarks_test.go`: `BenchmarkHub_OnEvent` (1/10/100/1000 subscribers — zero allocations on hot path), `BenchmarkHub_SubscribeUnsubscribe`, `BenchmarkServer_ServeHTTP_Dashboard/Report/Health`. Results: 30ns/op for 1 subscriber, 0 allocs.
- [x] **Downstream AGENTS.md UPDATED** — Both `go-workflow-auditlog/AGENTS.md` and `samber-do-auditlog/AGENTS.md` now document their dependency on `auditlog-core`, the provider pattern, and the `go.work` workspace.
- [x] **auditlog-core docs ADDED** — `docs/DOMAIN_LANGUAGE.md` (ubiquitous language), `docs/adr-001-extraction-decision.md` (ADR explaining why core was extracted), `CODEOWNERS`, updated `AGENTS.md` with lint config + WriteToFile signature change.
- [x] **ALL THREE PROJECTS VERIFIED** — Build: OK. Tests: pass with `-race`. Lint: 0 issues. Across auditlog-core (root+live), samber-do-auditlog (root+live+cmd), go-workflow-auditlog (root+live).

## b) PARTIALLY DONE

- [ ] **go-workflow `WriteToFile` still duplicated** — The core's `WriteToFile` now takes `context.Context`, but go-workflow's `helpers.go` still has its own copy with the old signature (`func WriteToFile(path string, fn func(io.Writer) error) error`). The `go-workflow-auditlog/helpers.go:82` duplicate is ~45 LOC that should delegate to `auditlogcore.WriteToFile`. The reason it wasn't consolidated: go-workflow's root module doesn't currently import `auditlog-core` (only `live/` does), and the callers (`ExportJSON`, `ExportNDJSON`, viz exporters) don't have `context.Context` in their signatures yet. Full consolidation requires cascading context through the entire export API surface.
- [ ] **Samber-do's `WriteToFile` — not affected** — samber-do doesn't use `WriteToFile` at all (no file exports in that project). So no duplication to fix there.

## c) NOT STARTED

- [ ] **Extract NDJSON read/write into `auditlog-core/ndjson/`** — ~150 LOC duplicated across both consumers. Blocked: go-workflow uses `encoding/json/v2` for NDJSON, samber-do uses standard `encoding/json`. Extraction requires standardizing on one JSON library.
- [ ] **Extract format detection/loader into `auditlog-core/loader/`** — Same blocker as NDJSON (json/v2 divergence).
- [ ] **Publish `auditlog-core` to GitHub with tag `v0.1.0`** — Module exists on GitHub but sum verification fails (`sum.golang.org` returns 500). Needs `GONOSUMCHECK` or `GONOSUMDB` or `GOFLAGS=-insecure` or proper publishing. The `replace` directives remain in all `go.mod` files until this is resolved.
- [ ] **Remove `replace` directives** — Blocked by publish.
- [ ] **`.github/workflows/ci.yml` for auditlog-core** — No CI exists. Should run `go test ./... -race -count=1` + `golangci-lint run`.
- [ ] **Cross-repo CI workflow** — Tests all three projects together to catch breaking changes in core.
- [ ] **`go test -race` in CI** — Currently only run manually.
- [ ] **`flake.nix` for auditlog-core** — No flake exists (intentionally minimal, but means no `nix run .#check` parity).
- [ ] **`FEATURES.md` updates in both consumers** — Both `FEATURES.md` files should reflect the extraction.
- [ ] **`docs/DOMAIN_LANGUAGE.md` for samber-do** — Already exists for go-workflow. samber-do lacks one.
- [ ] **Migrate go-workflow dashboard to standard `encoding/json`** — Would unblock NDJSON/loader extraction and remove `GOEXPERIMENT=jsonv2` requirement for the live module.
- [ ] **`MIGRATION.md` for downstream projects** — Guide for upgrading from in-tree Hub/Server to core.
- [ ] **`auditlog-core/examples/minimal`** — Runnable demo for adoption.

## d) TOTALLY FUCKED UP

- **The report from the previous session (14:42) was significantly stale** — It claimed "No `README.md`, no `CONTRIBUTING.md`, no `LICENSE`, no `CODEOWNERS`" for auditlog-core. In reality, README.md (2652 bytes), CONTRIBUTING.md, and LICENSE already existed. Only CODEOWNERS was missing. I trusted the report initially and almost recreated files that already existed. **Lesson: ALWAYS `ls` the actual files before acting on a status report's claims. Reports go stale between sessions.**
- **The `live/go.mod` build break was invisible to the previous session** — The 14:42 report claimed "all tests pass" but the live sub-module couldn't even build (`auditlog-core/live` not required). The previous session ran `go test ./...` at the project root, which doesn't cover sub-modules. The live tests "passed" only because they were cached from an earlier run. **Lesson (already noted in the 14:42 report but not fixed): for multi-module repos, run `go test ./...` in EACH module's root, not just the project root.**
- **My `sseConnect` fix initially hung for 600 seconds** — My first attempt replaced `context.WithTimeout(5s)` with `t.Context()`. But `t.Context()` is cancelled when the test finishes, and `t.Cleanup` runs in reverse order — `ts.Close()` (from `defer`) ran BEFORE `resp.Body.Close()` (from `t.Cleanup`), deadlocking. The httptest server waits for active connections, and the body close waits for the server. I had to kill the test run. Fixed by returning a `cleanup func()` instead of using `t.Cleanup`, so `defer closeSSE()` runs before `defer ts.Close()`. **Lesson: `t.Cleanup` runs in LIFO order AFTER all defers. For SSE tests, body close must happen before server close.**
- **The `go-workflow/live/go.mod` `replace` directive path was `../../auditlog-core`** — I initially tried `go get github.com/larsartmann/auditlog-core` which tried to fetch from the network and failed (sum verification error). Had to use a local `replace` directive instead. The `replace` path is `../../auditlog-core` (relative to `live/`), not `../auditlog-core` (relative to project root). Easy to get wrong.

## e) WHAT WE SHOULD IMPROVE

1. **Multi-module test runner script** — We need a script (or Makefile target, or flake.nix check) that walks the repo tree and runs `go test ./...` in every directory containing a `go.mod`. This would have caught the `live/go.mod` break immediately. Both go-workflow and samber-do have sub-modules.
2. **`GOEXPERIMENT=jsonv2` must be documented as a project-level requirement** — It's in `.golangci.yml` build-tags, but `go test` and `go build` need it too. The `WORKSPACE.md` I created documents it, but a `.envrc` (direnv) or `make` wrapper would be better. Currently developers must remember to `export GOEXPERIMENT=jsonv2`.
3. **The `sseConnect` pattern should be in auditlog-core as a test helper** — Both consumers have identical `sseConnect` + `readSSEEvent` + `skipSnapshot` helpers. These should live in `auditlog-core/live/testutil` or similar, imported by both consumer test suites. This is the next-biggest duplication after NDJSON.
4. **`WriteToFile` context cascade** — The core now takes `context.Context`, but go-workflow's callers (`ExportJSON`, `ExportNDJSON`, viz exporters) don't expose context. This is an API design gap. Either: (a) add `ctx` to all export methods, or (b) provide both `ExportJSON(path)` and `ExportJSONCtx(ctx, path)`. Decision needed.
5. **Benchmark coverage is thin** — I added Hub and HTTP handler benchmarks, but there's no SSE end-to-end benchmark (connect, receive events, disconnect). The SSE handler is the most complex code path and has no perf coverage.
6. **The `HealthInfo` type name is still suboptimal** — The 14:42 report noted `HealthInfo` should be `HealthStatus` or `HealthResponse`. I didn't fix this because it's a public API rename that affects both consumers. Should be done before v0.1.0 tag.
7. **`Subscriber` is still concrete, not an interface** — The 14:42 report suggested making `Subscriber` an interface for testability. I didn't do this. It would allow mock subscribers in tests.
8. **No concurrent-access test for `WriteToFile`** — I added context support but no test for concurrent writes to the same path. The atomic rename should handle this, but it's untested.
9. **The `go.work` file is not gitignored** — It's in `/home/lars/projects/` (the parent directory), not in any of the three repos. It works for local dev but will confuse `go mod tidy` in CI. Need a strategy: either commit `go.work` to a meta-repo, or document that developers create it locally.
10. **The integration test doesn't test reconnection** — My integration test covers connect → events → complete, but not: disconnect, reconnect, verify snapshot includes all past events. This is the key SSE recovery path.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

| #  | Task                                                                                                      | Impact | Effort |
| -- | --------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1  | Publish `auditlog-core` to GitHub (fix sum verification — `GONOSUMDB`/`GONOSUMCHECK`)                      | High   | S      |
| 2  | Tag `auditlog-core` `v0.1.0`                                                                              | High   | XS     |
| 3  | Remove `replace` directives from all `go.mod` files after publish                                         | High   | XS     |
| 4  | Consolidate go-workflow `helpers.go:WriteToFile` to delegate to `auditlogcore.WriteToFile`                | High   | S      |
| 5  | Cascade `context.Context` through go-workflow export API (`ExportJSON`, `ExportNDJSON`)                   | Medium | M      |
| 6  | Extract `sseConnect`/`readSSEEvent`/`skipSnapshot` into `auditlog-core/live/testutil`                     | High   | S      |
| 7  | Standardize on `encoding/json` OR `encoding/json/v2` across all three projects                            | High   | M      |
| 8  | Extract NDJSON read/write into `auditlog-core/ndjson/` (after #7)                                         | High   | M      |
| 9  | Extract format detection/loader into `auditlog-core/loader/` (after #7)                                   | High   | M      |
| 10 | Add `.github/workflows/ci.yml` to auditlog-core (build + test -race + lint)                               | High   | S      |
| 11 | Add cross-repo CI that tests all three together                                                           | High   | M      |
| 12 | Add `go test -race -count=1` to all CI pipelines                                                          | High   | XS     |
| 13 | Add reconnection test to integration test (disconnect, reconnect, verify snapshot)                        | High   | S      |
| 14 | Add concurrent-write test for `WriteToFile`                                                               | Medium | XS     |
| 15 | Add SSE end-to-end benchmark (connect → N events → disconnect)                                            | Medium | S      |
| 16 | Rename `HealthInfo` → `HealthResponse` before v0.1.0 (breaking change, do it now)                         | Medium | S      |
| 17 | Make `Subscriber` an interface (`ID()`, `Events()`, `Done()`)                                             | Low    | S      |
| 18 | Create multi-module test runner script (`find . -name go.mod -execdir go test ./... \;`)                  | High   | XS     |
| 19 | Add `.envrc` with `export GOEXPERIMENT=jsonv2` to all three projects                                      | Medium | XS     |
| 20 | Migrate go-workflow dashboard.go to standard `encoding/json` (drop json/v2)                               | High   | M      |
| 21 | Add `//go:build goexperiment.jsonv2` constraint OR remove json/v2 from go-workflow                        | High   | S      |
| 22 | Update both `FEATURES.md` to reflect auditlog-core extraction                                             | Low    | XS     |
| 23 | Add `docs/DOMAIN_LANGUAGE.md` to samber-do-auditlog                                                       | Low    | S      |
| 24 | Write `MIGRATION.md` for downstream projects upgrading to core                                            | Medium | S      |
| 25 | Add `auditlog-core/examples/minimal` runnable demo                                                        | Medium | S      |
| 26 | Add `auditlog-core/cmd/auditlog-core-demo` CLI                                                            | Low    | M      |
| 27 | Add `flake.nix` to auditlog-core for devShell parity                                                      | Low    | S      |
| 28 | Add `go mod tidy` to pre-commit hooks                                                                     | Medium | XS     |
| 29 | Add Prometheus metrics interface (events-sent, clients-connected)                                         | Low    | M      |
| 30 | Add `OnSubscribe`/`OnUnsubscribe` callbacks to Hub for metrics                                            | Low    | XS     |
| 31 | Add `Server.Handle(pattern, handler)` for extensibility                                                   | Low    | S      |
| 32 | Document `SnapshotProvider`/`CompleteProvider` lifecycle in README                                        | Medium | XS     |
| 33 | Add `ErrInvalidPrefix` sentinel for malformed route prefixes                                              | Low    | XS     |
| 34 | Add `WithHeartbeatInterval` as a public test helper option                                                | Low    | XS     |
| 35 | Consolidate `makeReportProvider`/`Snapshot`/`Complete`/`Health` factories into generic helper             | Low    | S      |
| 36 | Refactor `With*Provider` options to use `Option func(*Server) error`                                      | Low    | S      |
| 37 | Add test for `handleReport` returning nil provider error                                                  | Medium | XS     |
| 38 | Add test for SSE handler when `Flusher` assertion fails                                                   | Low    | XS     |
| 39 | Add test for `WriteToFile` directory-creation failure path                                                | Low    | XS     |
| 40 | Tag go-workflow and samber-do versions that use core `v0.1.0`                                             | High   | XS     |
| 41 | Verify both `replace` directives are path-consistent (catch drift)                                        | Low    | XS     |
| 42 | Add `go work sync` to CI to keep workspace in sync                                                        | Low    | XS     |
| 43 | Create `auditlog-core/CHANGELOG.md` entry for v0.1.0                                                      | Medium | XS     |
| 44 | Add `healthResponse` version field for API stability                                                      | Low    | XS     |
| 45 | Add `context.Context` to `CheckNoClobber` (currently takes only path)                                     | Low    | XS     |
| 46 | Add SSE backpressure test (fill buffer, verify drop, verify no block)                                     | Medium | S      |
| 47 | Add `go vet` to CI alongside golangci-lint                                                                | Low    | XS     |
| 48 | Add `gosec` to CI for security scanning                                                                   | Low    | XS     |
| 49 | Add `govulncheck` to CI for vulnerability scanning                                                        | Low    | XS     |
| 50 | Create architecture diagram (D2 or Mermaid) showing all three projects + dependencies                     | Low    | S      |

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Should I publish `auditlog-core` to GitHub NOW (with `GONOSUMDB` workaround for sum verification) and remove all `replace` directives, OR wait until the json/v2 divergence is resolved first?** Publishing now locks in the current API (including `HealthInfo` name, `Subscriber` concrete type) before the improvements in section (e) are made. Publishing later means the `go.work` workspace remains the only way to develop locally. The tradeoff is API stability vs. development convenience.

2. **Should go-workflow's root module (not just `live/`) depend on `auditlog-core`?** Currently only `live/go.mod` imports core. But the root module has its own duplicate `WriteToFile` (`helpers.go:82`) that should delegate to core. Adding core as a root dependency would allow consolidation, but it also means the root module (which has no json/v2 dependency today) would pull in a module that's still pre-v1. Is this acceptable, or should the root module stay decoupled until core is published?

3. **Should the `go.work` file live in `/home/lars/projects/` (where it is now, covering all repos) or should each repo have its own smaller `go.work`?** The current setup works for local dev but `go.work` in a parent directory affects ALL Go projects under it (I verified this — `cd AI-Speed-Test && go list ./...` shows the workspace warning). This could cause confusion if unrelated projects are added to `/home/lars/projects/` later. Should I move it, gitignore it, or leave it?
