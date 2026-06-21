package auditlog_test

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestBuildTypeMetadata(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	if len(meta.Providers) != 4 {
		t.Errorf("providers: want 4 entries, got %d", len(meta.Providers))
	}

	wantProviders := map[string]struct {
		icon  string
		label string
	}{
		"lazy":      {icon: "\U0001F634", label: "Lazy"},
		"eager":     {icon: "\U0001F501", label: "Eager"},
		"transient": {icon: "\U0001F3ED", label: "Transient"},
		"alias":     {icon: "\U0001F517", label: "Alias"},
	}

	for name, want := range wantProviders {
		got, ok := meta.Providers[name]
		if !ok {
			t.Errorf("providers: missing %q", name)

			continue
		}

		if got.Icon != want.icon {
			t.Errorf("providers[%q].Icon: want %q, got %q", name, want.icon, got.Icon)
		}

		assertMetadataLabel(t, got.Label, want.label, "providers", name)
	}

	if len(meta.Statuses) != 5 {
		t.Errorf("statuses: want 5 entries, got %d", len(meta.Statuses))
	}

	wantStatuses := map[string]string{
		"registered":       "\u26AA",
		"active":           "\U0001F7E2",
		"shutdown":         "\U0001F535",
		"invocation_error": "\U0001F534",
		"shutdown_error":   "\U0001F534",
	}

	for name, wantIcon := range wantStatuses {
		got, ok := meta.Statuses[name]
		if !ok {
			t.Errorf("statuses: missing %q", name)

			continue
		}

		if got.Icon != wantIcon {
			t.Errorf("statuses[%q].Icon: want %q, got %q", name, wantIcon, got.Icon)
		}
	}

	if len(meta.Events) != 4 {
		t.Errorf("events: want 4 entries, got %d", len(meta.Events))
	}

	wantEvents := map[string]struct {
		label string
		color string
	}{
		"registration": {label: "Registration", color: "var(--accent)"},
		"invocation":   {label: "Invocation", color: "var(--success)"},
		"shutdown":     {label: "Shutdown", color: "var(--warning)"},
		"health_check": {label: "Health", color: "var(--info)"},
	}

	for name, want := range wantEvents {
		got, ok := meta.Events[name]
		if !ok {
			t.Errorf("events: missing %q", name)

			continue
		}

		assertMetadataLabel(t, got.Label, want.label, "events", name)

		if got.Color != want.color {
			t.Errorf("events[%q].Color: want %q, got %q", name, want.color, got.Color)
		}
	}
}

// assertMetadataLabel fails the test if got does not equal want. Used by the
// metadata builder tests, which all assert Label against a table of fixtures.
func assertMetadataLabel(t *testing.T, got, want, collection, name string) {
	t.Helper()

	if got != want {
		t.Errorf("%s[%q].Label: want %q, got %q", collection, name, want, got)
	}
}
