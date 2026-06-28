// Package calculate defines structured calculation error helpers.
// Authored by: OpenCode
package calculate

import (
	"errors"
	"slices"
	"strings"
	"time"

	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// diagnosticCalculationError carries calculate-owned persisted source data for
// runtime diagnostics without exposing sync models from the report model layer.
// Authored by: OpenCode
type diagnosticCalculationError struct {
	*reportmodel.CalculationError
	record *syncmodel.ActivityRecord
}

// ConversionFailureContext stores non-secret conversion failure fields for
// runtime copy without requiring runtime to parse formatted error strings.
// Authored by: OpenCode
type ConversionFailureContext struct {
	SourceID           string
	SourceCurrency     string
	ReportBaseCurrency string
	ActivityDate       time.Time
	Reason             string
	ProviderCategory   string
}

// ConversionFailureContextCarrier exposes typed conversion failure context from
// calculation errors.
// Authored by: OpenCode
type ConversionFailureContextCarrier interface {
	ReportConversionFailureContext() ConversionFailureContext
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
func newConversionLookupCalculationError(input reportmodel.ActivityCalculationInput, sourceCurrency string, baseCurrency string, fallbackMessage string, cause error) error {
	var failure *currencyintegration.ConversionFailure
	if errors.As(cause, &failure) && failure != nil {
		var context = conversionFailureContextFromIntegrationFailure(input, failure)
		return newInputCalculationError(
			reportmodel.CalculationErrorKindActivityInput,
			input,
			failure.SafeMessage(),
			conversionFailureContextCause{context: context, cause: safeConversionFailureCause{failure: failure}},
		)
	}
	if reason := conversionFailureReasonFromPrefix(cause); reason != "" {
		var context = ConversionFailureContext{
			SourceID:           strings.TrimSpace(input.SourceID),
			SourceCurrency:     strings.TrimSpace(sourceCurrency),
			ReportBaseCurrency: strings.TrimSpace(baseCurrency),
			ActivityDate:       datesupport.CalendarDate(input.OccurredAt),
			Reason:             string(reason),
		}
		return newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			safeConversionLookupMessage(fallbackMessage, reason),
			conversionFailureContextCause{context: context},
		)
	}

	return newInputCalculationError(
		reportmodel.CalculationErrorKindBasisAllocation,
		input,
		strings.TrimSpace(fallbackMessage),
		nil,
	)
}

// safeConversionLookupMessage carries a classified non-secret reason from a
// test seam or provider adapter without retaining raw provider detail.
// Authored by: OpenCode
func safeConversionLookupMessage(fallbackMessage string, reason currencyintegration.ConversionFailureReason) string {
	var message = strings.TrimSpace(fallbackMessage)
	if message == "" {
		return "currency conversion lookup failed: reason=" + string(reason)
	}

	return message + ": reason=" + string(reason)
}

// conversionFailureReasonFromPrefix recognizes stable conversion failure reason
// prefixes from legacy test seams and adapters that do not return typed failures.
// Authored by: OpenCode
func conversionFailureReasonFromPrefix(cause error) currencyintegration.ConversionFailureReason {
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
		return currencyintegration.ConversionFailureReason(strings.TrimSpace(prefix))
	default:
		return ""
	}
}

// newConversionPreparationCalculationError creates one activity-input error with
// typed conversion context for invalid lookup request construction.
// Authored by: OpenCode
func newConversionPreparationCalculationError(input reportmodel.ActivityCalculationInput, sourceCurrency string, baseCurrency string, message string, cause error) error {
	var context = ConversionFailureContext{
		SourceID:           strings.TrimSpace(input.SourceID),
		SourceCurrency:     strings.TrimSpace(sourceCurrency),
		ReportBaseCurrency: strings.TrimSpace(baseCurrency),
		ActivityDate:       datesupport.CalendarDate(input.OccurredAt),
		Reason:             string(currencyintegration.ConversionFailureReasonInvalidActivityCurrency),
	}

	return newInputCalculationError(
		reportmodel.CalculationErrorKindActivityInput,
		input,
		strings.TrimSpace(message),
		conversionFailureContextCause{context: context, cause: cause},
	)
}

// conversionFailureContextFromIntegrationFailure maps integration failure
// identity into a calculate-owned context value.
// Authored by: OpenCode
func conversionFailureContextFromIntegrationFailure(input reportmodel.ActivityCalculationInput, failure *currencyintegration.ConversionFailure) ConversionFailureContext {
	return ConversionFailureContext{
		SourceID:           strings.TrimSpace(input.SourceID),
		SourceCurrency:     strings.TrimSpace(failure.SourceCurrency),
		ReportBaseCurrency: strings.TrimSpace(failure.ReportBaseCurrency),
		ActivityDate:       datesupport.CalendarDate(failure.ActivityDate),
		Reason:             string(failure.Reason),
		ProviderCategory:   string(failure.ProviderID),
	}
}

// conversionFailureContextCause carries typed conversion context through the
// calculation error cause chain.
// Authored by: OpenCode
type conversionFailureContextCause struct {
	context ConversionFailureContext
	cause   error
}

// Error returns the wrapped safe cause text.
// Authored by: OpenCode
func (cause conversionFailureContextCause) Error() string {
	if cause.cause == nil {
		return "conversion failed"
	}

	return cause.cause.Error()
}

// Unwrap exposes the wrapped safe cause for existing classification tests.
// Authored by: OpenCode
func (cause conversionFailureContextCause) Unwrap() error {
	return cause.cause
}

// ReportConversionFailureContext returns non-secret conversion failure fields.
// Authored by: OpenCode
func (cause conversionFailureContextCause) ReportConversionFailureContext() ConversionFailureContext {
	return cause.context
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
