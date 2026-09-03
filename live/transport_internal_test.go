package live

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSSEKeyedLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "single line", key: "elements", value: "<div></div>", want: "elements <div></div>"},
		{name: "multi line", key: "elements", value: "<div>\n</div>", want: "elements <div>\nelements </div>"},
		{name: "empty value", key: "elements", value: "", want: ""},
		{
			name:  "crlf normalized",
			key:   "signals",
			value: "{\"a\":1}\r\n{\"b\":2}",
			want:  "signals {\"a\":1}\nsignals {\"b\":2}",
		},
		{name: "trailing newline dropped", key: "k", value: "a\n", want: "k a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := sseKeyedLines(tt.key, tt.value)
			if got != tt.want {
				t.Errorf("sseKeyedLines(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.want)
			}
		})
	}
}

func TestWriteSSEEventWireFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		evt  sseEvent
		want string
	}{
		{
			name: "named event with data",
			evt:  sseEvent{Name: "datastar-patch-signals", Data: "signals {}"},
			want: "event: datastar-patch-signals\ndata: signals {}\n\n",
		},
		{
			name: "multi-line data split",
			evt:  sseEvent{Name: "evt", Data: "selector #a\nmode inner\nelements <div>"},
			want: "event: evt\ndata: selector #a\ndata: mode inner\ndata: elements <div>\n\n",
		},
		{
			name: "with id and retry",
			evt:  sseEvent{Name: "evt", Data: "x", ID: "42", Retry: 3000},
			want: "event: evt\ndata: x\nid: 42\nretry: 3000\n\n",
		},
		{
			name: "id only",
			evt:  sseEvent{Name: "event", Data: "{}", ID: "7"},
			want: "event: event\ndata: {}\nid: 7\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			if err := writeSSEEvent(&buf, tt.evt); err != nil {
				t.Fatalf("writeSSEEvent: %v", err)
			}

			if buf.String() != tt.want {
				t.Errorf("wire format mismatch:\ngot:  %q\nwant: %q", buf.String(), tt.want)
			}
		})
	}
}

func TestWriteSSEEventWrapsWriteError(t *testing.T) {
	t.Parallel()

	sentinel := errBrokenPipe

	err := writeSSEEvent(failingWriter{err: sentinel}, sseEvent{Name: "evt", Data: "x"})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}

	if !errors.Is(err, sentinel) {
		t.Errorf("error should wrap sentinel, got: %v", err)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestSSEStreamSendAndFlush(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	stream := newSSEStream(rec)

	if ct := rec.Header().Get("Content-Type"); ct != sseContentType {
		t.Errorf("Content-Type = %q, want %q", ct, sseContentType)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if err := stream.sendKeyed("datastar-patch-signals", "signals", `{"complete":true}`); err != nil {
		t.Fatalf("sendKeyed: %v", err)
	}

	want := "event: datastar-patch-signals\ndata: signals {\"complete\":true}\n\n"
	if rec.Body.String() != want {
		t.Errorf("body = %q, want %q", rec.Body.String(), want)
	}
}

func TestSSEStreamConcurrentWritesAreSerialized(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	stream := newSSEStream(rec)

	var wg sync.WaitGroup

	for i := range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = stream.send(sseEvent{Name: "evt", Data: strconv.Itoa(i)})
		}()
	}

	wg.Wait()

	body := rec.Body.String()
	if got := strings.Count(body, "event: evt\n"); got != 8 {
		t.Errorf("expected 8 complete events, got %d in %q", got, body)
	}
}

func TestLastEventIDFromRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "present", header: "42", want: "42"},
		{name: "missing", header: "", want: ""},
		{name: "newline rejected", header: "42\nevent: injected", want: ""},
		{name: "nul rejected", header: "4\x002", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
			if tt.header != "" && !strings.ContainsAny(tt.header, "\n\r\x00") {
				req.Header.Set("Last-Event-ID", tt.header)
			}

			if got := lastEventIDFromRequest(req); got != tt.want {
				t.Errorf("lastEventIDFromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventRingBufferFIFOEviction(t *testing.T) {
	t.Parallel()

	rb := newEventRingBuffer(3)

	for i := range 5 {
		rb.add(sseEvent{ID: strconv.Itoa(i), Data: "d"})
	}

	if got := rb.len(); got != 3 {
		t.Fatalf("len = %d, want 3", got)
	}

	events := rb.eventsAfter("1")
	if len(events) != 3 || events[0].ID != "2" || events[1].ID != "3" || events[2].ID != "4" {
		t.Errorf("eventsAfter(1) = %+v, want IDs [2 3 4]", events)
	}
}

func TestEventRingBufferEventsAfterEdgeCases(t *testing.T) {
	t.Parallel()

	rb := newEventRingBuffer(0)

	for i := 1; i <= 4; i++ {
		rb.add(sseEvent{ID: strconv.Itoa(i)})
	}

	if got := rb.eventsAfter("4"); got != nil {
		t.Errorf("eventsAfter(latest) = %+v, want nil", got)
	}

	if got := rb.eventsAfter("not-a-number"); got != nil {
		t.Errorf("eventsAfter(garbage) = %+v, want nil", got)
	}

	if got := rb.eventsAfter("0"); len(got) != 4 {
		t.Errorf("eventsAfter(0) = %d events, want 4", len(got))
	}
}

func TestBroadcasterFanOutAndUnsubscribe(t *testing.T) {
	t.Parallel()

	bc := newBroadcaster(4)

	ch1 := bc.subscribe()
	ch2 := bc.subscribe()

	if got := bc.subscriberCount(); got != 2 {
		t.Fatalf("subscriberCount = %d, want 2", got)
	}

	bc.broadcast(sseEvent{ID: "1", Data: "a"})
	bc.unsubscribe(ch1)
	bc.broadcast(sseEvent{ID: "2", Data: "b"})

	if got := bc.subscriberCount(); got != 1 {
		t.Fatalf("subscriberCount after unsubscribe = %d, want 1", got)
	}

	evt, ok := <-ch2
	if !ok || evt.ID != "1" {
		t.Fatalf("first event on ch2 = %+v (ok=%v), want ID 1", evt, ok)
	}

	evt, ok = <-ch2
	if !ok || evt.ID != "2" {
		t.Fatalf("second event on ch2 = %+v (ok=%v), want ID 2", evt, ok)
	}

	if _, ok := <-ch1; ok {
		// The buffered broadcast (ID 1) is still readable after close; the
		// receive after it must report a closed channel.
		if _, ok := <-ch1; ok {
			t.Error("unsubscribed channel should be closed after buffered events are drained")
		}
	}
}

func TestBroadcasterDropsWhenSubscriberSlow(t *testing.T) {
	t.Parallel()

	bc := newBroadcaster(2)
	ch := bc.subscribe()

	for i := range 10 {
		bc.broadcast(sseEvent{ID: strconv.Itoa(i)})
	}

	bc.unsubscribe(ch)

	drained := 0

	for range ch {
		drained++
	}

	if drained != 2 {
		t.Errorf("drained %d events from buffer of 2, want 2", drained)
	}
}

func TestBroadcasterShutdownDrainsThenCloses(t *testing.T) {
	t.Parallel()

	bc := newBroadcaster(8)
	ch := bc.subscribe()

	bc.broadcast(sseEvent{ID: "1"})
	bc.broadcast(sseEvent{ID: "2"})

	read := make([]string, 0, 2)
	done := make(chan struct{})

	go func() {
		defer close(done)

		for evt := range ch {
			read = append(read, evt.ID)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := bc.shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	<-done

	if len(read) != 2 || read[0] != "1" || read[1] != "2" {
		t.Errorf("consumer read %v, want [1 2]", read)
	}

	if got := bc.subscriberCount(); got != 0 {
		t.Errorf("subscriberCount after shutdown = %d, want 0", got)
	}

	if err := bc.shutdown(ctx); err != nil {
		t.Errorf("second shutdown should be a no-op, got %v", err)
	}
}

func TestBroadcasterShutdownHonoursContext(t *testing.T) {
	t.Parallel()

	bc := newBroadcaster(2)

	ch := bc.subscribe()
	defer func() { bc.unsubscribe(ch) }()

	bc.broadcast(sseEvent{ID: "1"})
	bc.broadcast(sseEvent{ID: "2"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	if err := bc.shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown with full buffer + expired ctx = %v, want deadline exceeded", err)
	}

	if h := bc.health(); !h.Draining {
		t.Error("broadcaster should stay in draining state after failed shutdown")
	}
}

func TestBroadcasterHealth(t *testing.T) {
	t.Parallel()

	bc := newBroadcaster(64)
	_ = bc.subscribe()

	h := bc.health()

	if h.BufferSize != 64 || h.SubscriberCount != 1 || h.Draining {
		t.Errorf("health = %+v, want BufferSize 64, SubscriberCount 1, Draining false", h)
	}
}

func TestNewSSEStreamHeartbeat(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	stream := newSSEStream(rec)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)

		stream.heartbeat(ctx, 5*time.Millisecond)
	}()

	time.Sleep(40 * time.Millisecond)
	cancel()
	<-done

	if got := strings.Count(rec.Body.String(), sseHeartbeatFrame); got < 2 {
		t.Errorf("expected at least 2 heartbeat frames in 40ms, got %d", got)
	}
}

var errBrokenPipe = errors.New("broken pipe")
