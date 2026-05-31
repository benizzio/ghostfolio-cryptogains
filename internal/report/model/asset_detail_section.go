// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// NewAssetDetailSection creates one validated per-asset detail section.
//
// Example:
//
//	section, err := model.NewAssetDetailSection("asset-btc", "BTC", openingQty, openingBasis, closingQty, closingBasis, "USD", nil, nil)
//	if err != nil {
//		panic(err)
//	}
//	_ = section.AssetIdentityKey
//
// Authored by: OpenCode
func NewAssetDetailSection(
	assetIdentityKey string,
	displayLabel string,
	openingQuantity apd.Decimal,
	openingCostBasis apd.Decimal,
	closingQuantity apd.Decimal,
	closingCostBasis apd.Decimal,
	calculationCurrency string,
	activityRows []AssetActivityRow,
	liquidationSummaries []LiquidationCalculation,
) (AssetDetailSection, error) {
	var section = AssetDetailSection{
		AssetIdentityKey:     strings.TrimSpace(assetIdentityKey),
		DisplayLabel:         strings.TrimSpace(displayLabel),
		OpeningQuantity:      openingQuantity,
		OpeningCostBasis:     openingCostBasis,
		ClosingQuantity:      closingQuantity,
		ClosingCostBasis:     closingCostBasis,
		CalculationCurrency:  strings.TrimSpace(calculationCurrency),
		ActivityRows:         cloneAssetActivityRows(activityRows),
		LiquidationSummaries: cloneLiquidationCalculations(liquidationSummaries),
	}

	if err := section.Validate(); err != nil {
		return AssetDetailSection{}, err
	}

	return section, nil
}

// Validate verifies one per-asset detail section and its nested rows.
// Authored by: OpenCode
func (section AssetDetailSection) Validate() error {
	if strings.TrimSpace(section.AssetIdentityKey) == "" {
		return fmt.Errorf("asset detail section asset identity key is required")
	}
	if err := validateNonNegativeDecimal(section.OpeningQuantity, "asset detail section opening quantity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(
		section.OpeningCostBasis,
		"asset detail section opening cost basis",
	); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(section.ClosingQuantity, "asset detail section closing quantity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(
		section.ClosingCostBasis,
		"asset detail section closing cost basis",
	); err != nil {
		return err
	}

	for index, row := range section.ActivityRows {
		if err := row.Validate(); err != nil {
			return fmt.Errorf("asset detail section activity row %d: %w", index, err)
		}
	}
	for index, liquidation := range section.LiquidationSummaries {
		if err := liquidation.Validate(); err != nil {
			return fmt.Errorf("asset detail section liquidation summary %d: %w", index, err)
		}
	}

	return nil
}
