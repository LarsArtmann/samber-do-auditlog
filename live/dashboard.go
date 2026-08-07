package live

import (
	_ "embed"
	"fmt"
	"strings"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

//go:embed dashboard.css
var liveCSS string

//go:embed dashboard.js
var liveJS string

//go:embed datastar.js
var datastarJS string

// liveTemplate is the HTML skeleton for the datastar-powered live dashboard.
// The seven %s verbs receive: 1) base CSS, 2) live-specific CSS, 3) schema
// version, 4) event filter chips HTML, 5) prefix (injected as JS variable),
// 6) datastar.js runtime, 7) inline JS (keyboard nav + export + scope toggle).
//
// Unlike the old dashboard, no report data is embedded and no rendering JS
// exists — all data arrives via SSE as datastar-patch-elements events that
// morph the DOM by element ID. The datastar.js runtime (~56KB) handles SSE
// parsing, DOM morphing, and signal reactivity.
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
<body data-signals="{activeTab: 'services', serviceSearch: '', eventFilter: '', showAllServices: false, showAllEvents: false, complete: false, connStatus: 'connecting', servicesOverflow: false, eventsOverflow: false}" data-class:lifecycle-complete="$complete">
<a href="#main-content" class="skip-link">Skip to main content</a>
<header>
  <div class="header-left">
    <h1><span class="logo-dot live-dot"></span>samber-do-auditlog<span class="version">v%s</span> <span class="live-badge" id="live-badge"><span class="live-pulse"></span><span data-text="$complete ? 'DONE' : 'LIVE'">LIVE</span></span></h1>
    <p class="subtitle">Container <span class="mono" id="container-id">&mdash;</span> &mdash; <span id="connection-status" class="conn-status connecting" data-text="$connStatus">connecting...</span></p>
  </div>
  <div class="header-actions">
    <button class="export-btn" data-on:click="js:exportReport('json')" title="Download report as JSON">JSON</button>
    <button class="export-btn" data-on:click="js:exportReport('ndjson')" title="Download events as NDJSON">NDJSON</button>
    <button class="export-btn" data-on:click="js:exportReport('html')" title="Download self-contained HTML report">HTML</button>
  </div>
  <div class="legend" id="legend"></div>
</header>
<main id="main-content" tabindex="-1">
<div data-init="@get('%s/api/events')" data-on:keydown="js:handleKeydown(evt)"></div>
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
  <button class="tab" data-tab="services" data-on:click="$activeTab = 'services'" data-class:active="$activeTab === 'services'" role="tab" aria-selected="true" aria-controls="tab-services" id="tab-btn-services">Services <span class="tab-hint">(1)</span></button>
  <button class="tab" data-tab="scopes" data-on:click="$activeTab = 'scopes'" data-class:active="$activeTab === 'scopes'" role="tab" aria-selected="false" aria-controls="tab-scopes" id="tab-btn-scopes">Scopes <span class="tab-hint">(2)</span></button>
  <button class="tab" data-tab="graph" data-on:click="$activeTab = 'graph'" data-class:active="$activeTab === 'graph'" role="tab" aria-selected="false" aria-controls="tab-graph" id="tab-btn-graph">Graph <span class="tab-hint">(3)</span></button>
  <button class="tab" data-tab="timeline" data-on:click="$activeTab = 'timeline'" data-class:active="$activeTab === 'timeline'" role="tab" aria-selected="false" aria-controls="tab-timeline" id="tab-btn-timeline">Timeline <span class="tab-hint">(4)</span></button>
  <button class="tab" data-tab="events" data-on:click="$activeTab = 'events'" data-class:active="$activeTab === 'events'" role="tab" aria-selected="false" aria-controls="tab-events" id="tab-btn-events">Events <span class="tab-hint">(5)</span></button>
</div>
<div class="tab-content active" id="tab-services" role="tabpanel" aria-labelledby="tab-btn-services" data-class:active="$activeTab === 'services'">
  <div class="filter-bar">
    <label for="service-search" class="sr-only">Filter services by name</label>
    <input type="text" id="service-search" placeholder="Filter services..." aria-label="Filter services by name" data-bind="serviceSearch">
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
    </table>
  </div>
  <div class="pagination-bar" data-show="!$showAllServices && $servicesOverflow">
    <button class="show-all-btn" data-on:click="$showAllServices = true">Show all</button>
  </div>
</div>
<div class="tab-content" id="tab-scopes" role="tabpanel" aria-labelledby="tab-btn-scopes" data-class:active="$activeTab === 'scopes'">
  <div id="scope-tree-container">
    <div class="graph-placeholder">Scope tree will appear here once services register...</div>
  </div>
</div>
<div class="tab-content" id="tab-graph" role="tabpanel" aria-labelledby="tab-btn-graph" data-class:active="$activeTab === 'graph'">
  <div id="graph-container">
    <div class="graph-placeholder" id="graph-placeholder">Dependency graph will appear here as services register...</div>
  </div>
</div>
<div class="tab-content" id="tab-timeline" role="tabpanel" aria-labelledby="tab-btn-timeline" data-class:active="$activeTab === 'timeline'">
  <div id="timeline-container">
    <div class="graph-placeholder">Timeline will appear here as events arrive...</div>
  </div>
</div>
<div class="tab-content" id="tab-events" role="tabpanel" aria-labelledby="tab-btn-events" data-class:active="$activeTab === 'events'">
  <div class="filter-bar" id="event-filters" role="group" aria-label="Filter events by type">%s</div>
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th scope="col">#</th><th scope="col">Time</th><th scope="col">Type</th><th scope="col">Phase</th><th scope="col">Service</th><th scope="col">Duration</th><th scope="col">Error</th></tr>
      </thead>
      <tbody id="events-tbody"></tbody>
    </table>
  </div>
  <div class="pagination-bar" data-show="!$showAllEvents && $eventsOverflow">
    <button class="show-all-btn" data-on:click="$showAllEvents = true">Show all</button>
  </div>
</div>
<script>window.__LIVE_PREFIX="%s";</script>
<script type="module">
%s
</script>
<script>
%s
</script>
</main>
<div class="footer">
  <span>Generated by <strong>samber-do-auditlog live</strong> &middot; <span id="footer-ts"></span></span>
  <span id="footer-stats"></span> &middot; Press ? for keyboard shortcuts
</div>
</body>
</html>`

// renderEventFilterChips builds the event type filter chip buttons with
// datastar attributes for reactive filtering.
func renderEventFilterChips() string {
	meta := auditlog.BuildTypeMetadata()

	types := []string{"registration", "invocation", "shutdown", "health_check"}

	var b strings.Builder

	b.WriteString(`<button class="chip active" data-on:click="$eventFilter = ''" data-class:active="!$eventFilter" aria-pressed="true">All</button>`) //nolint:golines,lll // single HTML element

	for _, t := range types {
		label := t
		color := cssVarTextMuted

		if evtMeta, ok := meta.Events[t]; ok {
			if evtMeta.Label != "" {
				label = evtMeta.Label
			}

			if evtMeta.Color != "" {
				color = evtMeta.Color
			}
		}

		fmt.Fprintf(&b,
			`<button class="chip" data-on:click="$eventFilter = '%s'" data-class:active="$eventFilter === '%s'" aria-pressed="false" style="border-color:%s">%s</button>`,
			t, t, color, label)
	}

	return b.String()
}

// renderDashboardHTML builds the static HTML dashboard string. This is called
// once at server startup (not per-request) since all dynamic data flows via
// SSE as datastar-patch-elements events.
func renderDashboardHTML(prefix string) string {
	return fmt.Sprintf(
		liveTemplate,
		baseCSS,
		liveCSS,
		auditlog.SchemaVersion,
		prefix,
		renderEventFilterChips(),
		prefix,
		datastarJS,
		liveJS,
	)
}
