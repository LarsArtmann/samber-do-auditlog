package auditlog_test

import (
	"bytes"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestReport_WriteCSV_FailingWriter(t *testing.T) {
	t.Parallel()

	report := activeSvcReport("csv-err", "svc")

	err := report.WriteCSV(errWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
}

func TestReport_WriteCSV_WithErrors(t *testing.T) {
	t.Parallel()

	invErr := "connection refused"
	shutdownErr := "timeout"
	hcErr := "unhealthy"

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "csv-errors",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef:       rootRef("failing-svc"),
				Status:           auditlog.ServiceStatusInvocationError,
				InvocationError:  &invErr,
				ShutdownError:    &shutdownErr,
				HealthCheckError: &hcErr,
				HealthCheckCount: 3,
			},
		},
	}

	var buf bytes.Buffer

	err := report.WriteCSV(&buf)
	if err != nil {
		t.Fatalf("WriteCSV error: %v", err)
	}

	out := buf.String()
	assertOutputContains(t, "csv-errors", out, "connection refused")
	assertOutputContains(t, "csv-errors", out, "timeout")
	assertOutputContains(t, "csv-errors", out, "unhealthy")
}

func TestReport_WriteHTMLTree_FailingWriter(t *testing.T) {
	t.Parallel()

	report := activeSvcReport("htmltree-err", "svc")

	err := report.WriteHTMLTree(errWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
}

func TestReport_WriteTree_EmptyContainerID(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version: auditlog.SchemaVersion,
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: rootRef("svc"),
				Status:     auditlog.ServiceStatusRegistered,
			},
		},
	}

	var buf bytes.Buffer

	err := report.WriteTree(&buf)
	if err != nil {
		t.Fatalf("WriteTree error: %v", err)
	}

	out := buf.String()
	assertOutputContains(t, "tree-empty-cid", out, "container")
	assertOutputContains(t, "tree-empty-cid", out, "svc")
}

func TestReport_WriteTree_DanglingDependent(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "dangling-test",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: rootRef("root-svc"),
				Status:     auditlog.ServiceStatusRegistered,
				Dependents: []auditlog.ServiceRef{
					rootRef("nonexistent-service"),
				},
			},
		},
	}

	var buf bytes.Buffer

	err := report.WriteTree(&buf)
	if err != nil {
		t.Fatalf("WriteTree error: %v", err)
	}

	// Should still render without crashing.
	out := buf.String()
	assertOutputContains(t, "dangling", out, "root-svc")
}
