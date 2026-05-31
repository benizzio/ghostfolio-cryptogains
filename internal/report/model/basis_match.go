// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// BasisMatch stores one acquisition fragment consumed by one liquidation.
// Authored by: OpenCode
type BasisMatch struct {
	AcquisitionSourceID string
	MatchedQuantity     apd.Decimal
	MatchedBasis        apd.Decimal
	MatchedProceeds     *apd.Decimal
	MatchedGainOrLoss   *apd.Decimal
}

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
