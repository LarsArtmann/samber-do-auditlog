package live

import (
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestComputeLegendItems(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	report := auditlog.Report{
		Services: []auditlog.ServiceInfo{
			{ServiceType: "lazy"},
			{ServiceType: "lazy"},
			{ServiceType: "eager"},
		},
	}

	items := computeLegendItems(report, meta)
	if len(items) != 2 {
		t.Fatalf("expected 2 legend items, got %d", len(items))
	}

	if items[0].Count != 2 {
		t.Errorf("first item count = %d, want 2", items[0].Count)
	}
}

func TestRowSignalsJSON(t *testing.T) {
	t.Parallel()

	svc := auditlog.ServiceInfo{
		ServiceName: "db", ScopeName: "[root]",
	}

	json := rowSignalsJSON(svc, 3)
	if !strings.Contains(json, `"rowName":"db"`) {
		t.Errorf("rowSignalsJSON missing rowName: %s", json)
	}
}

func TestEventRowSignalsJSON(t *testing.T) {
	t.Parallel()

	evt := auditlog.Event{EventType: "invocation"}

	json := eventRowSignalsJSON(evt, 5)
	if !strings.Contains(json, `"evtType":"invocation"`) {
		t.Errorf("eventRowSignalsJSON missing evtType: %s", json)
	}
}

// --- Templ component rendering tests ---
