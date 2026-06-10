# templ-components: Dependency or Inspiration?

**Date**: 2026-06-10 · **Decision**: Learn from it, don't depend on it

---

## Context

`samber-do-auditlog` has a self-contained HTML export (`html.templ`) that renders a full audit report as a single HTML file with inline CSS and JS. `templ-components` is a mature templ component library (~25 components, 1100+ tests, Tailwind-based). The question: should auditlog depend on `templ-components` for its HTML visualization, or just learn from its patterns?

---

## Analysis

### Why NOT depend on `templ-components`

| Factor                       | Detail                                                                                                                                                                                                                                                                                                                     |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **depguard blocks it**       | `.golangci.yml` only allows `$gostd`, `a-h/templ`, `samber`, and `samber-do-auditlog`. Adding `templ-components` pulls in `tailwind-merge-go` and `go-error-family` — 2 transitive deps for a self-contained export.                                                                                                       |
| **Different design purpose** | `templ-components` builds composable UI widgets (Badge, Card, Table, Tabs) that render via Tailwind CSS classes, designed for embedding in web apps. Our `html.templ` is a self-contained, single-file HTML export with inline CSS/JS — no Tailwind, no external CSS, no framework. Fundamentally different architectures. |
| **Tailwind dependency**      | `templ-components` requires Tailwind (class-based styling + `tailwind-merge-go` for conflict resolution). Our export deliberately uses inline CSS custom properties for zero-dependency portability. Adding Tailwind as a build dep or rewriting every component's styling makes no sense.                                 |
| **Coupling risk**            | Consumers `go get samber-do-auditlog` for DI observability. Pulling in an entire component library (+ transitive deps) for a single self-contained HTML export is overreach. Every `templ-components` breaking change becomes our breaking change.                                                                         |
| **Scale mismatch**           | We'd use maybe 3-4 components (Card, Table, Badge, Tabs) out of 25+, plus icons. The dependency surface is massive relative to the value.                                                                                                                                                                                  |

### What `templ-components` does well (patterns worth learning)

| Pattern                       | How `templ-components` does it                                                                                                 | How we do it now                                                      | Gap                                                 |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------- | --------------------------------------------------- |
| **Tabs**                      | `role="tablist"`, `role="tab"`, `role="tabpanel"`, `aria-selected`, `aria-controls`, keyboard nav (ArrowLeft/Right, Home, End) | `<button class="tab">` with click handlers + keyboard shortcuts `1-5` | **Missing ARIA roles** — our tabs aren't accessible |
| **Status badges**             | `StatusBadge(status)` auto-maps status strings to badge types via lookup map                                                   | CSS classes `status-badge.{status}` with inline styles                | Equivalent, both valid                              |
| **Lookup maps over switches** | `badgeStyleMap`, `cardPaddingLookup`, `progressHeightLookup` — map-based style resolution                                      | JS object maps in inline script                                       | Equivalent pattern                                  |
| **Default constructors**      | `DefaultXxxProps()` for every component                                                                                        | No props structs (monolithic template)                                | N/A for our single-template design                  |
| **Props composition**         | `BaseProps` embedded in all props structs, `ComponentProps` interface                                                          | Single `Report` parameter                                             | N/A for our single-template design                  |
| **Snapshot/golden testing**   | `internal/golden.Assert(t, name, got)` with CSS normalization and `-update` flag                                               | String-contains assertions on HTML output                             | Could improve our HTML test coverage                |
| **Table**                     | `TableProps` with `Headers []string`, `Rows []TableRow`, `TableCell.Content` (templ.Component) for rich cells                  | JS-generated `<table>` from report JSON                               | Adequate for self-contained export                  |
| **Card**                      | `CardProps` with Title, Subtitle, Footer, HeaderAction, Padding                                                                | CSS `.stat-card` with grid layout                                     | Equivalent, ours is simpler                         |
| **Accessibility**             | `motion-reduce:` on all transitions/animations, `aria-live` for dynamic content                                                | No motion-reduce, no `aria-live`                                      | Worth adding for accessibility                      |

### Architecture comparison

```
templ-components                    samber-do-auditlog
──────────────────                  ──────────────────
Composable widgets                  Single monolithic template
Tailwind CSS classes                Inline CSS custom properties
External CSS framework              Zero external dependencies
props structs per component         Report struct → JS rendering
Designed for web app embedding      Designed for self-contained export
~1100 tests                         ~34 tests
```

These are not competing approaches — they solve different problems.

---

## Recommendations

### Do

1. **Add ARIA attributes to tabs** — `role="tablist"`, `role="tab"`, `role="tabpanel"`, `aria-selected`, `aria-controls`. This is the single highest-value takeaway from `templ-components`' accessibility patterns.
2. **Add `motion-reduce:` media query** for CSS transitions and animations.
3. **Consider golden/snapshot testing** for the HTML export — `internal/golden` pattern is clean.
4. **Keep the self-contained architecture** — it's the correct design for a DI audit export.

### Don't

1. **Don't add `templ-components` as a dependency** — wrong abstraction level, wrong styling model, unnecessary coupling.
2. **Don't refactor into composable components** — the single-template design is simpler and more appropriate for this use case.
3. **Don't adopt Tailwind** — inline CSS custom properties give us zero-dependency portability.

---

## If requirements change

If `samber-do-auditlog` ever needs to:

- **Embed reports in a web app** (not just export standalone HTML) → reconsider `templ-components` as the rendering layer
- **Support theming/skinning** → inline CSS custom properties are already the right foundation
- **Add interactive widgets** (not just static visualization) → evaluate specific components individually, not the whole library

---

## Summary

`templ-components` is a well-designed library that solves a different problem. Our self-contained HTML export is the correct architecture for a DI audit tool. The main actionable takeaway is accessibility: proper ARIA roles on tabs and `prefers-reduced-motion` support. Everything else is architecturally sound as-is.
