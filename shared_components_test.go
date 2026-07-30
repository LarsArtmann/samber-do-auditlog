package auditlog

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// cssRulePattern matches CSS rules: selector(s) { declarations }.
var cssRulePattern = regexp.MustCompile(`([^{]+)\{([^}]+)\}`)

// parseCSSRules extracts a map of normalized-selector → normalized-declarations.
// Whitespace is collapsed so the comparison is layout-independent.
func parseCSSRules(t *testing.T, css string) map[string]string {
	t.Helper()

	matches := cssRulePattern.FindAllStringSubmatch(css, -1)
	result := make(map[string]string, len(matches))

	for _, m := range matches {
		selector := collapseWhitespace(m[1])
		declarations := collapseWhitespace(m[2])

		if selector != "" && declarations != "" {
			result[selector] = declarations
		}
	}

	return result
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

// TestSharedComponentCSSInSync verifies that the keyboard-navigation overlay
// CSS rules (.skip-link, .kbd-help, .kbd-help-content) in html.templ match the
// SharedComponentCSS constant exactly. This prevents visual and behavioural
// drift between the static HTML report and the live dashboard.
func TestSharedComponentCSSInSync(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("html.templ")
	if err != nil {
		t.Fatalf("read html.templ: %v", err)
	}

	templRules := parseCSSRules(t, string(source))
	canonicalRules := parseCSSRules(t, SharedComponentCSS)

	for selector, expected := range canonicalRules {
		actual, ok := templRules[selector]
		if !ok {
			t.Errorf("html.templ is missing CSS rule for %s — add it or update SharedComponentCSS", selector)

			continue
		}

		if actual != expected {
			t.Errorf("CSS rule %s drifted:\n  SharedComponentCSS: %s\n  html.templ:           %s",
				selector, expected, actual)
		}
	}
}
