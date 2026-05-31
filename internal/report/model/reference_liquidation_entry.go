// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
)

// ReferenceLiquidationEntry stores one row in the report reference section.
// Authored by: OpenCode
type ReferenceLiquidationEntry struct {
	AssetIdentityKey                   string
	DisplayLabel                       string
	FullLiquidationCountThroughYearEnd int
	MainSectionStatus                  ReferenceSectionStatus
}

// NewReferenceLiquidationEntry creates one validated reference-section row.
//
// Example:
//
//	entry, err := model.NewReferenceLiquidationEntry("asset-btc", "BTC", 1, model.ReferenceSectionStatusReferenceOnly)
//	if err != nil {
//		panic(err)
//	}
//	_ = entry.MainSectionStatus
//
// Authored by: OpenCode
func NewReferenceLiquidationEntry(
	assetIdentityKey string,
	displayLabel string,
	fullLiquidationCountThroughYearEnd int,
	mainSectionStatus ReferenceSectionStatus,
) (ReferenceLiquidationEntry, error) {
	var entry = ReferenceLiquidationEntry{
		AssetIdentityKey:                   strings.TrimSpace(assetIdentityKey),
		DisplayLabel:                       strings.TrimSpace(displayLabel),
		FullLiquidationCountThroughYearEnd: fullLiquidationCountThroughYearEnd,
		MainSectionStatus:                  mainSectionStatus,
	}

	if err := entry.Validate(); err != nil {
		return ReferenceLiquidationEntry{}, err
	}

	return entry, nil
}

// Validate verifies one reference-section row.
// Authored by: OpenCode
func (entry ReferenceLiquidationEntry) Validate() error {
	if strings.TrimSpace(entry.AssetIdentityKey) == "" {
		return fmt.Errorf("reference entry asset identity key is required")
	}
	if entry.FullLiquidationCountThroughYearEnd < 0 {
		return fmt.Errorf("reference entry full liquidation count must not be negative")
	}
	if err := validateReferenceSectionStatus(entry.MainSectionStatus); err != nil {
		return fmt.Errorf("reference entry main section status: %w", err)
	}

	return nil
}
