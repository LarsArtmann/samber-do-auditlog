# Merge Proposal: auditlog → `github.com/samber/do` debug sub-package

From: Lars Artmann (`github.com/larsartmann/samber-do-auditlog`)
To: Samuel Berthe (`github.com/samber/do`)
Date: 2026-09-04 · Branch: `go1.23-compat` (all claims verified there)

## 1. The offer

Donate the audit-log plugin to `samber/do` as the replacement for the current
`http` debug sub-package, per your note that "it could replace the current
http sub-package. Your API is much better." The branch is merge-ready:

- **Go 1.23 floor** (your call: "1.23 means 2 years back. Seems good for a
  UI"). CI runs 7 green jobs on 1.23, including `-race`, a 94% coverage gate,
  and a first-class `example --live` smoke test.
- **Zero third-party runtime dependencies.** The only runtime requirement is
  `samber/do/v2` itself (+ `samber/go-type-to-string`, which do already
  pulls). This answers your module-split concern without a split: there is
  nothing left to isolate.
- **Live updates included** — you said "I would prefer keeping live update vs
  backward compatibility", so the real-time dashboard was rewritten on the
  standard library (`net/http` SSE + `html/template`) and works on the same
  1.23 floor. Browser assets (datastar runtime, keyboard nav) are carried
  over verbatim; the wire format is identical.

## 2. What the sub-package gives do users

A `debug`/`di` package (name yours) that instruments a do container and
serves:

1. **Live dashboard** (SSE): services light up as they register, status
   changes on invoke/error/shutdown, scope tree, dependency graph, build /
   shutdown timeline, event stream — all updating in real time in the
   browser, zero build step, one `datastar.js` asset embedded via `go:embed`.
2. **Self-contained HTML report**: the same visualization as a single static
   file for post-mortems (`Report.WriteHTML`).
3. **Exports**: JSON, NDJSON (streamable), CSV/TSV, Markdown/ASCII tables,
   Mermaid / PlantUML / DOT / D2 dependency diagrams — all stdlib-rendered.
4. **JSON schema** for the report format, generated and embedded.
5. **Programmatic API**: typed event stream (`OnEvent`), report diffing,
   filtering, replay from NDJSON, health-check recording.
6. **CLI** (`cmd/auditlog`): inspect/convert/diff/validate report files.

The recording hooks are fail-open: a disabled plugin costs samber/do's own
hook dispatch only (~131 ns, 4 allocs in our benchmark), and `Opts()` returns
empty hooks when disabled.

## 3. Integration shape (your choice, both cheap)

- **Option A (recommended)**: import the package as
  `github.com/samber/do/internal/auditlog` (or `/debug`), re-exported through
  a small `do.DebugPlugin(opts...)` helper. Users call one function and get
  `InjectorOpts` wired.
- **Option B**: keep it as a separate module `github.com/samber/do-auditlog`
  under the do org — one `go get`, no impact on do's own dependency tree.

Either way the consumer code is:

```go
plugin, err := auditlog.New(auditlog.Config{ContainerID: "my-app"})
injector := do.NewWithOpts(plugin.Opts())
go plugin.ServeHTTPDebug(":8080") // live dashboard + endpoints
```

## 4. Phasing

1. **Phase 1 — land the sub-package** replacing `http/` (this branch as-is;
   package path/naming to your liking). Its tests, CI, and docs travel with
   it. The existing `http/` handlers can delegate to it during a deprecation
   window or be deleted outright.
2. **Phase 2 — (optional, later) converge master's dependency-rich line**:
   master keeps extra table formats and error-family classification that a
   1.23 floor cannot carry; they can become master-only or be ported as
   hand-rolled renderers if wanted upstream.

## 5. Credit

As agreed: a mention in the do GitHub release notes and a README mention
("live debug dashboard contributed by Lars Artmann"). No LICENSE change.

## 6. Verification you can rerun

```sh
git clone -b go1.23-compat https://github.com/LarsArtmann/samber-do-auditlog
cd samber-do-auditlog
GOTOOLCHAIN=go1.23.12 go build ./... && go vet ./...
GOTOOLCHAIN=go1.23.12 go test -race ./...
sh scripts/coverage-gate.sh          # 94.9% (gate ≥94%, live/ 94.2%)
DO_AUDITLOG_ENABLED=true go run ./example --live   # open http://localhost:7777/debug/di/
```

CI evidence: run `33813979691` — all 7 jobs green (Test+race+coverage, Lint,
actionlint, vulncheck, mod-tidy, stale-generation, example smoke).
