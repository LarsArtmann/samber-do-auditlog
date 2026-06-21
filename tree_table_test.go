package auditlog_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReport_WriteTree_BasicDAG(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-tree"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideUserServiceWithDeps(injector, "user-service", "db", "cache")

	_, _ = do.InvokeNamed[*UserService](injector, "user-service")

	report := plugin.Report()

	var buf bytes.Buffer

	err := report.WriteTree(&buf)
	if err != nil {
		t.Fatalf("WriteTree error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "db") {
		t.Errorf("tree output missing 'db':\n%s", out)
	}

	if !strings.Contains(out, "cache") {
		t.Errorf("tree output missing 'cache':\n%s", out)
	}
}

func TestReport_WriteTree_EmptyReport(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "empty",
	}

	var buf bytes.Buffer

	err := report.WriteTree(&buf)
	if err != nil {
		t.Fatalf("WriteTree error on empty report: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("tree output should not be empty even for empty report")
	}
}

func TestReport_WriteHTMLTree_BasicDAG(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-html-tree"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	provideUserServiceWithDB(injector, "user-service", "db")

	_, _ = do.InvokeNamed[*UserService](injector, "user-service")

	report := plugin.Report()

	var buf bytes.Buffer

	err := report.WriteHTMLTree(&buf)
	if err != nil {
		t.Fatalf("WriteHTMLTree error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "<ul") && !strings.Contains(out, "<ol") {
		t.Errorf("HTML tree output should contain list tags:\n%s", out)
	}

	if !strings.Contains(out, "db") {
		t.Errorf("HTML tree output missing 'db':\n%s", out)
	}
}

func TestReport_WriteTable_CSV(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-table"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")

	_, _ = do.InvokeNamed[*Database](injector, "db")
	_, _ = do.InvokeNamed[*Cache](injector, "cache")

	report := plugin.Report()

	var buf bytes.Buffer

	err := report.WriteTable(&buf, "csv", auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("WriteTable csv error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "Service") {
		t.Errorf("CSV table missing header 'Service':\n%s", out)
	}

	if !strings.Contains(out, "db") {
		t.Errorf("CSV table missing 'db' row:\n%s", out)
	}
}

func TestReport_WriteTable_JSON(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-table-json"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_, _ = do.InvokeNamed[*Database](injector, "db")

	report := plugin.Report()

	var buf bytes.Buffer

	err := report.WriteTable(&buf, "json", auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("WriteTable json error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "db") {
		t.Errorf("JSON table missing 'db':\n%s", out)
	}
}

func TestPlugin_WriteTree_DelegatesToReport(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-plugin-tree"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_, _ = do.InvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteTree(&buf)
	if err != nil {
		t.Fatalf("Plugin.WriteTree error: %v", err)
	}

	if !strings.Contains(buf.String(), "db") {
		t.Errorf("Plugin.WriteTree output missing 'db':\n%s", buf.String())
	}
}

func TestReport_WriteTreeString(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "str-test",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{ScopeID: "r", ScopeName: "[root]", ServiceName: "svc-a"},
				Status:     auditlog.ServiceStatusActive,
			},
		},
	}

	out, err := report.WriteTreeString()
	if err != nil {
		t.Fatalf("WriteTreeString error: %v", err)
	}

	if !strings.Contains(out, "svc-a") {
		t.Errorf("WriteTreeString missing 'svc-a':\n%s", out)
	}
}

func TestReport_WriteHTMLTreeString(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "str-test",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{ScopeID: "r", ScopeName: "[root]", ServiceName: "svc-b"},
				Status:     auditlog.ServiceStatusActive,
			},
		},
	}

	out, err := report.WriteHTMLTreeString()
	if err != nil {
		t.Fatalf("WriteHTMLTreeString error: %v", err)
	}

	if !strings.Contains(out, "svc-b") {
		t.Errorf("WriteHTMLTreeString missing 'svc-b':\n%s", out)
	}
}

func TestReport_WriteTableString(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "str-test",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{ScopeID: "r", ScopeName: "[root]", ServiceName: "svc-c"},
				Status:     auditlog.ServiceStatusActive,
			},
		},
	}

	out, err := report.WriteTableString("markdown", auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("WriteTableString error: %v", err)
	}

	if !strings.Contains(out, "svc-c") {
		t.Errorf("WriteTableString missing 'svc-c':\n%s", out)
	}
}

func TestReport_WriteTree_FailingWriter(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "fail-test",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{ScopeID: "r", ScopeName: "[root]", ServiceName: "svc"},
				Status:     auditlog.ServiceStatusActive,
			},
		},
	}

	assertWriteFails(t, "WriteTree", func(w io.Writer) error {
		return report.WriteTree(w)
	})
	assertWriteFails(t, "WriteHTMLTree", func(w io.Writer) error {
		return report.WriteHTMLTree(w)
	})
	assertWriteFails(t, "WriteTable", func(w io.Writer) error {
		return report.WriteTable(w, "csv", auditlog.DefaultTableOpts())
	})
}

func TestReport_WriteTreeString_FailingWriter(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "fail-test",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{ScopeID: "r", ScopeName: "[root]", ServiceName: "svc"},
				Status:     auditlog.ServiceStatusActive,
			},
		},
	}

	_, err := report.WriteTreeString()
	if err != nil {
		t.Errorf("WriteTreeString should not error with strings.Builder: %v", err)
	}

	_, err = report.WriteHTMLTreeString()
	if err != nil {
		t.Errorf("WriteHTMLTreeString should not error with strings.Builder: %v", err)
	}

	_, err = report.WriteTableString("csv", auditlog.DefaultTableOpts())
	if err != nil {
		t.Errorf("WriteTableString should not error with strings.Builder: %v", err)
	}
}

func TestReport_WriteTable_ShutdownError(t *testing.T) {
	t.Parallel()

	shutdownErr := "connection reset"
	buildMs := 42.5

	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "err-test",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{
					ScopeID:     "r",
					ScopeName:   "[root]",
					ServiceName: "crashing-svc",
				},
				Status:               auditlog.ServiceStatusShutdownError,
				ShutdownError:        &shutdownErr,
				FirstBuildDurationMs: &buildMs,
			},
		},
	}

	var buf bytes.Buffer

	err := report.WriteTable(&buf, "csv", auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("WriteTable error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "connection reset") {
		t.Errorf("table should contain shutdown error:\n%s", out)
	}

	if !strings.Contains(out, "42.5") {
		t.Errorf("table should contain build duration:\n%s", out)
	}
}

func TestReport_WriteTree_AllServicesHaveDeps(t *testing.T) {
	t.Parallel()

	// When every service has dependencies (e.g. all depend on something
	// external not in the report), the tree falls back to using the
	// first service as root.
	report := auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "all-deps",
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: auditlog.ServiceRef{ScopeID: "r", ScopeName: "[root]", ServiceName: "svc-a"},
				Status:     auditlog.ServiceStatusActive,
				Dependencies: []auditlog.ServiceRef{
					{ScopeID: "ext", ScopeName: "external", ServiceName: "ext-svc"},
				},
			},
		},
	}

	out, err := report.WriteTreeString()
	if err != nil {
		t.Fatalf("WriteTreeString error: %v", err)
	}

	if !strings.Contains(out, "svc-a") {
		t.Errorf("tree should contain fallback root 'svc-a':\n%s", out)
	}
}
