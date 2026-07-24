package auditlog_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-output"
	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestTableColumns_DefaultColumns(t *testing.T) {
	t.Parallel()

	report, buf := singleServiceWithExternalDepReportAndBuf()

	err := report.WriteTable(buf, output.FormatCSV, auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("WriteTable: %v", err)
	}

	output := buf.String()
	for _, header := range []string{"Service", "Scope", "Type", "Status", "Invocations", "Build(ms)", "Error"} {
		if !strings.Contains(output, header) {
			t.Errorf("expected header %q in default output", header)
		}
	}
}

func TestTableColumns_CustomSelection(t *testing.T) {
	t.Parallel()

	report, buf := singleServiceWithExternalDepReportAndBuf()

	err := report.WriteTable(buf, output.FormatCSV, auditlog.DefaultTableOpts(),
		auditlog.WithColumns(auditlog.ColumnService, auditlog.ColumnStatus),
	)
	if err != nil {
		t.Fatalf("WriteTable: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Service") {
		t.Errorf("expected 'Service' header")
	}

	if !strings.Contains(output, "Status") {
		t.Errorf("expected 'Status' header")
	}

	for _, absent := range []string{"Scope", "Type", "Invocations", "Build(ms)", "Error"} {
		if strings.Contains(output, absent) {
			t.Errorf("did not expect column %q in custom selection", absent)
		}
	}
}

func TestTableColumns_AllColumns(t *testing.T) {
	t.Parallel()

	report, buf := singleServiceWithExternalDepReportAndBuf()

	err := report.WriteTable(buf, output.FormatCSV, auditlog.DefaultTableOpts(),
		auditlog.WithColumns(auditlog.AllTableColumns()...),
	)
	if err != nil {
		t.Fatalf("WriteTable: %v", err)
	}

	output := buf.String()

	for _, col := range auditlog.AllTableColumns() {
		if !strings.Contains(output, col.String()) {
			t.Errorf("expected column %q in all-columns output", col.String())
		}
	}
}

func TestTableColumns_ColumnOrderPreserved(t *testing.T) {
	t.Parallel()

	report, buf := singleServiceWithExternalDepReportAndBuf()

	err := report.WriteTable(buf, output.FormatCSV, auditlog.DefaultTableOpts(),
		auditlog.WithColumns(auditlog.ColumnError, auditlog.ColumnService),
	)
	if err != nil {
		t.Fatalf("WriteTable: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 1 {
		t.Fatal("expected at least one line of output")
	}

	header := lines[0]

	errorIdx := strings.Index(header, "Error")
	serviceIdx := strings.Index(header, "Service")

	if errorIdx == -1 || serviceIdx == -1 {
		t.Fatalf("expected both Error and Service headers, got: %s", header)
	}

	if errorIdx > serviceIdx {
		t.Errorf("expected Error before Service, got header: %s", header)
	}
}

func TestTableColumns_WithDependentsColumn(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")
	provideUserServiceWithDB(injector, "users", "db")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*UserService](injector, "users")
	report := p.Report()

	var buf bytes.Buffer

	err := report.WriteTable(&buf, output.FormatCSV, auditlog.DefaultTableOpts(),
		auditlog.WithColumns(auditlog.ColumnService, auditlog.ColumnDependencies, auditlog.ColumnDependents),
	)
	if err != nil {
		t.Fatalf("WriteTable: %v", err)
	}

	result := buf.String()

	if !strings.Contains(result, "Dependencies") {
		t.Error("expected 'Dependencies' header")
	}

	if !strings.Contains(result, "Dependents") {
		t.Error("expected 'Dependents' header")
	}

	// The "db" service should have "users" in its Dependents.
	if !strings.Contains(result, "users") {
		t.Error("expected 'users' in dependents data")
	}
}

func TestTableColumns_WriteTableString(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	result, err := report.WriteTableString(output.FormatCSV, auditlog.DefaultTableOpts(),
		auditlog.WithColumns(auditlog.ColumnService),
	)
	if err != nil {
		t.Fatalf("WriteTableString: %v", err)
	}

	if !strings.Contains(result, "Service") {
		t.Error("expected 'Service' header in WriteTableString output")
	}
}

func TestTableColumns_UnknownColumnString(t *testing.T) {
	t.Parallel()

	col := auditlog.TableColumn(999)
	if col.String() != "Unknown" {
		t.Errorf("expected 'Unknown' for invalid column, got %q", col.String())
	}
}

func TestTableColumns_DefaultTableColumnsImmutable(t *testing.T) {
	t.Parallel()

	report := singleServiceWithExternalDepReport()

	// Call WithColumns with a custom selection that should NOT affect the default.
	var buf1 bytes.Buffer

	err := report.WriteTable(&buf1, output.FormatCSV, auditlog.DefaultTableOpts(),
		auditlog.WithColumns(auditlog.ColumnService),
	)
	if err != nil {
		t.Fatalf("WriteTable (custom): %v", err)
	}

	// Now call without WithColumns — should use the original default columns.
	var buf2 bytes.Buffer

	err = report.WriteTable(&buf2, output.FormatCSV, auditlog.DefaultTableOpts())
	if err != nil {
		t.Fatalf("WriteTable (default): %v", err)
	}

	if !strings.Contains(buf2.String(), "Status") {
		t.Error("DefaultTableColumns was mutated by previous WithColumns call")
	}
}
