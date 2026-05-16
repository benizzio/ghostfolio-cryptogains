// Package validate defines the normalized activity-history validation boundary.
// Authored by: OpenCode
package validate

import (
	"fmt"
	"strings"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// ValidationError stores offending-record context for synced-data validation
// failures.
// Authored by: OpenCode
type ValidationError struct {
	message string
	context syncmodel.DiagnosticContext
}

// Error returns the non-secret validation failure detail.
// Authored by: OpenCode
func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

// DiagnosticContext returns the structured troubleshooting context for one validation failure.
// Authored by: OpenCode
func (e *ValidationError) DiagnosticContext() syncmodel.DiagnosticContext {
	if e == nil {
		return syncmodel.DiagnosticContext{}
	}
	return e.context
}

// Validator defines the normalized activity-history validation contract used by
// the sync workflow.
//
// Example:
//
//	var validator Validator
//	_ = validator.Validate(syncmodel.ProtectedActivityCache{})
//
// Implementations are expected to reject unsupported activity histories and
// preserve only defensible normalized cache data.
// Authored by: OpenCode
type Validator interface {
	Validate(syncmodel.ProtectedActivityCache) error
}

// defaultValidator enforces the currently supported activity-history rules.
// Authored by: OpenCode
type defaultValidator struct{}

// NewValidator creates the foundational validation service used by runtime
// wiring.
//
// Example:
//
//	validator := validate.NewValidator()
//	_ = validator.Validate(syncmodel.ProtectedActivityCache{})
//
// Authored by: OpenCode
func NewValidator() Validator {
	return defaultValidator{}
}

// Validate enforces the phase-3 supported-history rules for BUY and SELL data.
// Authored by: OpenCode
func (defaultValidator) Validate(cache syncmodel.ProtectedActivityCache) error {
	if cache.ActivityCount != len(cache.Activities) {
		return newValidationError("activity_count does not match the normalized activity list")
	}

	var zero apd.Decimal
	var state = validationState{runningQuantityByAsset: map[string]apd.Decimal{}}

	for _, record := range cache.Activities {
		var currentKey, err = validateActivityOrdering(record, state)
		if err != nil {
			return err
		}
		if err := validateActivityFields(record, &zero); err != nil {
			return err
		}
		if err := applyRunningQuantity(record, state.runningQuantityByAsset, &zero); err != nil {
			return err
		}

		state.previousKey = currentKey
		state.previousRecord = record
		state.havePrevious = true
	}

	return nil
}

// validationState keeps the rolling ordering and holdings state needed while validating one normalized cache.
// Authored by: OpenCode
type validationState struct {
	previousKey            syncmodel.ActivityOrderingKey
	previousRecord         syncmodel.ActivityRecord
	havePrevious           bool
	runningQuantityByAsset map[string]apd.Decimal
}

// validateActivityOrdering parses the ordering key for one record and rejects chronology or ambiguity violations.
// Authored by: OpenCode
func validateActivityOrdering(record syncmodel.ActivityRecord, state validationState) (syncmodel.ActivityOrderingKey, error) {
	if strings.TrimSpace(record.OccurredAt) == "" {
		return syncmodel.ActivityOrderingKey{}, newValidationError("normalized activity timestamp is incomplete", record)
	}

	var currentKey, _, err = syncmodel.NewActivityOrderingKeyFromRecord(record)
	if err != nil {
		return syncmodel.ActivityOrderingKey{}, newValidationError(fmt.Sprintf("activity %q has an unreadable timestamp: %v", record.SourceID, err), record)
	}
	if !state.havePrevious {
		return currentKey, nil
	}
	if syncmodel.CompareActivityOrdering(currentKey, state.previousKey) < 0 {
		return syncmodel.ActivityOrderingKey{}, newValidationError("normalized activities are not ordered chronologically", state.previousRecord, record)
	}
	if syncmodel.HasAmbiguousActivityOrdering(state.previousKey, currentKey) {
		return syncmodel.ActivityOrderingKey{}, newValidationError(
			fmt.Sprintf("supported activity ordering is ambiguous for source %q", record.SourceID),
			state.previousRecord,
			record,
		)
	}

	return currentKey, nil
}

// validateActivityFields enforces identity, amount, and supported-activity rules for one normalized record.
// Authored by: OpenCode
func validateActivityFields(record syncmodel.ActivityRecord, zero *apd.Decimal) error {
	if strings.TrimSpace(record.SourceID) == "" || strings.TrimSpace(record.AssetSymbol) == "" {
		return newValidationError("normalized activity identity is incomplete", record)
	}
	if record.Quantity.Cmp(zero) <= 0 {
		return newValidationError(fmt.Sprintf("activity %q quantity must be greater than zero", record.SourceID), record)
	}
	if record.GrossValue.Cmp(zero) < 0 {
		return newValidationError(fmt.Sprintf("activity %q gross value cannot be negative", record.SourceID), record)
	}
	if record.FeeAmount != nil && record.FeeAmount.Cmp(zero) < 0 {
		return newValidationError(fmt.Sprintf("activity %q fee amount cannot be negative", record.SourceID), record)
	}

	return validateActivityType(record, zero)
}

// validateActivityType enforces the supported BUY and SELL price rules for one normalized record.
// Authored by: OpenCode
func validateActivityType(record syncmodel.ActivityRecord, zero *apd.Decimal) error {
	switch record.ActivityType {
	case syncmodel.ActivityTypeBuy:
		if record.UnitPrice.Cmp(zero) <= 0 {
			return newValidationError(fmt.Sprintf("BUY activity %q must have a unit price greater than zero", record.SourceID), record)
		}
	case syncmodel.ActivityTypeSell:
		if record.UnitPrice.Cmp(zero) < 0 {
			return newValidationError(fmt.Sprintf("SELL activity %q cannot have a negative unit price", record.SourceID), record)
		}
		if record.UnitPrice.Cmp(zero) == 0 && strings.TrimSpace(record.Comment) == "" {
			return newValidationError(fmt.Sprintf("SELL activity %q requires a comment when unit price is zero", record.SourceID), record)
		}
	default:
		return newValidationError(fmt.Sprintf("activity %q uses unsupported type %q", record.SourceID, record.ActivityType), record)
	}

	return nil
}

// applyRunningQuantity replays one record into the per-asset holdings timeline and rejects negative holdings.
// Authored by: OpenCode
func applyRunningQuantity(record syncmodel.ActivityRecord, runningQuantityByAsset map[string]apd.Decimal, zero *apd.Decimal) error {
	var runningQuantity = runningQuantityByAsset[record.AssetSymbol]
	switch record.ActivityType {
	case syncmodel.ActivityTypeBuy:
		_, _ = apd.BaseContext.Add(&runningQuantity, &runningQuantity, &record.Quantity)
	case syncmodel.ActivityTypeSell:
		_, _ = apd.BaseContext.Sub(&runningQuantity, &runningQuantity, &record.Quantity)
		if runningQuantity.Cmp(zero) < 0 {
			return newValidationError(fmt.Sprintf("activity %q would drive holdings below zero", record.SourceID), record)
		}
	}
	runningQuantityByAsset[record.AssetSymbol] = runningQuantity

	return nil
}

// newValidationError captures the offending normalized records for one validation failure.
// Authored by: OpenCode
func newValidationError(message string, records ...syncmodel.ActivityRecord) error {
	var diagnosticRecords = make([]syncmodel.DiagnosticRecord, 0, len(records))
	for _, record := range records {
		diagnosticRecords = append(diagnosticRecords, syncmodel.DiagnosticRecordFromActivityRecord(record))
	}

	return &ValidationError{
		message: message,
		context: syncmodel.DiagnosticContext{
			FailureStage:  syncmodel.DiagnosticFailureStageValidation,
			FailureDetail: message,
			Records:       diagnosticRecords,
		},
	}
}
