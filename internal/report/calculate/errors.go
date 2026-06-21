// Package calculate defines structured calculation error helpers.
// Authored by: OpenCode
package calculate

import (
	"errors"
	"slices"
	"strings"

	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
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

// newConversionLookupCalculationError maps a classified integration conversion
// failure into the report calculation error surface without provider DTOs or raw
// provider detail.
// Authored by: OpenCode
func newConversionLookupCalculationError(input reportmodel.ActivityCalculationInput, fallbackMessage string, cause error) error {
	var failure *currencyintegration.ConversionFailure
	if errors.As(cause, &failure) && failure != nil {
		return newInputCalculationError(
			reportmodel.CalculationErrorKindActivityInput,
			input,
			failure.SafeMessage(),
			safeConversionFailureCause{failure: failure},
		)
	}

	return newInputCalculationError(
		reportmodel.CalculationErrorKindBasisAllocation,
		input,
		safeConversionLookupFallbackMessage(fallbackMessage, cause),
		nil,
	)
}

// safeConversionLookupFallbackMessage carries a known non-secret conversion
// reason from unclassified lookup errors without raw provider detail.
// Authored by: OpenCode
func safeConversionLookupFallbackMessage(fallbackMessage string, cause error) string {
	var message = strings.TrimSpace(fallbackMessage)
	var reason = safeConversionReasonPrefix(cause)
	if reason == "" {
		return message
	}
	if message == "" {
		return "currency conversion lookup failed: reason=" + reason
	}

	return message + ": reason=" + reason
}

// safeConversionReasonPrefix extracts only stable known conversion reason
// tokens from unclassified lookup errors.
// Authored by: OpenCode
func safeConversionReasonPrefix(cause error) string {
	if cause == nil {
		return ""
	}

	var detail = strings.TrimSpace(cause.Error())
	var prefix, _, _ = strings.Cut(detail, ":")
	switch currencyintegration.ConversionFailureReason(strings.TrimSpace(prefix)) {
	case currencyintegration.ConversionFailureReasonUnsupportedCurrency,
		currencyintegration.ConversionFailureReasonMissingRate,
		currencyintegration.ConversionFailureReasonProviderUnavailable,
		currencyintegration.ConversionFailureReasonMalformedRate,
		currencyintegration.ConversionFailureReasonAmbiguousQuote,
		currencyintegration.ConversionFailureReasonInvalidActivityCurrency,
		currencyintegration.ConversionFailureReasonAuthorityMismatch:
		return strings.TrimSpace(prefix)
	default:
		return ""
	}
}

// safeConversionFailureCause preserves classified conversion-failure matching
// while stopping raw provider detail before diagnostic cause-chain construction.
// Authored by: OpenCode
type safeConversionFailureCause struct {
	failure *currencyintegration.ConversionFailure
}

// Error returns the classified non-secret conversion failure message.
// Authored by: OpenCode
func (cause safeConversionFailureCause) Error() string {
	if cause.failure == nil {
		return "conversion failed"
	}

	return cause.failure.SafeMessage()
}

// As exposes the original classified failure for reason extraction.
// Authored by: OpenCode
func (cause safeConversionFailureCause) As(target any) bool {
	if failureTarget, ok := target.(**currencyintegration.ConversionFailure); ok {
		*failureTarget = cause.failure
		return true
	}

	return false
}

// withPersistedActivityRecord attaches the original synced activity to one
// calculation error for downstream report diagnostics.
// Authored by: OpenCode
func withPersistedActivityRecord(err *reportmodel.CalculationError, record *syncmodel.ActivityRecord) error {
	if err == nil {
		return nil
	}
	if record == nil {
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
