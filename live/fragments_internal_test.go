package live

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestHumanizeDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ms   float64
		want string
	}{
		{"negative", -1, "&mdash;"},
		{"sub-millisecond", 0.5, "0.500ms"},
		{"millisecond", 5, "5.0ms"},
		{"sub-second", 500, "500.0ms"},
		{"seconds", 2500, "2.5s"},
		{"minutes", 125000, "2m 5s"},
		{"hours", 3900000, "1h 5m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := humanizeDuration(tt.ms)
			if got != tt.want {
				t.Errorf("humanizeDuration(%v) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	t.Parallel()

	if got := truncateString("hello", 10); got != "hello" {
		t.Errorf("truncateString short = %q", got)
	}

	if got := truncateString("hello world", 5); got != "hello" {
		t.Errorf("truncateString long = %q", got)
	}
}

func TestHealthLabel(t *testing.T) {
	t.Parallel()

	if got := healthLabel(true); got != "Pass" {
		t.Errorf("healthLabel(true) = %q, want Pass", got)
	}

	if got := healthLabel(false); got != "Fail" {
		t.Errorf("healthLabel(false) = %q, want Fail", got)
	}
}

func TestErrorCountClass(t *testing.T) {
	t.Parallel()

	if got := errorCountClass(0); got != "success" {
		t.Errorf("errorCountClass(0) = %q, want success", got)
	}

	if got := errorCountClass(5); got != "error" {
		t.Errorf("errorCountClass(5) = %q, want error", got)
	}
}

func TestDepNamesString(t *testing.T) {
	t.Parallel()

	deps := []auditlog.ServiceRef{
		{ServiceName: "db"},
		{ServiceName: "cache"},
	}

	if got := depNamesString(deps); got != "db, cache" {
		t.Errorf("depNamesString = %q, want 'db, cache'", got)
	}

	if got := depNamesString(nil); got != "&mdash;" {
		t.Errorf("depNamesString(nil) = %q, want &mdash;", got)
	}
}

func TestSafeDuration(t *testing.T) {
	t.Parallel()

	if got := safeDuration(nil); got != 0 {
		t.Errorf("safeDuration(nil) = %v, want 0", got)
	}

	val := 42.5
	if got := safeDuration(&val); got != 42.5 {
		t.Errorf("safeDuration(&42.5) = %v, want 42.5", got)
	}
}

func TestTimelineBarWidth(t *testing.T) {
	t.Parallel()

	if got := timelineBarWidth(nil, 100); got != "0%" {
		t.Errorf("timelineBarWidth(nil) = %q, want 0%%", got)
	}

	val := 50.0
	if got := timelineBarWidth(&val, 100); got != "50.0%" {
		t.Errorf("timelineBarWidth(&50, 100) = %q, want 50.0%%", got)
	}

	if got := timelineBarWidth(&val, 0); got != "0%" {
		t.Errorf("timelineBarWidth(&50, 0) = %q, want 0%%", got)
	}
}

func TestScopeNodeName(t *testing.T) {
	t.Parallel()

	if got := scopeNodeName(auditlog.ScopeNode{Name: "test"}); got != "test" {
		t.Errorf("scopeNodeName with name = %q", got)
	}

	if got := scopeNodeName(auditlog.ScopeNode{ID: "scope-1"}); got != "scope-1" {
		t.Errorf("scopeNodeName with ID = %q", got)
	}

	if got := scopeNodeName(auditlog.ScopeNode{}); got != "scope" {
		t.Errorf("scopeNodeName empty = %q", got)
	}
}

func TestFooterVersion(t *testing.T) {
	t.Parallel()

	if got := footerVersion(auditlog.Report{Version: "1.0"}); got != "1.0" {
		t.Errorf("footerVersion with version = %q", got)
	}

	if got := footerVersion(auditlog.Report{}); got != "?" {
		t.Errorf("footerVersion empty = %q", got)
	}
}

func TestComputeWaveformMarks(t *testing.T) {
	t.Parallel()

	marks := computeWaveformMarks(nil, auditlog.TypeMetadata{})
	if len(marks) != 0 {
		t.Errorf("computeWaveformMarks(nil) returned %d marks, want 0", len(marks))
	}
}
