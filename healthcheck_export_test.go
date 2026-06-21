package auditlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// healthySvc is a simple service that always passes health checks.
type healthySvc struct{}

var _ do.Healthchecker = (*healthySvc)(nil)

func (healthySvc) HealthCheck() error { return nil }

func TestPlugin_HealthCheckReportSucceeded(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideUnhealthyCache(injector, "cache", "down")

	_ = do.MustInvokeNamed[*UnhealthyCache](injector, "cache")

	_ = p.RecordHealthCheck(injector)
	report := p.Report()

	if report.HealthCheckSucceeded {
		t.Error("HealthCheckSucceeded should be false when a service is unhealthy")
	}

	unhealthy := report.UnhealthyServices()
	assertUnhealthyServiceCount(t, unhealthy, "cache")
}

func TestPlugin_HealthCheckSucceededFalseWhenNoChecks(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")

	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	if report.HealthCheckSucceeded {
		t.Error("HealthCheckSucceeded should be false when no health checks have been recorded")
	}

	assertEqual(t, "HealthCheckedCount", report.HealthCheckedCount, 0)
}

func TestPlugin_HealthCheckOnEventCallback(t *testing.T) {
	t.Parallel()

	p, captured, injector := newPluginWithCapture()

	provideHealthyDB(injector, "db", "test")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	_ = p.RecordHealthCheck(injector)

	var healthCallbacks []auditlog.Event

	for _, e := range *captured {
		if e.IsHealthCheck() {
			healthCallbacks = append(healthCallbacks, e)
		}
	}

	if len(healthCallbacks) != 1 {
		t.Fatalf("expected 1 health check event via OnEvent, got %d", len(healthCallbacks))
	}

	evt := healthCallbacks[0]
	if evt.EventType != auditlog.EventTypeHealthCheck {
		t.Errorf("event type: want health_check, got %s", evt.EventType)
	}

	if evt.Phase != auditlog.PhaseAfter {
		t.Errorf("phase: want after, got %s", evt.Phase)
	}

	assertEqual(t, "service name", evt.ServiceName, "db")

	if evt.Error != nil {
		t.Errorf("expected no error for healthy service, got %s", *evt.Error)
	}
}

func TestPlugin_HealthCheckPhaseIsAfterOnly(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "test")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")
	_ = p.RecordHealthCheck(injector)

	report := p.Report()
	healthEvents := report.EventsByType(auditlog.EventTypeHealthCheck)

	if len(healthEvents) != 1 {
		t.Fatalf("expected 1 health check event, got %d", len(healthEvents))
	}

	if healthEvents[0].Phase != auditlog.PhaseAfter {
		t.Errorf("health check events should always be PhaseAfter, got %s", healthEvents[0].Phase)
	}

	if healthEvents[0].DurationMs != nil {
		t.Error("health check events should not have DurationMs (per-service timing unavailable)")
	}
}

func TestPlugin_HealthCheckJSONExport(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "test")
	provideUnhealthyCache(injector, "cache", "down")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")
	_ = do.MustInvokeNamed[*UnhealthyCache](injector, "cache")

	_ = p.RecordHealthCheck(injector)

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	var report map[string]any

	err = json.Unmarshal(buf.Bytes(), &report)
	if err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if report["health_check_succeeded"] != false {
		t.Error("expected health_check_succeeded to be false")
	}

	services, ok := report["services"].([]any)
	if !ok {
		t.Fatal("services should be an array")
	}

	for _, svc := range services {
		svcMap, ok := svc.(map[string]any)
		if !ok {
			continue
		}

		name, _ := svcMap["service_name"].(string)
		if name == "cache" {
			if svcMap["health_check_error"] == nil {
				t.Error("cache should have health_check_error in JSON export")
			}
		}
	}
}

func TestPlugin_HealthCheckNDJSONExport(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "test")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")
	_ = p.RecordHealthCheck(injector)

	var buf bytes.Buffer

	err := p.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	lines := ndjsonLines(buf.String())
	foundHealthCheck := false

	for _, line := range lines {
		var evt map[string]any

		err := json.Unmarshal([]byte(line), &evt)
		if err != nil {
			t.Fatalf("json unmarshal line: %v", err)
		}

		if evt["event_type"] == "health_check" {
			foundHealthCheck = true

			if evt["phase"] != "after" {
				t.Errorf("health check event phase: want after, got %v", evt["phase"])
			}

			if _, ok := evt["duration_ms"]; ok {
				t.Error("health check events should not have duration_ms in NDJSON export")
			}
		}
	}

	if !foundHealthCheck {
		t.Error("expected at least one health_check event in NDJSON export")
	}
}

func TestPlugin_HealthCheckDiscoversUnregisteredService(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	p.RecordHealthCheck(injector)

	report := p.Report()

	dbSvc := findServiceByName(t, report, "db")
	if dbSvc == nil {
		t.Fatal("db not found in report")
	}

	if dbSvc.HealthCheckCount != 1 {
		t.Errorf("health check count: want 1, got %d", dbSvc.HealthCheckCount)
	}
}

func TestPlugin_HealthCheckWithContextCancelled(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := p.RecordHealthCheckWithContext(ctx, injector)
	_ = results

	report := p.Report()

	events := report.EventsByType(auditlog.EventTypeHealthCheck)
	if len(events) == 0 {
		t.Error("expected health check events even with cancelled context")
	}
}

func TestReport_HealthCheckSucceeded_NoChecks(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideValue(injector, &Database{URL: "test"})
	_ = do.MustInvoke[*Database](injector)

	report := p.Report()
	if report.HealthCheckSucceeded {
		t.Error("expected false when no health checks ran")
	}
}

func TestReport_AllHealthChecksPassed_AllHealthy(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "healthy", func(_ do.Injector) (*healthySvc, error) {
		return &healthySvc{}, nil
	})

	_ = do.MustInvokeNamed[*healthySvc](injector, "healthy")

	p.RecordHealthCheck(injector)

	report := p.Report()
	if !report.HealthCheckSucceeded {
		t.Error("expected health checks to succeed")
	}
}
