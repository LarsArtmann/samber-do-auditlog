package auditlog_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

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
