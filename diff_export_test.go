package auditlog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReport_WriteNDJSON(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	var buf bytes.Buffer

	err := report.WriteNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteNDJSON: %v", err)
	}

	lines := ndjsonLines(buf.String())
	if len(lines) != report.EventCount {
		t.Fatalf("expected %d NDJSON lines, got %d", report.EventCount, len(lines))
	}

	// Each line must be valid JSON with a sequence field.
	for i, line := range lines {
		var event map[string]any

		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i, err)
		}

		if _, ok := event["sequence"]; !ok {
			t.Errorf("line %d missing sequence field", i)
		}
	}
}

func TestReport_WriteJSON(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	var buf bytes.Buffer

	err := report.WriteJSON(&buf)
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	// Must be valid JSON with the version field.
	var parsed map[string]any

	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed["version"] != auditlog.SchemaVersion {
		t.Errorf("version: want %s, got %v", auditlog.SchemaVersion, parsed["version"])
	}
}

func TestReport_Diff_IdenticalReports(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	diff := report.Diff(report)

	if !diff.IsEmpty() {
		t.Fatalf("identical reports should have empty diff, got %+v", diff)
	}
}

func TestReport_Diff_AddedRemovedServices(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")

	r1 := p.Report()

	provideCache(injector, "cache")

	r2 := p.Report()

	diff := r1.Diff(r2)

	// r2 has "cache" that r1 doesn't.
	if len(diff.AddedServices) != 1 {
		t.Fatalf("expected 1 added service, got %d: %+v", len(diff.AddedServices), diff.AddedServices)
	}

	if diff.AddedServices[0].ServiceName != "cache" {
		t.Errorf("added service: want cache, got %s", diff.AddedServices[0].ServiceName)
	}

	// Reverse: r1 should see cache as removed.
	reverseDiff := r2.Diff(r1)

	if len(reverseDiff.RemovedServices) != 1 {
		t.Fatalf("expected 1 removed service, got %d", len(reverseDiff.RemovedServices))
	}
}

func TestReport_Diff_ChangedService(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	r1 := p.Report()

	_ = do.MustInvokeNamed[*Database](injector, "db") // extra invocation

	r2 := p.Report()

	diff := r1.Diff(r2)

	// "db" should appear as changed (invocation count delta = 1).
	var found bool

	for _, changed := range diff.ChangedServices {
		if changed.ServiceName == "db" {
			found = true

			if changed.InvocationCountDelta != 1 {
				t.Errorf("invocation count delta: want 1, got %d", changed.InvocationCountDelta)
			}
		}
	}

	if !found {
		t.Error("expected db to appear in changed services")
	}
}

func TestReport_Diff_NewError(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	r1 := p.Report()

	provideFailing(injector, "failing")
	_, _ = do.InvokeNamed[*Database](injector, "failing") // triggers invocation error

	r2 := p.Report()

	diff := r1.Diff(r2)

	// "failing" is added (new service), and it has an error.
	var foundFailing bool

	for _, added := range diff.AddedServices {
		if added.ServiceName == "failing" {
			foundFailing = true
		}
	}

	if !foundFailing {
		t.Error("expected failing to be in added services")
	}
}
