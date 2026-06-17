# Bridging OnEvent to OpenTelemetry

This is a **reference example** — `samber-do-auditlog` does **not** depend on
OpenTelemetry. The `Config.OnEvent` callback is designed so you can bridge
audit events to any observability backend (OTel, Prometheus, Datadog, etc.)
without coupling the library to a specific vendor.

## Concept

Every lifecycle event (registration, invocation, shutdown, health check) fires
the `OnEvent` callback outside the recorder mutex, so it is safe to do I/O
inside it. To bridge to OTel, create a tracer and start/end spans based on the
event phase.

## Reference Implementation

```go
package main

import (
	"context"
	"fmt"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// spanTracker maps a service+phase to its active span so that the "after"
// event can end the span started by the "before" event.
type spanTracker struct {
	tracer  trace.Tracer
	pending map[string]trace.Span // key: "serviceName:eventType"
}

func newOTelBridge(tp trace.TracerProvider) *spanTracker {
	return &spanTracker{
		tracer:  tp.Tracer("samber-do-auditlog"),
		pending: make(map[string]trace.Span),
	}
}

// bridge is the OnEvent callback. It creates spans for before-events and
// ends them on after-events, recording errors and durations.
func (st *spanTracker) bridge(evt auditlog.Event) {
	ctx := context.Background()
	key := fmt.Sprintf("%s:%s", evt.ServiceName, evt.EventType)

	attrs := []attribute.KeyValue{
		attribute.String("service.name", evt.ServiceName),
		attribute.String("service.scope", evt.ScopeName),
		attribute.String("audit.event_type", string(evt.EventType)),
		attribute.String("audit.container_id", evt.ContainerID),
		attribute.String("audit.service_type", string(evt.ServiceType)),
	}

	if evt.IsBefore() {
		// Start a span — it will be ended when the matching "after" event fires.
		_, span := st.tracer.Start(ctx, fmt.Sprintf("do.%s.%s", evt.EventType, evt.ServiceName),
			trace.WithAttributes(attrs...),
			trace.WithTimestamp(evt.Timestamp),
		)
		st.pending[key] = span
		return
	}

	// After event: end the span started by the before event.
	span, ok := st.pending[key]
	if !ok {
		// No matching before event (e.g. health checks are after-only).
		_, span = st.tracer.Start(ctx, fmt.Sprintf("do.%s.%s", evt.EventType, evt.ServiceName),
			trace.WithAttributes(attrs...),
			trace.WithTimestamp(evt.Timestamp),
		)
	}

	// Record duration if available.
	if evt.DurationMs != nil {
		span.SetAttributes(attribute.Float64("audit.duration_ms", *evt.DurationMs))
	}

	// Record errors.
	if evt.HasError() && evt.Error != nil {
		span.SetStatus(codes.Error, *evt.Error)
		span.RecordError(fmt.Errorf(*evt.Error))
	}

	span.End(trace.WithTimestamp(evt.Timestamp.Add(
		time.Duration(evt.Duration()) * time.Millisecond,
	)))
	delete(st.pending, key)
}
```

## Usage

```go
func main() {
	// Initialize your OTel tracer provider (provider setup omitted for brevity).
	tp := otel.GetTracerProvider()

	bridge := newOTelBridge(tp)

	plugin, err := auditlog.New(auditlog.Config{
		Enabled: true,
		OnEvent: bridge.bridge,
	})
	if err != nil {
		panic(err)
	}

	injector := do.NewWithOpts(plugin.Opts())
	// ... register and invoke services ...

	// Every lifecycle event now appears as an OTel span.
}
```

## Key Points

- **No dependency**: The `go.opentelemetry.io/otel` import is only in YOUR code,
  not in `samber-do-auditlog`. The library stays lightweight.
- **Non-blocking**: `OnEvent` is called outside the recorder mutex but must not
  block the hot path. If your OTel exporter is slow, buffer events in a channel
  and process them asynchronously.
- **Health checks are after-only**: There is no `PhaseBefore` for health check
  events (samber/do v2 has no pre-health-check hook). The bridge handles this
  by creating a self-contained span.
