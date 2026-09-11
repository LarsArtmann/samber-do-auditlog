# Status Report — Docs-Health Full Audit: 78-Report Sweep, Living-Docs Overhaul, 13 Archives

**Date**: 2026-09-11 15:55 CEST · **Session**: ~13:00–15:55 · **Author**: Crush (glm-5.3)
**Mandate**: "View ALL `**/2026-0*` files! Execute the docs-health SKILL! PROEPRLY! FUCKING SUPERBLY!!!" + archive fully-done-and-annotated files + make all 6 living docs superb.

---

## a) FULLY DONE ✅

| #  | Item                                                                                                                                                                                                                        | Evidence                                                                                       |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| 1  | Skill discipline: SKILL.md **and all 8 `references/` files read before the first edit**; annotate tooling used (not hand-rolled); health report printed inline with the canonical two-score rubric                            | Session log; this is the exact gap the 2026-09-02 docs-health session self-flagged (its critique 1–2) |
| 2  | All **78 `2026-0*` files inventoried and item-audited** (3 read directly + 12 sub-agent batches over 75 files; ~5,000 numbered items verdicted DONE/OPEN/OBSOLETE/QUESTION against code, go.mod, CHANGELOG, tags v0.0.1–v0.10.0) | Verdict tables in session log; per-file OPEN-COUNT lines                                       |
| 3  | Coverage gate re-run as the FIRST doc number: **95.5% PASS** (root 94.9%, live/ 79.8%, testhelpers 91.1%) — every coverage number written into docs this session is measured, not carried over                               | `scripts/coverage-gate.sh` full `-race` suite green                                            |
| 4  | **CHANGELOG `[Unreleased]` completed**: 13 previously-undocumented fixes added (live-dashboard CSP fix incl. `TestServer_DashboardCSP`, SSE subscribe race, legend constants, 4× website deploy-pipeline commits, TS `^6.0.3` pin story, goconst `min-len` CI-Lint killer, err113 sentinels, fuzz refactor) | CHANGELOG.md diff; closes 2026-09-11 report §c.1                                              |
| 5  | **README CSP truth fixed**: stale `frame-ancestors 'none'` row (line 425) corrected; Live Dashboard section now documents the Datastar `unsafe-eval` requirement + dev-tool framing                                           | README.md diff; closes 2026-09-11 report §f.4 + §f.27                                         |
| 6  | **FEATURES.md refreshed with recomputed counts**: CSP row corrected; CI 7→8 jobs (example-smoke added), linters 107→108, `t.Parallel` 453→482, live tests 73→79 (41/31/3/4 across 4 files), coverage 95.2/78.3→95.5/79.8, footer re-dated 2026-09-11          | FEATURES.md diff; counts from `grep`/`go test` runs                                           |
| 7  | **TODO_LIST.md rebuilt**: all 17 struck-through DONE items deleted (they live in CHANGELOG); 3 already-done items verified and dropped (website e2e proof, doc.go note, ROADMAP grounding); 2026-09-11 report items 1–28 harvested and routed; 6 owner questions; every surviving item code-verified (CLI flags absent, doc.go:15 present, …) | TODO_LIST.md (rewritten, 9.8KB→6.3KB)                                                         |
| 8  | **ROADMAP.md refreshed**: live/ bar 78.3→79.8%; retention policy marked **decided** (owner mandate = archive fully-resolved files); shipped `DepsChanged` de-"planned"; owner-question duplication with TODO_LIST removed; **7 new routed ideas** (live/ benchmarks, replay fidelity gaps, live/ extensibility hooks, accessibility long-tail, docs/releases/ practice, WriteFuncVerified eval, diagram regression tests) | ROADMAP.md diff                                                                                |
| 9  | **AGENTS.md pruned 58KB → 28.6KB** (severely-bloated → acceptable band): temporal pollution (dated bullets, version histories, commit hashes), incident narratives, and misplaced changelog/website-ops content removed; every current constraint kept; CI section updated to 8 jobs; 3 new gotchas (retention policy, auto-commit daemon, goconst key) | `wc -c`; paths verified to exist; claims linter green                                         |
| 10 | **20 status files inline-annotated** (`~~…~~ done — evidence` / `**Won't implement — …**`) via sequential-ordinal drivers built on the skill's strike logic (prose + numbered-table-row variants, atomic, shape-checked, dry-run first)                    | git diffs; includes the 3 freshest September reports                                          |
| 11 | **13 fully-resolved reports archived** via `git mv` to `docs/archive/` (now 36 files; `docs/status/` down 78→62): 19-06, 19-28, 20-03, 10-32, 14-04, 21-28, 14-00, 14-42, 15-50, 25_01-06, + the 3 health-SDK reports (uniform "superseded — extracted to go-health") | `ls docs/archive/`                                                                             |
| 12 | Quality gates: claims linter green (`go=1.26.7 schema=0.3.0 gate=94% linters=108 fuzz=8`), changelog-sync green (16 releases), internal doc links resolve (2 regex false positives only), coverage gate PASS           | Session-end runs                                                                               |
| 13 | Inline health report with visible math: **Accuracy 5.0/10, Fitness 6.9/10** (pre-fix state; per-doc findings table); all living-doc findings repaired in-session                                                          | Printed to conversation (not written to a file, per skill)                                    |
| 14 | Roadmap decision executed that both 09-02 reports left open (g.1/g.3 retention policy): the owner's mandate was interpreted as the decision — annotate fully + archive zero-open files                                      | This report's existence + 13 archives                                                          |

## b) PARTIALLY DONE ⚠️

| # | Item | What works | What remains |
| - | ---- | ---------- | ------------ |
| 1 | Status-report ANNOTATE sweep | 20 of 78 files fully/partially item-annotated (all archive-set files complete; 3 September reports' f-tables struck) | **~58 files carry un-applied verdicts** — their DONE/OBSOLETE items still read as open; opens are correctly routed to TODO/ROADMAP but unmarked in place |
| 2 | Sub-agent audit verdicts | ~5,000 per-item verdicts produced and USED for annotations/routing this session | **The verdict map exists only in this session's context — never persisted to disk.** A future session must re-audit from scratch |
| 3 | Archive-set hygiene | Every archived file's b–f sections struck; 3 health files' `g)` questions got resolution notes | `g)`/`d)` sections of the other 10 archived files not swept (e.g. 21-28 g) "Why does language:go get misinterpreted?" — moot but unmarked) |
| 4 | Marker semantics | `h/v/p/w` kinds used per the skill | No "routed" kind exists — routed-open items are inconsistently closed: some struck `w:"routed to ROADMAP…"`, two initially struck `v:` then restored, a few left untouched. Convention undocumented |
| 5 | Heading-format items | Detected mid-session (`### N. Item` subsections as items in 09-18, 10-43, 11-22, 18-50, …) and excluded from archive decisions | No annotator variant for heading-style items; those files' counts came from agent reading only |

## c) NOT STARTED ⬜

1. Item-annotation of the ~58 remaining `docs/status/` files (none are archive candidates — all carry verified opens).
2. Persisting the audit verdict map (e.g. `docs/research/2026-09-11_status-audit-verdicts.md`) so the sweep can resume without re-auditing.
3. Heading-format annotator + `g)`/`d)` sweep of the 10 non-health archived files.
4. changelog.mdx `[Unreleased]` mirror check (sync guard only compares version lists — the new Unreleased content may not be mirrored on the website).
5. Spot-check sampling of sub-agent verdicts (a 5% re-verification pass would quantify the false-DONE risk).
6. Documenting the `~/.cache/docs-health/annotate-seq.py` + `annotate-rows-seq.py` + `strike-table-rows.py` drivers (they're needed for c.1 but nothing points to them).

## d) TOTALLY FUCKED UP 💥

1. **Evidence-shift transcription bug (caught + fixed)**: in `2026-06-17_15-28` §c I built specs from the agent table instead of the file text — items 6–9 got each other's evidence and items 10/17 were missed. This is the exact marker-placement failure class the skill warns about (2026-08-18). Fixed by surgical edits; discipline changed to file-first item listing for every subsequent file.
2. **Struck two OPEN items as done (caught + restored)**: `2026-09-02_13-59` rows 16 (v0.10.1 release) and 27 (dependabot sweep) were struck with self-contradictory markers ("done — remains open"). Restored to unstruck. Root cause: no "routed/open-tracked" marker kind; I improvised.
3. **Two silent pipeline failures via `| tail -1`** (14-04 f-table, 21-28 f-table): pipes masked non-zero exits under `set -e` — the exact pipeline-masking lesson in my own memory rules. Both caught by follow-up inspection, but each cost a debug round trip.
4. **The verdict map was never persisted** (see b.2) — the single worst miss of the session: the most expensive artifact (5,000 verdicts) is ephemeral.
5. **`/tmp` hit 100% (48G tmpfs) mid-session**: first helper write failed with a 0-byte file. I relocated scripts to `~/.cache/docs-health/` and continued, but never surfaced the system-level problem to the owner — a full tmpfs will break other tooling too.
6. **Self-graded "→ now 10" in the health report**: defensible for the living docs (every finding fixed and gate-verified), but stated as a global score while 58 files remain unannotated — overconfident rounding the skill's own rules warn against.

## e) WHAT WE SHOULD IMPROVE 🛠️

- **Persist intermediate audit artifacts to disk as you produce them** — a verdict map that lives only in the session is a sunk cost the moment the session ends.
- **Never build annotation specs from a secondary table when the file is available** — print the file's items first, map ordinals against reality, then strike (this converts the #1 bug class into a non-issue).
- **`set -o pipefail` (or no pipes) around annotation drivers** — `cmd | tail` swallowed two failures this session.
- **A "routed" marker kind is missing from the skill's grammar** — until then, pick ONE convention (e.g. `w:routed to …`) and document it in the file or the sweep notes.
- **Surface system anomalies immediately** (full /tmp) instead of silently working around them.
- **Don't self-grade post-fix scores at 10** — report "all findings repaired; gates green" and leave the number to a fresh VERIFY pass.

## f) Up to 50 things we should get done next

Impact-sorted. Effort S <30 min, M 30 min–2 h, L >2 h.

| #  | Task                                                                                                                                    | Impact   | Effort | Category     |
| -- | --------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------ |
| 1  | Push master → confirm 8/8 CI jobs green (all local gates pass; fixes are local-verified only)                                           | 🔴 High  | XS     | Release      |
| 2  | Persist the session's audit verdict map to `docs/research/` before context is lost                                                      | 🔴 High  | S      | Docs         |
| 3  | Continue the annotation sweep over the ~58 remaining reports (batch by month; drivers already exist at `~/.cache/docs-health/`)          | 🟠 Med   | L      | Docs         |
| 4  | Free /tmp (48G tmpfs at 100%) — identify the filler; system-level risk to all tooling                                                   | 🔴 High  | S      | Owner/System |
| 5  | Release v0.10.1 from green master (`[Unreleased]` is full: DepsChanged, strict enums, CI hardening, CSP/SSE fixes)                      | 🔴 High  | S      | Release      |
| 6  | Headless-chromium E2E of the live dashboard (zero console errors, exports, `live/demo`) — the CSP fix is still browser-unproven         | 🔴 High  | M      | QA           |
| 7  | Triage the 3 Dependabot PRs (go-sse/ssetest 0.3.0, astro 7.3.1, html-validate 11.15.0) — blocked on owner Q1                            | 🟠 Med   | S      | Cleanup      |
| 8  | Upgrade CI golangci-lint pin ≥ v2.13; retire `live/fragments.go:181` nolint                                                             | 🟠 Med   | M      | CI           |
| 9  | Investigate newer Datastar for CSP-safe evaluation → drop `'unsafe-eval'`                                                               | 🟠 Med   | M      | Live-dash    |
| 10 | Heading-format annotator variant + `g)`/`d)` sweep of the 10 non-health archived files                                                  | 🟡 Low   | M      | Docs         |
| 11 | Define + document the "routed" marker convention; retro-fit the ~40 `w:routed` markers if the owner prefers a different form            | 🟡 Low   | S      | Docs         |
| 12 | 5% spot-check re-verification of sub-agent verdicts (sample ~250 items) to quantify false-DONE risk                                     | 🟡 Low   | M      | Process      |
| 13 | changelog.mdx `[Unreleased]` mirror check (sync guard only compares version lists)                                                      | 🟡 Low   | XS     | Website      |
| 14 | Add `X-Frame-Options: DENY` next to the frame-ancestors header                                                                          | 🟡 Low   | XS     | Live-dash    |
| 15 | live/ coverage → 90% (79.8% today; fragment error/empty-state tests are the gap)                                                        | 🟠 Med   | L      | Quality      |
| 16 | Investigate the empty `injector.Shutdown()` error in the live fragment fixture                                                          | 🟠 Med   | S      | Bug          |
| 17 | CLI `--input-format` / `--verbose` flags (oldest open TODO, v0.1.0-era)                                                                 | 🟡 Low   | S      | Feature      |
| 18 | Script CHANGELOG → changelog.mdx sync; markdown formatter pass over living docs                                                         | 🟡 Low   | M      | Docs         |
| 19 | Review the AGENTS.md prune diff (58→28.6KB) — confirm nothing load-bearing was cut for future sessions                                  | 🟠 Med   | S      | Docs         |
| 20 | Run docs-health VERIFY on THIS report next session (per the skill's own cadence)                                                        | ⚪ Nice  | S      | Process      |

## g) Questions I cannot answer myself ❓

1. **`/tmp` is at 100% (48G tmpfs) — something outside this session filled it.** Do you want me to investigate and clean it, or will you handle it? (I won't delete unknown files without approval.)
2. **Routed-marker semantics**: when an open idea is transferred to TODO_LIST/ROADMAP, should the source report's item be struck (`w: routed to …`, current convention), or left unstruck so "open" stays visible in place? This affects ~40 markers already placed and the remaining ~58 files.
3. **Should the annotation sweep (f.3) continue now** despite the remaining files having no archive potential, or is the current state (opens routed, archive set complete) sufficient and the rest waits for the next docs-health pass?

---

_Point-in-time snapshot. Re-verify before treating any claim as current truth._
