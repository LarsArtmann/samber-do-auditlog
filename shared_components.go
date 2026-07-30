package auditlog

// SharedComponentCSS contains keyboard-navigation overlay styles that are used
// by both the static HTML report (html.templ) and the live dashboard
// (live/base_css.go). Keeping these in a single constant prevents visual and
// behavioural drift between the two dashboards.
//
// TestSharedComponentCSSInSync (in shared_components_test.go) verifies that the
// inline styles in html.templ match this constant exactly.
//
// Token names used here (--bg-elevated, --border-active, etc.) exist in both
// dashboards via DesignTokensCSS. The live dashboard aliases some of them
// (--bg-card = --bg-elevated, --border-light = --border-active) but the
// canonical names are used here so both dashboards resolve the same values.
//
//nolint:gochecknoglobals // read-only CSS constant
var SharedComponentCSS = `.skip-link { position: absolute; left: -9999px; top: auto; width: 1px; height: 1px; overflow: hidden; z-index: 100; }
.skip-link:focus { position: fixed; left: 1rem; top: 1rem; width: auto; height: auto; padding: 0.75rem 1rem; background: var(--accent); color: var(--bg); font-weight: 600; border-radius: var(--radius); text-decoration: none; outline: none; box-shadow: 0 4px 24px rgba(0,0,0,0.4); }
.kbd-help { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(20,17,13,0.85); z-index: 200; display: flex; align-items: center; justify-content: center; }
.kbd-help-content { background: var(--bg-elevated); border: 1px solid var(--border-active); border-radius: var(--radius); padding: 1.5rem; max-width: 420px; width: 90%; color: var(--text); box-shadow: 0 16px 48px rgba(0,0,0,0.5); }
.kbd-help-content h2 { font-size: 1.1rem; margin-bottom: 1rem; color: var(--accent); }
.kbd-help-content ul { list-style: none; margin: 0; padding: 0; }
.kbd-help-content li { display: flex; justify-content: space-between; padding: 0.4rem 0; border-bottom: 1px solid var(--border); font-size: 0.85rem; }
.kbd-help-content li:last-child { border-bottom: none; }
.kbd-help-content kbd { font-family: 'IBM Plex Mono', monospace; background: var(--bg); border: 1px solid var(--border-active); border-radius: 4px; padding: 0.1rem 0.35rem; color: var(--accent); }`
