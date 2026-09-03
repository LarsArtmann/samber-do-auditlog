package auditlog

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
)

// The Go 1.23 branch renders the self-contained HTML report with
// html/template instead of templ (a-h/templ requires Go 1.25). The output
// keeps the warm-amber "Container Telemetry" identity, the five-tab layout,
// sortable/filterable tables, keyboard navigation, and a Mermaid rendering of
// the dependency graph — all server-rendered, zero external resources.

// htmlReportFuncs provides the template.CSS wrapper so embedded stylesheets
// are not HTML-escaped.
var htmlReportFuncs = template.FuncMap{ //nolint:gochecknoglobals // immutable func map
	"css": func(s string) template.CSS { return template.CSS(s) },
}

// ExportToHTML writes a self-contained HTML visualization to a file.
func (p *Plugin) ExportToHTML(path string) error {
	return writeToFile(path, p.WriteHTML)
}

// WriteHTML writes a self-contained HTML visualization to writer.
func (p *Plugin) WriteHTML(writer io.Writer) error {
	return p.Report().WriteHTML(writer)
}

// WriteHTML renders a self-contained HTML visualization of the report to
// writer. This enables offline report rendering from a loaded Report (e.g.
// via LoadReport) without a live Plugin/container.
func (r Report) WriteHTML(writer io.Writer) error {
	view, err := r.buildHTMLView()
	if err != nil {
		return fmt.Errorf("render HTML report: %w", err)
	}

	if err := htmlReportTemplate.Execute(writer, view); err != nil {
		return fmt.Errorf("render HTML report: %w", err)
	}

	return nil
}

// WriteHTMLString returns the HTML dashboard as a string. It is a convenience
// wrapper around [Report.WriteHTML] for use in tests, CLI output, and any
// context where a string is preferred over an [io.Writer].
func (r Report) WriteHTMLString() (string, error) {
	var buf bytes.Buffer

	if err := r.WriteHTML(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// htmlReportTemplate renders the complete self-contained report. Every string
// is data-bound through html/template's contextual auto-escaping; only the
// two audited CSS constants pass through the css() func.
//
//nolint:gochecknoglobals // parsed once, immutable
var htmlReportTemplate = template.Must(template.New("report").Funcs(htmlReportFuncs).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'">
<title>do-auditlog — {{.ContainerID}}</title>
<style>
{{css .DesignTokens}}
* { margin: 0; padding: 0; box-sizing: border-box; }
body { background: var(--bg); color: var(--text); font-family: 'Space Grotesk', system-ui, -apple-system, sans-serif;
  font-size: 15px; line-height: 1.5; padding: 2rem; }
.mono { font-family: 'IBM Plex Mono', ui-monospace, monospace; }
header { margin-bottom: 1.5rem; }
h1 { font-size: 1.5rem; color: var(--accent); letter-spacing: 0.02em; }
.meta { color: var(--text-muted); font-size: 0.85rem; margin-top: 0.25rem; }
.stats { display: flex; flex-wrap: wrap; gap: 0.75rem; margin: 1.25rem 0; }
.stat-card { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: var(--radius);
  padding: 0.75rem 1.1rem; min-width: 120px; border-left: 3px solid transparent; }
.stat-card:hover { border-left-color: var(--accent); }
.stat-card .label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-muted); }
.stat-card .value { font-size: 1.15rem; font-weight: 600; margin-top: 0.15rem; }
.stat-card.success .value { color: var(--success); }
.stat-card.error .value { color: var(--error); }
.tabs { display: flex; gap: 0.25rem; border-bottom: 1px solid var(--border); margin-bottom: 1rem; }
.tab { background: none; border: none; color: var(--text-muted); padding: 0.6rem 1rem; cursor: pointer;
  font: inherit; border-bottom: 2px solid transparent; }
.tab:hover { color: var(--text); }
.tab.active { color: var(--accent); border-bottom-color: var(--accent); }
.tab-content { display: none; }
.tab-content.active { display: block; }
.toolbar { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 0.75rem; align-items: center; }
.toolbar input { background: var(--surface); border: 1px solid var(--border); color: var(--text);
  border-radius: var(--radius); padding: 0.45rem 0.75rem; min-width: 220px; font: inherit; }
.toolbar input:focus { outline: none; border-color: var(--border-active); }
.chip { background: var(--surface); border: 1px solid var(--border); color: var(--text-muted);
  border-radius: 999px; padding: 0.3rem 0.8rem; cursor: pointer; font-size: 0.8rem; }
.chip:hover { color: var(--text); border-color: var(--border-active); }
.chip.active { color: var(--bg); background: var(--accent); border-color: var(--accent); font-weight: 600; }
.result-count { color: var(--text-muted); font-size: 0.8rem; }
.table-wrap { overflow-x: auto; background: var(--bg-elevated); border: 1px solid var(--border); border-radius: var(--radius); }
table { width: 100%; border-collapse: collapse; font-size: 0.85rem; }
th, td { text-align: left; padding: 0.55rem 0.8rem; border-bottom: 1px solid var(--border); white-space: nowrap; }
th { color: var(--text-muted); font-weight: 600; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.07em; }
th.sortable { cursor: pointer; user-select: none; }
th.sortable:focus { outline: 2px solid var(--accent); outline-offset: -2px; color: var(--text); }
th.sort-asc::after { content: ' ▴'; color: var(--accent); }
th.sort-desc::after { content: ' ▾'; color: var(--accent); }
tr:last-child td { border-bottom: none; }
tbody tr:hover { background: var(--surface); }
.type-badge, .status-badge, .event-badge { display: inline-block; border-radius: 999px; padding: 0.1rem 0.55rem;
  font-size: 0.72rem; border: 1px solid var(--border); }
.type-badge.lazy { color: var(--lazy); border-color: var(--lazy); }
.type-badge.eager { color: var(--eager); border-color: var(--eager); }
.type-badge.transient { color: var(--transient); border-color: var(--transient); }
.type-badge.alias { color: var(--alias); border-color: var(--alias); }
.status-badge.active { color: var(--success); border-color: var(--success); }
.status-badge.shutdown { color: var(--info); border-color: var(--info); }
.status-badge.registered { color: var(--text-muted); border-color: var(--border-active); }
.status-badge.invocation_error, .status-badge.shutdown_error { color: var(--error); border-color: var(--error); cursor: help; }
.event-badge.registration { color: var(--info); border-color: var(--info); }
.event-badge.invocation { color: var(--accent); border-color: var(--accent); }
.event-badge.shutdown { color: var(--lazy); border-color: var(--lazy); }
.event-badge.health_check { color: var(--success); border-color: var(--success); }
tr.has-error td { color: var(--error); }
.scope-tree { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: var(--radius); padding: 1rem; }
.scope-tree ul { list-style: none; }
.scope-tree ul ul { padding-left: 1.25rem; border-left: 1px solid var(--border); }
.scope-tree li { padding: 0.25rem 0; }
.scope-name { color: var(--accent); font-weight: 600; }
.scope-count { color: var(--text-dim); font-size: 0.78rem; margin-left: 0.5rem; }
.scope-svc { display: inline-block; margin: 0.1rem 0.5rem 0.1rem 0; color: var(--text); }
.mermaid-box { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: var(--radius);
  padding: 1rem; overflow-x: auto; white-space: pre; color: var(--text); }
.mermaid-hint { color: var(--text-dim); font-size: 0.78rem; margin-top: 0.5rem; }
.timeline-row { display: flex; align-items: center; gap: 0.75rem; padding: 0.3rem 0; }
.timeline-label { width: 240px; min-width: 240px; font-size: 0.8rem; overflow: hidden; text-overflow: ellipsis; }
.timeline-track { flex: 1; display: flex; height: 14px; background: var(--bg-elevated); border-radius: 4px; overflow: hidden; }
.timeline-bar { height: 100%; }
.timeline-bar.build { background: var(--accent); }
.timeline-bar.shutdown { background: var(--warning); }
.timeline-dur { width: 140px; text-align: right; font-size: 0.75rem; color: var(--text-muted); }
.empty { color: var(--text-dim); padding: 1.5rem; text-align: center; }
footer { margin-top: 2rem; color: var(--text-dim); font-size: 0.78rem; display: flex; justify-content: space-between; }
{{css .SharedCSS}}
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation: none !important; transition: none !important; }
}
@media (max-width: 720px) {
  body { padding: 1rem; }
  .timeline-label { width: 120px; min-width: 120px; }
}
</style>
</head>
<body>
<a class="skip-link" href="#main-content">Skip to main content</a>
<header>
  <h1>Container Telemetry — {{.ContainerID}}</h1>
  <div class="meta mono">exported {{.ExportedAt}} · schema v{{.SchemaVersion}}</div>
</header>
<section class="stats" id="stats">
  {{range .Stats}}<div class="stat-card {{.Class}}"><div class="label">{{.Label}}</div><div class="value">{{.Value}}</div></div>
  {{end}}
</section>
<nav class="tabs" role="tablist" aria-label="Report sections">
  <button class="tab active" role="tab" aria-selected="true" aria-controls="tab-services" data-tab="services" id="tabbtn-services" tabindex="0">Services</button>
  <button class="tab" role="tab" aria-selected="false" aria-controls="tab-scopes" data-tab="scopes" tabindex="-1">Scopes</button>
  <button class="tab" role="tab" aria-selected="false" aria-controls="tab-graph" data-tab="graph" tabindex="-1">Graph</button>
  <button class="tab" role="tab" aria-selected="false" aria-controls="tab-timeline" data-tab="timeline" tabindex="-1">Timeline</button>
  <button class="tab" role="tab" aria-selected="false" aria-controls="tab-events" data-tab="events" tabindex="-1">Events</button>
</nav>
<main id="main-content" tabindex="-1">

<section class="tab-content active" id="tab-services" role="tabpanel">
  <div class="toolbar">
    <input type="search" id="service-search" placeholder="Filter services…" aria-label="Filter services">
    <button class="chip" id="svc-errors-only" aria-pressed="false">errors only</button>
    <span class="result-count" id="svc-result-count"></span>
  </div>
  <div class="table-wrap">
  <table>
    <thead><tr>
      <th class="sortable" data-sort="name" tabindex="0" aria-sort="none">Service</th>
      <th class="sortable" data-sort="type" tabindex="0" aria-sort="none">Type</th>
      <th class="sortable" data-sort="scope" tabindex="0" aria-sort="none">Scope</th>
      <th class="sortable" data-sort="status" tabindex="0" aria-sort="none">Status</th>
      <th class="sortable" data-sort="order" tabindex="0" aria-sort="none">Order</th>
      <th class="sortable" data-sort="invocations" tabindex="0" aria-sort="none">Invocations</th>
      <th class="sortable" data-sort="build" tabindex="0" aria-sort="none">Build (ms)</th>
      <th class="sortable" data-sort="shutdown" tabindex="0" aria-sort="none">Shutdown (ms)</th>
      <th>Depends on</th><th>Dependents</th><th>Health</th>
    </tr></thead>
    <tbody id="services-tbody">
    {{if .Services}}{{range .Services}}<tr data-search="{{.Name}} {{.Scope}} {{.Type}}" data-has-error="{{if .HasError}}1{{else}}0{{end}}"
      data-sort-name="{{.Name}}" data-sort-type="{{.Type}}" data-sort-scope="{{.Scope}}" data-sort-status="{{.Status}}"
      data-sort-order="{{.Order}}" data-sort-invocations="{{.Invocations}}" data-sort-build="{{.BuildMs}}" data-sort-shutdown="{{.ShutdownMs}}">
      <td class="mono" title="{{.Times}}">{{if .Icon}}{{.Icon}} {{end}}{{.Name}}</td>
      <td>{{if .Type}}<span class="type-badge {{.Type}}" title="{{.TypeBadge}}">{{.TypeBadge}}</span>{{end}}</td>
      <td>{{.Scope}}</td>
      <td><span class="status-badge {{.Status}}" {{if .ErrorMsg}}data-error="{{.ErrorMsg}}"{{end}}>{{.StatusIcon}} {{.Status}}</span></td>
      <td class="mono">{{.Order}}</td>
      <td class="mono">{{.Invocations}}</td>
      <td class="mono">{{.BuildMs}}</td>
      <td class="mono">{{.ShutdownMs}}</td>
      <td>{{.Deps}}</td>
      <td>{{.Dependents}}</td>
      <td>{{.HealthCell}}</td>
    </tr>
    {{end}}{{else}}<tr><td colspan="11" class="empty">No services recorded</td></tr>{{end}}
    </tbody>
  </table>
  </div>
</section>

<section class="tab-content" id="tab-scopes" role="tabpanel">
  <div class="scope-tree">
    {{if .ScopeTree}}{{template "scopeNode" .ScopeTree}}{{else}}<div class="empty">No scopes recorded</div>{{end}}
  </div>
</section>

<section class="tab-content" id="tab-graph" role="tabpanel">
  <div class="mermaid-box mono" id="mermaid-src">{{.Mermaid}}</div>
  <div class="mermaid-hint">Mermaid flowchart of the dependency graph (dependent → dependency). Paste into any Mermaid renderer.</div>
</section>

<section class="tab-content" id="tab-timeline" role="tabpanel">
  {{if .Timeline}}{{range .Timeline}}<div class="timeline-row">
    <div class="timeline-label">{{.Label}}</div>
    <div class="timeline-track">
      <div class="timeline-bar build" style="width: {{.BuildPct}}%" title="{{.BuildTip}}"></div>
      <div class="timeline-bar shutdown" style="width: {{.ShutPct}}%" title="{{.ShutTip}}"></div>
    </div>
    <div class="timeline-dur mono">{{.Durations}}</div>
  </div>
  {{end}}{{else}}<div class="empty">No timing data recorded</div>{{end}}
</section>

<section class="tab-content" id="tab-events" role="tabpanel">
  <div class="toolbar" id="event-filters">
    {{range .EventTypes}}<button class="chip event-chip{{if eq . "all"}} active{{end}}" data-filter="{{.}}" aria-pressed="{{if eq . "all"}}true{{else}}false{{end}}">{{.}}</button>
    {{end}}
  </div>
  <div class="table-wrap">
  <table>
    <thead><tr><th>Seq</th><th>Time</th><th>Type</th><th>Phase</th><th>Scope</th><th>Service</th><th>Duration</th><th>Error</th></tr></thead>
    <tbody id="events-tbody">
    {{if .Events}}{{range .Events}}<tr data-type="{{.Type}}"{{if .HasError}} class="has-error"{{end}}>
      <td class="mono">{{.Seq}}</td>
      <td class="mono">{{.Time}}</td>
      <td><span class="event-badge {{.Type}}" title="{{.TypeLabel}}">{{.TypeLabel}}</span></td>
      <td class="mono">{{.Phase}}</td>
      <td>{{.Scope}}</td>
      <td class="mono">{{.Name}}</td>
      <td class="mono">{{.Dur}}</td>
      <td>{{if .HasError}}<span class="status-badge shutdown_error" data-error="{{.ErrorMsg}}">error</span>{{end}}</td>
    </tr>
    {{end}}{{else}}<tr><td colspan="8" class="empty">No events recorded</td></tr>{{end}}
    </tbody>
  </table>
  </div>
</section>

</main>
<footer>
  <span>Generated by <strong>do-auditlog</strong> · Press ? for keyboard shortcuts</span>
  <span class="mono">schema v{{.SchemaVersion}} · {{len .Events}} events · {{len .Services}} services</span>
</footer>
<script>
(function () {
  var tabs = Array.prototype.slice.call(document.querySelectorAll('.tab'));
  var panels = {
    services: document.getElementById('tab-services'),
    scopes: document.getElementById('tab-scopes'),
    graph: document.getElementById('tab-graph'),
    timeline: document.getElementById('tab-timeline'),
    events: document.getElementById('tab-events')
  };

  function switchTab(btn) {
    tabs.forEach(function (t) {
      t.classList.remove('active');
      t.setAttribute('aria-selected', 'false');
      t.setAttribute('tabindex', '-1');
    });
    Object.keys(panels).forEach(function (key) {
      panels[key].classList.remove('active');
    });
    btn.classList.add('active');
    btn.setAttribute('aria-selected', 'true');
    btn.setAttribute('tabindex', '0');
    panels[btn.dataset.tab].classList.add('active');
  }

  tabs.forEach(function (tab) {
    tab.addEventListener('click', function () { switchTab(tab); });
  });

  var kbdHelpPrevFocus = null;

  function closeKbdHelp() {
    var help = document.getElementById('kbd-help');
    if (!help) { return; }
    help.remove();
    if (kbdHelpPrevFocus) {
      kbdHelpPrevFocus.focus();
      kbdHelpPrevFocus = null;
    }
  }

  function showShortcutsHelp() {
    if (document.getElementById('kbd-help')) { closeKbdHelp(); return; }
    kbdHelpPrevFocus = document.activeElement;
    var div = document.createElement('div');
    div.id = 'kbd-help';
    div.className = 'kbd-help';
    div.setAttribute('role', 'dialog');
    div.setAttribute('aria-modal', 'true');
    div.setAttribute('aria-label', 'Keyboard shortcuts');
    div.innerHTML = '<div class="kbd-help-content"><h2>Keyboard shortcuts</h2><ul>'
      + '<li><span>Switch to tab 1–5</span><kbd>1</kbd>–<kbd>5</kbd></li>'
      + '<li><span>Next / previous tab</span><kbd>←</kbd> <kbd>→</kbd></li>'
      + '<li><span>Focus service search</span><kbd>/</kbd></li>'
      + '<li><span>Toggle errors-only filter</span><kbd>e</kbd></li>'
      + '<li><span>Sort services table</span><kbd>Enter</kbd>/<kbd>Space</kbd> on headers</li>'
      + '<li><span>Show this help</span><kbd>?</kbd></li>'
      + '<li><span>Close help</span><kbd>Esc</kbd></li>'
      + '</ul><button class="chip" id="kbd-help-close">Close</button></div>';
    document.body.appendChild(div);
    var closeBtn = document.getElementById('kbd-help-close');
    closeBtn.addEventListener('click', closeKbdHelp);
    closeBtn.focus();
    div.addEventListener('keydown', function (e) {
      if (e.key!=='Tab') { return; }
      var focusables = div.querySelectorAll('button,[href],input,select,textarea,[tabindex]:not([tabindex="-1"])');
      if (focusables.length === 0) { return; }
      if (e.shiftKey && document.activeElement === focusables[0]) {
        e.preventDefault();
        focusables[focusables.length - 1].focus();
      } else if (!e.shiftKey && document.activeElement === focusables[focusables.length - 1]) {
        e.preventDefault();
        focusables[0].focus();
      }
    });
  }

  document.addEventListener('keydown', function (e) {
    if (document.getElementById('kbd-help') && e.key === 'Escape') {
      closeKbdHelp();
      return;
    }
    var onTab = e.target.classList && e.target.classList.contains('tab');
    var current = tabs.indexOf(e.target);
    if (onTab && e.key === 'ArrowRight') {
      e.preventDefault();
      var next = tabs[(current + 1) % tabs.length];
      next.focus();
      switchTab(next);
      return;
    }
    if (onTab && e.key === 'ArrowLeft') {
      e.preventDefault();
      var prev = tabs[(current - 1 + tabs.length) % tabs.length];
      prev.focus();
      switchTab(prev);
      return;
    }
    var tag = e.target.tagName;
    var inField = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || tag === 'BUTTON';
    if (e.key==='?') {
      e.preventDefault();
      showShortcutsHelp();
      return;
    }
    if (inField) { return; }
    var num = parseInt(e.key, 10);
    if (num >= 1 && num <= tabs.length) {
      tabs[num - 1].focus();
      switchTab(tabs[num - 1]);
      return;
    }
    if (e.key==='/') {
      e.preventDefault();
      var search = document.getElementById('service-search');
      if (search) { search.focus(); }
      return;
    }
    if (e.key==='e') {
      var errBtn = document.getElementById('svc-errors-only');
      if (errBtn) { errBtn.click(); }
    }
  });

  var svcTbody = document.getElementById('services-tbody');
  var svcRows = Array.prototype.slice.call(svcTbody.querySelectorAll('tr[data-search]'));
  var countEl = document.getElementById('svc-result-count');
  var sortKey = 'order';
  var sortDir = 1;
  var numericSortKeys = ['order', 'invocations', 'build', 'shutdown'];

  function applySvcSort() {
    svcRows.sort(function (a, b) {
      var av = a.dataset['sort' + sortKey.charAt(0).toUpperCase() + sortKey.slice(1)];
      var bv = b.dataset['sort' + sortKey.charAt(0).toUpperCase() + sortKey.slice(1)];
      var cmp = 0;
      if (numericSortKeys.indexOf(sortKey) !== -1) {
        cmp = parseFloat(av || '0') - parseFloat(bv || '0');
      } else {
        av = (av || '').toLowerCase();
        bv = (bv || '').toLowerCase();
        cmp = av < bv ? -1 : (av > bv ? 1 : 0);
      }
      return cmp * sortDir;
    });
    svcRows.forEach(function (tr) { svcTbody.appendChild(tr); });
    Array.prototype.slice.call(svcTbody.closest('table').querySelectorAll('th.sortable')).forEach(function (th) {
      th.classList.remove('sort-asc', 'sort-desc');
      if (th.dataset.sort === sortKey) {
        th.classList.add(sortDir === 1 ? 'sort-asc' : 'sort-desc');
        th.setAttribute('aria-sort', sortDir === 1 ? 'ascending' : 'descending');
      } else {
        th.setAttribute('aria-sort', 'none');
      }
    });
  }

  Array.prototype.slice.call(document.querySelectorAll('th.sortable')).forEach(function (th) {
    function activate() {
      var key = th.dataset.sort;
      if (sortKey === key) {
        sortDir *= -1;
      } else {
        sortKey = key;
        sortDir = 1;
      }
      applySvcSort();
    }
    th.addEventListener('click', activate);
    th.addEventListener('keydown', function (e) {
      if (e.key==='Enter' || e.key === ' ') {
        e.preventDefault();
        activate();
      }
    });
  });

  function applySvcFilter() {
    var q = document.getElementById('service-search').value.toLowerCase();
    var errorsOnly = document.getElementById('svc-errors-only').getAttribute('aria-pressed') === 'true';
    var visible = 0;
    svcRows.forEach(function (tr) {
      var show = true;
      if (q && tr.dataset.search.toLowerCase().indexOf(q) === -1) { show = false; }
      if (errorsOnly && tr.dataset.hasError !== '1') { show = false; }
      tr.style.display = show ? '' : 'none';
      if (show) { visible++; }
    });
    if (countEl) {
      countEl.textContent = (q || errorsOnly) ? visible + ' / ' + svcRows.length + ' services' : '';
    }
  }

  document.getElementById('service-search').addEventListener('input', applySvcFilter);

  document.getElementById('svc-errors-only').addEventListener('click', function () {
    var pressed = this.getAttribute('aria-pressed') === 'true';
    this.setAttribute('aria-pressed', String(!pressed));
    this.classList.toggle('active', !pressed);
    applySvcFilter();
  });

  var eventChips = Array.prototype.slice.call(document.querySelectorAll('.event-chip'));
  var eventRows = Array.prototype.slice.call(document.querySelectorAll('#events-tbody tr[data-type]'));

  eventChips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      eventChips.forEach(function (c) {
        c.classList.remove('active');
        c.setAttribute('aria-pressed', 'false');
      });
      chip.classList.add('active');
      chip.setAttribute('aria-pressed', 'true');
      var filter = chip.dataset.filter;
      eventRows.forEach(function (tr) {
        var show = filter === 'all' || tr.dataset.type === filter;
        tr.style.display = show ? '' : 'none';
      });
    });
  });

  applySvcSort();
})();
</script>
</body>
</html>
` + scopeNodeTemplate,
))

// scopeNodeTemplate recursively renders a scope and its children.
const scopeNodeTemplate = `{{define "scopeNode"}}<div>
  <span class="scope-name">{{.Name}}</span>{{if .Services}}<span class="scope-count">{{len .Services}} service{{if ne (len .Services) 1}}s{{end}}</span>{{end}}
  {{if .Services}}<div>
    {{range .Services}}<span class="scope-svc mono">{{.}}</span>
    {{end}}
  </div>{{end}}
  {{range .Children}}{{template "scopeNode" .}}{{end}}
</div>
{{end}}`
