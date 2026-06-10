package auditlog_test

import (
	"bytes"
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
	}

	for _, m := range malicious {
		f.Add(m)
	}

	f.Fuzz(func(t *testing.T, svcName string) {
		if svcName == "" {
			t.Skip()
		}

		plugin := auditlog.New(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(plugin.Opts())

		do.ProvideNamed(injector, svcName, func(_ do.Injector) (string, error) {
			return "val", nil
		})

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

		// Verify the service name is HTML-escaped in data attributes.
		// The page has legitimate <script> blocks for the visualization JS,
		// so we check that the raw <script>alert string is not present.
		raw := "<script>alert"
		if strings.Contains(output, raw) {
			t.Errorf("unescaped %q in HTML output for service %q", raw, svcName)
		}
	})
}
