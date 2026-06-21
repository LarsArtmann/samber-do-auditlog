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

func TestReport_WriteMermaid(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
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

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
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

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
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
				ServiceRef: auditlog.ServiceRef{
					ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: `svc"quote`,
				},
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
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
	if !strings.Contains(output, `\"`) {
		t.Errorf("expected escaped quote in DOT label, got:\n%s", output)
	}
}

func TestWriteDOT_WriterError(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.Report().WriteDOT(failingWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
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

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.Report().WriteMermaid(failingWriter{})
	if err == nil {
		t.Error("expected error from failing writer")
	}
}

func TestWritePlantUML_WriterError(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.Report().WritePlantUML(failingWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
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
				ServiceRef: auditlog.ServiceRef{
					ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: "svc-a",
				},
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
				Dependencies: []auditlog.ServiceRef{
					{ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: "svc-b"},
					{ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: "svc-b"},
				},
			},
			{
				ServiceRef: auditlog.ServiceRef{
					ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: "svc-b",
				},
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
	assertSingleEdge(t, "root_svc_a --> root_svc_b", report.WriteMermaid)
}

func TestWriteMermaid_ExternalDependency(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{
					ScopeID:     "root",
					ScopeName:   auditlog.RootScopeName,
					ServiceName: "my-service",
				},
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
				Dependencies: []auditlog.ServiceRef{
					{
						ScopeID:     "root",
						ScopeName:   auditlog.RootScopeName,
						ServiceName: "external-dep",
					},
				},
			},
		},
	}

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
				ServiceRef: auditlog.ServiceRef{
					ScopeID:     "root",
					ScopeName:   auditlog.RootScopeName,
					ServiceName: `evil]"svc`,
				},
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
			},
		},
	}
}

func TestReport_WriteD2(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
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
	assertStringContains(t, output, ":")
	// D2 edge syntax: id -> id
	assertStringContains(t, output, "->")
	// Warm-amber per-node styling
	assertStringContains(t, output, "style.fill:")
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
	// D2 escapes double-quotes as \" in labels.
	assertStringContains(t, output, `\"`)
}

func TestWriteD2_WriterError(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.Report().WriteD2(failingWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
}

func TestWriteD2_DuplicateEdges(t *testing.T) {
	t.Parallel()

	report := reportWithDuplicateEdges()
	assertSingleEdge(t, "root_svc_a -> root_svc_b", report.WriteD2)
}

func TestWriteD2_ExternalDependency(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "test",
		ExportedAt:  time.Now(),
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{
					ScopeID:     "root",
					ScopeName:   auditlog.RootScopeName,
					ServiceName: "my-service",
				},
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: time.Now(),
				Dependencies: []auditlog.ServiceRef{
					{
						ScopeID:     "root",
						ScopeName:   auditlog.RootScopeName,
						ServiceName: "external-dep",
					},
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
	assertStringContains(t, output, ":")
	assertStringContains(t, output, "external-dep")
	assertStringContains(t, output, "->")
}
