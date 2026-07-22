# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items have been moved to [CHANGELOG.md](CHANGELOG.md).
Last updated: 2026-07-22

---

## Completed Architecture Changes (Shipped)

Both items shipped together as a single breaking-change batch.

- [x] **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` are now distinct named string types. Compiler rejects accidental swaps at every boundary. External library calls wrap with `string()` at the IO boundary.
- [x] **Split `ServiceInfo` lifecycle concerns** — 19-field struct is now four embedded structs: `ServiceIdentity` / `ServiceLifecycle` / `ServiceHealth` / `ServiceGraph`. JSON output stays flat via Go embedding.

---

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package, ~2500 LOC). Revisit at 5+ packages.
- **External storage backends** — File and io.Writer exports are sufficient.
- **Prometheus/OpenTelemetry integration as a dependency** — Out of scope. Use OnEvent callback instead.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient for this project size.
- **`encoding/json/v2` migration** — Current `encoding/json` works fine. Risk of breaking JSON output format for consumers.
