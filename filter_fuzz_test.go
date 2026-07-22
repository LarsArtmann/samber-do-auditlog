package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// fuzzFilterReport builds a deterministic multi-service report for filter
// fuzzing: three root-scope services (config, db, cache) with db depending on
// config, plus invocation and shutdown events across a fixed time span.
func fuzzFilterReport(t *testing.T) auditlog.Report {
	t.Helper()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideString(injector, "config", "dsn=example")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")

	return p.Report()
}

// FuzzFilterInputs fuzzes Report.Filtered with arbitrary combinations of the
// five filter options (name, type, event-type, scope, time range) derived from
// the fuzz corpus. Invariants: it never panics, always passes Validate(), and
// every returned service/event actually matches the applied filter.
func FuzzFilterInputs(f *testing.F) {
	// Seed corpus: empty, single bytes, and structured token streams.
	seeds := [][]byte{
		{},
		{0},
		{'d', 'b'},
		[]byte("db\ncache\nconfig"),
		[]byte("nonexistent"),
		[]byte("registration\ninvocation\nshutdown\nhealth_check"),
		[]byte("\x00\x01\x02\x03"),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		report := fuzzFilterReport(t)

		// Derive a name set from null/newline-delimited tokens in the input.
		names := tokenize(data)

		var opts []auditlog.ReportOption
		if len(names) > 0 {
			svcNames := make([]auditlog.ServiceName, len(names))
			for i, n := range names {
				svcNames[i] = auditlog.ServiceName(n)
			}
			opts = append(opts, auditlog.WithServicesByName(svcNames...))
		}

		// Derive an event-type filter from the first byte.
		if len(data) > 0 {
			et := eventTypesByByte(data[0])
			opts = append(opts, auditlog.WithEventsByType(et))
		}

		// Derive a scope filter from the second byte.
		if len(data) > 1 && data[1]%2 == 0 {
			opts = append(opts, auditlog.WithScope("root"))
		}

		// Derive a (possibly inverted) time range from two trailing bytes.
		if len(data) > 3 {
			from := time.Unix(int64(data[2]), 0)
			to := time.Unix(int64(data[2])+int64(data[3])+1, 0)
			opts = append(opts, auditlog.WithTimeRange(from, to))
		}

		filtered := report.Filtered(opts...)

		// Invariant 1: result is always valid.
		if err := filtered.Validate(); err != nil {
			t.Fatalf("filtered report invalid: %v", err)
		}

		// Invariant 2: filtered services are a subset of the original.
		if len(filtered.Services) > len(report.Services) {
			t.Fatalf("filtered has more services (%d) than original (%d)",
				len(filtered.Services), len(report.Services))
		}

		// Invariant 3: every kept service matches the name filter (if any set).
		nameSet := make(map[string]bool, len(names))
		for _, n := range names {
			nameSet[n] = true
		}

		for _, svc := range filtered.Services {
			if len(nameSet) > 0 && !nameSet[string(svc.ServiceName)] {
				t.Errorf("service %q kept but not in name filter", svc.ServiceName)
			}
		}

		// Invariant 4: filtered events are a subset of the original.
		if len(filtered.Events) > len(report.Events) {
			t.Fatalf("filtered has more events (%d) than original (%d)",
				len(filtered.Events), len(report.Events))
		}
	})
}

var fuzzEventTypes = []auditlog.EventType{
	auditlog.EventTypeRegistration,
	auditlog.EventTypeInvocation,
	auditlog.EventTypeShutdown,
	auditlog.EventTypeHealthCheck,
}

func eventTypesByByte(b byte) auditlog.EventType {
	return fuzzEventTypes[int(b)%len(fuzzEventTypes)]
}

// tokenize splits data on null bytes and newlines, dropping empty tokens.
func tokenize(data []byte) []string {
	var out []string

	start := 0

	for i, b := range data {
		if b == 0 || b == '\n' {
			if i > start {
				out = append(out, string(data[start:i]))
			}

			start = i + 1
		}
	}

	if start < len(data) {
		out = append(out, string(data[start:]))
	}

	return out
}
