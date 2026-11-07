package live

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/go-sse"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// --- HTTP Handlers ---

// dashboardFramePolicy is delivered as an HTTP response header because the
// CSP frame-ancestors directive is ignored by browsers when set via a
// <meta> element — the meta tag in liveTemplate intentionally omits it.
const dashboardFramePolicy = "frame-ancestors 'none'"

func (srv *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	pfx := srv.config.Prefix
	if r.URL.Path != pfx && r.URL.Path != pfx+"/" {
		http.NotFound(w, r)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Security-Policy", dashboardFramePolicy)
	w.Header().Set("X-Frame-Options", "DENY")
	_, _ = w.Write([]byte(srv.dashboardHTML))
}

func (srv *Server) handleReport(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	if !srv.requirePlugin(w) {
		return
	}

	data, err := makeReportJSON(srv.plugin)
	if err != nil {
		http.Error(w, fmt.Sprintf("generate report: %v", err), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(data)
}

// requirePlugin writes a 503 if no plugin is available and returns false.
// Returns true when the plugin is present and the handler may continue.
func (srv *Server) requirePlugin(w http.ResponseWriter) bool {
	if srv.plugin == nil {
		http.Error(w, "no plugin available", http.StatusServiceUnavailable)

		return false
	}

	return true
}

// setDownloadHeaders configures response headers for an attachment download
// of the given content type and filename.
func setDownloadHeaders(w http.ResponseWriter, contentType, filename string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
}

func (srv *Server) handleExportNDJSON(w http.ResponseWriter, _ *http.Request) {
	if !srv.requirePlugin(w) {
		return
	}

	setDownloadHeaders(w, "application/x-ndjson", "auditlog-events.ndjson")

	if err := srv.plugin.WriteEventsNDJSON(w); err != nil {
		http.Error(w, fmt.Sprintf("export ndjson: %v", err), http.StatusInternalServerError)

		return
	}
}

func (srv *Server) handleExportHTML(w http.ResponseWriter, _ *http.Request) {
	if !srv.requirePlugin(w) {
		return
	}

	setDownloadHeaders(w, "text/html; charset=utf-8", "auditlog-report.html")

	if err := srv.plugin.WriteHTML(w); err != nil { //nolint:contextcheck // WriteHTML takes io.Writer, not context
		http.Error(w, fmt.Sprintf("export html: %v", err), http.StatusInternalServerError)

		return
	}
}

type healthResponse struct {
	Status   string  `json:"status"`
	UptimeS  float64 `json:"uptime_s"`
	Clients  int     `json:"clients"`
	Events   int     `json:"events"`
	Complete bool    `json:"complete"`
	Dropped  int64   `json:"dropped"`
	Draining bool    `json:"draining"`
	BufferSz int     `json:"buffer_size"`
}

func (srv *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	bcHealth := srv.hub.Health()

	resp := healthResponse{
		Status:   "ok",
		UptimeS:  time.Since(srv.startTime).Seconds(),
		Clients:  srv.hub.ClientCount(),
		Complete: srv.hub.IsComplete(),
		Draining: bcHealth.Draining,
		BufferSz: bcHealth.BufferSize,
	}

	plugin := srv.plugin
	if plugin != nil {
		resp.Events = plugin.EventsCount()
		resp.Dropped = plugin.DroppedEventCount()
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "marshal health response", http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(payload)
}

func (srv *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	_, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)

		return
	}

	// X-Accel-Buffering tells Nginx not to buffer this response.
	// Must be set before NewStream, which calls WriteHeader internally.
	w.Header().Set("X-Accel-Buffering", "no")

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	ctx := r.Context()

	// Subscribe BEFORE replay/snapshot. If the subscription were registered
	// after the snapshot was rendered, an event emitted in between would be
	// in neither the snapshot nor the live stream — silently lost for this
	// client (and the live-delivery test can hang waiting for it). With the
	// subscription first, the replay covers everything up to now, the
	// snapshot covers current state, and anything after Subscribe arrives
	// via the live stream — no gap, no duplication.
	eventCh := srv.hub.Subscribe()
	defer srv.hub.Unsubscribe(eventCh)

	// Reconnection replay: if the client sends Last-Event-ID, replay any
	// missed events from the ring buffer before sending the snapshot.
	if lastID := stream.LastEventID(); !lastID.IsZero() {
		if _, err := sse.Replay(stream, srv.hub.EventStore(), lastID); err != nil {
			return
		}
	}

	// Send initial snapshot as datastar patch-elements + patch-signals.
	// On reconnect this IS the replay — the full current state replaces
	// any missed events.
	if err := srv.sendDatastarSnapshot(ctx, stream); err != nil {
		return
	}

	go stream.Heartbeat(r.Context(), srv.config.HeartbeatInterval)

	for {
		select {
		case <-ctx.Done():
			return

		case <-srv.hub.Done():
			srv.sendDatastarComplete(ctx, stream)

			return

		case <-eventCh:
			// Non-blocking drain: coalesce burst events into a single render.
			drainEvents(eventCh)

			if err := srv.sendDatastarSnapshot(ctx, stream); err != nil {
				return
			}
		}
	}
}

// drainEvents non-blocking-drains any immediately-available events from the
// channel. This coalesces event bursts (e.g. rapid registrations) into a
// single HTML re-render instead of one render per event.
func drainEvents(ch <-chan sse.Event) {
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

// sendDatastarSnapshot renders all dashboard sections from the plugin's
// current state and sends them as datastar-patch-elements events. It also
// sends a datastar-patch-signals event with connection status.
func (srv *Server) sendDatastarSnapshot(ctx context.Context, stream *sse.Stream) error {
	plugin := srv.plugin
	if plugin == nil {
		return nil
	}

	report := plugin.Report()
	events := plugin.Events()
	meta := auditlog.BuildTypeMetadata()

	// Send connection status + overflow signals.
	signals := snapshotSignals{
		ConnStatus:       "connected",
		Complete:         srv.hub.IsComplete(),
		ServicesOverflow: len(report.Services) > maxServiceRows,
		EventsOverflow:   len(events) > maxEventRows,
	}

	signalsJSON, err := json.Marshal(signals)
	if err != nil {
		return fmt.Errorf("marshal signals: %w", err)
	}

	if err := stream.SendKeyed("datastar-patch-signals", "signals", string(signalsJSON)); err != nil {
		return fmt.Errorf("send signals: %w", err)
	}

	// Send all HTML fragments.
	for _, frag := range renderAllFragments(ctx, report, events, meta) {
		if err := sendPatchElements(stream, frag.selector, frag.html); err != nil {
			return fmt.Errorf("send fragment %s: %w", frag.selector, err)
		}
	}

	return nil
}

// sendDatastarComplete sends the final full render and marks the lifecycle
// as complete via a signal patch.
func (srv *Server) sendDatastarComplete(ctx context.Context, stream *sse.Stream) {
	if srv.plugin == nil {
		return
	}

	// Send final full render.
	_ = srv.sendDatastarSnapshot(ctx, stream)

	// Signal completion.
	_ = stream.SendKeyed("datastar-patch-signals", "signals", `{"complete":true,"connStatus":"complete"}`)
}

// sendPatchElements sends a datastar-patch-elements SSE event that morphs
// the inner HTML of the element matching the given CSS selector.
func sendPatchElements(stream *sse.Stream, selector, htmlContent string) error {
	return stream.SendLines("datastar-patch-elements",
		"selector "+selector,
		"mode inner",
		sse.KeyedLines("elements", htmlContent),
	)
}

// --- Helpers ---

func makeReportJSON(plugin *auditlog.Plugin) ([]byte, error) {
	report := plugin.Report()

	var buf bytes.Buffer

	encoder := jsontext.NewEncoder(&buf)

	if err := json.MarshalEncode(encoder, report); err != nil {
		return nil, fmt.Errorf("encode report: %w", err)
	}

	return buf.Bytes(), nil
}

// normalizePrefix ensures the prefix starts with "/" and has no trailing "/".
func normalizePrefix(prefix string) string {
	if prefix == "/" {
		return "/"
	}

	prefix = strings.TrimRight(prefix, "/")

	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	return prefix
}
