package auditlog

import (
	"strings"
	"testing"
)

// TestDesignTokensInSync verifies that the rendered HTML report embeds
// DesignTokensCSS verbatim. On this branch the report is a single
// html/template document, so "in sync" means: the constant IS the stylesheet
// the template ships — no second hand-maintained copy exists to drift.
func TestDesignTokensInSync(t *testing.T) {
	t.Parallel()

	html, err := Report{}.WriteHTMLString()
	if err != nil {
		t.Fatalf("WriteHTMLString: %v", err)
	}

	if !strings.Contains(html, DesignTokensCSS) {
		t.Error("rendered HTML report does not embed DesignTokensCSS verbatim — " +
			"the template must inject the shared constant via {{css .DesignTokens}}")
	}
}
