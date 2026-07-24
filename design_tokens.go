package auditlog

// DesignTokensCSS is the single source of truth for the warm amber "Container
// Telemetry" design palette. Both the static HTML report (html.templ) and the
// live dashboard (live/base_css.go) reference these exact CSS custom properties
// to prevent visual drift.
//
// TestDesignTokensInSync (in design_tokens_test.go) verifies that the inline
// :root block in html.templ matches this constant exactly.
//
//nolint:gochecknoglobals,gosec // read-only CSS constant (gosec G101 false positive on "Token" in name)
var DesignTokensCSS = `:root {
    --bg: #14110d;
    --bg-elevated: #1c1812;
    --surface: #241f18;
    --surface-hover: #2c2620;
    --border: #2e2820;
    --border-active: #4a4030;
    --text: #ddd4c4;
    --text-muted: #9a8d78;
    --text-dim: #7d7260;
    --accent: #e8a838;
    --accent-dim: rgba(232,168,56,0.12);
    --success: #6fbf8e;
    --success-dim: rgba(111,191,142,0.12);
    --warning: #d4a843;
    --warning-dim: rgba(212,168,67,0.12);
    --error: #e07060;
    --error-dim: rgba(224,112,96,0.12);
    --info: #8ec5d4;
    --lazy: #9d8cd4;
    --eager: #e8a838;
    --transient: #d49060;
    --alias: #6fbf8e;
    --radius: 8px;
    --transition: 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }`
