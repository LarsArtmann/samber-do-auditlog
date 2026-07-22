package auditlog_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// activeSvcReport builds a minimal Report with one Active service in the root
// scope. Centralizes the 6-line struct literal repeated by every tree/table
// string-returning test.
func activeSvcReport(containerID auditlog.ContainerID, serviceName auditlog.ServiceName) auditlog.Report {
	return auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: containerID,
		Services: []auditlog.ServiceInfo{
			{
				ServiceRef: rootRef(serviceName),
				Status:     auditlog.ServiceStatusActive,
			},
		},
	}
}

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

	assertOutputContains(t, "tree", out, "db")
	assertOutputContains(t, "tree", out, "cache")
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

	assertOutputContains(t, "HTML tree", out, "db")
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

	assertOutputContains(t, "CSV table", out, "Service")
	assertOutputContains(t, "CSV table", out, "db")
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

	assertOutputContains(t, "JSON table", buf.String(), "db")
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

	assertOutputContains(t, "Plugin.WriteTree", buf.String(), "db")
}

func TestReport_WriteTreeString(t *testing.T) {
	t.Parallel()

	out, err := activeSvcReport("str-test", "svc-a").WriteTreeString()
	if err != nil {
		t.Fatalf("WriteTreeString error: %v", err)
	}

	assertOutputContains(t, "WriteTreeString", out, "svc-a")
}

func TestReport_WriteHTMLTreeString(t *testing.T) {
	t.Parallel()

	out, err := activeSvcReport("str-test", "svc-b").WriteHTMLTreeString()
	if err != nil {
		t.Fatalf("WriteHTMLTreeString error: %v", err)
	}

	assertOutputContains(t, "WriteHTMLTreeString", out, "svc-b")
}

func TestReport_WriteTableString(t *testing.T) {
	t.Parallel()

	out, err := activeSvcReport("str-test", "svc-c").WriteTableString("markdown", auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("WriteTableString error: %v", err)
	}

	assertOutputContains(t, "WriteTableString", out, "svc-c")
}

func TestReport_WriteTree_FailingWriter(t *testing.T) {
	t.Parallel()

	report := activeSvcReport("fail-test", "svc")

	assertWriteFails(t, "WriteTree", report.WriteTree)
	assertWriteFails(t, "WriteHTMLTree", report.WriteHTMLTree)
	assertWriteFails(t, "WriteTable", func(w io.Writer) error {
		return report.WriteTable(w, "csv", auditlog.DefaultTableOpts())
	})
}

func TestReport_WriteTreeString_FailingWriter(t *testing.T) {
	t.Parallel()

	report := activeSvcReport("fail-test", "svc")

	if _, err := report.WriteTreeString(); err != nil {
		t.Errorf("WriteTreeString should not error with strings.Builder: %v", err)
	}

	if _, err := report.WriteHTMLTreeString(); err != nil {
		t.Errorf("WriteHTMLTreeString should not error with strings.Builder: %v", err)
	}

	if _, err := report.WriteTableString("csv", auditlog.DefaultTableOpts()); err != nil {
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
				ServiceRef:           rootRef("crashing-svc"),
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

	assertOutputContains(t, "table", out, "connection reset")
	assertOutputContains(t, "table", out, "42.5")
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
				ServiceRef: rootRef("svc-a"),
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

	assertOutputContains(t, "tree", out, "svc-a")
}
