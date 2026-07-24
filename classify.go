package auditlog

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// RegisterClassifications registers all auditlog sentinel errors into the
// provided registry with their behavioral [errorfamily.Family] classification.
//
// Consumers using a custom [errorfamily.Registry] (rather than the package-level
// [errorfamily.DefaultRegistry]) must call this to receive classification
// metadata. For the common case, auditlog's [init] already registers into
// [errorfamily.DefaultRegistry], so most consumers never need to call this.
func RegisterClassifications(reg *errorfamily.Registry) {
	reg.RegisterClassifications(ErrorClassifications())
}

// ErrorClassifications returns the canonical mapping of auditlog sentinel
// errors to their behavioral [errorfamily.Family].
//
// The mapping encodes domain knowledge that only auditlog owns: whether a
// given error is a data-integrity violation ([errorfamily.Corruption] — the
// report is structurally invalid), the caller's fault
// ([errorfamily.Rejection] — bad input or invalid operation), or a
// system-level failure ([errorfamily.Infrastructure] — not retryable).
func ErrorClassifications() map[error]errorfamily.Family {
	return map[error]errorfamily.Family{
		// Corruption — internal data integrity violations. The report or
		// event stream is structurally invalid; no caller action can fix it
		// short of regenerating the data.
		errReportEventCountMismatch:  errorfamily.Corruption,
		errReportServiceCountMismatch: errorfamily.Corruption,
		errReportScopeCountMismatch:   errorfamily.Corruption,
		errReportHealthCheckedMismatch: errorfamily.Corruption,
		errReportStatusDrift:          errorfamily.Corruption,
		errReportEmptyVersion:         errorfamily.Corruption,
		errUnknownEventType:           errorfamily.Corruption,
		errUnknownPhase:               errorfamily.Corruption,
		errReplayValidationFailed:     errorfamily.Corruption,

		// Rejection — bad caller input or invalid operation. The caller sent
		// empty data, invalid config, an unsupported format, or attempted an
		// impossible operation.
		errContainerIDPathSep:    errorfamily.Rejection,
		errMigrationEmptyInput:   errorfamily.Rejection,
		errMigrationMissingVersion: errorfamily.Rejection,
		errUnsupportedFormat:   errorfamily.Rejection,
	}
}

// init registers all sentinel error classifications into the
// [errorfamily.DefaultRegistry] so that consumers who import auditlog
// automatically get [errorfamily.Classify], [errorfamily.IsRetryable], and
// [errorfamily.ExitCode] on auditlog errors without any additional setup.
// This follows the standard Go driver-registration pattern (cf. database/sql,
// image codec registration). Consumers who prefer a separate registry should
// call [RegisterClassifications] with their own [errorfamily.Registry].
//
//nolint:gochecknoinits // Standard Go self-registration pattern.
func init() {
	RegisterClassifications(errorfamily.DefaultRegistry)
}
