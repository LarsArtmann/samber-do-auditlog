package auditlog_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// goldenExportedAt is a fixed timestamp so the rendered HTML is reproducible
// across runs and machines. Every time-based field in the golden report
// derives from this or from the fixed event timestamps below.
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

	// Pin the timestamp so the output is byte-stable.
	report.ExportedAt = goldenExportedAt

	assertReportValid(t, report, "golden")

	return report
}

// goldenHTML renders the deterministic golden report to HTML.
func goldenHTML(t *testing.T) string {
	t.Helper()

	var buf bytes.Buffer

	if err := goldenReport(t).WriteHTML(&buf); err != nil {
		t.Fatalf("WriteHTML: %v", err)
	}

	return buf.String()
}

// TestReport_WriteHTML_Structure asserts the structural contract of the
// self-contained HTML report: all five tabs, the shared design tokens, the
// embedded Mermaid graph, the CSP meta, and the per-section data. Replaces
// the master branch's byte-for-byte golden-file test — the html/template
// renderer is intentionally self-contained (no templ runtime on the Go 1.23
// branch).
func TestReport_WriteHTML_Structure(t *testing.T) {
	t.Parallel()

	html := goldenHTML(t)

	structural := []string{
		"<!DOCTYPE html>",
		"Content-Security-Policy",
		"<title>do-auditlog — golden</title>",
		"data-tab=\"services\"",
		"data-tab=\"scopes\"",
		"data-tab=\"graph\"",
		"data-tab=\"timeline\"",
		"data-tab=\"events\"",
		"id=\"services-tbody\"",
		"id=\"events-tbody\"",
		">config<",
		">db<",
		"flowchart TD",
		"timeline-bar build",
		"timeline-bar shutdown",
		"schema v" + auditlog.SchemaVersion,
		"--accent: #e8a838",
		"skip-link",
		"id=\"service-search\"",
		"id=\"svc-errors-only\"",
		"class=\"chip event-chip active\"",
	}

	for _, want := range structural {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in HTML report", want)
		}
	}
}

// TestReport_WriteHTMLString_MatchesWriter verifies the string convenience
// wrapper produces identical output to the writer entry point.
func TestReport_WriteHTMLString_MatchesWriter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := goldenReport(t).WriteHTML(&buf); err != nil {
		t.Fatalf("WriteHTML: %v", err)
	}

	got, err := goldenReport(t).WriteHTMLString()
	if err != nil {
		t.Fatalf("WriteHTMLString: %v", err)
	}

	if got != buf.String() {
		t.Errorf("WriteHTMLString diverges from WriteHTML output")
	}
}
