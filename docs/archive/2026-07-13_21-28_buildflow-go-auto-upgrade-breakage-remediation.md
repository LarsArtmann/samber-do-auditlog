# Status Report — 2026-07-13 21:28

## Session: BuildFlow `go-auto-upgrade` Breakage Remediation

---

## Executive Summary

A BuildFlow run (commit `ceca5b4`) triggered `go-auto-upgrade`, which catastrophically broke the entire project by migrating `encoding/json` to `encoding/json/v2` + `encoding/json/jsontext` — two experimental stdlib packages that are **build-constraint-excluded in Go 1.26.4**. It also deleted `CompareServiceRefs()` from `diff.go` (breaking 4 call sites), upgraded `go-output` from v0.30.1 to v0.30.4 (which transitively imports the same unavailable packages), and left the project in a completely uncompilable state. All 4 packages failed at setup. This session reverted the 9 broken files to HEAD, verified full compilation, vet, test suite, and code generation. The project is back to green.

---

## a) FULLY DONE

| # | Item                       | Details                                                                                                                                                  |
| - | -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Identified root cause      | `go-auto-upgrade` migrated to `encoding/json/v2` + `jsontext` (build-constraint-excluded in Go 1.26.4) AND deleted `CompareServiceRefs()` from `diff.go` |
| 2 | Reverted 9 files           | `export.go`, `report.go`, `ndjson.go`, `loader.go`, `migration.go`, `diff.go`, `cmd/genschema/main.go`, `go.mod`, `go.sum` — all restored to HEAD        |
| 3 | Preserved harmless changes | `.gitignore` (JS/TS patterns from gitignore-upserter) and `flake.lock` (nixpkgs bump) kept — both benign                                                 |
| 4 | `go mod tidy`              | Clean — go.sum consistent with go.mod, no drift                                                                                                          |
| 5 | `go vet ./...`             | Clean                                                                                                                                                    |
| 6 | `go build ./...`           | Clean — all 4 packages compile                                                                                                                           |
| 7 | `go test -race ./...`      | All pass: `auditlog` (1.3s), `cmd/auditlog` (1.5s), `cmd/genschema` + `example` (no test files)                                                          |
| 8 | `go generate ./...`        | Clean — schema regenerated (5777 bytes)                                                                                                                  |

**Project state: GREEN.** Only `.gitignore` and `flake.lock` differ from HEAD.

---

## b) PARTIALLY DONE

Nothing partially done. The revert was binary — either the code compiles or it doesn't.

---

## c) NOT STARTED

| # | Item                                                | Notes                                                                                                                                                                                                                                         |
| - | --------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~ | ~~Go-output v0.30.4 upgrade~~ done — go-output v0.38.0 lockstep | ~~The upgrade itself may be desirable (v0.30.4 could have real fixes), but it's blocked until Go 1.26.4's build constraints are resolved or go-output drops `encoding/json/v2`. Not attempted this session — out of scope for emergency repair.~~ |
| ~~2~~ | ~~Committing the revert~~ done — committed — see Resolution | ~~Not committed — per project rules, no commit without explicit user instruction.~~ |
| ~~3~~ | ~~Investigating whether Go 1.27+ would enable json/v2~~ done — GOEXPERIMENT=jsonv2 adopted | ~~Not researched.~~ |

---

## d) TOTALLY FUCKED UP (by `go-auto-upgrade`, not by this session)

### The Carnage

The `go-auto-upgrade` migrator in BuildFlow committed 7 categories of damage simultaneously:

1. ~~**Import rewrite to non-existent packages**: Rewrote `encoding/json` → `encoding/json/v2` + `encoding/json/jsontext` in 5 files (`export.go`, `report.go`, `ndjson.go`, `loader.go`, `migration.go`). Both packages are **build-constraint-excluded** in Go 1.26.4 — they physically exist in the nix store but `//go:build goexperiment.jsonv2` (or equivalent) is not satisfied.~~ done (reverted — see Resolution)

2. ~~**API migration with wrong API**: `export.go` changed `json.NewEncoder(w)` → `jsontext.NewEncoder(w)` then `enc.Encode()` → `json.MarshalEncode()`. `report.go` used `enc.SetIndent("", "  ")` which doesn't exist on `*jsontext.Encoder` — it should have been `jsontext.WithIndent("  ")` passed to the encoder constructor. The migration was half-baked.~~ done (reverted — see Resolution)

3. ~~**Deleted a public function**: `CompareServiceRefs()` was deleted from `diff.go` but **4 call sites still referenced it** (`diff.go:76`, `diff.go:77`, `report_builder.go:38`, `report_builder.go:141`). The tool tried to inline it into `sortServiceDiffs` but botched the refactor — `sortServiceDiffs` now takes `ServiceDiff` params and reads `.ServiceName`/`.ScopeID` directly, but the function signature lost the `ServiceRef` comparison logic.~~ done (restored at diff.go:169)

4. ~~**Dependency upgrade**: Bumped `go-output` from v0.30.1 → v0.30.4 across 11 module lines. This made the breakage **transitive** — even reverting local code wouldn't help because go-output v0.30.4 itself imports `encoding/json/v2`.~~ done (now v0.38.0 lockstep)

5. ~~**Import ordering violation**: `cmd/genschema/main.go` had `encoding/json/jsontext` placed in the third-party import group (after `github.com/...`), violating gci grouping rules.~~ done (resolved)

6. ~~**Removed `go-output/testhelpers` from go.mod**: An indirect dependency was dropped, potentially affecting test infrastructure.~~ done (resolved — v0.38.0 clean manifests)

7. ~~**Cascading build failures**: All 6 BuildFlow steps downstream of `go-auto-upgrade` failed: `go-fix`, `go-generate`, `govalid-generate`, `golangci-lint:repair`, `test-race`, and `test-coverage`. 11 more steps were skipped as blocked.~~ done (repaired — CI green since)

### Why It Happened

`go-auto-upgrade` saw Go 1.26 in the toolchain and assumed `encoding/json/v2` was available. It's not — the package exists in the Go source tree behind build constraints that require a Go experiment flag (`GOEXPERIMENT=jsonv2` or similar) that the nix devShell doesn't enable. The tool doesn't check whether imported packages actually compile before migrating to them.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate / This Session

1. ~~**Pin or exclude `go-auto-upgrade` from BuildFlow**: This migrator is dangerous in its current form — it doesn't verify post-migration compilation before committing changes. Consider `buildflow -s go-auto-upgrade` exclusion or pin to `--detect` only (no `--repair`).~~ done (.buildflow.yml skip_steps)

2. ~~**Add a CI guard for `encoding/json/v2` imports**: A simple grep-based check in the pre-commit hook or CI that fails if any file imports `encoding/json/v2` or `encoding/json/jsontext` — since the project targets Go 1.26.x which doesn't expose these packages without experiment flags.~~ **Won't implement — moot — json/v2 required via GOEXPERIMENT.**

3. ~~**Consider whether go-output v0.30.4 is worth pursuing**: If it has real fixes we need, we should either (a) enable `GOEXPERIMENT=jsonv2` in the devShell, or (b) wait for Go 1.27 where json/v2 may be stable. If not, pin go-output at v0.30.1 and add a comment explaining why.~~ done (v0.38.0 adopted)

### Structural / Process

4. ~~**BuildFlow `--max-time` for fuzz tests**: The BuildFlow output noted that the default 2m timeout is too short for 5 fuzz targets (30s each = 2.5m). Already documented in AGENTS.md but still relevant.~~ done (.buildflow.yml 5m timeout)

5. ~~**`nix-fmt` failure on `website/flake.nix`**: The `nixfmt` formatter choked on `website/flake.nix` because it appears to be JSON-in-nix syntax (the `"description"` string with `:` confused the parser). This is a pre-existing issue unrelated to this session but surfaced in the BuildFlow output.~~ done (flake checks.format)

---

## f) Up to 50 Things We Should Get Done Next

### Critical (blocks future BuildFlow runs)

1. ~~**Exclude `go-auto-upgrade` from BuildFlow `--fix`** or run it in detect-only mode~~ done (.buildflow.yml skip_steps)
2. ~~**Add pre-commit guard**: fail if `encoding/json/v2` or `encoding/json/jsontext` appears in any `.go` file~~ **Won't implement — moot — json/v2 required.**
3. ~~**Decide on go-output version policy**: pin v0.30.1 with comment, or plan migration path to v0.30.4+~~ done (go.mod lockstep comment)
4. ~~**Investigate `GOEXPERIMENT=jsonv2`**: can it be enabled in `flake.nix` devShell? Would it unblock go-output v0.30.4?~~ done (flake.nix:50)

### BuildFlow Configuration

5. ~~Fix `nix-fmt` failure on `website/flake.nix` (JSON-in-nix syntax confusing nixfmt)~~ done (checks.format)
6. ~~Investigate `govalid-generate` failures (prerequisites blocked by compilation cascade)~~ **Won't implement — superseded.**
7. ~~Set `--max-time=5m` as default for BuildFlow runs with fuzz tests~~ done (5m timeout)
8. ~~Review whether `gitignore-upserter:repair` should be promoted from `○` (skipped) to active~~ **Won't implement — one-off review.**

### Dependency Hygiene

9. ~~Evaluate go-output v0.30.2, v0.30.3, v0.30.4 changelogs for relevant fixes~~ done (v0.38.0 adopted)
10. ~~Check if any other dependencies have migrated to `encoding/json/v2` upstream~~ done (README:96)
11. ~~Pin `charmbracelet/ultraviolet` — BuildFlow bumped it from `2026-07-03` to `2026-07-13`~~ **Won't implement — indirect dep, moot.**

### CI Hardening

12. ~~Add a CI job that runs `go build ./...` before `go-auto-upgrade` could touch anything (early warning)~~ **Won't implement — CI builds anyway.**
13. ~~Add fuzz test timeout configuration to avoid `--max-time` issues~~ done (5m timeout)
14. ~~Verify the `stale-generation` CI check still passes after the schema regeneration~~ done (stale-generation job)

### Code Quality (surfaced by BuildFlow warnings)

15. ~~`goimports:detect` was `○` (not run) — verify import ordering is clean~~ done (lint CI green)
16. ~~`gofumpt:repair` was `○` — verify formatting is clean~~ done (lint CI green)
17. ~~`jscpd` (copy-paste detection) was `○` — run manually to verify clone-free status~~ **Won't implement — one-off tool run.**
18. ~~`branching-flow` was `○` — run manually if desired~~ **Won't implement — one-off tool run.**
19. ~~`hierarchical-error` checks were `○` — run manually~~ done (go-error-family v0.10.0)

### Uncommitted Changes

20. ~~**Commit `.gitignore` + `flake.lock` changes** or discard them — currently in limbo~~ done (committed)
21. ~~Review `.gitignore` JS/TS patterns: are they relevant to this project? (Only if the website/ dir uses Node.js)~~ done (committed)
22. ~~Review flake.lock nixpkgs bump: `0bb7ec5` → `e7a3ca8` — any breaking changes?~~ done (committed + drift guard)

### Documentation

23. ~~Update AGENTS.md Gotchas section: document the `go-auto-upgrade` → `encoding/json/v2` failure mode~~ done (AGENTS.md gotcha)
24. ~~Add a "Known BuildFlow Issues" section to AGENTS.md~~ done (AGENTS.md gotchas)
25. ~~Document the go-output version pinning decision when made~~ done (go.mod comment)

### Future-Proofing

26. ~~Plan migration to `encoding/json/v2` when Go 1.27 stabilizes it~~ **Won't implement — moot — json/v2 in use.**
27. ~~Evaluate whether `CompareServiceRefs` should be exported (it was public before the bot deleted it)~~ done (diff.go:169)
28. ~~Review whether the `go-output/testhelpers` indirect dep removal is safe~~ done (testhelpers/ public)
29. ~~Add a `golangci-lint` custom rule to flag `encoding/json/v2` imports~~ **Won't implement — moot — json/v2 allowed.**
30. ~~Consider a `.buildflow.yaml` or equivalent config to exclude dangerous migrators~~ done (.buildflow.yml exists)

### Testing

31. ~~Run the full coverage gate: `sh scripts/coverage-gate.sh` (CI gate: >=95%)~~ done (gate green ~95.5%)
32. ~~Run `golangci-lint run` locally to verify the strict lint config passes~~ done (lint CI green)
33. ~~Run `golangci-lint config verify` to validate lint config~~ done (ci.yml:87)
34. ~~Run `govulncheck` to verify no new vulnerabilities~~ done (ci.yml vulncheck)
35. ~~Run the 3 fuzz targets individually with extended time~~ done (8 targets, 5m)

### Release Preparation

36. ~~Review CHANGELOG.md for accuracy after this fix~~ done (CHANGELOG maintained)
37. ~~Tag a patch release if this breakage affected any published artifacts~~ done (tags v0.0.1..v0.10.0)
38. ~~Verify `go install ./cmd/auditlog` works from clean checkout~~ done (goreleaser CI)
39. ~~Verify `nix run .#auditlog -- help` works~~ done (apps.auditlog)
40. ~~Verify `nix run .#coverage` passes~~ done (apps.coverage)

### Nix

41. ~~Run `nix build` to verify the Nix build still works after flake.lock bump~~ done (checks.build)
42. ~~Run `nix flake check` for full validation~~ done (checks build+format)
43. ~~Verify `nix develop` still provides the correct toolchain~~ done (GOTOOLCHAIN pin)
44. ~~Review whether the nixpkgs bump affects Go version (should still be 1.26.4)~~ done (check-go-version.sh)

### Website

45. ~~Check `website/flake.nix` syntax — is it intentionally JSON or is it malformed nix?~~ done (formatted)
46. ~~Verify the website still builds after flake.lock bump~~ done (website.yml green)

### Process

47. ~~Consider adding a `Makefile`-equivalent `justfile`-equivalent `flake.nix` target: `nix run .#buildflow-check` that runs BuildFlow in detect-only mode~~ **Won't implement — migrator permanently skipped.**
48. ~~Review BuildFlow's `go-auto-upgrade` source to understand why it targeted json/v2~~ **Won't implement — migrator permanently skipped.**
49. ~~Consider filing a bug against BuildFlow's `go-auto-upgrade` migrator~~ **Won't implement — migrator permanently skipped.**
50. ~~Evaluate whether other BuildFlow migrators (`go-fix`, `go-generate`) are safe to run with `--fix`~~ **Won't implement — BuildFlow configured.**

---

## g) Top 2 Questions I Cannot Answer Myself

### 1. Should I commit the `.gitignore` and `flake.lock` changes, or revert them?

The `.gitignore` adds JS/TS patterns (from the `gitignore-upserter`) and `flake.lock` bumps nixpkgs from `2026-07-05` (`0bb7ec5`) to `2026-07-13` (`e7a3ca8`). Both are harmless, but I don't know if:

- The JS/TS patterns are relevant (is there a `website/` Node.js project?)
- The nixpkgs bump was intentional or another automated change to keep
- You want these committed together or separately

I left them uncommitted per project rules (no commit without explicit instruction).

### 2. Should go-output be upgraded to v0.30.4, and if so, how?

The upgrade was attempted and broke everything because v0.30.4 uses `encoding/json/v2`. I don't know:

- Whether v0.30.4 contains fixes/features you actually need
- Whether enabling `GOEXPERIMENT=jsonv2` in the devShell/CI is acceptable
- Whether we should wait for Go 1.27 (where json/v2 may be stable)
- Whether go-output v0.30.4 is even a real release or if the bot hallucinated it

This requires your decision on the Go version / json/v2 adoption timeline.

---

## Session Metadata

| Field          | Value                                                               |
| -------------- | ------------------------------------------------------------------- |
| Date           | 2026-07-13 21:28                                                    |
| Session type   | Emergency repair                                                    |
| Root cause     | BuildFlow `go-auto-upgrade` migrator (commit `ceca5b4`)             |
| Files reverted | 9 (7 Go source + go.mod + go.sum)                                   |
| Files kept     | 2 (.gitignore, flake.lock)                                          |
| Verification   | `go vet` + `go build` + `go test -race` + `go generate` — all green |
| Time to fix    | ~3 minutes (diagnosis + revert + verification)                      |

---

## Resolution (2026-07-22)

The revert was committed and follow-up hardening was applied.

| Item                                | Section        | Resolution                                                                                      | Commit    |
| ----------------------------------- | -------------- | ----------------------------------------------------------------------------------------------- | --------- |
| Committing the revert               | §c NOT STARTED | DONE: revert committed with full documentation of the incident                                  | `fb56b6a` |
| `.gitignore` + `flake.lock` changes | §g Q1          | DONE: committed alongside the revert and subsequent commits                                     | `c272da5` |
| `GOEXPERIMENT=jsonv2` in devShell   | §f item 4      | DONE: added to flake.nix to unblock potential go-output v0.30.4+ adoption                       | `c5e1f2c` |
| govulncheck CI                      | §e item 2      | DONE: replaced `govulncheck-action@v1` with `go run golang.org/x/vuln/cmd/govulncheck` approach | `c5e1f2c` |
| `encoding/json/v2` exclusion policy | §f item 23     | DONE: documented in AGENTS.md Gotchas (this session's remediation already added it)             | `fb56b6a` |

**Still open** (lower priority): `go-auto-upgrade` exclusion from BuildFlow config (item 1), go-output v0.30.4 evaluation (item 3), `nix-fmt` failure on `website/flake.nix` (item 5).
