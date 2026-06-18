# Live Event Stream over WebSocket

This is a **reference example** — `samber-do-auditlog` does **not** depend on any
WebSocket library. The `Config.OnEvent` callback fires for every lifecycle
event outside the recorder mutex, making it a natural source for a live stream
that a browser dashboard can subscribe to.

## Concept

The `OnEvent` callback receives each `Event` the instant it is captured. To
stream them to browsers:

1. Run a WebSocket **hub** that fans out messages to all connected clients.
2. Feed the hub from `OnEvent` by marshaling each event to JSON.
3. Expose an HTTP endpoint that upgrades to WebSocket and registers the client
   with the hub.

This gives you a real-time view of container activity (registrations,
invocations, shutdowns, health checks) identical to the Events tab in the HTML
report, but live.

## Reference Implementation

```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// hub fans out audit events to every connected WebSocket client. It is safe
// for concurrent use.
type hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func newHub() *hub {
	return &hub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *hub) subscribe(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *hub) unsubscribe(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// broadcast sends an event to every client. A slow or closed client is dropped.
func (h *hub) broadcast(evt auditlog.Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		// Non-blocking write with a short timeout; drop on failure.
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		if err := c.Write(ctx, websocket.MessageText, data); err != nil {
			h.unsubscribe(c)
		}
		cancel()
	}
}

func main() {
	h := newHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled: true,
		OnEvent: h.broadcast, // ← every event is streamed to subscribers
	})
	if err != nil {
		log.Fatal(err)
	}
	_ = plugin // → do.NewWithOpts(plugin.Opts())

	// WebSocket endpoint for live subscribers.
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// In production, restrict Origin to your dashboard host.
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		h.subscribe(conn)

		// Block until the client disconnects or the context is cancelled.
		<-r.Context().Done()
		h.unsubscribe(conn)
	})

	log.Println("live events on ws://localhost:8080/events")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Browser Client (snippet)

```html
<ul id="log"></ul>
<script>
  const ws = new WebSocket("ws://localhost:8080/events");
  ws.onmessage = (msg) => {
    const evt = JSON.parse(msg.data);
    const li = document.createElement("li");
    li.textContent = `${evt.event_type} ${evt.phase} → ${evt.service_name}`;
    document.getElementById("log").prepend(li);
  };
</script>
```

## Notes

- **Backpressure**: the example drops slow clients. For guaranteed delivery,
  add a per-client buffered channel and a dedicated writer goroutine.
- **No replay on connect**: new subscribers only see events from the moment
  they connect. To offer a backlog, keep a ring buffer of recent events and
  flush it on subscribe.
- **Combine with the report**: the WebSocket stream is for live observability;
  `Report.WriteJSON` / `ExportToHTML` remain the source of truth for a full,
  point-in-time snapshot.
