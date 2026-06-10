# Execution Plan — samber-do-auditlog TODO List

Every task is designed to take ≤12 minutes. Sorted by impact (customer value + risk reduction).
All tasks are verified with: `go build ./... && go test -timeout 60s -count=1 ./... && golangci-lint run`

---

## Task Table

| #   | Task                                                            | Source       | Impact                            | Effort | Category    |
| --- | --------------------------------------------------------------- | ------------ | --------------------------------- | ------ | ----------- |
| 1   | Document Recorder locking protocol (doc comment)                | P2 TODO      | HIGH — prevents deadlocks         | 10min  | Safety      |
| 2   | Cover `mermaidLabelForRef` (0% → 100%)                          | Coverage gap | MED — last 0% function in lib     | 5min   | Quality     |
| 3   | Cover `WriteMermaid` error paths (80% → 100%)                   | Coverage gap | MED — error handling confidence   | 8min   | Quality     |
| 4   | Cover `ExportFilteredToFile` error path (87.5% → 100%)          | Coverage gap | MED — export robustness           | 8min   | Quality     |
| 5   | Cover `RecordHealthCheckWithContext` cancel path (88.9% → 100%) | Coverage gap | MED — ctx cancellation            | 8min   | Quality     |
| 6   | Cover `recordInvocationResult` edge cases (88.9% → 100%)        | Coverage gap | MED — invocation tracking         | 10min  | Quality     |
| 7   | Cover `enrichCapabilities` empty-scope path (91.7% → 100%)      | Coverage gap | LOW — edge case                   | 8min   | Quality     |
| 8   | Cover `ResolveServiceScope` error paths (90% → 100%)            | Coverage gap | LOW — scope resolution            | 8min   | Quality     |
| 9   | Cover `matchEvent` nil-timefilter path (90.9% → 100%)           | Coverage gap | LOW — filter logic                | 5min   | Quality     |
| 10  | Cover `buildScopeTreeLocked` edge cases (95.8% → 100%)          | Coverage gap | LOW — tree building               | 10min  | Quality     |
| 11  | Add godoc Example for New() + Opts()                            | P2 TODO      | HIGH — pkg.go.dev visibility      | 10min  | Docs        |
| 12  | Add godoc Example for Report() + ServiceByName()                | P2 TODO      | MED — API discoverability         | 8min   | Docs        |
| 13  | Add godoc Example for ExportToFile + ExportToHTML               | P2 TODO      | MED — usage discoverability       | 8min   | Docs        |
| 14  | Add godoc Example for Report.Filtered + WithServicesByName      | P2 TODO      | MED — new feature discoverability | 8min   | Docs        |
| 15  | Add godoc Example for RecordHealthCheck                         | P2 TODO      | MED — health check API            | 8min   | Docs        |
| 16  | Add godoc Example for WriteMermaid                              | P2 TODO      | LOW — Mermaid usage               | 5min   | Docs        |
| 17  | Refactor buildCapabilityMap to iterative                        | P2 TODO      | MED — removes recursion risk      | 10min  | Code health |
| 18  | Validate Config.Enabled explicitly in New()                     | Code health  | MED — defensive programming       | 5min   | Code health |
| 19  | Add Config.Validate() actual checks                             | Code health  | LOW — Validate() is a no-op       | 10min  | Code health |
| 20  | Fuzz test for HTML template                                     | P1 TODO      | LOW — templ provides escaping     | 12min  | Security    |
| 21  | Single-lock Recorder optimization — design phase                | P3 TODO      | HIGH perf but HIGH risk           | 12min  | Performance |
| 22  | Single-lock Recorder — refactor OnBeforeInvocation              | P3 TODO      | HIGH perf                         | 12min  | Performance |
| 23  | Single-lock Recorder — refactor OnAfterInvocation               | P3 TODO      | HIGH perf                         | 10min  | Performance |
| 24  | Single-lock Recorder — refactor shutdown hooks                  | P3 TODO      | HIGH perf                         | 10min  | Performance |
| 25  | Single-lock Recorder — remove dead mutexes                      | P3 TODO      | HIGH perf                         | 8min   | Performance |
| 26  | Single-lock Recorder — benchmark comparison                     | P3 TODO      | HIGH — verify perf gain           | 10min  | Performance |
| 27  | Versioned report schema — define migration interface            | P1 TODO      | MED — forward compat              | 10min  | Feature     |
| 28  | Versioned report schema — implement v0.1.0 → v0.2.0             | P1 TODO      | MED — actual migration            | 12min  | Feature     |
| 29  | Versioned report schema — test round-trip                       | P1 TODO      | MED — correctness                 | 8min   | Feature     |

---

## Execution Order (sorted by impact × urgency)

### Wave 1: Safety + Quality (95% → 98%+ coverage, deadlock prevention)

1. **Task 1** — Document locking protocol ← prevents the #1 production risk
2. **Task 2** — Cover mermaidLabelForRef ← trivial, closes last 0%
3. **Task 3** — Cover WriteMermaid error paths
4. **Task 4** — Cover ExportFilteredToFile error path
5. **Task 5** — Cover RecordHealthCheckWithContext cancel
6. **Task 6** — Cover recordInvocationResult edge cases
7. **Task 7** — Cover enrichCapabilities empty-scope
8. **Task 8** — Cover ResolveServiceScope error paths
9. **Task 9** — Cover matchEvent nil-timefilter
10. **Task 10** — Cover buildScopeTreeLocked edge cases

### Wave 2: Discoverability (godoc examples for pkg.go.dev)

11. **Task 11** — Example: New() + Opts()
12. **Task 12** — Example: Report() + ServiceByName()
13. **Task 13** — Example: ExportToFile + ExportToHTML
14. **Task 14** — Example: Report.Filtered + options
15. **Task 15** — Example: RecordHealthCheck
16. **Task 16** — Example: WriteMermaid

### Wave 3: Code Health

17. **Task 17** — buildCapabilityMap iterative refactor
18. **Task 18** — Validate Config.Enabled in New()
19. **Task 19** — Config.Validate() real checks
20. **Task 20** — Fuzz test for HTML template

### Wave 4: Performance (Single-lock optimization)

21. **Task 21** — Design single-lock protocol
22. **Task 22** — Refactor OnBeforeInvocation
23. **Task 23** — Refactor OnAfterInvocation
24. **Task 24** — Refactor shutdown hooks
25. **Task 25** — Remove dead mutexes
26. **Task 26** — Benchmark comparison

### Wave 5: Features

27. **Task 27** — Schema migration interface
28. **Task 28** — Implement v0.1.0 → v0.2.0 migration
29. **Task 29** — Test round-trip

---

## Summary Statistics

| Category              | Tasks  | Total Est. |
| --------------------- | ------ | ---------- |
| Safety (locking docs) | 1      | 10min      |
| Coverage (quality)    | 9      | 70min      |
| Godoc examples        | 6      | 47min      |
| Code health           | 3      | 25min      |
| Security (fuzz)       | 1      | 12min      |
| Performance           | 6      | 64min      |
| Features (schema)     | 3      | 30min      |
| **Total**             | **29** | **~4h**    |

## PlantUML — Skipped

Not included. Mermaid export exists. PlantUML only if users request it (per TODO_LIST).
