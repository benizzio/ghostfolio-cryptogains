// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// NewAssetSummaryEntry creates one validated summary-section row.
//
// Example:
//
//	entry, err := model.NewAssetSummaryEntry("asset-btc", "BTC", net, "USD")
//	if err != nil {
//		panic(err)
//	}
//	_ = entry.DisplayLabel
//
// Authored by: OpenCode
func NewAssetSummaryEntry(
	assetIdentityKey string,
	displayLabel string,
	netGainOrLoss apd.Decimal,
	reportCalculationCurrency string,
) (AssetSummaryEntry, error) {
	var entry = AssetSummaryEntry{
		AssetIdentityKey:          strings.TrimSpace(assetIdentityKey),
		DisplayLabel:              strings.TrimSpace(displayLabel),
		NetGainOrLoss:             netGainOrLoss,
		ReportCalculationCurrency: strings.TrimSpace(reportCalculationCurrency),
	}

	if err := entry.Validate(); err != nil {
		return AssetSummaryEntry{}, err
	}

	return entry, nil
}

// Validate verifies one summary-section row.
// Authored by: OpenCode
func (entry AssetSummaryEntry) Validate() error {
	if strings.TrimSpace(entry.AssetIdentityKey) == "" {
		return fmt.Errorf("asset summary entry asset identity key is required")
	}
	if err := validateFiniteDecimal(entry.NetGainOrLoss, "asset summary entry net gain or loss"); err != nil {
		return err
	}

	return nil
}
