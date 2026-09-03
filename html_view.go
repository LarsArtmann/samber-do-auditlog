package auditlog

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The Go 1.18 branch renders the self-contained HTML report with
// html/template instead of templ (a-h/templ requires Go 1.25). The output
// keeps the warm-amber "Container Telemetry" identity, the five-tab layout,
// search/error filters, keyboard navigation, and a Mermaid rendering of the
// dependency graph — all server-rendered, zero external resources.

// htmlView is the top-level data model for the HTML report template.
type htmlView struct {
	ContainerID   string
	ExportedAt    string
	SchemaVersion string
	DesignTokens  string
	SharedCSS     string
	Stats         []htmlStat
	Services      []htmlService
	Events        []htmlEvent
	Timeline      []htmlTimelineRow
	ScopeTree     *ScopeNode
	Mermaid       string
	EventTypes    []string
	HasHealth     bool
	HealthLabel   string
	HealthClass   string
}

type htmlStat struct {
	Label string
	Value string
	Class string
}

type htmlService struct {
	Icon        string
	TypeIcon    string
	Name        string
	Type        string
	TypeBadge   string
	Scope       string
	Status      string
	StatusIcon  string
	HasError    bool
	ErrorMsg    string
	Order       int
	Invocations int
	BuildMs     string
	ShutdownMs  string
	Deps        string
	Dependents  string
	HealthCell  string
	Times       string
}

type htmlEvent struct {
	Seq       int
	Time      string
	Type      string
	TypeLabel string
	Phase     string
	Scope     string
	Name      string
	Dur       string
	HasError  bool
	ErrorMsg  string
}

type htmlTimelineRow struct {
	Label     string
	BuildPct  string
	BuildTip  string
	ShutPct   string
	ShutTip   string
	Durations string
}

// htmlEventTypes are the filter chips rendered for the events tab.
var htmlEventTypes = []string{ //nolint:gochecknoglobals // read-only filter list
	"all",
	string(EventTypeRegistration),
	string(EventTypeInvocation),
	string(EventTypeShutdown),
	string(EventTypeHealthCheck),
}

// ExportToHTML writes a self-contained HTML visualization to a file.
// Implemented in html.go together with WriteHTML/WriteHTMLString and the
// report template.

// buildHTMLView converts a Report into the template-facing view model,
// precomputing every formatted string so the template stays logic-free.
func (r Report) buildHTMLView() (htmlView, error) {
	mermaid, err := r.WriteMermaidString()
	if err != nil {
		return htmlView{}, fmt.Errorf("render dependency graph: %w", err)
	}

	view := htmlView{
		ContainerID:   string(r.ContainerID),
		ExportedAt:    r.ExportedAt.Format(time.RFC3339),
		SchemaVersion: SchemaVersion,
		DesignTokens:  DesignTokensCSS,
		SharedCSS:     SharedComponentCSS,
		ScopeTree:     &r.ScopeTree,
		Mermaid:       mermaid,
		EventTypes:    htmlEventTypes,
	}

	unhealthy := len(r.UnhealthyServices())

	errorCount := 0
	depCount := 0

	for i := range r.Services {
		svc := &r.Services[i]

		if svc.Status.IsError() {
			errorCount++
		}

		depCount += len(svc.Dependencies)

		view.Services = append(view.Services, buildHTMLService(svc))
	}

	view.Events = buildHTMLEvents(r.Events)
	view.Stats = buildHTMLStats(r, len(view.Services), depCount, errorCount, unhealthy)
	view.Timeline = buildHTMLTimeline(r.Services)
	view.HasHealth = r.HealthCheckedCount > 0
	view.HealthLabel = strconv.Itoa(r.HealthCheckedCount) + " checked"
	view.HealthClass = "success"

	if r.HealthCheckedCount > 0 && !r.HealthCheckSucceeded {
		view.HealthLabel += ", " + strconv.Itoa(unhealthy) + " unhealthy"
		view.HealthClass = "error"
	}

	return view, nil
}

// buildHTMLService projects one ServiceInfo into its template row model.
func buildHTMLService(svc *ServiceInfo) htmlService {
	deps := make([]string, 0, len(svc.Dependencies))
	for _, dep := range svc.Dependencies {
		deps = append(deps, string(dep.ServiceName))
	}

	dependents := make([]string, 0, len(svc.Dependents))
	for _, dep := range svc.Dependents {
		dependents = append(dependents, string(dep.ServiceName))
	}

	row := htmlService{
		Icon:        svc.ServiceType.Icon(),
		TypeIcon:    svc.ServiceType.Icon(),
		Name:        string(svc.ServiceName),
		Type:        string(svc.ServiceType),
		TypeBadge:   svc.ServiceType.Label(),
		Scope:       svc.ScopeName,
		Status:      string(svc.Status),
		StatusIcon:  svc.Status.Icon(),
		HasError:    svc.Status.IsError(),
		Order:       svc.InvocationOrder,
		Invocations: svc.InvocationCount,
		BuildMs:     formatMsPtr(svc.FirstBuildDurationMs),
		ShutdownMs:  formatMsPtr(svc.ShutdownDurationMs),
		Deps:        joinOrDash(deps),
		Dependents:  joinOrDash(dependents),
		Times:       formatServiceTimes(svc),
	}

	if svc.InvocationError != nil {
		row.ErrorMsg = *svc.InvocationError
	} else if svc.ShutdownError != nil {
		row.ErrorMsg = *svc.ShutdownError
	}

	switch {
	case svc.HealthCheckCount == 0:
		row.HealthCell = ""
	case svc.HealthCheckError != nil:
		row.HealthCell = "unhealthy: " + *svc.HealthCheckError
	default:
		row.HealthCell = "healthy"
	}

	return row
}

// buildHTMLEvents projects the report's event stream into template rows.
func buildHTMLEvents(events []Event) []htmlEvent {
	out := make([]htmlEvent, 0, len(events))

	for _, evt := range events {
		dur := ""

		if evt.DurationMs != nil {
			dur = strconv.FormatFloat(*evt.DurationMs, 'f', 3, 64) + "ms"
		}

		row := htmlEvent{
			Seq:       evt.Sequence,
			Time:      evt.Timestamp.Format("15:04:05.000"),
			Type:      string(evt.EventType),
			TypeLabel: evt.EventType.Label(),
			Phase:     "\u25BE",
			Scope:     evt.ScopeName,
			Name:      string(evt.ServiceName),
			Dur:       dur,
			HasError:  evt.Error != nil,
			ErrorMsg:  derefString(evt.Error),
		}

		if evt.Phase == PhaseBefore {
			row.Phase = "\u25B4"
		}

		out = append(out, row)
	}

	return out
}

// buildHTMLStats assembles the stat cards shown under the header.
func buildHTMLStats(r Report, serviceCount, depCount, errorCount, unhealthy int) []htmlStat {
	stats := []htmlStat{
		{Label: "Services", Value: strconv.Itoa(serviceCount)},
		{Label: "Scopes", Value: strconv.Itoa(r.ScopeCount)},
		{Label: "Events", Value: strconv.Itoa(r.EventCount)},
		{Label: "Dependencies", Value: strconv.Itoa(depCount)},
		{Label: "Total Build", Value: strconv.FormatFloat(r.TotalBuildDurationMs, 'f', 2, 64) + "ms"},
		{
			Label: "Errors",
			Value: strconv.Itoa(errorCount),
			Class: boolClass(errorCount > 0, "error", "success"),
		},
	}

	if r.HealthCheckedCount > 0 {
		health := htmlStat{
			Label: "Health Checks",
			Value: strconv.Itoa(r.HealthCheckedCount) + " checked",
			Class: "success",
		}

		if unhealthy > 0 {
			health.Value += ", " + strconv.Itoa(unhealthy) + " unhealthy"
			health.Class = "error"
		}

		stats = append(stats, health)
	}

	return stats
}

// buildHTMLTimeline projects services with timing data into timeline bars,
// scaled against the largest duration in the report.
func buildHTMLTimeline(services []ServiceInfo) []htmlTimelineRow {
	var maxMs float64

	for i := range services {
		svc := &services[i]
		if svc.FirstBuildDurationMs != nil && *svc.FirstBuildDurationMs > maxMs {
			maxMs = *svc.FirstBuildDurationMs
		}

		if svc.ShutdownDurationMs != nil && *svc.ShutdownDurationMs > maxMs {
			maxMs = *svc.ShutdownDurationMs
		}
	}

	rows := make([]htmlTimelineRow, 0, len(services))

	for i := range services {
		svc := &services[i]
		if svc.FirstBuildDurationMs == nil && svc.ShutdownDurationMs == nil {
			continue
		}

		rows = append(rows, htmlTimelineRow{
			Label:     strconv.Itoa(svc.InvocationOrder) + ". " + svc.ServiceType.Icon() + " " + string(svc.ServiceName),
			BuildPct:  percentOf(svc.FirstBuildDurationMs, maxMs),
			BuildTip:  "Build: " + formatMsPtr(svc.FirstBuildDurationMs) + "ms",
			ShutPct:   percentOf(svc.ShutdownDurationMs, maxMs),
			ShutTip:   "Shutdown: " + formatMsPtr(svc.ShutdownDurationMs) + "ms",
			Durations: formatTimelineDurations(svc.FirstBuildDurationMs, svc.ShutdownDurationMs),
		})
	}

	return rows
}

// --- small formatting helpers ---

func derefString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func formatMsPtr(ms *float64) string {
	if ms == nil {
		return ""
	}

	return strconv.FormatFloat(*ms, 'f', 3, 64)
}

func percentOf(ms *float64, maxMs float64) string {
	if ms == nil || maxMs <= 0 {
		return "0"
	}

	return strconv.FormatFloat(*ms/maxMs*100, 'f', 1, 64)
}

func joinOrDash(items []string) string {
	if len(items) == 0 {
		return "–"
	}

	return strings.Join(items, ", ")
}

func boolClass(cond bool, truthy, falsy string) string {
	if cond {
		return truthy
	}

	return falsy
}

func formatServiceTimes(svc *ServiceInfo) string {
	parts := []string{"Registered: " + svc.RegisteredAt.Format(time.RFC3339)}

	if svc.FirstInvokedAt != nil {
		parts = append(parts, "Invoked: "+svc.FirstInvokedAt.Format(time.RFC3339))
	}

	return strings.Join(parts, "\n")
}

func formatTimelineDurations(buildMs, shutdownMs *float64) string {
	parts := make([]string, 0, 2)

	if buildMs != nil {
		parts = append(parts, strconv.FormatFloat(*buildMs, 'f', 2, 64)+"ms")
	}

	if shutdownMs != nil {
		parts = append(parts, strconv.FormatFloat(*shutdownMs, 'f', 2, 64)+"ms")
	}

	return strings.Join(parts, " / ")
}
