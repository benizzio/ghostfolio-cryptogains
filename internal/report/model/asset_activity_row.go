// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
)

// Validate verifies one in-year asset activity row.
// Authored by: OpenCode
func (row AssetActivityRow) Validate() error {
	if strings.TrimSpace(row.SourceID) == "" {
		return fmt.Errorf("asset activity row source ID is required")
	}
	if row.OccurredAt.IsZero() {
		return fmt.Errorf("asset activity row occurred-at timestamp is required")
	}
	if err := validateActivityType(row.ActivityType); err != nil {
		return fmt.Errorf("asset activity row activity type: %w", err)
	}
	if err := validatePositiveDecimal(row.Quantity, "asset activity row quantity"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(row.UnitPrice, "asset activity row unit price"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(row.GrossValue, "asset activity row gross value"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(row.FeeAmount, "asset activity row fee amount"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(row.BasisAfterRow, "asset activity row basis after row"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(row.QuantityAfterRow, "asset activity row quantity after row"); err != nil {
		return err
	}
	if row.LiquidationCalculation != nil {
		if err := row.LiquidationCalculation.Validate(); err != nil {
			return fmt.Errorf("asset activity row liquidation calculation: %w", err)
		}
	}

	return nil
}
