// Package live provides a real-time HTTP dashboard for samber/do v2
// container lifecycle events.
//
// It serves an interactive HTML dashboard that updates live via Server-Sent
// Events (SSE) as services register, invoke, and shut down. Services light up
// as they register, change status as they invoke, and the full dependency graph
// snaps into place when the container lifecycle completes.
//
// # Quick Start
//
//	server, plugin, err := live.New(auditlog.Config{
//		ContainerID: "my-app",
//	}, live.Config{
//		Addr: ":8080",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	injector := do.NewWithOpts(plugin.Opts())
//	go server.ListenAndServe()
//
//	fmt.Println("Live dashboard: http://localhost:8080")
//
//	// ... register and invoke services ...
//
//	server.SignalComplete()
//
// # Architecture
//
// The live server uses SSE (Server-Sent Events) for real-time communication:
//
//   - GET /                   - Interactive dashboard HTML (static, cached)
//   - GET /api/report         - Current report as JSON (point-in-time snapshot)
//   - GET /api/events         - SSE stream (snapshots + live updates, below)
//   - GET /api/health         - Health check
//   - GET /api/export/ndjson  - Full event stream as NDJSON
//   - GET /api/export/html    - Self-contained static HTML report
//
// SSE was chosen over WebSocket because the data flow is one-way
// (server to browser), SSE has native browser support via EventSource,
// auto-reconnects on disconnect, and requires no framing protocol.
//
// # Protocol
//
// The dashboard is driven by Datastar (embedded, no build step): the server
// renders templ fragments and ships them as SSE events that the client
// runtime morphs into the DOM. The stream carries two named event types:
//
//   - datastar-patch-signals: a JSON "signals" payload for client state
//     (connStatus: connected|complete, complete, servicesOverflow,
//     eventsOverflow)
//   - datastar-patch-elements: an HTML fragment with "selector <css>" and
//     "mode inner" directives; Datastar replaces the inner HTML of the
//     matching element (tabs, stats, table rows, waveform, legend)
//
// On connect the server sends one full snapshot (all fragments + signals).
// Reconnecting clients that present a Last-Event-ID first receive a replay
// of missed events, then a fresh full snapshot — the snapshot IS the replay.
// During the lifecycle, event bursts are coalesced into a single re-render.
// On container completion the server sends the final snapshot plus a
// {"complete":true,"connStatus":"complete"} signal patch and closes the
// stream. Periodic SSE heartbeats run for the lifetime of the connection
// to keep proxies from closing it.
package live
