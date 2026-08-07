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
	maxServiceRows   = 50
	maxEventRows     = 100
	maxEventErrLen   = 30
	maxServiceErrLen = 40
	scopeIndentPx    = 20

	// Duration conversion constants.
	msPerSecond  = 1000.0
	secPerMinute = 60.0
	minPerHour   = 60.0

	// Waveform rendering constants.
	waveformMinHeight = 4.0
	waveformMaxHeight = 28.0
	waveformPctScale  = 100.0

	// Shared CSS/HTML tokens.
	cssVarTextMuted = "var(--text-muted)"
	mdash           = "&mdash;"
)

// servicesShowExpr is the data-show expression for each service row.
// It cannot be a const because it interpolates maxServiceRows via strconv.Itoa.
//
//nolint:gochecknoglobals // effectively immutable; interpolates a const
var servicesShowExpr = "(!$serviceSearch || $rowName.toLowerCase().includes($serviceSearch.toLowerCase()) || $rowScope.toLowerCase().includes($serviceSearch.toLowerCase())) && ($showAllServices || $rowIdx < " + strconv.Itoa(maxServiceRows) + ")" //nolint:golines // single expression

// eventsShowExpr is the data-show expression for each event row.
//
//nolint:gochecknoglobals // effectively immutable; interpolates a const
var eventsShowExpr = "(!$eventFilter || $evtType === $eventFilter) && ($showAllEvents || $evtIdx < " + strconv.Itoa(maxEventRows) + ")"

// rowSignals is the JSON payload embedded in each table row's data-signals
// attribute, enabling client-side search/filter/pagination via datastar.
// Tags use camelCase because datastar's signal system requires camelCase keys.
//
//nolint:tagliatelle // camelCase required by datastar signal system
type rowSignals struct {
	RowName  string `json:"rowName"`
	RowScope string `json:"rowScope"`
	RowIdx   int    `json:"rowIdx"`
}

// eventRowSignals is the JSON payload for event table rows. Tags use camelCase
// for datastar signal compatibility.
//
//nolint:tagliatelle // camelCase required by datastar signal system
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

	stats := buildStatsEntries(report, errorCount)

	for _, stat := range stats {
		cls := stat.cls

		if cls != "" {
			cls = " " + cls
		}

		fmt.Fprintf(&b, `<div class="stat-card%s"><div class="label">%s</div><div class="value">%s</div></div>`,
			cls, stat.label, stat.value)
	}

	return b.String()
}

type statsEntry struct {
	label string
	value string
	cls   string
}

func buildStatsEntries(report auditlog.Report, errorCount int) []statsEntry {
	stats := []statsEntry{
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

		stats = append(stats, statsEntry{"Health", healthLabel(report.HealthCheckSucceeded), cls})
	}

	return stats
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

	for _, providerType := range order {
		count := counts[providerType]
		if count == 0 {
			continue
		}

		icon := ""
		label := providerType

		if providerMeta, ok := meta.Providers[providerType]; ok {
			icon = providerMeta.Icon
			label = providerMeta.Label
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

	minT, maxT, maxDur := waveformBounds(events)

	rangeMs := maxT - minT
	if rangeMs == 0 {
		rangeMs = 1
	}

	var b strings.Builder

	for _, evt := range events {
		b.WriteString(waveformEventDiv(evt, minT, rangeMs, maxDur, meta))
	}

	return b.String()
}

// waveformBounds returns (minTimestamp, maxTimestamp, maxDuration) from a list
// of events. maxDuration defaults to 1.0 if no event has a duration.
func waveformBounds(events []auditlog.Event) (minT, maxT int64, maxDur float64) {
	minT = events[0].Timestamp.UnixMilli()
	maxT = minT
	maxDur = 1.0

	for _, evt := range events {
		ts := evt.Timestamp.UnixMilli()
		if ts < minT {
			minT = ts
		}

		if ts > maxT {
			maxT = ts
		}

		if evt.DurationMs != nil && *evt.DurationMs > maxDur {
			maxDur = *evt.DurationMs
		}
	}

	return minT, maxT, maxDur
}

// waveformEventDiv renders a single waveform event mark.
func waveformEventDiv(evt auditlog.Event, minT, rangeMs int64, maxDur float64, meta auditlog.TypeMetadata) string {
	ts := evt.Timestamp.UnixMilli()
	pct := float64(ts-minT) / float64(rangeMs) * waveformPctScale

	color := cssVarTextMuted

	if evtMeta, ok := meta.Events[string(evt.EventType)]; ok && evtMeta.Color != "" {
		color = evtMeta.Color
	}

	if evt.Error != nil {
		color = "var(--error)"
	}

	height := waveformMinHeight
	if evt.DurationMs != nil && *evt.DurationMs > 0 {
		height = math.Max(waveformMinHeight, *evt.DurationMs/maxDur*waveformMaxHeight)
	}

	tip := waveformTooltip(evt)

	return fmt.Sprintf(
		`<div class="wf-event" style="left:%.2f%%;height:%.0fpx;background:%s" title="%s"></div>`,
		pct, height, color, html.EscapeString(tip))
}

func waveformTooltip(evt auditlog.Event) string {
	tip := string(evt.EventType)
	if evt.ServiceName != "" {
		tip += " " + string(evt.ServiceName)
	}

	if evt.Phase != "" {
		tip += " " + string(evt.Phase)
	}

	if evt.DurationMs != nil {
		tip += " " + humanizeDuration(*evt.DurationMs)
	}

	return tip
}

// renderServicesTbody renders the services table body with datastar attributes
// for client-side search/filter/pagination.
func renderServicesTbody(report auditlog.Report, meta auditlog.TypeMetadata) string {
	services := report.Services

	if len(services) == 0 {
		return `<tr class="empty-state"><td colspan="8">No services registered yet.</td></tr>`
	}

	var b strings.Builder

	for idx, svc := range services {
		signals, err := json.Marshal(rowSignals{
			RowName:  string(svc.ServiceName),
			RowScope: svc.ScopeName,
			RowIdx:   idx,
		})
		if err != nil {
			continue
		}

		b.WriteString(`<tr data-signals="`)
		b.WriteString(html.EscapeString(string(signals)))
		b.WriteString(`" data-show="`)
		b.WriteString(html.EscapeString(servicesShowExpr))
		b.WriteString(`">`)

		b.WriteString(renderServiceRowCells(svc, meta))
		b.WriteString(`</tr>`)
	}

	return b.String()
}

func renderServiceRowCells(svc auditlog.ServiceInfo, meta auditlog.TypeMetadata) string {
	var b strings.Builder

	icon := providerIcon(meta, string(svc.ServiceType))
	statusIconStr := statusIcon(meta, string(svc.Status))
	buildMs := mdash

	if svc.FirstBuildDurationMs != nil {
		buildMs = humanizeDuration(*svc.FirstBuildDurationMs)
	}

	depNames := depNamesString(svc.Dependencies)
	errorText, errorTooltip := serviceError(svc)

	fmt.Fprintf(&b, `<td>%s %s</td>`, html.EscapeString(icon), html.EscapeString(string(svc.ServiceName)))
	fmt.Fprintf(&b, `<td>%s</td>`, html.EscapeString(svc.ScopeName))
	fmt.Fprintf(&b, `<td>%s</td>`, html.EscapeString(string(svc.ServiceType)))
	fmt.Fprintf(&b, `<td>%s %s</td>`, html.EscapeString(statusIconStr), html.EscapeString(string(svc.Status)))
	fmt.Fprintf(&b, `<td>%d</td>`, svc.InvocationCount)
	fmt.Fprintf(&b, `<td>%s</td>`, buildMs)
	fmt.Fprintf(&b, `<td>%s</td>`, depNames)
	fmt.Fprintf(&b, `<td title="%s">%s</td>`, errorTooltip, errorText)

	return b.String()
}

// renderEventsTbody renders the events table body with datastar attributes.
func renderEventsTbody(events []auditlog.Event, meta auditlog.TypeMetadata) string {
	if len(events) == 0 {
		return `<tr class="empty-state"><td colspan="7">No events recorded yet.</td></tr>`
	}

	var b strings.Builder

	for idx, evt := range events {
		b.WriteString(renderEventRow(evt, idx, meta))
	}

	return b.String()
}

func renderEventRow(evt auditlog.Event, idx int, meta auditlog.TypeMetadata) string {
	signals, err := json.Marshal(eventRowSignals{
		EvtType: string(evt.EventType),
		EvtIdx:  idx,
	})
	if err != nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(`<tr data-signals="`)
	b.WriteString(html.EscapeString(string(signals)))
	b.WriteString(`" data-show="`)
	b.WriteString(html.EscapeString(eventsShowExpr))
	b.WriteString(`">`)

	label := string(evt.EventType)
	color := cssVarTextMuted

	if evtMeta, ok := meta.Events[string(evt.EventType)]; ok {
		if evtMeta.Label != "" {
			label = evtMeta.Label
		}

		if evtMeta.Color != "" {
			color = evtMeta.Color
		}
	}

	phase := "&#9662;"
	if evt.Phase == auditlog.PhaseBefore {
		phase = "&#9652;"
	}

	dur := mdash
	if evt.DurationMs != nil {
		dur = humanizeDuration(*evt.DurationMs)
	}

	errStr, errTooltip := eventErrorFields(evt)

	fmt.Fprintf(&b, `<td>%d</td>`, idx+1)
	fmt.Fprintf(&b, `<td>%s</td>`, evt.Timestamp.Format("15:04:05"))
	fmt.Fprintf(&b, `<td><span class="event-badge" style="background:%s">%s</span></td>`, color, html.EscapeString(label))
	fmt.Fprintf(&b, `<td>%s %s</td>`, phase, html.EscapeString(string(evt.Phase)))
	fmt.Fprintf(&b, `<td>%s</td>`, html.EscapeString(string(evt.ServiceName)))
	fmt.Fprintf(&b, `<td>%s</td>`, dur)
	fmt.Fprintf(&b, `<td title="%s">%s</td>`, errTooltip, errStr)

	b.WriteString(`</tr>`)

	return b.String()
}

// eventErrorFields returns (displayText, tooltipText) for an event's error.
func eventErrorFields(evt auditlog.Event) (displayText, tooltipText string) {
	if evt.Error == nil {
		return "", ""
	}

	tooltipText = html.EscapeString(*evt.Error)

	truncErr := *evt.Error

	if len(truncErr) > maxEventErrLen {
		truncErr = truncErr[:maxEventErrLen]
	}

	displayText = "&#9888; " + html.EscapeString(truncErr)

	return displayText, tooltipText
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

	marginLeft := depth * scopeIndentPx

	fmt.Fprintf(&b, `<div class="scope-node" style="margin-left:%dpx">`, marginLeft)
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

// renderContainerID renders the container ID for the header.
func renderContainerID(report auditlog.Report) string {
	if report.ContainerID == "" {
		return mdash
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
		{"#graph-container", renderGraphFragment(report)},
		{"#timeline-container", renderTimelineFragment(report)},
		{"#footer-stats", renderFooterStats(report, len(events))},
		{"#container-id", renderContainerID(report)},
	}
}

// --- Helpers ---

func humanizeDuration(milliseconds float64) string {
	if milliseconds < 0 {
		return mdash
	}

	if milliseconds < 1 {
		return strconv.FormatFloat(milliseconds, 'f', 3, 64) + "ms"
	}

	if milliseconds < msPerSecond {
		return strconv.FormatFloat(milliseconds, 'f', 1, 64) + "ms"
	}

	secs := milliseconds / msPerSecond
	if secs < secPerMinute {
		return strconv.FormatFloat(secs, 'f', 1, 64) + "s"
	}

	minutes := math.Floor(secs / secPerMinute)
	remSecs := secs - minutes*secPerMinute
	if minutes < minPerHour {
		return strconv.Itoa(int(minutes)) + "m " + strconv.Itoa(int(math.Round(remSecs))) + "s"
	}

	hours := math.Floor(minutes / minPerHour)
	remMins := minutes - hours*minPerHour

	return strconv.Itoa(int(hours)) + "h " + strconv.Itoa(int(remMins)) + "m"
}

func providerIcon(meta auditlog.TypeMetadata, svcType string) string {
	if providerMeta, ok := meta.Providers[svcType]; ok {
		return providerMeta.Icon
	}

	return ""
}

func statusIcon(meta auditlog.TypeMetadata, status string) string {
	if statusMeta, ok := meta.Statuses[status]; ok {
		return statusMeta.Icon
	}

	return ""
}

func depNamesString(deps []auditlog.ServiceRef) string {
	if len(deps) == 0 {
		return mdash
	}

	parts := make([]string, 0, len(deps))
	for _, dep := range deps {
		parts = append(parts, html.EscapeString(string(dep.ServiceName)))
	}

	return strings.Join(parts, ", ")
}

func serviceError(svc auditlog.ServiceInfo) (string, string) {
	errStr := ""
	if svc.InvocationError != nil {
		errStr = *svc.InvocationError
	} else if svc.ShutdownError != nil {
		errStr = *svc.ShutdownError
	}

	if errStr == "" {
		return mdash, ""
	}

	escErr := html.EscapeString(errStr)
	truncErr := errStr

	if len(truncErr) > maxServiceErrLen {
		truncErr = truncErr[:maxServiceErrLen]
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
