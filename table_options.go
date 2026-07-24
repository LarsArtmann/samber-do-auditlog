package auditlog

import (
	"strconv"
	"strings"
)

// TableColumn identifies a column available in the service summary table.
type TableColumn int

const (
	// ColumnService is the service name.
	ColumnService TableColumn = iota
	// ColumnScope is the scope name.
	ColumnScope
	// ColumnType is the provider type (lazy, eager, transient, alias).
	ColumnType
	// ColumnStatus is the derived service status.
	ColumnStatus
	// ColumnInvocations is the number of times the service was invoked.
	ColumnInvocations
	// ColumnBuildMs is the first build duration in milliseconds.
	ColumnBuildMs
	// ColumnError is the error message (invocation or shutdown).
	ColumnError
	// ColumnDependencies is a comma-separated list of dependency names.
	ColumnDependencies
	// ColumnDependents is a comma-separated list of dependent service names.
	ColumnDependents
	// ColumnHealthChecks is the number of health checks performed.
	ColumnHealthChecks
)

// DefaultTableColumns is the column set used when WithColumns is not called.
//
//nolint:gochecknoglobals // Read-only default; internal callers always copy before use.
var DefaultTableColumns = []TableColumn{
	ColumnService,
	ColumnScope,
	ColumnType,
	ColumnStatus,
	ColumnInvocations,
	ColumnBuildMs,
	ColumnError,
}

// String returns the column header for debug/logging.
func (c TableColumn) String() string {
	if def, ok := columnDefs[c]; ok {
		return def.header
	}

	return "Unknown"
}

// AllTableColumns returns every available table column in canonical order.
func AllTableColumns() []TableColumn {
	return []TableColumn{
		ColumnService,
		ColumnScope,
		ColumnType,
		ColumnStatus,
		ColumnInvocations,
		ColumnBuildMs,
		ColumnError,
		ColumnDependencies,
		ColumnDependents,
		ColumnHealthChecks,
	}
}

// columnDefinition pairs a header label with a cell extractor function.
type columnDefinition struct {
	header  string
	extract func(ServiceInfo) string
}

// columnDefs is the single source of truth mapping TableColumn values to
// their header text and data extraction logic.
//
//nolint:gochecknoglobals // Lookup table, treated as immutable after init.
var columnDefs = map[TableColumn]columnDefinition{
	ColumnService: {
		header:  "Service",
		extract: func(s ServiceInfo) string { return string(s.ServiceName) },
	},
	ColumnScope: {
		header:  "Scope",
		extract: func(s ServiceInfo) string { return s.ScopeName },
	},
	ColumnType: {
		header:  "Type",
		extract: func(s ServiceInfo) string { return string(s.ServiceType) },
	},
	ColumnStatus: {
		header:  "Status",
		extract: func(s ServiceInfo) string { return string(s.Status) },
	},
	ColumnInvocations: {
		header:  "Invocations",
		extract: func(s ServiceInfo) string { return strconv.Itoa(s.InvocationCount) },
	},
	ColumnBuildMs: {
		header:  "Build(ms)",
		extract: extractBuildMsCell,
	},
	ColumnError: {
		header:  "Error",
		extract: extractErrorCell,
	},
	ColumnDependencies: {
		header:  "Dependencies",
		extract: extractDependenciesCell,
	},
	ColumnDependents: {
		header:  "Dependents",
		extract: extractDependentsCell,
	},
	ColumnHealthChecks: {
		header:  "Health Checks",
		extract: func(s ServiceInfo) string { return strconv.Itoa(s.HealthCheckCount) },
	},
}

func extractBuildMsCell(s ServiceInfo) string {
	if s.FirstBuildDurationMs != nil {
		return strconv.FormatFloat(*s.FirstBuildDurationMs, 'f', 2, 64)
	}

	return ""
}

func extractErrorCell(s ServiceInfo) string {
	if s.InvocationError != nil {
		return *s.InvocationError
	}

	if s.ShutdownError != nil {
		return *s.ShutdownError
	}

	return ""
}

func extractDependenciesCell(s ServiceInfo) string {
	if len(s.Dependencies) == 0 {
		return ""
	}

	names := make([]string, 0, len(s.Dependencies))
	for _, dep := range s.Dependencies {
		names = append(names, string(dep.ServiceName))
	}

	return strings.Join(names, ", ")
}

func extractDependentsCell(s ServiceInfo) string {
	if len(s.Dependents) == 0 {
		return ""
	}

	names := make([]string, 0, len(s.Dependents))
	for _, dep := range s.Dependents {
		names = append(names, string(dep.ServiceName))
	}

	return strings.Join(names, ", ")
}

// TableOption configures service summary table output.
type TableOption func(*tableConfig)

type tableConfig struct {
	columns []TableColumn
}

// WithColumns selects which columns appear in the table output.
// Columns appear in the order specified. If not called, DefaultTableColumns
// is used (Service, Scope, Type, Status, Invocations, Build(ms), Error).
//
// Example: show only service name and status:
//
//	report.WriteTable(w, format, opts, auditlog.WithColumns(auditlog.ColumnService, auditlog.ColumnStatus))
func WithColumns(cols ...TableColumn) TableOption {
	return func(c *tableConfig) { c.columns = cols }
}

func applyTableOpts(opts []TableOption) tableConfig {
	var cfg tableConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	if len(cfg.columns) == 0 {
		cfg.columns = append([]TableColumn(nil), DefaultTableColumns...)
	}

	return cfg
}
