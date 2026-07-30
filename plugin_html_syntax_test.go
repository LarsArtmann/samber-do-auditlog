package auditlog_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/samber-do-auditlog/internal/testhelpers"
	"github.com/samber/do/v2"
)

// TestHTMLJavaScriptSyntax validates that the JavaScript embedded in the HTML
// report has balanced delimiters (braces, parens, brackets) after stripping
// string literals, comments, and regex literals. This catches syntax errors
// like a stray '}' that the golden byte-for-byte test would miss (since it
// compares against a golden file that may also contain the error).
func TestHTMLJavaScriptSyntax(t *testing.T) {
	t.Parallel()

	html := writeHTMLToString(t)
	js := testhelpers.ExtractExecutableJS(t, html)

	if len(js) < 100 {
		t.Fatalf("extracted JS too short (%d bytes) — extraction may have failed", len(js))
	}

	testhelpers.AssertJSBalanced(t, js)
}

// TestHTMLJavaScriptSyntax_MultiService runs the same check on a report with
// multiple scopes and services to ensure template expansion doesn't break JS.
func TestHTMLJavaScriptSyntax_MultiService(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	child := injector.Scope("child-scope")

	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideUserServiceWithDB(injector, "user-service", "db")
	provideDB(child, "child-db", "postgres://child")

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")
	_ = do.MustInvokeNamed[*UserService](injector, "user-service")
	_ = do.MustInvokeNamed[*Database](child, "child-db")

	var buf bytes.Buffer

	if err := p.WriteHTML(&buf); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	js := testhelpers.ExtractExecutableJS(t, buf.String())

	if len(js) < 100 {
		t.Fatalf("extracted JS too short (%d bytes)", len(js))
	}

	testhelpers.AssertJSBalanced(t, js)
}

// TestHTMLKeyboardShortcutsSyntax validates that the keyboard-navigation
// code added to the report is syntactically balanced.
func TestHTMLKeyboardShortcutsSyntax(t *testing.T) {
	t.Parallel()

	html := writeHTMLToString(t)

	if !strings.Contains(html, "showShortcutsHelp") {
		t.Fatal("keyboard shortcut help function missing from HTML")
	}

	if !strings.Contains(html, "e.key==='?'") || !strings.Contains(html, "e.key==='/'") || !strings.Contains(html, "e.key==='e'") {
		t.Fatal("keyboard shortcuts missing from HTML")
	}

	js := testhelpers.ExtractExecutableJS(t, html)
	testhelpers.AssertJSBalanced(t, js)
}
