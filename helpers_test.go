package auditlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// --- Shared test types ---.

// epochTime is the canonical fixture base time used by every test that needs
// a deterministic timestamp (replay, merge, new-report). Centralizes the
// `time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)` literal that previously
// appeared in 6 different test files.
var epochTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// mustNew is a test helper that wraps auditlog.New and panics on error.
// Use in tests where the config is known to be valid.
func mustNew(config auditlog.Config) *auditlog.Plugin {
	p, err := auditlog.New(config)
	if err != nil {
		panic(fmt.Sprintf("auditlog.New failed: %v", err))
	}

	return p
}

type Database struct {
	URL string
}

type Cache struct {
	Entries map[string]string
}

type UserService struct {
	DB    *Database
	Cache *Cache
}
type CrashingService struct{}

var errConnectionReset = errors.New("connection reset")

func (c *CrashingService) Shutdown() error {
	return errConnectionReset
}

type HealthyDB struct {
	DSN string
}

var _ do.Healthchecker = (*HealthyDB)(nil)

func (d *HealthyDB) HealthCheck() error {
	return nil
}

type UnhealthyCache struct {
	Reason string
}

var _ do.HealthcheckerWithContext = (*UnhealthyCache)(nil)

var errCacheUnhealthy = errors.New("cache: unhealthy")

func (c *UnhealthyCache) HealthCheck(_ context.Context) error {
	return errCacheUnhealthy
}

// --- Health check tests ---

type HTTPServer struct {
	Users *UserService
}

type Config struct {
	Port int
}

// --- Provider helpers ---.
func provideDB(injector do.Injector, name, url string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Database, error) {
		return &Database{URL: url}, nil
	})
}

// provideCacheWithSleep is a named *Cache provider kept for test compatibility.
func provideCacheWithSleep(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Cache, error) {
		return &Cache{Entries: make(map[string]string)}, nil
	})
}

// provideUserServiceWithDB is a named *UserService provider that depends on a single *Database.
func provideUserServiceWithDB(
	injector do.Injector,
	name, dbName string,
) {
	do.ProvideNamed(injector, name, func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, dbName)

		return &UserService{DB: db}, nil
	})
}

// provideUserServiceWithDeps is a named *UserService provider that depends on a *Database and *Cache.
//

func provideUserServiceWithDeps(injector do.Injector, name, dbName, cacheName string) {
	do.ProvideNamed(injector, name, func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, dbName)
		cache := do.MustInvokeNamed[*Cache](i, cacheName)

		return &UserService{DB: db, Cache: cache}, nil
	})
}

// provideHTTPServerWithUsers is a named *HTTPServer provider that depends on a *UserService.
func provideHTTPServerWithUsers(injector do.Injector, name, usersName string) {
	do.ProvideNamed(injector, name, func(i do.Injector) (*HTTPServer, error) {
		users := do.MustInvokeNamed[*UserService](i, usersName)

		return &HTTPServer{Users: users}, nil
	})
}

func provideHealthyDB(injector do.Injector, name, dsn string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: dsn}, nil
	})
}

// provideString is a named *string* provider that returns the given value.
// Used in example/fuzz tests where the exact string value is the test data.
func provideString(injector do.Injector, name, value string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (string, error) {
		return value, nil
	})
}

func provideUnhealthyCache(injector do.Injector, name, reason string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*UnhealthyCache, error) {
		return &UnhealthyCache{Reason: reason}, nil
	})
}

func provideFailing(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Database, error) {
		return nil, os.ErrNotExist
	})
}

func provideCache(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Cache, error) {
		return &Cache{}, nil
	})
}

func provideCrashing(injector do.Injector, name string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*CrashingService, error) {
		return &CrashingService{}, nil
	})
}

func findServiceByName(t *testing.T, report auditlog.Report, name auditlog.ServiceName) *auditlog.ServiceInfo {
	t.Helper()

	for i := range report.Services {
		if report.Services[i].ServiceName == name {
			return &report.Services[i]
		}
	}

	return nil
}

func findServiceBySuffix(t *testing.T, report auditlog.Report, suffix string) *auditlog.ServiceInfo {
	t.Helper()

	for i := range report.Services {
		if strings.HasSuffix(string(report.Services[i].ServiceName), suffix) {
			return &report.Services[i]
		}
	}

	return nil
}

func assertVersion(t *testing.T, report auditlog.Report) {
	t.Helper()

	if report.Version != auditlog.SchemaVersion {
		t.Errorf("version: want %s, got %s", auditlog.SchemaVersion, report.Version)
	}
}

// assertAllEventsOfType fails the test if any event in the slice does not match the expected type.
func assertAllEventsOfType(t *testing.T, events []auditlog.Event, expected auditlog.EventType) {
	t.Helper()

	for _, evt := range events {
		if evt.EventType != expected {
			t.Errorf("expected %s event, got %s", expected, evt.EventType)
		}
	}
}

// assertAllEventsForService fails the test if any event in the slice is not for the given service.
func assertAllEventsForService(t *testing.T, events []auditlog.Event, serviceName auditlog.ServiceName) {
	t.Helper()

	for _, evt := range events {
		assertEqual(t, "service name", evt.ServiceName, serviceName)
	}
}

// assertDependenciesCount fails the test if the service's dependency count does not match.
func assertDependenciesCount(t *testing.T, svc *auditlog.ServiceInfo, want int) {
	t.Helper()

	assertServiceIntField(t, svc, "dependencies count", len(svc.Dependencies), want)
}

// assertHTMLContains fails the test if the HTML does not contain the expected substring.
func assertHTMLContains(t *testing.T, html, want string) {
	t.Helper()

	if !strings.Contains(html, want) {
		t.Errorf("expected %q in HTML output", want)
	}
}

// assertStringContains fails the test if the string does not contain the expected substring.
func assertStringContains(t *testing.T, s, want string) {
	t.Helper()

	if !strings.Contains(s, want) {
		t.Errorf("expected %q in output", want)
	}
}

// assertOutputContains fails the test if output does not contain want. Unlike
// assertStringContains, it prints the full output in the error message — used
// by tree/table/HTML export tests where the buffer content is the only signal.
func assertOutputContains(t *testing.T, label, output, want string) {
	t.Helper()

	if !strings.Contains(output, want) {
		t.Errorf("%s missing %q:\n%s", label, want, output)
	}
}

// assertEqual fails the test if got != want for any comparable type.
// The fieldName is used in the error message.
func assertEqual[T comparable](t *testing.T, fieldName string, got, want T) {
	t.Helper()

	if got != want {
		t.Errorf("%s: want %v, got %v", fieldName, want, got)
	}
}

// assertServiceCount fails the test if the report's service count does not match.
func assertServiceCount(t *testing.T, report auditlog.Report, want int) {
	t.Helper()

	assertEqual(t, "service_count", report.ServiceCount, want)
}

// assertEventCount fails the test if the report's event count does not match.
func assertEventCount(t *testing.T, report auditlog.Report, want int) {
	t.Helper()

	assertEqual(t, "event_count", report.EventCount, want)
}

// assertContainerID fails the test if the report's container_id does not match.
func assertContainerID(t *testing.T, report auditlog.Report, want auditlog.ContainerID) {
	t.Helper()

	assertEqual(t, "container_id", report.ContainerID, want)
}

// assertServiceInvocationCount fails the test if the service's invocation count does not match.
func assertServiceInvocationCount(t *testing.T, svc *auditlog.ServiceInfo, want int) {
	t.Helper()

	assertServiceIntField(t, svc, "invocation_count", svc.InvocationCount, want)
}

// assertServiceStatus fails the test if the service's status does not match.
func assertServiceStatus(t *testing.T, svc *auditlog.ServiceInfo, want auditlog.ServiceStatus) {
	t.Helper()

	if svc.Status != want {
		t.Errorf("status: want %q, got %q", want, svc.Status)
	}
}

// assertServiceHealthCheckCount fails the test if the service's health check count does not match.
func assertServiceHealthCheckCount(t *testing.T, svc *auditlog.ServiceInfo, want int) {
	t.Helper()

	assertServiceIntField(t, svc, "health_check_count", svc.HealthCheckCount, want)
}

// assertServiceIntField is the shared nil-checked assertion for ServiceInfo count fields.
func assertServiceIntField(t *testing.T, svc *auditlog.ServiceInfo, fieldName string, got, want int) {
	t.Helper()

	if svc == nil {
		t.Errorf("%s: want %d, got <nil service>", fieldName, want)

		return
	}

	assertEqual(t, fieldName, got, want)
}

// assertFilteredServiceCount fails the test unless filtered has exactly one matching service
// with the given name.
func assertFilteredServiceCount(t *testing.T, filtered auditlog.Report, wantName auditlog.ServiceName) {
	t.Helper()

	requireOneService(t, "", filtered.Services)

	assertEqual(t, "service name", filtered.Services[0].ServiceName, wantName)
}

// assertReportServiceCount fails the test (with Fatalf) if the report's services slice
// does not have exactly one element. The test must not continue when this fails.
func assertReportServiceCount(t *testing.T, report auditlog.Report) {
	t.Helper()

	requireOneService(t, "", report.Services)
}

// assertReportValid fails the test (with Fatalf) if the report fails validation.
// wantValidMsg describes the scenario (e.g. "empty", "with scopes+health") and is
// interpolated into the error message so test output identifies which scenario broke.
func assertReportValid(t *testing.T, report auditlog.Report, wantValidMsg string) {
	t.Helper()

	if err := report.Validate(); err != nil {
		t.Fatalf("expected valid %s report, got: %v", wantValidMsg, err)
	}
}

// assertReportValidNoFatal is like assertReportValid but uses Errorf so the
// caller continues checking additional reports (used inside fuzz loops where
// each iteration is independent).
func assertReportValidNoFatal(t *testing.T, report auditlog.Report, wantValidMsg string) {
	t.Helper()

	if err := report.Validate(); err != nil {
		t.Errorf("expected valid %s report, got: %v", wantValidMsg, err)
	}
}

// requireOneService fails the test (with Fatalf) if the services slice does not have
// exactly one element. label describes what the slice represents (e.g. "child",
// "failed", "unhealthy", "eager") and is interpolated into the error message.
func requireOneService(t *testing.T, label string, services []auditlog.ServiceInfo) {
	t.Helper()

	if len(services) != 1 {
		t.Fatalf("expected 1 %s service, got %d", label, len(services))
	}
}

// assertUnhealthyServiceCount fails the test (with Fatalf) if the slice does not have
// exactly one unhealthy service, then checks the name of that service.
func assertUnhealthyServiceCount(t *testing.T, unhealthy []auditlog.ServiceInfo, wantName auditlog.ServiceName) {
	t.Helper()

	requireOneService(t, "unhealthy", unhealthy)

	assertEqual(t, "unhealthy service", unhealthy[0].ServiceName, wantName)
}

// assertErrorExpected fails the test if err is nil.
func assertErrorExpected(t *testing.T, err error, reason string) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error %s", reason)
	}
}

// unmarshalJSONForTest unmarshals data into out, failing the test on error.
// The op string is used in the failure message (e.g. "unmarshal", "Unmarshal").
func unmarshalJSONForTest(t *testing.T, data []byte, out any, op string) {
	t.Helper()

	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("%s: %v", op, err)
	}
}

func newPluginAndInjector() (*auditlog.Plugin, do.Injector) { //nolint:ireturn
	p := mustNew(auditlog.Config{Enabled: true})

	return p, do.NewWithOpts(p.Opts())
}

func newPluginAndInjectorWithID(containerID auditlog.ContainerID) (*auditlog.Plugin, do.Injector) { //nolint:ireturn
	p := mustNew(auditlog.Config{Enabled: true, ContainerID: containerID})

	return p, do.NewWithOpts(p.Opts())
}

// setupWithDB returns a plugin and injector with a single *Database registered
// under "db" and already invoked. The standard 4-line "register + invoke"
// preamble that opens most plugin-level tests.
func setupWithDB(url string) (*auditlog.Plugin, do.Injector) { //nolint:ireturn
	p, injector := newPluginAndInjector()
	provideDB(injector, "db", url)
	_ = do.MustInvokeNamed[*Database](injector, "db")

	return p, injector
}

// setupWithDBReport returns the populated Report from a fresh plugin with a
// single *Database invoked — the 5-line preamble (`mustNew + NewWithOpts +
// provideDB + invoke + Report()`) shared by every "build a populated report
// and assert on it" test.
func setupWithDBReport() auditlog.Report {
	p, _ := setupWithDB("test")

	return p.Report()
}

// writeHTMLToString runs the standard "plugin + provideDB + invoke + WriteHTML"
// pipeline and returns the rendered HTML string. Fails the test on any
// pipeline error. Centralizes the 7-line preamble shared by every HTML
// rendering test.
func writeHTMLToString(t *testing.T) string {
	t.Helper()

	p, _ := setupWithDB("postgres://localhost")

	var buf bytes.Buffer

	if err := p.WriteHTML(&buf); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	return buf.String()
}

// replayFromPlugin drives the standard round-trip: writes plugin events to
// NDJSON, reads them back, and replays them into a Report. Centralizes the
// 8-line preamble that opens every replay round-trip test.
func replayFromPlugin(t *testing.T, p *auditlog.Plugin) auditlog.Report {
	t.Helper()

	var buf bytes.Buffer

	if err := p.WriteEventsNDJSON(&buf); err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	return report
}

// assertWriteFails expects a non-nil error from fn writing to a failingWriter.
// Centralizes the "failingWriter error path" assertion used by every plugin
// write-method test.
func assertWriteFails(t *testing.T, label string, fn func(io.Writer) error) {
	t.Helper()

	if err := fn(failingWriter{}); err == nil {
		t.Fatalf("%s: expected error from failing writer", label)
	}
}

// assertErrIs fails the test if err does not match want via errors.Is. Shared
// by every "should return sentinel error" test in the loader/reader paths.
func assertErrIs(t *testing.T, err, want error, label string) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Errorf("%s: want %v, got %v", label, want, err)
	}
}

// assertLen fails the test if the slice length does not equal want.
func assertLen[T any](t *testing.T, label string, slice []T, want int) {
	t.Helper()

	if len(slice) != want {
		t.Errorf("%s length: want %d, got %d", label, want, len(slice))
	}
}

func newPluginWithCapture() (*auditlog.Plugin, *[]auditlog.Event, do.Injector) { //nolint:ireturn
	var captured []auditlog.Event

	p := mustNew(auditlog.Config{
		Enabled: true,
		OnEvent: func(e auditlog.Event) {
			captured = append(captured, e)
		},
	})

	return p, &captured, do.NewWithOpts(p.Opts())
}

// --- Writer error types ---

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

// --- Error sentinels for writer tests ---

var (
	errWriteFailed       = errors.New("write failed")
	errConnectionRefused = errors.New("connection refused")
)
