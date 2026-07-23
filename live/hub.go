package live

import (
	"encoding/json"
	"sync"

	corelive "github.com/larsartmann/auditlog-core/live"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// Hub wraps corelive.Hub, adding domain-specific OnEvent method.
type Hub struct {
	core   *corelive.Hub
	plugin *auditlog.Plugin
	mu     sync.RWMutex
}

// NewHub creates a Hub. Pass nil when using live.New() (set internally).
func NewHub(plugin *auditlog.Plugin) *Hub {
	return &Hub{
		core:   corelive.NewHub(),
		plugin: plugin,
	}
}

// SetPlugin sets the plugin after construction.
func (h *Hub) SetPlugin(plugin *auditlog.Plugin) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.plugin = plugin
}

// OnEvent broadcasts an auditlog.Event to all connected SSE clients.
func (h *Hub) OnEvent(evt auditlog.Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}

	h.core.OnEvent(data)
}

// SignalComplete marks the container lifecycle as finished.
func (h *Hub) SignalComplete() {
	h.core.SignalComplete()
}

// IsComplete returns whether the lifecycle has been marked as complete.
func (h *Hub) IsComplete() bool {
	return h.core.IsComplete()
}

// ClientCount returns the number of currently connected SSE clients.
func (h *Hub) ClientCount() int {
	return h.core.ClientCount()
}

// Subscribe registers a new SSE client. For testing only.
func (h *Hub) Subscribe() *corelive.Subscriber {
	return h.core.Subscribe()
}

// Unsubscribe removes a subscriber by ID. For testing only.
func (h *Hub) Unsubscribe(id uint64) {
	h.core.Unsubscribe(id)
}
