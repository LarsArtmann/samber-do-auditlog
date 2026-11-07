package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

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

// runCLIExpectError runs the CLI expecting a non-zero exit and returns the
// combined output for message assertions.
func runCLIExpectError(t *testing.T, bin string, args ...string) string {
	t.Helper()

	var out bytes.Buffer

	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err == nil {
		t.Fatalf("auditlog %s: expected non-zero exit, got success\n%s", strings.Join(args, " "), out.String())
	}

	return out.String()
}

// writeSampleNDJSONReport writes a deterministic event stream as NDJSON under
// path — deliberately usable with a .json extension to prove --input-format
// overrides auto-detection.
func writeSampleNDJSONReport(t *testing.T, path string, containerID auditlog.ContainerID) {
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
	if err := report.WriteNDJSON(&buf); err != nil {
		t.Fatalf("WriteNDJSON: %v", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestCLI_InputFormatFlag(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)
	// NDJSON content behind a .json name: only --input-format ndjson loads it.
	reportPath := filepath.Join(t.TempDir(), "misleading.json")
	writeSampleNDJSONReport(t, reportPath, "fmt-test")

	out := runCLI(t, bin, "info", "--input-format", "ndjson", reportPath)
	assertCLIOutputContains(t, "info --input-format ndjson", out, "services:")

	errOut := runCLIExpectError(t, bin, "info", "--input-format", "yaml", reportPath)
	assertCLIOutputContains(t, "invalid --input-format", errOut, "want: auto, json, ndjson")
}

func TestCLI_VerboseFlag(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	writeSampleReport(t, reportPath, "verbose-test")

	out := runCLI(t, bin, "validate", "--verbose", reportPath)
	assertCLIOutputContains(t, "validate --verbose", out, "loaded")
	assertCLIOutputContains(t, "validate --verbose", out, "format=json")
}

func TestCLI_QuietFlag(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	writeSampleReport(t, reportPath, "quiet-test")

	out := runCLI(t, bin, "validate", "--quiet", reportPath)
	if strings.Contains(out, "OK:") {
		t.Errorf("validate --quiet still printed the OK line:\n%s", out)
	}
}

func TestCLI_VerboseQuietConflict(t *testing.T) {
	t.Parallel()

	bin := buildCLIBinary(t)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	writeSampleReport(t, reportPath, "conflict-test")

	out := runCLIExpectError(t, bin, "validate", "--verbose", "--quiet", reportPath)
	assertCLIOutputContains(t, "verbose+quiet", out, "mutually exclusive")
}
