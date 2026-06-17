package auditlog_test

import (
	"bytes"
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

func TestWriteMermaid_DuplicateEdges(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
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

	var buf bytes.Buffer

	err := report.WriteMermaid(&buf)
	if err != nil {
		t.Fatalf("WriteMermaid: %v", err)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")
	edgeCount := 0

	for _, line := range lines {
		if strings.Contains(line, "root_svc_a --> root_svc_b") {
			edgeCount++
		}
	}

	if edgeCount != 1 {
		t.Errorf("expected exactly 1 deduplicated edge, got %d", edgeCount)
	}
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

	report := auditlog.Report{
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

	report := auditlog.Report{
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

	var buf bytes.Buffer

	err := report.WritePlantUML(&buf)
	if err != nil {
		t.Fatalf("WritePlantUML: %v", err)
	}

	output := buf.String()
	// The quote inside the service name is escaped to an apostrophe so the
	// quoted component declaration stays well-formed.
	assertStringContains(t, output, `component "evil]'svc" as root_evil_svc`)
}
