package auditlog_test

import (
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReport_Validate_ConsistentReport(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	provideDB(injector, "db", "test")
	provideCache(injector, "cache")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")

	report := p.Report()

	assertReportValid(t, report, "")
}

func TestReport_Validate_WithScopesAndHealthChecks(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	child := injector.Scope("child")

	provideHealthyDB(injector, "healthy-svc", "ok")
	provideUnhealthyCache(child, "sick-svc", "down")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "healthy-svc")
	_ = do.MustInvokeNamed[*UnhealthyCache](child, "sick-svc")

	_ = p.RecordHealthCheck(injector)

	report := p.Report()

	assertReportValid(t, report, "with scopes+health")
}

func TestReport_Validate_DetectsCountMismatch(t *testing.T) {
	t.Parallel()

	makeDBReport := func(t *testing.T) auditlog.Report {
		t.Helper()

		p, injector := newPluginAndInjector()

		provideDB(injector, "db", "test")
		_ = do.MustInvokeNamed[*Database](injector, "db")

		return p.Report()
	}

	makeHealthCheckedReport := func(t *testing.T) auditlog.Report {
		t.Helper()

		p, injector := newPluginAndInjector()

		provideHealthyDB(injector, "healthy-svc", "ok")
		_ = do.MustInvokeNamed[*HealthyDB](injector, "healthy-svc")
		_ = p.RecordHealthCheck(injector)

		return p.Report()
	}

	tests := []struct {
		name        string
		make        func(t *testing.T) auditlog.Report
		corrupt     func(*auditlog.Report)
		errContains string
	}{
		{
			name:        "event_count",
			make:        makeDBReport,
			corrupt:     func(r *auditlog.Report) { r.EventCount = 999 },
			errContains: "event_count",
		},
		{
			name:        "service_count",
			make:        makeDBReport,
			corrupt:     func(r *auditlog.Report) { r.ServiceCount = 999 },
			errContains: "service_count",
		},
		{
			name:        "scope_count",
			make:        makeDBReport,
			corrupt:     func(r *auditlog.Report) { r.ScopeCount = 999 },
			errContains: "scope_count",
		},
		{
			name:        "health_checked_count",
			make:        makeHealthCheckedReport,
			corrupt:     func(r *auditlog.Report) { r.HealthCheckedCount = 999 },
			errContains: "health_checked_count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			report := tt.make(t)
			tt.corrupt(&report)

			if err := report.Validate(); err == nil || !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("expected error containing %q, got %v", tt.errContains, err)
			}
		})
	}
}

func TestReport_Validate_EmptyReport(t *testing.T) {
	t.Parallel()

	p, _ := newPluginAndInjector()

	report := p.Report()

	assertReportValid(t, report, "empty")
}
