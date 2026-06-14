# Comprehensive Execution Plan — Post Review Session

**Created**: 2026-06-14 13:01
**Status**: Planning phase — awaiting approval
**Scope**: ALL remaining TODOs from status report, self-review, and code review

---

## Pareto Breakdown

### 1% → 51% impact (Critical correctness + security)

Wire Config.Validate() into New(), harden CSP, fix keyboard nav.

### 4% → 64% impact (Architecture + docs)

Go enum metadata injection, Report.Validate(), update all project docs.

### 20% → 80% impact (Polish + completeness)

Empty states, debounce, diagram themes, fuzz test robustness, HTML integration test.

### Remaining (Nice-to-have)

Virtual scrolling, touch events, CI security, archived doc cleanup.

---

## Fine-Grained Task List (all ≤12 min)

Sorted by: Impact × Customer-Value ÷ Effort (descending).

| #   | Task                                                                                                 | Files                                               | Effort | Impact   | Tier      |
| --- | ---------------------------------------------------------------------------------------------------- | --------------------------------------------------- | ------ | -------- | --------- |
| 1   | Wire Config.Validate() into New() — change signature to `(*Plugin, error)`                           | `plugin.go`                                         | 10m    | CRITICAL | 1%        |
| 2   | Update helpers_test.go: newPluginAndInjector, newPluginAndInjectorWithID, newPluginWithCapture       | `helpers_test.go`                                   | 10m    | CRITICAL | 1%        |
| 3   | Update plugin_basic_test.go: all 6 test callers                                                      | `plugin_basic_test.go`                              | 12m    | CRITICAL | 1%        |
| 4   | Update plugin_lifecycle + plugin_errors_test.go: all 11 callers                                      | `plugin_lifecycle_test.go`, `plugin_errors_test.go` | 12m    | CRITICAL | 1%        |
| 5   | Update plugin_export + plugin_html + plugin_scope + plugin_provider_test.go: all 22 callers          | 4 files                                             | 12m    | CRITICAL | 1%        |
| 6   | Update healthcheck_basic + healthcheck_export + report_query + report_filter_test.go: all 20 callers | 4 files                                             | 12m    | CRITICAL | 1%        |
| 7   | Update type_method + extra + diagram + migration + benchmarks + fuzz_test.go: all 18 callers         | 6 files                                             | 12m    | CRITICAL | 1%        |
| 8   | Update example_test.go: 7 Example callers                                                            | `example_test.go`                                   | 10m    | CRITICAL | 1%        |
| 9   | Update example/main.go: setupPlugin caller                                                           | `example/main.go`                                   | 5m     | CRITICAL | 1%        |
| 10  | Run go test + lint + vet — verify all callers updated                                                | —                                                   | 5m     | CRITICAL | 1%        |
| 11  | Add CSP `base-uri 'none'; frame-ancestors 'none'` to HTML meta tag                                   | `html.templ`                                        | 5m     | HIGH     | 1%        |
| 12  | Add keyboard nav exclusion for TEXTAREA/SELECT/BUTTON (not just INPUT)                               | `html.templ`                                        | 8m     | MEDIUM   | 1%        |
| 13  | Regenerate html_templ.go + test + lint                                                               | `html_templ.go`                                     | 5m     | HIGH     | 1%        |
| 14  | Add Report.Validate() method — check counts match actual slices                                      | `report.go`                                         | 10m    | HIGH     | 4%        |
| 15  | Add TestReport_Validate for Report.Validate()                                                        | `report_query_test.go`                              | 10m    | HIGH     | 4%        |
| 16  | Update CHANGELOG.md — add all session work under [Unreleased]                                        | `CHANGELOG.md`                                      | 10m    | HIGH     | 4%        |
| 17  | Update TODO_LIST.md — mark session items, add new TODOs                                              | `TODO_LIST.md`                                      | 10m    | HIGH     | 4%        |
| 18  | Update FEATURES.md part 1 — diagram, accessibility, test quality                                     | `FEATURES.md`                                       | 12m    | HIGH     | 4%        |
| 19  | Update FEATURES.md part 2 — security hardening, Config.Validate                                      | `FEATURES.md`                                       | 12m    | HIGH     | 4%        |
| 20  | Update README.md part 1 — add Mermaid/PlantUML API docs                                              | `README.md`                                         | 12m    | HIGH     | 4%        |
| 21  | Update README.md part 2 — add Report filtering + query method docs                                   | `README.md`                                         | 12m    | HIGH     | 4%        |
| 22  | Update README.md part 3 — add health check auditing docs                                             | `README.md`                                         | 10m    | MEDIUM   | 4%        |
| 23  | Design TypeMetadata struct in Go (icons, labels, colors per enum)                                    | `types.go`                                          | 12m    | HIGH     | 4%        |
| 24  | Add Report.TypeMetadata() method that builds metadata from Go enums                                  | `report_builder.go`                                 | 12m    | HIGH     | 4%        |
| 25  | Inject metadata JSON into templ template via @templ.JSONScript                                       | `html.templ`                                        | 12m    | HIGH     | 4%        |
| 26  | Replace hardcoded typeIcons JS object with metadata-driven lookup                                    | `html.templ`                                        | 12m    | HIGH     | 4%        |
| 27  | Replace hardcoded statusIcons/typeLabels/eventLabels/colors with metadata                            | `html.templ`                                        | 12m    | HIGH     | 4%        |
| 28  | Regenerate html_templ.go + test metadata injection works                                             | `html_templ.go`                                     | 10m    | HIGH     | 4%        |
| 29  | Add aria-pressed="false" to event filter chip buttons + toggle in JS                                 | `html.templ`                                        | 8m     | MEDIUM   | 20%       |
| 30  | Add scope="col" to all services table `<th>` elements                                                | `html.templ`                                        | 5m     | MEDIUM   | 20%       |
| 31  | Add scope="col" to all events table `<th>` elements                                                  | `html.templ`                                        | 5m     | MEDIUM   | 20%       |
| 32  | Add empty-state message divs for services/events/scopes tables                                       | `html.templ`                                        | 10m    | MEDIUM   | 20%       |
| 33  | Add debounce (150ms) to service search input event listener                                          | `html.templ`                                        | 8m     | MEDIUM   | 20%       |
| 34  | Regenerate html_templ.go + run HTML tests after a11y/UX changes                                      | `html_templ.go`                                     | 5m     | HIGH     | 20%       |
| 35  | Replace stripScriptTags: use template.HTML() safe check instead                                      | `fuzz_test.go`                                      | 12m    | MEDIUM   | 20%       |
| 36  | Add TestWriteHTML_MultiService realistic integration test — setup                                    | `plugin_html_test.go`                               | 12m    | MEDIUM   | 20%       |
| 37  | Add TestWriteHTML_MultiService — assertions (deps, scopes, events)                                   | `plugin_html_test.go`                               | 12m    | MEDIUM   | 20%       |
| 38  | Add PlantUML skinparam directives for better defaults                                                | `plantuml.go`                                       | 10m    | LOW      | 20%       |
| 39  | Add Mermaid theme styling (%%{init: {...}}%%)                                                        | `mermaid.go`                                        | 10m    | LOW      | 20%       |
| 40  | Clean stale mermaidNodeID references in docs/status/ files                                           | `docs/status/*.md`                                  | 10m    | LOW      | 20%       |
| 41  | Pin go.mod to `go 1.26` (remove patch number)                                                        | `go.mod`                                            | 5m     | LOW      | 20%       |
| 42  | Clean docs/archive/ — remove 10+ oldest/stalest files                                                | `docs/archive/`                                     | 12m    | LOW      | 20%       |
| 43  | Clean docs/archive/ — remove remaining stale files                                                   | `docs/archive/`                                     | 12m    | LOW      | 20%       |
| 44  | Add gosec security scanner run + fix findings                                                        | `.golangci.yml` or Makefile                         | 12m    | MEDIUM   | Remaining |
| 45  | Add govulncheck vulnerability scan                                                                   | CI config                                           | 12m    | MEDIUM   | Remaining |
| 46  | Add Go Report Card badge to README.md                                                                | `README.md`                                         | 5m     | LOW      | Remaining |
| 47  | Add touch event support for graph pan (touchstart/touchmove)                                         | `html.templ`                                        | 12m    | LOW      | Remaining |
| 48  | Add touch event support for graph zoom (touchstart 2-finger)                                         | `html.templ`                                        | 12m    | LOW      | Remaining |
| 49  | Regenerate html_templ.go + test touch events don't break mouse                                       | `html_templ.go`                                     | 8m     | LOW      | Remaining |
| 50  | Research virtual scrolling approach for large tables                                                 | —                                                   | 12m    | LOW      | Remaining |
| 51  | Implement "Show first N" pagination for services table                                               | `html.templ`                                        | 12m    | LOW      | Remaining |
| 52  | Implement "Show more" button for services table                                                      | `html.templ`                                        | 12m    | LOW      | Remaining |
| 53  | Implement "Show first N" pagination for events table                                                 | `html.templ`                                        | 12m    | LOW      | Remaining |
| 54  | Implement "Show more" button for events table                                                        | `html.templ`                                        | 12m    | LOW      | Remaining |
| 55  | Regenerate html_templ.go + final full verification                                                   | `html_templ.go`                                     | 5m     | HIGH     | Remaining |

**Total: 55 tasks.** Estimated total effort: ~9.5 hours.

---

## D2 Execution Graph

```d2
direction: down

tier1: Tier 1 — Critical (1% → 51%) {
  validate_new: Wire Config.Validate() into New()
  csp: CSP hardening
  keyboard: Keyboard nav fix
}

tier2: Tier 2 — Architecture + Docs (4% → 64%) {
  report_validate: Report.Validate() method
  enum_metadata: Go enum metadata injection
  docs: Update CHANGELOG/TODO/FEATURES/README
}

tier3: Tier 3 — Polish (20% → 80%) {
  a11y: aria-pressed + scope=col + empty states
  debounce: Debounce search
  fuzz: Replace stripScriptTags
  html_test: HTML integration test
  diagrams: PlantUML/Mermaid themes
}

tier4: Tier 4 — Nice-to-have {
  ci_sec: gosec + govulncheck
  badge: Go Report Card
  touch: Touch event support
  virtual_scroll: Virtual scrolling
  archive_cleanup: Clean docs/archive/
}

tier1.validate_new -> tier2.report_validate
tier1.validate_new -> tier2.docs
tier1.csp -> tier3.a11y
tier1.keyboard -> tier3.a11y
tier2.enum_metadata -> tier3.a11y
tier2.docs -> tier4.badge
tier3.a11y -> tier4.touch
tier3.a11y -> tier4.virtual_scroll
```

---

## Task Grouping for Parallel Execution

Tasks that can be done simultaneously (no file conflicts):

- **Group A** (docs): Tasks 16-22 (CHANGELOG, TODO, FEATURES, README) — all independent files
- **Group B** (html.templ): Tasks 11-13, 29-34 — sequential on same file
- **Group C** (Go enum metadata): Tasks 23-28 — sequential, new feature
- **Group D** (tests): Tasks 35-37 — independent test files

---

## Definition of Done

- [ ] Config.Validate() is called by New() — breaking change complete
- [ ] CSP includes base-uri and frame-ancestors
- [ ] Keyboard nav excludes TEXTAREA/SELECT/BUTTON
- [ ] Report.Validate() method exists and is tested
- [ ] Go enum metadata injected into HTML (no JS hardcoded constants)
- [ ] All HTML a11y improvements done (aria-pressed, scope=col, empty states)
- [ ] Service search debounced
- [ ] stripScriptTags replaced with robust approach
- [ ] HTML integration test added
- [ ] CHANGELOG, TODO_LIST, FEATURES, README all current
- [ ] Diagram themes added (Mermaid + PlantUML)
- [ ] gosec + govulncheck pass clean
- [ ] `go test -race ./...` passes
- [ ] `golangci-lint run ./...` passes
- [ ] Example runs correctly
- [ ] Everything pushed to origin/master
