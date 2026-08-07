package live

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/larsartmann/go-sse"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultHeartbeatInterval = 15 * time.Second
	defaultAddr              = ":0"
	defaultPrefix            = "/debug/di"
)

// ErrServerAlreadyRunning is returned when ListenAndServe is called on a
// server that is already serving.
var ErrServerAlreadyRunning = errors.New("live server is already running")

// Config controls the live dashboard server behaviour.
type Config struct {
	// Addr is the TCP address to listen on. Default ":0" (random port).
	Addr string
	// Prefix is the URL path prefix for all dashboard routes.
	// Default "/debug/di". Routes: {prefix}/, {prefix}/api/report,
	// {prefix}/api/events, {prefix}/api/health.
	// Set to "/" to mount at root. Trailing slash is stripped.
	Prefix string
	// ReadHeaderTimeout is the maximum duration for reading the request
	// headers. Default 5 seconds. Set to 0 to disable.
	ReadHeaderTimeout time.Duration
	// HeartbeatInterval is how often to send SSE keepalive comments.
	// Default 15 seconds. Set 0 to disable heartbeats.
	HeartbeatInterval time.Duration
	// CORSAllowedOrigins controls the Access-Control-Allow-Origin header
	// on all API endpoints. Default "*" (allow all origins). Set to a
	// specific origin (e.g. "https://dashboard.example.com") to restrict.
	// Set to "" to disable CORS headers entirely.
	CORSAllowedOrigins string
}

// HealthInfo provides dynamic health check data.
type HealthInfo struct {
	Events  int   `json:"events"`
	Dropped int64 `json:"dropped"`
}

// snapshotSignals is the initial signal payload sent on SSE connect.
// These are server-owned signals; client-owned signals (activeTab,
// serviceSearch, etc.) are declared in the HTML template's data-signals.
// Tags use camelCase because datastar's signal system requires camelCase keys.
//
//nolint:tagliatelle // camelCase required by datastar signal system
type snapshotSignals struct {
	ConnStatus       string `json:"connStatus"`
	Complete         bool   `json:"complete"`
	ServicesOverflow bool   `json:"servicesOverflow"`
	EventsOverflow   bool   `json:"eventsOverflow"`
}

// Server serves the real-time DI container dashboard over HTTP.
type Server struct {
	hub    *Hub
	plugin *auditlog.Plugin
	config Config

	serverMu   sync.Mutex
	httpServer *http.Server
	mux        *http.ServeMux

	prefix string

	dashboardHTML string
	startTime     time.Time
}

// New is the convenience constructor. It creates a Hub, wires it as the
// auditlog OnEvent callback, creates the Plugin, and returns a ready-to-use
// Server.
func New(auditCfg auditlog.Config, serverCfg Config) (*Server, *auditlog.Plugin, error) {
	hub := NewHub()

	auditCfg.OnEvent = hub.OnEvent
	auditCfg.Enabled = true

	plugin, err := auditlog.New(auditCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create plugin: %w", err)
	}

	server := NewServer(hub, plugin, serverCfg)

	return server, plugin, nil
}

// NewServer creates a Server from an existing Hub and Plugin.
func NewServer(hub *Hub, plugin *auditlog.Plugin, cfg Config) *Server {
	if cfg.Addr == "" {
		cfg.Addr = defaultAddr
	}

	if cfg.Prefix == "" {
		cfg.Prefix = defaultPrefix
	}

	cfg.Prefix = normalizePrefix(cfg.Prefix)

	if cfg.ReadHeaderTimeout == 0 {
		cfg.ReadHeaderTimeout = defaultReadHeaderTimeout
	}

	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = defaultHeartbeatInterval
	}

	if cfg.CORSAllowedOrigins == "" {
		cfg.CORSAllowedOrigins = "*"
	}

	srv := &Server{
		hub:    hub,
		plugin: plugin,
		config: cfg,
		mux:    http.NewServeMux(),
		prefix: cfg.Prefix,
	}

	srv.dashboardHTML = renderDashboardHTML(cfg.Prefix)

	srv.setupRoutes()

	return srv
}

func (srv *Server) setupRoutes() {
	pfx := srv.config.Prefix
	if pfx == "/" {
		srv.mux.HandleFunc("/", srv.handleDashboard)
		srv.mux.HandleFunc("/api/report", srv.corsMiddleware(srv.handleReport))
		srv.mux.HandleFunc("/api/events", srv.corsMiddleware(srv.handleSSE))
		srv.mux.HandleFunc("/api/health", srv.corsMiddleware(srv.handleHealth))
		srv.mux.HandleFunc("/api/export/ndjson", srv.corsMiddleware(srv.handleExportNDJSON))
		srv.mux.HandleFunc("/api/export/html", srv.corsMiddleware(srv.handleExportHTML))
	} else {
		srv.mux.HandleFunc(pfx+"/", srv.handleDashboard)
		srv.mux.HandleFunc(pfx+"/api/report", srv.corsMiddleware(srv.handleReport))
		srv.mux.HandleFunc(pfx+"/api/events", srv.corsMiddleware(srv.handleSSE))
		srv.mux.HandleFunc(pfx+"/api/health", srv.corsMiddleware(srv.handleHealth))
		srv.mux.HandleFunc(pfx+"/api/export/ndjson", srv.corsMiddleware(srv.handleExportNDJSON))
		srv.mux.HandleFunc(pfx+"/api/export/html", srv.corsMiddleware(srv.handleExportHTML))
	}
}

// corsMiddleware adds CORS headers and handles OPTIONS preflight for API endpoints.
func (srv *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	origin := srv.config.CORSAllowedOrigins

	return func(w http.ResponseWriter, r *http.Request) {
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next(w, r)
	}
}

// ListenAndServe starts the HTTP server.
func (srv *Server) ListenAndServe() error {
	srv.serverMu.Lock()

	if srv.httpServer != nil {
		srv.serverMu.Unlock()

		return ErrServerAlreadyRunning
	}

	srv.startTime = time.Now()

	srv.httpServer = &http.Server{ //nolint:exhaustruct // minimal config
		Addr:              srv.config.Addr,
		Handler:           srv.mux,
		ReadHeaderTimeout: srv.config.ReadHeaderTimeout,
	}

	srv.serverMu.Unlock()

	return fmt.Errorf("listen and serve: %w", srv.httpServer.ListenAndServe())
}

// Addr returns the server's listen address.
func (srv *Server) Addr() string {
	srv.serverMu.Lock()
	defer srv.serverMu.Unlock()

	if srv.httpServer == nil {
		return srv.config.Addr
	}

	return srv.httpServer.Addr
}

// Shutdown gracefully shuts down the server and drains the broadcaster.
func (srv *Server) Shutdown(ctx context.Context) error {
	srv.serverMu.Lock()
	server := srv.httpServer
	srv.serverMu.Unlock()

	if server == nil {
		return nil
	}

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := srv.hub.Shutdown(ctx); err != nil {
		return fmt.Errorf("drain broadcaster: %w", err)
	}

	return nil
}

// SignalComplete marks the container lifecycle as finished.
func (srv *Server) SignalComplete() {
	srv.hub.SignalComplete()
}

// OnEvent broadcasts an event to all connected SSE clients.
func (srv *Server) OnEvent(evt auditlog.Event) {
	srv.hub.OnEvent(evt)
}

// ClientCount returns the number of currently connected SSE clients.
func (srv *Server) ClientCount() int {
	return srv.hub.ClientCount()
}

// ServeHTTP implements http.Handler.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.mux.ServeHTTP(w, r)
}

// --- HTTP Handlers ---

func (srv *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	pfx := srv.config.Prefix
	if r.URL.Path != pfx && r.URL.Path != pfx+"/" {
		http.NotFound(w, r)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
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

	// Send initial snapshot as datastar patch-elements + patch-signals.
	// On reconnect this IS the replay — the full current state replaces
	// any missed events.
	if err := srv.sendDatastarSnapshot(stream); err != nil { //nolint:contextcheck // stream derives ctx from request
		return
	}

	eventCh := srv.hub.Subscribe()
	defer srv.hub.Unsubscribe(eventCh)

	go stream.Heartbeat(r.Context(), srv.config.HeartbeatInterval)

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return

		case <-srv.hub.Done():
			srv.sendDatastarComplete(stream) //nolint:contextcheck // stream derives ctx

			return

		case <-eventCh:
			// Non-blocking drain: coalesce burst events into a single render.
			drainEvents(eventCh)

			if err := srv.sendDatastarSnapshot(stream); err != nil { //nolint:contextcheck // ctx from stream
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
func (srv *Server) sendDatastarSnapshot(stream *sse.Stream) error {
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
	for _, frag := range renderAllFragments(stream.Context(), report, events, meta) {
		if err := sendPatchElements(stream, frag.selector, frag.html); err != nil {
			return fmt.Errorf("send fragment %s: %w", frag.selector, err)
		}
	}

	return nil
}

// sendDatastarComplete sends the final full render and marks the lifecycle
// as complete via a signal patch.
func (srv *Server) sendDatastarComplete(stream *sse.Stream) {
	if srv.plugin == nil {
		return
	}

	// Send final full render.
	_ = srv.sendDatastarSnapshot(stream)

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

	encoder := json.NewEncoder(&buf)

	if err := encoder.Encode(report); err != nil {
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
