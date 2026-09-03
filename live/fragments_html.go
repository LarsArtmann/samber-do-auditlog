package live

import (
	"html/template"
	"strings"
)

// fragmentTemplates parses all live-dashboard fragment templates once.
// The templates render precomputed view structs (see fragments.go); every
// dynamic value is HTML-escaped by html/template, matching the escaping the
// previous templ-based implementation provided.
//
// Element IDs (#stats, #legend, #waveform, #services-tbody, #events-tbody,
// #scope-tree-container, #graph-container, #timeline-container, #footer-stats,
// #container-id) and the datastar attributes (data-signals, data-show) are
// the wire contract with dashboard.js + datastar.js and MUST NOT change.
var fragmentTemplates = template.Must(template.New("fragments").Parse(fragmentTemplateSource))

// renderFragment executes the named fragment template into a string.
// On a template error it returns "" — a broken fragment must not kill the
// SSE snapshot loop; template errors are caught by the fragment tests.
func renderFragment(name string, data any) string {
	var buf strings.Builder

	if err := fragmentTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return ""
	}

	return buf.String()
}

// fragmentTemplateSource holds the html/template sources for every dashboard
// fragment. The markup mirrors the previous fragments.templ templates
// element-for-element: same tag structure, same classes, same entities.
const fragmentTemplateSource = `
{{define "statsFragment"}}{{range .Entries}}<div class="stat-card {{.Class}}"><div class="label">{{.Label}}</div><div class="value">{{.Value}}</div></div>{{end}}{{end}}

{{define "legendFragment"}}{{range .Items}}<div class="legend-item"><span class="icon">{{.Icon}}</span> {{.Label}} <span style="opacity:0.5">({{.Count}})</span></div>{{end}}{{end}}

{{define "waveformFragment"}}{{if .Marks}}{{range .Marks}}<div class="wf-event" style="{{.Style}}" title="{{.Tooltip}}"></div>{{end}}{{else}}<span class="waveform-placeholder">Waiting for events...</span>{{end}}{{end}}

{{define "servicesTbody"}}{{if .Rows}}{{range .Rows}}<tr data-signals="{{.Signals}}" data-show="{{.ShowExpr}}"><td>{{.TypeIcon}}{{.Name}}</td><td>{{.Scope}}</td><td>{{.Type}}</td><td>{{.StatusIcon}} {{.Status}}</td><td>{{.Invocations}}</td><td>{{.Build}}</td><td>{{.Deps}}</td><td>{{if .HasError}}<span title="{{.ErrTitle}}">&#9888; {{.ErrMsg}}</span>{{else}}&mdash;{{end}}</td></tr>{{end}}{{else}}<tr class="empty-state"><td colspan="8">No services registered yet.</td></tr>{{end}}{{end}}

{{define "eventsTbody"}}{{if .Rows}}{{range .Rows}}<tr data-signals="{{.Signals}}" data-show="{{.ShowExpr}}"><td>{{.Num}}</td><td>{{.Time}}</td><td><span class="event-badge" style="background:{{.Color}}">{{.Label}}</span></td><td>{{if .PhaseUp}}&#9652;{{else}}&#9662;{{end}} {{.Phase}}</td><td>{{.Name}}</td><td>{{.Duration}}</td><td>{{if .HasError}}<span title="{{.ErrTitle}}">&#9888; {{.ErrMsg}}</span>{{end}}</td></tr>{{end}}{{else}}<tr class="empty-state"><td colspan="7">No events recorded yet.</td></tr>{{end}}{{end}}

{{define "scopeTreeFragment"}}{{if .Root}}{{template "scopeNode" .Root}}{{else}}<div class="graph-placeholder">Scope tree will appear here once services register...</div>{{end}}{{end}}

{{define "scopeNode"}}<div class="scope-node" style="{{.Indent}}"><div class="scope-label" role="button" tabindex="0" aria-expanded="true"><span class="scope-icon" aria-hidden="true">&#9660;</span> {{.Name}}</div><div class="scope-body">{{if .Services}}<div class="scope-services">{{range .Services}}<span class="scope-service-chip">{{.}}</span>{{end}}</div>{{end}}{{if .Children}}<div class="scope-children">{{range .Children}}{{template "scopeNode" .}}{{end}}</div>{{end}}</div></div>{{end}}

{{define "graphFragment"}}{{if .Nodes}}<div class="dep-graph">{{range .Nodes}}<div class="dep-node"><div class="dep-node-header"><span class="dep-node-name">{{.Name}}</span>{{if .HasType}}<span class="dep-node-type">{{.Type}}</span>{{end}}</div>{{if .Deps}}<div class="dep-node-deps">{{range .Deps}}<span class="dep-arrow">{{.}}</span>{{end}}</div>{{end}}</div>{{end}}</div>{{else}}<div class="graph-placeholder" id="graph-placeholder">Dependency graph will appear here as services register...</div>{{end}}{{end}}

{{define "timelineFragment"}}{{if .Rows}}<div class="timeline">{{range .Rows}}<div class="timeline-row"><div class="timeline-label">{{.Name}}</div><div class="timeline-bars"><div class="timeline-bar build" style="width:{{.BuildW}}" title="{{.BuildTip}}"></div><div class="timeline-bar shutdown" style="width:{{.ShutW}}" title="{{.ShutTip}}"></div></div></div>{{end}}</div>{{else}}<div class="graph-placeholder">Timeline will appear here as events arrive...</div>{{end}}{{end}}

{{define "footerStatsFragment"}}Schema v{{.Version}} | {{.EventCount}} events | {{.ServiceCount}} services{{end}}

{{define "containerIDFragment"}}{{if .Value}}{{.Value}}{{else}}&mdash;{{end}}{{end}}
`
