package auditlog_test

import (
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestWriteMermaidString_MatchesWriteMermaid(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	str, err := report.WriteMermaidString()
	if err != nil {
		t.Fatalf("WriteMermaidString: %v", err)
	}

	if str == "" {
		t.Fatal("expected non-empty Mermaid output")
	}

	if !strings.Contains(str, "flowchart") {
		t.Errorf("expected Mermaid flowchart, got: %s", str)
	}
}

func TestWritePlantUMLString_MatchesWritePlantUML(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	str, err := report.WritePlantUMLString()
	if err != nil {
		t.Fatalf("WritePlantUMLString: %v", err)
	}

	if str == "" {
		t.Fatal("expected non-empty PlantUML output")
	}
}

func TestWriteDOTString_MatchesWriteDOT(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	str, err := report.WriteDOTString()
	if err != nil {
		t.Fatalf("WriteDOTString: %v", err)
	}

	if str == "" {
		t.Fatal("expected non-empty DOT output")
	}

	if !strings.Contains(str, "digraph") {
		t.Errorf("expected DOT digraph, got: %s", str)
	}
}

func TestWriteD2String_MatchesWriteD2(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	str, err := report.WriteD2String()
	if err != nil {
		t.Fatalf("WriteD2String: %v", err)
	}

	if str == "" {
		t.Fatal("expected non-empty D2 output")
	}
}

func TestWriteHTMLString_MatchesWriteHTML(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	str, err := report.WriteHTMLString()
	if err != nil {
		t.Fatalf("WriteHTMLString: %v", err)
	}

	if str == "" {
		t.Fatal("expected non-empty HTML output")
	}
}
