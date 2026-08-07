package live

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

const (
	maxServiceRows   = 50
	maxEventRows     = 100
	maxEventErrLen   = 30
	maxServiceErrLen = 40
	scopeIndentPx    = 20

	msPerSecond  = 1000.0
	secPerMinute = 60.0
	minPerHour   = 60.0

	waveformMinHeight = 4.0
	waveformMaxHeight = 28.0
	waveformPctScale  = 100.0

	cssVarTextMuted = "var(--text-muted)"
	mdash           = "&mdash;"
)

// servicesShowExpr is the data-show expression for each service row.
//
//nolint:gochecknoglobals // effectively immutable; interpolates a const
var servicesShowExpr = "(!$serviceSearch || $rowName.toLowerCase().includes($serviceSearch.toLowerCase()) || $rowScope.toLowerCase().includes($serviceSearch.toLowerCase())) && ($showAllServices || $rowIdx < " + strconv.Itoa(maxServiceRows) + ")" //nolint:golines // single expression

// eventsShowExpr is the data-show expression for each event row.
//
//nolint:gochecknoglobals // effectively immutable; interpolates a const
var eventsShowExpr = "(!$eventFilter || $evtType === $eventFilter) && ($showAllEvents || $evtIdx < " + strconv.Itoa(maxEventRows) + ")" //nolint:golines,lll // single expression

// --- Data types for templ components ---

type statsEntry struct {
	label string
	value string
	cls   string
}

type legendItem struct {
	Icon  string
	Label string
	Count int
}

type waveformMark struct {
	Style   string
	Tooltip string
}

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

// eventRowSignals is the JSON payload for event table rows.
//
//nolint:tagliatelle // camelCase required by datastar signal system
type eventRowSignals struct {
	EvtType string `json:"evtType"`
	EvtIdx  int    `json:"evtIdx"`
}

// --- Fragment descriptor ---

type fragmentPatch struct {
	selector string
	html     string
}

// renderAllFragments renders every dashboard section and returns the list of
// (selector, html) pairs to send as datastar-patch-elements events.
//
//nolint:contextcheck,golines // templ components don't take ctx; context passed via renderToString
func renderAllFragments(ctx context.Context, report auditlog.Report, events []auditlog.Event, meta auditlog.TypeMetadata) []fragmentPatch {
	errorCount := countErrors(report.Services)
	legendItems := computeLegendItems(report, meta)
	waveformMarks := computeWaveformMarks(events, meta)
	maxBuildMs, maxShutdownMs := timelineMaxDurations(report.Services)

	return []fragmentPatch{
		{"#stats", renderToString(ctx, statsFragment(buildStatsEntries(report, errorCount)))},
		{"#legend", renderToString(ctx, legendFragment(legendItems))},
		{"#waveform", renderToString(ctx, waveformFragment(waveformMarks))},
		{"#services-tbody", renderToString(ctx, servicesTbody(report, meta))},
		{"#events-tbody", renderToString(ctx, eventsTbody(events, meta))},
		{"#scope-tree-container", renderToString(ctx, scopeTreeFragment(report))},
		{"#graph-container", renderToString(ctx, graphFragment(report))},
		{"#timeline-container", renderToString(ctx, timelineFragment(report, maxBuildMs, maxShutdownMs))},
		{"#footer-stats", renderToString(ctx, footerStatsFragment(report, len(events)))},
		{"#container-id", renderToString(ctx, containerIDFragment(report))},
	}
}

// renderToString renders a templ component to a string. Uses the provided
// context for cancellation.
func renderToString(ctx context.Context, component templ.Component) string {
	var buf strings.Builder

	if err := component.Render(ctx, &buf); err != nil {
		return ""
	}

	return buf.String()
}

// --- Pure-Go helpers used by templ components ---

func countErrors(services []auditlog.ServiceInfo) int {
	errorCount := 0

	for _, svc := range services {
		if svc.Status.IsError() {
			errorCount++
		}
	}

	return errorCount
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

func computeLegendItems(report auditlog.Report, meta auditlog.TypeMetadata) []legendItem {
	counts := map[string]int{}

	for _, svc := range report.Services {
		if svc.ServiceType != "" {
			counts[string(svc.ServiceType)]++
		}
	}

	order := []string{"lazy", "eager", "transient", "alias"}

	var items []legendItem

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

		items = append(items, legendItem{Icon: icon, Label: label, Count: count})
	}

	return items
}

func computeWaveformMarks(events []auditlog.Event, meta auditlog.TypeMetadata) []waveformMark {
	if len(events) == 0 {
		return nil
	}

	minT, maxT, maxDur := waveformBounds(events)

	rangeMs := maxT - minT
	if rangeMs == 0 {
		rangeMs = 1
	}

	marks := make([]waveformMark, 0, len(events))

	for _, evt := range events {
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

		marks = append(marks, waveformMark{
			Style:   fmt.Sprintf("left:%.2f%%;height:%.0fpx;background:%s", pct, height, color),
			Tooltip: waveformTooltip(evt),
		})
	}

	return marks
}

func waveformBounds(events []auditlog.Event) (int64, int64, float64) {
	minTimestamp := events[0].Timestamp.UnixMilli()
	maxTimestamp := minTimestamp
	maxDuration := 1.0

	for _, evt := range events {
		millis := evt.Timestamp.UnixMilli()
		if millis < minTimestamp {
			minTimestamp = millis
		}

		if millis > maxTimestamp {
			maxTimestamp = millis
		}

		if evt.DurationMs != nil && *evt.DurationMs > maxDuration {
			maxDuration = *evt.DurationMs
		}
	}

	return minTimestamp, maxTimestamp, maxDuration
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
		parts = append(parts, string(dep.ServiceName))
	}

	return strings.Join(parts, ", ")
}

func rowSignalsJSON(svc auditlog.ServiceInfo, idx int) string {
	signals, err := json.Marshal(rowSignals{
		RowName:  string(svc.ServiceName),
		RowScope: svc.ScopeName,
		RowIdx:   idx,
	})
	if err != nil {
		return "{}"
	}

	return string(signals)
}

func eventRowSignalsJSON(evt auditlog.Event, idx int) string {
	signals, err := json.Marshal(eventRowSignals{
		EvtType: string(evt.EventType),
		EvtIdx:  idx,
	})
	if err != nil {
		return "{}"
	}

	return string(signals)
}

func eventBadgeColor(evt auditlog.Event, meta auditlog.TypeMetadata) string {
	if evtMeta, ok := meta.Events[string(evt.EventType)]; ok && evtMeta.Color != "" {
		return evtMeta.Color
	}

	return cssVarTextMuted
}

func eventBadgeLabel(evt auditlog.Event, meta auditlog.TypeMetadata) string {
	if evtMeta, ok := meta.Events[string(evt.EventType)]; ok && evtMeta.Label != "" {
		return evtMeta.Label
	}

	return string(evt.EventType)
}

func scopeNodeName(node auditlog.ScopeNode) string {
	name := node.Name
	if name == "" {
		name = string(node.ID)
	}

	if name == "" {
		name = "scope"
	}

	return name
}

func footerVersion(report auditlog.Report) string {
	if report.Version == "" {
		return "?"
	}

	return report.Version
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}

	return s
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

// --- Timeline helpers ---

func timelineMaxDurations(services []auditlog.ServiceInfo) (float64, float64) {
	var maxBuildMs float64
	var maxShutdownMs float64

	for _, svc := range services {
		if svc.FirstBuildDurationMs != nil && *svc.FirstBuildDurationMs > maxBuildMs {
			maxBuildMs = *svc.FirstBuildDurationMs
		}

		if svc.ShutdownDurationMs != nil && *svc.ShutdownDurationMs > maxShutdownMs {
			maxShutdownMs = *svc.ShutdownDurationMs
		}
	}

	if maxBuildMs == 0 {
		maxBuildMs = 1
	}

	if maxShutdownMs == 0 {
		maxShutdownMs = 1
	}

	return maxBuildMs, maxShutdownMs
}

func timelineBarWidth(durationMs *float64, maxMs float64) string {
	if durationMs == nil || *durationMs <= 0 || maxMs <= 0 {
		return "0%"
	}

	pct := *durationMs / maxMs * waveformPctScale

	return fmt.Sprintf("%.1f%%", pct)
}

func safeDuration(durationMs *float64) float64 {
	if durationMs == nil {
		return 0
	}

	return *durationMs
}
