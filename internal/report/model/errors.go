// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// CalculationErrorKind identifies one report-calculation failure branch that
// callers can classify without parsing free-form error strings.
// Authored by: OpenCode
type CalculationErrorKind string

const (
	// CalculationErrorKindInvalidRequest identifies invalid report-request input
	// detected before history replay begins.
	// Authored by: OpenCode
	CalculationErrorKindInvalidRequest CalculationErrorKind = "invalid_request"

	// CalculationErrorKindUnavailableReportYear identifies a requested year that
	// is not present in the synced protected cache metadata.
	// Authored by: OpenCode
	CalculationErrorKindUnavailableReportYear CalculationErrorKind = "unavailable_report_year"

	// CalculationErrorKindUnsupportedCostBasisMethod identifies a cost-basis
	// method that is known by the model layer but not yet implemented by the
	// calculator.
	// Authored by: OpenCode
	CalculationErrorKindUnsupportedCostBasisMethod CalculationErrorKind = "unsupported_cost_basis_method"

	// CalculationErrorKindActivityInput identifies one offending activity row that
	// could not supply the required calculation inputs safely.
	// Authored by: OpenCode
	CalculationErrorKindActivityInput CalculationErrorKind = "activity_input"

	// CalculationErrorKindBasisAllocation identifies one failure while replaying
	// holdings, basis, or liquidation allocation.
	// Authored by: OpenCode
	CalculationErrorKindBasisAllocation CalculationErrorKind = "basis_allocation"
)

// CalculationError stores one non-secret report-calculation failure together
// with optional offending-activity references that runtime and TUI code can
// surface directly to the user.
// Authored by: OpenCode
type CalculationError struct {
	kind         CalculationErrorKind
	message      string
	sourceID     string
	displayLabel string
	record       *syncmodel.ActivityRecord
	cause        error
}

// NewCalculationError creates one structured non-secret report-calculation
// error.
//
// Example:
//
//	err := model.NewCalculationError(
//		model.CalculationErrorKindActivityInput,
//		"could not choose one complete activity currency context",
//		"buy-1",
//		"BTC",
//		nil,
//	)
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func NewCalculationError(kind CalculationErrorKind, message string, sourceID string, displayLabel string, cause error) *CalculationError {
	var detail = strings.TrimSpace(message)
	if detail == "" && cause != nil {
		detail = strings.TrimSpace(cause.Error())
	}
	if detail == "" {
		detail = "unsupported report calculation"
	}

	return &CalculationError{
		kind:         kind,
		message:      detail,
		sourceID:     strings.TrimSpace(sourceID),
		displayLabel: strings.TrimSpace(displayLabel),
		cause:        cause,
	}
}

// WithPersistedActivityRecord attaches the original persisted activity record to
// one calculation error for downstream diagnostics.
// Authored by: OpenCode
func (e *CalculationError) WithPersistedActivityRecord(record *syncmodel.ActivityRecord) *CalculationError {
	if e == nil || record == nil {
		return e
	}

	e.record = record
	return e
}

// Error returns the non-secret user-visible calculation failure detail.
// Authored by: OpenCode
func (e *CalculationError) Error() string {
	if e == nil {
		return ""
	}

	var references []string
	if e.displayLabel != "" {
		references = append(references, fmt.Sprintf("asset %q", e.displayLabel))
	}
	if e.sourceID != "" {
		references = append(references, fmt.Sprintf("source %q", e.sourceID))
	}
	if len(references) == 0 {
		return e.message
	}

	return fmt.Sprintf("%s (%s)", e.message, strings.Join(references, ", "))
}

// Unwrap returns the underlying implementation cause when one exists.
// Authored by: OpenCode
func (e *CalculationError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

// Kind returns the stable calculation-error kind used for classification.
// Authored by: OpenCode
func (e *CalculationError) Kind() CalculationErrorKind {
	if e == nil {
		return ""
	}

	return e.kind
}

// SourceID returns the offending activity source identifier when the failure is
// tied to one specific activity row.
// Authored by: OpenCode
func (e *CalculationError) SourceID() string {
	if e == nil {
		return ""
	}

	return e.sourceID
}

// DisplayLabel returns the non-secret asset display label associated with the
// offending activity when one is available.
// Authored by: OpenCode
func (e *CalculationError) DisplayLabel() string {
	if e == nil {
		return ""
	}

	return e.displayLabel
}

// DiagnosticReportContext returns the source-faithful persisted-record context
// used by report-failure diagnostics.
// Authored by: OpenCode
func (e *CalculationError) DiagnosticReportContext() syncmodel.DiagnosticContext {
	if e == nil {
		return syncmodel.DiagnosticContext{}
	}

	return syncmodel.DiagnosticContext{
		FailureDetail:           e.Error(),
		OffendingActivityRecord: e.record,
	}
}
