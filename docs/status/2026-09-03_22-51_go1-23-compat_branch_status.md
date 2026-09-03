# Status Report — `go1.23-compat` branch: Go 1.23 downgrade for the samber/do merge

| | |
|---|---|
| **Date** | 2026-09-03 22:51 CEST |
| **Branch** | `go1.23-compat` (created from `master` @ `59bc651`; renamed from `go1.18-compat` mid-session) |
| **Session scope** | Downgrade the entire project to Go 1.18 (original ask) → retargeted to Go 1.23 after samber approved it |
| **Driver** | samber is considering merging the plugin into `github.com/samber/do` (whose floor is `go 1.18`); samber starred the repo 2026-09-01 |
| **Toolchains verified** | go1.26.7 (dev) **and** real go1.23.12 (auto-downloaded via `GOTOOLCHAIN`) |
| **Final state** | build ✅ · vet ✅ · tests ✅ · `-race` ✅ · coverage **95.2%** (gate 94) ✅ · golangci-lint v2.1.6 **0 issues** ✅ · `go generate` no drift ✅ · version drift guard ✅ · example smoke ✅ |

---

## Executive Summary

The branch is **functionally complete and fully verified**: the module now declares `go 1.23` with exactly **one runtime dependency (`samber/do/v2`)**, every exporter that previously pulled go-1.26-only Lars libraries is re-implemented on the stdlib, GOEXPERIMENT=jsonv2 is gone everywhere, and the whole CI/lint/flake/docs stack was re-pinned to the 1.23 reality. All green gates pass, most of them validated against a real go1.23.12 toolchain, not just language-version gating.

The honest caveats: (1) roughly the first third of the session's downgrade work was done against Go 1.18 and had to be reverted when the target moved to 1.23 — cleanly, via `git restore --source=master`, but it was churn; (2) the HTML report is a **rewrite, not a port** — functionally equivalent and a11y-complete, but visually/interactively reduced vs master (no interactive DAG, no waveform, no pagination); (3) the first hour had an unusually high rate of self-inflicted defects from sloppy generated-file payloads (8 distinct incidents, all caught, none shipped — see section d); (4) CI itself has not run — the new pipeline is locally validated but unproven on a runner, and the flake is unevaluated.

---

## a) FULLY DONE

### Research & decisions
1. **Dependency floor audit** — `go list -m` over the full graph: every `larsartmann/*` dep (go-output family, go-ndjson, go-sse, go-atomic-write, go-error-family, go-branded-id) requires go 1.26.x; `a-h/templ` 1.25; `invopop/jsonschema` v0.14 needs 1.24 but **v0.13.0 needs only 1.18**; `samber/do v2.1.0` itself declares `go 1.18` (confirmed via GitHub API).
2. **Target validation** — 1.23 is the *exact* floor of master's own code (`slices.Backward` = 1.23), so no core file needed any semantic change.
3. **invopop archaeology** — v0.13.0 regenerates `schema/report.schema.json` **byte-identically** (verified via `go generate` + git diff), so the schema generator and the `stale-generation` CI job survive.

### Code — dependency elimination (all stdlib ports)
4. **go.mod**: `go 1.23`; requires = `samber/do/v2 v2.1.0` + `invopop/jsonschema v0.13.0` (tooling); `tool` directive (templ) removed; `retract` preserved.
5. **Core restored verbatim from master** — `recorder.go`, `hooks.go`, `report_builder.go`, `diff.go`, `replay.go`, `example/services.go`, `example/register.go` via `git restore --source=master` (atomics, slices, maps, cmp, multi-`%w` all legal again).
6. **`diagram.go`** — Mermaid / DOT / PlantUML / D2 renderers + all five escapers (SlugifyID, MermaidID, MermaidText, D2/DOT, PlantUML) ported line-for-line from go-output v0.37.0 for our boxed-node/unlabeled-edge shapes; dedup semantics preserved (`\x00` key, first-wins). All pre-existing diagram tests pass **unchanged** — the port is output-identical.
7. **`table.go`** — local `TableFormat` string type (string literals keep compiling), 5 formats: `table` (ASCII grid), `json`, `csv`, `tsv` (encoding/csv), `markdown`; unknown formats → `errUnsupportedTableFormat`. `RenderOptions`/`ColorMode` API shape preserved.
8. **`ndjson.go` / `loader.go`** — full stdlib ports of go-ndjson: same sentinels (`ErrEmpty`, `ErrNoEvents`, `ErrOversizedLine`), same error strings, 1 MB line cap, same `Detect` heuristics, `StreamEvents`/`ReadEvents`/`LoadReport*` signatures unchanged.
9. **`plugin.go` `writeToFile`** — stdlib atomic write (CreateTemp in target dir → write → Sync → Close → Rename, deferred cleanup on failure); named return + `//nolint:nonamedreturns`.
10. **go-error-family removed** — `classify.go`/`classify_test.go` deleted; `FuzzClassifyAdversarialChains` reworked to assert `errors.Is` survives adversarial `fmt.Errorf` chains.
11. **`schema.go`** — `go:generate go run ./cmd/genschema` restored (master-identical), `cmd/genschema` back on invopop v0.13.
12. **`html.go` + `html_view.go`** — templ replaced by `html/template`: five tabs, warm-amber tokens (`DesignTokensCSS`/`SharedComponentCSS` embedded verbatim), CSP `default-src 'none'`, zero external resources, **master's full a11y set**: roving-tabindex tabs with Arrow/Home/End, `?` help dialog with focus trap + restore, `/` search, `e` errors-toggle, sortable columns with `aria-sort` + Enter/Space, `data-error` tooltips, skip link. Mermaid graph embedded as escaped text (pasteable).
13. **`live/` deleted** (18 files) + `example --live`/`--live-addr` removed; example keeps the 22 other feature demos and passes its self-check.
14. **Test downgrades where 1.23 demands it**: `b.Loop` → `for range b.N` (9 sites), `wg.Go` → `Add`/`go`/`Done` (2 sites); `math/rand/v2`, int-range loops, `atomic.Int64` in tests all stay (legal ≥1.22/1.19).
15. **Fuzz XSS check recalibrated**: `assertNoRawXSS` now neutralizes escaped quote entities (`&#34;`/`&#39;`/`&quot;`) before matching, and vectors are anchored on **raw** quotes (` onload="`, `href="javascript:` …) — escaped attribute data can no longer false-positive, genuine breakouts still trip all five seeds.
16. **New coverage tests** (`compat_export_extra_test.go`): ASCII table end-to-end, unsupported table format, Up/Left direction matrix across all four diagram formats, health-stat HTML (healthy + unhealthy cells, stat card), loader unknown-format rejection.

### Infra
17. **`ci.yml`**: all 7 jobs `go-version: "1.23"`; workflow-level `GOEXPERIMENT` removed; golangci-lint pinned **v2.1.6** (newest v2 whose own go directive ≤ 1.23 — verified: v2.4+ need 1.24, v2.12.2 needs 1.25); govulncheck pinned **v1.1.4** (go 1.22); goreleaser job **dropped** (its `go install` can't build on a 1.23 runner; releases remain a master activity); stale-generation now guards only `schema/report.schema.json`.
18. **`.golangci.yml`**: `run.go: "1.23"` (quoted — unquoted 1.23 parses as a YAML number and fails v2.1.6 schema validation); 9 post-2.1.6 linters removed (arangolint, clickhouselint, embeddedstructfieldcheck, godoclint, gomodguard_v2, iotamixing, modernize, unqueryvet, wsl_v5); templ/live paths and the godoclint text-rule removed; `config verify` passes on v2.1.6.
19. **`flake.nix`**: `go_1_23` no longer exists in nixpkgs (EOL-removed — verified via `nix eval`), so devShell uses bootstrap `pkgs.go` + `GOTOOLCHAIN = "go1.23.12"` (bare `go1.23` is rejected by the go command: "a language version but not a toolchain version"); GOEXPERIMENT removed; coverage/auditlog apps re-pinned.
20. **`scripts/check-go-version.sh`**: accepts flake pins with patch extension (`1.23.x`) and quoted `.golangci.yml` values; exits green on the branch.
21. **`scripts/coverage-gate.sh` + `coverage-exclusions.txt`**: GOEXPERIMENT export removed; dead exclusions (`/live/demo/`, `_templ\.go`) removed.
22. **`.buildflow.yml`**: `env: GOEXPERIMENT` removed; go-auto-upgrade skip kept with an updated rationale.

### Docs
23. **AGENTS.md**: new top section "THIS BRANCH: `go1.23-compat`" — the full divergence contract (deps, ports, API deltas, CI pins, verification ledger).
24. **README.md**: badge → Go 1.23+, GOEXPERIMENT callout deleted, deps row → "samber/do/v2 is the only runtime dependency", Live Dashboard section rewritten as a master-only pointer.
25. **CONTRIBUTING.md**: prerequisites → Go 1.23+/golangci v2.1.6 rationale, GOEXPERIMENT section deleted.
26. **BENCHMARKS.md**: baseline explicitly marked as master's (Go 1.26.7), bench command scrubbed of GOEXPERIMENT.

### Verification ledger (all executed this session)
27. `gofmt -l` clean · `go build ./...` ✅ · `go vet ./...` ✅ (both toolchains)
28. `go test -count=1 ./...` ✅ and `go test -race -count=1 ./...` ✅ (both toolchains)
29. `go tool cover` gate: **95.2% ≥ 94%** after adding real tests (never lowered the threshold)
30. golangci-lint **v2.1.6** (the CI pin, prebuilt binary): `config verify` ✅, `run` → **0 issues** (incl. `--fix` pass)
31. `go generate ./...` → zero diff; `go mod tidy` → zero drift; `sh scripts/check-go-version.sh` → OK
32. **Real go1.23.12**: `go version` / build / vet / full test suite ✅
33. Example smoke: `DO_AUDITLOG_ENABLED=true go run ./example` — full lifecycle demo runs clean
34. Diagram output equivalence: all pre-existing diagram tests (direction matrix, escaping, dedup, fuzz seeds) pass against the ported renderers unchanged

---

## b) PARTIALLY DONE

1. **Docs parity on the branch** — `FEATURES.md`, `TODO_LIST.md`, `ROADMAP.md` still describe master (live/ features, `html.templ` file references, error-family classification, 16 table formats). Only README/CONTRIBUTING/BENCHMARKS/AGENTS were brought current.
2. **README soft references** — the "SREs wiring DI metrics … SSE live dashboard" bullet and the docs-links table row pointing at the live-dashboard guide remain; only the main Live Dashboard section was rewritten.
3. **AGENTS.md dual-identity** — the master-era sections (version-skew ledger entry for deleted `live/fragments.go:181`, example feature table row "Live dashboard `--live`") still describe master inside the branch's file. The new branch section supersedes but doesn't fully re-tailor the file.
4. **BENCHMARKS on 1.23** — table flags the baseline as master's; no actual go1.23.12 re-measurement was run.
5. **HTML UX parity** — intentionally reduced: no interactive Sugiyama DAG (master: pan/zoom/click-highlight via daghtml SDK), no event waveform, no "show all" pagination, no debounced search, no JSON data island, system fonts instead of Google Fonts. All documented, none implemented.
6. **CI validation** — locally verified everything that can run locally (config verify, lint, drift guard, script), but **the pipeline itself has never executed on a runner**, and `actionlint` was not run on the modified `ci.yml` (binary unavailable; `go install` blocked by shell policy).
7. **flake.nix** — edited and attribute-checked, but `nix flake check` / `nix develop` / `nix run .#coverage` were **never executed**; the bootstrap-`go` + `GOTOOLCHAIN` pattern is sound in theory, unproven in practice.
8. **direnv/GOEXPERIMENT footgun** — my own shell carried stale `GOEXPERIMENT=jsonv2` from the old devShell and every go1.23.12 invocation needed an explicit `GOEXPERIMENT=` prefix. Contributors switching branches without `direnv reload` will hit `unknown GOEXPERIMENT jsonv2`. Known, worked around, not documented anywhere.
9. **Fuzz depth** — fuzz targets ran as seed-corpus tests only; no timed fuzzing session (`-fuzztime 30s`) was done on the new HTML/diagram code.
10. **CHANGELOG** — no entry describing the branch/divergence.
11. **Website** (`do-auditlog.lars.software`) — documents master (GOEXPERIMENT install note, live-dashboard guide); its relationship to this branch is unaddressed (probably fine — deploys from master — but undecided).

---

## c) NOT STARTED

1. Pushing `go1.23-compat` to origin; opening any PR.
2. The actual samber/do merge work: module-path decision, subpackage layout inside samber/do, upstream CI wiring.
3. `actionlint` on the modified workflow.
4. `nix flake check` / devShell smoke on the edited flake.
5. Benchmarks re-baseline on go1.23.12.
6. Timed fuzzing session on the branch.
7. CHANGELOG entry.
8. FEATURES/TODO_LIST/ROADMAP branch adaptation (HARVEST from section f).
9. `GOOS`/`GOARCH` matrix checks (windows, darwin, 386 — 32-bit alignment of the restored `atomic.Int64` fields).
10. Pre-commit hook smoke after all changes (`git config core.hooksPath scripts/hooks` state on this checkout unverified).
11. `.goreleaser.yml` keep-or-delete decision (file still present, CI job gone).
12. Decision + work for backporting any HTML/dashboard improvements master-ward or forward-porting master's interactive DAG.
13. Evaluate whether the floor could go lower (1.21/1.22) via a `slices.Backward` shim — explicitly not attempted; 1.23 was the mandate.
14. `live/` stdlib-SSE reimplementation scoping (net/http Flusher + ES5) — not started, only listed as an option.

---

## d) TOTALLY FUCKED UP (self-inflicted incidents — all caught, none shipped)

1. **Eight silent write rejections masquerading as success.** The auto-commit daemon touched files between my read and write; the write tool rejected 8 rewrites (`d2.go`, `table.go`, `dot.go`, `mermaid.go`, `plantuml.go`, `tree.go`, `diagram_options.go`, `html.go`) with a "modified since last read" warning. I misread those responses as success confirmations, and the next build failed with imports of packages I had deleted. Cost: one full discovery cycle + re-reading and re-writing all 8 files. **Root cause: batching writes without per-write verification in a daemon-active repo.**
2. **Garbage in generated payloads (6 incidents).** Shipped-in-draft defects I wrote myself and then patched: a placeholder `'>' + "" + "Services"` loop with `_ = want` in `html_golden_test.go`; `errorsNew()` (nonexistent) plus a broken `%-*s`-verb table and an `optsTitle` placeholder in the first `table.go`; a dead `delivered == 0` block referencing an unimported `strings` in `ndjson.go`'s StreamEvents; `_ = children` dead code from a malformed prealloc "fix" in `report_builder.go`; a hallucinated `_healthyMarker` field in `healthReport()`; `buildHTMLEventsForService` — a dead stub I created and then deleted. Every one was caught before merge (mostly by me, two by the compiler), but the density was too high for one session.
3. **The 1.18 → 1.23 mid-flight pivot.** The full 1.18 downgrade (manual `sort.Slice` conversions, atomic-type removal, `slices.Backward` rewrites) was completed *before* samber's 1.23 approval arrived, then reverted. The revert itself was clean and correct (`git restore --source=master` on seven files), but it was avoidable churn: the target version was the single most consequential unknown and went unconfirmed for an hour.
4. **invocationSeq semantics bug (introduced and fixed in the same breath).** My 1.18 rewrite read-then-incremented, making the first invocation order `-1`. Caught on self-review; mooted minutes later by the 1.23 revert. Still: I briefly corrupted a documented monotonic contract.
5. **nolint placement thrash.** gosec reports at the sensitive *call*, exhaustruct at the *literal*, nolintlint then flagged my wrongly-scoped directives as unused, and one justification comment pushed the line past golines' 120 columns — three full lint cycles (~1 min each) for four directives. Should have been one.
6. **sed-on-YAML collateral.** Bulk seds on `.golangci.yml` left a dangling empty `paths:` key (schema-validation failure) and two shell-sanitizer rejections from quote-heavy regexes before I switched to the structured edit tool. YAML + sed = predictable damage.
7. **Coverage optimism.** I proceeded as if the 94% gate would hold after deleting live/ and adding the HTML rewrite; the first gate run said 92.7%. Fixed properly with real tests (95.2%), but the gap should have been forecast the moment file deletions landed.
8. **Verification claims written before verification.** I drafted the AGENTS.md "verification status" line (including "real go1.23.12 ✓") *before* running the battery. Every claim did end up true, but the order was wrong — that's how false status lines get shipped.

---

## e) WHAT WE SHOULD IMPROVE

1. **Verify-after-write discipline in daemon repos.** One `head -3` after each write would have caught all 8 rejections instantly. Batch-write only files the daemon provably never touches.
2. **Compose, don't draft-and-patch.** The 6 payload-garbage incidents share a root cause: writing large files with placeholder scaffolding intended "to fix next". Write final text or mark the file explicitly WIP.
3. **Confirm the load-bearing unknown first.** "Downgrade to Go 1.18" vs "1.23 is fine" was worth one clarifying sentence before hours of work — or an explicit assumption note and a version-parametrized plan.
4. **Structured edits for structured files.** No more sed on YAML/Nix; the edit tool with exact context is cheaper than repairing sed fallout.
5. **Real toolchain early.** Downloading go1.23.12 should have happened at retarget time, not as the final step — it's the ground truth for every "does 1.23 accept this" question.
6. **Coverage-impact forecasting.** Any deletion of covered code + addition of new untested code should trigger an immediate projected-gate check.
7. **Claims-after-evidence.** Status lines in AGENTS.md (and reports) get written after the commands run, with the run outputs in between.
8. **Lint-aware codegen.** Before writing new files in a strict-lint repo, re-check the enabled linter list (noinlineerr, exhaustruct, golines 120, wrapcheck, varnamelen) and write to it — instead of 3 fix-up cycles.
9. **HTML parity ledger.** If a rewrite intentionally drops features, enumerate them (as now done in AGENTS/README) at rewrite time, not at review time.
10. **Keep the port-with-tests pattern.** The go-output byte-compatibility port (existing tests as the spec) was the single best decision of the session — generalize it for any future dependency elimination.

---

## f) 50 things to get done next

| # | Action | Impact |
|---|--------|--------|
| 1 | Push `go1.23-compat` to origin | Unblocks everything downstream |
| 2 | Run CI on the branch — first real validation of ci.yml/golangci v2.1.6/govulncheck v1.1.4 | High |
| 3 | `actionlint` on modified ci.yml (prebuilt binary) | High |
| 4 | `nix flake check` + `nix develop` smoke + `nix run .#coverage` (flake is unevaluated) | High |
| 5 | Decide merge vehicle with samber: in-repo subpackage vs standalone module (module path!) | Critical |
| 6 | Write the samber/do merge proposal: what merges, what stays, maintenance split | Critical |
| 7 | Public API diff doc (master vs branch) for samber review | High |
| 8 | Confirm every `samber/do` API used is stable-public (ExplainInjector deadlock note included) | High |
| 9 | Decide release strategy: branch line vs pure staging for the merge | High |
| 10 | `compat_export_extra_test.go` — confirm daemon committed it / commit | Medium |
| 11 | Update `FEATURES.md` for the branch (live/ master-only, html/template, 5 table formats) | Medium |
| 12 | HARVEST this report's section f into `TODO_LIST.md`/`ROADMAP.md` (docs-health) | Medium |
| 13 | CHANGELOG entry for the divergence | Medium |
| 14 | Annotate `TODO_LIST.md` live/ items as master-only on this branch | Low |
| 15 | README: scrub the two remaining soft live/ references (SRE bullet, docs table row) | Low |
| 16 | AGENTS.md: move the master-only nolint-ledger mention out of branch context | Low |
| 17 | Document the direnv/GOEXPERIMENT stale-env footgun (`direnv reload` after branch switch) | Medium |
| 18 | Check `.envrc.example` matches the new flake | Low |
| 19 | Run `nixfmt`/treefmt on `flake.nix` | Low |
| 20 | Benchmarks re-baseline on go1.23.12 | Medium |
| 21 | Timed fuzz session (30s × 5 targets) on branch code | Medium |
| 22 | GOARCH matrix smoke: windows/darwin build, 386 (32-bit atomic alignment) | Medium |
| 23 | Pre-commit hook smoke on this checkout (`core.hooksPath` re-install per AGENTS) | Medium |
| 24 | `.goreleaser.yml` keep-or-delete decision | Low |
| 25 | HTML: re-add pagination for large reports (50 services / 100 events like master) | Medium |
| 26 | HTML: debounced search (master uses ~120 ms) | Low |
| 27 | HTML: decide on a JSON data island (downstream tooling + gives `stripJSONScripts` a purpose again) | Medium |
| 28 | `stripJSONScripts`: delete consciously or keep with a comment (currently dead-ish) | Low |
| 29 | `toRankDir(DirectionDown)` is unreachable via public API — collapse or document | Low |
| 30 | Golden-fixture tests (full output) for the 4 diagram renderers to lock the port | High |
| 31 | Test the `mermaidID` "node" fallback collision (two services collapsing to the same id) | Low |
| 32 | Port extra dedupGraphEdges edge-case tests from go-output graphtest | Low |
| 33 | Evaluate cheap extra table formats (xml/asciidoc via stdlib) — scope decision | Low |
| 34 | Windows test for `writeToFile` temp+rename semantics | Medium |
| 35 | Verify `robustness_test` MaxEvents race coverage still exercises the restored atomic path | Medium |
| 36 | Flake-test loop: `test -count=10` stability run | Low |
| 37 | `govulncheck` local run on the branch (pinned v1.1.4) | Medium |
| 38 | `go mod verify` | Low |
| 39 | Consider CI belt: explicit `GOTOOLCHAIN=go1.23.12` in one job (setup-go resolves latest 1.23.x today) | Low |
| 40 | Monitor golangci v2.2+ for a go-directive ≤ 1.23 to restore newer linters | Low |
| 41 | Website: decide whether the compat line gets a page/note (or stays master-only) | Low |
| 42 | Post-merge: wire the new subpackage into samber/do's CI (if merged) | Deferred |
| 43 | Post-merge: decide whether master downshifts to 1.23 (making this the mainline) | Critical |
| 44 | Evaluate 1.22/1.21 floor via `slices.Backward` shim (only if samber wants it) | Deferred |
| 45 | Scope a stdlib-SSE lite dashboard for the 1.23 line (scoping doc only) | Deferred |
| 46 | Backport policy: which of the branch's HTML test ideas improve master | Low |
| 47 | Rename-check: grep for lingering `go1.18` references anywhere (docs/scripts) | Quick |
| 48 | `testhelpers` package: confirm JS-balance helper handles the new single-script layout long-term | Low |
| 49 | Add the branch to STABILITY.md (or note BETA status per line) | Low |
| 50 | Celebrate, then delete `/tmp/gcl` golangci binary note from local docs (CI installs its own) | Trivial |

---

## g) Questions I cannot answer myself

1. **Merge vehicle & module identity:** does samber want this *inside* `github.com/samber/do` as a subpackage (which import path — `do/audit`? separate nested module?), or as a standalone module that do merely links/recommends? This decides whether `module github.com/larsartmann/samber-do-auditlog` survives, whether we re-version, and what "API compatible" even means for the merge.
2. **Go floor ambition:** is **1.23 the agreed hard floor**, or should I invest in a 1.21/1.22-compatible shim (the only blocker is `slices.Backward` — a three-line reverse loop)? Wider floor = wider adopter compatibility for the merged artifact; I will not guess samber's supported-version policy.
3. **Live dashboard fate:** is "live/ stays master-only" acceptable for the merged artifact, or does the go-1.23 line need a live story (a stdlib-SSE + vanilla-JS reimplementation is feasible but is a project of its own)? Related: should any of this branch's HTML work flow *back* into master, or do the two renderers intentionally diverge forever?

---

*Point-in-time snapshot — 2026-09-03 22:51 CEST, branch `go1.23-compat`. Section (f) is HARVEST fuel for `TODO_LIST.md`/`ROADMAP.md` (docs-health), not a commitment list.*
