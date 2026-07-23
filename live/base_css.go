package live

// baseCSS contains the essential CSS custom properties and base styles
// extracted from the static HTML dashboard. This is the warm amber "Container
// Telemetry" aesthetic shared between the static and live dashboards.
//
//nolint:gochecknoglobals // read-only CSS constant, not mutable shared state
var baseCSS = `/* samber-do-auditlog base CSS — warm amber Container Telemetry aesthetic */
:root {
  --bg: #14110d;
  --bg-card: #1c1914;
  --bg-hover: #252118;
  --text: #e8dcc8;
  --text-dim: #a09080;
  --text-muted: #6b5f52;
  --accent: #e8a838;
  --accent-dim: rgba(232, 168, 56, 0.12);
  --success: #6fbf8e;
  --success-dim: rgba(111, 191, 142, 0.12);
  --warning: #e8a838;
  --warning-dim: rgba(232, 168, 56, 0.12);
  --error: #e07060;
  --error-dim: rgba(224, 112, 96, 0.12);
  --info: #7eb8d8;
  --lazy: #b89fd8;
  --eager: #e8a838;
  --transient: #d8976b;
  --alias: #6fbf8e;
  --border: #2a2520;
  --border-light: #3a3530;
  --font: 'Space Grotesk', -apple-system, BlinkMacSystemFont, sans-serif;
  --font-mono: 'IBM Plex Mono', 'Fira Code', monospace;
  --transition: 0.2s ease;
  --radius: 6px;
}
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: var(--font);
  background: var(--bg);
  color: var(--text);
  line-height: 1.5;
  min-height: 100vh;
  background-image: radial-gradient(ellipse at 50% 0%, rgba(232,168,56,0.04) 0%, transparent 60%);
}
header { display: flex; justify-content: space-between; align-items: flex-start; padding: 1.5rem 2rem; border-bottom: 1px solid var(--border); flex-wrap: wrap; gap: 1rem; }
.header-left h1 { font-size: 1.25rem; font-weight: 700; display: flex; align-items: center; gap: 0.5rem; }
.logo-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--accent); display: inline-block; }
.version { font-size: 0.65rem; color: var(--text-dim); font-family: var(--font-mono); background: var(--bg-card); padding: 0.1rem 0.4rem; border-radius: 3px; }
.subtitle { font-size: 0.8rem; color: var(--text-dim); margin-top: 0.25rem; }
.mono { font-family: var(--font-mono); }
.legend { display: flex; gap: 0.75rem; flex-wrap: wrap; align-items: center; }
.legend-item { display: flex; align-items: center; gap: 0.3rem; font-size: 0.75rem; color: var(--text-dim); }
.legend-item .icon { font-size: 0.85rem; }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 0.75rem; padding: 1rem 2rem; }
.stat-card { background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius); padding: 0.75rem 1rem; text-align: center; position: relative; overflow: hidden; }
.stat-card::after { content: ''; position: absolute; top: 0; left: 0; right: 0; height: 2px; background: var(--accent); }
.stat-card.success::after { background: var(--success); }
.stat-card.error::after { background: var(--error); }
.stat-card .label { font-size: 0.7rem; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.05em; }
.stat-card .value { font-size: 1.25rem; font-weight: 700; font-family: var(--font-mono); margin-top: 0.15rem; }
.stat-card.success .value { color: var(--success); }
.stat-card.error .value { color: var(--error); }
.waveform-section { padding: 0.75rem 2rem; }
.waveform-label { font-size: 0.7rem; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.05em; display: block; margin-bottom: 0.35rem; }
.waveform { display: flex; align-items: flex-end; gap: 1px; height: 32px; background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius); padding: 2px 4px; overflow: hidden; position: relative; }
.wf-event { position: absolute; bottom: 2px; width: 3px; border-radius: 1px; min-height: 2px; }
.waveform-legend { display: flex; gap: 1rem; margin-top: 0.3rem; }
.wf-legend-item { display: flex; align-items: center; gap: 0.3rem; font-size: 0.65rem; color: var(--text-dim); }
.wf-legend-dot { width: 6px; height: 6px; border-radius: 50%; }
.tab-bar { display: flex; gap: 0; border-bottom: 1px solid var(--border); padding: 0 2rem; }
.tab { background: none; border: none; color: var(--text-dim); font-family: var(--font); font-size: 0.8rem; padding: 0.6rem 1.25rem; cursor: pointer; border-bottom: 2px solid transparent; transition: all var(--transition); }
.tab:hover { color: var(--text); }
.tab.active { color: var(--accent); border-bottom-color: var(--accent); }
.tab-content { display: none; padding: 1rem 2rem; }
.tab-content.active { display: block; }
.filter-bar { display: flex; gap: 0.5rem; margin-bottom: 0.75rem; align-items: center; }
.filter-bar input { background: var(--bg-card); border: 1px solid var(--border); color: var(--text); padding: 0.4rem 0.75rem; border-radius: var(--radius); font-family: var(--font); font-size: 0.8rem; min-width: 200px; }
.filter-bar input:focus { outline: none; border-color: var(--accent); }
.chip { background: var(--bg-card); border: 1px solid var(--border); color: var(--text-dim); padding: 0.25rem 0.6rem; border-radius: var(--radius); font-size: 0.7rem; cursor: pointer; transition: all var(--transition); font-family: var(--font); }
.chip:hover, .chip.active { color: var(--text); border-color: var(--accent); background: var(--accent-dim); }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
th { text-align: left; padding: 0.5rem 0.75rem; color: var(--text-dim); font-weight: 600; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.04em; border-bottom: 1px solid var(--border); position: sticky; top: 0; background: var(--bg); }
td { padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--border); vertical-align: top; }
tr:hover td { background: var(--bg-hover); }
.event-badge { display: inline-block; padding: 0.1rem 0.4rem; border-radius: 3px; font-size: 0.7rem; font-weight: 600; color: var(--bg); }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); border: 0; }
.empty-state td { text-align: center; color: var(--text-muted); font-style: italic; padding: 2rem; }
.footer { display: flex; justify-content: space-between; padding: 1rem 2rem; border-top: 1px solid var(--border); font-size: 0.7rem; color: var(--text-dim); }
.graph-controls { position: absolute; top: 0.5rem; right: 0.5rem; display: flex; gap: 0.25rem; z-index: 10; }
.graph-controls button { background: var(--bg-card); border: 1px solid var(--border); color: var(--text-dim); width: 28px; height: 28px; border-radius: var(--radius); cursor: pointer; font-size: 0.85rem; display: flex; align-items: center; justify-content: center; }
.graph-controls button:hover { color: var(--text); border-color: var(--accent); }
.graph-info { font-size: 0.7rem; color: var(--text-muted); text-align: center; padding: 0.5rem; }
#graph-container { position: relative; min-height: 500px; background: var(--bg-card); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.tooltip { position: absolute; background: var(--bg-card); border: 1px solid var(--border); color: var(--text); padding: 0.5rem 0.75rem; border-radius: var(--radius); font-size: 0.75rem; pointer-events: none; z-index: 100; max-width: 300px; display: none; }
@media (max-width: 768px) {
  header { padding: 1rem; }
  .stats { padding: 0.75rem 1rem; }
  .tab-content { padding: 0.75rem 1rem; }
  .waveform-section { padding: 0.5rem 1rem; }
  .tab { font-size: 0.7rem; padding: 0.5rem 0.75rem; }
}
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation-duration: 0.01ms !important; animation-iteration-count: 1 !important; transition-duration: 0.01ms !important; }
}`
