package auditlog_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// reportWithDuplicateEdges builds a report where svc-a depends on svc-b twice.
// Used by duplicate-edge tests to verify deduplication across diagram formats.
func reportWithDuplicateEdges() auditlog.Report {
	return auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef:   rootRef("svc-a"),
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
				Dependencies: []auditlog.ServiceRef{
					rootRef("svc-b"),
					rootRef("svc-b"),
				},
			},
			{
				ServiceRef:   rootRef("svc-b"),
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
			},
		},
	}
}

// assertSingleEdge counts occurrences of edgeMarker in the output of writeFn and
// fails the test if the count is not exactly 1. Shared by duplicate-edge tests to
// eliminate duplication flagged by the dupl linter.
func assertSingleEdge(t *testing.T, edgeMarker string, writeFn func(io.Writer) error) {
	t.Helper()

	var buf bytes.Buffer

	err := writeFn(&buf)
	if err != nil {
		t.Fatalf("write diagram: %v", err)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")
	edgeCount := 0

	for _, line := range lines {
		if strings.Contains(line, edgeMarker) {
			edgeCount++
		}
	}

	if edgeCount != 1 {
		t.Errorf("expected exactly 1 deduplicated edge, got %d", edgeCount)
	}
}

func TestWriteMermaid_DuplicateEdges(t *testing.T) {
	t.Parallel()

	report := reportWithDuplicateEdges()

	assertSingleEdge(t, "root_svc_a --> root_svc_b", func(w io.Writer) error { return report.WriteMermaid(w) })
}

func TestWriteMermaid_ExternalDependency(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer

	err := report.WriteMermaid(&buf)
	if err != nil {
		t.Fatalf("WriteMermaid: %v", err)
	}

	output := buf.String()
	assertStringContains(t, output, "flowchart TD")
	assertStringContains(t, output, "external-dep")
	assertStringContains(t, output, "-->")
}

func TestWriteMermaid_EscapesSpecialChars(t *testing.T) {
	t.Parallel()

	report := reportWithSpecialCharService()

	var buf bytes.Buffer

	err := report.WriteMermaid(&buf)
	if err != nil {
		t.Fatalf("WriteMermaid: %v", err)
	}

	output := buf.String()
	// The label must be escaped (] -> ), " -> ') and the identifier sanitized
	// so the node syntax id[label] stays balanced and valid.
	assertStringContains(t, output, "root_evil_svc[evil)'svc]")

	// No raw double-quote should leak into the rendered Mermaid: the theme
	// header uses single quotes, so any " indicates an unescaped label.
	if strings.Contains(output, `"`) {
		t.Errorf("mermaid output must not contain unescaped double quotes: %s", output)
	}
}

func TestWritePlantUML_EscapesSpecialChars(t *testing.T) {
	t.Parallel()

	report := reportWithSpecialCharService()

	var buf bytes.Buffer

	err := report.WritePlantUML(&buf)
	if err != nil {
		t.Fatalf("WritePlantUML: %v", err)
	}

	output := buf.String()
	// go-output renders PlantUML nodes in bracket notation [label] as id. The
	// hostile characters in the service name are escaped so the brackets stay
	// balanced: ] -> \] (prevents early bracket close) and " -> \".
	assertStringContains(t, output, `[evil\]\"svc] as root_evil_svc`)
}

// reportWithSpecialCharService builds a report containing a single service
// whose name contains characters that are hostile to diagram syntax (brackets,
// quotes, brackets). Shared by the Mermaid and PlantUML escaping tests so both
// formats are verified against the same fixture.
func reportWithSpecialCharService() auditlog.Report {
	return auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef:   rootRef(`evil]"svc`),
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
			},
		},
	}
}
