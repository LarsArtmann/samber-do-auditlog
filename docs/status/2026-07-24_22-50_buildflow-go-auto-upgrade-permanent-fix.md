# Status Report: BuildFlow go-auto-upgrade Fix & AGENTS.md Reconciliation

**Date**: 2026-07-24 22:50  
**Session Scope**: Diagnose and permanently fix the `go-auto-upgrade` buildflow failure, reconcile stale documentation.  
**Commits**: `1bfb9a8` (`.buildflow.yml`), `9d17856` (AGENTS.md gotcha updates)  
**Branch**: `master` — pushed to `origin/master`

---

## 1. What Happened This Session

### The Problem

`buildflow` reported:

```
go-auto-upgrade failed during execution: migrations broke compilation
./report.go:439:6: enc.SetIndent undefined (type *jsontext.Encoder has no field or method SetIndent)
```

### Root Cause (fully traced)

The `jsonv1tov2` migrator in [`go-auto-upgrade`](https://github.com/larsartmann/go-auto-upgrade) (`internal/migrators/jsonv1tov2/rewrites.go:156-162`) detects that `GOEXPERIMENT=jsonv2` is active (required by this project's transitive deps: `go-output`, `go-branded-id`, `go-ndjson`). It then rewrites `encoding/json` imports to `encoding/json/v2` + `encoding/json/jsontext`.

The critical bug: the migrator's `SetIndent` handler (`kindSetIndent` at `rewrites.go:156`) only emits a **warning** via `methodWarning()` — it does NOT rewrite the call. It converts `json.NewEncoder` to `jsontext.NewEncoder` (via `kindNewEncoderStored`), but leaves `enc.SetIndent("", "  ")` untouched. Since `jsontext.Encoder` has no `SetIndent` method (indent is configured at construction time via `jsontext.WithIndent`), compilation breaks.

This is fundamentally incompatible with the project's `encoding/json/v2` exclusion policy (AGENTS.md line 239): no `.go` file in this project may import `encoding/json/v2` until Go 1.27 stabilizes it.

### The Fix

Created `.buildflow.yml` with three permanent settings:

```yaml
skip_steps:
  - go-auto-upgrade          # permanently incompatible with json/v2 exclusion policy

env:
  GOEXPERIMENT: jsonv2       # belt-and-suspenders for non-devShell runs

max_time: 5m                 # fuzz tests need ~30s each × 5 targets
```

**Verification**: `buildflow --no-tui` passes cleanly — **27/29 steps green**, `go-auto-upgrade` skipped via config, `gitleaks` skipped by build mode.

### Documentation Reconciliation

Updated 3 stale AGENTS.md gotchas + 1 section:

| Location | Before | After |
|----------|--------|-------|
| Line 141 (GOEXPERIMENT section) | Listed 4 sources where GOEXPERIMENT is set | Added 5th: BuildFlow config via `ApplyConfigEnv` |
| Line 236 (`--max-time`) | Described as open problem needing CLI flag workaround | Marked **RESOLVED via `.buildflow.yml`** (`max_time: 5m`) |
| Line 237 (GOEXPERIMENT env) | Falsely claimed `env:` config key "doesn't work" (`config view` ignores it) | Corrected: `env:` IS applied at runtime via `ApplyConfigEnv` (`pipeline.go:102`); `config view` display limitation noted |
| Line 238 (`go-auto-upgrade`) | Described as "DANGEROUS", recommended ad-hoc exclusion | Marked **RESOLVED via `.buildflow.yml`** `skip_steps`; permanent rationale documented |
| Line 239 (json/v2 exclusion) | Listed `go-output`, `go-branded-id` as transitive deps using json/v2 | Added `go-ndjson` (was missing) |

---

## 2. FULLY DONE ✅

1. **`.buildflow.yml` created and committed** — `skip_steps: [go-auto-upgrade]`, `env: GOEXPERIMENT: jsonv2`, `max_time: 5m`, `build_mode: full`, `parallel: true`, `max_concurrency: 4`.
2. **Config validated** — `buildflow config validate` passes all checks.
3. **Full buildflow run passes** — 27/29 steps green, 2 skipped via config (go-auto-upgrade + gitleaks). Zero failures.
4. **AGENTS.md gotchas reconciled** — 3 buildflow gotchas marked RESOLVED with permanent fix references. False `env:` claim corrected.
5. **`go-ndjson` added to exclusion policy note** — was missing from the transitive dependency list.
6. **Pushed to `origin/master`** — 4 commits total (2 from prior session + 2 from this session).

---

## 3. PARTIALLY DONE 🟡

1. **AGENTS.md `env:` claim correction** — The old text said "has no working `env:` config key (the key is silently ignored by `config view`)". The new text explains it IS applied at runtime but `config view` doesn't display it. However, the underlying BuildFlow display bug (`config view` doesn't show `env:` or `skip_steps`) is NOT fixed in BuildFlow itself — only documented as a known limitation.

---

## 4. NOT STARTED ⬜

1. **`gomod-check` finding** — `go.mod:32: direct and indirect requires are mixed (should be separate blocks since Go 1.17+)`. Surfaced during the full buildflow run. Not fixed — pre-existing.
2. **`go-structure-linter` findings** — 29 `root-package-files` errors. Pre-existing architectural decision (AGENTS.md explicitly says "Do NOT modularize — Project is 1 package, ~2,500 LOC"). Not actionable without reversing that decision.
3. **`.github/workflows/website.yml` SHA pinning** — 6 `github-actions-pinned` errors (actions using `@v6` tags instead of commit SHAs). Pre-existing, not related to this session.
4. **BuildFlow `config view` display bug** — Doesn't show `env:` or `skip_steps` blocks. Upstream issue in BuildFlow, not this project.

---

## 5. TOTALLY FUCKED UP 💥

1. **First commit message was auto-generated by hook** — The `.buildflow.yml` commit (`1bfb9a8`) was auto-committed by the buildflow pre-commit hook with a generic message ("chore(ci): add buildflow configuration for CI/CD pipeline automation" + 7 bullet points of corporate-speak filler). I did not write or review this message before it landed. It's technically accurate but doesn't explain the root cause or the `skip_steps` rationale. The AGENTS.md commit (`9d17856`) has a proper message.

---

## 6. WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Review hook-auto-generated commits immediately** — The pre-commit hook auto-committed `.buildflow.yml` with a filler message before I could review it. Should either disable auto-commit on hooks or immediately `git commit --amend` to fix the message.
2. **Run `buildflow config view` after creating config** — I validated syntax but didn't thoroughly inspect the materialized config. `config view` has display gaps (doesn't show `env:` or `skip_steps`), which delayed my confidence in the fix.

### Codebase Improvements (noticed during this session)

3. **`report.go:437-438` uses `json.NewEncoder` + `SetIndent`** — This is the exact code that the migrator broke. It's standard `encoding/json` v1 code and works fine, but it's a magnet for automated migration tools. Consider a helper function (`writeJSONIndented(w, v)`) to centralize the pattern and make future migration (when Go 1.27 lands) a single-point change.
4. **`export.go:13` also uses `json.NewEncoder`** — Same pattern, same migration magnet.
5. **`cmd/genschema/main.go:33` uses `json.MarshalIndent`** — Same pattern in the schema generator.
6. **`live/server.go:478` uses `json.NewEncoder`** — Same pattern in the live dashboard sub-package.
7. **17 `.go` files import `encoding/json`** — All are migration targets for `jsonv1tov2`. The `skip_steps` config protects them all, but if anyone re-enables `go-auto-upgrade` without reading the rationale, all 17 files will be rewritten.

### Documentation Improvements

8. **AGENTS.md gotcha style** — The "RESOLVED via" prefix I added is ad-hoc. Consider a consistent annotation system for resolved gotchas (e.g., a `~~/strikethrough~~` convention or moving resolved items to a separate "Historical" section).
9. **`.buildflow.yml` comment references "Go 1.27"** — This is a forward-looking assumption. When Go 1.27 actually ships, someone needs to verify json/v2 is truly stable (not just experimentally available) before re-enabling `go-auto-upgrade`.

---

## 7. NEXT 50 THINGS TO DO

### High Priority (blockers / correctness)

1. Fix `gomod-check` finding: `go.mod:32` mixed direct/indirect requires — run `go mod tidy` to separate blocks
2. Pin GitHub Actions to commit SHAs in `.github/workflows/website.yml` (6 findings)
3. Verify the Dependabot vulnerabilities flagged on push (1 high, 1 moderate) — check `github.com/LarsArtmann/samber-do-auditlog/security/dependabot`

### BuildFlow & CI

4. Consider adding `build_mode: full` to CI workflow (`.github/workflows/ci.yml`) to run buildflow in CI, not just locally
5. Add `.buildflow.yml` validation step to CI (run `buildflow config validate`)
6. Consider whether `test-fuzz` step needs its own `max_time` override (currently covered by global `max_time: 5m`)
7. Document the BuildFlow `config view` display bug upstream (env:/skip_steps not shown)
8. Consider adding `gitleaks` to a CI workflow (currently skipped by build mode 'full')

### Code Quality (noticed this session)

9. Extract `writeJSONIndented(writer io.Writer, v any) error` helper to centralize `json.NewEncoder` + `SetIndent` pattern (used in `report.go:437` and `export.go:13`)
10. Consider whether `cmd/genschema/main.go:33` (`json.MarshalIndent`) should use the same helper
11. Review if `live/server.go:478` (`json.NewEncoder`) should be centralized too

### Architecture / Type Model

12. Consider a `JSONWriter` interface or type that encapsulates JSON encoding options (indent, escape HTML) — would make the json/v1→v2 migration a single-point change when Go 1.27 lands
13. Consider whether `encoding/json` usage should be restricted via depguard to a single helper file (similar to how `invopop/jsonschema` is restricted to `cmd/`)

### Documentation

14. Add `.buildflow.yml` to the Commands table in AGENTS.md (currently not mentioned)
15. Update `CHANGELOG.md` with the `.buildflow.yml` addition and the AGENTS.md reconciliation
16. Consider whether the 3 buildflow gotchas should be moved to a "Historical Incidents" section now that they're resolved
17. Update `FEATURES.md` if buildflow integration is considered a feature (currently not listed)

### Testing

18. Add a test that verifies `.buildflow.yml` parse correctness (defends against YAML typos)
19. Add a test or CI check that verifies `go-auto-upgrade` remains in `skip_steps` (guards against accidental removal)
20. Consider a golden-file test for `.buildflow.yml` content (catches drift)

### Security

21. Run `govulncheck` to identify the 2 Dependabot vulnerabilities
22. Review whether the flagged vulnerabilities affect production code or only dev dependencies
23. Update vulnerable dependencies if patches are available

### Developer Experience

24. Add `.buildflow.yml` to `.gitignore` managed block? (No — it should be tracked. But verify it's not accidentally in a gitignore pattern)
25. Consider adding `buildflow config validate` to the pre-commit hook
26. Document the `max_time: 5m` rationale in a comment in `.buildflow.yml` (currently has a comment but could be more specific about fuzz target count)

### Cleanup

27. Remove `cover.out` and `cover-filtered.out` from the repo root if they're committed (they appear in `ls` output — check `.gitignore`)
28. Review whether the auto-generated `.gitignore` block (buildflow-managed) needs updating
29. Clean up `docs/status/` — verify old status reports are still accurate or need annotation

### Future-Proofing

30. Create a tracking issue for Go 1.27 json/v2 stabilization — re-evaluate `skip_steps` when it ships
31. Create a tracking issue for the `jsonv1tov2` migrator's `SetIndent` bug upstream in `go-auto-upgrade`
32. Consider whether `lo2stdlib`, `stdlib2lo`, `stdlibwrappers` and other migrators in `go-auto-upgrade` would also cause problems if re-enabled individually
33. Evaluate whether BuildFlow's `circuit_breaker` would auto-skip `go-auto-upgrade` after enough failures (reducing need for the manual `skip_steps`)

### Observability

34. Add a CI badge for buildflow status (if buildflow runs in CI)
35. Consider telemetry/logging for buildflow runs (BuildFlow has built-in audit logging)
36. Review the `workflow-audit-log.*` pattern in `.gitignore` — is audit logging enabled?

### Minor / Polish

37. Standardize `.buildflow.yml` comment style (currently mixed `#` styles)
38. Consider adding `todo_min_severity: warning` instead of `info` to reduce noise
39. Review whether `max_file_size: 350` and `dupl_threshold: 30` are appropriate for this project
40. Consider whether `parallel: true` and `max_concurrency: 4` are optimal for this project size
41. Add a `.buildflow.yml` reference to `CONTRIBUTING.md` (currently not mentioned)
42. Verify `.buildflow.yml` works with `nix develop` (it should, since GOEXPERIMENT is set by both)
43. Consider whether `auto_fix: false` should be `true` for local development (CI should stay `false`)
44. Review whether `verbose: false` should be `true` for better debugging
45. Consider adding `log_level: debug` for development builds
46. Add a comment in `.buildflow.yml` explaining why `gitleaks` is skipped by build mode 'full'
47. Document the relationship between `.buildflow.yml` `env:` and `.envrc` `use flake` (both set GOEXPERIMENT)
48. Consider whether the `.buildflow.yml` should be split into dev/CI variants
49. Review whether `build_mode: full` is appropriate for pre-commit (the hook overrides to `pre-commit` mode anyway)
50. Consider adding a `Makefile` target or `flake.nix` app for `buildflow` (e.g., `nix run .#buildflow`)

---

## 8. Questions

1. **Should I amend the auto-generated commit message for `1bfb9a8`?** The pre-commit hook auto-committed `.buildflow.yml` with a generic corporate-speak message. I can `git commit --amend` on that commit to replace it with a proper root-cause explanation, but it would require a force-push (`--force-with-lease`). Is that worth doing, or should we leave history as-is?

2. **Should I fix the `gomod-check` finding now (`go.mod:32` mixed direct/indirect requires)?** It's a one-line fix (`go mod tidy` separates the blocks), but it's unrelated to the buildflow fix and would be a separate commit. Want me to do it in a follow-up?

3. **Should the 3 resolved BuildFlow gotchas in AGENTS.md be moved to a "Historical Incidents" section?** Currently they're marked `(RESOLVED via .buildflow.yml)` inline. An alternative is to move them to a separate section or remove them entirely, since the problem is permanently solved. What's your preference for documenting resolved issues?
