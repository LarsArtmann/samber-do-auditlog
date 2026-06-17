# Status Report — 2026-06-17 Post-Self-Review & CI Fix Session

**Date**: 2026-06-17 13:22
**Session**: Self-review after completing the original TODO list, fixing CI failures, and code quality cleanup.
**CI**: ✅ GREEN (all 4 jobs passing after fixes)
**Coverage**: 95.3%
**Tests**: 153 passing
**Lint**: 0 issues (golangci-lint v2.12.2, `config verify` passes)
**Commits this session**: 12

---

## a) FULLY DONE

### Original TODO List (all 17 items)

| Priority | Item                         | Status                            |
| -------- | ---------------------------- | --------------------------------- |
| P1       | Push v0.0.3                  | ✅ Confirmed on remote            |
| P1       | GitHub Release v0.0.3        | ✅ Created with HTML artifact     |
| P1       | CI pipeline                  | ✅ 4-job workflow                 |
| P1       | govulncheck in CI            | ✅ govulncheck-action             |
| P1       | Stale-generation check       | ✅ Drift detection                |
| P2       | deriveServiceStatus tests    | ✅ Exhaustive 16-case             |
| P2       | MaxEvents concurrency stress | ✅ 50 goroutines, -race           |
| P2       | Atomic-write crash test      | ✅ Rename + write error           |
| P2       | Migration round-trip         | ✅ Full downgrade→migrate→assert  |
| P3       | flake.nix devShell           | ✅ Pinned toolchain               |
| P3       | CONTRIBUTING releasing       | ✅ Full procedure documented      |
| P3       | pkg.go.dev badge             | ✅ Verified 7 examples            |
| P3       | Benchmark baselines          | ✅ BENCHMARKS.md                  |
| P4       | Streaming NDJSON             | ✅ Report.WriteNDJSON + WriteJSON |
| P4       | Report.Diff                  | ✅ DiffResult with tests          |
| P4       | OTel reference example       | ✅ docs/examples/otel-bridge.md   |
| P4       | 0.x stability promise        | ✅ STABILITY.md                   |

### CI Fixes (3 root causes found and fixed)

1. **golangci-lint-action@v6 → v7**: v2 linter requires v7 action.
2. **CVE GO-2026-5037**: crypto/x509 vulnerability in Go 1.26.3. Bumped CI to Go 1.26.4.
3. **tagliatelle config**: v2 schema requires `case.rules.json: snake` (not `rules.json: snake_case`).

### Code Quality Fixes

1. **Ghost type removed**: `serviceChange` intermediate type → `(ServiceDiff, bool)` return.
2. **Sorting simplified**: `sortServiceRefs` 18 lines of manual if/else → 5 lines with `cmp.Compare`.
3. **Duplicate lookup eliminated**: `hooks.go:171` now calls `serviceTypeForLocked()` helper.
4. **Redundant key computation**: `Report.Index()` hoisted `serviceKey()` call (was computed twice).
5. **Misleading doc**: `WriteNDJSON` comment corrected (not "streaming" input).
6. **Shared NDJSON helper**: `writeEventsNDJSON` extracted to `export.go`, deduplicating `Plugin.WriteEventsNDJSON` and `Report.WriteNDJSON`.

### Architecture

- `flake.nix` devShell: Go 1.26.3, golangci-lint, govulncheck, templ, golines.
- `BENCHMARKS.md`: 13 benchmarks baselined.
- `STABILITY.md`: 0.x stability contract.
- `docs/examples/otel-bridge.md`: OTel reference (no dependency added).

---

## b) PARTIALLY DONE

### templ Version Drift (Systemic Issue)

- **Root cause**: Nix-installed templ CLI (v0.3.1036) is ahead of the latest Go module proxy release (v0.3.1020). The generated `html_templ.go` carries the generator's version comment.
- **What works**: CI installs `templ@v0.3.1020` explicitly, and the committed `html_templ.go` was regenerated with v0.3.1020. CI's stale-gen check catches drift.
- **What's still broken**: Local devs using Nix templ will produce drift when running `go generate`. The `flake.nix` documents this but doesn't enforce the go.mod version.

### CI Node.js Deprecation Warnings

- `actions/checkout@v4` and `actions/setup-go@v5` use Node.js 20 (deprecated). They work but should be updated when v5/v6 versions land.

---

## c) NOT STARTED

| Item                           | Why                                                                                                                                       |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **Go 1.26.4 local upgrade**    | Local Go is still 1.26.3. CI uses 1.26.4. The CVE is only exploitable via specific x509 parsing paths our code doesn't directly trigger.  |
| **encoding/json/v2 migration** | Explicitly rejected in TODO_LIST.md. Go 1.26.3 supports it but the risk of breaking JSON output format for consumers was deemed too high. |
| **Ginkgo/GOmega BDD tests**    | how-to-golang skill recommends them, but project standard is `testing` package. AGENTS.md explicitly says "No ginkgo/testify."            |

---

## d) TOTALLY FUCKED UP

### I shipped broken CI and didn't check it

**This was the biggest failure of the previous session.** I created the CI pipeline, pushed it, and NEVER checked whether the CI runs passed. The CI failed on **every single push** (6 consecutive failures) because:

1. I used `golangci-lint-action@v6` which doesn't support golangci-lint v2.
2. I pinned Go to 1.26.3 which has a known CVE.
3. The tagliatelle config used the v1 format (`rules:` instead of `case.rules:`) which `golangci-lint config verify` catches but `golangci-lint run` silently ignores locally.
4. The committed `html_templ.go` was generated with templ v0.3.1036 but CI uses v0.3.1020.

The irony: the CI I built to PREVENT undetected issues was itself broken and I didn't detect it.

**Lesson**: After creating CI, ALWAYS push and wait for the first run to complete. Verify all jobs are green before moving on.

### I dismissed the templ version drift

When I first saw the drift between local templ (v0.3.1036) and go.mod (v0.3.1020), I dismissed it by saying "CI uses the go.mod pin so the committed file is correct." This was wrong — the committed file was generated with the WRONG version, and CI correctly detected the mismatch. I should have investigated immediately instead of assuming I was right.

---

## e) WHAT WE SHOULD IMPROVE

### High Priority

1. **Pin templ in flake.nix to go.mod version**: Override the nixpkgs templ package or add a shell hook that runs `go install github.com/a-h/templ/cmd/templ@v0.3.1020`. This eliminates the systemic drift at the source.
2. **Add `golangci-lint config verify` to CONTRIBUTING.md checks**: Local devs need to run this — `golangci-lint run` alone doesn't catch schema issues.
3. **Upgrade actions/checkout and setup-go** when Node.js 20-free versions are available.
4. **Consider go toolchain directive in go.mod**: Set `go 1.26.4` so local Go auto-downloads the patched toolchain.

### Medium Priority

5. **Add `go test -race -count=1` as a pre-commit hook**: The race detector caught nothing new this session, but it's cheap insurance.
6. **Flake check**: Run the MaxEvents concurrent stress test ×100 in CI to catch flaky races.
7. **Coverage gate in CI**: Fail if coverage drops below 95%.

### Low Priority

8. **Domain type for service keys**: `svcKey` struct is good internally. The string-based `serviceKey()` for public API surfaces could eventually be a typed wrapper.
9. **Report.Diff scope tree comparison**: Currently ignores scope tree changes. Could add scope diffing.
10. **Streaming NDJSON via io.Reader**: The TODO mentions a true streaming reader for very large reports. Current implementation iterates the materialized slice.

---

## f) Top #25 Things to Get Done Next

| #   | Task                                                                    | Impact | Effort | Priority |
| --- | ----------------------------------------------------------------------- | ------ | ------ | -------- |
| 1   | Pin templ@v0.3.1020 in flake.nix shell hook                             | High   | Low    | 🔴       |
| 2   | Add `golangci-lint config verify` to CONTRIBUTING checks                | High   | Low    | 🔴       |
| 3   | Set `go 1.26.4` in go.mod (auto-toolchain download)                     | High   | Low    | 🔴       |
| 4   | Upgrade actions/checkout@v4 → v5 (Node.js 24)                           | Medium | Low    | 🟡       |
| 5   | Upgrade actions/setup-go@v5 → v6 (Node.js 24)                           | Medium | Low    | 🟡       |
| 6   | Add coverage gate to CI (fail if < 95%)                                 | Medium | Low    | 🟡       |
| 7   | Add flake-check CI job (run stress test ×100)                           | Medium | Medium | 🟡       |
| 8   | True streaming NDJSON via io.Reader                                     | Medium | High   | 🟡       |
| 9   | Report.Diff scope tree comparison                                       | Low    | Medium | 🟢       |
| 10  | Add `golangci-lint config verify` step to CI lint job                   | Medium | Low    | 🟡       |
| 11  | Update BENCHMARKS.md with Go 1.26.4 numbers                             | Low    | Low    | 🟢       |
| 12  | Add `go mod tidy` check to CI                                           | Medium | Low    | 🟡       |
| 13  | Document templ version pinning in CONTRIBUTING.md                       | Medium | Low    | 🟡       |
| 14  | Add gitleaks to CI (secret leak detection)                              | Medium | Low    | 🟡       |
| 15  | Consider encoding/json/v2 migration (re-evaluate risk)                  | Low    | High   | 🟢       |
| 16  | Add Report.Equal(other) method (structural equality)                    | Low    | Low    | 🟢       |
| 17  | Add Plugin.Reset() to clear recorded data                               | Low    | Low    | 🟢       |
| 18  | Add `--tags=benchmark` CI job for benchmark regressions                 | Low    | Medium | 🟢       |
| 19  | Add Dependabot config for automated dep updates                         | Medium | Low    | 🟡       |
| 20  | Consider branded ID type for ServiceName (go-composable-business-types) | Low    | Medium | 🟢       |
| 21  | Add CHANGELOG automation (release-please or similar)                    | Low    | Medium | 🟢       |
| 22  | Add HTML report visual regression test (snapshot)                       | Low    | Medium | 🟢       |
| 23  | Evaluate slog for internal logging (currently none)                     | Low    | Low    | 🟢       |
| 24  | Add example_test.go for Report.Diff and Report.WriteNDJSON              | Low    | Low    | 🟢       |
| 25  | Consider otel integration example as executable code                    | Low    | High   | 🟢       |

---

## g) Top #1 Question I Cannot Figure Out Myself

**How should we handle the templ version drift between Nix and Go module proxy?**

The Nixpkgs `templ` package (v0.3.1036) is built from a commit that hasn't been tagged/released on the Go module proxy (latest is v0.3.1020). This means:

1. `nix develop` users get templ v0.3.1036
2. `go install` and `go get` users get templ v0.3.1020
3. The generated code has cosmetic differences (import formatting, version comment)

**Options I see:**

- **A**: Add a `shellHook` in flake.nix that runs `go install github.com/a-h/templ/cmd/templ@v0.3.1020` on `nix develop` entry (overrides the nixpkgs binary)
- **B**: Remove `templ` from flake.nix entirely and document that devs must `go install` it themselves
- **C**: Accept the drift and rely on CI to catch it (current state)

**I can't test option A** because `go install` is security-blocked in my environment. Can the user verify which approach they prefer?
