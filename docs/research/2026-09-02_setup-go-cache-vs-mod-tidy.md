# Research — Why setup-go's cache doesn't shield `go mod tidy` from proxy downloads

**Date:** 2026-09-02 · **Source task:** Pareto plan T113 (CI Resilience)

## Observation

2026-09-01: 2 of 3 CI runs died on transient `proxy.golang.org` transport
errors (`stream error … INTERNAL_ERROR; received from peer`) — one in
`mod-tidy`, one in `stale-generation`. Both jobs use `actions/setup-go` with
`cache: true`, yet both still hit the network.

## Root cause analysis

1. **The cache key includes `go.sum`** (setup-go default: hash of
   `**/go.sum`). The `mod-tidy` job exists to VERIFY that `go mod tidy`
   produces no diff — on a PR where `go.mod`/`go.sum` changed, the key misses
   and the job downloads the world from the proxy. That is precisely the run
   where a flake is most damaging (it reddens an otherwise-good PR).
2. **`go generate` builds tool binaries.** The `tool` directive
   (`go tool templ`) resolves templ through the module cache, but tool
   EXECUTABLE builds can still consult the proxy for missing
   module-version metadata, and setup-go's cache does not cover
   `$GOMODCACHE/cache/download` lockfiles state fully across runner images.
3. **Transport errors are mid-connection failures**, not 404s: the module
   cache only helps for content already fetched. Any first-time fetch
   (new dependency, re-keyed cache, image refresh) is exposed.

## Mitigations (implemented or proposed)

| Mitigation | Status |
|---|---|
| Retry wrapper (3 attempts, 15s) around `go mod tidy` and `go generate` in CI | ✅ implemented 2026-09-02 (transport flakes only; drift checks never retry) |
| `GONOSUMDB`/`GOFLAGS=-mod=mod` tweaks | ❌ rejected — no effect on transport flakes |
| Alternate proxy (GOPROXY fallback list `https://proxy.golang.org,direct`) | possible; adds variance, not reliability |
| `actions/cache` of `~/go/pkg/mod` keyed independently of go.sum | ⚠️ risks stale-module false greens; setup-go's built-in cache is safer — do not layer |
| Vendor the module graph | ❌ rejected (Explicitly Rejected in ROADMAP: repo-wide vendoring is churn for a mono-consumer library) |

## Conclusion

The residual flake exposure is bounded and now tolerated: a transport flake
needs to fail 3 times in a row (~30s window) to redden a run. Revisit only if
flakes recur at >1 per month after the retry wrapper lands.
