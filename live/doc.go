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
//   - GET /            - Interactive dashboard HTML (static, cached)
//   - GET /api/report  - Current report as JSON (point-in-time snapshot)
//   - GET /api/events  - SSE stream (snapshot + live events + completion)
//   - GET /api/health  - Health check
//
// SSE was chosen over WebSocket because the data flow is one-way
// (server to browser), SSE has native browser support via EventSource,
// auto-reconnects on disconnect, and requires no framing protocol.
//
// # Protocol
//
// The SSE stream sends three named event types:
//
//   - snapshot: Initial state on connect (report + events + metadata + DAG)
//   - event: Individual events as they fire during container lifecycle
//   - complete: Final report with full DAG structure after completion
//
// Late clients receive the full state via the snapshot event, including all
// events captured so far. After completion, new clients get the final report.
package live
