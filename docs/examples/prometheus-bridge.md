# Bridging OnEvent to Prometheus

This is a **reference example** — `samber-do-auditlog` does **not** depend on
Prometheus. The `Config.OnEvent` callback is designed so you can bridge audit
events to any observability backend (Prometheus, OTel, Datadog, etc.) without
coupling the library to a specific vendor.

## Concept

Every lifecycle event (registration, invocation, shutdown, health check) fires
the `OnEvent` callback outside the recorder mutex, so it is safe to do I/O
inside it. To bridge to Prometheus, register metric instruments
(`Counter`, `Histogram`) with the `prometheus/client_golang` registry and update
them from the callback. A `/metrics` endpoint then exposes them for scraping.

The metrics chosen mirror the report's own data model:

| Metric                                         | Type      | Labels                                               | Source                    |
| ---------------------------------------------- | --------- | ---------------------------------------------------- | ------------------------- |
| `do_auditlog_events_total`                     | counter   | event_type, phase, service_name, scope, container_id | every event               |
| `do_auditlog_invocation_duration_milliseconds` | histogram | service_name, scope                                  | invocation `after` events |
| `do_auditlog_service_errors_total`             | counter   | service_name, scope, kind                            | events carrying an error  |

## Reference Implementation

```go
package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// promBridge holds the Prometheus instruments updated from OnEvent. All methods
// are safe to call concurrently — Prometheus collectors are designed for it.
type promBridge struct {
	events   *prometheus.CounterVec
	duration *prometheus.HistogramVec
	errors   *prometheus.CounterVec
}

func newPromBridge(reg prometheus.Registerer) *promBridge {
	pb := &promBridge{
		events: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "do_auditlog_events_total",
			Help: "Total audit events captured by samber-do-auditlog.",
		}, []string{"event_type", "phase", "service_name", "scope", "container_id"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "do_auditlog_invocation_duration_milliseconds",
			Help:    "Service invocation (build) duration in milliseconds.",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 0.1ms … ~409ms
		}, []string{"service_name", "scope"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "do_auditlog_service_errors_total",
			Help: "Service lifecycle errors (invocation or shutdown).",
		}, []string{"service_name", "scope", "kind"}),
	}

	reg.MustRegister(pb.events, pb.duration, pb.errors)

	return pb
}

// bridge is the OnEvent callback. It is cheap (a few atomic increments) and
// safe to call from the hot hook path.
func (pb *promBridge) bridge(evt auditlog.Event) {
	labels := prometheus.Labels{
		"event_type":   string(evt.EventType),
		"phase":        string(evt.Phase),
		"service_name": evt.ServiceName,
		"scope":        evt.ScopeName,
		"container_id": evt.ContainerID,
	}
	pb.events.With(labels).Inc()

	// Record build duration when an invocation completes.
	if evt.EventType == auditlog.EventTypeInvocation && evt.IsAfter() && evt.DurationMs != nil {
		pb.duration.With(prometheus.Labels{
			"service_name": evt.ServiceName,
			"scope":        evt.ScopeName,
		}).Observe(*evt.DurationMs)
	}

	// Surface any lifecycle error.
	if evt.HasError() {
		kind := "invocation"
		if evt.IsShutdown() {
			kind = "shutdown"
		}
		pb.errors.With(prometheus.Labels{
			"service_name": evt.ServiceName,
			"scope":        evt.ScopeName,
			"kind":         kind,
		}).Inc()
	}
}

func main() {
	pb := newPromBridge(prometheus.DefaultRegisterer)

	plugin, err := auditlog.New(auditlog.Config{
		Enabled: true,
		OnEvent: pb.bridge,
	})
	if err != nil {
		panic(err)
	}

	// Wire the plugin into samber/do, then register services and invoke them…
	_ = plugin // → do.NewWithOpts(plugin.Opts())

	// Expose the metrics for scraping.
	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(":9100", nil)
}
```

## Notes

- **No vendor lock-in**: the library only calls your `OnEvent` func. The
  Prometheus dependency lives entirely in your application.
- **Cardinality**: labels include `service_name` and `scope`. For very large
  container graphs, prefer dropping the `scope` label or aggregating by service
  type to keep cardinality bounded.
- **Durations are nil for health-check events** (the bulk health-check API does
  not report per-service timing), so the histogram is only observed for
  invocation `after` events.
- **Combine with the HTML/JSON report** for offline analysis: Prometheus gives
  you real-time trends, while `Report.WriteJSON` gives you a point-in-time
  snapshot for deep dives.
