package auditlog_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// FuzzPluginHTML fuzzes all HTML XSS vectors simultaneously: service names,
// error messages, and dependency-chain names. Merged from three separate
// targets to reduce the total fuzz-target count (each target costs ~30 s in
// buildflow's sequential fuzz runner; keeping the count at 3 keeps the full
// suite under the default 2-minute hard timeout).
func FuzzPluginHTML(f *testing.F) {
	malicious := []string{
		// General XSS / injection payloads.
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
		"\" onclick=\"alert(1)",
	}

	for _, m := range malicious {
		f.Add(m)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if input == "" {
			t.Skip()
		}

		plugin := mustNew(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(plugin.Opts())

		// Vector 1: service name XSS.
		provideString(injector, input, "val")

		_, err := do.InvokeNamed[string](injector, input)
		if err != nil {
			t.Skip()
		}

		// Vector 2: error-message XSS.
		do.ProvideNamed(injector, "error-svc", func(_ do.Injector) (string, error) {
			return "", fmt.Errorf("%s", input) //nolint:err113
		})

		_, _ = do.InvokeNamed[string](injector, "error-svc")

		// Vector 3: dependency-chain XSS.
		do.ProvideNamed(injector, "parent-svc", func(i do.Injector) (string, error) {
			_, _ = do.InvokeNamed[string](i, input)

			return "parent-val", nil
		})

		_, _ = do.InvokeNamed[string](injector, "parent-svc")

		var buf bytes.Buffer

		writeErr := plugin.WriteHTML(&buf)
		if writeErr != nil {
			return
		}

		output := buf.String()
		assertNoRawXSS(t, output, input)
	})
}
