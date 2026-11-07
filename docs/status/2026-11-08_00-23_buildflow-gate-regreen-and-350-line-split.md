# Status Report — BuildFlow Gate Regreen, 350-Line Split & Docs-Health Marathon

> **Live remainder:** none at write time — open §f items below are harvested into TODO_LIST.md/ROADMAP.md in the same pass.

## Scope Line

One session, started 2026-11-07 ~20:08 CET on a red `buildflow --fix --build-mode=full` run (exit 69, 24 error findings: file-size-check 20, docs-gates 4). Session ended 2026-11-08 00:23 CET with the full pipeline green (exit 0) and all five docs gates PASS. A second session was live in this checkout throughout (daemon commits 20:06/20:12, BuildFlow HEAD moved twice underneath the pinned binary).

## a) FULLY DONE

1. **Gate-failure diagnosis.** Ruled out the stale-binary theory by querying the BuildFlow state DB (`step_outputs`): file-size findings flipped warning → error between 2026-11-04 and 2026-11-05 runs, docs-gates errors appear from 2026-11-05, and every run since (including the concurrent session's, 3 minutes before mine) carried the same 24 findings. Fleet query showed every active Go repo carrying error-severity file-size findings (go-cqrs-lite 231, dnsblockd 121, this repo 20 — among the mildest). Conclusion: the 350-line bar is real fleet policy; the findings were real.
2. **file-size-check: all 20 oversized files split** into ~40 focused files, every hand-written `.go` file now ≤344 lines. Production: hooks.go split (hook methods + event_builders.go), plugin.go (+plugin_export.go), report.go (+report_query.go), report_builder.go (+scope_tree.go), live/server.go (+server_handlers.go), live/fragments.go (+fragments_format.go), forward/forward.go (+target.go, deliver.go). Tests split by concern (replay_test.go → 6 files, live/server_test.go → 6, fragments tests → 5, forward tests → 3, plus diagram/d2, migration, tree-table, stream, cli-flags, fuzz, helpers/fixtures/assertions). `go vet`, `go test ./...`, and `go test -race ./...` all green; coverage gate ran inside the full buildflow pass.
3. **docs-gates: all five gates PASS.** 102 unharvested §f items across 27 reports resolved: verified against code and annotated done (e.g. ServiceInfo split, README sections, guides shipping, CI actionlint job, coverage flake app), harvested into TODO_LIST.md/ROADMAP.md, or closed Won't-implement/moot with reasons. 12 rule-F inline-code spacing fixes, 2 un-verdicted strikes given markers from the fixed vocabulary.
4. **Conflict corruption repaired.** The 2026-11-04 report carried unresolved stash-conflict markers (`<<<<<<< Updated upstream` / `>>>>>>> Stashed changes`) around duplicated §b/§d items and a fully duplicated `>>>>>>>`-prefixed §f table — committed in that state at 20:06. Reconstructed: markers removed, duplicate copies dropped, the one clean copy of every item kept, off-by-one loss restored.
5. **Archive ritual.** 63 fully-resolved reports moved to docs/status/archived/ with `> **Live remainder:**` banners; 3 recent reports kept unarchived. Cadence gate: ARCHIVE-HEALTHY.
6. **Small findings closed.** SECURITY.md response-targets wording now matches the securitymd pattern grammar ("Initial response"); shellcheck SC2034 fixed (`for _ in`); golangci-lint install URL updated to the live docs path; new lychee.toml with intent-encoded excludes (video-production captures, archived corpus, docs/research font-400 artifacts) — lychee 0 findings; `tool_options.go-version-auto-configure.respect_patch_floor: true` added to .buildflow.yml.
7. **vendorHash repaired, flake validated.** `nix flake check` exposed the real hash drift (go-mod-update had rewritten go.sum during the original failing run); hand-applied the `got:` hash (nix-hash-fix is defectively not-applicable on this Go+Nix repo). Second `nix flake check`: all checks passed. The deferred "Run `nix flake check`" report item annotated with that evidence.
8. **Concurrent-session lesson recorded and committed** in the crush-config repo (4be72f5): inventory foreign commits before diagnosing aggregate-gate failures; exact-match read-modify-write for shared historical docs.
9. **Living docs updated in the same pass.** AGENTS.md (file-size bar, docs-gates grammar, known-tool non-fixes incl. the pnpm-baseline offline artifact, manual vendorHash repair, helpers split map, tool_options + gvac advisory note), docs/architecture.md source map (9 entries rewritten for the new layout), TODO_LIST.md (5 harvest rows + re-verified date), ROADMAP.md (harvested website/docs ideas section). Claims linter 12/12 green.
10. **Final verification.** Full `buildflow --fix --build-mode=full` re-run: exit 0, "BuildFlow passed with warnings" — every remaining finding is below the error gate and accounted for (see b/c/e).

## b) PARTIALLY DONE

1. **go-version-auto-configure: 9 warnings remain, by upstream design.** The added `respect_patch_floor` option follows the fleet #232 configuration, but gvac's `ci-pin-patch-form` rule is documented detect-only/suggestion ("NO (suggestion)" in the gvac rule×pin matrix), so it keeps flagging the deliberate CI patch pin (1.27.3, required for GO-2026-6809). Config is correct; the advisory is permanent until policy changes.
2. **Final verification ran under the stale binary.** The green full run used buildflow e35e902 while its repo HEAD moved 940aeae → 6d42bc9 during the session. Source diff for the file-size/docs logic between build and HEAD was empty, so results should carry over — but a fresh-binary re-run is still owed.
3. **`buildflow format` had 2 failed steps I never identified.** Both format invocations ended "2 step(s) failed"; the log noise shows an ignored shellcheck SQLite-busy message and go-licenses "no license found" lines, but the failing step names were never drilled into. Formatting itself landed (imports fixed, build green).
4. **Formatter-fingerprint-drift preflight warning persists** even after a format run — likely a consequence of the 2 failed steps or cached verdict state; not chased.
5. **TODO_LIST growth.** Five harvest rows plus three new rows added today keep every §f item matched, but the list is heavier; next re-verify pass should prune.

## c) NOT STARTED

1. Upstream filings: BuildFlow nix-steps not-applicable defect (Open Question 8, needs a valid `gh` token), gvac `ci-pin-patch-form` policy question, dependabot-auto-configure pnpm ecosystem gap (orphan false positive on a correct `/website` entry — confirmed at source level, `ecosystemDirs` has no pnpm case).
2. On-demand tools never run: gitleaks, codespell, markdown-lint (config-skipped by policy; still owed a run).
3. CHANGELOG entry for the maintenance wave (split + gate regreen + vendorHash) — docs coverage chosen instead; an Unreleased-section policy decision is open.
4. `nix flake check --all-systems` (aarch64-darwin/aarch64-linux omitted by the default run).
5. Deep look at the step-timing regressions (all read as cold-cache/network noise; no `buildflow timings --regressions` analysis done).
6. Extract-zip upstream watch (2.0.1 still latest, no fixed release; baseline accepts stay).

## d) TOTALLY FUCKED UP

1. **The AGENTS.md multiedit replaced three bullets instead of adding alongside them** — the pre-commit hook, auto-commit daemon, and version-skew ledger entries were momentarily destroyed by my own edit. Caught on read-back and all three restored, but the mistake class (replace-the-nearest-bullet instead of insert) is exactly what the memory-update rules warn about.
2. **The file splitter took three iterations and damaged three files on the way.** First run: `end=None` meant "stop at next function", mangling report.go, live/server.go, and live/fragments.go into 6–30-line fragment moves; a job crashed mid-table because an earlier job had already moved the block a later `end` regex pointed at. Second run re-moved the same fragments before I noticed. Repairs + append-mode moves + fixed semantics converged, and the build/tests prove the end state — but the path was sloppy for a session whose theme was exactness.
3. **Off-by-one in the conflict reconstruction deleted the clean §b item 3** from the 2026-11-04 report; caught by seam inspection and re-inserted verbatim.
4. **`force_pkg` strings missing the `package ` prefix** produced two test files with invalid package lines (`main`, `forward_test` as bare words) — the build caught it.
5. **The `rg -r` trap hit three times** — the host hook blocked the same replace-flag misread three separate times. A known pattern on this machine; should be muscle memory by now.
6. **Misread my own offset-past-EOF view of SECURITY.md as concurrent-session interference** and spent a detour on git provenance before re-reading. The file had shrunk by exactly my own view window.
7. **Observed, not mine: the concurrent session committed the corrupted 2026-11-04 report** (conflict markers and all) at 20:06 while the docs gates it feeds were red — the collision this repo's Open Question 5/49 keeps predicting.

## e) WHAT WE SHOULD IMPROVE

1. **Check run history before diagnosing.** The `step_outputs` DB query was the decisive tool and it came after several speculative reads. First move for any findings-gate failure: `buildflow history` + the DB, then fresh `-s <step> --format finding`.
2. **One split → build → test cycle before batching.** A single-file pilot would have exposed the `end=None` semantics bug before it mangled three files.
3. **Additive AGENTS.md edits.** New bullets get inserted; existing bullets get edited in place — never replaced by a near-looking new one.
4. **Read the owning repo's AGENTS.md first.** BuildFlow's own AGENTS.md line 196 names `respect_patch_floor` and the gvac rule×pin matrix explicitly; I re-derived it from source.
5. **Close format-step failures before moving on** — "2 steps failed" is a loose end that compounds.
6. **Shared-checkout discipline worked and should stay:** exact-match edits with loud skip-on-drift prevented collisions with the live second session; the daemon's 78-file commit absorbed the split work cleanly.

## f) 50 Things To Get Done Next

Harvested into TODO_LIST.md/ROADMAP.md in this pass; tokens match the living docs so the freshness replay sees them.

| #       | Task                                                                                                             | Impact | Effort | Category      |
| ------- | ---------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1       | Migrate the live dashboard to Datastar ≥ v1.0.4 CSP mode (nonce directive, drop `unsafe-eval`)                    | High   | M      | Bug           |
| 2       | Add direct unit tests for `drainEvents` (burst coalescing + channel-close) and the `makeReportJSON` error path    | Medium | S      | Quality       |
| 3       | Benchmark `renderAllFragments`, add an SSE end-to-end broadcast benchmark, Mermaid/PlantUML/DOT companions        | Medium | M      | Quality       |
| 4       | Add an `ErrExportWriteFailed` sentinel wrapping go-atomic-write errors                                            | Low    | S      | Quality       |
| 5       | CLI color flags: `--no-color`, respect `NO_COLOR`, honor `FORCE_COLOR`                                            | Low    | S      | DevEx         |
| 6       | Decide the concurrent-session protocol: file-ownership map or per-session worktrees                               | High   | M      | Process       |
| 7       | live/ coverage owner decision: exclude `live/fragments_templ.go` from the gate or accept 84%                      | Medium | S      | Quality       |
| 8       | Push master and confirm all CI jobs green (owner approval pending)                                                | High   | S      | CI            |
| 9       | Register the SSH commit-signing key; enable branch protection, Dependabot auto-merge, failure notifications        | Medium | M      | CI            |
| 10      | Ratify the auto-commit daemon policy (pause before bulk work?)                                                    | Medium | S      | Process       |
| 11      | Decide CI docs-only `paths-ignore` for the website workflow                                                       | Low    | S      | CI            |
| 12      | Markdown formatter pass over the living docs (ragged table padding)                                               | Low    | M      | Documentation |
| 13      | Rebuild and reinstall the `buildflow` binary; verify PATH precedence; re-run the battery fresh                     | High   | S      | Tooling       |
| 14      | File the BuildFlow nix-steps not-applicable defect (needs a valid `gh` token)                                      | Medium | S      | Upstream      |
| 15      | Online re-run of `pnpm-audit` + `govulncheck` to validate the extract-zip baseline accepts                         | Medium | S      | Security      |
| 16      | Add `meta.description` to the three flake apps (only remaining `nix flake check` warnings)                         | Low    | S      | DevEx         |
| 17      | Run the on-demand tools: gitleaks, codespell, markdown-lint                                                        | Medium | S      | Security      |
| 18      | Drill into the two failed `buildflow format` steps with verbose output                                             | Medium | S      | Tooling       |
| 19      | Dedicated `diff` docs page                                                                                         | Low    | M      | Documentation |
| 20      | Dedicated `replay` docs page                                                                                       | Low    | M      | Documentation |
| 21      | Dedicated `scope` tree docs page                                                                                   | Low    | M      | Documentation |
| 22      | JSON `schema` validation guide page                                                                                | Medium | M      | Documentation |
| 23      | `ResolveServiceScope` advanced health-check pattern in the docs                                                    | Low    | S      | Documentation |
| 24      | Dependency detail popover on the live dashboard graph                                                              | Low    | M      | Feature       |
| 25      | Design a more distinctive favicon/logo                                                                             | Low    | M      | Website       |
| 26      | Lighthouse CI workflow (currently run by hand per change)                                                          | Low    | S      | CI            |
| 27      | `md-go-validator` for docs code-block validation                                                                   | Low    | M      | Documentation |
| 28      | Stale-reference check in the spirit of gogenfilter's deleted-files check                                           | Low    | M      | Documentation |
| 29      | Docs-site version selector (when v1.0.0 lands)                                                                     | Low    | S      | Website       |
| 30      | "Star this repo" call-to-action on the docs site                                                                   | Low    | S      | Website       |
| 31      | Social-proof "Used by" section for the docs site                                                                           | Low    | S      | Website       |
| 32      | Sponsors section (when there are sponsors)                                                                         | Low    | S      | Website       |
| 33      | Uptime monitoring for do-auditlog.lars.software                                                                    | Low    | S      | Ops           |
| 34      | Sitemap verification                                                                                               | Low    | S      | Ops           |
| 35      | SOPS or age-encrypted tfvars for Namecheap credentials                                                             | Low    | M      | Ops           |
| 36      | `terraform plan` CI check (needs credentials in CI)                                                                | Low    | M      | Ops           |
| 37      | Open Graph tags for shared dashboard URLs                                                                        | Low    | S      | Website       |
| 38      | Virtual scrolling for 500+ service reports                                                                         | Low    | L      | Feature       |
| 39      | WebP screenshot conversion                                                                                         | Low    | S      | Website       |
| 40      | Coordinated tab slide+fade animation orchestration                                                                 | Low    | M      | Website       |
| 41      | Identify the templ watcher/LSP behind the 02:15:44 mass-touch; decide system-templ fleet policy                    | Medium | M      | Bug           |
| 42      | Golden test verification in a clean clone; `nix flake check --all-systems` for aarch64                             | Medium | M      | Quality       |
| 43      | Watch extract-zip upstream for a fixed release, then drop the two baseline accepts                                 | Medium | S      | Security      |
| 44      | Investigate the go-licenses "no license found" noise on fleet pseudo-versions                                      | Low    | S      | Tooling       |
| 45      | Decide an Unreleased CHANGELOG policy for maintenance-only waves                                                   | Low    | S      | Process       |

## g) Questions I Cannot Answer Myself

1. **May master be pushed** so CI confirms the split + gate-regreen wave (the 78-file daemon blob plus the follow-ups)? All gates are verified locally; push needs your approval per the standing Q3.
2. **Rebuild the `buildflow` binary now from the shared checkout** (HEAD 6d42bc9 carries a concurrent session's same-day commits), or wait until that session lands? Rebuilding mid-flight risks adopting someone else's unfinished tool state; staying stale keeps the advisory alive.
3. **`ci-pin-patch-form` policy:** leave the 9 permanent advisories as-is, silence them via baseline accepts, or file upstream gvac proposing that the rule honor `respect_patch_floor` for repos with govulncheck-required patch pins?
