package auditlog_test

import (
	"bytes"
	"os"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// TestSchemaNoDrift verifies the committed schema/report.schema.json matches
// the embedded JSONSchema() byte-for-byte. Catches drift when someone changes
// a Report type but forgets to run `go generate ./...`.
func TestSchemaNoDrift(t *testing.T) {
	t.Parallel()

	committed, err := os.ReadFile("schema/report.schema.json")
	if err != nil {
		t.Fatalf("read committed schema: %v", err)
	}

	embedded := auditlog.JSONSchema()

	if !bytes.Equal(committed, []byte(embedded)) {
		t.Errorf("schema drift: committed file does not match embedded schema\n"+
			"  committed: %d bytes\n"+
			"  embedded:  %d bytes\n"+
			"hint: run `go generate ./...` to regenerate", len(committed), len(embedded))
	}
}
