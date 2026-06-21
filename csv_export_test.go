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

	lines := requireMinCSVRows(t, buf.String(), 2)

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

	assertStringContains(t, output, "db")
	assertStringContains(t, output, "active")
	assertStringContains(t, output, "lazy")
}

func TestReport_WriteCSV_EmptyReport(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{}

	var buf bytes.Buffer

	err := report.WriteCSV(&buf)
	if err != nil {
		t.Fatalf("WriteCSV on empty report failed: %v", err)
	}

	_ = requireExactCSVRows(t, buf.String(), 1)
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

	lines := requireMinCSVRows(t, output, 2)

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

	assertStringContains(t, output, "config")
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
				ServiceRef:   csvServiceRef("bare-svc"),
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

	lines := requireExactCSVRows(t, buf.String(), 2)

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

	configRef := csvServiceRef("config")
	dbRef := csvServiceRef("db")

	return auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test-container",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef:      configRef,
				Status:          auditlog.ServiceStatusActive,
				ServiceType:     auditlog.ProviderTypeLazy,
				RegisteredAt:    registeredAt,
				FirstInvokedAt:  &invokedAt,
				InvocationCount: 1,
			},
			{
				ServiceRef:           dbRef,
				Status:               auditlog.ServiceStatusActive,
				ServiceType:          auditlog.ProviderTypeLazy,
				RegisteredAt:         registeredAt,
				FirstInvokedAt:       &invokedAt,
				InvocationCount:      1,
				FirstBuildDurationMs: &buildMs,
				Dependencies:         []auditlog.ServiceRef{configRef},
			},
		},
	}
}

// csvServiceRef is a 3-line constructor for the empty-scope ServiceRef that
// every CSV test fixture shares (ScopeID/ScopeName intentionally blank — the
// CSV export tests the nil-pointer render path).
func csvServiceRef(name string) auditlog.ServiceRef {
	return auditlog.ServiceRef{
		ScopeID:     "",
		ScopeName:   "",
		ServiceName: name,
	}
}

// csvSplitLines splits a CSV/TSV buffer into non-trailing-empty lines. Centralizes
// the 1-line preamble shared by every "expected N rows" assertion in CSV tests.
func csvSplitLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

// ndjsonLines splits an NDJSON buffer into non-trailing-whitespace lines.
// Centralizes the 1-line preamble shared by every NDJSON line-iteration test.
func ndjsonLines(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}

// requireMinCSVRows fails the test (with Fatalf) unless the output contains at
// least minLines lines (including the header). Returns the split lines so the
// caller can inspect the header / row contents.
func requireMinCSVRows(t *testing.T, output string, minLines int) []string {
	t.Helper()

	lines := csvSplitLines(output)
	if len(lines) < minLines {
		t.Fatalf("expected at least %d lines, got %d", minLines, len(lines))
	}

	return lines
}

// requireExactCSVRows fails the test (with Fatalf) unless the output contains
// exactly wantLines lines. Returns the split lines so the caller can inspect
// the header / row contents.
func requireExactCSVRows(t *testing.T, output string, wantLines int) []string {
	t.Helper()

	lines := csvSplitLines(output)
	if len(lines) != wantLines {
		t.Fatalf("expected %d lines, got %d", wantLines, len(lines))
	}

	return lines
}
