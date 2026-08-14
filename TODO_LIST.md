# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).
Last updated: 2026-08-07

---

## Publishing & Release

- [ ] **Tag and release v0.7.0** — All [Unreleased] items in CHANGELOG.md are complete. Verify the full test/lint/coverage suite passes, then tag and release.

---

## Cross-Project Feature Gaps (from go-workflow-auditlog)

Identified during cross-project review. These patterns existed in the sibling project [`go-workflow-auditlog`](https://github.com/larsartmann/go-workflow-auditlog) and have now been ported.

- [x] **Adopt go-error-family classification** — Ported the `classify.go` pattern: all sentinel errors are registered into families (Corruption/Rejection) via `ErrorClassifications()` and auto-registered in `init()`. Upgraded `go-error-family` from v0.8.0 to v0.9.0 (direct dependency).
- [x] **Adopt go-atomic-write** — Replaced the custom `writeToFile()` helper in `plugin.go` with `go-atomic-write` v0.3.0 (`atomicwrite.WriteFunc`) for crash-safe file writes with fsync durability and cross-platform atomic rename.
- [x] **Add NDJSON streaming** — Ported the `NDJSONStreamer` pattern from `go-workflow-auditlog/stream.go`: configurable auto-flush and buffer size for streaming events to a file or writer as they occur. Uses standard `encoding/json` (no `jsontext` dependency, respecting the project's `encoding/json/v2` exclusion policy).
- [x] **Add diagram direction option** — Added `WithDirection(output.Direction)` across all 4 diagram formats (Mermaid, PlantUML, DOT, D2).
- [x] **Add table column selection** — Added `WithColumns(TableColumn...)` option to `WriteTable` for selectable columns (10 available). The sibling project's pattern was fully ported.
- [x] **Port MultiWriter** — Event fan-out to multiple `OnEvent` callbacks (`multi_writer.go`).
- [x] **Port StreamEvents** — Callback-based NDJSON reader without full materialization (`ndjson.go`).
- [x] **Port WithFlushInterval** — Time-based flush option on `NDJSONStreamer` (`stream.go`).
- [x] **Port SSE ring buffer replay** — In-memory ring buffer for reconnection replay (`live/hub.go`, `live/server.go`).
- [x] **Port RunID concept** — 128-bit hex branded type for cross-system correlation (`runid.go`). Breaking: `NewRecorder`/`NewReport` signatures changed.
- [x] **Add Write\*String methods** — Mermaid, PlantUML, DOT, D2, HTML convenience methods returning `string`.
- [x] **Add timing deltas to Diff** — `TotalBuildDurationMsDelta` and `TotalShutdownDurationMsDelta` in `DiffResult`.
- [x] **Export testhelpers** — Moved from `internal/testhelpers/` to public `testhelpers/`.
- [x] **Add dependabot** — Automated dependency updates for gomod, github-actions, pnpm.
- [x] **Add goreleaser + RELEASE.md** — Release automation for CLI binary.
- [x] **Bump schema version** — `0.2.0` → `0.3.0` (RunID field added to Event/Report).
