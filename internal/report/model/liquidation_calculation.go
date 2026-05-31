// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/apd/v3"
)

// LiquidationCalculation stores one priced liquidation calculation rendered in
// an asset detail section.
// Authored by: OpenCode
type LiquidationCalculation struct {
	SourceID               string
	OccurredAt             time.Time
	DisposedQuantity       apd.Decimal
	AllocatedBasis         apd.Decimal
	NetLiquidationProceeds apd.Decimal
	GainOrLoss             apd.Decimal
	ActivityCurrency       string
	CalculationCurrency    string
	Matches                []BasisMatch
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
