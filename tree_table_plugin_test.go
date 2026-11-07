package auditlog_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_WriteHTMLTree_DelegatesToReport(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-plugin-htmltree"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_, _ = do.InvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteHTMLTree(&buf)
	if err != nil {
		t.Fatalf("Plugin.WriteHTMLTree error: %v", err)
	}

	assertOutputContains(t, "Plugin.WriteHTMLTree", buf.String(), "db")
}

func TestPlugin_WriteTable_DelegatesToReport(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-plugin-table"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_, _ = do.InvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteTable(&buf, "csv", auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("Plugin.WriteTable error: %v", err)
	}

	assertOutputContains(t, "Plugin.WriteTable", buf.String(), "db")
}

func TestPlugin_ExportToTree(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-plugin-export-tree"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_, _ = do.InvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/tree.txt"

	err := plugin.ExportToTree(path)
	if err != nil {
		t.Fatalf("Plugin.ExportToTree error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read exported tree: %v", err)
	}

	assertOutputContains(t, "Plugin.ExportToTree", string(data), "db")
}

func TestPlugin_ExportToHTMLTree(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-plugin-export-htmltree"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_, _ = do.InvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/tree.html"

	err := plugin.ExportToHTMLTree(path)
	if err != nil {
		t.Fatalf("Plugin.ExportToHTMLTree error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read exported html tree: %v", err)
	}

	assertOutputContains(t, "Plugin.ExportToHTMLTree", string(data), "db")
}

func TestPlugin_ExportToTable(t *testing.T) {
	t.Parallel()

	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "test-plugin-export-table"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_, _ = do.InvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/table.csv"

	err := plugin.ExportToTable(path, "csv", auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("Plugin.ExportToTable error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read exported table: %v", err)
	}

	assertOutputContains(t, "Plugin.ExportToTable", string(data), "db")
}

// errWriter is an io.Writer that always returns an error.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
