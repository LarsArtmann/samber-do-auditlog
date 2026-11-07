package auditlog_test

import (
	"bytes"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// FuzzDiagramSpecialChars fuzzes diagram export (Mermaid + PlantUML) with
// service names containing special characters, injection payloads, and edge
// cases.
func FuzzDiagramSpecialChars(f *testing.F) {
	special := []string{
		"svc]",
		`svc"`,
		"svc-->",
		"@enduml",
		"%%",
		"svc\nother", // newline injection
		"svc|pipe",
		"a]b[c",
		`a"b"c`,
		`evil]"svc`,    // combined bracket+quote (the regression test case)
		`{inj}ect[ion`, //nolint:misspell // adversarial fuzz seed, not a real word
		"<script>alert(1)</script>",
		strings.Repeat("A", 500),
		"svc:colon",  // D2 colon injection
		"svc\tother", // D2 tab injection
	}

	for _, s := range special {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, svcName string) {
		t.Parallel()

		if svcName == "" {
			t.Skip()
		}

		plugin := mustNew(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(plugin.Opts())

		provideString(injector, svcName, "val")

		_, err := do.InvokeNamed[string](injector, svcName)
		if err != nil {
			t.Skip()
		}

		report := plugin.Report()

		var mBuf bytes.Buffer

		mErr := report.WriteMermaid(&mBuf)
		if mErr != nil {
			t.Fatalf("WriteMermaid error: %v", mErr)
		}

		mOut := mBuf.String()
		assertStringContains(t, mOut, "flowchart TD")

		// Structural integrity: the Mermaid node syntax id[label] must stay
		// balanced. A raw double-quote in the label would indicate the escaping
		// regressed (the theme header uses single quotes exclusively).
		if strings.Contains(mOut, "\"") {
			t.Errorf("raw double-quote in Mermaid output: %s", mOut)
		}

		var pBuf bytes.Buffer

		pErr := report.WritePlantUML(&pBuf)
		if pErr != nil {
			t.Fatalf("WritePlantUML error: %v", pErr)
		}

		pOut := pBuf.String()
		assertStringContains(t, pOut, "@startuml")
		assertStringContains(t, pOut, "@enduml")

		var dBuf bytes.Buffer

		dErr := report.WriteDOT(&dBuf)
		if dErr != nil {
			t.Fatalf("WriteDOT error: %v", dErr)
		}

		dOut := dBuf.String()
		assertStringContains(t, dOut, "digraph do_auditlog")

		var d2Buf bytes.Buffer

		d2Err := report.WriteD2(&d2Buf)
		if d2Err != nil {
			t.Fatalf("WriteD2 error: %v", d2Err)
		}

		d2Out := d2Buf.String()
		assertStringContains(t, d2Out, ":")
		assertStringContains(t, d2Out, "title:")
	})
}

func assertNoRawXSS(t *testing.T, output, context string) {
	t.Helper()

	vectors := []string{
		"<script>alert",
		"<img src=x onerror=",
		"<svg onload=",
		"<iframe ",
		" onmouseover=\"alert",
	}

	for _, v := range vectors {
		if strings.Contains(output, v) {
			t.Errorf("unescaped %q in HTML output for context %q", v, context)
		}
	}

	// Check for javascript: and onerror= only in HTML portions (outside JSON script blocks).
	// Error messages like "javascript:alert(1)" are safely encoded inside JSON in script tags.
	// templ's JSONScript escapes </ to prevent premature closing, so the data is inert.
	htmlOnly := stripJSONScripts(output)

	htmlVectors := []string{
		"javascript:alert",
		" onerror=",
		" onclick=",
		" onload=",
	}

	for _, v := range htmlVectors {
		if strings.Contains(htmlOnly, v) {
			t.Errorf("unescaped %q in HTML portion for context %q", v, context)
		}
	}
}

// stripJSONScripts removes all <script ... type="application/json">...</script>
// blocks from the HTML output. These blocks contain report data as JSON
// (including user-controlled strings), which is inert in the browser.
// The remaining HTML is checked for unescaped XSS vectors.
func stripJSONScripts(html string) string {
	const marker = `type="application/json"`

	const closeTag = `</script>`

	for {
		idx := strings.Index(html, marker)
		if idx < 0 {
			break
		}

		// Backtrack to the opening <script tag.
		scriptStart := strings.LastIndex(html[:idx], "<script")
		if scriptStart < 0 {
			break
		}

		closeIdx := strings.Index(html[scriptStart:], closeTag)
		if closeIdx < 0 {
			break
		}

		end := scriptStart + closeIdx + len(closeTag)
		html = html[:scriptStart] + html[end:]
	}

	return html
}

// TestNestedScopeExport is a table-driven test covering deeply nested scope
// trees across a range of depths. Converted from a fuzz target to keep the
// total fuzz-target count at 3 (each costs ~30 s in buildflow's sequential
// runner; 3 × 30 s < the 2-minute default hard timeout).
