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

// singleServiceWithExternalDepReport builds a Report containing one
// service ("my-service") that depends on a non-registered
// "external-dep". Used by tests that assert the external-dep edge
// appears in the rendered diagram.
func singleServiceWithExternalDepReport() auditlog.Report {
	now := time.Now()

	return auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test",
		ExportedAt:  now,
		Services: []auditlog.ServiceInfo{
			{
				ServiceIdentity: auditlog.ServiceIdentity{
					ServiceRef: rootRef("my-service"),
				},
				ServiceLifecycle: auditlog.ServiceLifecycle{
					Status:       auditlog.ServiceStatusActive,
					RegisteredAt: now,
				},
				ServiceGraph: auditlog.ServiceGraph{
					Dependencies: []auditlog.ServiceRef{
						rootRef("external-dep"),
					},
				},
			},
		},
	}
}

// singleServiceWithExternalDepReportAndBuf is the standard preamble for tests
// that render the singleServiceWithExternalDepReport into a bytes.Buffer — it
// returns both so each test can keep its own setup boilerplate-free.
func singleServiceWithExternalDepReportAndBuf() (auditlog.Report, *bytes.Buffer) {
	return singleServiceWithExternalDepReport(), &bytes.Buffer{}
}

func TestReport_WriteMermaid(t *testing.T) {
	t.Parallel()

	p, injector := setupWithDB("test")
	provideUserServiceWithDB(injector, "user-svc", "db")

	_ = do.MustInvokeNamed[*UserService](injector, "user-svc")

	report := p.Report()

	var buf bytes.Buffer

	err := report.WriteMermaid(&buf)
	if err != nil {
		t.Fatalf("WriteMermaid: %v", err)
	}

	output := buf.String()

	assertStringContains(t, output, "flowchart TD")
	assertStringContains(t, output, "-->")
}

func TestReport_WritePlantUML(t *testing.T) {
	t.Parallel()

	p, injector := setupWithDB("test")
	provideUserServiceWithDB(injector, "user-svc", "db")

	_ = do.MustInvokeNamed[*UserService](injector, "user-svc")

	report := p.Report()

	var buf bytes.Buffer

	err := report.WritePlantUML(&buf)
	if err != nil {
		t.Fatalf("WritePlantUML: %v", err)
	}

	output := buf.String()

	assertStringContains(t, output, "@startuml")
	assertStringContains(t, output, "@enduml")
	assertStringContains(t, output, "component")

	assertStringContains(t, output, "-->")
	assertStringContains(t, output, "db")
	assertStringContains(t, output, "user-svc")
}

func TestReport_WriteDOT(t *testing.T) {
	t.Parallel()

	p, injector := setupWithDB("test")
	provideUserServiceWithDB(injector, "user-svc", "db")

	_ = do.MustInvokeNamed[*UserService](injector, "user-svc")

	report := p.Report()

	var buf bytes.Buffer

	err := report.WriteDOT(&buf)
	if err != nil {
		t.Fatalf("WriteDOT: %v", err)
	}

	output := buf.String()

	assertStringContains(t, output, "digraph do_auditlog")
	assertStringContains(t, output, "->")
	assertStringContains(t, output, "label=")
	assertStringContains(t, output, "db")
	assertStringContains(t, output, "user-svc")

	// A closing brace terminates the digraph.
	if !strings.Contains(output, "}") {
		t.Error("DOT output missing closing brace")
	}
}

func TestReport_WriteDOT_LabelEscaping(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceIdentity: auditlog.ServiceIdentity{
					ServiceRef: rootRef(`svc"quote`),
				},
				ServiceLifecycle: auditlog.ServiceLifecycle{
					Status:       auditlog.ServiceStatusActive,
					RegisteredAt: time.Now(),
				},
			},
		},
	}

	var buf bytes.Buffer

	err := report.WriteDOT(&buf)
	if err != nil {
		t.Fatalf("WriteDOT: %v", err)
	}

	output := buf.String()
	// The literal double-quote in the service name must be escaped as \".
	assertStringContains(t, output, `\"`)
}

func TestWriteDOT_WriterError(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")

	assertWriteFails(t, "WriteDOT", func(w io.Writer) error { return p.Report().WriteDOT(w) })
}

func TestWriteMermaid_WithDependencies_LabelsDeps(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideValue(injector, &Database{URL: "test"})

	do.ProvideNamed(injector, "usersvc", func(i do.Injector) (*UserService, error) {
		return &UserService{
			DB:    do.MustInvoke[*Database](i),
			Cache: &Cache{},
		}, nil
	})

	_ = do.MustInvokeNamed[*UserService](injector, "usersvc")

	var buf bytes.Buffer

	err := p.Report().WriteMermaid(&buf)
	if err != nil {
		t.Fatalf("WriteMermaid: %v", err)
	}

	output := buf.String()
	assertStringContains(t, output, "flowchart TD")
	assertStringContains(t, output, "-->")
	assertStringContains(t, output, "Database")
	assertStringContains(t, output, "usersvc")
}

func TestWriteMermaid_WriterError(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")

	assertWriteFails(t, "WriteMermaid", func(w io.Writer) error { return p.Report().WriteMermaid(w) })
}

func TestWritePlantUML_WriterError(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")

	assertWriteFails(t, "WritePlantUML", func(w io.Writer) error { return p.Report().WritePlantUML(w) })
}

// reportWithDuplicateEdges builds a report where svc-a depends on svc-b twice.
// Used by duplicate-edge tests to verify deduplication across diagram formats.
func reportWithDuplicateEdges() auditlog.Report {
	return auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceIdentity: auditlog.ServiceIdentity{
					ServiceRef: rootRef("svc-a"),
				},
				ServiceLifecycle: auditlog.ServiceLifecycle{
					Status:       auditlog.ServiceStatusActive,
					RegisteredAt: time.Now(),
				},
				ServiceGraph: auditlog.ServiceGraph{
					Dependencies: []auditlog.ServiceRef{
						rootRef("svc-b"),
						rootRef("svc-b"),
					},
				},
			},
			{
				ServiceIdentity: auditlog.ServiceIdentity{
					ServiceRef: rootRef("svc-b"),
				},
				ServiceLifecycle: auditlog.ServiceLifecycle{
					Status:       auditlog.ServiceStatusActive,
					RegisteredAt: time.Now(),
				},
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
				ServiceIdentity: auditlog.ServiceIdentity{
					ServiceRef: rootRef(`evil]"svc`),
				},
				ServiceLifecycle: auditlog.ServiceLifecycle{
					Status:       auditlog.ServiceStatusActive,
					RegisteredAt: time.Now(),
				},
			},
		},
	}
}

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
				ServiceIdentity: auditlog.ServiceIdentity{
					ServiceRef: auditlog.ServiceRef{
						ScopeName:   auditlog.RootScopeName,
						ScopeID:     auditlog.RootScopeName,
						ServiceName: "a\\b\n\tc",
					},
				},
				ServiceLifecycle: auditlog.ServiceLifecycle{
					Status:       auditlog.ServiceStatusActive,
					RegisteredAt: time.Now(),
				},
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
