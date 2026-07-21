package auditlog_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// These tests exercise algebraic properties of Report.Diff over many
// randomly-generated report pairs. A fixed seed makes failures reproducible.

var diffStatuses = []auditlog.ServiceStatus{
	auditlog.ServiceStatusRegistered,
	auditlog.ServiceStatusActive,
	auditlog.ServiceStatusInvocationError,
	auditlog.ServiceStatusShutdownError,
	auditlog.ServiceStatusShutdown,
}

var diffNames = []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}

// randReport builds a pseudo-random Report from a deterministic RNG. Only the
// fields Diff inspects (Services, EventCount) are populated; the rest are left
// zero since Diff intentionally ignores timestamps, durations and scopes.
func randReport(rng *rand.Rand) auditlog.Report {
	n := rng.IntN(len(diffNames) + 1)
	namePool := slices.Clone(diffNames)

	for i := len(namePool) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		namePool[i], namePool[j] = namePool[j], namePool[i]
	}

	services := make([]auditlog.ServiceInfo, 0, n)
	for i := range n {
		services = append(services, auditlog.ServiceInfo{
			ServiceRef:       rootRef(namePool[i]),
			Status:           diffStatuses[rng.IntN(len(diffStatuses))],
			InvocationCount:  rng.IntN(10),
			HealthCheckCount: rng.IntN(5),
		})
	}

	return auditlog.Report{
		Services:   services,
		EventCount: rng.IntN(50),
	}
}

func TestReport_Diff_Identity(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 1))

	for range 200 {
		a := randReport(rng)
		if d := a.Diff(a); !d.IsEmpty() {
			t.Fatalf("Diff(a,a) must be empty, got %+v", d)
		}
	}
}

func TestReport_Diff_AddedRemovedDuality(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(2, 2))

	for range 200 {
		a, b := randReport(rng), randReport(rng)
		forward, reverse := a.Diff(b), b.Diff(a)

		if !slices.Equal(forward.AddedServices, reverse.RemovedServices) {
			t.Errorf("Added(a→b) != Removed(b→a)\n  forward added: %v\n  reverse removed: %v",
				forward.AddedServices, reverse.RemovedServices)
		}

		if !slices.Equal(forward.RemovedServices, reverse.AddedServices) {
			t.Errorf("Removed(a→b) != Added(b→a)\n  forward removed: %v\n  reverse added: %v",
				forward.RemovedServices, reverse.AddedServices)
		}
	}
}

func TestReport_Diff_EventCountAntiSymmetry(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(3, 3))

	for range 200 {
		a, b := randReport(rng), randReport(rng)
		forward, reverse := a.Diff(b), b.Diff(a)

		if forward.EventCountDelta != -reverse.EventCountDelta {
			t.Errorf("Δ(a→b)=%d should equal -Δ(b→a)=%d",
				forward.EventCountDelta, -reverse.EventCountDelta)
		}
	}
}

func TestReport_Diff_ChangedSymmetry(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(4, 4))

	for range 200 {
		a, b := randReport(rng), randReport(rng)
		forward, reverse := a.Diff(b), b.Diff(a)

		fwdBy := indexDiffs(forward.ChangedServices)
		revBy := indexDiffs(reverse.ChangedServices)

		if len(fwdBy) != len(revBy) {
			t.Fatalf("changed count mismatch: forward=%d reverse=%d", len(fwdBy), len(revBy))
		}

		for key, fd := range fwdBy {
			rd, ok := revBy[key]
			if !ok {
				t.Fatalf("service %q changed forward but not reverse", key)
			}

			if fd.InvocationCountDelta != -rd.InvocationCountDelta {
				t.Errorf("service %q invocation delta: fwd=%d should equal -rev=%d",
					key, fd.InvocationCountDelta, -rd.InvocationCountDelta)
			}

			if fd.HealthCheckCountDelta != -rd.HealthCheckCountDelta {
				t.Errorf("service %q health delta: fwd=%d should equal -rev=%d",
					key, fd.HealthCheckCountDelta, -rd.HealthCheckCountDelta)
			}
		}
	}
}

func TestReport_Diff_OutputSorted(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(5, 5))

	for range 200 {
		a, b := randReport(rng), randReport(rng)
		d := a.Diff(b)

		if !slices.IsSortedFunc(d.AddedServices, cmpServiceRef) {
			t.Error("AddedServices not sorted")
		}

		if !slices.IsSortedFunc(d.RemovedServices, cmpServiceRef) {
			t.Error("RemovedServices not sorted")
		}
	}
}

func indexDiffs(diffs []auditlog.ServiceDiff) map[string]auditlog.ServiceDiff {
	out := make(map[string]auditlog.ServiceDiff, len(diffs))
	for _, d := range diffs {
		out[d.ServiceName] = d
	}

	return out
}

func cmpServiceRef(a, b auditlog.ServiceRef) int {
	if a.ServiceName != b.ServiceName {
		if a.ServiceName < b.ServiceName {
			return -1
		}

		return 1
	}

	if a.ScopeID == b.ScopeID {
		return 0
	}

	if a.ScopeID < b.ScopeID {
		return -1
	}

	return 1
}
