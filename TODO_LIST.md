# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items have been moved to [CHANGELOG.md](CHANGELOG.md).
Last updated: 2026-07-22

---

## Future Architecture (Deferred to Next Breaking Release)

These two items are paired and should ship together as a single breaking-change batch.

- [ ] **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as distinct named string types. Compiler rejects accidental swaps; validation moves into constructors. Low effort, high safety. **DEFERRED**: blast radius measured at 65+ compile errors across production + tests + generated templ, with zero existing bugs from string usage.
- [ ] **Split `ServiceInfo` lifecycle concerns** — 19-field struct into `ServiceIdentity` / `ServiceLifecycle` / `ServiceHealth` / `ServiceGraph`. Breaking API change. **DEFERRED**: embedding breaks all struct literals (~50 sites), nesting breaks all field access; YAGNI applies (no consumer complaints, JSON flattens fine). Do alongside typed identifiers.

---

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package, ~2500 LOC). Revisit at 5+ packages.
- **External storage backends** — File and io.Writer exports are sufficient.
- **Prometheus/OpenTelemetry integration as a dependency** — Out of scope. Use OnEvent callback instead.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient for this project size.
- **`encoding/json/v2` migration** — Current `encoding/json` works fine. Risk of breaking JSON output format for consumers.
