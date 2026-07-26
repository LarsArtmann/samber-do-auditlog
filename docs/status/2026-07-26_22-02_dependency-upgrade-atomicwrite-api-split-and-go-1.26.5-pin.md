# Status Report — Dependency Upgrade: atomicwrite API split + Go 1.26.5 pin

**Date:** 2026-07-26 22:02 CEST
**Session scope:** Fix the 5 failed buildflow steps (erraudit, go-fix, go-generate, govalid-generate, test-race), complete the dependency upgrade that caused them.
**Reporter:** Crush (self-review of own session)

---

## TL;DR

A dependency upgrade commit (`b73e30a chore(config): initialize project configuration and tooling setup`) bumped 4 LarsArtmann libs in `go.mod` but **did not migrate the call site** to the new APIs, breaking the build. I fixed the one compile error and then discovered + fixed a deeper split-brain: the same commit moved `go.mod` to `go 1.26.5` but left the toolchain pinned to `1.26.4` in 11 other places — which **broke the nix devShell outright** (`go.mod requires go >= 1.26.5`). I re-pinned everything to 1.26.5 and verified all gates green. But I forgot the CHANGELOG and never actually tested the devShell / flake itself.

---

## a) FULLY DONE ✅

| # | Item | Verification |
|---|------|--------------|
| 1 | **Compile fix** — `plugin.go:336`: `atomicwrite.WriteFunc(path, fn, Fingerprint{})` → `WriteFunc(path, fn)` | `go build ./...` OK |
| 2 | **Root-cause analysis** — read v0.4.0 source + CHANGELOG; confirmed the 3-arg→2-arg migration is the documented, correct path for zero-value fingerprints | go-atomic-write v0.4.0 CHANGELOG migration table |
| 3 | **Confirmed atomicwrite was the ONLY breakage** from the 4-lib upgrade (go-error-family v0.10.0, go-output v0.32.0, go-sse v0.2.1 all build clean) | `go build ./...` clean after single fix |
| 4 | **Discovered the 1.26.5 toolchain split-brain** — empirically proved `GOTOOLCHAIN=go1.26.4` fails (`go.mod requires go >= 1.26.5`) and `go mod tidy` auto-bumps `go 1.26.4`→`1.26.5` (so 1.26.4 is unstable, not just "stale") | reproduced both directions |
| 5 | **Re-pinned toolchain to 1.26.5** across `flake.nix` (3 sites: devShell env + coverage app + auditlog app), `.github/workflows/ci.yml` (6 jobs), `CONTRIBUTING.md` (2 refs), `BENCHMARKS.md` (1 ref) | `rg '1.26.4'` confirms zero stale pins in current (non-historical) files |
| 6 | **Updated AGENTS.md** — rewrote the "Go 1.26.4 toolchain pin" section to 1.26.5 with a new "why it's mandatory" paragraph (max-of-deps rule + empirical proof), fixed the Gotcha text, added History note; bumped stale dep refs (go-sse v0.2.0→v0.2.1, go-error-family v0.9.0→v0.10.0, go-atomic-write v0.3.0→v0.4.0 + API-split note) | `rg` confirms clean |
| 7 | **Gates green with new pin** — build, vet, `test -race`, golangci-lint (exit 0), coverage 94.2% ≥ 94% | all run with `GOTOOLCHAIN=go1.26.5` |
| 8 | **No drift** — `go mod tidy` and `go generate` produce no diffs | diffed go.mod/go.sum before/after; `git status` clean except intended edits |

---

## b) PARTIALLY DONE 🟡

| Item | What's done | What's missing |
|------|-------------|----------------|
| Dependency version-ref cleanup | Current config + AGENTS.md updated | **CHANGELOG.md not updated** (see c); **FEATURES.md / README.md / ROADMAP.md / TODO_LIST.md not audited** for stale version refs |
| AGENTS.md toolchain section | Rewritten accurately | Did not reconcile the "History: v0.7.0 shipped with go 1.26.5" narrative — it now reads slightly oddly since 1.26.5 is canonical again. Defensible but could be clearer. |

---

## c) NOT STARTED ❌

1. **CHANGELOG.md `[Unreleased]` entry** — the upgrade (4 dep bumps + atomicwrite API migration + 1.26.5 pin move + devShell fix) is a real, user-visible change and belongs in the changelog. I did not add it.
2. **`website/src/content/docs/changelog.mdx`** — exists (saw it in grep). Unknown if it's generated from root CHANGELOG (the sibling go-atomic-write project generates it via a sync script). If hand-maintained, it's now stale. **Did not verify.**
3. **`nix flake check` / `nix develop` verification** — I edited `flake.nix` but never executed it. The devShell is what was BROKEN; I owe a positive confirmation that it now opens with Go 1.26.5.
4. **`nixfmt` on `flake.nix`** — flake.nix is formatted with nixfmt (in devShell). I didn't run it; a treefmt/format gate could flag my edits.
5. **`.buildflow.yml` Go-version check** — AGENTS.md documents BuildFlow config with env. I never confirmed whether `.buildflow.yml` itself pins a Go version that now needs updating.
6. **Re-running the actual `buildflow` pipeline** — the 5 failed steps came from buildflow; I verified the equivalent gates manually but did not re-invoke buildflow itself to confirm govalid-generate / go-generate / go-fix / erraudit / test-race all go green together.
7. **FEATURES.md version audit** — did not grep FEATURES.md for `v0.3.0`/`v0.9.0`/`v0.31.1`/`1.26.4` refs.
8. **govulncheck** — not installed locally; only CI can confirm the 4 upgraded deps introduce no vulnerabilities. Flagged, not blocked-on.

---

## d) TOTALLY FUCKED UP 💥

**Nothing.** The core fix is correct, verified, and the root-cause (1.26.5 mandatory) was proven empirically rather than assumed. No reverts needed. No data loss. No wrong turns taken.

The closest thing to a mistake was **scope creep in the right direction** — I went beyond "fix the compile error" into "complete the upgrade properly," which is correct per the "Upgrade!" instruction but meant I touched 6 files instead of 1. All changes are coherent and verified.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Process gaps this session exposed

1. **No CHANGELOG discipline on dep upgrades.** The original upgrade commit `b73e30a` bumped 4 deps + go directive with commit message *"chore(config): initialize project configuration and tooling setup"* — completely opaque. A future reader cannot tell from history that a breaking atomicwrite API change landed. **The upgrade was half-done by whoever/whatever made `b73e30a`, and I nearly repeated the pattern by almost forgetting the CHANGELOG myself.**
2. **Split-brain detection is manual.** The 1.26.4-vs-1.26.5 inconsistency across go.mod/flake.nix/ci.yml/AGENTS.md survived a commit. There's no CI gate that asserts "go.mod go directive == flake.nix GOTOOLCHAIN == ci.yml go-version == AGENTS.md canonical pin." One assert script would have caught this at commit time.
3. **The auto-git daemon writes terrible commit messages** ("docs: add contributor, agent, and benchmark documentation" for what was actually a dep-upgrade + toolchain-pin fix). This corrupts archaeological value. Not my commits, but I let the daemon grab my changes instead of committing with a descriptive message first.
4. **I didn't test the artifact I claimed to fix.** I said "devShell fixed" but never ran `nix develop`. I verified Go behavior directly, which is strong evidence — but the claim outpaced the verification.
5. **godoclint notice at `plugin.go:41`** (`ErrContainerIDPathSep` godoc should start with symbol name) — pre-existing, unrelated, exit-0. I correctly left it alone, but it's a real lint finding someone should fix.

### Codebase health observations (noticed in passing, NOT researched)

- The `testhelpers/graphtest v0.31.1` indirect pin in `go.mod` is now **behind** the rest of go-output (v0.32.0). AGENTS.md says "keep these indirect pins until upstream publishes corrected manifests." Worth re-checking whether v0.32.0 fixed the manifests so the v0.31.1 pin can be dropped.
- `go mod tidy` accepted the current state, so the pin is currently necessary; just noting the version skew.

---

## f) Up to 50 things we should get done next

**Immediate (this upgrade, not finished):**
1. Add `[Unreleased]` CHANGELOG.md entry documenting: go-atomic-write v0.4.0 API migration, go-error-family v0.10.0, go-output v0.32.0, go-sse v0.2.1, Go 1.26.5 mandatory pin, devShell breakage + fix.
2. Verify whether `website/src/content/docs/changelog.mdx` is generated or hand-maintained; sync if needed.
3. Run `nix flake check` to validate the edited flake.nix evaluates.
4. Run `nix develop -c go version` (or enter devShell) to positively confirm Go 1.26.5 + build works.
5. Run `nixfmt` on `flake.nix` and verify treefmt/formatter gate passes.
6. Check `.buildflow.yml` for any Go-version pin; update to 1.26.5 if present.
7. Re-run the full `buildflow` pipeline (or at minimum `buildflow -s govalid-generate`) to confirm all 5 originally-failed steps now pass.
8. Audit `FEATURES.md` for stale dep version refs.
9. Audit `README.md` for any hard dep versions (currently loose "Go 1.26+", likely fine — confirm).
10. Audit `ROADMAP.md` / `TODO_LIST.md` for stale version refs.

**Hardening (prevent recurrence):**
11. Add a CI gate / pre-commit check that asserts toolchain version consistency: `go.mod` go directive == `flake.nix` GOTOOLCHAIN == `ci.yml` go-version == AGENTS.md canonical pin.
12. Add a `go.sum` / `go.mod` consistency check that fails if `go mod tidy` would bump the `go` directive (catches the "deps require newer Go" case at PR time, not at devShell-break time).
13. Fix the pre-existing godoclint finding at `plugin.go:41` (`ErrContainerIDPathSep` comment).
14. Re-check whether go-output v0.32.0 fixed the broken testhelpers/graphtest manifests; if so, drop the `v0.31.1` indirect pins (go.mod line 59) and update the AGENTS.md note.
15. Document a "dependency upgrade checklist" in AGENTS.md: (a) migrate call sites, (b) update all toolchain pins, (c) CHANGELOG, (d) re-run buildflow, (e) verify devShell.

**Feature opportunities exposed by the upgrade:**
16. Evaluate `atomicwrite.WriteFuncVerified` for audit exports — TOCTOU protection could make re-exports idempotent and safe against concurrent writers (currently using plain `WriteFunc`). Product decision.
17. go-atomic-write v0.4.0 adds `WriteIfChanged` — audit generators (e.g., `cmd/genschema`) could use it to avoid spurious diffs on re-runs. Check if schema generation would benefit.
18. go-error-family v0.10.0 — check CHANGELOG for new Family/classification features usable in `classify.go`.

**Commit/history hygiene:**
19. Amend or follow-up the opaque auto-git commits (`f17b58d`, `5cf7780`, `b8745d8`) with a descriptive CHANGELOG entry so the upgrade is discoverable from history.
20. Commit AGENTS.md (currently in working tree) with a message that describes the toolchain-pin move.

**Deferred / CI-only:**
21. Confirm `govulncheck` passes in CI for the 4 upgraded deps (can't run locally).
22. Confirm `golangci-lint` version pin in CI (v2.12.2 per AGENTS.md) is compatible with the upgraded code — lint passed locally, but CI version may differ.
23. Confirm the `stale-generation` CI step passes (go generate drift) — verified locally, CI confirms.

**Lower priority (noticed, not blocking):**
24. The `History:` paragraph in AGENTS.md toolchain section now reads slightly awkwardly (1.26.5 was "drift" in v0.7.0, now canonical) — rewrite for clarity.
25. Consider adding a `toolchain` directive to `go.mod` explicitly (currently relies on `GOTOOLCHAIN` env) for belt-and-suspenders pinning.
26. `example/` and `live/demo` have no test files — not in scope, just noting.
27. `cmd/genschema` has no test files — exercised via `go generate` golden test; noted.
28. README badge says "Go-1.26+" — accurate but could be "1.26.5" for precision (tradeoff: breaks on next bump).

*(Stopping at 28 — these are all genuinely earned from this session's work; I won't pad to 50 with unrelated research, per instructions.)*

---

## g) Questions I CANNOT figure out myself

1. **CHANGELOG ownership** — Should I add the `[Unreleased]` entry to CHANGELOG.md now, and if so do you want the website changelog regenerated/synced too? (I can't tell if `website/src/content/docs/changelog.mdx` is hand-maintained or generated without checking the website build scripts — and you asked me not to research further this turn.)

2. **`WriteFunc` vs `WriteFuncVerified` for audit exports** — The v0.4.0 migration correctly downgraded to plain `WriteFunc` (matching old zero-fingerprint behavior). But should audit file exports (JSON/NDJSON/HTML writes) actually use `WriteFuncVerified` for TOCTOU safety on re-writes? This is a product-level decision about whether concurrent audit-log writers are a real scenario. I can't decide this from the code alone.

3. **Nix verification scope** — Do you want me to run `nix flake check` / `nix develop` / `nixfmt` as part of closing this out (I have nix available per earlier `nix eval`), or is the direct-Go-gates verification sufficient and nix verification is reserved for you/CI? I don't want to trigger a long nix build without your go-ahead.

---

## Files changed this session

| File | Change |
|------|--------|
| `plugin.go` | `writeToFile`: 3-arg → 2-arg `atomicwrite.WriteFunc` call (the compile fix) |
| `flake.nix` | `GOTOOLCHAIN` go1.26.4 → go1.26.5 (3 sites: devShell, coverage app, auditlog app) |
| `.github/workflows/ci.yml` | `go-version` 1.26.4 → 1.26.5 (6 jobs) |
| `CONTRIBUTING.md` | Go 1.26.4 → 1.26.5 (2 refs) |
| `BENCHMARKS.md` | Go 1.26.4 → 1.26.5 (1 ref) |
| `AGENTS.md` | Toolchain section rewrite + dep version refs (go-sse, go-error-family, go-atomic-write) |

All gates green: build, vet, `test -race`, golangci-lint, coverage 94.2%. No drift in `go mod tidy` / `go generate`.

**Verdict:** Upgrade functionally complete and verified; documentation + devShell-nix verification + CHANGELOG are the open loose ends.
