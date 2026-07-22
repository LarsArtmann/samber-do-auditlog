package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// buildCLIBinary compiles the auditlog CLI to a temp path and returns it.
func buildCLIBinary(t *testing.T) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "auditlog")

	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}

	return binPath
}

// cliBaseTime is a fixed timestamp shared by every CLI test fixture so the
// event stream is deterministic across runs.
var cliBaseTime = time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

// assertCLIOutputContains fails the test if the CLI output does not contain
// the expected substring. Centralizes the 3-line "if !strings.Contains(out, X)"
// pattern shared by every CLI assertion test.
func assertCLIOutputContains(t *testing.T, label, output, want string) {
	t.Helper()

	if !strings.Contains(output, want) {
		t.Errorf("%s missing %q:\n%s", label, want, output)
	}
}

// mkRegEvent creates a registration-after event for CLI test fixtures.
func mkRegEvent(seq int, ts time.Time, serviceName auditlog.ServiceName, containerID auditlog.ContainerID) auditlog.Event {
	return auditlog.Event{
		ServiceRef: auditlog.ServiceRef{
			ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: serviceName,
		},
		Sequence:    seq,
		Timestamp:   ts,
		EventType:   auditlog.EventTypeRegistration,
		Phase:       auditlog.PhaseAfter,
		ContainerID: containerID,
		ServiceType: auditlog.ProviderTypeLazy,
	}
}

// writeSampleReport builds a deterministic report and writes it as JSON.
func writeSampleReport(t *testing.T, path string, containerID auditlog.ContainerID) {
	t.Helper()

	events := []auditlog.Event{
		mkRegEvent(1, cliBaseTime, "config", containerID),
		mkRegEvent(2, cliBaseTime.Add(time.Millisecond), "db", containerID),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	report.ExportedAt = cliBaseTime

	var buf bytes.Buffer
	if err := report.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func runCLI(t *testing.T, bin string, args ...string) string {
	t.Helper()

	var out bytes.Buffer

	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("auditlog %s: %v\n%s", strings.Join(args, " "), err, out.String())
	}

	return out.String()
}

func TestCLI_Info(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	writeSampleReport(t, reportPath, "cli-test")

	out := runCLI(t, bin, "info", reportPath)

	assertCLIOutputContains(t, "info", out, "services:")
	assertCLIOutputContains(t, "info", out, "config")
}

func TestCLI_InfoFromStdin(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)

	// Build the JSON report in-memory and pipe it via stdin.
	report := writeReportWithExtraService // reuse builder
	_ = report

	var jsonBuf bytes.Buffer

	events := []auditlog.Event{
		mkRegEvent(1, cliBaseTime, "stdin-svc", "stdin-test"),
	}

	r, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatal(err)
	}

	r.ExportedAt = cliBaseTime
	if err := r.WriteJSON(&jsonBuf); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(context.Background(), bin, "info", "-")
	cmd.Stdin = &jsonBuf

	var out bytes.Buffer

	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("info stdin: %v\n%s", err, out.String())
	}

	assertCLIOutputContains(t, "info from stdin", out.String(), "stdin-svc")
}

func TestCLI_Validate(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	writeSampleReport(t, reportPath, "cli-test")

	out := runCLI(t, bin, "validate", reportPath)

	assertCLIOutputContains(t, "validate", out, "OK")
}

func TestCLI_Convert_Formats(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)

	formats := []string{"csv", "tsv", "ndjson", "mermaid", "plantuml", "dot", "d2"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			reportPath := filepath.Join(t.TempDir(), "report.json")
			writeSampleReport(t, reportPath, "cli-convert")

			out := runCLI(t, bin, "convert", reportPath, "-f", format)
			if out == "" {
				t.Errorf("convert %s produced empty output", format)
			}
		})
	}
}

func TestCLI_Convert_HTMLToFile(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	writeSampleReport(t, reportPath, "cli-html")

	outPath := filepath.Join(t.TempDir(), "out.html")
	_ = runCLI(t, bin, "convert", reportPath, "-o", outPath)

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !strings.Contains(strings.ToLower(string(data)), "<!doctype html>") {
		t.Errorf("HTML output missing DOCTYPE")
	}
}

func TestCLI_Diff(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.json")
	bPath := filepath.Join(dir, "b.json")

	writeSampleReport(t, aPath, "diff-a")
	writeReportWithExtraService(t, bPath)

	out := runCLI(t, bin, "diff", aPath, bPath)

	assertCLIOutputContains(t, "diff", out, "added services")
}

func TestCLI_Schema(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)

	out := runCLI(t, bin, "schema")

	var schema map[string]any
	if err := json.Unmarshal([]byte(out), &schema); err != nil {
		t.Fatalf("schema output is not valid JSON: %v", err)
	}

	if schema["$schema"] == nil {
		t.Error("schema output missing $schema")
	}
}

func TestCLI_NoArgs_PrintsUsage(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)

	cmd := exec.CommandContext(context.Background(), bin)

	var out bytes.Buffer

	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit with no args")
	}

	assertCLIOutputContains(t, "usage output", out.String(), "auditlog")
}

func writeReportWithExtraService(t *testing.T, path string) {
	t.Helper()

	events := []auditlog.Event{
		mkRegEvent(1, cliBaseTime, "config", "diff-b"),
		mkRegEvent(2, cliBaseTime.Add(time.Millisecond), "db", "diff-b"),
		mkRegEvent(3, cliBaseTime.Add(2*time.Millisecond), "cache", "diff-b"),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	report.ExportedAt = cliBaseTime

	var buf bytes.Buffer
	if err := report.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
