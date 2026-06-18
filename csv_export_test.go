package auditlog_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestReport_WriteCSV_HeaderAndRows(t *testing.T) {
	t.Parallel()

	report := buildCSVTestReport()

	var buf bytes.Buffer

	err := report.WriteCSV(&buf)
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least header + 1 row, got %d lines", len(lines))
	}

	expectedCols := []string{
		"scope_id", "scope_name", "service_name",
		"status", "service_type",
	}
	for _, col := range expectedCols {
		if !strings.Contains(lines[0], col) {
			t.Errorf("header missing column %q. got: %s", col, lines[0])
		}
	}
}

func TestReport_WriteCSV_ServiceData(t *testing.T) {
	t.Parallel()

	report := buildCSVTestReport()

	var buf bytes.Buffer

	err := report.WriteCSV(&buf)
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "db") {
		t.Errorf("CSV output missing service name 'db'. got:\n%s", output)
	}

	if !strings.Contains(output, "active") {
		t.Errorf("CSV output missing status 'active'. got:\n%s", output)
	}

	if !strings.Contains(output, "lazy") {
		t.Errorf("CSV output missing service type 'lazy'. got:\n%s", output)
	}
}

func TestReport_WriteCSV_EmptyReport(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{}

	var buf bytes.Buffer

	err := report.WriteCSV(&buf)
	if err != nil {
		t.Fatalf("WriteCSV on empty report failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected only header row for empty report, got %d lines", len(lines))
	}
}

func TestReport_WriteTSV_TabDelimited(t *testing.T) {
	t.Parallel()

	report := buildCSVTestReport()

	var buf bytes.Buffer

	err := report.WriteTSV(&buf)
	if err != nil {
		t.Fatalf("WriteTSV failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "\t") {
		t.Errorf("TSV output should contain tabs. got:\n%s", output)
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least header + 1 row, got %d lines", len(lines))
	}

	firstLineFields := strings.Split(lines[0], "\t")
	if len(firstLineFields) < 10 {
		t.Errorf("expected at least 10 tab-separated columns in header, got %d", len(firstLineFields))
	}
}

func TestReport_WriteCSV_DependenciesFormatted(t *testing.T) {
	t.Parallel()

	report := buildCSVTestReport()

	var buf bytes.Buffer

	err := report.WriteCSV(&buf)
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "config") {
		t.Errorf("CSV output should contain dependency 'config'. got:\n%s", output)
	}
}

func TestReport_WriteCSV_NilPointersEmpty(t *testing.T) {
	t.Parallel()

	registeredAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test-container",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{
					ScopeID:     "",
					ScopeName:   "",
					ServiceName: "bare-svc",
				},
				Status:       auditlog.ServiceStatusRegistered,
				ServiceType:  auditlog.ProviderTypeLazy,
				RegisteredAt: registeredAt,
			},
		},
	}

	var buf bytes.Buffer

	err := report.WriteCSV(&buf)
	if err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 row, got %d lines", len(lines))
	}

	// The row should not contain "nil" or "<nil>" — nil pointers render as empty.
	if strings.Contains(lines[1], "nil") {
		t.Errorf("row should not contain literal 'nil'. got: %s", lines[1])
	}
}

// buildCSVTestReport creates a report with known service data for CSV testing.
func buildCSVTestReport() auditlog.Report {
	registeredAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	invokedAt := time.Date(2026, 1, 1, 12, 0, 1, 0, time.UTC)
	buildMs := 15.5

	return auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test-container",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{
					ScopeID:     "",
					ScopeName:   "",
					ServiceName: "config",
				},
				Status:          auditlog.ServiceStatusActive,
				ServiceType:     auditlog.ProviderTypeLazy,
				RegisteredAt:    registeredAt,
				FirstInvokedAt:  &invokedAt,
				InvocationCount: 1,
			},
			{
				ServiceRef: auditlog.ServiceRef{
					ScopeID:     "",
					ScopeName:   "",
					ServiceName: "db",
				},
				Status:               auditlog.ServiceStatusActive,
				ServiceType:          auditlog.ProviderTypeLazy,
				RegisteredAt:         registeredAt,
				FirstInvokedAt:       &invokedAt,
				InvocationCount:      1,
				FirstBuildDurationMs: &buildMs,
				Dependencies: []auditlog.ServiceRef{
					{
						ScopeName:   "",
						ServiceName: "config",
					},
				},
			},
		},
	}
}
