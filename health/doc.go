// Package health provides a production-ready health-probe SDK for samber/do v2
// containers. It turns the three-probe Kubernetes pattern (liveness, readiness,
// startup) into a single [Probe] type with sensible defaults.
//
// The package separates three distinct concerns that are often wrongly
// conflated into a single /health endpoint:
//
//   - Liveness: "Is the process alive?" — trivially fast, dependency-free.
//   - Readiness: "Can I serve traffic?" — checks all services, gates on critical.
//   - Startup: "Am I done booting?" — latches once all critical services pass.
//
// # Quick Start
//
//	plugin, _ := auditlog.New(auditlog.Config{
//	    Enabled:    true,
//	    ContainerID: "my-app",
//	})
//	injector := do.NewWithOpts(plugin.Opts())
//
//	// ... register and invoke services ...
//
//	probe := health.New(injector,
//	    health.WithPlugin(plugin),
//	    health.WithCriticalServices("database", "redis"),
//	    health.WithVersion("1.0.0"),
//	)
//
//	probe.Start(ctx)
//	defer probe.Shutdown()
//
//	mux := http.NewServeMux()
//	probe.RegisterRoutes(mux, health.DefaultRoutes())
//
// # Why Three Probes?
//
// A single /health endpoint conflates "process alive" with "dependencies
// reachable." When a dependency blips, the endpoint returns 503, the kubelet
// restarts the pod, and a restart cascade follows — even though the process
// itself is fine. Splitting probes breaks this coupling:
//
//   - /healthz (liveness) never checks dependencies. Only a deadlocked or
//     crashed process fails.
//   - /readyz (readiness) checks dependencies but only returns 503 for
//     critical failures. Non-critical failures (e.g. metrics exporter) are
//     surfaced in the response body without removing the pod from rotation.
//   - /startupz (startup) lets slow-booting apps use a generous kubelet
//     failureThreshold without affecting liveness/readiness sensitivity.
//
// # Background Caching
//
// Kubelet and load balancers poll health endpoints frequently (often every
// second). Without caching, each readiness check calls Ping() on every
// dependency, hammering downstream systems. The Probe runs health checks on a
// bounded background loop (default: every 1 second) and serves cached results
// so the HTTP endpoint is always O(1).
//
// Disable caching for low-traffic or development scenarios:
//
//	probe := health.New(injector, health.WithRefreshInterval(0))
//
// # Shutdown Awareness
//
// Call [Probe.Shutdown] (or [Probe.MarkShuttingDown] for two-phase graceful
// shutdown) during your server's graceful-drain path. Readiness immediately
// returns 503 so load balancers stop sending traffic before connections close.
// Liveness stays 200 because the process is still alive.
//
// # Audit Integration
//
// When an [auditlog.Plugin] is provided via [WithPlugin], every health-check
// batch is recorded as a timed audit event, giving you full observability into
// dependency health over time.
package health
