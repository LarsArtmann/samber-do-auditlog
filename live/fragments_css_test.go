package live

import (
	"strings"
	"testing"
)

func TestCSSCompleteness(t *testing.T) {
	t.Parallel()

	combinedCSS := baseCSS + "\n" + liveCSS

	for _, cls := range cssClassesFromFragments {
		t.Run("class/"+cls, func(t *testing.T) {
			t.Parallel()

			if !cssHasSelector(combinedCSS, cls) {
				t.Errorf("CSS class %q used in fragments.templ has no selector in dashboard.css or base_css.go", cls)
			}
		})
	}

	for _, compound := range cssCompoundSelectors {
		t.Run("compound/"+compound, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(combinedCSS, "."+compound) {
				t.Errorf("Compound selector %q used in fragments.templ has no rule in CSS", compound)
			}
		})
	}
}

// cssHasSelector checks whether the CSS text defines a rule for the given class
// name. It looks for ".classname" followed by a non-identifier character (or end
// of string) to avoid false positives like ".dep-node" matching ".dep-node-header".
func cssHasSelector(css, className string) bool {
	target := "." + className

	idx := strings.Index(css, target)

	for idx != -1 {
		afterIdx := idx + len(target)

		if afterIdx >= len(css) {
			return true
		}

		next := css[afterIdx]

		if !isCSSIdentChar(next) {
			return true
		}

		// False positive (e.g., ".dep-node" inside ".dep-node-header").
		// Search again from the next position.
		rest := css[idx+1:]
		nextIdx := strings.Index(rest, target)

		if nextIdx == -1 {
			return false
		}

		idx = idx + 1 + nextIdx
	}

	return false
}

func isCSSIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

// --- Coverage gap tests ---
