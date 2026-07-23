package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/larsartmann/go-output/daghtml"
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
	// Use ":8080" for a fixed port.
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
}

// Server serves the real-time DI container dashboard over HTTP.
// Create one with [New] or [NewServer].
type Server struct {
	hub    *Hub
	plugin *auditlog.Plugin
	config Config

	serverMu   sync.Mutex
	httpServer *http.Server
	mux        *http.ServeMux

	dashboardHTML string
	startTime     time.Time
}

// New is the convenience constructor. It creates a Hub, wires it as the
// auditlog OnEvent callback, creates the Plugin, and returns a ready-to-use
// Server. This is the recommended way to use the live package.
//
// For advanced cases (e.g. chaining NDJSON streaming alongside the live
// server), use [NewHub] + [NewServer] and wire callbacks manually:
//
//	hub := live.NewHub(nil)
//	plugin, _ := auditlog.New(auditlog.Config{
//	    Enabled: true,
//	    OnEvent: func(evt auditlog.Event) {
//	        hub.OnEvent(evt)
//	        streamer.OnEvent(evt)
//	    },
//	})
//	server := live.NewServer(hub, plugin, live.Config{Addr: ":8080"})
func New(auditCfg auditlog.Config, serverCfg Config) (*Server, *auditlog.Plugin, error) {
	hub := NewHub(nil)

	auditCfg.OnEvent = hub.OnEvent
	auditCfg.Enabled = true

	plugin, err := auditlog.New(auditCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create plugin: %w", err)
	}

	hub.SetPlugin(plugin)

	server := NewServer(hub, plugin, serverCfg)

	return server, plugin, nil
}

// NewServer creates a Server from an existing Hub and Plugin. Use this when
// you need to chain OnEvent callbacks (e.g. live dashboard + NDJSON streaming).
//
// See [New] for the simpler convenience constructor.
func NewServer(hub *Hub, plugin *auditlog.Plugin, cfg Config) *Server {
	if cfg.Addr == "" {
		cfg.Addr = defaultAddr
	}

	if cfg.ReadHeaderTimeout == 0 {
		cfg.ReadHeaderTimeout = defaultReadHeaderTimeout
	}

	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = defaultHeartbeatInterval
	}

	if cfg.Prefix == "" {
		cfg.Prefix = defaultPrefix
	}

	cfg.Prefix = normalizePrefix(cfg.Prefix)

	s := &Server{
		hub:    hub,
		plugin: plugin,
		config: cfg,
		mux:    http.NewServeMux(),
	}

	s.dashboardHTML = renderDashboardHTML(cfg.Prefix)
	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	pfx := s.config.Prefix
	s.mux.HandleFunc(pfx+"/", s.handleDashboard)
	s.mux.HandleFunc(pfx+"/api/report", s.handleReport)
	s.mux.HandleFunc(pfx+"/api/events", s.handleSSE)
	s.mux.HandleFunc(pfx+"/api/health", s.handleHealth)
}

// ListenAndServe starts the HTTP server. It blocks until Shutdown is called
// or the server encounters a fatal error.
func (s *Server) ListenAndServe() error {
	s.serverMu.Lock()

	if s.httpServer != nil {
		s.serverMu.Unlock()

		return ErrServerAlreadyRunning
	}

	s.startTime = time.Now()

	s.httpServer = &http.Server{ //nolint:exhaustruct // stdlib http.Server with minimal config
		Addr:              s.config.Addr,
		Handler:           s.mux,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
	}

	s.serverMu.Unlock()

	err := s.httpServer.ListenAndServe()

	return fmt.Errorf("listen and serve: %w", err)
}

// Addr returns the server's listen address. If the server was started with
// ":0", this returns the actual address after ListenAndServe binds. Call
// after ListenAndServe (e.g. from a goroutine that checks s.httpServer).
func (s *Server) Addr() string {
	s.serverMu.Lock()
	defer s.serverMu.Unlock()

	if s.httpServer == nil {
		return s.config.Addr
	}

	return s.httpServer.Addr
}

// Shutdown gracefully shuts down the server, waiting for in-flight requests
// to complete (up to the context deadline).
func (s *Server) Shutdown(ctx context.Context) error {
	s.serverMu.Lock()
	server := s.httpServer
	s.serverMu.Unlock()

	if server == nil {
		return nil
	}

	err := server.Shutdown(ctx)

	return fmt.Errorf("shutdown: %w", err)
}

// SignalComplete marks the container lifecycle as finished. All connected SSE
// clients receive the final report.
func (s *Server) SignalComplete() {
	s.hub.SignalComplete()
}

// OnEvent broadcasts an event to all connected SSE clients. This method is
// safe to pass directly as auditlog.Config.OnEvent.
func (s *Server) OnEvent(evt auditlog.Event) {
	s.hub.OnEvent(evt)
}

// ClientCount returns the number of currently connected SSE clients.
func (s *Server) ClientCount() int {
	return s.hub.ClientCount()
}

// ServeHTTP implements http.Handler, delegating to the internal mux.
// This allows the Server to be used with httptest.NewServer.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// --- HTTP Handlers ---

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	pfx := s.config.Prefix
	if r.URL.Path != pfx && r.URL.Path != pfx+"/" {
		http.NotFound(w, r)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	_, _ = w.Write([]byte(s.dashboardHTML))
}

func (s *Server) handleReport(w http.ResponseWriter, _ *http.Request) {
	report := s.plugin.Report()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	encoder := json.NewEncoder(w)

	err := encoder.Encode(report)
	if err != nil {
		http.Error(w, "encode report", http.StatusInternalServerError)

		return
	}
}

// healthResponse is the JSON payload returned by the health endpoint.
type healthResponse struct {
	Status   string  `json:"status"`
	UptimeS  float64 `json:"uptime_s"`
	Clients  int     `json:"clients"`
	Events   int     `json:"events"`
	Complete bool    `json:"complete"`
	Dropped  int64   `json:"dropped"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := healthResponse{
		Status:   "ok",
		UptimeS:  time.Since(s.startTime).Seconds(),
		Clients:  s.hub.ClientCount(),
		Events:   s.plugin.EventsCount(),
		Complete: s.hub.IsComplete(),
		Dropped:  s.plugin.DroppedEventCount(),
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// snapshotData is the payload sent as the initial SSE event.
type snapshotData struct {
	Report   auditlog.Report       `json:"report"`
	Events   []auditlog.Event      `json:"events"`
	Metadata auditlog.TypeMetadata `json:"metadata"`
	DAG      daghtml.DAG           `json:"dag"`
	Complete bool                  `json:"complete"`
}

// completeData is the payload sent when the container lifecycle finishes.
type completeData struct {
	Report auditlog.Report `json:"report"`
	DAG    daghtml.DAG     `json:"dag"`
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub := s.hub.Subscribe()
	defer s.hub.Unsubscribe(sub.id)

	err := s.sendSnapshot(w, flusher)
	if err != nil {
		return
	}

	heartbeat := time.NewTicker(s.config.HeartbeatInterval)
	defer heartbeat.Stop()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return

		case <-sub.done:
			s.sendComplete(w, flusher)

			return

		case evt := <-sub.ch:
			err = writeSSE(w, "event", evt)
			if err != nil {
				return
			}

			flusher.Flush()

		case <-heartbeat.C:
			_, err = w.Write([]byte(": heartbeat\n\n"))
			if err != nil {
				return
			}

			flusher.Flush()
		}
	}
}

func (s *Server) sendSnapshot(w http.ResponseWriter, flusher http.Flusher) error {
	report := s.plugin.Report()
	events := s.plugin.Events()

	data := snapshotData{
		Report:   report,
		Events:   events,
		Metadata: auditlog.BuildTypeMetadata(),
		DAG:      auditlog.BuildDAGHTML(report),
		Complete: s.hub.IsComplete(),
	}

	err := writeSSE(w, "snapshot", data)
	if err != nil {
		return err
	}

	flusher.Flush()

	return nil
}

func (s *Server) sendComplete(w http.ResponseWriter, flusher http.Flusher) {
	report := s.plugin.Report()

	data := completeData{
		Report: report,
		DAG:    auditlog.BuildDAGHTML(report),
	}

	_ = writeSSE(w, "complete", data)

	flusher.Flush()
}

// writeSSE writes a named SSE event with JSON-encoded data.
func writeSSE(w http.ResponseWriter, eventName string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal SSE data: %w", err)
	}

	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, payload)
	if err != nil {
		return fmt.Errorf("write SSE: %w", err)
	}

	return nil
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
