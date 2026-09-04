// Package-level SSE transport primitives implemented with the Go standard
// library only. This file replaces the go-sse dependency with a minimal,
// byte-compatible subset: the wire format, a per-connection stream writer,
// and Last-Event-ID extraction.
//
// Wire format (https://html.spec.whatwg.org/multipage/server-sent-events.html):
//
//	event: <name>\n
//	data: <line>\n        (repeated for every line of Data)
//	id: <id>\n            (omitted when ID is empty)
//	retry: <ms>\n         (omitted when Retry is 0)
//	\n
package live

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// sseContentType is the HTTP content type for Server-Sent Events.
	sseContentType = "text/event-stream"

	// sseHeartbeatFrame is the SSE comment frame used for keep-alive pings.
	// Browsers ignore comment lines, but they reset idle timers on reverse
	// proxies.
	sseHeartbeatFrame = ": heartbeat\n\n"

	// base10 is the numeric base for decimal integer formatting.
	base10 = 10

	// initialEventCap is the starting capacity of the wire-format buffer;
	// most events fit without a re-allocation.
	initialEventCap = 64
)

// sseEvent is a single Server-Sent Event in wire-ready form.
type sseEvent struct {
	// Name maps to the event: field. Empty means the default "message" event.
	Name string
	// Data maps to the data: field. Multi-line data is split so each line
	// gets its own "data: " prefix (required by the spec).
	Data string
	// ID maps to the id: field. Empty omits the field entirely. Browsers
	// send it back via Last-Event-ID on reconnect.
	ID string
	// Retry maps to the retry: field in milliseconds. Zero omits the field.
	Retry uint
}

// sseSplitLines splits the input into lines for SSE data field formatting. Per the
// SSE spec, CR, LF, and CRLF are all valid line endings and are normalized
// to LF here. An empty input yields a single empty line.
func sseSplitLines(input string) []string {
	if input == "" {
		return []string{""}
	}

	if !strings.ContainsAny(input, "\n\r") {
		return []string{input}
	}

	lines := make([]string, 0, strings.Count(input, "\n")+1)

	start := 0

	for i := 0; i < len(input); {
		switch input[i] {
		case '\r':
			lines = append(lines, input[start:i])
			i++

			if i < len(input) && input[i] == '\n' {
				i++
			}

			start = i
		case '\n':
			lines = append(lines, input[start:i])
			i++
			start = i
		default:
			i++
		}
	}

	if start < len(input) {
		lines = append(lines, input[start:])
	}

	return lines
}

// sseJoinLines joins lines with "\n", producing the Data payload for a
// multi-line sseEvent. This is the inverse of sseSplitLines at the data
// level: writeSSEEvent splits the result back into individual "data:" lines.
func sseJoinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}

// sseStripNewlines removes CR and LF from a single-line SSE field value
// (event name, id). Newlines inside these fields would inject additional
// fields into the wire frame; multi-line payloads belong in Data.
func sseStripNewlines(value string) string {
	if !strings.ContainsAny(value, "\n\r") {
		return value
	}

	replacer := strings.NewReplacer("\r", "", "\n", "")

	return replacer.Replace(value)
}

// sseKeyedLines prefixes every line of value with "key ", producing the
// newline-joined string that writeSSEEvent splits into individual "data:"
// lines. This is the building block for the datastar wire format, whose
// multi-line values repeat the key on every line:
//
//	sseKeyedLines("elements", "<div>\n</div>")
//	// → "elements <div>\nelements </div>"
//
// Returns "" when value is empty (no data line emitted).
func sseKeyedLines(key, value string) string {
	if value == "" {
		return ""
	}

	lines := sseSplitLines(value)

	var b strings.Builder

	b.Grow(len(value) + len(lines)*(len(key)+1))

	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}

		b.WriteString(key)
		b.WriteByte(' ')
		b.WriteString(line)
	}

	return b.String()
}

// writeSSEEvent writes a single SSE event to w in the standard wire format.
// Name and ID have CR/LF stripped: a newline inside a single-line field
// would let arbitrary data inject additional SSE fields into the frame
// (field-injection). Multi-line payloads belong in Data, which is split
// into per-line "data:" fields. The caller is responsible for flushing
// (sseStream.send does this).
func writeSSEEvent(w io.Writer, evt sseEvent) error {
	buf := make([]byte, 0, initialEventCap)

	if name := sseStripNewlines(evt.Name); name != "" {
		buf = append(buf, "event: "...)
		buf = append(buf, name...)
		buf = append(buf, '\n')
	}

	for _, line := range sseSplitLines(evt.Data) {
		buf = append(buf, "data: "...)
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	if id := sseStripNewlines(evt.ID); id != "" {
		buf = append(buf, "id: "...)
		buf = append(buf, id...)
		buf = append(buf, '\n')
	}

	if evt.Retry > 0 {
		buf = append(buf, "retry: "...)
		buf = strconv.AppendUint(buf, uint64(evt.Retry), base10)
		buf = append(buf, '\n')
	}

	buf = append(buf, '\n')

	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("write sse event %q: %w", evt.Name, err)
	}

	return nil
}

// sseSetHeaders sets the response headers required for Server-Sent Events:
// text/event-stream content type, no caching, and keep-alive connection.
// Call before writing the status code.
func sseSetHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", sseContentType)
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
}

// sseStream manages a single Server-Sent Events connection. It owns the
// response writer, serializes all writes behind a mutex (the event loop and
// the heartbeat goroutine both write), and flushes after every event.
type sseStream struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
}

// newSSEStream creates an SSE stream from an HTTP response writer. It sets
// the required SSE headers and writes the 200 OK status code. The caller
// must have verified that w implements http.Flusher.
func newSSEStream(w http.ResponseWriter) *sseStream {
	sseSetHeaders(w)
	w.WriteHeader(http.StatusOK)

	fw, _ := w.(http.Flusher)

	return &sseStream{
		w:       w,
		flusher: fw,
		mu:      sync.Mutex{},
	}
}

// send writes an SSE event to the stream and flushes the response. Returns
// an error if the write fails (e.g., client disconnected).
func (s *sseStream) send(evt sseEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := writeSSEEvent(s.w, evt); err != nil {
		return err
	}

	if s.flusher != nil {
		s.flusher.Flush()
	}

	return nil
}

// sendLines sends an SSE event with multiple data lines; each argument
// becomes a separate "data:" line. Lines containing embedded newlines are
// split further by writeSSEEvent, so sseKeyedLines results compose cleanly.
func (s *sseStream) sendLines(eventName string, lines ...string) error {
	return s.send(sseEvent{Name: eventName, Data: sseJoinLines(lines...)})
}

// sendKeyed sends a single-key SSE event, prefixing every line of value
// with "key " (the datastar patch-signals pattern).
func (s *sseStream) sendKeyed(eventName, key, value string) error {
	return s.send(sseEvent{Name: eventName, Data: sseKeyedLines(key, value)})
}

// heartbeat sends SSE comment-frame pings at the given interval until ctx
// is cancelled. Run it in a goroutine alongside the event loop; it keeps
// proxies (Nginx, Cloudflare, AWS ALB) from killing idle SSE connections.
func (s *sseStream) heartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()

			_, err := io.WriteString(s.w, sseHeartbeatFrame)
			if err == nil && s.flusher != nil {
				s.flusher.Flush()
			}

			s.mu.Unlock()

			if err != nil {
				return
			}
		}
	}
}

// lastEventIDFromRequest extracts the Last-Event-ID header from an HTTP
// request. Values containing NUL or line terminators (which would corrupt
// the SSE wire format) are treated as if no Last-Event-ID was sent and the
// empty string is returned.
func lastEventIDFromRequest(r *http.Request) string {
	id := r.Header.Get("Last-Event-ID")
	if strings.ContainsAny(id, "\n\r\x00") {
		return ""
	}

	return id
}
