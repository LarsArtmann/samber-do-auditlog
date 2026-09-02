package auditlog_test

import (
	"encoding/json"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// depServiceFixture builds a root-scope service with the given dependency
// refs. Centralizes the struct literal so each DepsChanged test stays 1 line.
func depServiceFixture(name auditlog.ServiceName, exported time.Time, deps []auditlog.ServiceRef) auditlog.ServiceInfo {
	return auditlog.ServiceInfo{
		ServiceIdentity:  auditlog.ServiceIdentity{ServiceRef: rootRef(name)},
		ServiceLifecycle: auditlog.ServiceLifecycle{RegisteredAt: exported},
		ServiceGraph:     auditlog.ServiceGraph{Dependencies: deps},
	}
}

func TestReport_Diff_DepsAdded(t *testing.T) {
	t.Parallel()

	now := epochTime

	r1 := mkNewReport(t, "c1", now, []auditlog.ServiceInfo{
		depServiceFixture("api", now, []auditlog.ServiceRef{rootRef("db")}),
		depServiceFixture("db", now, nil),
	}, rootScopeTree("api", "db"))

	r2 := mkNewReport(t, "c1", now, []auditlog.ServiceInfo{
		depServiceFixture("api", now, []auditlog.ServiceRef{rootRef("cache"), rootRef("db")}),
		depServiceFixture("db", now, nil),
	}, rootScopeTree("api", "db"))

	diff := r1.Diff(r2)

	if len(diff.ChangedServices) != 1 {
		t.Fatalf("expected exactly 1 changed service, got %d", len(diff.ChangedServices))
	}

	changed := diff.ChangedServices[0]
	if changed.ServiceName != "api" {
		t.Fatalf("expected api to change, got %s", changed.ServiceName)
	}

	if len(changed.AddedDeps) != 1 || changed.AddedDeps[0].ServiceName != "cache" {
		t.Errorf("AddedDeps: want [cache], got %v", changed.AddedDeps)
	}

	if len(changed.RemovedDeps) != 0 {
		t.Errorf("RemovedDeps: want empty, got %v", changed.RemovedDeps)
	}
}

func TestReport_Diff_DepsRemoved(t *testing.T) {
	t.Parallel()

	now := epochTime

	r1 := mkNewReport(t, "c1", now, []auditlog.ServiceInfo{
		depServiceFixture("api", now, []auditlog.ServiceRef{rootRef("cache"), rootRef("db")}),
		depServiceFixture("cache", now, nil),
		depServiceFixture("db", now, nil),
	}, rootScopeTree("api", "cache", "db"))

	r2 := mkNewReport(t, "c1", now, []auditlog.ServiceInfo{
		depServiceFixture("api", now, []auditlog.ServiceRef{rootRef("db")}),
		depServiceFixture("cache", now, nil),
		depServiceFixture("db", now, nil),
	}, rootScopeTree("api", "cache", "db"))

	diff := r1.Diff(r2)

	if len(diff.ChangedServices) != 1 {
		t.Fatalf("expected exactly 1 changed service, got %d", len(diff.ChangedServices))
	}

	changed := diff.ChangedServices[0]
	if changed.ServiceName != "api" {
		t.Fatalf("expected api to change, got %s", changed.ServiceName)
	}

	if len(changed.RemovedDeps) != 1 || changed.RemovedDeps[0].ServiceName != "cache" {
		t.Errorf("RemovedDeps: want [cache], got %v", changed.RemovedDeps)
	}

	if len(changed.AddedDeps) != 0 {
		t.Errorf("AddedDeps: want empty, got %v", changed.AddedDeps)
	}
}

func TestReport_Diff_DepsRewired(t *testing.T) {
	t.Parallel()

	now := epochTime

	r1 := mkNewReport(t, "c1", now, []auditlog.ServiceInfo{
		depServiceFixture("api", now, []auditlog.ServiceRef{rootRef("db")}),
		depServiceFixture("cache", now, nil),
		depServiceFixture("db", now, nil),
	}, rootScopeTree("api", "cache", "db"))

	r2 := mkNewReport(t, "c1", now, []auditlog.ServiceInfo{
		depServiceFixture("api", now, []auditlog.ServiceRef{rootRef("cache")}),
		depServiceFixture("cache", now, nil),
		depServiceFixture("db", now, nil),
	}, rootScopeTree("api", "cache", "db"))

	diff := r1.Diff(r2)

	if len(diff.ChangedServices) != 1 {
		t.Fatalf("expected exactly 1 changed service, got %d", len(diff.ChangedServices))
	}

	changed := diff.ChangedServices[0]
	if len(changed.AddedDeps) != 1 || changed.AddedDeps[0].ServiceName != "cache" {
		t.Errorf("AddedDeps: want [cache], got %v", changed.AddedDeps)
	}

	if len(changed.RemovedDeps) != 1 || changed.RemovedDeps[0].ServiceName != "db" {
		t.Errorf("RemovedDeps: want [db], got %v", changed.RemovedDeps)
	}
}

func TestReport_Diff_DepsUnchanged_NoEntry(t *testing.T) {
	t.Parallel()

	now := epochTime

	services := []auditlog.ServiceInfo{
		depServiceFixture("api", now, []auditlog.ServiceRef{rootRef("db")}),
		depServiceFixture("db", now, nil),
	}

	r1 := mkNewReport(t, "c1", now, services, rootScopeTree("api", "db"))
	r2 := mkNewReport(t, "c1", now, services, rootScopeTree("api", "db"))

	diff := r1.Diff(r2)

	if !diff.IsEmpty() {
		t.Errorf("expected identical reports to produce an empty diff, got %+v", diff)
	}
}

func TestServiceDiff_DepsOmitEmptyJSON(t *testing.T) {
	t.Parallel()

	// A ServiceDiff with no dep changes must not emit the dep keys at all.
	diff := auditlog.ServiceDiff{
		ServiceRef:    rootRef("api"),
		StatusChanged: true,
	}

	data, err := json.Marshal(diff)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded map[string]any

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	for _, key := range []string{"added_deps", "removed_deps"} {
		if _, exists := decoded[key]; exists {
			t.Errorf("JSON output must omit %q when empty, got %s", key, data)
		}
	}
}
