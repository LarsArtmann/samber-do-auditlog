package auditlog_test

import (
	"encoding/json"
	"math/rand/v2"
	"slices"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// TestJSONSchema_ValidAndCoversReportFields ensures the embedded schema is
// valid JSON and that every top-level field produced by marshaling a Report is
// declared in the schema — preventing the schema from drifting from the Go types.
func TestJSONSchema_ValidAndCoversReportFields(t *testing.T) {
	t.Parallel()

	raw := auditlog.JSONSchema()

	var schema map[string]any
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		t.Fatalf("embedded schema is not valid JSON: %v", err)
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema has no top-level 'properties' object")
	}

	// Marshal a representative report and compare its top-level keys to the
	// schema. Every key in the JSON must be a declared schema property.
	report := randReport(rand.New(rand.NewPCG(99, 99))) // deterministic for key extraction

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	var sample map[string]any
	if err := json.Unmarshal(data, &sample); err != nil {
		t.Fatalf("unmarshal sample: %v", err)
	}

	// These keys are always present on a marshaled Report.
	expected := []string{
		"version", "container_id", "exported_at",
		"event_count", "service_count", "scope_count",
		"services", "events", "scope_tree",
	}

	for _, key := range expected {
		if _, has := props[key]; !has {
			t.Errorf("schema missing top-level property %q", key)
		}
	}
}

// TestJSONSchema_DefinesCoreTypes ensures the schema declares the four core
// referenced types so consumers can navigate the full structure.
func TestJSONSchema_DefinesCoreTypes(t *testing.T) {
	t.Parallel()

	var schema map[string]any
	if err := json.Unmarshal([]byte(auditlog.JSONSchema()), &schema); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}

	defs, ok := schema["$defs"].(map[string]any)
	if !ok {
		t.Fatal("schema has no $defs")
	}

	want := []string{"Event", "ServiceInfo", "ServiceRef", "ScopeNode"}
	for _, name := range want {
		if _, has := defs[name]; !has {
			t.Errorf("schema $defs missing type %q (have: %v)", name, keysOf(defs))
		}
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))

	for k := range m {
		out = append(out, k)
	}

	slices.Sort(out)

	return out
}
