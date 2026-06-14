package auditlog_test

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_DisabledIsNoOp(t *testing.T) {
	t.Setenv(auditlog.EnvKeyEnabled, "")

	p := mustNew(auditlog.Config{})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideValue(injector, &Database{URL: "postgres://localhost"})
	_ = do.MustInvoke[*Database](injector)

	report := p.Report()
	if report.EventCount != 0 {
		t.Fatalf("expected 0 events when disabled, got %d", report.EventCount)
	}
}

func TestPlugin_EnvVarEnables(t *testing.T) {
	t.Setenv(auditlog.EnvKeyEnabled, "true")

	p := mustNew(auditlog.Config{})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	if report.EventCount == 0 {
		t.Fatal("expected events when env var is set")
	}

	assertServiceCount(t, report, 1)
}

func TestPlugin_EnvVarValues(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"", false},
		{"random", false},
	}
	for _, tc := range tests {
		t.Run(tc.val, func(t *testing.T) {
			t.Setenv(auditlog.EnvKeyEnabled, tc.val)

			p := mustNew(auditlog.Config{})
			injector := do.NewWithOpts(p.Opts())

			provideDB(injector, "db", "test")
			_ = do.MustInvokeNamed[*Database](injector, "db")

			report := p.Report()
			if tc.want && report.EventCount == 0 {
				t.Errorf("env %q: expected events", tc.val)
			}

			if !tc.want && report.EventCount != 0 {
				t.Errorf("env %q: expected no events, got %d", tc.val, report.EventCount)
			}
		})
	}
}

func TestPlugin_ExplicitEnabledOverridesEnv(t *testing.T) {
	t.Setenv(auditlog.EnvKeyEnabled, "")

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	if report.EventCount == 0 {
		t.Fatal("explicit Enabled:true should work even when env is unset")
	}
}

func TestPlugin_ContainerID(t *testing.T) {
	p, injector := newPluginAndInjectorWithID("test-container")

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	events := p.Events()
	if len(events) == 0 {
		t.Fatal("expected events")
	}

	for _, e := range events {
		if e.ContainerID != "test-container" {
			t.Errorf("event %d container_id: want test-container, got %s", e.Sequence, e.ContainerID)
		}
	}
}

func TestPlugin_ReportVersion(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	assertVersion(t, report)
}

func TestNew_RejectsInvalidContainerID(t *testing.T) {
	_, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "has/slash",
	})

	assertErrorExpected(t, err, "ContainerID with path separator")
}

func TestNew_AcceptsValidConfig(t *testing.T) {
	p, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "valid-id",
	})
	if err != nil {
		t.Fatalf("expected no error for valid config, got: %v", err)
	}

	if p == nil {
		t.Fatal("expected non-nil plugin")
	}
}
