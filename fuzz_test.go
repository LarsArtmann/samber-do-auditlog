package auditlog_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func FuzzPluginHTML(f *testing.F) {
	malicious := []string{
		"<script>alert('xss')</script>",
		"\" onload=\"alert(1)",
		"'; DROP TABLE--",
		"<img src=x onerror=alert(1)>",
		"\x00null\x00bytes",
		strings.Repeat("A", 1000),
		"\n\r\t",
		"{{.ServiceName}}",
		"${7*7}",
		"<svg onload=alert(1)>",
		"javascript:alert(1)",
		"<iframe src=\"evil.com\">",
		"<a href=\"javascript:alert(1)\">click</a>",
		"'><script>alert(1)</script>",
		"\" onmouseover=\"alert(1)",
	}

	for _, m := range malicious {
		f.Add(m)
	}

	f.Fuzz(func(t *testing.T, svcName string) {
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

		var buf bytes.Buffer

		writeErr := plugin.WriteHTML(&buf)
		if writeErr != nil {
			return
		}

		output := buf.String()
		assertNoRawXSS(t, output, svcName)
	})
}

func FuzzPluginHTML_ErrorMessages(f *testing.F) {
	maliciousErrors := []string{
		"<script>alert('err')</script>",
		"<img src=x onerror=alert(1)>",
		"\" onclick=\"alert(1)",
		"<svg onload=alert(1)>",
		"javascript:alert(1)",
		"'><script>alert(1)</script>",
	}

	for _, m := range maliciousErrors {
		f.Add(m)
	}

	f.Fuzz(func(t *testing.T, errMsg string) {
		if errMsg == "" {
			t.Skip()
		}

		plugin := mustNew(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(plugin.Opts())

		do.ProvideNamed(injector, "error-svc", func(_ do.Injector) (string, error) {
			return "", fmt.Errorf("%s", errMsg) //nolint:err113
		})

		_, _ = do.InvokeNamed[string](injector, "error-svc")

		var buf bytes.Buffer

		writeErr := plugin.WriteHTML(&buf)
		if writeErr != nil {
			return
		}

		output := buf.String()
		assertNoRawXSS(t, output, errMsg)
	})
}

func FuzzMigrateReport(f *testing.F) {
	seeds := []string{
		`{"version":"0.1.0"}`,
		`{"version":"0.2.0"}`,
		`{"version":"0.1.0","services":[{"service_name":"svc","scope_id":"r","scope_name":"[root]"}]}`,
		`{"version":"0.1.0","scope_tree":{"id":"root","name":"[root]","services":[],"children":[{"id":"child","name":"child","services":[],"children":[]}]}}`,
		`not json`,
		`{}`,
		``,
	}

	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		report, err := auditlog.MigrateReport(data)
		if err != nil {
			// Errors are expected for malformed/empty input.
			return
		}

		if validateErr := report.Validate(); validateErr != nil {
			t.Errorf("migrated report failed validation: %v", validateErr)
		}

		// Re-migrating a current-schema report should be a no-op and stay valid.
		var buf bytes.Buffer

		enc := json.NewEncoder(&buf)
		if encodeErr := enc.Encode(report); encodeErr != nil {
			t.Fatalf("encode migrated report: %v", encodeErr)
		}

		reMigrated, reErr := auditlog.MigrateReport(buf.Bytes())
		if reErr != nil {
			t.Errorf("re-migrating current-schema report failed: %v", reErr)
		}

		if reValidateErr := reMigrated.Validate(); reValidateErr != nil {
			t.Errorf("re-migrated report failed validation: %v", reValidateErr)
		}
	})
}

func FuzzPluginHTML_DepChain(f *testing.F) {
	maliciousDeps := []string{
		"<script>alert('dep')</script>",
		"<img src=x onerror=alert(1)>",
		"\" onclick=\"alert(1)",
		"<svg onload=alert(1)>",
	}

	for _, m := range maliciousDeps {
		f.Add(m)
	}

	f.Fuzz(func(t *testing.T, depName string) {
		if depName == "" {
			t.Skip()
		}

		plugin := mustNew(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(plugin.Opts())

		provideString(injector, depName, "dep-val")

		do.ProvideNamed(injector, "parent-svc", func(i do.Injector) (string, error) {
			_, _ = do.InvokeNamed[string](i, depName)

			return "parent-val", nil
		})

		_, _ = do.InvokeNamed[string](injector, "parent-svc")

		var buf bytes.Buffer

		writeErr := plugin.WriteHTML(&buf)
		if writeErr != nil {
			return
		}

		output := buf.String()
		assertNoRawXSS(t, output, depName)
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
