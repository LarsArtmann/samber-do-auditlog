package auditlog

import (
	"testing"
	"time"
)

// strPtr returns a pointer to s, or nil when s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

// TestDeriveServiceStatus_AllTransitions exhaustively covers every combination
// of the four inputs (2^4 = 16 cases) and asserts the priority ordering:
// invocation_error > shutdown_error > shutdown > active > registered.
func TestDeriveServiceStatus_AllTransitions(t *testing.T) {
	t.Parallel()

	someTime := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	type input struct {
		invocationError *string
		shutdownError   *string
		shutdownAt      *time.Time
		firstInvokedAt  *time.Time
	}

	// Build the exhaustive 16-case matrix. Each field is either set or nil.
	cases := []struct {
		name  string
		input input
		want  ServiceStatus
	}{
		// --- Single-cause states (5 base cases) ---
		{name: "all nil -> registered", input: input{}, want: ServiceStatusRegistered},
		{name: "only invoked -> active", input: input{firstInvokedAt: &someTime}, want: ServiceStatusActive},
		{name: "only shutdownAt -> shutdown", input: input{shutdownAt: &someTime}, want: ServiceStatusShutdown},
		{
			name:  "only shutdownError -> shutdown_error",
			input: input{shutdownError: strPtr("boom")},
			want:  ServiceStatusShutdownError,
		},
		{
			name:  "only invocationError -> invocation_error",
			input: input{invocationError: strPtr("fail")},
			want:  ServiceStatusInvocationError,
		},

		// --- Two-cause priority conflicts (6 cases) ---
		{
			name:  "invoked+shutdownAt -> shutdown",
			input: input{shutdownAt: &someTime, firstInvokedAt: &someTime},
			want:  ServiceStatusShutdown,
		},
		{
			name:  "invoked+shutdownError -> shutdown_error",
			input: input{shutdownError: strPtr("e"), firstInvokedAt: &someTime},
			want:  ServiceStatusShutdownError,
		},
		{
			name:  "invoked+invocationError -> invocation_error",
			input: input{invocationError: strPtr("e"), firstInvokedAt: &someTime},
			want:  ServiceStatusInvocationError,
		},
		{
			name:  "shutdownAt+shutdownError -> shutdown_error",
			input: input{shutdownError: strPtr("e"), shutdownAt: &someTime},
			want:  ServiceStatusShutdownError,
		},
		{
			name:  "shutdownAt+invocationError -> invocation_error",
			input: input{invocationError: strPtr("e"), shutdownAt: &someTime},
			want:  ServiceStatusInvocationError,
		},
		{
			name:  "shutdownError+invocationError -> invocation_error",
			input: input{invocationError: strPtr("e"), shutdownError: strPtr("e")},
			want:  ServiceStatusInvocationError,
		},

		// --- Three-cause priority conflicts (4 cases) ---
		{
			name:  "invoked+shutdownAt+shutdownError -> shutdown_error",
			input: input{shutdownError: strPtr("e"), shutdownAt: &someTime, firstInvokedAt: &someTime},
			want:  ServiceStatusShutdownError,
		},
		{
			name:  "invoked+shutdownAt+invocationError -> invocation_error",
			input: input{invocationError: strPtr("e"), shutdownAt: &someTime, firstInvokedAt: &someTime},
			want:  ServiceStatusInvocationError,
		},
		{
			name:  "invoked+shutdownError+invocationError -> invocation_error",
			input: input{invocationError: strPtr("e"), shutdownError: strPtr("e"), firstInvokedAt: &someTime},
			want:  ServiceStatusInvocationError,
		},
		{
			name:  "shutdownAt+shutdownError+invocationError -> invocation_error",
			input: input{invocationError: strPtr("e"), shutdownError: strPtr("e"), shutdownAt: &someTime},
			want:  ServiceStatusInvocationError,
		},

		// --- All four causes (1 case) ---
		{
			name: "all four set -> invocation_error",
			input: input{
				invocationError: strPtr("e"),
				shutdownError:   strPtr("e"),
				shutdownAt:      &someTime,
				firstInvokedAt:  &someTime,
			},
			want: ServiceStatusInvocationError,
		},
	}

	if len(cases) != 16 {
		t.Fatalf("expected 16 exhaustive cases, defined %d", len(cases))
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := deriveServiceStatus(
				tc.input.invocationError,
				tc.input.shutdownError,
				tc.input.shutdownAt,
				tc.input.firstInvokedAt,
			)
			if got != tc.want {
				t.Errorf("deriveServiceStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDeriveServiceStatus_PriorityOrdering verifies that each priority level
// dominates every lower level regardless of how many lower signals are present.
func TestDeriveServiceStatus_PriorityOrdering(t *testing.T) {
	t.Parallel()

	now := time.Now()
	errVal := "error"

	// invocation_error must dominate all three lower signals.
	got := deriveServiceStatus(&errVal, &errVal, &now, &now)
	if got != ServiceStatusInvocationError {
		t.Fatalf("invocation_error must win over all, got %q", got)
	}

	// shutdown_error must dominate shutdown + active (but not invocation_error).
	got = deriveServiceStatus(nil, &errVal, &now, &now)
	if got != ServiceStatusShutdownError {
		t.Fatalf("shutdown_error must win over shutdown+active, got %q", got)
	}

	// shutdown must dominate active.
	got = deriveServiceStatus(nil, nil, &now, &now)
	if got != ServiceStatusShutdown {
		t.Fatalf("shutdown must win over active, got %q", got)
	}

	// active must dominate registered.
	got = deriveServiceStatus(nil, nil, nil, &now)
	if got != ServiceStatusActive {
		t.Fatalf("active must win over registered, got %q", got)
	}
}

// TestDeriveServiceStatus_NilPointerSemantics verifies that a non-nil pointer
// to an empty string is still treated as "set" (the function checks nil, not
// string emptiness).
func TestDeriveServiceStatus_NilPointerSemantics(t *testing.T) {
	t.Parallel()

	empty := ""

	// A non-nil pointer to "" still triggers the error state.
	got := deriveServiceStatus(&empty, nil, nil, nil)
	if got != ServiceStatusInvocationError {
		t.Fatalf("non-nil ptr to empty string should be invocation_error, got %q", got)
	}
}
