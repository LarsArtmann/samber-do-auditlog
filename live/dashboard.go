package live

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-output/daghtml"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

//go:embed dashboard.css
var liveCSS string

//go:embed dashboard.js
var liveJS string

// liveTemplate is the HTML skeleton for the live dashboard. The six %s verbs
// receive: 1) base CSS, 2) live-specific CSS, 3) schema version, 4) type-metadata
// JSON, 5) live JS, 6) daghtml graph JS.
//
// Unlike the static dashboard, no report data is embedded — all data
// arrives via SSE at runtime.
const liveTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none';">
<title>samber-do-auditlog Live</title>
<style>
%s
%s</style>
</head>
<body>
<header>
  <div class="header-left">
    <h1><span class="logo-dot live-dot"></span>samber-do-auditlog<span class="version">v%s</span> <span class="live-badge" id="live-badge"><span class="live-pulse"></span>LIVE</span></h1>
    <p class="subtitle">Container <span class="mono" id="container-id">&mdash;</span> &mdash; <span id="connection-status" class="conn-status connecting">connecting...</span></p>
  </div>
  <div class="legend" id="legend"></div>
</header>
<div class="waveform-section">
  <span class="waveform-label">Event Timeline</span>
  <div class="waveform" id="waveform">
    <span class="waveform-placeholder">Waiting for events...</span>
  </div>
  <div class="waveform-legend">
    <span class="wf-legend-item"><span class="wf-legend-dot" style="background:var(--accent)"></span>registration</span>
    <span class="wf-legend-item"><span class="wf-legend-dot" style="background:var(--success)"></span>invocation</span>
    <span class="wf-legend-item"><span class="wf-legend-dot" style="background:var(--warning)"></span>shutdown</span>
    <span class="wf-legend-item"><span class="wf-legend-dot" style="background:var(--info)"></span>health_check</span>
  </div>
</div>
<div class="stats" id="stats">
  <div class="stat-placeholder">Connect to see live stats...</div>
</div>
<div class="tab-bar" role="tablist" aria-label="Report sections">
  <button class="tab active" data-tab="services" role="tab" aria-selected="true" aria-controls="tab-services" id="tab-btn-services">Services (1)</button>
  <button class="tab" data-tab="graph" role="tab" aria-selected="false" aria-controls="tab-graph" id="tab-btn-graph">Graph (2)</button>
  <button class="tab" data-tab="timeline" role="tab" aria-selected="false" aria-controls="tab-timeline" id="tab-btn-timeline">Timeline (3)</button>
  <button class="tab" data-tab="events" role="tab" aria-selected="false" aria-controls="tab-events" id="tab-btn-events">Events (4)</button>
</div>
<div class="tab-content active" id="tab-services" role="tabpanel" aria-labelledby="tab-btn-services">
  <div class="filter-bar">
    <label for="service-search" class="sr-only">Filter services by name</label>
    <input type="text" id="service-search" placeholder="Filter services..." aria-label="Filter services by name">
    <span id="service-result-count" style="font-size:0.75rem;color:var(--text-dim);font-family:var(--font-mono)"></span>
  </div>
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>Service</th>
          <th>Scope</th>
          <th>Type</th>
          <th>Status</th>
          <th>Invocations</th>
          <th>Build (ms)</th>
          <th>Dependencies</th>
          <th>Error</th>
        </tr>
      </thead>
      <tbody id="services-tbody"></tbody>
      <tbody id="services-empty" class="empty-state" style="display:none"><tr><td colspan="8">No services registered yet.</td></tr></tbody>
    </table>
  </div>
</div>
<div class="tab-content" id="tab-graph" role="tabpanel" aria-labelledby="tab-btn-graph">
  <div id="graph-container">
    <div class="graph-controls">
      <button class="graph-zoom-in" title="Zoom in" aria-label="Zoom in">+</button>
      <button class="graph-zoom-out" title="Zoom out" aria-label="Zoom out">&minus;</button>
      <button class="graph-fit" title="Fit to view" aria-label="Fit to view">&#8982;</button>
    </div>
    <div class="graph-info">Scroll/pinch to zoom &middot; Drag to pan &middot; Click node to highlight</div>
    <div class="graph-placeholder" id="graph-placeholder">Dependency graph will appear here as services register...</div>
  </div>
</div>
<div class="tab-content" id="tab-timeline" role="tabpanel" aria-labelledby="tab-btn-timeline">
  <div id="timeline-container">
    <div class="graph-placeholder">Timeline will appear here as events arrive...</div>
  </div>
</div>
<div class="tab-content" id="tab-events" role="tabpanel" aria-labelledby="tab-btn-events">
  <div class="filter-bar" id="event-filters" role="group" aria-label="Filter events by type"></div>
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th scope="col">#</th><th scope="col">Time</th><th scope="col">Type</th><th scope="col">Phase</th><th scope="col">Service</th><th scope="col">Duration</th><th scope="col">Error</th></tr>
      </thead>
      <tbody id="events-tbody"></tbody>
      <tbody id="events-empty" class="empty-state" style="display:none"><tr><td colspan="7">No events recorded yet.</td></tr></tbody>
    </table>
  </div>
</div>
<script type="application/json" id="type-metadata">%s</script>
<script>
%s</script>
<script>
%s</script>
<div class="footer">
  <span>Generated by <strong>samber-do-auditlog live</strong> &middot; <span id="footer-ts"></span></span>
  <span id="footer-stats"></span>
</div>
</body>
</html>`

// renderDashboardHTML builds the static HTML dashboard string. This is called
// once at server startup (not per-request) since all dynamic data flows via SSE.
func renderDashboardHTML() string {
	metadata := auditlog.BuildTypeMetadata()

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return "<html><body>failed to render dashboard</body></html>"
	}

	return fmt.Sprintf(
		liveTemplate,
		baseCSS,
		liveCSS,
		auditlog.SchemaVersion,
		metadataJSON,
		liveJS,
		daghtml.Script(),
	)
}
