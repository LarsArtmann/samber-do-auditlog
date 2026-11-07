package auditlog_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReport_WriteD2(t *testing.T) {
	t.Parallel()

	p, injector := setupWithDB("test")
	provideUserServiceWithDB(injector, "user-svc", "db")

	_ = do.MustInvokeNamed[*UserService](injector, "user-svc")

	report := p.Report()

	var buf bytes.Buffer

	err := report.WriteD2(&buf)
	if err != nil {
		t.Fatalf("WriteD2: %v", err)
	}

	output := buf.String()

	// D2 node syntax: id: label { ... }
	assertStringContains(t, output, "db")
	assertStringContains(t, output, "user-svc")
	// D2 edge syntax: id -> id
	assertStringContains(t, output, "->")
	// Warm-amber per-node styling
	assertStringContains(t, output, "style.fill:")
	// Diagram title from container ID
	assertStringContains(t, output, "title:")
}

func TestWriteD2_EscapesSpecialChars(t *testing.T) {
	t.Parallel()

	report := reportWithSpecialCharService()

	var buf bytes.Buffer

	err := report.WriteD2(&buf)
	if err != nil {
		t.Fatalf("WriteD2: %v", err)
	}

	output := buf.String()
	// D2 escapes double-quote as \" in labels. The bracket is left as-is (D2
	// has no bracket delimiter to break), so the full escaped label is present.
	assertStringContains(t, output, `evil]\"svc`)
}

func TestWriteD2_EscapesControlChars(t *testing.T) {
	t.Parallel()

	// D2's d2Replacer escapes backslash, newline, and tab (quote is covered
	// above). Verify each control character is escaped, not passed through raw.
	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ScopeName:    auditlog.RootScopeName,
				ScopeID:      auditlog.RootScopeName,
				ServiceName:  "a\\b\n\tc",
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
			},
		},
	}

	var buf bytes.Buffer

	err := report.WriteD2(&buf)
	if err != nil {
		t.Fatalf("WriteD2: %v", err)
	}

	output := buf.String()
	// Backslash -> \\, newline -> \n, tab -> \t (literal backslash sequences in output).
	assertStringContains(t, output, `a\\b\n\tc`)
}

func TestWriteD2_WriterError(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")

	assertWriteFails(t, "WriteD2", func(w io.Writer) error { return p.Report().WriteD2(w) })
}

func TestWriteD2_DuplicateEdges(t *testing.T) {
	t.Parallel()

	report := reportWithDuplicateEdges()

	assertSingleEdge(t, "root_svc_a -> root_svc_b", func(w io.Writer) error { return report.WriteD2(w) })
}

func TestWriteD2_ExternalDependency(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer

	err := report.WriteD2(&buf)
	if err != nil {
		t.Fatalf("WriteD2: %v", err)
	}

	output := buf.String()
	assertStringContains(t, output, ":")
	assertStringContains(t, output, "external-dep")
	assertStringContains(t, output, "->")
}

func TestWriteD2_HexColorsQuoted(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer

	err := report.WriteD2(&buf)
	if err != nil {
		t.Fatalf("WriteD2: %v", err)
	}

	output := buf.String()
	// D2 treats # as a comment character, so hex color values MUST be
	// double-quoted or the rest of the line is silently ignored.
	// Regression test for go-output v0.31.1 d2Quote() fix.
	assertStringContains(t, output, `style.fill: "#e8a838"`)
	assertStringContains(t, output, `style.stroke: "#4a4030"`)

	if strings.Contains(output, `style.fill: #`) {
		t.Errorf("D2 hex color must be quoted — unquoted form is treated as a comment:\n%s", output)
	}
}

func TestWriteDOT_HexColorsQuoted(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	var buf bytes.Buffer

	err := report.WriteDOT(&buf)
	if err != nil {
		t.Fatalf("WriteDOT: %v", err)
	}

	output := buf.String()
	// DOT hex color values must be double-quoted for standards compliance.
	// Regression test for go-output v0.31.1 quoting fix.
	assertStringContains(t, output, `fillcolor="#e8a838"`)
	assertStringContains(t, output, `color="#4a4030"`)

	if strings.Contains(output, `fillcolor=#`) {
		t.Errorf("DOT hex color must be quoted:\n%s", output)
	}
}
