package auditlog_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestXSSReproServiceName(t *testing.T) {
	input := "0000000 onload="
	plugin, err := auditlog.New(auditlog.Config{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	injector := do.NewWithOpts(plugin.Opts())
	do.ProvideNamed(injector, input, func(do.Injector) (string, error) {
		return "v", nil
	})
	_, _ = do.InvokeNamed[string](injector, input)
	do.ProvideNamed(injector, "error-svc", func(_ do.Injector) (string, error) {
		return "", fmt.Errorf("%s", input)
	})
	_, _ = do.InvokeNamed[string](injector, "error-svc")
	var buf bytes.Buffer
	if err := plugin.WriteHTML(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	n := 0
	idx := 0
	for {
		idx = strings.Index(out[idx:], ` onload="`)
		if idx < 0 {
			break
		}
		n++
		start := idx - 100
		if start < 0 {
			start = 0
		}
		t.Logf("RAW #%d: %q", n, out[start:idx+30])
		idx++
	}
	t.Logf("total raw hits: %d", n)
}
