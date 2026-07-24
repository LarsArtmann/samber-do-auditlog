package auditlog_test

import (
	"bytes"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/go-output"
)

func TestDiagram_MermaidDefaultDirectionTD(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer
	err := report.WriteMermaid(&buf)
	if err != nil {
		t.Fatalf("WriteMermaid: %v", err)
	}

	if !strings.Contains(buf.String(), "flowchart TD") {
		t.Errorf("expected 'flowchart TD' in output, got:\n%s", buf.String())
	}
}

func TestDiagram_MermaidDirectionRight(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer
	err := report.WriteMermaid(&buf, auditlog.WithDirection(output.DirectionRight))
	if err != nil {
		t.Fatalf("WriteMermaid: %v", err)
	}

	if !strings.Contains(buf.String(), "flowchart LR") {
		t.Errorf("expected 'flowchart LR' in output, got:\n%s", buf.String())
	}

	if strings.Contains(buf.String(), "flowchart TD") {
		t.Errorf("did not expect 'flowchart TD' in output")
	}
}

func TestDiagram_DOTDefaultDirectionLR(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer
	err := report.WriteDOT(&buf)
	if err != nil {
		t.Fatalf("WriteDOT: %v", err)
	}

	if !strings.Contains(buf.String(), "rankdir=LR") {
		t.Errorf("expected 'rankdir=LR' in output, got:\n%s", buf.String())
	}
}

func TestDiagram_DOTDirectionDown(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer
	err := report.WriteDOT(&buf, auditlog.WithDirection(output.DirectionUp))
	if err != nil {
		t.Fatalf("WriteDOT: %v", err)
	}

	if !strings.Contains(buf.String(), "rankdir=BT") {
		t.Errorf("expected 'rankdir=BT' in output, got:\n%s", buf.String())
	}
}

func TestDiagram_D2DirectionRight(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer
	err := report.WriteD2(&buf, auditlog.WithDirection(output.DirectionRight))
	if err != nil {
		t.Fatalf("WriteD2: %v", err)
	}

	if !strings.Contains(buf.String(), "direction: right") {
		t.Errorf("expected 'direction: right' in output, got:\n%s", buf.String())
	}
}

func TestDiagram_D2DefaultNoDirection(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer
	err := report.WriteD2(&buf)
	if err != nil {
		t.Fatalf("WriteD2: %v", err)
	}

	if strings.Contains(buf.String(), "direction:") {
		t.Errorf("did not expect explicit direction in default output, got:\n%s", buf.String())
	}
}

func TestDiagram_PlantUMLDirectionRight(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer
	err := report.WritePlantUML(&buf, auditlog.WithDirection(output.DirectionRight))
	if err != nil {
		t.Fatalf("WritePlantUML: %v", err)
	}

	if !strings.Contains(buf.String(), "left to right direction") {
		t.Errorf("expected 'left to right direction' in output, got:\n%s", buf.String())
	}
}

func TestDiagram_PlantUMLDefaultNoDirectionCommand(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer
	err := report.WritePlantUML(&buf)
	if err != nil {
		t.Fatalf("WritePlantUML: %v", err)
	}

	if strings.Contains(buf.String(), "left to right direction") {
		t.Errorf("did not expect 'left to right direction' in default output")
	}
}

func TestDiagram_AllFormatsAcceptDirectionOption(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()
	opts := []auditlog.DiagramOption{auditlog.WithDirection(output.DirectionRight)}

	tests := []struct {
		name string
		fn   func(w *bytes.Buffer) error
	}{
		{"Mermaid", func(w *bytes.Buffer) error { return report.WriteMermaid(w, opts...) }},
		{"PlantUML", func(w *bytes.Buffer) error { return report.WritePlantUML(w, opts...) }},
		{"DOT", func(w *bytes.Buffer) error { return report.WriteDOT(w, opts...) }},
		{"D2", func(w *bytes.Buffer) error { return report.WriteD2(w, opts...) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := tt.fn(&buf)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			if buf.Len() == 0 {
				t.Errorf("%s produced empty output", tt.name)
			}
		})
	}
}
