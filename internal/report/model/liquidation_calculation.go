// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
)

// Validate verifies one matched acquisition fragment used by a liquidation.
// Authored by: OpenCode
func (match BasisMatch) Validate() error {
	if strings.TrimSpace(match.AcquisitionSourceID) == "" {
		return fmt.Errorf("basis match acquisition source ID is required")
	}
	if err := validatePositiveDecimal(match.MatchedQuantity, "basis match matched quantity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(match.MatchedBasis, "basis match matched basis"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(match.MatchedProceeds, "basis match matched proceeds"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(match.MatchedGainOrLoss, "basis match matched gain or loss"); err != nil {
		return err
	}

	return nil
}

// Validate verifies one priced liquidation calculation row.
// Authored by: OpenCode
func (calculation LiquidationCalculation) Validate() error {
	if strings.TrimSpace(calculation.SourceID) == "" {
		return fmt.Errorf("liquidation calculation source ID is required")
	}
	if calculation.OccurredAt.IsZero() {
		return fmt.Errorf("liquidation calculation occurred-at timestamp is required")
	}
	if err := validatePositiveDecimal(
		calculation.DisposedQuantity,
		"liquidation calculation disposed quantity",
	); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(
		calculation.AllocatedBasis,
		"liquidation calculation allocated basis",
	); err != nil {
		return err
	}
	if err := validateFiniteDecimal(
		calculation.NetLiquidationProceeds,
		"liquidation calculation net liquidation proceeds",
	); err != nil {
		return err
	}
	if err := validateFiniteDecimal(calculation.GainOrLoss, "liquidation calculation gain or loss"); err != nil {
		return err
	}
	if strings.TrimSpace(calculation.ActivityCurrency) == "" {
		return fmt.Errorf("liquidation calculation activity currency is required")
	}
	for index, match := range calculation.Matches {
		if err := match.Validate(); err != nil {
			return fmt.Errorf("liquidation calculation basis match %d: %w", index, err)
		}
	}

	return nil
}
