package auditlog

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// tokenPattern extracts --name: value; pairs from CSS.
var tokenPattern = regexp.MustCompile(`(--[a-z-]+)\s*:\s*([^;]+);`)

func parseCSSTokens(t *testing.T, css string) map[string]string {
	t.Helper()

	matches := tokenPattern.FindAllStringSubmatch(css, -1)
	result := make(map[string]string, len(matches))

	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		value := strings.TrimSpace(m[2])
		result[name] = value
	}

	return result
}

// TestDesignTokensInSync verifies that the inline :root block in html.templ
// matches DesignTokensCSS exactly. This prevents visual drift between the
// static HTML report and the live dashboard.
func TestDesignTokensInSync(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("html.templ")
	if err != nil {
		t.Fatalf("read html.templ: %v", err)
	}

	templTokens := parseCSSTokens(t, string(source))
	canonicalTokens := parseCSSTokens(t, DesignTokensCSS)

	// Every canonical token must appear in html.templ with the same value.
	for name, expected := range canonicalTokens {
		actual, ok := templTokens[name]
		if !ok {
			t.Errorf("html.templ is missing design token %s — add it or update DesignTokensCSS", name)

			continue
		}

		if actual != expected {
			t.Errorf("design token %s drifted:\n  DesignTokensCSS: %s\n  html.templ:       %s",
				name, expected, actual)
		}
	}

	// Check the reverse: html.templ tokens that are NOT in DesignTokensCSS
	// (these are fine as long as they're not standard palette colors).
	var extra []string

	for name := range templTokens {
		if _, ok := canonicalTokens[name]; !ok {
			extra = append(extra, name)
		}
	}

	if len(extra) > 0 {
		sort.Strings(extra)
		t.Logf("html.templ has tokens not in DesignTokensCSS (OK if non-palette): %v", extra)
	}
}
