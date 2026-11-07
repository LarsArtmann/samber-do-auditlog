package auditlog_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func FuzzMigrateReport(f *testing.F) {
	seeds := []string{
		`{"version":"0.1.0"}`,
		`{"version":"0.2.0"}`,
		`{"version":"0.3.0"}`,
		`{"version":"0.1.0","services":[{"service_name":"svc","scope_id":"r","scope_name":"[root]"}]}`,
		`{"version":"0.1.0","scope_tree":{"id":"root","name":"[root]","services":[],"children":[{"id":"child","name":"child","services":[],"children":[]}]}}`,
		`not json`,
		`{}`,
		``,
	}

	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		report, err := auditlog.MigrateReport(data)
		if err != nil {
			// Errors are expected for malformed/empty input.
			return
		}

		assertReportValidNoFatal(t, report, "migrated")

		// Re-migrating a current-schema report should be a no-op and stay valid.
		var buf bytes.Buffer

		enc := jsontext.NewEncoder(&buf)
		if encodeErr := json.MarshalEncode(enc, report); encodeErr != nil {
			t.Fatalf("encode migrated report: %v", encodeErr)
		}

		reMigrated, reErr := auditlog.MigrateReport(buf.Bytes())
		if reErr != nil {
			t.Errorf("re-migrating current-schema report failed: %v", reErr)
		}

		if reValidateErr := reMigrated.Validate(); reValidateErr != nil {
			t.Errorf("re-migrated report failed validation: %v", reValidateErr)
		}
	})
}

func TestNestedScopeExport(t *testing.T) {
	t.Parallel()

	depths := []int{0, 1, 5, 10, 50, 100, 200}

	for _, depth := range depths {
		t.Run(fmt.Sprintf("depth-%d", depth), func(t *testing.T) {
			t.Parallel()

			node := auditlog.ScopeNode{ID: "root", Name: "[root]", Services: []auditlog.ServiceName{"root-svc"}}
			current := &node

			for i := range depth {
				child := auditlog.ScopeNode{
					ID:       auditlog.ScopeID(fmt.Sprintf("scope-%d", i)),
					Name:     fmt.Sprintf("scope-%d", i),
					Services: []auditlog.ServiceName{auditlog.ServiceName(fmt.Sprintf("svc-%d", i))},
				}
				current.Children = []auditlog.ScopeNode{child}
				current = &current.Children[0]
			}

			services := make([]auditlog.ServiceInfo, 0, depth+1)
			services = append(services, auditlog.ServiceInfo{
				ServiceRef: rootRef("root-svc"),
				Status:     auditlog.ServiceStatusActive,
			})

			for i := range depth {
				services = append(services, auditlog.ServiceInfo{
					ScopeID:     auditlog.ScopeID(fmt.Sprintf("scope-%d", i)),
					ScopeName:   fmt.Sprintf("scope-%d", i),
					ServiceName: auditlog.ServiceName(fmt.Sprintf("svc-%d", i)),
					Status:      auditlog.ServiceStatusActive,
				})
			}

			rawReport := auditlog.Report{
				Version:   auditlog.SchemaVersion,
				ScopeTree: node,
				Services:  services,
			}

			// Normalize via MigrateReport so all denormalized fields are set.
			var rawBuf bytes.Buffer

			enc := jsontext.NewEncoder(&rawBuf)
			if encodeErr := json.MarshalEncode(enc, rawReport); encodeErr != nil {
				t.Fatalf("JSON encode error: %v", encodeErr)
			}

			report, migErr := auditlog.MigrateReport(rawBuf.Bytes())
			if migErr != nil {
				t.Fatalf("MigrateReport error: %v", migErr)
			}

			assertReportValidNoFatal(t, report, "deeply nested")

			var jsonBuf bytes.Buffer

			if jsonErr := report.WriteJSON(&jsonBuf); jsonErr != nil {
				t.Fatalf("WriteJSON error: %v", jsonErr)
			}

			var mBuf bytes.Buffer

			_ = report.WriteMermaid(&mBuf)

			var pBuf bytes.Buffer

			_ = report.WritePlantUML(&pBuf)
		})
	}
}
