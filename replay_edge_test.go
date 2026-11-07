package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestReplayEvents_NonLIFOStackPop(t *testing.T) {
	t.Parallel()

	now := time.Now()

	// Stack: push A, push B. Pop A FIRST (not last), then B.
	// This triggers the append(stack[:i], stack[i+1:]...) path.
	events := []auditlog.Event{
		mkRegEvent(1, now, "a", "c"),
		mkRegEvent(2, now, "b", "c"),
		mkInvBefore(3, now, "a"),
		mkInvBefore(4, now, "b"),
		// A finishes while B is still on stack — pops from middle.
		mkInvAfter(5, now, "a"),
		mkInvAfter(6, now, "b"),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svcA := findServiceByName(t, report, "a")
	if svcA == nil {
		t.Fatal("a not found")
	}

	assertServiceInvocationCount(t, svcA, 1)
}

// TestReplayEvents_DoubleInvocation covers the firstInvokedAt-already-set
// branch in applyInvocationAfter.
func TestReplayEvents_DoubleInvocation(t *testing.T) {
	t.Parallel()

	now := time.Now()

	events := []auditlog.Event{
		mkRegEvent(1, now, "svc", "c"),
		// First invocation.
		mkInvBefore(2, now, "svc"),
		mkInvAfterWithDur(3, now, "svc", "c", 3.3),
		// Second invocation — firstInvokedAt already set, firstBuildDurationMs already set.
		mkInvBefore(4, now, "svc"),
		mkInvAfterWithDur(5, now, "svc", "c", 3.3),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "svc")
	if svc == nil {
		t.Fatal("svc not found")
	}

	if svc.InvocationCount != 2 {
		t.Errorf("invocation count: want 2, got %d", svc.InvocationCount)
	}
}

// TestReplayEvents_ShutdownWithMatchingBefore covers the shutdownStart
// path in applyShutdownAfter where a matching before event exists.
func TestReplayEvents_ShutdownWithMatchingBefore(t *testing.T) {
	t.Parallel()

	t0 := time.Now()
	t1 := t0.Add(5 * time.Millisecond)

	events := []auditlog.Event{
		mkRegEvent(1, t0, "svc", "c"),
		mkInvBefore(2, t0, "svc"),
		mkInvAfter(3, t0, "svc"),
		// Shutdown with matching before event.
		mkEvent(4, t0, auditlog.EventTypeShutdown, auditlog.PhaseBefore, "svc", "c", ""),
		mkEvent(5, t1, auditlog.EventTypeShutdown, auditlog.PhaseAfter, "svc", "c", auditlog.ProviderTypeLazy),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "svc")
	if svc == nil {
		t.Fatal("svc not found")
	}

	assertServiceStatus(t, svc, auditlog.ServiceStatusShutdown)

	if svc.ShutdownDurationMs == nil {
		t.Error("expected non-nil shutdown duration from matching before")
	}
}

// TestReplayEvents_InvocationWithoutRegistration covers the !ok branch
// in applyInvocationAfter where a service is invoked without prior registration.
func TestReplayEvents_InvocationWithoutRegistration(t *testing.T) {
	t.Parallel()

	now := time.Now()

	events := []auditlog.Event{
		mkInvBefore(1, now, "ghost"),
		mkInvAfterWithDur(2, now, "ghost", "c", 2.0),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "ghost")
	if svc == nil {
		t.Fatal("ghost not found")
	}

	assertServiceInvocationCount(t, svc, 1)
}

// TestReplayEvents_ShutdownWithoutRegistration covers the !ok branch
// in applyShutdownAfter where a service is shut down without prior registration.
func TestReplayEvents_ShutdownWithoutRegistration(t *testing.T) {
	t.Parallel()

	now := time.Now()

	events := []auditlog.Event{
		mkEvent(1, now, auditlog.EventTypeShutdown, auditlog.PhaseBefore, "phantom", "c", ""),
		mkEvent(
			2,
			now.Add(time.Millisecond),
			auditlog.EventTypeShutdown,
			auditlog.PhaseAfter,
			"phantom",
			"c",
			auditlog.ProviderTypeLazy,
		),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "phantom")
	if svc == nil {
		t.Fatal("phantom not found")
	}

	assertServiceStatus(t, svc, auditlog.ServiceStatusShutdown)
}

// TestReplayEvents_EmptyContainerID covers the containerID fallback
// path in ReplayEvents.
func TestReplayEvents_EmptyContainerID(t *testing.T) {
	t.Parallel()

	now := time.Now()

	events := []auditlog.Event{
		mkRegEvent(1, now, "svc", ""),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	// ContainerID should be empty (from the event).
	if report.ContainerID != "" {
		t.Errorf("expected empty container ID, got %q", report.ContainerID)
	}
}

// TestReplayEvents_InvocationOrder verifies that the replay engine assigns
// correct cross-service invocation order (regression test for a bug where
// invocationOrder was always 0 for every service).
func TestReplayEvents_InvocationOrder(t *testing.T) {
	t.Parallel()

	base := epochTime

	// Two services invoked in sequence: config first, then db.
	// invocationOrder must reflect the global invocation sequence, not per-service count.
	mk := func(seq int, name string, offset time.Duration, eventType auditlog.EventType, phase auditlog.Phase) auditlog.Event {
		return mkEvent(
			seq,
			base.Add(offset),
			eventType,
			phase,
			auditlog.ServiceName(name),
			auditlog.ContainerID("test"),
			auditlog.ProviderTypeLazy,
		)
	}

	events := []auditlog.Event{
		mk(1, "config", 0, auditlog.EventTypeRegistration, auditlog.PhaseAfter),
		mk(2, "db", time.Millisecond, auditlog.EventTypeRegistration, auditlog.PhaseAfter),
		mk(3, "config", 2*time.Millisecond, auditlog.EventTypeInvocation, auditlog.PhaseBefore),
		mk(4, "config", 3*time.Millisecond, auditlog.EventTypeInvocation, auditlog.PhaseAfter),
		mk(5, "db", 4*time.Millisecond, auditlog.EventTypeInvocation, auditlog.PhaseBefore),
		mk(6, "db", 5*time.Millisecond, auditlog.EventTypeInvocation, auditlog.PhaseAfter),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	configSvc := findServiceByName(t, report, "config")
	if configSvc.InvocationOrder != 0 {
		t.Errorf("config invocationOrder: want 0, got %d", configSvc.InvocationOrder)
	}

	dbSvc := findServiceByName(t, report, "db")
	if dbSvc.InvocationOrder != 1 {
		t.Errorf("db invocationOrder: want 1, got %d", dbSvc.InvocationOrder)
	}
}
