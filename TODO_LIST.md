# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).
Last updated: 2026-07-24

---

## Publishing & Release

- [ ] **Tag and release v0.7.0** — All [Unreleased] items in CHANGELOG.md are complete. Two `replace` directives for `go-output/testhelpers` (release-bug workaround) remain — these are technical debt, not a release blocker, since they redirect broken pseudo-versions to real published tags. Verify the full test/lint/coverage suite passes, then tag and release.

---

## Technical Debt

- [ ] **Remove go-output testhelpers replace directives** — Two `replace` directives in `go.mod` (`go-output/testhelpers` and `go-output/testhelpers/graphtest`) redirect broken pseudo-versions (`v0.0.0-00010101000000-000000000000`) to real tags (`v0.31.1`). Remove them — and the `replace-allow-list` entry in `.golangci.yml` — once go-output fixes their release process (strip local `replace` directives before tagging).

---

## Cross-Project Feature Gaps (from go-workflow-auditlog)

Identified during cross-project review. These patterns existed in the sibling project [`go-workflow-auditlog`](https://github.com/larsartmann/go-workflow-auditlog) and have now been ported.

- [x] **Adopt go-error-family classification** — Ported the `classify.go` pattern: all sentinel errors are registered into families (Corruption/Rejection) via `ErrorClassifications()` and auto-registered in `init()`. Upgraded `go-error-family` from v0.8.0 to v0.9.0 (direct dependency).
- [x] **Adopt go-atomic-write** — Replaced the custom `writeToFile()` helper in `plugin.go` with `go-atomic-write` v0.3.0 (`atomicwrite.WriteFunc`) for crash-safe file writes with fsync durability and cross-platform atomic rename.
- [x] **Add NDJSON streaming** — Ported the `NDJSONStreamer` pattern from `go-workflow-auditlog/stream.go`: configurable auto-flush and buffer size for streaming events to a file or writer as they occur. Uses standard `encoding/json` (no `jsontext` dependency, respecting the project's `encoding/json/v2` exclusion policy).
- [x] **Add diagram direction option** — Added `WithDirection(output.Direction)` across all 4 diagram formats (Mermaid, PlantUML, DOT, D2).
- [x] **Add table column selection** — Added `WithColumns(TableColumn...)` option to `WriteTable` for selectable columns (10 available). The sibling project's pattern was fully ported.
