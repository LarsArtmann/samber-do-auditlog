package live

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

const (
	maxServiceRows = 50
	maxEventRows   = 100
)

// servicesShowExpr is the data-show expression for each service row.
var servicesShowExpr = "(!$serviceSearch || $rowName.toLowerCase().includes($serviceSearch.toLowerCase()) || $rowScope.toLowerCase().includes($serviceSearch.toLowerCase())) && ($showAllServices || $rowIdx < " + strconv.Itoa(maxServiceRows) + ")"

// eventsShowExpr is the data-show expression for each event row.
var eventsShowExpr = "(!$eventFilter || $evtType === $eventFilter) && ($showAllEvents || $evtIdx < " + strconv.Itoa(maxEventRows) + ")"

// rowSignals is the JSON payload embedded in each table row's data-signals
// attribute, enabling client-side search/filter/pagination via datastar.
type rowSignals struct {
	RowName  string `json:"rowName"`
	RowScope string `json:"rowScope"`
	RowIdx   int    `json:"rowIdx"`
}

type eventRowSignals struct {
	EvtType string `json:"evtType"`
	EvtIdx  int    `json:"evtIdx"`
}

// --- Fragment renderers ---

// renderStatsFragment renders the stats card row.
func renderStatsFragment(report auditlog.Report) string {
	var b strings.Builder

	errorCount := 0

	for _, svc := range report.Services {
		if svc.Status.IsError() {
			errorCount++
		}
	}

	stats := []struct {
		label string
		value string
		cls   string
	}{
		{"Services", strconv.Itoa(report.ServiceCount), ""},
		{"Events", strconv.Itoa(report.EventCount), ""},
		{"Scopes", strconv.Itoa(report.ScopeCount), ""},
		{"Errors", strconv.Itoa(errorCount), errorCountClass(errorCount)},
		{"Build (ms)", humanizeDuration(report.TotalBuildDurationMs), ""},
		{"Shutdown (ms)", humanizeDuration(report.TotalShutdownDurationMs), ""},
	}

	if report.HealthCheckedCount > 0 {
		cls := "success"
		if !report.HealthCheckSucceeded {
			cls = "error"
		}

		stats = append(stats, struct {
			label string
			value string
			cls   string
		}{"Health", healthLabel(report.HealthCheckSucceeded), cls})
	}

	for _, s := range stats {
		cls := s.cls
		if cls != "" {
			cls = " " + cls
		}

		fmt.Fprintf(&b, `<div class="stat-card%s"><div class="label">%s</div><div class="value">%s</div></div>`,
			cls, s.label, s.value)
	}

	return b.String()
}

// renderLegendFragment renders the provider type legend.
func renderLegendFragment(report auditlog.Report, meta auditlog.TypeMetadata) string {
	var b strings.Builder

	counts := map[string]int{}

	for _, svc := range report.Services {
		if svc.ServiceType != "" {
			counts[string(svc.ServiceType)]++
		}
	}

	order := []string{"lazy", "eager", "transient", "alias"}

	for _, k := range order {
		count := counts[k]
		if count == 0 {
			continue
		}

		icon := ""
		label := k

		if p, ok := meta.Providers[k]; ok {
			icon = p.Icon
			label = p.Label
		}

		fmt.Fprintf(&b, `<div class="legend-item"><span class="icon">%s</span>%s <span style="opacity:0.5">(%d)</span></div>`,
			html.EscapeString(icon), html.EscapeString(label), count)
	}

	return b.String()
}

// renderWaveformFragment renders the event timeline waveform.
func renderWaveformFragment(events []auditlog.Event, meta auditlog.TypeMetadata) string {
	if len(events) == 0 {
		return `<span class="waveform-placeholder">Waiting for events...</span>`
	}

	var b strings.Builder

	minT := events[0].Timestamp.UnixMilli()
	maxT := minT

	for _, e := range events {
		t := e.Timestamp.UnixMilli()
		if t < minT {
			minT = t
		}

		if t > maxT {
			maxT = t
		}
	}

	rangeMs := maxT - minT
	if rangeMs == 0 {
		rangeMs = 1
	}

	maxDur := 1.0

	for _, e := range events {
		if e.DurationMs != nil && *e.DurationMs > maxDur {
			maxDur = *e.DurationMs
		}
	}

	for _, e := range events {
		t := e.Timestamp.UnixMilli()
		pct := float64(t-minT) / float64(rangeMs) * 100

		color := "var(--text-muted)"
		if em, ok := meta.Events[string(e.EventType)]; ok && em.Color != "" {
			color = em.Color
		}

		hasErr := e.Error != nil
		if hasErr {
			color = "var(--error)"
		}

		height := 4.0
		if e.DurationMs != nil && *e.DurationMs > 0 {
			height = math.Max(4, *e.DurationMs/maxDur*28)
		}

		tip := string(e.EventType)
		if e.ServiceName != "" {
			tip += " " + string(e.ServiceName)
		}

		if e.Phase != "" {
			tip += " " + string(e.Phase)
		}

		if e.DurationMs != nil {
			tip += " " + humanizeDuration(*e.DurationMs)
		}

		fmt.Fprintf(&b,
			`<div class="wf-event" style="left:%.2f%%;height:%.0fpx;background:%s" title="%s"></div>`,
			pct, height, color, html.EscapeString(tip))
	}

	return b.String()
}

// renderServicesTbody renders the services table body with datastar attributes
// for client-side search/filter/pagination.
func renderServicesTbody(report auditlog.Report, meta auditlog.TypeMetadata) string {
	services := report.Services

	if len(services) == 0 {
		return `<tr class="empty-state"><td colspan="8">No services registered yet.</td></tr>`
	}

	var b strings.Builder

	for i, svc := range services {
		signals, _ := json.Marshal(rowSignals{
			RowName:  string(svc.ServiceName),
			RowScope: svc.ScopeName,
			RowIdx:   i,
		})

		b.WriteString(`<tr data-signals="`)
		b.WriteString(html.EscapeString(string(signals)))
		b.WriteString(`" data-show="`)
		b.WriteString(html.EscapeString(servicesShowExpr))
		b.WriteString(`">`)

		icon := providerIcon(meta, string(svc.ServiceType))
		statusIcon := statusIcon(meta, string(svc.Status))
		buildMs := "&mdash;"

		if svc.FirstBuildDurationMs != nil {
			buildMs = humanizeDuration(*svc.FirstBuildDurationMs)
		}

		depNames := depNamesString(svc.Dependencies)

		errorText, errorTooltip := serviceError(svc)

		fmt.Fprintf(&b, `<td>%s %s</td>`, html.EscapeString(icon), html.EscapeString(string(svc.ServiceName)))
		fmt.Fprintf(&b, `<td>%s</td>`, html.EscapeString(svc.ScopeName))
		fmt.Fprintf(&b, `<td>%s</td>`, html.EscapeString(string(svc.ServiceType)))
		fmt.Fprintf(&b, `<td>%s %s</td>`, html.EscapeString(statusIcon), html.EscapeString(string(svc.Status)))
		fmt.Fprintf(&b, `<td>%d</td>`, svc.InvocationCount)
		fmt.Fprintf(&b, `<td>%s</td>`, buildMs)
		fmt.Fprintf(&b, `<td>%s</td>`, depNames)
		fmt.Fprintf(&b, `<td title="%s">%s</td>`, errorTooltip, errorText)

		b.WriteString(`</tr>`)
	}

	return b.String()
}

// renderEventsTbody renders the events table body with datastar attributes.
func renderEventsTbody(events []auditlog.Event, meta auditlog.TypeMetadata) string {
	if len(events) == 0 {
		return `<tr class="empty-state"><td colspan="7">No events recorded yet.</td></tr>`
	}

	var b strings.Builder

	for i, e := range events {
		signals, _ := json.Marshal(eventRowSignals{
			EvtType: string(e.EventType),
			EvtIdx:  i,
		})

		b.WriteString(`<tr data-signals="`)
		b.WriteString(html.EscapeString(string(signals)))
		b.WriteString(`" data-show="`)
		b.WriteString(html.EscapeString(eventsShowExpr))
		b.WriteString(`">`)

		label := string(e.EventType)
		color := "var(--text-muted)"

		if em, ok := meta.Events[string(e.EventType)]; ok {
			if em.Label != "" {
				label = em.Label
			}

			if em.Color != "" {
				color = em.Color
			}
		}

		phase := "&#9662;"
		if e.Phase == auditlog.PhaseBefore {
			phase = "&#9652;"
		}

		dur := "&mdash;"
		if e.DurationMs != nil {
			dur = humanizeDuration(*e.DurationMs)
		}

		errStr := ""
		errTooltip := ""

		if e.Error != nil {
			errTooltip = html.EscapeString(*e.Error)
			truncErr := *e.Error
			if len(truncErr) > 30 {
				truncErr = truncErr[:30]
			}

			errStr = "&#9888; " + html.EscapeString(truncErr)
		}

		fmt.Fprintf(&b, `<td>%d</td>`, i+1)
		fmt.Fprintf(&b, `<td>%s</td>`, e.Timestamp.Format("15:04:05"))
		fmt.Fprintf(&b, `<td><span class="event-badge" style="background:%s">%s</span></td>`, color, html.EscapeString(label))
		fmt.Fprintf(&b, `<td>%s %s</td>`, phase, html.EscapeString(string(e.Phase)))
		fmt.Fprintf(&b, `<td>%s</td>`, html.EscapeString(string(e.ServiceName)))
		fmt.Fprintf(&b, `<td>%s</td>`, dur)
		fmt.Fprintf(&b, `<td title="%s">%s</td>`, errTooltip, errStr)

		b.WriteString(`</tr>`)
	}

	return b.String()
}

// renderScopeTreeFragment renders the scope tree as nested HTML.
func renderScopeTreeFragment(report auditlog.Report) string {
	tree := report.ScopeTree
	if tree.ID == "" && tree.Name == "" && len(tree.Children) == 0 {
		return `<div class="graph-placeholder">Scope tree will appear here once services register...</div>`
	}

	return renderScopeNode(tree, 0)
}

func renderScopeNode(node auditlog.ScopeNode, depth int) string {
	var b strings.Builder

	marginLeft := depth * 20

	b.WriteString(fmt.Sprintf(`<div class="scope-node" style="margin-left:%dpx">`, marginLeft))
	b.WriteString(`<div class="scope-label" role="button" tabindex="0" aria-expanded="true">`)
	b.WriteString(`<span class="scope-icon" aria-hidden="true">&#9660;</span>`)

	name := node.Name
	if name == "" {
		name = string(node.ID)
	}

	if name == "" {
		name = "scope"
	}

	b.WriteString(html.EscapeString(name))
	b.WriteString(`</div>`)
	b.WriteString(`<div class="scope-body">`)

	if len(node.Services) > 0 {
		b.WriteString(`<div class="scope-services">`)

		for _, svc := range node.Services {
			b.WriteString(`<span class="scope-service-chip">`)
			b.WriteString(html.EscapeString(string(svc)))
			b.WriteString(`</span>`)
		}

		b.WriteString(`</div>`)
	}

	if len(node.Children) > 0 {
		b.WriteString(`<div class="scope-children">`)

		for _, child := range node.Children {
			b.WriteString(renderScopeNode(child, depth+1))
		}

		b.WriteString(`</div>`)
	}

	b.WriteString(`</div>`)
	b.WriteString(`</div>`)

	return b.String()
}

// renderFooterStats renders the footer stats text.
func renderFooterStats(report auditlog.Report, eventCount int) string {
	version := report.Version
	if version == "" {
		version = "?"
	}

	return fmt.Sprintf("Schema v%s | %d events | %d services",
		html.EscapeString(version), eventCount, report.ServiceCount)
}

// renderContainerId renders the container ID for the header.
func renderContainerID(report auditlog.Report) string {
	if report.ContainerID == "" {
		return "&mdash;"
	}

	return html.EscapeString(string(report.ContainerID))
}

// --- Fragment descriptor ---

type fragmentPatch struct {
	selector string
	html     string
}

// renderAllFragments renders every dashboard section and returns the list of
// (selector, html) pairs to send as datastar-patch-elements events.
func renderAllFragments(report auditlog.Report, events []auditlog.Event, meta auditlog.TypeMetadata) []fragmentPatch {
	return []fragmentPatch{
		{"#stats", renderStatsFragment(report)},
		{"#legend", renderLegendFragment(report, meta)},
		{"#waveform", renderWaveformFragment(events, meta)},
		{"#services-tbody", renderServicesTbody(report, meta)},
		{"#events-tbody", renderEventsTbody(events, meta)},
		{"#scope-tree-container", renderScopeTreeFragment(report)},
		{"#footer-stats", renderFooterStats(report, len(events))},
		{"#container-id", renderContainerID(report)},
	}
}

// --- Helpers ---

func humanizeDuration(ms float64) string {
	if ms < 0 {
		return "&mdash;"
	}

	if ms < 1 {
		return strconv.FormatFloat(ms, 'f', 3, 64) + "ms"
	}

	if ms < 1000 {
		return strconv.FormatFloat(ms, 'f', 1, 64) + "ms"
	}

	s := ms / 1000
	if s < 60 {
		return strconv.FormatFloat(s, 'f', 1, 64) + "s"
	}

	m := math.Floor(s / 60)
	rem := s - m*60
	if m < 60 {
		return strconv.Itoa(int(m)) + "m " + strconv.Itoa(int(math.Round(rem))) + "s"
	}

	h := math.Floor(m / 60)
	remM := m - h*60

	return strconv.Itoa(int(h)) + "h " + strconv.Itoa(int(remM)) + "m"
}

func providerIcon(meta auditlog.TypeMetadata, svcType string) string {
	if p, ok := meta.Providers[svcType]; ok {
		return p.Icon
	}

	return ""
}

func statusIcon(meta auditlog.TypeMetadata, status string) string {
	if s, ok := meta.Statuses[status]; ok {
		return s.Icon
	}

	return ""
}

func depNamesString(deps []auditlog.ServiceRef) string {
	if len(deps) == 0 {
		return "&mdash;"
	}

	parts := make([]string, len(deps))
	for i, d := range deps {
		parts[i] = html.EscapeString(string(d.ServiceName))
	}

	return strings.Join(parts, ", ")
}

func serviceError(svc auditlog.ServiceInfo) (text, tooltip string) {
	errStr := ""
	if svc.InvocationError != nil {
		errStr = *svc.InvocationError
	} else if svc.ShutdownError != nil {
		errStr = *svc.ShutdownError
	}

	if errStr == "" {
		return "&mdash;", ""
	}

	escErr := html.EscapeString(errStr)
	truncErr := errStr
	if len(truncErr) > 40 {
		truncErr = truncErr[:40]
	}

	return "&#9888; " + html.EscapeString(truncErr), escErr
}

func errorCountClass(count int) string {
	if count > 0 {
		return "error"
	}

	return "success"
}

func healthLabel(succeeded bool) string {
	if succeeded {
		return "Pass"
	}

	return "Fail"
}
