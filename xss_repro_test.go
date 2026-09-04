package auditlog_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestXSSReproOnload(t *testing.T) {
	input := "0000000 onload="
	plugin, err := auditlog.New(auditlog.Config{ContainerID: auditlog.ContainerID(input), Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	injector := do.NewWithOpts(plugin.Opts())
	do.Provide(injector, func(do.Injector) (string, error) { return "v", nil })
	_ = injector
	var buf bytes.Buffer
	if err := plugin.WriteHTML(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	idx := strings.Index(out, ` onload="`)
	for idx >= 0 {
		start := idx - 80
		if start < 0 {
			start = 0
		}
		t.Logf("RAW HIT at %d: ...%q...", idx, out[start:idx+20])
		idx = strings.Index(out[idx+1:], ` onload="`)
		if idx >= 0 {
			idx += strings.Index(out[:0], "") + 1
			break
		}
	}
	if !strings.Contains(out, ` onload="`) {
		t.Log("no raw hit found in this path")
	}
	htmlOnly := strings.ReplaceAll(strings.ReplaceAll(out, "htmlOnly := stripJSONScriptsForTest(t, out)#34;", "\x00"), "htmlOnly := stripJSONScriptsForTest(t, out)#39;", "\x00")
	t.Log("htmlOnly len", len(htmlOnly))
}
