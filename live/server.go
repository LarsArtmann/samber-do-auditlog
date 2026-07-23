package live

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	corelive "github.com/larsartmann/auditlog-core/live"
	"github.com/larsartmann/go-output/daghtml"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// ErrServerAlreadyRunning is returned when ListenAndServe is called on a
// server that is already serving.
var ErrServerAlreadyRunning = corelive.ErrServerAlreadyRunning

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
	core   *corelive.Server
	hub    *Hub
	plugin *auditlog.Plugin
}

// New is the convenience constructor. It creates a Hub, wires it as the
// auditlog OnEvent callback, creates the Plugin, and returns a ready-to-use
// Server.
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

// NewServer creates a Server from an existing Hub and Plugin.
func NewServer(hub *Hub, plugin *auditlog.Plugin, cfg Config) *Server {
	server := &Server{
		hub:    hub,
		plugin: plugin,
	}

	if cfg.Prefix == "" {
		cfg.Prefix = "/debug/di"
	}

	coreCfg := corelive.Config{
		Addr:              cfg.Addr,
		Prefix:            cfg.Prefix,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
	}

	server.core = corelive.New(hub.core, coreCfg,
		corelive.WithReportProvider(makeReportProvider(plugin)),
		corelive.WithSnapshotProvider(makeSnapshotProvider(plugin)),
		corelive.WithCompleteProvider(makeCompleteProvider(plugin)),
		corelive.WithDashboardProvider(func() string { return renderDashboardHTML(cfg.Prefix) }),
		corelive.WithHealthProvider(makeHealthProvider(plugin)),
	)

	return server
}

// SignalComplete marks the container lifecycle as finished.
func (srv *Server) SignalComplete() {
	srv.core.SignalComplete()
}

// OnEvent broadcasts an event to all connected SSE clients.
func (srv *Server) OnEvent(evt auditlog.Event) {
	srv.hub.OnEvent(evt)
}

// ClientCount returns the number of currently connected SSE clients.
func (srv *Server) ClientCount() int {
	return srv.core.ClientCount()
}

// ServeHTTP implements http.Handler, delegating to the core server.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.core.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server.
func (srv *Server) ListenAndServe() error {
	return srv.core.ListenAndServe()
}

// Addr returns the server's listen address.
func (srv *Server) Addr() string {
	return srv.core.Addr()
}

// Shutdown gracefully shuts down the server.
func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.core.Shutdown(ctx)
}

func makeReportProvider(plugin *auditlog.Plugin) corelive.ReportProvider {
	return func() ([]byte, error) {
		report := plugin.Report()

		var buf bytes.Buffer

		encoder := json.NewEncoder(&buf)

		if err := encoder.Encode(report); err != nil {
			return nil, fmt.Errorf("encode report: %w", err)
		}

		return buf.Bytes(), nil
	}
}

func makeSnapshotProvider(plugin *auditlog.Plugin) corelive.SnapshotProvider {
	return func(isComplete bool) (json.RawMessage, error) {
		report := plugin.Report()
		events := plugin.Events()

		data := snapshotData{
			Report:   report,
			Events:   events,
			Metadata: auditlog.BuildTypeMetadata(),
			DAG:      auditlog.BuildDAGHTML(report),
			Complete: isComplete,
		}

		return json.Marshal(data)
	}
}

func makeCompleteProvider(plugin *auditlog.Plugin) corelive.CompleteProvider {
	return func() (json.RawMessage, error) {
		report := plugin.Report()

		data := completeData{
			Report: report,
			DAG:    auditlog.BuildDAGHTML(report),
		}

		return json.Marshal(data)
	}
}

func makeHealthProvider(plugin *auditlog.Plugin) corelive.HealthProvider {
	return func() corelive.HealthInfo {
		return corelive.HealthInfo{
			Events:  plugin.EventsCount(),
			Dropped: plugin.DroppedEventCount(),
		}
	}
}
