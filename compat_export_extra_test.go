package auditlog_test

import (
	"bytes"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// compat_export_extra_test.go exercises the Go 1.23 branch's stdlib
// re-implementations: the plain-text table format (new renderer), the
// unsupported-format error path, all diagram direction variants, and the
// health-stat section of the HTML report.

// TestWriteTable_PlainTextFormat covers the ASCII table renderer end to end.
func TestWriteTable_PlainTextFormat(t *testing.T) {
	t.Parallel()

	report := activeSvcReport("txt-test", "svc-a")

	var buf bytes.Buffer

	if err := report.WriteTable(&buf, auditlog.TableFormatTable, auditlog.DefaultTableOpts()); err != nil {
		t.Fatalf("WriteTable(table): %v", err)
	}

	out := buf.String()

	for _, want := range []string{"Service", "Scope", "svc-a", "+---", "| "} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in ASCII table output:\n%s", want, out)
		}
	}
}

// TestWriteTable_UnsupportedFormat verifies the branch-reduced format matrix
// rejects unknown formats instead of silently emitting garbage.
func TestWriteTable_UnsupportedFormat(t *testing.T) {
	t.Parallel()

	report := activeSvcReport("badfmt", "svc-a")

	var buf bytes.Buffer

	if err := report.WriteTable(&buf, auditlog.TableFormat("yaml"), auditlog.DefaultTableOpts()); err == nil {
		t.Error("expected error for unsupported table format, got nil")
	}
}

// TestDiagram_DirectionUpAndLeft closes the direction matrix: Up and Left map
// to BT/RL in Mermaid, up/left in D2, BT/RL rankdir in DOT, and the
// left-to-right command in PlantUML.
func TestDiagram_DirectionUpAndLeft(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		write    func(*bytes.Buffer, ...auditlog.DiagramOption) error
		wantSub  string
		notInDOT bool
	}{
		{
			name: "mermaid-up",
			write: func(b *bytes.Buffer, opts ...auditlog.DiagramOption) error {
				return activeSvcReport("dir", "svc").WriteMermaid(b, opts...)
			},
			wantSub: "flowchart BT",
		},
		{
			name: "mermaid-left",
			write: func(b *bytes.Buffer, opts ...auditlog.DiagramOption) error {
				return activeSvcReport("dir", "svc").WriteMermaid(b, opts...)
			},
			wantSub: "flowchart RL",
		},
		{
			name: "d2-up",
			write: func(b *bytes.Buffer, opts ...auditlog.DiagramOption) error {
				return activeSvcReport("dir", "svc").WriteD2(b, opts...)
			},
			wantSub: "direction: up",
		},
		{
			name: "d2-left",
			write: func(b *bytes.Buffer, opts ...auditlog.DiagramOption) error {
				return activeSvcReport("dir", "svc").WriteD2(b, opts...)
			},
			wantSub: "direction: left",
		},
		{
			name: "dot-up",
			write: func(b *bytes.Buffer, opts ...auditlog.DiagramOption) error {
				return activeSvcReport("dir", "svc").WriteDOT(b, opts...)
			},
			wantSub: "rankdir=BT",
		},
		{
			name: "dot-left",
			write: func(b *bytes.Buffer, opts ...auditlog.DiagramOption) error {
				return activeSvcReport("dir", "svc").WriteDOT(b, opts...)
			},
			wantSub: "rankdir=RL",
		},
		{
			name: "plantuml-left",
			write: func(b *bytes.Buffer, opts ...auditlog.DiagramOption) error {
				return activeSvcReport("dir", "svc").WritePlantUML(b, opts...)
			},
			wantSub: "left to right direction",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			direction := auditlog.DirectionUp
			if strings.Contains(tc.name, "left") {
				direction = auditlog.DirectionLeft
			}

			var buf bytes.Buffer

			if err := tc.write(&buf, auditlog.WithDirection(direction)); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}

			if !strings.Contains(buf.String(), tc.wantSub) {
				t.Errorf("expected %q in %s output:\n%s", tc.wantSub, tc.name, buf.String())
			}
		})
	}
}

// healthReport builds a report with two health-checked services (one healthy,
// one unhealthy) so the HTML health stat and per-service health cells render.
func healthReport() auditlog.Report {
	unhealthyErr := "connection refused"

	checkedAt := goldenExportedAt

	return auditlog.Report{
		Version:     auditlog.SchemaVersion,
		ContainerID: "health-test",
		Services: []auditlog.ServiceInfo{
			{
				ServiceIdentity: auditlog.ServiceIdentity{ServiceRef: rootRef("healthy-svc")},
				ServiceLifecycle: auditlog.ServiceLifecycle{
					Status: auditlog.ServiceStatusActive,
				},
				ServiceHealth: auditlog.ServiceHealth{
					IsHealthchecker:   true,
					LastHealthCheckAt: &checkedAt,
					HealthCheckCount:  2,
				},
			},
			{
				ServiceIdentity: auditlog.ServiceIdentity{ServiceRef: rootRef("sick-svc")},
				ServiceLifecycle: auditlog.ServiceLifecycle{
					Status: auditlog.ServiceStatusActive,
				},
				ServiceHealth: auditlog.ServiceHealth{
					IsHealthchecker:   true,
					LastHealthCheckAt: &checkedAt,
					HealthCheckError:  &unhealthyErr,
					HealthCheckCount:  1,
				},
			},
		},
		ScopeTree:            auditlog.ScopeNode{ID: "root", Name: "root"},
		TotalBuildDurationMs: 0,
		HealthCheckedCount:   2,
	}
}

// TestWriteHTML_HealthStats covers the health stat card, the healthy cell,
// and the unhealthy cell with embedded error message.
func TestWriteHTML_HealthStats(t *testing.T) {
	t.Parallel()

	report := healthReport()

	var buf bytes.Buffer

	if err := report.WriteHTML(&buf); err != nil {
		t.Fatalf("WriteHTML: %v", err)
	}

	html := buf.String()

	for _, want := range []string{
		"Health Checks",
		"2 checked",
		"1 unhealthy",
		"healthy",
		`stat-card error`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in HTML report with health checks", want)
		}
	}

	// The unhealthy error message must be embedded escaped, never raw HTML.
	if strings.Contains(html, ">connection refused<") && !strings.Contains(html, "unhealthy: connection refused") {
		t.Error("expected escaped unhealthy detail in HTML report")
	}
}

// TestLoadReport_UnsupportedFormat verifies the loader rejects out-of-range
// format codes through the public entry point.
func TestLoadReport_UnsupportedFormat(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromBytes([]byte("{}"), auditlog.Format(42))
	if err == nil {
		t.Fatal("expected error for unknown format code")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("expected 'unsupported format' in error, got: %v", err)
	}
}
