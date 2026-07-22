package auditlog

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// WriteCSV writes all services as comma-separated values to the writer.
// The first row is a header. Pointer fields render as empty strings when nil.
// Dependencies and dependents are semicolon-separated "scope/name" refs.
func (r Report) WriteCSV(writer io.Writer) error {
	return r.writeDelimited(writer, ',')
}

// WriteTSV writes all services as tab-separated values to the writer.
// Identical to WriteCSV but with a tab delimiter for tools that prefer TSV.
func (r Report) WriteTSV(writer io.Writer) error {
	return r.writeDelimited(writer, '\t')
}

func (r Report) writeDelimited(writer io.Writer, comma rune) error {
	w := csv.NewWriter(writer) //nolint:varnamelen // idiomatic short name for csv.Writer
	w.Comma = comma

	header := []string{
		"scope_id", "scope_name", "service_name",
		"status", "service_type",
		"registered_at", "first_invoked_at",
		"invocation_count", "invocation_order",
		"first_build_duration_ms",
		"shutdown_at", "shutdown_duration_ms",
		"invocation_error", "shutdown_error",
		"is_healthchecker", "is_shutdowner",
		"health_check_count", "last_health_check_at", "health_check_error",
		"dependencies", "dependents",
	}

	err := w.Write(header)
	if err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, svc := range r.Services {
		err := w.Write(serviceToCSVRow(svc))
		if err != nil {
			return fmt.Errorf("write service %q: %w", svc.ServiceName, err)
		}
	}

	w.Flush()

	err = w.Error()
	if err != nil {
		return fmt.Errorf("flush delimited writer: %w", err)
	}

	return nil
}

// serviceToCSVRow converts a ServiceInfo to a string slice matching the header.
func serviceToCSVRow(svc ServiceInfo) []string {
	return []string{
		string(svc.ScopeID),
		svc.ScopeName,
		string(svc.ServiceName),
		string(svc.Status),
		string(svc.ServiceType),
		svc.RegisteredAt.Format(time.RFC3339Nano),
		formatTimePtr(svc.FirstInvokedAt),
		strconv.Itoa(svc.InvocationCount),
		strconv.Itoa(svc.InvocationOrder),
		formatFloatPtr(svc.FirstBuildDurationMs),
		formatTimePtr(svc.ShutdownAt),
		formatFloatPtr(svc.ShutdownDurationMs),
		formatStrPtr(svc.InvocationError),
		formatStrPtr(svc.ShutdownError),
		strconv.FormatBool(svc.IsHealthchecker),
		strconv.FormatBool(svc.IsShutdowner),
		strconv.Itoa(svc.HealthCheckCount),
		formatTimePtr(svc.LastHealthCheckAt),
		formatStrPtr(svc.HealthCheckError),
		formatServiceRefs(svc.Dependencies),
		formatServiceRefs(svc.Dependents),
	}
}

// formatTimePtr renders a *time.Time as RFC3339Nano or empty string when nil.
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}

	return t.Format(time.RFC3339Nano)
}

// formatFloatPtr renders a *float64 as a decimal string or empty when nil.
func formatFloatPtr(f *float64) string {
	if f == nil {
		return ""
	}

	return strconv.FormatFloat(*f, 'f', -1, 64)
}

// formatStrPtr dereferences a *string or returns empty when nil.
func formatStrPtr(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// formatServiceRefs joins ServiceRef values as semicolon-separated "scope/name"
// strings. Returns empty string for nil or empty slices.
func formatServiceRefs(refs []ServiceRef) string {
	if len(refs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(refs))

	for _, ref := range refs {
		parts = append(parts, ref.String())
	}

	return strings.Join(parts, ";")
}
