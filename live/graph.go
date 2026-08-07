package live

import (
	"fmt"
	"html"
	"strings"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// renderGraphFragment renders the dependency graph as a server-side HTML
// structure. Each service is a node with its direct dependencies listed.
// No client-side JS or SVG rendering required — datastar morphs the HTML.
func renderGraphFragment(report auditlog.Report) string {
	if len(report.Services) == 0 {
		return `<div class="graph-placeholder" id="graph-placeholder">Dependency graph will appear here as services register...</div>`
	}

	var b strings.Builder

	b.WriteString(`<div class="dep-graph">`)

	for _, svc := range report.Services {
		b.WriteString(`<div class="dep-node">`)
		b.WriteString(`<div class="dep-node-header">`)
		b.WriteString(`<span class="dep-node-name">`)
		b.WriteString(html.EscapeString(string(svc.ServiceName)))
		b.WriteString(`</span>`)

		if svc.ServiceType != "" {
			b.WriteString(`<span class="dep-node-type">`)
			b.WriteString(html.EscapeString(string(svc.ServiceType)))
			b.WriteString(`</span>`)
		}

		b.WriteString(`</div>`)

		if len(svc.Dependencies) > 0 {
			b.WriteString(`<div class="dep-node-deps">`)

			for _, dep := range svc.Dependencies {
				fmt.Fprintf(&b, `<span class="dep-arrow">%s</span>`,
					html.EscapeString(string(dep.ServiceName)))
			}

			b.WriteString(`</div>`)
		}

		b.WriteString(`</div>`)
	}

	b.WriteString(`</div>`)

	return b.String()
}

// renderTimelineFragment renders build and shutdown duration bars for each
// service as horizontal HTML bars. The width of each bar is proportional to
// the service's build or shutdown duration relative to the maximum.
func renderTimelineFragment(report auditlog.Report) string {
	services := report.Services

	if len(services) == 0 {
		return `<div class="graph-placeholder">Timeline will appear here as events arrive...</div>`
	}

	maxBuildMs, maxShutdownMs := timelineMaxDurations(services)

	var b strings.Builder

	b.WriteString(`<div class="timeline">`)

	for _, svc := range services {
		b.WriteString(`<div class="timeline-row">`)

		b.WriteString(`<div class="timeline-label">`)
		b.WriteString(html.EscapeString(string(svc.ServiceName)))
		b.WriteString(`</div>`)

		b.WriteString(`<div class="timeline-bars">`)

		buildWidth := timelineBarWidth(svc.FirstBuildDurationMs, maxBuildMs)
		shutdownWidth := timelineBarWidth(svc.ShutdownDurationMs, maxShutdownMs)

		fmt.Fprintf(&b, `<div class="timeline-bar build" style="width:%s" title="Build: %s"></div>`,
			buildWidth, humanizeDuration(safeDuration(svc.FirstBuildDurationMs)))
		fmt.Fprintf(&b, `<div class="timeline-bar shutdown" style="width:%s" title="Shutdown: %s"></div>`,
			shutdownWidth, humanizeDuration(safeDuration(svc.ShutdownDurationMs)))

		b.WriteString(`</div>`)
		b.WriteString(`</div>`)
	}

	b.WriteString(`</div>`)

	return b.String()
}

// timelineMaxDurations returns the maximum build and shutdown durations
// across all services. Used to scale the timeline bars.
func timelineMaxDurations(services []auditlog.ServiceInfo) (maxBuild, maxShutdown float64) {
	for _, svc := range services {
		if svc.FirstBuildDurationMs != nil && *svc.FirstBuildDurationMs > maxBuild {
			maxBuild = *svc.FirstBuildDurationMs
		}

		if svc.ShutdownDurationMs != nil && *svc.ShutdownDurationMs > maxShutdown {
			maxShutdown = *svc.ShutdownDurationMs
		}
	}

	if maxBuild == 0 {
		maxBuild = 1
	}

	if maxShutdown == 0 {
		maxShutdown = 1
	}

	return maxBuild, maxShutdown
}

// timelineBarWidth returns a CSS width percentage string for a duration bar.
func timelineBarWidth(durationMs *float64, maxMs float64) string {
	if durationMs == nil || *durationMs <= 0 || maxMs <= 0 {
		return "0%"
	}

	pct := *durationMs / maxMs * 100

	return fmt.Sprintf("%.1f%%", pct)
}

// safeDuration dereferences a *float64, returning 0 if nil.
func safeDuration(durationMs *float64) float64 {
	if durationMs == nil {
		return 0
	}

	return *durationMs
}
