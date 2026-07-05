// Package model defines detailed audit activity entries for Annex 1 report
// rendering.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/apd/v3"
)

// AuditActivityEntry stores one replayed source activity row for the Annex 1
// per-asset audit report.
// Authored by: OpenCode
type AuditActivityEntry struct {
	SourceID               string
	OccurredAt             time.Time
	ActivityType           ActivityType
	Quantity               apd.Decimal
	UnitPrice              *apd.Decimal
	GrossValue             *apd.Decimal
	FeeAmount              *apd.Decimal
	ActivityCurrency       string
	CalculationCurrency    string
	QuantityAfterActivity  apd.Decimal
	BasisAfterActivity     apd.Decimal
	FullLiquidationEvent   bool
	AllocatedBasis         *apd.Decimal
	NetLiquidationProceeds *apd.Decimal
	GainOrLoss             *apd.Decimal
	ConversionStatus       ConversionStatus
	Note                   string
}

// Validate verifies one Annex 1 audit activity entry.
// Authored by: OpenCode
func (entry AuditActivityEntry) Validate() error {
	if strings.TrimSpace(entry.SourceID) == "" {
		return fmt.Errorf("audit activity entry source ID is required")
	}
	if entry.OccurredAt.IsZero() {
		return fmt.Errorf("audit activity entry occurred-at timestamp is required")
	}
	if err := validateActivityType(entry.ActivityType); err != nil {
		return fmt.Errorf("audit activity entry activity type: %w", err)
	}
	if err := validatePositiveDecimal(entry.Quantity, "audit activity entry quantity"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(entry.UnitPrice, "audit activity entry unit price"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(entry.GrossValue, "audit activity entry gross value"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(entry.FeeAmount, "audit activity entry fee amount"); err != nil {
		return err
	}
	if strings.TrimSpace(entry.CalculationCurrency) == "" {
		return fmt.Errorf("audit activity entry calculation currency is required")
	}
	if err := validateNonNegativeDecimal(entry.QuantityAfterActivity, "audit activity entry quantity after activity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(entry.BasisAfterActivity, "audit activity entry basis after activity"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(entry.AllocatedBasis, "audit activity entry allocated basis"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(entry.NetLiquidationProceeds, "audit activity entry net liquidation proceeds"); err != nil {
		return err
	}
	if err := validateOptionalDecimal(entry.GainOrLoss, "audit activity entry gain or loss"); err != nil {
		return err
	}
	if strings.TrimSpace(string(entry.ConversionStatus)) != "" {
		if err := validateConversionStatus(entry.ConversionStatus); err != nil {
			return fmt.Errorf("audit activity entry conversion status: %w", err)
		}
	}

	return nil
}

// cloneAuditActivityEntries returns a defensive copy of Annex 1 audit activity
// entries.
// Authored by: OpenCode
func cloneAuditActivityEntries(entries []AuditActivityEntry) []AuditActivityEntry {
	var cloned = make([]AuditActivityEntry, 0, len(entries))
	for _, entry := range entries {
		var entryCopy = entry
		entryCopy.UnitPrice = cloneOptionalDecimal(entry.UnitPrice)
		entryCopy.GrossValue = cloneOptionalDecimal(entry.GrossValue)
		entryCopy.FeeAmount = cloneOptionalDecimal(entry.FeeAmount)
		entryCopy.AllocatedBasis = cloneOptionalDecimal(entry.AllocatedBasis)
		entryCopy.NetLiquidationProceeds = cloneOptionalDecimal(entry.NetLiquidationProceeds)
		entryCopy.GainOrLoss = cloneOptionalDecimal(entry.GainOrLoss)
		cloned = append(cloned, entryCopy)
	}

	return cloned
}
