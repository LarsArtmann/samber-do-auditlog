# Status Report: Auditlog Core Integration & Lint Hygiene

> Session: 2026-07-23 14:37 - 14:42
> Scope: Build/test/lint verification of auditlog-core integration in both consumers

---

## a) FULLY DONE

- [x] **go-workflow-auditlog build**: Compiles with `GOEXPERIMENT=jsonv2` (fix: removed unused `errors` import from `live/server.go`)
- [x] **go-workflow-auditlog tests**: All 15 `live/` tests pass (10 server + 5 hub, ~5s runtime)
- [x] **go-workflow-auditlog lint**: 0 issues (fixed `depguard` allowlist, `wrapcheck` ignore, `gci` import order, `noinlineerr`, `funlen`, `varnamelen`)
- [x] **samber-do-auditlog lint**: 0 issues (fixed `depguard` allowlist, `wrapcheck` ignore, `funlen`, `varnamelen`)
- [x] **samber-do-auditlog tests**: All packages still pass after refactor (root, live, cmd/auditlog)
- [x] **auditlog-core tests**: All 24 still pass (uncached verified, cached re-verified)
- [x] **Hub wrapper delegation**: Both projects now expose `Subscribe()` / `Unsubscribe()` on wrapper Hub (samber-do had it, go-workflow was missing → aligned)
- [x] **Provider extraction**: Both `NewServer` functions now delegate to named `makeReportProvider` / `makeSnapshotProvider` / `makeCompleteProvider` / `makeHealthProvider` factories (shrinks each constructor under 60-line `funlen` cap)
- [x] **Test correctness**: go-workflow `TestHub_OnEventDelivery` now properly unmarshals `json.RawMessage` before field access (samber-do was already correct, go-workflow was wrong)
- [x] **Committed**: `1e6d510` (go-workflow fixes), `4450f87` (samber-do fixes)
- [x] **BuildFlow pre-commit**: Both commits passed pre-commit hooks with only pre-existing structure warnings remaining (root-package-files, github-actions-pinned)

## b) PARTIALLY DONE

- [ ] **NDJSON/loader extraction** (plan phase 2) — still not started; ~300 LOC of identical code remains duplicated across both projects
- [ ] **`auditlog-core/.golangci.yml`** — Not added; core module still has no lint config (runs are clean because no config means minimal linters, not because the project is lint-clean by neighbor standards)
- [ ] **`auditlog-core` documentation** — No `README.md`, no `CONTRIBUTING.md`, no `LICENSE`, no `CODEOWNERS`
- [ ] **`flake.nix` updates** — Neither project's flake references `../auditlog-core` in devShell
- [ ] **Downstream `AGENTS.md` references** — Neither consumer documents its dependency on `auditlog-core`

## c) NOT STARTED

- [ ] Publish `auditlog-core` to GitHub and remove `replace` directives from both `go.mod` files
- [ ] Extract NDJSON read/write into `auditlog-core/ndjson/`
- [ ] Extract format detection/loader into `auditlog-core/loader/`
- [ ] Add `context.Context` to `WriteToFile` for cancellation
- [ ] Tag initial release `v0.1.0`
- [ ] Create GitHub release notes
- [ ] Run integration test: create core server, connect SSE, send events, verify snapshot
- [ ] Add Hub.OnEvent benchmark with N concurrent subscribers
- [ ] Add Server SSE handler benchmark
- [ ] Migrate go-workflow dashboard HTML to standard `encoding/json` (remove `json/v2` dependency)
- [ ] Or: add `//go:build goexperiment.jsonv2` constraint to dashboard.go
- [ ] Speed up SSE tests (currently 5s each due to 15s heartbeat interval default)
- [ ] Create `docs/DOMAIN_LANGUAGE.md` for auditlog-core
- [ ] Write ADR explaining the extraction decision
- [ ] Set up `go.work` workspace for all three projects
- [ ] Update both `FEATURES.md` to reflect extraction

## d) TOTALLY FUCKED UP

- **Missed missing `Subscribe`/`Unsubscribe` delegation earlier** — The summary at session start claimed "all tests pass" but go-workflow `live/server_test.go` calls `hub.Subscribe()` and the wrapper didn't expose it. Only the `samber-do` wrapper had these methods. The earlier "all tests pass" was based on `samber-do` only. **Lesson learned: when two projects run nearly identical test suites, ALWAYS verify both.**
- **Caught the build but missed the test contract** — I fixed the `errors` unused import for the build, then ran tests but at first only saw "no output" (cached). The `go test -v` would have surfaced the missing `Subscribe` immediately. **Lesson learned: always run `-count=1` for uncached, `-v` for full failure detail, and check ALL projects not just one.**
- **`go-workflow-auditlog/live/go.mod` is its own module** — When I ran `go test ./...` at project root, it didn't include the `live/` sub-module. Tests for the sub-module passed via the cached run from earlier in the session. **Lesson learned: for multi-module repos, run `go test ./...` in each module's root, not just the project root.**
- **`replace` directive is fragile** — Both projects use `replace github.com/larsartmann/auditlog-core => ../auditlog-core`. This works locally but means publishing the core module and testing against a real version requires touching three repos at once. Should use `go.work` instead.
- **`encoding/json/v2` pre-existing issue is a blocker in disguise** — The "pre-existing" `json/v2` build issue masks the fact that go-workflow's dashboard HTML rendering can't be tested in this environment at all. I treated it as "not my problem" but it actually means my extraction refactor was validated against 5 tests of hub behavior and 10 tests of server behavior, NOT against the dashboard rendering. There's still unverified behavior in this code path.
- **Auto-commit cleanup** — During the earlier session, multiple commits were made with auto-generated messages. I amended once but reflog shows the originals still exist as orphans.

## e) WHAT WE SHOULD IMPROVE

1. **Multi-module verification discipline** — For any project with sub-modules (`go-workflow-auditlog/live/`), add a script that walks the tree and runs `go test ./...` in each module. Bake it into CI.
2. **`GOEXPERIMENT=jsonv2` should be a project-level setting** — The `.golangci.yml` has it for linters but `go test`, `go build` need it too. Add a project Makefile or shell wrapper script.
3. **`go.work` instead of `replace`** — Three sibling repos referencing each other is fragile. A top-level `go.work` with `./auditlog-core`, `./samber-do-auditlog`, `./go-workflow-auditlog` would let devs work without managing replace directives.
4. **`golangci-lint` config inheritance** — `auditlog-core` has no config so it lints as a fresh project. Either reuse `.golangci.yml` from one of the siblings, or define a shared base config that all three extend.
5. **The "wait, did I just trust the summary?" failure mode** — The session-start summary said "all tests pass" and I nearly accepted it at face value. The summary was stale relative to today's changes. **ALWAYS re-verify state when starting work, regardless of what summaries claim.**
6. **Provider factories should live in `auditlog-core` as a generic option helper** — Both projects now have `makeReportProvider`, etc. with identical signatures. A `corelive.Provider[*PluginType]` generic helper would remove the boilerplate. But this is over-engineering until we have a third consumer.
7. **Test speed is a real CI cost** — 5s × 3 SSE tests × 2 projects = 30s of CI time waiting for heartbeat timers. A `WithHeartbeatInterval(10 * time.Millisecond)` option in test setup would shave this to <1s.
8. **`healthInfo` type semantic name** — `corelive.HealthInfo` is fine but the convention in Go prefers `HealthStatus` or `HealthResponse` (it's the response payload, not "information in general"). Minor.
9. **The `Subscriber` type could be an interface** — Currently exposed as concrete `*Subscriber` from `Subscribe()`. An interface (`type Subscriber interface { ID() uint64; Events() <-chan json.RawMessage; Done() <-chan struct{} }`) would allow alternative implementations for testing.
10. **`.golangci.yml` repos missing from auditlog-core** — Adding even a minimal config would catch the `errcheck` warnings on `defer resp.Body.Close()` patterns that aren't in the current code (auditlog-core has none, but if helpers expand it'll come up).

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Migrate go-workflow dashboard to standard `encoding/json` (drop json/v2) | High — unblocks dashboard code path in CI and production-like envs | M |
| 2 | Add `GOEXPERIMENT=jsonv2` wrapper or document in README | High — saves devs from confusion | XS |
| 3 | Publish `auditlog-core` to GitHub with tag `v0.1.0` | High — unblocks downstream remove-replace work | M |
| 4 | Remove `replace` directives from both consumer `go.mod` after tag exists | High — required for real version resolution | XS |
| 5 | Set up `go.work` for all three projects | High — eliminates fragile local setup | M |
| 6 | Add `auditlog-core/.golangci.yml` matching sibling standards | Medium — proactive hygiene | XS |
| 7 | Add `auditlog-core/README.md` with Hub+Server usage example | High — required for adoption | M |
| 8 | Add `auditlog-core/LICENSE` (MIT) | High — required for open-source publish | XS |
| 9 | Add `auditlog-core/CONTRIBUTING.md` | Low — nice-to-have | S |
| 10 | Add `auditlog-core/CODEOWNERS` | Low — scales governance | XS |
| 11 | Extract NDJSON read/write into `auditlog-core/ndjson/` | High — 2nd 80/20 chunk of duplication | M |
| 12 | Extract format detection/loader into `auditlog-core/loader/` | High — completes plan phase 2 | M |
| 13 | Add `WithHeartbeatInterval(10ms)` to SSE tests via test setup helper | Medium — CI speed | S |
| 14 | Add benchmark for `Hub.OnEvent` with 1/10/100/1000 subscribers | Medium — regression detection | XS |
| 15 | Add benchmark for `Server` SSE handler throughput | Medium — regression detection | XS |
| 16 | Add `context.Context` to `WriteToFile` for cancellation | Medium — modern Go API hygiene | XS |
| 17 | Add integration test: create core server, connect SSE, send events, verify snapshot | High — real-world behavior validation | M |
| 18 | Add test for `WriteToFile` concurrent access | Medium — robustness | XS |
| 19 | Add test for `WriteToFile` directory-creation failure path | Low — robustness | XS |
| 20 | Add test for `handleReport` returning nil provider error | Medium — error path coverage | XS |
| 21 | Add test for SSE handler when `Flusher` assertion fails | Low — edge case | XS |
| 22 | Update go-workflow `AGENTS.md` to reference `auditlog-core` | Medium — knowledge persistence | XS |
| 23 | Update samber-do `AGENTS.md` to reference `auditlog-core` | Medium — knowledge persistence | XS |
| 24 | Update go-workflow `FEATURES.md` to reflect extraction | Low — doc drift prevention | XS |
| 25 | Update samber-do `FEATURES.md` to reflect extraction | Low — doc drift prevention | XS |
| 26 | Write ADR explaining the auditlog-core extraction decision | Medium — institutional memory | S |
| 27 | Add `docs/DOMAIN_LANGUAGE.md` to auditlog-core | Low — scope definition | S |
| 28 | Update both `flake.nix` devShells to include `../auditlog-core` | High — local dev parity | S |
| 29 | Add `.github/workflows/ci.yml` to auditlog-core | High — CI on its own repo | M |
| 30 | Add cross-repo GitHub Actions workflow that tests all three together | High — catches breaking changes in core | M |
| 31 | Tag `go-workflow-auditlog` and `samber-do-auditlog` with versions that use core `v0.1.0` | High — release coordination | XS |
| 32 | Add `go test -race` to CI for all three repos | Medium — concurrency correctness | XS |
| 33 | Add `go test -count=1` and `-v` to CI test commands | High — surfaces stale-cache issues | XS |
| 34 | Consolidate `makeReportProvider`/`Snapshot`/`Complete`/`Health` into shared helper | Low — saves ~80 LOC per consumer | S |
| 35 | Add `HealthInfo` to a generic response envelope with version field | Low — API stability | XS |
| 36 | Add `ErrInvalidPrefix` to auditlog-core for malformed route prefixes | Low — error UX | XS |
| 37 | Document the `replace` → published-version migration in both consumers' READMEs | Medium — migration story | XS |
| 38 | Add `go mod tidy` to BuildFlow pre-commit (currently skipped) | Medium — go.mod hygiene | XS |
| 39 | Refactor `With*Provider` to use `Option func(*Server) error` instead of fields | Low — ergonomics | S |
| 40 | Add `Server.Handle(pattern string, handler http.Handler)` for extensibility | Medium — embedding use case | S |
| 41 | Replace the SSE 15s default heartbeat with per-`Accept` negotiated value | Low — over-engineering unless needed | M |
| 42 | Add `Subscriber` interface in auditlog-core instead of concrete pointer | Low — testability | XS |
| 43 | Document `SnapshotProvider`/`CompleteProvider` lifecycle in auditlog-core README | Medium — adoption clarity | XS |
| 44 | Add `OnSubscribe`/`OnUnsubscribe` callbacks to core Hub for metrics | Low — optional future use | XS |
| 45 | Add Prometheus metrics interface for events-sent, clients-connected, etc. | Low — optional future use | M |
| 46 | Add `auditlog-core/examples/minimal` runnable demo | Medium — adoption clarity | S |
| 47 | Add `auditlog-core/cmd/auditlog-core-demo` CLI | Low — nice-to-have | M |
| 48 | Run full `Taskfile` / `BuildFlow` on all three repos, document remaining warnings | Medium — debt visibility | XS |
| 49 | Verify both `replace` directives are identical (path-wise) — catch drift | Low — pre-publish sanity | XS |
| 50 | Write a `MIGRATION.md` for downstream projects upgrading from in-tree Hub/Server to core | Medium — clear upgrade path | M |

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Should `go-workflow-auditlog/live/dashboard.go` be migrated to standard `encoding/json`** (matching `samber-do-auditlog`) to eliminate the `json/v2` build-constraint dependency, OR is `json/v2` intentional for a reason I don't know (escaping semantics, performance, etc.) that must be preserved? This blocks publishing `auditlog-core` without a known-broken downstream consumer.

2. **Should `auditlog-core` be published FIRST (current plan: remove `replace` directives after publish) or should we set up a `go.work` workspace FIRST** (allows all three to develop in lockstep without permanent replace directives in `go.mod`)? Each path has different implications: publish-first forces version coordination, workspace-first makes a `v0.1.0` tag meaningless.

3. **Should the BuildFlow "root-package-files" warnings (every `.go` file in project root should be in `internal/`) be addressed now in `go-workflow-auditlog` and `samber-do-auditlog`** (this would mean restructuring into `internal/auditlog/` package — a breaking change to imports in both projects and their downstream consumers), **OR should they be accepted as the project's chosen layout**? The current layout is intentional in many Go projects.
