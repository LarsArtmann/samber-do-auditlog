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
				ServiceRef:   rootRef("my-service"),
				Status:       auditlog.ServiceStatusActive,
				RegisteredAt: now,
				Dependencies: []auditlog.ServiceRef{
					rootRef("external-dep"),
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
				ServiceRef:   rootRef(`svc"quote`),
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
