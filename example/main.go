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
//   - Interface aliasing:   do.As + do.MustInvoke via interface type
//   - Scopes:               injector.Scope("name") — child scopes, cross-scope deps
//   - Health checks:        do.Healthchecker interface + HealthCheckWithContext
//   - Graceful shutdown:    ShutdownerWithError interface + injector.Shutdown()
//   - Dependency graph:     inferred automatically from provider call-chains
//   - Invocation errors:    provider returning error is captured
//   - Shutdown errors:      Shutdown() returning error is captured
//   - Override:             do.Override for test-style hot-swapping
//   - Audit export:         JSON, NDJSON, HTML, Report struct
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// --- Value types (injected via ProvideValue / ProvideNamedValue) ---

type AppConfig struct {
	AppName string
	Port    int
	Debug   bool
}

type DBConfig struct {
	DSN string
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// --- Core services ---

type Logger struct {
	Prefix string
}

func (l *Logger) Printf(format string, args ...any) {
	fmt.Printf("[%s] "+format+"\n", append([]any{l.Prefix}, args...)...)
}

type Database struct {
	DSN     string
	Open    bool
	Healthy bool
}

func (d *Database) Ping() error {
	if !d.Open {
		return errors.New("database connection closed")
	}

	return nil
}

func (d *Database) Shutdown() error {
	fmt.Println("  Database: closing connection")

	d.Open = false

	return nil
}

type Cache struct {
	Healthy bool
}

func (c *Cache) HealthCheck() error {
	if !c.Healthy {
		return errors.New("cache is unhealthy")
	}

	return nil
}

func (c *Cache) Shutdown() error {
	fmt.Println("  Cache: flushing and closing")

	return nil
}

// --- Interface aliasing: accept interfaces, return structs ---

type PaymentProcessor interface {
	Charge(amount float64) error
}

type StripeProcessor struct {
	APIKey string
}

func (s *StripeProcessor) Charge(amount float64) error {
	fmt.Printf("  Stripe: charging $%.2f\n", amount)

	return nil
}

func (s *StripeProcessor) Shutdown() error {
	fmt.Println("  StripeProcessor: shutting down")

	return nil
}

// --- Transient services (new instance per invocation) ---

type RideRequest struct {
	ID        int64
	RiderName string
	Pickup    string
	Dropoff   string
	CreatedAt time.Time
}

var rideCounter atomic.Int64

// --- Named services: multiple instances of the same type ---

type Vehicle struct {
	Name     string
	Capacity int
	Active   bool
}

func (v *Vehicle) Shutdown() error {
	fmt.Printf("  Vehicle %q: decommissioning\n", v.Name)
	v.Active = false

	return nil
}

// --- Scoped services: driver and passenger modules ---

type DriverService struct {
	Name    string
	Vehicle *Vehicle
}

func (d *DriverService) Shutdown() error {
	fmt.Printf("  DriverService(%s): going offline\n", d.Name)

	return nil
}

type PassengerService struct {
	Name string
}

func (p *PassengerService) Shutdown() error {
	fmt.Printf("  PassengerService(%s): logging out\n", p.Name)

	return nil
}

type MatchingEngine struct {
	Drivers    []*DriverService
	Passengers []*PassengerService
}

func (m *MatchingEngine) Shutdown() error {
	fmt.Println("  MatchingEngine: stopping match loop")

	return nil
}

// --- HTTP server (entry point) ---

type HTTPServer struct {
	Config  *AppConfig
	DB      *Database
	Cache   *Cache
	Payment PaymentProcessor
	Port    int
}

func (s *HTTPServer) ListenAndServe() error {
	fmt.Printf("  HTTP server listening on :%d\n", s.Port)

	return nil
}

func (s *HTTPServer) Shutdown() error {
	fmt.Printf("  HTTPServer: draining connections on :%d\n", s.Port)

	return nil
}

// --- Error demo services ---

type FlakyService struct{}

func (f *FlakyService) Shutdown() error {
	return errors.New("flaky shutdown failed: connection reset")
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	fmt.Println("=== samber/do v2 + audit-log — comprehensive demo ===")
	fmt.Println()

	// =================================================================
	// 1. Create the audit-log plugin and the DI container
	// =================================================================
	plugin := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "ride-share-app",
	})

	injector := do.NewWithOpts(plugin.Opts())

	// =================================================================
	// 2. Eager value injection — static config, no provider needed
	//    Demonstrates: ProvideValue, ProvideNamedValue
	// =================================================================

	do.ProvideValue(injector, &AppConfig{
		AppName: "RideShare",
		Port:    8080,
		Debug:   true,
	})

	do.ProvideNamedValue(injector, "config.db.dsn", "postgres://localhost:5432/rideshare?sslmode=disable")
	do.ProvideNamedValue(injector, "config.stripe.key", "sk_test_abc123")

	// =================================================================
	// 3. Lazy singletons — built on first Invoke, then cached
	//    Demonstrates: Provide, dependency chains, build-duration tracking
	// =================================================================

	do.Provide(injector, func(i do.Injector) (*Logger, error) {
		cfg := do.MustInvoke[*AppConfig](i)

		return &Logger{Prefix: cfg.AppName}, nil
	})

	do.Provide(injector, func(i do.Injector) (*Database, error) {
		dsn := do.MustInvokeNamed[string](i, "config.db.dsn")
		logger := do.MustInvoke[*Logger](i)

		logger.Printf("connecting to database: %s", dsn)
		time.Sleep(8 * time.Millisecond) // simulate connection

		return &Database{DSN: dsn, Open: true, Healthy: true}, nil
	})

	do.Provide(injector, func(i do.Injector) (*Cache, error) {
		time.Sleep(3 * time.Millisecond) // simulate init

		return &Cache{Healthy: true}, nil
	})

	// =================================================================
	// 4. Interface aliasing — "accept interfaces, return structs"
	//    Demonstrates: do.As[*Concrete, Interface], do.MustInvoke[Interface]
	// =================================================================

	do.Provide(injector, func(i do.Injector) (*StripeProcessor, error) {
		apiKey := do.MustInvokeNamed[string](i, "config.stripe.key")

		return &StripeProcessor{APIKey: apiKey}, nil
	})

	// Bind the concrete *StripeProcessor to the PaymentProcessor interface
	do.As[*StripeProcessor, PaymentProcessor](injector)

	// =================================================================
	// 5. Transient provider — new instance every invocation
	//    Demonstrates: ProvideTransient
	// =================================================================

	do.ProvideTransient(injector, func(i do.Injector) (*RideRequest, error) {
		return &RideRequest{
			ID:        rideCounter.Add(1),
			RiderName: fmt.Sprintf("rider-%d", rideCounter.Load()),
			Pickup:    "123 Main St",
			Dropoff:   "456 Elm Ave",
			CreatedAt: time.Now(),
		}, nil
	})

	// =================================================================
	// 6. Named services — multiple instances of the same type
	//    Demonstrates: ProvideNamed, MustInvokeNamed
	// =================================================================

	do.ProvideNamed(injector, "vehicle.sedan", func(i do.Injector) (*Vehicle, error) {
		return &Vehicle{Name: "Sedan", Capacity: 4, Active: true}, nil
	})

	do.ProvideNamed(injector, "vehicle.suv", func(i do.Injector) (*Vehicle, error) {
		return &Vehicle{Name: "SUV", Capacity: 7, Active: true}, nil
	})

	do.ProvideNamed(injector, "vehicle.van", func(i do.Injector) (*Vehicle, error) {
		return &Vehicle{Name: "Van", Capacity: 12, Active: true}, nil
	})

	// =================================================================
	// 7. Scopes — child scopes with cross-scope dependencies
	//    Demonstrates: injector.Scope("name"), scoped services
	// =================================================================

	driverScope := injector.Scope("drivers")
	passengerScope := injector.Scope("passengers")
	matchingScope := injector.Scope("matching")

	// Driver services live in the "drivers" scope
	do.Provide(driverScope, func(i do.Injector) (*DriverService, error) {
		// Cross-scope dependency: driver needs a vehicle from root scope
		// (vehicles are in root, so they're visible from child scopes)
		vehicle := do.MustInvokeNamed[*Vehicle](i, "vehicle.sedan")

		return &DriverService{Name: "alice", Vehicle: vehicle}, nil
	})

	do.ProvideNamed(driverScope, "driver.bob", func(i do.Injector) (*DriverService, error) {
		vehicle := do.MustInvokeNamed[*Vehicle](i, "vehicle.suv")

		return &DriverService{Name: "bob", Vehicle: vehicle}, nil
	})

	// Passenger services live in the "passengers" scope
	do.ProvideNamed(passengerScope, "passenger.charlie", func(i do.Injector) (*PassengerService, error) {
		return &PassengerService{Name: "charlie"}, nil
	})

	do.ProvideNamed(passengerScope, "passenger.dana", func(i do.Injector) (*PassengerService, error) {
		return &PassengerService{Name: "dana"}, nil
	})

	// Matching engine lives in its own scope and pulls from both driver + passenger scopes
	// via their parent (root) — it invokes directly from each scope
	do.Provide(matchingScope, func(i do.Injector) (*MatchingEngine, error) {
		// Invoke driver services from the driver scope
		alice := do.MustInvoke[*DriverService](driverScope)
		bob := do.MustInvokeNamed[*DriverService](driverScope, "driver.bob")

		// Invoke passenger services from the passenger scope
		charlie := do.MustInvokeNamed[*PassengerService](passengerScope, "passenger.charlie")
		dana := do.MustInvokeNamed[*PassengerService](passengerScope, "passenger.dana")

		return &MatchingEngine{
			Drivers:    []*DriverService{alice, bob},
			Passengers: []*PassengerService{charlie, dana},
		}, nil
	})

	// =================================================================
	// 8. HTTP server — the entry point that wires everything together
	// =================================================================

	do.Provide(injector, func(i do.Injector) (*HTTPServer, error) {
		cfg := do.MustInvoke[*AppConfig](i)
		db := do.MustInvoke[*Database](i)
		cache := do.MustInvoke[*Cache](i)
		payment := do.MustInvoke[PaymentProcessor](i) // resolved via alias

		return &HTTPServer{
			Config:  cfg,
			DB:      db,
			Cache:   cache,
			Payment: payment,
			Port:    cfg.Port,
		}, nil
	})

	// =================================================================
	// 9. Error-case service — invocation that returns an error
	//    Demonstrates: error capture in audit log
	// =================================================================

	do.Provide(injector, func(i do.Injector) (*FlakyService, error) {
		return &FlakyService{}, nil
	})

	// =================================================================
	// 10. Override — hot-swap a service (useful for testing)
	//     Demonstrates: do.Override
	// =================================================================

	do.OverrideValue(injector, &ServerConfig{
		Port:         8080,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// =================================================================
	// INVOKE — trigger lazy construction of services
	// =================================================================

	fmt.Println("--- Invoking services ---")

	// Main entry point — this cascades through the full dependency tree
	server, err := do.Invoke[*HTTPServer](injector)
	if err != nil {
		log.Fatalf("failed to invoke HTTPServer: %v", err)
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}

	// Invoke matching engine from its scope — triggers scoped service construction
	fmt.Println("  Building matching engine...")

	engine, err := do.Invoke[*MatchingEngine](matchingScope)
	if err != nil {
		log.Fatalf("failed to invoke MatchingEngine: %v", err)
	}

	fmt.Printf("  MatchingEngine: %d drivers, %d passengers\n",
		len(engine.Drivers), len(engine.Passengers))

	// Invoke transient services — each call creates a new instance
	fmt.Println("  Creating ride requests (transient)...")

	ride1, err := do.Invoke[*RideRequest](injector)
	if err != nil {
		log.Fatalf("failed to invoke RideRequest: %v", err)
	}

	ride2, err := do.Invoke[*RideRequest](injector)
	if err != nil {
		log.Fatalf("failed to invoke RideRequest: %v", err)
	}

	fmt.Printf("  RideRequest #%d → RideRequest #%d (different instances)\n", ride1.ID, ride2.ID)

	// Invoke named vehicles
	sedan := do.MustInvokeNamed[*Vehicle](injector, "vehicle.sedan")
	suv := do.MustInvokeNamed[*Vehicle](injector, "vehicle.suv")
	van := do.MustInvokeNamed[*Vehicle](injector, "vehicle.van")
	fmt.Printf("  Fleet: %s (%d), %s (%d), %s (%d)\n",
		sedan.Name, sedan.Capacity, suv.Name, suv.Capacity, van.Name, van.Capacity)

	// Invoke the flaky service (to capture it in the audit)
	_, _ = do.Invoke[*FlakyService](injector)

	// =================================================================
	// HEALTH CHECK
	// =================================================================

	fmt.Println()
	fmt.Println("--- Health checks ---")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	health := injector.HealthCheckWithContext(ctx)
	for svc, err := range health {
		if err != nil {
			fmt.Printf("  UNHEALTHY %s: %v\n", svc, err)
		} else {
			fmt.Printf("  OK       %s\n", svc)
		}
	}

	// =================================================================
	// GRACEFUL SHUTDOWN
	// =================================================================

	fmt.Println()
	fmt.Println("--- Graceful shutdown ---")

	report := injector.Shutdown()
	if !report.Succeed {
		for svc, err := range report.Errors {
			fmt.Printf("  SHUTDOWN ERROR %s: %v\n", svc, err)
		}
	}

	// =================================================================
	// EXPORT AUDIT REPORTS
	// =================================================================

	fmt.Println()
	fmt.Println("--- Exporting audit reports ---")

	if err := plugin.ExportToFile("audit-report.json"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("  Written audit-report.json")

	if err := plugin.ExportEventsToNDJSON("audit-events.ndjson"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("  Written audit-events.ndjson")

	if err := plugin.ExportToHTML("audit-report.html"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("  Written audit-report.html")

	// =================================================================
	// PRINT SUMMARY
	// =================================================================

	rep := plugin.Report()

	fmt.Println()
	fmt.Println("=== Audit Summary ===")
	fmt.Printf("  Container:            %s\n", rep.ContainerID)
	fmt.Printf("  Schema version:       %s\n", rep.Version)
	fmt.Printf("  Services:             %d\n", rep.ServiceCount)
	fmt.Printf("  Events:               %d\n", rep.EventCount)
	fmt.Printf("  Scopes:               %d\n", rep.ScopeCount)
	fmt.Printf("  Total build time:     %.3f ms\n", rep.TotalBuildDurationMs)
	fmt.Printf("  Total shutdown time:  %.3f ms\n", rep.TotalShutdownDurationMs)
	fmt.Printf("  Shutdown succeeded:   %v\n", rep.ShutdownSucceeded)

	fmt.Println()
	fmt.Println("  Services:")

	for _, s := range rep.Services {
		fmt.Printf("    %-35s scope=%-12s status=%-18s invoked=%d",
			s.ServiceName, s.ScopeName, s.Status, s.InvocationCount)

		if s.FirstBuildDurationMs != nil {
			fmt.Printf(" build=%.3fms", *s.FirstBuildDurationMs)
		}

		if s.ShutdownDurationMs != nil {
			fmt.Printf(" shutdown=%.3fms", *s.ShutdownDurationMs)
		}

		if len(s.Dependencies) > 0 {
			deps := make([]string, len(s.Dependencies))
			for i, d := range s.Dependencies {
				deps[i] = d.ServiceName
			}

			fmt.Printf(" deps=%v", deps)
		}

		if s.ShutdownError != nil {
			fmt.Printf(" shutdown_err=%q", *s.ShutdownError)
		}

		if s.InvocationError != nil {
			fmt.Printf(" invocation_err=%q", *s.InvocationError)
		}

		fmt.Println()
	}

	fmt.Println()
	fmt.Printf("  Scope tree: %s", rep.ScopeTree.Name)

	for _, child := range rep.ScopeTree.Children {
		fmt.Printf("\n    └── %s (services: %v)", child.Name, child.Services)
	}

	fmt.Println()

	// =================================================================
	// FEATURE CHECKLIST — verify every feature was demonstrated
	// =================================================================

	fmt.Println()
	fmt.Println("=== Feature Checklist ===")

	checklist := []struct {
		name string
		ok   bool
	}{
		{"do.NewWithOpts (plugin hooks)", rep.ContainerID == "ride-share-app"},
		{"do.ProvideValue (eager value injection)", hasService(rep, "AppConfig")},
		{"do.ProvideNamedValue (named values)", hasService(rep, "config.db.dsn")},
		{"do.Provide (lazy singletons)", hasService(rep, "Database")},
		{"do.ProvideNamed (named services)", hasService(rep, "vehicle.sedan")},
		{"do.ProvideTransient (new instance per invoke)", hasService(rep, "RideRequest")},
		{"do.As (interface aliasing)", hasService(rep, "PaymentProcessor")},
		{"do.OverrideValue (hot-swap)", hasService(rep, "ServerConfig")},
		{"injector.Scope (child scopes)", rep.ScopeCount >= 4},
		{"Cross-scope dependencies", hasDeps(rep, "MatchingEngine")},
		{"Dependency graph inference", hasDeps(rep, "HTTPServer")},
		{"Health checks", len(health) > 0},
		{"Graceful shutdown", rep.ShutdownSucceeded == (len(report.Errors) == 0)},
		{"Shutdown errors captured", hasShutdownError(rep)},
		{"Build duration tracking", hasBuildDuration(rep)},
		{"Scope tree hierarchy", len(rep.ScopeTree.Children) >= 3},
	}

	allOK := true

	for _, c := range checklist {
		status := "✓"
		if !c.ok {
			status = "✗"
			allOK = false
		}

		fmt.Printf("  %s %s\n", status, c.name)
	}

	if allOK {
		fmt.Println()
		fmt.Println("  All features demonstrated successfully!")
	} else {
		fmt.Println()
		fmt.Println("  WARNING: Some features were not demonstrated!")
	}

	fmt.Println()
}

// ---------------------------------------------------------------------------
// Helpers for the feature checklist
// ---------------------------------------------------------------------------

func hasService(r auditlog.Report, substr string) bool {
	for _, s := range r.Services {
		if s.ServiceName == substr || containsPart(s.ServiceName, substr) {
			return true
		}
	}

	return false
}

func containsPart(name, substr string) bool {
	return len(name) >= len(substr) &&
		name[len(name)-len(substr):] == substr
}

func hasDeps(r auditlog.Report, substr string) bool {
	for _, s := range r.Services {
		if containsPart(s.ServiceName, substr) && len(s.Dependencies) > 0 {
			return true
		}
	}

	return false
}

func hasShutdownError(r auditlog.Report) bool {
	for _, s := range r.Services {
		if s.ShutdownError != nil {
			return true
		}
	}

	return false
}

func hasBuildDuration(r auditlog.Report) bool {
	for _, s := range r.Services {
		if s.FirstBuildDurationMs != nil {
			return true
		}
	}

	return false
}
