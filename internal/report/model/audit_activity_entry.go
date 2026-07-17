// Package model defines detailed audit activity entries for Annex 1 report
// rendering.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// AuditActivityEntry stores one replayed source activity row for the Annex 1
// per-asset audit report.
// Authored by: OpenCode
type AuditActivityEntry struct {
	SourceID              string
	OccurredAt            time.Time
	ActivityType          ActivityType
	Quantity              apd.Decimal
	UnitPrice             *apd.Decimal
	GrossValue            *apd.Decimal
	FeeAmount             *apd.Decimal
	ActivityCurrency      string
	CalculationCurrency   string
	QuantityAfterActivity apd.Decimal
	BasisAfterActivity    apd.Decimal
	FullLiquidationEvent  bool
	// IsZeroPricedHoldingReduction preserves the inherited exact classification
	// for transient Annex 1 presentation without changing the retained activity
	// currency or being persisted as report state.
	// Authored by: OpenCode
	IsZeroPricedHoldingReduction bool
	AllocatedBasis               *apd.Decimal
	NetLiquidationProceeds       *apd.Decimal
	GainOrLoss                   *apd.Decimal
	ConversionStatus             ConversionStatus
	Note                         string
}

// Validate verifies that an Annex 1 activity has the required identity, exact
// decimal values, replay state, and optional conversion classification in the
// documented validation order. For example, call `err := entry.Validate()`
// before adding entry to a PerAssetAuditSection.
// Authored by: OpenCode
func (entry AuditActivityEntry) Validate() error {
	if err := entry.validateIdentity(); err != nil {
		return err
	}
	if err := entry.validateAmounts(); err != nil {
		return err
	}
	if err := entry.validateReplayState(); err != nil {
		return err
	}
	return entry.validateConversionStatus()
}

// validateIdentity verifies the required source identity and activity fields.
// Authored by: OpenCode
func (entry AuditActivityEntry) validateIdentity() error {
	if strings.TrimSpace(entry.SourceID) == "" {
		return fmt.Errorf("audit activity entry source ID is required")
	}
	if entry.OccurredAt.IsZero() {
		return fmt.Errorf("audit activity entry occurred-at timestamp is required")
	}
	if err := validateActivityType(entry.ActivityType); err != nil {
		return fmt.Errorf("audit activity entry activity type: %w", err)
	}
	return nil
}

// validateAmounts verifies activity monetary values in their existing order.
// Authored by: OpenCode
func (entry AuditActivityEntry) validateAmounts() error {
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
	return nil
}

// validateReplayState verifies the post-activity holdings evidence.
// Authored by: OpenCode
func (entry AuditActivityEntry) validateReplayState() error {
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
	return nil
}

// validateConversionStatus verifies the optional visible conversion classification.
// Authored by: OpenCode
func (entry AuditActivityEntry) validateConversionStatus() error {
	if strings.TrimSpace(string(entry.ConversionStatus)) == "" {
		return nil
	}
	if err := validateConversionStatus(entry.ConversionStatus); err != nil {
		return fmt.Errorf("audit activity entry conversion status: %w", err)
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
		entryCopy.UnitPrice = decimalsupport.ClonePointer(entry.UnitPrice)
		entryCopy.GrossValue = decimalsupport.ClonePointer(entry.GrossValue)
		entryCopy.FeeAmount = decimalsupport.ClonePointer(entry.FeeAmount)
		entryCopy.AllocatedBasis = decimalsupport.ClonePointer(entry.AllocatedBasis)
		entryCopy.NetLiquidationProceeds = decimalsupport.ClonePointer(entry.NetLiquidationProceeds)
		entryCopy.GainOrLoss = decimalsupport.ClonePointer(entry.GainOrLoss)
		cloned = append(cloned, entryCopy)
	}

	return cloned
}
