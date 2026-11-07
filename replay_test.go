package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// --- Replay engine tests ---

func TestReplayEvents_EmptyInput(t *testing.T) {
	t.Parallel()

	_, err := auditlog.ReplayEvents(nil)
	if err == nil {
		t.Fatal("expected error for empty events")
	}
}

func TestReplayEvents_SingleRegistration(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	replayed := replayFromPlugin(t, plugin)

	assertReportServiceCount(t, replayed)
	assertReportValid(t, replayed, "single registration replayed")

	if !replayed.Reconstructed {
		t.Error("expected Reconstructed=true on replayed report")
	}

	if len(replayed.Events) != len(plugin.Report().Events) {
		t.Errorf("event count mismatch")
	}
}

func TestReplayEvents_RegistrationOnly(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")

	report := replayFromPlugin(t, plugin)

	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db service not found")
	}

	assertServiceStatus(t, svc, auditlog.ServiceStatusRegistered)
}

func TestReplayEvents_DependencyChain(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideUserServiceWithDeps(injector, "users", "db", "cache")
	do.MustInvokeNamed[*UserService](injector, "users")

	replayed := replayFromPlugin(t, plugin)

	users := findServiceByName(t, replayed, "users")
	if users == nil {
		t.Fatal("users service not found")
	}

	if len(users.Dependencies) != 2 {
		t.Errorf("users deps: want 2, got %d", len(users.Dependencies))
	}

	db := findServiceByName(t, replayed, "db")
	if db == nil {
		t.Fatal("db service not found")
	}

	if len(db.Dependents) != 1 {
		t.Errorf("db dependents: want 1, got %d", len(db.Dependents))
	}
}

func TestReplayEvents_PreservesContainerID(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjectorWithID("my-container")
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	report := replayFromPlugin(t, plugin)

	assertContainerID(t, report, "my-container")
}

func TestReplayEvents_ContainerIDFromEvents(t *testing.T) {
	t.Parallel()

	events := []auditlog.Event{
		mkRegEvent(1, time.Now(), "svc", "from-event"),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	assertContainerID(t, report, "from-event")
}

func TestReplayEvents_HealthCheckCount(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjectorWithID("test-hc")
	provideHealthyDB(injector, "db", "dsn")
	do.MustInvokeNamed[*HealthyDB](injector, "db")

	hcErrors := plugin.RecordHealthCheck(injector)
	for _, hcErr := range hcErrors {
		if hcErr != nil {
			t.Fatalf("RecordHealthCheck error: %v", hcErr)
		}
	}

	replayed := replayFromPlugin(t, plugin)

	svc := findServiceByName(t, replayed, "db")
	if svc == nil {
		t.Fatal("db not found")
	}

	assertServiceHealthCheckCount(t, svc, 1)
}
