package live

import (
	"encoding/json/v2"
	"fmt"
	"math"
	"strconv"
	"strings"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

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
// so the templ template can always render a valid signal struct.
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
