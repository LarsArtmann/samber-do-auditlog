package auditlog_test

import (
	"errors"
	"fmt"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestClassify_AllSentinelsHaveFamily(t *testing.T) {
	t.Parallel()

	classifications := auditlog.ErrorClassifications()
	if len(classifications) == 0 {
		t.Fatal("ErrorClassifications() returned empty map")
	}

	for sentinel, expectedFamily := range classifications {
		err := sentinel
		family := expectedFamily

		t.Run(fmt.Sprintf("%T_%s", err, family), func(t *testing.T) {
			t.Parallel()

			gotFamily := errorfamily.Classify(err)
			if gotFamily != family {
				t.Errorf("Classify(%v) = %v, want %v", err, gotFamily, family)
			}

			gotExit := errorfamily.ExitCode(err)
			if gotExit == 0 {
				t.Errorf("ExitCode(%v) = 0, want non-zero", err)
			}
		})
	}
}

func TestClassify_CorruptionErrors(t *testing.T) {
	t.Parallel()

	classifications := auditlog.ErrorClassifications()

	for sentinel, family := range classifications {
		if family != errorfamily.Corruption {
			continue
		}

		err := sentinel

		t.Run(err.Error(), func(t *testing.T) {
			t.Parallel()

			if got := errorfamily.IsRetryable(err); got {
				t.Errorf("IsRetryable(%v) = true, want false (Corruption is never retryable)", err)
			}

			if got := errorfamily.ExitCode(err); got != 65 {
				t.Errorf("ExitCode(%v) = %d, want 65", err, got)
			}
		})
	}
}

func TestClassify_RejectionErrors(t *testing.T) {
	t.Parallel()

	classifications := auditlog.ErrorClassifications()

	for sentinel, family := range classifications {
		if family != errorfamily.Rejection {
			continue
		}

		err := sentinel

		t.Run(err.Error(), func(t *testing.T) {
			t.Parallel()

			if got := errorfamily.IsRetryable(err); got {
				t.Errorf("IsRetryable(%v) = true, want false (Rejection is never retryable)", err)
			}

			if got := errorfamily.ExitCode(err); got != 1 {
				t.Errorf("ExitCode(%v) = %d, want 1", err, got)
			}
		})
	}
}

func TestClassify_WrappedErrorPreservesClassification(t *testing.T) {
	t.Parallel()

	for sentinel, family := range auditlog.ErrorClassifications() {
		wrapped := fmt.Errorf("%w: extra context: %w", sentinel, errors.New("inner"))

		t.Run(family.String(), func(t *testing.T) {
			t.Parallel()

			got := errorfamily.Classify(wrapped)
			if got != family {
				t.Errorf("Classify(wrapped) = %v, want %v", got, family)
			}
		})
	}
}

func TestClassify_RegisterClassificationsWithCustomRegistry(t *testing.T) {
	t.Parallel()

	reg := errorfamily.NewRegistry()
	auditlog.RegisterClassifications(reg)

	for sentinel, expectedFamily := range auditlog.ErrorClassifications() {
		gotFamily := reg.Classify(sentinel)

		if gotFamily != expectedFamily {
			t.Errorf("Registry.Classify(%v) = %v, want %v", sentinel, gotFamily, expectedFamily)
		}
	}
}
