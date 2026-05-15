// Package validate defines the normalized activity-history validation boundary.
// Authored by: OpenCode
package validate

import (
	"fmt"
	"strings"
	"time"

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
	var previousKey validationOrderKey
	var previousRecord syncmodel.ActivityRecord
	var havePrevious bool
	var runningQuantityByAsset = map[string]apd.Decimal{}

	for _, record := range cache.Activities {
		occurredAt, err := time.Parse(time.RFC3339Nano, record.OccurredAt)
		if err != nil {
			return newValidationError(fmt.Sprintf("activity %q has an unreadable timestamp: %v", record.SourceID, err), record)
		}
		var currentKey = validationOrderKey{
			SourceDate:    occurredAt.Format("2006-01-02"),
			AssetSymbol:   record.AssetSymbol,
			ActivityOrder: validationActivityTypeOrder(record.ActivityType),
			SourceID:      record.SourceID,
			OccurredAt:    record.OccurredAt,
			RawHash:       record.RawHash,
		}
		if havePrevious {
			if compareValidationOrderKeys(currentKey, previousKey) < 0 {
				return newValidationError("normalized activities are not ordered chronologically", previousRecord, record)
			}
			if previousKey.SourceDate == currentKey.SourceDate &&
				previousKey.AssetSymbol == currentKey.AssetSymbol &&
				previousRecord.ActivityType == record.ActivityType &&
				previousKey.SourceID == currentKey.SourceID &&
				previousKey.RawHash != currentKey.RawHash {
				return newValidationError(
					fmt.Sprintf("supported activity ordering is ambiguous for source %q", record.SourceID),
					previousRecord,
					record,
				)
			}
		}

		if strings.TrimSpace(record.SourceID) == "" || strings.TrimSpace(record.AssetSymbol) == "" {
			return newValidationError("normalized activity identity is incomplete", record)
		}
		if strings.TrimSpace(record.OccurredAt) == "" {
			return newValidationError("normalized activity timestamp is incomplete", record)
		}
		if record.Quantity.Cmp(&zero) <= 0 {
			return newValidationError(fmt.Sprintf("activity %q quantity must be greater than zero", record.SourceID), record)
		}
		if record.GrossValue.Cmp(&zero) < 0 {
			return newValidationError(fmt.Sprintf("activity %q gross value cannot be negative", record.SourceID), record)
		}
		if record.FeeAmount != nil && record.FeeAmount.Cmp(&zero) < 0 {
			return newValidationError(fmt.Sprintf("activity %q fee amount cannot be negative", record.SourceID), record)
		}

		switch record.ActivityType {
		case syncmodel.ActivityTypeBuy:
			if record.UnitPrice.Cmp(&zero) <= 0 {
				return newValidationError(fmt.Sprintf("BUY activity %q must have a unit price greater than zero", record.SourceID), record)
			}
		case syncmodel.ActivityTypeSell:
			if record.UnitPrice.Cmp(&zero) < 0 {
				return newValidationError(fmt.Sprintf("SELL activity %q cannot have a negative unit price", record.SourceID), record)
			}
			if record.UnitPrice.Cmp(&zero) == 0 && strings.TrimSpace(record.Comment) == "" {
				return newValidationError(fmt.Sprintf("SELL activity %q requires a comment when unit price is zero", record.SourceID), record)
			}
		default:
			return newValidationError(fmt.Sprintf("activity %q uses unsupported type %q", record.SourceID, record.ActivityType), record)
		}

		var runningQuantity = runningQuantityByAsset[record.AssetSymbol]
		switch record.ActivityType {
		case syncmodel.ActivityTypeBuy:
			_, _ = apd.BaseContext.Add(&runningQuantity, &runningQuantity, &record.Quantity)
		case syncmodel.ActivityTypeSell:
			_, _ = apd.BaseContext.Sub(&runningQuantity, &runningQuantity, &record.Quantity)
			if runningQuantity.Cmp(&zero) < 0 {
				return newValidationError(fmt.Sprintf("activity %q would drive holdings below zero", record.SourceID), record)
			}
		}
		runningQuantityByAsset[record.AssetSymbol] = runningQuantity

		previousKey = currentKey
		previousRecord = record
		havePrevious = true
	}

	return nil
}

// validationOrderKey stores the deterministic ordering tuple used by validation.
// Authored by: OpenCode
type validationOrderKey struct {
	SourceDate    string
	AssetSymbol   string
	ActivityOrder int
	SourceID      string
	OccurredAt    string
	RawHash       string
}

// compareValidationOrderKeys compares two normalized ordering tuples.
// Authored by: OpenCode
func compareValidationOrderKeys(left validationOrderKey, right validationOrderKey) int {
	if comparison := strings.Compare(left.SourceDate, right.SourceDate); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.AssetSymbol, right.AssetSymbol); comparison != 0 {
		return comparison
	}
	if left.ActivityOrder != right.ActivityOrder {
		if left.ActivityOrder < right.ActivityOrder {
			return -1
		}
		return 1
	}
	if comparison := strings.Compare(left.SourceID, right.SourceID); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.OccurredAt, right.OccurredAt); comparison != 0 {
		return comparison
	}

	return strings.Compare(left.RawHash, right.RawHash)
}

// validationActivityTypeOrder ranks activity types for same-asset same-day replay checks.
// Authored by: OpenCode
func validationActivityTypeOrder(activityType syncmodel.ActivityType) int {
	switch activityType {
	case syncmodel.ActivityTypeBuy:
		return 0
	case syncmodel.ActivityTypeSell:
		return 1
	default:
		return 2
	}
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
