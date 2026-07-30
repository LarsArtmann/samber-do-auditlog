package live

import auditlog "github.com/larsartmann/samber-do-auditlog"

// liveTokenAliases maps canonical design token names (from
// auditlog.DesignTokensCSS) to the variable names used by the live dashboard's
// component CSS. This keeps both dashboards on the same palette without
// duplicating color values.
//
//nolint:gosec // G101 false positive: CSS tokens, not credentials
const liveTokenAliases = `
:root {
  --bg-card: var(--bg-elevated);
  --bg-hover: var(--surface-hover);
  --border-light: var(--border-active);
  --font: 'Space Grotesk', -apple-system, BlinkMacSystemFont, sans-serif;
  --font-mono: 'IBM Plex Mono', 'Fira Code', monospace;
}`

// baseCSS composes the shared design tokens (auditlog.DesignTokensCSS) with
// live-dashboard-specific aliases and base component styles. The shared tokens
// guarantee visual consistency with the static HTML report.
//
//nolint:gochecknoglobals // read-only CSS constant, not mutable shared state
var baseCSS = auditlog.DesignTokensCSS + liveTokenAliases + `
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; border-radius: 3px; }
.skip-link { position: absolute; left: -9999px; top: auto; width: 1px; height: 1px; overflow: hidden; z-index: 100; font-family: var(--font); }
.skip-link:focus { position: fixed; left: 1rem; top: 1rem; width: auto; height: auto; padding: 0.75rem 1rem; background: var(--accent); color: var(--bg); font-weight: 600; border-radius: var(--radius); text-decoration: none; outline: none; box-shadow: 0 4px 24px rgba(0,0,0,0.4); }
.kbd-help { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(20,17,13,0.85); z-index: 200; display: flex; align-items: center; justify-content: center; }
.kbd-help-content { background: var(--bg-card); border: 1px solid var(--border-light); border-radius: var(--radius); padding: 1.5rem; max-width: 420px; width: 90%; color: var(--text); box-shadow: 0 16px 48px rgba(0,0,0,0.5); }
.kbd-help-content h2 { font-size: 1.1rem; margin-bottom: 1rem; color: var(--accent); }
.kbd-help-content ul { list-style: none; margin: 0; padding: 0; }
.kbd-help-content li { display: flex; justify-content: space-between; padding: 0.4rem 0; border-bottom: 1px solid var(--border); font-size: 0.85rem; }
.kbd-help-content li:last-child { border-bottom: none; }
.kbd-help-content kbd { font-family: var(--font-mono); background: var(--bg); border: 1px solid var(--border-light); border-radius: 4px; padding: 0.1rem 0.35rem; color: var(--accent); }
.scope-label:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; border-radius: 3px; }
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
