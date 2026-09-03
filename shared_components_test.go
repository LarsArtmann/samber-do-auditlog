package auditlog

import (
	"strings"
	"testing"
)

// TestSharedComponentCSSInSync verifies that the keyboard-navigation overlay
// CSS rules (.skip-link, .kbd-help, .kbd-help-content) are embedded in the
// rendered HTML report verbatim from the SharedComponentCSS constant. This
// keeps a single canonical source for the shared overlay styles.
func TestSharedComponentCSSInSync(t *testing.T) {
	t.Parallel()

	html, err := Report{}.WriteHTMLString()
	if err != nil {
		t.Fatalf("WriteHTMLString: %v", err)
	}

	if !strings.Contains(html, SharedComponentCSS) {
		t.Error("rendered HTML report does not embed SharedComponentCSS verbatim — " +
			"the template must inject the shared constant via {{css .SharedCSS}}")
	}
}
