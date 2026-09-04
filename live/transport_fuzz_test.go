package live

import (
	"strconv"
	"strings"
	"testing"
)

// FuzzSSEWireRoundTrip feeds arbitrary name/id/data payloads through
// writeSSEEvent and re-parses the frame, enforcing the SSE framing
// guarantees the live dashboard depends on:
//  1. no CR survives (spec normalization to LF),
//  2. exactly one frame terminator ("\n\n"), at the very end,
//  3. every non-terminator line is a recognized field ("event:", "data:",
//     "id:", "retry:") — i.e. multi-line data can never inject new fields,
//  4. the data lines join back to the CR/LF-normalized input.
func FuzzSSEWireRoundTrip(f *testing.F) {
	f.Add("patch-elements", "<div id=\"x\"></div>", "42", uint(0))
	f.Add("patch-signals", "signals {\"a\": 1}\nsignals {\"b\": 2}", "", uint(1500))
	f.Add("", "", "", uint(0))
	f.Add("event", "line1\r\nline2\rline3\nline4", "7", uint(1))
	f.Add("data: injection attempt", "\n\nevent: evil\n\n", "1", uint(0))
	f.Add("unicode 🚀", "🚀\r\n💥", "✔", uint(9))

	f.Fuzz(func(t *testing.T, name, data, id string, retry uint) {
		evt := sseEvent{Name: name, Data: data, ID: id, Retry: retry}

		var buf strings.Builder

		if err := writeSSEEvent(&buf, evt); err != nil {
			t.Fatalf("writeSSEEvent: %v", err)
		}

		out := buf.String()

		if strings.ContainsRune(out, '\r') {
			t.Fatalf("frame contains CR (not normalized): %q", out)
		}

		if n := strings.Count(out, "\n\n"); n != 1 || !strings.HasSuffix(out, "\n\n") {
			t.Fatalf("frame must end with exactly one terminator, got %d in %q", n, out)
		}

		lines := strings.Split(out, "\n")
		fields := lines[:len(lines)-2]

		var dataLines []string

		for i, line := range fields {
			switch {
			case strings.HasPrefix(line, "data: "):
				dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
			case strings.HasPrefix(line, "event: "),
				strings.HasPrefix(line, "id: "),
				strings.HasPrefix(line, "retry: "):
			default:
				t.Fatalf("line %d is not a recognized field: %q (frame %q)", i, line, out)
			}
		}

		wantData := strings.Join(sseSplitLines(data), "\n")
		gotData := strings.Join(dataLines, "\n")

		if gotData != wantData {
			t.Fatalf("data round-trip mismatch:\n got: %q\nwant: %q", gotData, wantData)
		}
	})
}

// FuzzRingEventsAfter fuzzes the reconnection-replay slice with arbitrary
// Last-Event-ID values and ring sizes, enforcing:
//  1. an unparseable lastID yields nil (never replays on garbage),
//  2. the result contains exactly the retained events with numeric ID
//     strictly greater than lastID,
//  3. the result is ordered ascending by numeric ID,
//  4. the result never exceeds the ring capacity.
func FuzzRingEventsAfter(f *testing.F) {
	f.Add("3", 8, 4)
	f.Add("", 8, 4)
	f.Add("-1", 8, 4)
	f.Add("99999999999999999999999", 8, 4)
	f.Add("0", 1, 1)
	f.Add("١٢٣", 4, 2)

	f.Fuzz(func(t *testing.T, lastID string, n, capRaw int) {
		n = clampNonNegative(n, 128)
		capacity := clampNonNegative(capRaw, 64)
		if capacity == 0 {
			capacity = 1
		}

		rb := newEventRingBuffer(capacity)

		ids := make([]uint64, 0, n)

		for i := range n {
			ids = append(ids, uint64(i)+1)

			rb.add(sseEvent{ID: strconv.FormatUint(uint64(i)+1, 10), Data: "d"})
		}

		got := rb.eventsAfter(lastID)

		lastSeq, parseErr := strconv.ParseUint(lastID, 10, 64)
		if parseErr != nil {
			if got != nil {
				t.Fatalf("unparseable lastID %q must yield nil, got %d events", lastID, len(got))
			}

			return
		}

		assertRingReplay(t, got, lastSeq, capacity, retainedIDs(ids, capacity))
	})
}

// assertRingReplay checks the ordering, membership, and size invariants of
// a replay result against the ids still retained by the ring.
func assertRingReplay(t *testing.T, got []sseEvent, lastSeq uint64, capacity int, retained []uint64) {
	t.Helper()

	var prev uint64

	for i, evt := range got {
		seq, err := strconv.ParseUint(evt.ID, 10, 64)
		if err != nil {
			t.Fatalf("result contains non-numeric ID %q", evt.ID)
		}

		if seq <= lastSeq {
			t.Fatalf("event %d (id %d) not after lastID %d", i, seq, lastSeq)
		}

		if i > 0 && seq <= prev {
			t.Fatalf("result not ascending: %d after %d", seq, prev)
		}

		prev = seq
	}

	if len(got) > capacity {
		t.Fatalf("got %d events, ring capacity is %d", len(got), capacity)
	}

	want := 0

	for _, id := range retained {
		if id > lastSeq {
			want++
		}
	}

	if len(got) != want {
		t.Fatalf("lastID %d: got %d events, want %d", lastSeq, len(got), want)
	}
}

// retainedIDs returns the suffix of inserted ids that a ring of the given
// capacity still holds (the ring evicts from the front).
func retainedIDs(ids []uint64, capacity int) []uint64 {
	if len(ids) > capacity {
		return ids[len(ids)-capacity:]
	}

	return ids
}

func clampNonNegative(v, ceiling int) int {
	if v < 0 {
		return 0
	}

	if v > ceiling {
		return ceiling
	}

	return v
}
