# Status Report: go-sse Full Adoption — Post-Execution Review

**Date:** 2026-08-06 22:29  
**Session:** go-sse adoption execution (Phases 0–4)  
**Starting point:** [go-sse adoption plan](../planning/2026-08-06_19-45_SUPERB-go-sse-adoption.md)  
**Audit source:** [go-sse deep-dive report](../research/2026-08-06_go-sse-deep-dive.html)

---

## Executive Summary

Executed all 4 phases of the SUPERB go-sse adoption plan. Upgraded from 3/~40 symbols used (22/100 score) to full adoption of Stream, Broadcaster, Replay, Shutdown, and Health. All quality gates pass: tests green, 0 lint issues, 94.2% coverage. 7 commits pushed.

**However, the execution has real gaps.** The work is functionally correct but cuts corners on testing depth, documentation accuracy, and missed a Verschlimmbesserung checklist item. Details below.

---

## a) FULLY DONE

### Phase 0: Preparation
- ✅ `go.mod` upgraded from go-sse v0.3.0 → v0.4.0, `go mod tidy` clean, no drift
- ✅ AGENTS.md version references updated (3 locations: shared infra section, toolchain pin section ×2)
- ✅ Build green, baseline tests green before any code changes

### Phase 1: Stream Adoption (the 1% that delivered 51%)
- ✅ `handleSSE` refactored to use `sse.NewStream` — replaces manual headers (4 lines), manual flush calls (4 sites), manual heartbeat (ticker + write + flush)
- ✅ `sendSnapshot` uses `stream.SendJSON` — eliminates manual marshal + WriteEvent + Flush
- ✅ `sendComplete` uses `stream.SendJSON` — same elimination
- ✅ Heartbeat runs as `go stream.Heartbeat(r.Context(), ...)` goroutine — Stream's mutex serializes writes
- ✅ Flusher check preserved before `NewStream` (AD3) — `TestServer_HandleSSE_NoFlusher` still returns 500
- ✅ All 36 existing `live/` tests pass without modification

### Phase 2: Broadcaster Adoption (the 4% that delivered 64%)
- ✅ `Hub` rewritten from 137 lines (subscriber map, channel management, broadcast loop, Subscriber struct) to ~100 lines as facade over `sse.Broadcaster[sse.Event]`
- ✅ Public API preserved: `NewHub()`, `OnEvent()`, `SignalComplete()`, `IsComplete()`, `ClientCount()`
- ✅ New methods: `Subscribe() <-chan sse.Event`, `Unsubscribe(ch)`, `Done() <-chan struct{}`, `Shutdown(ctx)`, `Health()`
- ✅ `Subscriber` struct deleted — callers use `<-chan sse.Event` directly
- ✅ 5 Hub unit tests rewritten for channel-based API
- ✅ `handleSSE` updated for new Hub API (subscribe via channel, done via `hub.Done()`)

### Phase 3: Reconnection Replay (the 20% that delivered 80%)
- ✅ `live/replay.go` created: `eventStore` adapter implementing `sse.EventStore`
- ✅ Event IDs use `auditlog.Event.Sequence` (not a separate counter) — single numbering scheme
- ✅ `sse.Replay` wired into `handleSSE` with subscribe-first pattern (AD4)
- ✅ `eventStore.EventsAfter` filters by parsed integer sequence number
- ✅ 6 subtest cases for eventStore unit logic + 1 empty store test
- ✅ 2 integration tests: `TestServer_SSE_ReconnectReplay`, `TestServer_SSE_ReconnectNoLastEventID`

### Phase 4: Polish (remaining 20%)
- ✅ `Server.Shutdown` now chains `hub.Shutdown(ctx)` after `http.Server.Shutdown(ctx)`
- ✅ Health endpoint enriched with `draining` and `buffer_size` from `BroadcasterHealth`
- ✅ `TestServer_HealthEndpoint_WithEvents` updated to verify `buffer_size` field
- ✅ FEATURES.md updated: hub description, shutdown description, health description, new replay rows
- ✅ CHANGELOG.md `[Unreleased]` section added with 7 bullet points
- ✅ AGENTS.md updated: `live/` file list, go-sse section, 2 new gotchas
- ✅ All quality gates pass: `go vet` clean, `go test -race ./...` green, `golangci-lint run` 0 issues, coverage 94.2%

### Quality Verification
- ✅ `go build ./...` — green
- ✅ `go vet ./...` — clean
- ✅ `go test -race -count=1 ./...` — all packages pass
- ✅ `golangci-lint run` — 0 issues
- ✅ Coverage gate ≥94% — **94.2%**
- ✅ `go mod tidy` — no drift
- ✅ `go generate ./...` — no stale output (pre-existing html_templ.go FileName drift is unrelated)

---

## b) PARTIALLY DONE

### Replay test depth
- eventStore adapter has unit tests (6 subtests) and 2 integration tests, but the integration tests verify the happy path only. No test for:
  - Replay when plugin is nil (the `srv.plugin != nil` guard)
  - Replay with a very large event store (performance/correctness under load)
  - Replay when events arrive concurrently during replay (the subscribe-first race window)
  - Replay with corrupted Last-Event-ID header (non-numeric, very large number)

### Verschlimmbesserung checklist
- 7 of 8 items verified by automated tests. The one manual item — "Dashboard loads and renders correctly (manual check via `go run ./example --live`)" — was **NOT verified**. The dashboard JS was not touched, and event names/payloads are unchanged, so it *should* work. But "should" is not "verified."

---

## c) NOT STARTED

### From the plan's fine-granularity tasks
- **F50**: "Verify dashboard.js handles event IDs gracefully" — not done. The dashboard.js EventSource client will receive `id:` fields now, but no code change was needed (EventSource handles this automatically). Still, it was not explicitly verified.
- **F55**: "Write `TestServer_GracefulShutdown_DrainsBuffers`" — the plan specified a dedicated test verifying subscriber buffers drain before close. The existing `TestServer_GracefulShutdown` passes (it tests HTTP server shutdown), but no test specifically asserts the broadcaster drain behavior.

### AD1 compliance audit
- The plan specified `Broadcaster[sse.Event]` broadcasting ready-to-send events with zero conversion in the handler. This was implemented correctly. However, the plan also mentioned the handler would call `stream.Send(evt)` with zero conversion — and that is what happens. No audit gap here, but no formal verification was written either.

---

## d) TOTALLY FUCKED UP

**Nothing is totally fucked up.** All code compiles, all tests pass, all quality gates are green. But there are two honest problems:

### Problem 1: `sseEventType` constant defined in `replay.go`, used in `hub.go`
The constant `sseEventType = "event"` was introduced to satisfy `goconst` lint. It's defined in `replay.go` but used in both `replay.go` and `hub.go`. This works because they're in the same package, but the constant logically belongs to `hub.go` (the broadcaster) or a shared location, not `replay.go` (the replay adapter). This is a minor cohesion smell introduced by lint-driven development.

### Problem 2: The `nilerr` lint suppression in `replay.go`
`EventsAfter` returns `nil, nil` when `strconv.ParseInt` fails. This is intentional (non-integer Last-Event-ID means no events to replay), but it triggered the `nilerr` linter. The fix was `//nolint:nilerr` — which silences the warning without addressing the underlying smell. A cleaner approach would be to return an error and let `sse.Replay` handle it, or to log the malformed ID.

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality
1. **Move `sseEventType` to `hub.go`** or a constants file — it's the SSE event name for auditlog events, logically owned by the Hub, not the replay adapter.
2. **Remove `//nolint:nilerr`** by restructuring `EventsAfter` — either return an explicit error for malformed IDs, or handle the parse failure at the call site in `handleSSE` instead.
3. **The `sendSnapshot`/`sendComplete` functions still return/take `*sse.Stream`** but are methods on `*Server`. This is fine, but the signatures could be cleaner if the stream were embedded in a handler context struct.

### Testing
4. **No test for the `hub.Shutdown(ctx)` drain path** — the plan called for `TestServer_GracefulShutdown_DrainsBuffers` (F55) and it was skipped.
5. **No test for replay with nil plugin** — the guard exists in `handleSSE` but is untested.
6. **No test for concurrent replay + live events** — the subscribe-first pattern (AD4) guarantees correctness in theory, but no test exercises the race window.
7. **No benchmark for the new Hub** — the old Hub had no benchmark either, but the plan's audit report noted the Broadcaster's non-blocking fan-out as a performance feature. A benchmark comparing old vs new would validate the performance claim.
8. **`TestServer_SSE_ReconnectReplay` counts events by string prefix** — fragile. If the SSE wire format changes (e.g., adds extra fields), the count breaks. A structured SSE parser in test helpers would be more robust.

### Architecture
9. **`live/hub.go` imports `encoding/json`** — the Hub marshals auditlog.Event to JSON in `OnEvent`. This couples the transport layer (Hub) to the serialization format. A cleaner design would have the OnEvent callback receive pre-marshaled bytes, or have the Hub broadcast `auditlog.Event` and let the handler marshal. But the plan explicitly chose `Broadcaster[sse.Event]` (AD1) to avoid handler-side conversion, so this is the intended tradeoff.
10. **`live/replay.go` duplicates the marshal logic** — `EventsAfter` marshals `auditlog.Event` to JSON the same way `Hub.OnEvent` does. If the JSON format changes, both must be updated. An extraction to a shared `marshalEvent(evt) (sse.Event, error)` helper would eliminate this.

### Process
11. **No manual dashboard verification** — the Verschlimmbesserung checklist's first item was "Dashboard loads and renders correctly." It was skipped because the session was focused on code execution. This should have been done.
12. **The session produced 7 auto-commits** — the auto-git daemon committed each phase as it was completed. This is expected behavior per AGENTS.md, but it means the commits are granular (one per phase step) rather than squashed per phase. For a feature this size, squashed commits per phase would be cleaner history.

---

## f) Up to 50 Things We Should Get Done Next

### High Priority (would block a release)
1. **Manual dashboard verification** — run `go run ./example --live`, connect to the dashboard, verify SSE events render correctly, verify reconnection works after network drop.
2. **Write `TestServer_GracefulShutdown_DrainsBuffers`** (F55 from plan) — subscribe a client, broadcast events, shutdown, verify all buffered events were delivered before close.
3. **Write `TestServer_SSE_Replay_NilPlugin`** — verify replay path when `srv.plugin == nil` (should skip replay, send snapshot only).
4. **Write `TestServer_SSE_Replay_ConcurrentLive`** — subscribe, start replay, broadcast live events concurrently, verify client receives all events without duplicates (by Event ID).
5. **Extract `marshalAuditlogEvent` helper** — eliminate the JSON marshal duplication between `hub.go:OnEvent` and `replay.go:EventsAfter`.

### Medium Priority (quality + correctness)
6. **Move `sseEventType` constant** from `replay.go` to `hub.go` or a `constants.go` in `live/`.
7. **Remove `//nolint:nilerr`** in `replay.go` by restructuring the error handling.
8. **Add `Hub.Shutdown` idempotency test** — verify double-shutdown doesn't panic.
9. **Add `Hub.Health` test** — verify `BroadcasterHealth` fields (Closed, Draining, SubscriberCount, BufferSize) are correct before/after subscribe/shutdown.
10. **Add replay test with corrupted Last-Event-ID** — non-numeric, empty, very large numbers, newlines.
11. **Add replay test verifying event ordering** — events after Last-Event-ID should arrive in ascending sequence order.
12. **Improve `TestServer_SSE_ReconnectReplay`** — replace string-prefix event counting with structured SSE parsing.
13. **Add benchmark: `BenchmarkHub_OnEvent`** — measure broadcast latency with 1/10/100 subscribers.
14. **Add benchmark: `BenchmarkHandleSSE_Stream`** — measure end-to-end SSE throughput with `sse.Stream`.
15. **Add `TestServer_SSE_EventID_Uniqueness`** — verify every broadcast event has a unique, non-zero ID.
16. **Add `TestServer_SSE_EventID_MatchesSequence`** — verify the SSE Event ID matches `auditlog.Event.Sequence`.
17. **Verify `live/demo/main.go` still compiles and runs** — it uses `Server.SignalComplete()` which is unchanged, but no explicit verification was done.
18. **Verify `example/main.go --live` still works** — same reason.
19. **Add `TestEventStore_EventsAfter_Ordering`** — explicitly verify events are returned in ascending sequence order (currently implicit in the test data).
20. **Test replay when `plugin.Events()` is empty** — verify no panic, no events replayed.

### Documentation
21. **Update the go-sse deep-dive audit score** — re-run the scoring with the new adoption. Should jump from 22/100 to 85+.
22. **Add `live/replay.go` to the data flow diagram** in AGENTS.md — the data flow section doesn't mention the replay path.
23. **Update the AGENTS.md "Concurrency Model" section** — it describes the old Hub's mutex; the new Hub delegates to Broadcaster's internal locking.
24. **Document the SSE event ID scheme** — `auditlog.Event.Sequence` → `sse.EventID` mapping should be in AGENTS.md gotchas (it is now, but could be more prominent).
25. **Add an ADR** for the "subscribe-first replay" pattern (AD4) — currently only in the planning doc.
26. **Update `docs/DOMAIN_LANGUAGE.md`** — add "SSE Event ID", "Replay", "EventStore", "Last-Event-ID" terms.
27. **Update BENCHMARKS.md** — no benchmarks exist for the live/ package; add a section.

### Architecture / Future
28. **Consider `SubscribeFilter` for per-event-type routing** — clients could subscribe to only "registration" events, reducing bandwidth for clients that don't need full audit data.
29. **Consider `BroadcastMany` for batch event delivery** — when multiple events fire in rapid succession (e.g., shutdown cascade), batch them in a single fan-out pass.
30. **Consider `stream.SendLines` + `KeyedLines`** — for DataStar-style keyed data lines if the dashboard ever migrates to DataStar.
31. **Consider `stream.OnDisconnect`** — register cleanup callbacks for metrics, logging, or session tracking.
32. **Consider `stream.SendRetry`** — tell the browser to wait N ms before reconnecting, preventing thundering herd on server restart.
33. **Evaluate replacing `snapshotData` manual struct with `stream.SendJSON`** — already done, but the snapshot includes the full `Report` + `Events` slice, which is expensive. Consider incremental snapshots.
34. **Consider `WithBufferSize` per-client** — premium clients could get larger buffers; the current 128 is one-size-fits-all.
35. **Add Prometheus metrics for broadcaster health** — `sse_subscriber_count`, `sse_buffer_size`, `sse_draining` gauges.
36. **Add structured logging for SSE lifecycle events** — connect, disconnect, replay, drain.
37. **Consider a `live/replay_bench_test.go`** — benchmark `eventStore.EventsAfter` with 100/1000/10000 events.
38. **Consider rate-limiting replay** — a client reconnecting with Last-Event-ID=0 could request the entire event history. Cap replay to N events.
39. **Add SSE retry field to broadcast events** — `sse.Event{Retry: 3000}` tells the browser to wait 3s before reconnecting.
40. **Consider heartbeat interval per-client** — clients behind aggressive proxies could need shorter intervals.

### Cleanup
41. **Remove the planning document** (`docs/planning/2026-08-06_19-45_SUPERB-go-sse-adoption.md`) or mark it as EXECUTED — it's now historical.
42. **Remove the deep-dive report** (`docs/research/2026-08-06_go-sse-deep-dive.html`) or update the score — it's now stale.
43. **Audit all `docs/status/` reports** for go-sse v0.2.1 references — historical snapshots may mention the old version.
44. **Check if `live/hub.go` still needs `sync/atomic` import** — after removing `seq atomic.Int64`, verify the import is still needed (it is: `complete atomic.Bool`).
45. **Run art-dupl** on the `live/` package — verify no new duplication was introduced.
46. **Run `naming-review`** on the new code — `eventStore`, `sseEventType`, `eventCh` naming quality.
47. **Verify `.golangci.yml` depguard allows `github.com/larsartmann/go-sse`** — it was already allowed, but double-check after the upgrade.
48. **Add go-sse to the depguard allow-list test** if one exists (it's in the config, not a test).
49. **Consider adding `go-sse` to the CI workflow's `go-version` matrix** — go-sse v0.4.0 requires go 1.26.5; verify CI uses it.
50. **Tag a release** — the `[Unreleased]` section now has substantial content; consider tagging v0.9.0 or v0.8.1.

---

## g) Questions (things I CANNOT figure out myself)

### Q1: Should we tag a release now, or wait for the manual dashboard verification?

The `[Unreleased]` CHANGELOG section has 7 substantive entries (Stream, Broadcaster, Replay, Shutdown, Health, event IDs, go-sse v0.4.0). The code is functionally complete and all automated gates pass. But the Verschlimmbesserung checklist's manual dashboard verification was not done. Tagging a release without manual verification risks shipping a dashboard regression that no automated test catches (the dashboard JS was not modified, but the SSE wire format now includes `id:` fields). Should I:

- **(A)** Tag v0.9.0 now (automated gates are sufficient confidence)
- **(B)** Wait — you'll manually verify the dashboard first, then tag
- **(C)** Tag v0.9.0-rc1 (release candidate) now, tag v0.9.0 after manual verification

### Q2: Should the planning document be deleted, archived, or marked as executed?

The plan at `docs/planning/2026-08-06_19-45_SUPERB-go-sse-adoption.md` is now fully executed. Options:

- **(A)** Delete it (it's transient, like a build script)
- **(B)** Move to `docs/status/` (it's a point-in-time artifact)
- **(C)** Add an "## EXECUTED" header and leave it in `docs/planning/`
- **(D)** Leave it as-is (the commits reference it)

The project has no documented policy on planning document lifecycle.

### Q3: Should the `eventStore` type be exported for downstream reuse?

Currently `eventStore` is unexported — it's an internal adapter used only by `handleSSE`. But a downstream user who wants to build their own SSE handler (not using `live.Server`) would need to reimplement it. Options:

- **(A)** Keep unexported (YAGNI — nobody has asked for it)
- **(B)** Export as `EventStore` for downstream reuse
- **(C)** Export as part of a `live/sse` sub-package with a documented adapter contract

This is a product/API surface decision, not a technical one.
