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
		return fmt.Errorf("activity_count does not match the normalized activity list")
	}

	var zero apd.Decimal
	var previousOccurredAt time.Time
	var previousSourceID string
	var runningQuantityByAsset = map[string]apd.Decimal{}

	for index, record := range cache.Activities {
		occurredAt, err := time.Parse(time.RFC3339Nano, record.OccurredAt)
		if err != nil {
			return fmt.Errorf("activity %q has an unreadable timestamp: %w", record.SourceID, err)
		}
		if index > 0 {
			if occurredAt.Before(previousOccurredAt) || (occurredAt.Equal(previousOccurredAt) && record.SourceID < previousSourceID) {
				return fmt.Errorf("normalized activities are not ordered chronologically")
			}
		}

		if strings.TrimSpace(record.SourceID) == "" || strings.TrimSpace(record.AssetSymbol) == "" {
			return fmt.Errorf("normalized activity identity is incomplete")
		}
		if strings.TrimSpace(record.OccurredAt) == "" {
			return fmt.Errorf("normalized activity timestamp is incomplete")
		}
		if record.Quantity.Cmp(&zero) <= 0 {
			return fmt.Errorf("activity %q quantity must be greater than zero", record.SourceID)
		}
		if record.GrossValue.Cmp(&zero) < 0 {
			return fmt.Errorf("activity %q gross value cannot be negative", record.SourceID)
		}
		if record.FeeAmount != nil && record.FeeAmount.Cmp(&zero) < 0 {
			return fmt.Errorf("activity %q fee amount cannot be negative", record.SourceID)
		}

		switch record.ActivityType {
		case syncmodel.ActivityTypeBuy:
			if record.UnitPrice.Cmp(&zero) <= 0 {
				return fmt.Errorf("BUY activity %q must have a unit price greater than zero", record.SourceID)
			}
		case syncmodel.ActivityTypeSell:
			if record.UnitPrice.Cmp(&zero) < 0 {
				return fmt.Errorf("SELL activity %q cannot have a negative unit price", record.SourceID)
			}
			if record.UnitPrice.Cmp(&zero) == 0 && strings.TrimSpace(record.Comment) == "" {
				return fmt.Errorf("SELL activity %q requires a comment when unit price is zero", record.SourceID)
			}
		default:
			return fmt.Errorf("activity %q uses unsupported type %q", record.SourceID, record.ActivityType)
		}

		var runningQuantity = runningQuantityByAsset[record.AssetSymbol]
		switch record.ActivityType {
		case syncmodel.ActivityTypeBuy:
			_, _ = apd.BaseContext.Add(&runningQuantity, &runningQuantity, &record.Quantity)
		case syncmodel.ActivityTypeSell:
			_, _ = apd.BaseContext.Sub(&runningQuantity, &runningQuantity, &record.Quantity)
			if runningQuantity.Cmp(&zero) < 0 {
				return fmt.Errorf("activity %q would drive holdings below zero", record.SourceID)
			}
		}
		runningQuantityByAsset[record.AssetSymbol] = runningQuantity

		previousOccurredAt = occurredAt
		previousSourceID = record.SourceID
	}

	return nil
}
