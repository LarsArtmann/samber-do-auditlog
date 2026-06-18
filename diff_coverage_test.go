package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// TestDiff_MultipleAddedRemoved covers sortServiceRefs in diff.go.
func TestDiff_MultipleAddedRemoved(t *testing.T) {
	t.Parallel()

	pluginA, injectorA := newPluginAndInjector()
	provideDB(injectorA, "db-a", "postgres://a")
	provideCache(injectorA, "cache-a")
	provideString(injectorA, "str-a", "value-a")

	reportA := pluginA.Report()

	pluginB, injectorB := newPluginAndInjector()
	provideDB(injectorB, "db-b", "postgres://b")
	provideCache(injectorB, "cache-b")
	provideString(injectorB, "str-b", "value-b")
	provideString(injectorB, "extra", "extra")

	reportB := pluginB.Report()

	diff := reportA.Diff(reportB)

	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}

	if len(diff.AddedServices) < 3 {
		t.Errorf("expected >=3 added services, got %d", len(diff.AddedServices))
	}

	if len(diff.RemovedServices) < 3 {
		t.Errorf("expected >=3 removed services, got %d", len(diff.RemovedServices))
	}
}

// TestDiff_MultipleChanged covers sortServiceDiffs in diff.go by using
// manually constructed events with matching scope IDs.
func TestDiff_MultipleChanged(t *testing.T) {
	t.Parallel()

	ref1 := auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc-a"}
	ref2 := auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc-b"}
	now := time.Now()

	// Report A: 2 services with health checks.
	eventsA := []auditlog.Event{
		{
			ServiceRef:  ref1,
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref2,
			Sequence:    2,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref1,
			Sequence:    3,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref1,
			Sequence:    4,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref2,
			Sequence:    5,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref2,
			Sequence:    6,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref1,
			Sequence:    7,
			Timestamp:   now,
			EventType:   auditlog.EventTypeHealthCheck,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref2,
			Sequence:    8,
			Timestamp:   now,
			EventType:   auditlog.EventTypeHealthCheck,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
		},
	}

	// Report B: same 2 services, no health checks (delta != 0).
	eventsB := []auditlog.Event{
		{
			ServiceRef:  ref1,
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref2,
			Sequence:    2,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref1,
			Sequence:    3,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref1,
			Sequence:    4,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref2,
			Sequence:    5,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref2,
			Sequence:    6,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	reportA, err := auditlog.ReplayEvents(eventsA)
	if err != nil {
		t.Fatalf("ReplayEvents A: %v", err)
	}

	reportB, err := auditlog.ReplayEvents(eventsB)
	if err != nil {
		t.Fatalf("ReplayEvents B: %v", err)
	}

	diff := reportA.Diff(reportB)

	if len(diff.ChangedServices) < 2 {
		t.Errorf("expected >=2 changed services, got %d", len(diff.ChangedServices))
	}
}
