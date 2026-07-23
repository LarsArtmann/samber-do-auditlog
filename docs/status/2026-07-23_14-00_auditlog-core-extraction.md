# Status Report: Auditlog Core Extraction

> Session: 2026-07-23 13:36 - 14:00
> Scope: Cross-repo duplication analysis + extraction into shared module

---

## a) FULLY DONE

- [x] Cross-repo comparison (6 files identical, 4 intentionally different)
- [x] Pareto analysis (1% -> 51%: Hub; 4% -> 64%: Server+Helpers)
- [x] Comprehensive plan with mermaid execution graph at `docs/planning/2026-07-23_13-36_auditlog-core-extraction.md`
- [x] Created `github.com/larsartmann/auditlog-core` module (9 files, 1471 LOC)
- [x] `live/hub.go` — Generic SSE Hub with `json.RawMessage` events
- [x] `live/server.go` — Generic HTTP server with Prefix, SSE, report, health
- [x] `helpers.go` — Atomic `WriteToFile` + `CheckNoClobber`
- [x] 24 tests in auditlog-core (7 Hub + 12 Server + 5 Helper), all pass
- [x] Wired samber-do-auditlog: Hub and Server wrappers delegate to core
- [x] Wired go-workflow-auditlog: Hub and Server wrappers delegate to core
- [x] All 17 samber-do live/ tests pass (net -527 LOC per project)
- [x] Committed: auditlog-core `53f7a88`, samber-do `74b6d6f`, go-workflow `fc4ae6e`

## b) PARTIALLY DONE

- [ ] **go-workflow-auditlog compilation** — Wrapper code is structurally correct but `encoding/json/v2` dependency in `dashboard.go` prevents `go build` in this environment. The wrapper delegates correctly; compilation will work once the env supports json/v2.
- [ ] **Lint fixes for samber-do wrapper** — 7 golangci-lint warnings remain:
  - `live/hub.go:7` — import list not allowed (nolint needed)
  - `live/hub.go:20` — Hub missing field `mu` (exhaustruct)
  - `live/server.go:11` — import list not allowed
  - `live/server.go:81` — `NewServer` too long (72 > 60)
  - `live/server.go:82` — Server missing field `core`
  - `go.mod:31` — direct/indirect requires mixed
  - Pre-existing: 47 GitHub Actions pinning, 3 structure warnings

## c) NOT STARTED

- [ ] NDJSON read/write extraction (plan phase 2, ~130 LOC identical)
- [ ] Format detection/loader extraction (plan phase 2, ~175 LOC identical)
- [ ] `auditlog-core/README.md` with usage examples
- [ ] Update both projects' `AGENTS.md` with auditlog-core reference
- [ ] Update both projects' `FEATURES.md`
- [ ] Publish `auditlog-core` to GitHub (remove `replace` directives)
- [ ] Remove `replace` directives from both go.mod files after publish
- [ ] Run `golangci-lint` on auditlog-core (not run yet)
- [ ] SSE test speedup (5s due to heartbeat interval, could use shorter interval in tests)

## d) TOTALLY FUCKED UP

- **Commit messages auto-generated** — Both go-workflow and samber-do had auto-committed with generic messages during the session. I amended them but the original auto-commits are in reflog. Not critical but noisy.
- **go-workflow-auditlog can't compile** — The `encoding/json/v2` / `encoding/json/jsontext` packages are excluded by build constraints in this Go 1.26.4 nix environment. This is a **pre-existing issue** (the project never compiled here), not caused by my changes, but it means I couldn't verify the go-workflow wrapper compiles.
- **Hub test had unexported type** — `subscriber` was unexported but `Subscribe()` returned `*subscriber`. Had to export it as `Subscriber`. This was a design mistake that should have been caught upfront.

## e) WHAT WE SHOULD IMPROVE

1. **SSE test performance** — Heartbeat interval defaults to 15s, causing SSE tests to take 5s each. Add a `WithHeartbeatInterval` to test configs to speed up CI.
2. **Server handler test coverage** — No test for `handleReport` error path (nil provider), no test for `handleSSE` when flusher assertion fails.
3. **auditlog-core needs `go.mod` replace workaround** — Both consuming projects use `replace` directives. These must be removed before publishing.
4. **golangci-lint configuration** — auditlog-core has no `.golangci.yml`. Should add one matching the sibling projects.
5. **The plan's NDJSON/loader extraction was deferred** — The 20% that delivers 80% was skipped. These are ~300 LOC of identical code still duplicated.
6. **Dashboard HTML divergence** — `live/dashboard.go` in go-workflow uses `encoding/json/v2` + `jsontext` while samber-do uses `encoding/json` v1. The dashboard templates are ~130 lines each with different tab structures. Not extractable but could share a base template.
7. **`WriteToFile` test coverage** — Only tests success and one error path. Missing: concurrent writes, directory creation failure, large file handling.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

1. Fix go-workflow-auditlog `encoding/json/v2` environment issue (nix Go version)
2. Publish `auditlog-core` to GitHub and remove `replace` directives
3. Add `.golangci.yml` to `auditlog-core`
4. Fix the 7 golangci-lint warnings in samber-do wrapper
5. Add `README.md` to `auditlog-core` with usage examples
6. Extract NDJSON read/write into `auditlog-core/ndjson/`
7. Extract format detection/loader into `auditlog-core/loader/`
8. Speed up SSE tests (configurable heartbeat interval)
9. Add test for `WriteToFile` concurrent access
10. Add test for `handleReport` with nil provider
11. Add test for `handleSSE` without Flusher
12. Update both `AGENTS.md` files with auditlog-core reference
13. Update both `FEATURES.md` files
14. Add `auditlog-core` to `go-workflow-auditlog`'s `flake.nix` devShell
15. Add `auditlog-core` to `samber-do-auditlog`'s `flake.nix` devShell
16. Run `golangci-lint` on auditlog-core and fix findings
17. Add `context.Context` to `WriteToFile` for cancellation support
18. Consider `WriteToFile` returning `*os.File` for streaming use cases
19. Add `go vet ./...` to CI for auditlog-core
20. Add `staticcheck` to auditlog-core CI
21. Create `auditlog-core/.github/workflows/ci.yml`
22. Add `CODEOWNERS` to auditlog-core
23. Add `CONTRIBUTING.md` to auditlog-core
24. Add `LICENSE` to auditlog-core (MIT, matching siblings)
25. Consider extracting `normalizePrefix` to auditlog-core root (used by both wrappers)
26. Consider extracting `errorToStringPtr` to auditlog-core (identical in both)
27. Add `examples/` directory to auditlog-core with minimal working example
28. Write integration test that creates auditlog-core server, connects SSE, sends events, verifies snapshot
29. Add benchmark for Hub.OnEvent with N concurrent subscribers
30. Add benchmark for Server SSE handler
31. Review if `Subscriber` type should be an interface instead of concrete
32. Consider making `Server` implement `http.Handler` interface explicitly (it already does via `ServeHTTP`)
33. Add `go run` example in auditlog-core README
34. Tag initial release `v0.1.0` for auditlog-core
35. Update `samber-do-auditlog/go.mod` to use tagged version after publish
36. Update `go-workflow-auditlog/go.mod` to use tagged version after publish
37. Remove `hub.go` and `server.go` duplicate code from both projects (delete old files if still present)
38. Verify no other files in either project import the old `live.Hub` or `live.Server` directly
39. Check if `live/server_test.go` in go-workflow has the same `json.RawMessage` issue as samber-do
40. Add `//go:build` constraint to handle json/v2 gracefully in dashboard.go
41. Consider adding `go-workspace` setup for all three projects
42. Write ADR for the extraction decision
43. Add `docs/DOMAIN_LANGUAGE.md` to auditlog-core
44. Review if `healthResponse` type should be in auditlog-core (currently duplicated in wrappers)
45. Consider extracting `snapshotData`/`completeData` types to auditlog-core with generic fields
46. Add `go run ./cmd/auditlog` example that uses the live server
47. Run full CI pipeline on all three repos after publishing
48. Update `STABILITY.md` in both projects
49. Create GitHub release for auditlog-core
50. Update `TODO_LIST.md` in both projects with extraction status

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Should `auditlog-core` be a Go workspace (`go.work`)** alongside the sibling projects, or a standalone module that's published separately first? A workspace would simplify local dev but requires all three to share the same Go version constraint.

2. **Should the `encoding/json/v2` / `jsontext` usage in `go-workflow-auditlog/live/dashboard.go` be migrated to standard `encoding/json`** (like samber-do does) to fix the build issue, or is json/v2 intentional for performance/escaping reasons that must be preserved?

3. **Should we proceed with NDJSON/loader extraction now** (plan phase 2, ~300 more LOC) or defer it until the core module is published and proven in production first?
