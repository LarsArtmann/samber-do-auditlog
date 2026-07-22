package auditlog_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// goldenExportedAt is a fixed timestamp so the rendered HTML is byte-for-byte
// reproducible across runs and machines. Every time-based field in the golden
// report derives from this or from the fixed event timestamps below.
var goldenExportedAt = time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC)

// Package-level duration values so we can take their address without a wrapper
// helper (avoids modernize false-positives on inline pointer factories).
var (
	goldenConfigBuildMs float64 = 1.2
	goldenDBBuildMs     float64 = 2.4
)

// goldenEvent builds a root-scope event pinned to the golden container/timestamp
// baseline. Shorter than the shared mkEvent so lines stay under the golines limit.
func goldenEvent(
	seq int,
	offsetMs time.Duration,
	eventType auditlog.EventType,
	phase auditlog.Phase,
	name string,
) auditlog.Event {
	return auditlog.Event{
		ServiceRef:  rootRef(auditlog.ServiceName(name)),
		Sequence:    seq,
		Timestamp:   goldenExportedAt.Add(offsetMs),
		EventType:   eventType,
		Phase:       phase,
		ContainerID: "golden",
		ServiceType: auditlog.ProviderTypeLazy,
	}
}

// goldenReport builds a deterministic, valid Report via ReplayEvents from a
// fixed event stream, then pins ExportedAt so the output is stable. The report
// has two root-scope services (config, db) where db depends on config, plus a
// shutdown pair — exercising services, events, timeline and graph tabs.
func goldenReport(t *testing.T) auditlog.Report {
	t.Helper()

	events := []auditlog.Event{
		goldenEvent(1, 0, auditlog.EventTypeRegistration, auditlog.PhaseAfter, "config"),
		goldenEvent(2, 1*time.Millisecond, auditlog.EventTypeRegistration, auditlog.PhaseAfter, "db"),
		goldenEvent(3, 2*time.Millisecond, auditlog.EventTypeInvocation, auditlog.PhaseBefore, "db"),
		goldenEvent(4, 3*time.Millisecond, auditlog.EventTypeInvocation, auditlog.PhaseBefore, "config"),
		goldenEvent(5, 4*time.Millisecond, auditlog.EventTypeInvocation, auditlog.PhaseAfter, "config"),
		goldenEvent(6, 5*time.Millisecond, auditlog.EventTypeInvocation, auditlog.PhaseAfter, "db"),
		goldenEvent(7, 6*time.Millisecond, auditlog.EventTypeShutdown, auditlog.PhaseBefore, "db"),
		goldenEvent(8, 7*time.Millisecond, auditlog.EventTypeShutdown, auditlog.PhaseAfter, "db"),
	}
	// Mark the invocation-after events with durations for richer timeline output.
	events[4].DurationMs = &goldenConfigBuildMs
	events[5].DurationMs = &goldenDBBuildMs

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	// Pin the timestamp so the golden file is byte-stable.
	report.ExportedAt = goldenExportedAt

	assertReportValid(t, report, "golden")

	return report
}

// TestReport_WriteHTML_GoldenFile renders the deterministic golden report to
// HTML and compares it against the committed golden file. Run with
// UPDATE_GOLDEN=1 to regenerate testdata/golden/report.html.
func TestReport_WriteHTML_GoldenFile(t *testing.T) {
	t.Parallel()

	report := goldenReport(t)

	var buf bytes.Buffer

	if err := report.WriteHTML(&buf); err != nil {
		t.Fatalf("WriteHTML: %v", err)
	}

	got := buf.Bytes()
	goldenPath := filepath.Join("testdata", "golden", "report.html")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		t.Skipf("golden file updated: %s", goldenPath)

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file (%s): %v\n"+
			"hint: run UPDATE_GOLDEN=1 go test -run %s to create it",
			goldenPath, err, t.Name())
	}

	if !bytes.Equal(got, want) {
		diffPath := filepath.Join(t.TempDir(), "report.actual.html")
		if err := os.WriteFile(diffPath, got, 0o644); err != nil {
			t.Fatalf("write actual: %v", err)
		}

		t.Errorf("HTML output does not match golden file.\n"+
			"  golden: %s (%d bytes)\n"+
			"  actual: %s (%d bytes)\n"+
			"hint: run UPDATE_GOLDEN=1 go test -run TestReport_WriteHTML_GoldenFile to update",
			goldenPath, len(want), diffPath, len(got))
	}
}
