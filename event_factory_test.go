package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// mkEvent builds a root-scope Event for replay tests. All replay test
// events use ScopeID "root" / ScopeName "[root]" — hardcoding it here
// reduces every 5-7 line struct literal to a single call. For events
// needing DurationMs or Error, set those fields on the returned value.
func mkEvent(
	seq int,
	ts time.Time,
	eventType auditlog.EventType,
	phase auditlog.Phase,
	serviceName auditlog.ServiceName, containerID auditlog.ContainerID,
	svcType auditlog.ProviderType,
) auditlog.Event {
	return auditlog.Event{
		ServiceRef:  rootRef(serviceName),
		Sequence:    seq,
		Timestamp:   ts,
		EventType:   eventType,
		Phase:       phase,
		ContainerID: containerID,
		ServiceType: svcType,
	}
}

// mkEventWithDur wraps mkEvent and sets DurationMs — used where the
// invocation-after event needs to carry timing data.
func mkEventWithDur(
	seq int,
	ts time.Time,
	eventType auditlog.EventType,
	phase auditlog.Phase,
	serviceName auditlog.ServiceName, containerID auditlog.ContainerID,
	svcType auditlog.ProviderType,
	dur float64,
) auditlog.Event {
	e := mkEvent(seq, ts, eventType, phase, serviceName, containerID, svcType)
	e.DurationMs = &dur

	return e
}

// mkRegEvent is a shorthand for a registration-after event. Centralizes the
// 9-line mkEvent(...) block used by every manual event-fixture test.
func mkRegEvent(
	seq int,
	ts time.Time,
	serviceName auditlog.ServiceName,
	containerID auditlog.ContainerID,
) auditlog.Event {
	return mkLazyEvent(seq, ts, auditlog.EventTypeRegistration, auditlog.PhaseAfter, serviceName, containerID)
}

// mkInvAfterWithDur is a shorthand for an invocation-after event with duration.
// Centralizes the 9-line mkEventWithDur(...) block used by every
// double-invocation replay test.
func mkInvAfterWithDur(
	seq int,
	ts time.Time,
	serviceName auditlog.ServiceName,
	containerID auditlog.ContainerID,
	dur float64,
) auditlog.Event {
	return mkLazyEventWithDur(
		seq, ts,
		auditlog.EventTypeInvocation, auditlog.PhaseAfter,
		serviceName, containerID,
		dur,
	)
}

// containerIDForTest is the fixed ContainerID used by every nested-invocation
// replay test. Centralizes the literal so callers stay short.
const containerIDForTest = "c"

// mkInvAfter is a shorthand for an invocation-after event (no duration).
// Centralizes the 9-line mkEvent(...) block used by every nested-invocation
// replay test where only the call order matters. ContainerID is fixed to
// containerIDForTest because every existing caller passes the same value; if
// a test ever needs a different container, inline a mkLazyEvent call.
func mkInvAfter(seq int, ts time.Time, serviceName string) auditlog.Event {
	return mkLazyEvent(
		seq,
		ts,
		auditlog.EventTypeInvocation,
		auditlog.PhaseAfter,
		auditlog.ServiceName(serviceName),
		containerIDForTest,
	)
}

// mkInvBefore is a shorthand for an invocation-before event with empty
// ServiceType (matches what recorder.go emits for the before phase). Used by
// every nested-invocation replay test for the "push to stack" half of a call.
// ContainerID is fixed to containerIDForTest; see mkInvAfter for rationale.
func mkInvBefore(seq int, ts time.Time, serviceName string) auditlog.Event {
	return mkEvent(
		seq,
		ts,
		auditlog.EventTypeInvocation,
		auditlog.PhaseBefore,
		auditlog.ServiceName(serviceName),
		containerIDForTest,
		"",
	)
}

// mkLazyEvent is the shared lazy-provider mkEvent(...) body used by every
// typed-event shorthand above. Centralizes the 9-line call shared by mkRegEvent,
// mkInvAfter, and any future shorthand for the ProviderTypeLazy default.
func mkLazyEvent(
	seq int,
	ts time.Time,
	eventType auditlog.EventType,
	phase auditlog.Phase,
	serviceName auditlog.ServiceName, containerID auditlog.ContainerID,
) auditlog.Event {
	return mkEvent(seq, ts, eventType, phase, serviceName, containerID, auditlog.ProviderTypeLazy)
}

// mkLazyEventWithDur is the duration-carrying variant of mkLazyEvent, used by
// mkInvAfterWithDur. Same 9-line shape with DurationMs appended.
func mkLazyEventWithDur(
	seq int,
	ts time.Time,
	eventType auditlog.EventType,
	phase auditlog.Phase,
	serviceName auditlog.ServiceName, containerID auditlog.ContainerID,
	dur float64,
) auditlog.Event {
	return mkEventWithDur(seq, ts, eventType, phase, serviceName, containerID, auditlog.ProviderTypeLazy, dur)
}

// loadReportOK calls auditlog.LoadReportFromBytes and fails the test on error.
// Centralizes the 4-line "LoadReportFromBytes(...) + t.Fatalf(err)" preamble
// shared by every successful round-trip test in this file.
func loadReportOK(t *testing.T, data []byte, format auditlog.Format) (auditlog.Report, auditlog.Format) {
	t.Helper()

	report, gotFormat, err := auditlog.LoadReportFromBytes(data, format)
	if err != nil {
		t.Fatalf("LoadReportFromBytes: %v", err)
	}

	return report, gotFormat
}
