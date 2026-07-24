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

Identified during cross-project review. These patterns exist in the sibling project [`go-workflow-auditlog`](https://github.com/larsartmann/go-workflow-auditlog) and would improve this project.

- [ ] **Adopt go-error-family classification** — Port the `classify.go` pattern: register sentinel errors into families for automatic error classification (e.g., `IsRetryable`, `IsTransient`). This project has `go-error-family` only as a transitive v0.8.0 dependency.
- [ ] **Adopt go-atomic-write** — Replace the custom `writeToFile()` helper in `plugin.go` with the shared `go-atomic-write` library for crash-safe file writes. The sibling project uses this for all file I/O.
- [ ] **Add NDJSON streaming** — Port the `NDJSONStreamer` pattern from `go-workflow-auditlog/stream.go`: configurable auto-flush and buffer size for streaming events to a file or writer as they occur, instead of batch export.
- [ ] **Add diagram direction option** — Add `WithDirection(output.Direction)` across all 4 diagram formats (Mermaid, PlantUML, DOT, D2). Currently rankdir/direction is hardcoded (DOT=LR, Mermaid=TD).
- [ ] **Add table column selection** — Add `WithColumns(TableColumn...)` option to `WriteTable` for selectable columns (currently fixed to 7 columns). The sibling project supports 10 columns.
