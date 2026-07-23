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

	"github.com/larsartmann/go-output/daghtml"
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
}

// HealthInfo provides dynamic health check data.
type HealthInfo struct {
	Events  int   `json:"events"`
	Dropped int64 `json:"dropped"`
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

	srv := &Server{ //nolint:exhaustruct
		hub:     hub,
		plugin:  plugin,
		config:  cfg,
		mux:     http.NewServeMux(),
		prefix:  cfg.Prefix,
	}

	srv.dashboardHTML = renderDashboardHTML(cfg.Prefix)

	srv.setupRoutes()

	return srv
}

func (srv *Server) setupRoutes() {
	pfx := srv.config.Prefix
	if pfx == "/" {
		srv.mux.HandleFunc("/", srv.handleDashboard)
		srv.mux.HandleFunc("/api/report", srv.handleReport)
		srv.mux.HandleFunc("/api/events", srv.handleSSE)
		srv.mux.HandleFunc("/api/health", srv.handleHealth)
	} else {
		srv.mux.HandleFunc(pfx+"/", srv.handleDashboard)
		srv.mux.HandleFunc(pfx+"/api/report", srv.handleReport)
		srv.mux.HandleFunc(pfx+"/api/events", srv.handleSSE)
		srv.mux.HandleFunc(pfx+"/api/health", srv.handleHealth)
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

// Shutdown gracefully shuts down the server.
func (srv *Server) Shutdown(ctx context.Context) error {
	srv.serverMu.Lock()
	server := srv.httpServer
	srv.serverMu.Unlock()

	if server == nil {
		return nil
	}

	return fmt.Errorf("shutdown: %w", server.Shutdown(ctx))
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

	data, err := makeReportJSON(srv.plugin)
	if err != nil {
		http.Error(w, fmt.Sprintf("generate report: %v", err), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(data)
}

type healthResponse struct {
	Status   string  `json:"status"`
	UptimeS  float64 `json:"uptime_s"`
	Clients  int     `json:"clients"`
	Events   int     `json:"events"`
	Complete bool    `json:"complete"`
	Dropped  int64   `json:"dropped"`
}

func (srv *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := healthResponse{
		Status:   "ok",
		UptimeS:  time.Since(srv.startTime).Seconds(),
		Clients:  srv.hub.ClientCount(),
		Complete: srv.hub.IsComplete(),
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
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", sse.ContentType)
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub := srv.hub.Subscribe()
	defer srv.hub.Unsubscribe(sub.id)

	if err := srv.sendSnapshot(w, flusher); err != nil {
		return
	}

	heartbeat := time.NewTicker(srv.config.HeartbeatInterval)
	defer heartbeat.Stop()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return

		case <-sub.done:
			srv.sendComplete(w, flusher)

			return

		case evt := <-sub.ch:
			if err := sse.WriteEvent(w, sse.Event{Event: "event", Data: string(evt)}); err != nil {
				return
			}

			flusher.Flush()

		case <-heartbeat.C:
			if _, err := w.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}

			flusher.Flush()
		}
	}
}

func (srv *Server) sendSnapshot(w http.ResponseWriter, flusher http.Flusher) error {
	plugin := srv.plugin
	if plugin == nil {
		return nil
	}

	report := plugin.Report()
	events := plugin.Events()

	data := snapshotData{
		Report:   report,
		Events:   events,
		Metadata: auditlog.BuildTypeMetadata(),
		DAG:      auditlog.BuildDAGHTML(report),
		Complete: srv.hub.IsComplete(),
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("build snapshot: %w", err)
	}

	if err := sse.WriteEvent(w, sse.Event{Event: "snapshot", Data: string(payload)}); err != nil {
		return err
	}

	flusher.Flush()

	return nil
}

func (srv *Server) sendComplete(w http.ResponseWriter, flusher http.Flusher) {
	plugin := srv.plugin
	if plugin == nil {
		return
	}

	report := plugin.Report()

	data := completeData{
		Report: report,
		DAG:    auditlog.BuildDAGHTML(report),
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	_ = sse.WriteEvent(w, sse.Event{Event: "complete", Data: string(payload)})

	flusher.Flush()
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
