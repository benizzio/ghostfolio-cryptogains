// Package calculate defines structured calculation error helpers.
// Authored by: OpenCode
package calculate

import (
	"slices"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// diagnosticCalculationError carries calculate-owned persisted source data for
// runtime diagnostics without exposing sync models from the report model layer.
// Authored by: OpenCode
type diagnosticCalculationError struct {
	*reportmodel.CalculationError
	record *syncmodel.ActivityRecord
}

// As exposes the embedded calculation error for standard error matching.
// Authored by: OpenCode
func (e diagnosticCalculationError) As(target any) bool {
	if calcErrTarget, ok := target.(**reportmodel.CalculationError); ok {
		*calcErrTarget = e.CalculationError
		return true
	}

	return false
}

// newGroupCalculationError creates one structured calculation error from grouped asset context.
// Authored by: OpenCode
func newGroupCalculationError(kind reportmodel.CalculationErrorKind, group assetInputGroup, message string, cause error) error {
	return reportmodel.NewCalculationError(kind, message, "", group.DisplayLabel, cause)
}

// newRecordCalculationError creates one structured calculation error from a
// normalized synced activity record.
// Authored by: OpenCode
func newRecordCalculationError(kind reportmodel.CalculationErrorKind, record syncmodel.ActivityRecord, message string, cause error) error {
	return withPersistedActivityRecord(
		reportmodel.NewCalculationError(kind, message, strings.TrimSpace(record.SourceID), activityDisplayLabel(record), cause),
		&record,
	)
}

// newInputCalculationError creates one structured calculation error from a
// selected activity calculation input.
// Authored by: OpenCode
func newInputCalculationError(kind reportmodel.CalculationErrorKind, input reportmodel.ActivityCalculationInput, message string, cause error) error {
	return reportmodel.NewCalculationError(kind, message, strings.TrimSpace(input.SourceID), strings.TrimSpace(input.DisplayLabel), cause)
}

// withPersistedActivityRecord attaches the original synced activity to one
// calculation error for downstream report diagnostics.
// Authored by: OpenCode
func withPersistedActivityRecord(err *reportmodel.CalculationError, record *syncmodel.ActivityRecord) error {
	if err == nil || record == nil {
		return err
	}

	return diagnosticCalculationError{CalculationError: err, record: record}
}

// DiagnosticReportContext returns source-faithful context for report-failure
// diagnostics.
// Authored by: OpenCode
func (e diagnosticCalculationError) DiagnosticReportContext() syncmodel.DiagnosticContext {
	if e.CalculationError == nil {
		return syncmodel.DiagnosticContext{}
	}

	return syncmodel.DiagnosticContext{
		FailureDetail:           e.DiagnosticFailureDetail(),
		FailureCauseChain:       slices.Clone(e.DiagnosticFailureCauseChain()),
		OffendingActivityRecord: e.record,
	}
}
