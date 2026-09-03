package live

import (
	"encoding/json"
	"fmt"
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

	msPerSecond  = 1000.0
	secPerMinute = 60.0
	minPerHour   = 60.0

	waveformMinHeight = 4.0
	waveformMaxHeight = 28.0
	waveformPctScale  = 100.0

	cssVarTextMuted = "var(--text-muted)"
	mdash           = "&mdash;"
	cssClassSuccess = "success"
	cssClassError   = "error"
)

// servicesShowExpr is the data-show expression for each service row.
//
//nolint:gochecknoglobals // effectively immutable; interpolates a const
var servicesShowExpr = "(!$serviceSearch || $rowName.toLowerCase().includes($serviceSearch.toLowerCase()) || $rowScope.toLowerCase().includes($serviceSearch.toLowerCase())) && ($showAllServices || $rowIdx < " + strconv.Itoa(
	maxServiceRows,
) + ")"

// eventsShowExpr is the data-show expression for each event row.
//
//nolint:gochecknoglobals // effectively immutable; interpolates a const
var eventsShowExpr = "(!$eventFilter || $evtType === $eventFilter) && ($showAllEvents || $evtIdx < " + strconv.Itoa(
	maxEventRows,
) + ")" //nolint:lll // single expression

// --- Display metadata (plain data for html/template) ---

type statsEntry struct {
	Label string
	Value string
	Class string
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

// serviceRow is one precomputed <tr> for the services table.
type serviceRow struct {
	Signals     string
	ShowExpr    string
	TypeIcon    string
	Name        string
	Scope       string
	Type        string
	StatusIcon  string
	Status      string
	Invocations int
	Build       string
	Deps        string
	HasError    bool
	ErrTitle    string
	ErrMsg      string
}

// eventRow is one precomputed <tr> for the events table.
type eventRow struct {
	Signals   string
	ShowExpr  string
	Num       int
	Time      string
	Color     string
	Label     string
	Phase     string
	PhaseUp   bool
	Name      string
	Duration  string
	HasError  bool
	ErrTitle  string
	ErrMsg    string
}

// scopeNodeView is one precomputed node of the scope tree.
type scopeNodeView struct {
	Name     string
	Indent   string
	Services []string
	Children []scopeNodeView
}

type graphNodeView struct {
	Name    string
	Type    string
	HasType bool
	Deps    []string
}

type timelineRowView struct {
	Name     string
	BuildW   string
	BuildTip string
	ShutW    string
	ShutTip  string
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
func renderAllFragments(
	report auditlog.Report,
	events []auditlog.Event,
	meta auditlog.TypeMetadata,
) []fragmentPatch {
	errorCount := countErrors(report.Services)

	return []fragmentPatch{
		{"#stats", renderFragment("statsFragment", statsFragmentData{Entries: buildStatsEntries(report, errorCount)})},
		{"#legend", renderFragment("legendFragment", legendFragmentData{Items: computeLegendItems(report, meta)})},
		{"#waveform", renderFragment("waveformFragment", waveformFragmentData{Marks: computeWaveformMarks(events, meta)})},
		{"#services-tbody", renderFragment("servicesTbody", servicesTbodyData{Rows: buildServiceRows(report.Services, meta)})},
		{"#events-tbody", renderFragment("eventsTbody", eventsTbodyData{Rows: buildEventRows(events, meta)})},
		{"#scope-tree-container", renderFragment("scopeTreeFragment", scopeTreeFragmentData{Root: buildScopeTreeViews(report.ScopeTree)})},
		{"#graph-container", renderFragment("graphFragment", graphFragmentData{Nodes: buildGraphNodeViews(report.Services)})},
		{"#timeline-container", renderFragment("timelineFragment", timelineFragmentData{Rows: buildTimelineRows(report.Services)})},
		{"#footer-stats", renderFragment("footerStatsFragment", footerStatsData{
			Version:      footerVersion(report),
			EventCount:   len(events),
			ServiceCount: report.ServiceCount,
		})},
		{"#container-id", renderFragment("containerIDFragment", containerIDData{Value: string(report.ContainerID)})},
	}
}

// --- View builders (pure functions from report/events to display rows) ---

type statsFragmentData struct {
	Entries []statsEntry
}

type legendFragmentData struct {
	Items []legendItem
}

type waveformFragmentData struct {
	Marks []waveformMark
}

type servicesTbodyData struct {
	Rows []serviceRow
}

type eventsTbodyData struct {
	Rows []eventRow
}

type scopeTreeFragmentData struct {
	Root *scopeNodeView
}

type graphFragmentData struct {
	Nodes []graphNodeView
}

type timelineFragmentData struct {
	Rows []timelineRowView
}

type footerStatsData struct {
	Version      string
	EventCount   int
	ServiceCount int
}

type containerIDData struct {
	Value string
}

func buildServiceRows(services []auditlog.ServiceInfo, meta auditlog.TypeMetadata) []serviceRow {
	rows := make([]serviceRow, 0, len(services))

	for idx, svc := range services {
		row := serviceRow{
			Signals:     rowSignalsJSON(svc, idx),
			ShowExpr:    servicesShowExpr,
			TypeIcon:    strings.TrimSpace(providerIcon(meta, string(svc.ServiceType)) + " "),
			Name:        string(svc.ServiceName),
			Scope:       svc.ScopeName,
			Type:        string(svc.ServiceType),
			StatusIcon:  statusIcon(meta, string(svc.Status)),
			Status:      string(svc.Status),
			Invocations: svc.InvocationCount,
			Build:       mdash,
			Deps:        depNamesString(svc.Dependencies),
		}

		if svc.FirstBuildDurationMs != nil {
			row.Build = humanizeDuration(*svc.FirstBuildDurationMs)
		}

		if svc.InvocationError != nil {
			row.HasError = true
			row.ErrTitle = *svc.InvocationError
			row.ErrMsg = truncateString(*svc.InvocationError, maxServiceErrLen)
		} else if svc.ShutdownError != nil {
			row.HasError = true
			row.ErrTitle = *svc.ShutdownError
			row.ErrMsg = truncateString(*svc.ShutdownError, maxServiceErrLen)
		}

		rows = append(rows, row)
	}

	return rows
}

func buildEventRows(events []auditlog.Event, meta auditlog.TypeMetadata) []eventRow {
	rows := make([]eventRow, 0, len(events))

	for idx, evt := range events {
		row := eventRow{
			Signals:   eventRowSignalsJSON(evt, idx),
			ShowExpr:  eventsShowExpr,
			Num:       idx + 1,
			Time:      evt.Timestamp.Format("15:04:05"),
			Color:     eventBadgeColor(evt, meta),
			Label:     eventBadgeLabel(evt, meta),
			PhaseUp:   evt.Phase == auditlog.PhaseBefore,
			Phase:     string(evt.Phase),
			Name:      string(evt.ServiceName),
			Duration:  mdash,
		}

		if evt.DurationMs != nil {
			row.Duration = humanizeDuration(*evt.DurationMs)
		}

		if evt.Error != nil {
			row.HasError = true
			row.ErrTitle = *evt.Error
			row.ErrMsg = truncateString(*evt.Error, maxEventErrLen)
		}

		rows = append(rows, row)
	}

	return rows
}

func buildScopeTreeViews(root auditlog.ScopeNode) *scopeNodeView {
	if root.ID == "" && root.Name == "" && len(root.Children) == 0 {
		return nil
	}

	view := buildScopeNodeView(root, 0)

	return &view
}

func buildScopeNodeView(node auditlog.ScopeNode, depth int) scopeNodeView {
	view := scopeNodeView{
		Name:   scopeNodeName(node),
		Indent: fmt.Sprintf("margin-left:%dpx", depth*scopeIndentPx),
	}

	for _, svc := range node.Services {
		view.Services = append(view.Services, string(svc))
	}

	for _, child := range node.Children {
		view.Children = append(view.Children, buildScopeNodeView(child, depth+1))
	}

	return view
}

func buildGraphNodeViews(services []auditlog.ServiceInfo) []graphNodeView {
	nodes := make([]graphNodeView, 0, len(services))

	for _, svc := range services {
		node := graphNodeView{
			Name:    string(svc.ServiceName),
			Type:    string(svc.ServiceType),
			HasType: svc.ServiceType != "",
			Deps:    make([]string, 0, len(svc.Dependencies)),
		}

		for _, dep := range svc.Dependencies {
			node.Deps = append(node.Deps, string(dep.ServiceName))
		}

		nodes = append(nodes, node)
	}

	return nodes
}

func buildTimelineRows(services []auditlog.ServiceInfo) []timelineRowView {
	maxBuildMs, maxShutdownMs := timelineMaxDurations(services)

	rows := make([]timelineRowView, 0, len(services))

	for _, svc := range services {
		rows = append(rows, timelineRowView{
			Name:     string(svc.ServiceName),
			BuildW:   timelineBarWidth(svc.FirstBuildDurationMs, maxBuildMs),
			BuildTip: "Build: " + humanizeDuration(safeDuration(svc.FirstBuildDurationMs)),
			ShutW:    timelineBarWidth(svc.ShutdownDurationMs, maxShutdownMs),
			ShutTip:  "Shutdown: " + humanizeDuration(safeDuration(svc.ShutdownDurationMs)),
		})
	}

	return rows
}

// --- Pure-Go helpers used by the view builders ---

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
		{Label: "Services", Value: strconv.Itoa(report.ServiceCount)},
		{Label: "Events", Value: strconv.Itoa(report.EventCount)},
		{Label: "Scopes", Value: strconv.Itoa(report.ScopeCount)},
		{Label: "Errors", Value: strconv.Itoa(errorCount), Class: errorCountClass(errorCount)},
		{Label: "Build (ms)", Value: humanizeDuration(report.TotalBuildDurationMs)},
		{Label: "Shutdown (ms)", Value: humanizeDuration(report.TotalShutdownDurationMs)},
	}

	if report.HealthCheckedCount > 0 {
		cls := cssClassSuccess
		if !report.HealthCheckSucceeded {
			cls = cssClassError
		}

		stats = append(stats, statsEntry{Label: "Health", Value: healthLabel(report.HealthCheckSucceeded), Class: cls})
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
	return marshalSignalsOrEmpty(rowSignals{
		RowName:  string(svc.ServiceName),
		RowScope: svc.ScopeName,
		RowIdx:   idx,
	})
}

func eventRowSignalsJSON(evt auditlog.Event, idx int) string {
	return marshalSignalsOrEmpty(eventRowSignals{
		EvtType: string(evt.EventType),
		EvtIdx:  idx,
	})
}

// marshalSignalsOrEmpty marshals v to JSON and returns the result. On error
// (which can only happen if v contains unsupported types), returns "{}"
// so the template can always render a valid signal struct.
func marshalSignalsOrEmpty(v any) string {
	signals, err := json.Marshal(v)
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
		return cssClassError
	}

	return cssClassSuccess
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
