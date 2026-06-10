# Comprehensive Execution Plan — 2026-06-10 Session 3

**Current State**: 109 tests, 94.3% coverage, 0 lint issues, clean build, pushed to origin.

---

## Pareto Analysis

### What delivers 80% of the value?

The codebase is already excellent — 94% coverage, zero lint, all features working. The remaining 20% effort yields diminishing returns. The HIGHEST-impact items are:

1. **Documentation accuracy** — TODO_LIST has stale items, FEATURES doesn't reflect migration.go
2. **Coverage gaps in new code** — migration.go has 55-75% coverage (new, untested paths)
3. **Real production robustness** — uncovered error paths in export/report functions

### What's the 4% that delivers 64%?

Closing coverage on `migration.go` (55.6% → 90%+) and `computeServiceStatusFromInfo` — these are the newest, least-tested code paths.

### What's the 1% that delivers 51%?

**One thing**: Updating TODO_LIST to reflect reality. Right now it says items are TODO that are DONE. That's the #1 risk — wasted future sessions re-doing completed work.

---

## Task Breakdown — Coarse (15 tasks, 30-100min each)

Sorted by impact × urgency:

| #   | Task                                                                                     | Impact | Effort | Why                        |
| --- | ---------------------------------------------------------------------------------------- | ------ | ------ | -------------------------- |
| 1   | Update TODO_LIST: mark 4 stale items done, add new completed items                       | HIGH   | 15min  | Prevents rework            |
| 2   | Update FEATURES.md: add migration, godoc examples, fuzz test, iterative BFS, single-lock | HIGH   | 15min  | Accurate feature inventory |
| 3   | Update AGENTS.md: reflect current 94.3% coverage, 109 tests, migration.go, new files     | MED    | 15min  | Session context accuracy   |
| 4   | Cover computeServiceStatusFromInfo (55.6% → 100%)                                        | HIGH   | 15min  | New code, low coverage     |
| 5   | Cover countUniqueScopes (75% → 100%)                                                     | MED    | 10min  | Nested scope tree          |
| 6   | Cover inferServiceType (75% → 100%)                                                      | MED    | 10min  | Provider type detection    |
| 7   | Cover mermaidLabelForRef (0% → 100%)                                                     | MED    | 5min   | Last 0% function           |
| 8   | Cover WriteMermaid error path (84% → 100%)                                               | MED    | 10min  | Export robustness          |
| 9   | Cover ExportFilteredToFile error path (87.5% → 100%)                                     | LOW    | 10min  | Error handling             |
| 10  | Cover updateInvocationAggregate (84.6% → 95%+)                                           | MED    | 15min  | Hot path coverage          |
| 11  | Cover RecordHealthCheckWithContext cancel (88.9% → 100%)                                 | LOW    | 10min  | Context cancel path        |
| 12  | Cover enrichCapabilities nil-ref + empty scope (91.7% → 100%)                            | LOW    | 10min  | Edge case                  |
| 13  | Cover ResolveServiceScope ancestor walking (90% → 100%)                                  | LOW    | 10min  | Scope resolution           |
| 14  | Update docs/planning/execution-plan.md to reflect completed items                        | LOW    | 10min  | Planning accuracy          |
| 15  | Verify: full build + test + lint + coverage report                                       | HIGH   | 5min   | Final gate                 |

---

## Task Breakdown — Fine (54 tasks, max 15min each)

### Wave A: Documentation (1% → 51% of value)

| #   | Micro-task                                                        | Est  | File         |
| --- | ----------------------------------------------------------------- | ---- | ------------ |
| A1  | Mark TODO_LIST "Versioned report schema" as done                  | 2min | TODO_LIST.md |
| A2  | Mark TODO_LIST "Document Recorder locking protocol" as done       | 2min | TODO_LIST.md |
| A3  | Mark TODO_LIST "Add runnable godoc examples" as done              | 2min | TODO_LIST.md |
| A4  | Mark TODO_LIST "buildCapabilityMap iterative" as done             | 2min | TODO_LIST.md |
| A5  | Mark TODO_LIST "Single-lock Recorder optimization" as done        | 2min | TODO_LIST.md |
| A6  | Mark TODO_LIST "Fuzz test for HTML template" as done              | 2min | TODO_LIST.md |
| A7  | Add "Schema migration (MigrateReport)" to P1 section or mark done | 2min | TODO_LIST.md |
| A8  | Add completed section entry for session 3 work                    | 3min | TODO_LIST.md |
| A9  | Add "Schema migration" to FEATURES.md DONE table                  | 2min | FEATURES.md  |
| A10 | Add "Godoc examples" to FEATURES.md DONE table                    | 2min | FEATURES.md  |
| A11 | Add "HTML fuzz test" to FEATURES.md DONE table                    | 2min | FEATURES.md  |
| A12 | Add "buildCapabilityMap iterative" to FEATURES.md DONE table      | 2min | FEATURES.md  |
| A13 | Add "Single-lock optimization" to FEATURES.md DONE table          | 2min | FEATURES.md  |
| A14 | Update FEATURES.md PLANNED: remove completed items                | 3min | FEATURES.md  |
| A15 | Update FEATURES.md PARTIALLY DONE: schema migration now done      | 2min | FEATURES.md  |
| A16 | Update AGENTS.md coverage to 94.3%                                | 2min | AGENTS.md    |
| A17 | Update AGENTS.md test count to 109                                | 2min | AGENTS.md    |
| A18 | Add migration.go to AGENTS.md architecture section                | 3min | AGENTS.md    |
| A19 | Add example_test.go and fuzz_test.go to AGENTS.md                 | 2min | AGENTS.md    |
| A20 | Update AGENTS.md locking protocol docs description                | 2min | AGENTS.md    |
| A21 | Remove stale "4 mutexes" references from AGENTS.md                | 3min | AGENTS.md    |

### Wave B: Coverage — migration.go (4% → 64% of value)

| #   | Micro-task                                                      | Est  | Target          |
| --- | --------------------------------------------------------------- | ---- | --------------- |
| B1  | Test computeServiceStatusFromInfo: registered status            | 3min | migration.go:56 |
| B2  | Test computeServiceStatusFromInfo: active status                | 3min | migration.go:56 |
| B3  | Test computeServiceStatusFromInfo: invocation_error status      | 3min | migration.go:56 |
| B4  | Test computeServiceStatusFromInfo: shutdown status              | 3min | migration.go:56 |
| B5  | Test computeServiceStatusFromInfo: shutdown_error status        | 3min | migration.go:56 |
| B6  | Test countUniqueScopes: nested children                         | 3min | migration.go:46 |
| B7  | Test countUniqueScopes: empty tree                              | 2min | migration.go:46 |
| B8  | Test MigrateReport: service with existing status (no overwrite) | 3min | migration.go    |

### Wave C: Coverage — recorder.go hot paths

| #   | Micro-task                                              | Est  | Target          |
| --- | ------------------------------------------------------- | ---- | --------------- |
| C1  | Test inferServiceType: eager provider                   | 3min | recorder.go:141 |
| C2  | Test inferServiceType: transient provider               | 3min | recorder.go:141 |
| C3  | Test inferServiceType: unknown provider                 | 3min | recorder.go:141 |
| C4  | Test mermaidLabelForRef: dependency not in service list | 5min | mermaid.go:81   |
| C5  | Test WriteMermaid: write error mid-line                 | 5min | mermaid.go:12   |
| C6  | Test ExportFilteredToFile: permission denied path       | 3min | plugin.go:143   |
| C7  | Test updateInvocationAggregate: late registration       | 5min | recorder.go:431 |
| C8  | Test updateInvocationAggregate: error + duration        | 5min | recorder.go:431 |
| C9  | Test RecordHealthCheckWithContext: cancelled context    | 5min | plugin.go:176   |
| C10 | Test enrichCapabilities: nil ref skip                   | 5min | recorder.go:153 |
| C11 | Test ResolveServiceScope: child scope ancestor walk     | 5min | recorder.go:904 |
| C12 | Test ResolveServiceScope: not found in any ancestor     | 5min | recorder.go:904 |

### Wave D: Final Verification

| #   | Micro-task                                     | Est  | Target       |
| --- | ---------------------------------------------- | ---- | ------------ |
| D1  | Run go build ./...                             | 1min | full project |
| D2  | Run go test -timeout 60s -count=1 -cover ./... | 2min | full project |
| D3  | Run golangci-lint run                          | 2min | full project |
| D4  | Check coverage report for remaining gaps       | 3min | coverage     |
| D5  | Verify no regressions in example_test.go       | 2min | examples     |
| D6  | Update docs/planning/execution-plan.md status  | 5min | planning     |

---

## Mermaid Execution Graph

```mermaid
graph TD
    START((Start)) --> A[WAVE A: Documentation<br/>21 tasks, ~45min]
    A --> A_DONE{Docs accurate?}
    A_DONE -->|Yes| B[WAVE B: Migration Coverage<br/>8 tasks, ~23min]
    B --> B_DONE{Migration ≥90%?}
    B_DONE -->|Yes| C[WAVE C: Hot Path Coverage<br/>12 tasks, ~55min]
    C --> C_DONE{All functions ≥90%?}
    C_DONE -->|Yes| D[WAVE D: Final Verification<br/>6 tasks, ~15min]
    D --> D_DONE{Build + Test + Lint<br/>all green?}
    D_DONE -->|Yes| DONE((Done ✅)])

    A_DONE -->|No| A_FIX[Fix docs]
    A_FIX --> A
    B_DONE -->|No| B_FIX[Add more tests]
    B_FIX --> B
    C_DONE -->|No| C_FIX[Add more tests]
    C_FIX --> C
    D_DONE -->|No| D_FIX[Fix issues]
    D_FIX --> D

    style START fill:#4CAF50,color:white
    style DONE fill:#4CAF50,color:white
    style A fill:#2196F3,color:white
    style B fill:#FF9800,color:white
    style C fill:#9C27B0,color:white
    style D fill:#F44336,color:white
```

---

## Effort Summary

| Wave                  | Tasks  | Est. Time | Cumulative Coverage Gain        |
| --------------------- | ------ | --------- | ------------------------------- |
| A: Docs               | 21     | 45min     | 0% (documentation only)         |
| B: Migration coverage | 8      | 23min     | +2-3% (migration.go → 90%+)     |
| C: Hot path coverage  | 12     | 55min     | +1-2% (recorder/mermaid/plugin) |
| D: Verification       | 6      | 15min     | — (gate)                        |
| **Total**             | **47** | **~2.5h** | **94.3% → ~97%**                |

## Items Explicitly NOT Included

| Item                                | Why                                         |
| ----------------------------------- | ------------------------------------------- |
| PlantUML export                     | Deferred until users request it             |
| Prometheus/OTel                     | Out of scope per AGENTS.md                  |
| Goroutine-local stacks              | Requires samber/do API changes              |
| `interface{}` → `any` in test files | Test-only, no production impact             |
| `err113` in test files              | Test-only, intentional for error simulation |
