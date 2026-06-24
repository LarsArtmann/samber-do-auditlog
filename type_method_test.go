package auditlog_test

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestEvent_ConvenienceMethods(t *testing.T) {
	t.Parallel()

	events := []struct {
		event      auditlog.Event
		wantReg    bool
		wantInv    bool
		wantShut   bool
		wantHealth bool
		wantBefore bool
		wantAfter  bool
	}{
		{
			event:   auditlog.Event{EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseBefore},
			wantReg: true, wantBefore: true,
		},
		{
			event:   auditlog.Event{EventType: auditlog.EventTypeInvocation, Phase: auditlog.PhaseAfter},
			wantInv: true, wantAfter: true,
		},
		{
			event:    auditlog.Event{EventType: auditlog.EventTypeShutdown, Phase: auditlog.PhaseBefore},
			wantShut: true, wantBefore: true,
		},
		{
			event:      auditlog.Event{EventType: auditlog.EventTypeHealthCheck, Phase: auditlog.PhaseAfter},
			wantHealth: true, wantAfter: true,
		},
	}

	for i, tc := range events {
		if got := tc.event.IsRegistration(); got != tc.wantReg {
			t.Errorf("case %d: IsRegistration() = %v, want %v", i, got, tc.wantReg)
		}

		if got := tc.event.IsInvocation(); got != tc.wantInv {
			t.Errorf("case %d: IsInvocation() = %v, want %v", i, got, tc.wantInv)
		}

		if got := tc.event.IsShutdown(); got != tc.wantShut {
			t.Errorf("case %d: IsShutdown() = %v, want %v", i, got, tc.wantShut)
		}

		if got := tc.event.IsHealthCheck(); got != tc.wantHealth {
			t.Errorf("case %d: IsHealthCheck() = %v, want %v", i, got, tc.wantHealth)
		}

		if got := tc.event.IsBefore(); got != tc.wantBefore {
			t.Errorf("case %d: IsBefore() = %v, want %v", i, got, tc.wantBefore)
		}

		if got := tc.event.IsAfter(); got != tc.wantAfter {
			t.Errorf("case %d: IsAfter() = %v, want %v", i, got, tc.wantAfter)
		}
	}
}

func TestEvent_Duration(t *testing.T) {
	t.Parallel()

	report := setupWithDBReport()

	invocations := report.EventsByType(auditlog.EventTypeInvocation)
	if len(invocations) == 0 {
		t.Fatal("expected invocation events")
	}

	afterInv := invocations[len(invocations)-1]

	d := afterInv.Duration()
	if d < 0 {
		t.Errorf("Duration() = %v, want >= 0", d)
	}

	beforeEvt := report.Events[0]
	if beforeEvt.Duration() != 0 {
		t.Errorf("Duration() on event with nil DurationMs = %v, want 0", beforeEvt.Duration())
	}
}

func TestEvent_HasError(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	provideDB(injector, "ok", "test")
	provideFailing(injector, "fail")

	_ = do.MustInvokeNamed[*Database](injector, "ok")
	_, _ = do.InvokeNamed[*Database](injector, "fail")

	report := p.Report()

	foundErrorEvent := false

	for _, e := range report.Events {
		if e.ServiceName == "fail" && e.IsAfter() && e.IsInvocation() {
			if !e.HasError() {
				t.Error("expected HasError for failing invocation")
			}

			foundErrorEvent = true
		}
	}

	if !foundErrorEvent {
		t.Fatal("expected at least one failing invocation event for 'fail' service")
	}
}

func TestServiceInfo_Uptime(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	db := findServiceByName(t, report, "db")
	if db == nil {
		t.Fatal("db not found")
	}

	uptime := db.Uptime()
	if uptime < 0 {
		t.Errorf("Uptime() = %v, want >= 0", uptime)
	}
}

func TestServiceInfo_HasHealthError(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	provideHealthyDB(injector, "ok", "test")
	provideUnhealthyCache(injector, "bad", "broken")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "ok")
	_ = do.MustInvokeNamed[*UnhealthyCache](injector, "bad")

	p.RecordHealthCheck(injector)

	report := p.Report()

	okSvc := findServiceByName(t, report, "ok")
	if okSvc == nil {
		t.Fatal("ok not found")
	}

	if okSvc.HasHealthError() {
		t.Error("healthy service should not have health error")
	}

	badSvc := findServiceByName(t, report, "bad")
	if badSvc == nil {
		t.Fatal("bad not found")
	}

	if !badSvc.HasHealthError() {
		t.Error("unhealthy service should have health error")
	}
}

func TestServiceStatus_IsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status auditlog.ServiceStatus
		want   bool
	}{
		{auditlog.ServiceStatusInvocationError, true},
		{auditlog.ServiceStatusShutdownError, true},
		{auditlog.ServiceStatusRegistered, false},
		{auditlog.ServiceStatusActive, false},
		{auditlog.ServiceStatusShutdown, false},
	}
	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			t.Parallel()

			if tc.status.IsError() != tc.want {
				t.Errorf("IsError() = %v, want %v", tc.status.IsError(), tc.want)
			}
		})
	}
}

func TestServiceRef_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref  auditlog.ServiceRef
		want string
	}{
		{auditlog.ServiceRef{ScopeName: "api", ServiceName: "db"}, "api/db"},
		{auditlog.ServiceRef{ScopeName: auditlog.RootScopeName, ServiceName: "db"}, "db"},
		{auditlog.ServiceRef{ScopeName: "", ServiceName: "db"}, "db"},
	}
	for _, tc := range tests {
		t.Run(tc.ref.String(), func(t *testing.T) {
			t.Parallel()

			if tc.ref.String() != tc.want {
				t.Errorf("String() = %q, want %q", tc.ref.String(), tc.want)
			}
		})
	}
}

func TestServiceRef_IsRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  auditlog.ServiceRef
		want bool
	}{
		{"empty scope", auditlog.ServiceRef{ScopeName: "", ServiceName: "db"}, true},
		{"root scope", auditlog.ServiceRef{ScopeName: auditlog.RootScopeName, ServiceName: "db"}, true},
		{"child scope", auditlog.ServiceRef{ScopeName: "child", ServiceName: "db"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.ref.IsRoot() != tt.want {
				t.Errorf("IsRoot() = %v, want %v", tt.ref.IsRoot(), tt.want)
			}
		})
	}
}

func TestProviderType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pt   auditlog.ProviderType
		want string
	}{
		{auditlog.ProviderTypeLazy, "lazy"},
		{auditlog.ProviderTypeEager, "eager"},
		{auditlog.ProviderTypeTransient, "transient"},
		{auditlog.ProviderTypeAlias, "alias"},
		{auditlog.ProviderType("unknown"), "unknown"},
	}

	for _, tc := range tests {
		got := tc.pt.String()
		if got != tc.want {
			t.Errorf("ProviderType(%q).String() = %q, want %q", tc.pt, got, tc.want)
		}
	}
}

func TestProviderType_Icon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pt auditlog.ProviderType
	}{
		{auditlog.ProviderTypeLazy},
		{auditlog.ProviderTypeEager},
		{auditlog.ProviderTypeTransient},
		{auditlog.ProviderTypeAlias},
	}
	for _, tc := range tests {
		icon := tc.pt.Icon()
		if icon == "" {
			t.Errorf("ProviderType(%q).Icon() returned empty string", tc.pt)
		}
	}

	unknown := auditlog.ProviderType("unknown")
	if icon := unknown.Icon(); icon != "" {
		t.Errorf("unknown ProviderType.Icon() = %q, want empty", icon)
	}
}

func TestProviderType_IsKnown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pt   auditlog.ProviderType
		want bool
	}{
		{auditlog.ProviderTypeLazy, true},
		{auditlog.ProviderTypeEager, true},
		{auditlog.ProviderTypeTransient, true},
		{auditlog.ProviderTypeAlias, true},
		{auditlog.ProviderType(""), false},
		{auditlog.ProviderType("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.pt), func(t *testing.T) {
			t.Parallel()

			if tt.pt.IsKnown() != tt.want {
				t.Errorf("IsKnown() = %v, want %v", tt.pt.IsKnown(), tt.want)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	err := auditlog.Config{}.Validate()
	if err != nil {
		t.Errorf("empty config should be valid, got: %v", err)
	}

	err = auditlog.Config{Enabled: true, OnEvent: func(e auditlog.Event) {}}.Validate()
	if err != nil {
		t.Errorf("valid config should pass, got: %v", err)
	}

	err = auditlog.Config{ContainerID: "my/app"}.Validate()
	if err == nil {
		t.Error("expected error for ContainerID with forward slash")
	}

	err = auditlog.Config{ContainerID: "my\\app"}.Validate()
	if err == nil {
		t.Error("expected error for ContainerID with backslash")
	}

	err = auditlog.Config{ContainerID: "my-app"}.Validate()
	if err != nil {
		t.Errorf("hyphenated ContainerID should be valid, got: %v", err)
	}
}
