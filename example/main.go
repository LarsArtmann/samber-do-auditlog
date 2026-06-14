// Package main demonstrates EVERY major samber/do v2 feature, all observed
// by the audit-log plugin.  Run with DO_AUDITLOG_ENABLED=true (or the inline
// Enabled flag below) to produce JSON, NDJSON and self-contained HTML reports.
//
// Features shown:
//
//   - Container creation: do.NewWithOpts (with plugin hooks)
//   - Eager value injection: do.ProvideValue, do.ProvideNamedValue
//   - Lazy singleton:       do.Provide
//   - Named services:       do.ProvideNamed / do.MustInvokeNamed
//   - Transient providers:  do.ProvideTransient (new instance per invoke)
//   - Interface aliasing:   do.As (explicit) + do.MustInvokeAs (implicit)
//   - Scopes:               injector.Scope("name") — child scopes, cross-scope deps
//   - Health checks:        do.Healthchecker interface + HealthCheckWithContext
//   - Graceful shutdown:    ShutdownerWithError interface + injector.Shutdown()
//   - Dependency graph:     inferred automatically from provider call-chains
//   - Invocation errors:    provider returning error is captured
//   - Shutdown errors:      Shutdown() returning error is captured
//   - Override:             do.OverrideValue for test-style hot-swapping
//   - OnEvent callback:     real-time event streaming via Config.OnEvent
//   - Audit export:         JSON, NDJSON, HTML, Report struct
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func main() {
	fmt.Println("=== samber/do v2 + audit-log — comprehensive demo ===")
	fmt.Println()

	plugin, injector, eventLog := setupPlugin()
	matchingScope := registerServices(injector)

	runInvocations(injector, matchingScope)
	runHealthChecks(plugin, injector)
	runShutdown(injector)
	exportReports(plugin)

	printSummary(plugin.Report(), eventLog)
}

// setupPlugin creates the audit-log plugin and the DI container with an
// OnEvent callback that records every successful invocation.
//
// The event log is returned as a pointer so the closure and the caller share
// the same slice header across append reallocations.
func setupPlugin() (*auditlog.Plugin, do.Injector, *[]string) { //nolint:ireturn
	eventLog := &[]string{}

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "ride-share-app",
		OnEvent: func(e auditlog.Event) {
			if e.IsAfter() && e.IsInvocation() {
				*eventLog = append(*eventLog, e.ServiceName)
			}
		},
	})
	if err != nil {
		log.Fatalf("failed to create audit-log plugin: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())

	return plugin, injector, eventLog
}

// runInvocations triggers lazy construction of services and prints progress.
func runInvocations(injector, matchingScope do.Injector) {
	fmt.Println("--- Invoking services ---")

	server, err := do.Invoke[*HTTPServer](injector)
	if err != nil {
		log.Fatalf("failed to invoke HTTPServer: %v", err)
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}

	fmt.Println("  Building matching engine...")

	engine, err := do.Invoke[*MatchingEngine](matchingScope)
	if err != nil {
		log.Fatalf("failed to invoke MatchingEngine: %v", err)
	}

	fmt.Printf("  MatchingEngine: %d drivers, %d passengers\n",
		len(engine.Drivers), len(engine.Passengers))

	fmt.Println("  Creating ride requests (transient)...")

	ride1, err := do.Invoke[*RideRequest](injector)
	if err != nil {
		log.Fatalf("failed to invoke RideRequest: %v", err)
	}

	ride2, err := do.Invoke[*RideRequest](injector)
	if err != nil {
		log.Fatalf("failed to invoke RideRequest: %v", err)
	}

	fmt.Printf("  RideRequest #%d and #%d (different instances, transient)\n", ride1.ID, ride2.ID)

	sedan := do.MustInvokeNamed[*Vehicle](injector, "vehicle.sedan")
	suv := do.MustInvokeNamed[*Vehicle](injector, "vehicle.suv")
	van := do.MustInvokeNamed[*Vehicle](injector, "vehicle.van")
	fmt.Printf("  Fleet: %s(%d), %s(%d), %s(%d)\n",
		sedan.Name, sedan.Capacity, suv.Name, suv.Capacity, van.Name, van.Capacity)

	_ = do.MustInvoke[*LeakyService](injector)

	_, invocationErr := do.Invoke[*UnreliableService](injector)
	if invocationErr != nil {
		fmt.Printf("  Expected invocation error: %v\n", invocationErr)
	}
}

// runHealthChecks runs health checks via the plugin wrapper, demonstrating
// both Healthchecker and HealthcheckerWithContext interfaces.
func runHealthChecks(plugin *auditlog.Plugin, injector do.Injector) {
	fmt.Println()
	fmt.Println("--- Health checks ---")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	health := plugin.RecordHealthCheckWithContext(ctx, injector)
	for svc, err := range health {
		if err != nil {
			fmt.Printf("  UNHEALTHY %s: %v\n", svc, err)
		} else {
			fmt.Printf("  OK       %s\n", svc)
		}
	}
}

// runShutdown gracefully shuts down the container, capturing any errors.
func runShutdown(injector do.Injector) {
	fmt.Println()
	fmt.Println("--- Graceful shutdown ---")

	if err := injector.Shutdown(); err != nil {
		fmt.Printf("SHUTDOWN ERROR: %v\n", err)
	}
}

// exportReports writes the JSON, NDJSON, and HTML audit reports to a temp dir.
func exportReports(plugin *auditlog.Plugin) {
	fmt.Println()
	fmt.Println("--- Exporting audit reports ---")

	tmpDir, err := os.MkdirTemp("", "auditlog-demo-")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}

	if err := plugin.ExportToFile(tmpDir + "/audit-report.json"); err != nil {
		log.Fatalf("JSON export failed: %v", err)
	}

	fmt.Println("  Written " + tmpDir + "/audit-report.json")

	if err := plugin.ExportEventsToNDJSON(tmpDir + "/audit-events.ndjson"); err != nil {
		log.Fatalf("NDJSON export failed: %v", err)
	}

	fmt.Println("  Written " + tmpDir + "/audit-events.ndjson")

	if err := plugin.ExportToHTML(tmpDir + "/audit-report.html"); err != nil {
		log.Fatalf("HTML export failed: %v", err)
	}

	fmt.Println("  Written " + tmpDir + "/audit-report.html")
}
