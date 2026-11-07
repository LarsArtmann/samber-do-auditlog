package auditlog_test

import (
	"bufio"
	"errors"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

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
		t.Errorf("expected >=3 added, got %d", len(diff.AddedServices))
	}

	if len(diff.RemovedServices) < 3 {
		t.Errorf("expected >=3 removed, got %d", len(diff.RemovedServices))
	}
}

func TestDiff_MultipleChanged(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mk := func(seq int, name string, eventType auditlog.EventType, phase auditlog.Phase) auditlog.Event {
		return mkEvent(
			seq,
			now,
			eventType,
			phase,
			auditlog.ServiceName(name),
			auditlog.ContainerID("c"),
			auditlog.ProviderTypeLazy,
		)
	}

	sharedRegInv := []auditlog.Event{
		mk(1, "svc-a", auditlog.EventTypeRegistration, auditlog.PhaseAfter),
		mk(2, "svc-b", auditlog.EventTypeRegistration, auditlog.PhaseAfter),
		mk(3, "svc-a", auditlog.EventTypeInvocation, auditlog.PhaseBefore),
		mk(4, "svc-a", auditlog.EventTypeInvocation, auditlog.PhaseAfter),
		mk(5, "svc-b", auditlog.EventTypeInvocation, auditlog.PhaseBefore),
		mk(6, "svc-b", auditlog.EventTypeInvocation, auditlog.PhaseAfter),
	}

	eventsA := append([]auditlog.Event(nil), sharedRegInv...)
	eventsA = append(
		eventsA,
		mk(7, "svc-a", auditlog.EventTypeHealthCheck, auditlog.PhaseAfter),
		mk(8, "svc-b", auditlog.EventTypeHealthCheck, auditlog.PhaseAfter),
	)

	eventsB := append([]auditlog.Event(nil), sharedRegInv...)

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

// --- Test helpers ---

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errReaderFailed
}

var errReaderFailed = errors.New("simulated read failure")

// Ensure bufio is used (referenced in oversized line tests via scanner internals).
var _ = bufio.ErrTooLong

// --- Targeted branch coverage tests ---

// TestReplayEvents_NonLIFOStackPop exercises the non-LIFO branch in
// applyInvocationAfter where i < len(stack)-1 (middle-of-stack pop).
