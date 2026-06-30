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

// AssetActivityRow stores one in-year activity row for an included asset.
// Authored by: OpenCode
type AssetActivityRow struct {
	SourceID                    string
	OccurredAt                  time.Time
	ActivityType                ActivityType
	Quantity                    apd.Decimal
	UnitPrice                   *apd.Decimal
	GrossValue                  *apd.Decimal
	FeeAmount                   *apd.Decimal
	ActivityCurrency            string
	BasisAfterRow               apd.Decimal
	CalculationCurrency         string
	QuantityAfterRow            apd.Decimal
	ConversionStatus            ConversionStatus
	HoldingReductionExplanation string
	LiquidationCalculation      *LiquidationCalculation
}

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
	if strings.TrimSpace(string(row.ConversionStatus)) != "" {
		var err = validateConversionStatus(row.ConversionStatus)
		if err != nil {
			return fmt.Errorf("asset activity row conversion status: %w", err)
		}
	}
	if row.LiquidationCalculation != nil {
		if err := row.LiquidationCalculation.Validate(); err != nil {
			return fmt.Errorf("asset activity row liquidation calculation: %w", err)
		}
	}

	return nil
}
