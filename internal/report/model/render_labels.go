// Package model defines closed user-facing render labels shared by report
// renderers.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

const zeroPricedSellActivityLabel = "BLOCKCHAIN OP"

// RenderConversionStatusLabel returns the closed user-facing conversion status
// label for report output.
//
// Example:
//
//	label, err := model.RenderConversionStatusLabel(model.ConversionStatusConverted)
//	if err != nil {
//		panic(err)
//	}
//	_ = label // "Converted"
//
// Authored by: OpenCode
func RenderConversionStatusLabel(status ConversionStatus) (string, error) {
	switch status {
	case ConversionStatusSameCurrency:
		return "Same currency", nil
	case ConversionStatusConverted:
		return "Converted", nil
	default:
		return "", fmt.Errorf("unsupported conversion status %q", status)
	}
}

// RenderQuoteDirectionLabel returns the closed user-facing quote direction label
// for later annex and PDF audit rendering.
//
// Example:
//
//	label, err := model.RenderQuoteDirectionLabel(model.QuoteDirectionSourcePerBase)
//	if err != nil {
//		panic(err)
//	}
//	_ = label
//
// Authored by: OpenCode
func RenderQuoteDirectionLabel(direction QuoteDirection) (string, error) {
	switch direction {
	case QuoteDirectionSourcePerBase:
		return "Source currency per base currency", nil
	case QuoteDirectionBasePerSource:
		return "Base currency per source currency", nil
	default:
		return "", fmt.Errorf("unsupported quote direction %q", direction)
	}
}

// RenderActivityTypeLabel returns the user-facing activity type label for one
// main-report activity row, including the zero-priced SELL custody-operation
// display rule.
//
// Example:
//
//	label, err := model.RenderActivityTypeLabel(row)
//	if err != nil {
//		panic(err)
//	}
//	_ = label
//
// Authored by: OpenCode
func RenderActivityTypeLabel(row AssetActivityRow) (string, error) {
	if row.ActivityType == ActivityTypeSell {
		var zeroPriced, err = isZeroPricedActivity(row)
		if err != nil {
			return "", fmt.Errorf("render activity type label zero-priced fields: %w", err)
		}
		if zeroPriced {
			return zeroPricedSellActivityLabel, nil
		}
	}
	if err := validateActivityType(row.ActivityType); err != nil {
		return "", err
	}

	return strings.TrimSpace(string(row.ActivityType)), nil
}

// RenderAuditActivityTypeLabel returns the user-facing activity type label for
// one Annex 1 audit activity entry.
//
// Example:
//
//	label, err := model.RenderAuditActivityTypeLabel(entry)
//	if err != nil {
//		panic(err)
//	}
//	_ = label
//
// Authored by: OpenCode
func RenderAuditActivityTypeLabel(entry AuditActivityEntry) (string, error) {
	if entry.ActivityType == ActivityTypeSell {
		var zeroPriced, err = auditEntryIsZeroPriced(entry)
		if err != nil {
			return "", fmt.Errorf("render audit activity type label zero-priced fields: %w", err)
		}
		if zeroPriced {
			return zeroPricedSellActivityLabel, nil
		}
	}
	if err := validateActivityType(entry.ActivityType); err != nil {
		return "", err
	}

	return strings.TrimSpace(string(entry.ActivityType)), nil
}

// auditEntryIsZeroPriced reports whether an audit entry carries explicit zero
// monetary slots for the custody-operation display rule.
// Authored by: OpenCode
func auditEntryIsZeroPriced(entry AuditActivityEntry) (bool, error) {
	return monetaryFieldsAreZero(entry.UnitPrice, entry.GrossValue, entry.FeeAmount)
}

// isZeroPricedActivity reports whether a row carries explicit zero monetary
// slots for the custody-operation display rule.
// Authored by: OpenCode
func isZeroPricedActivity(row AssetActivityRow) (bool, error) {
	return monetaryFieldsAreZero(row.UnitPrice, row.GrossValue, row.FeeAmount)
}

// monetaryFieldsAreZero verifies that all optional monetary fields are explicit
// finite zero values.
// Authored by: OpenCode
func monetaryFieldsAreZero(unitPrice *apd.Decimal, grossValue *apd.Decimal, feeAmount *apd.Decimal) (bool, error) {
	var unitPriceIsZero, err = supportmath.IsZeroPointer(unitPrice)
	if err != nil {
		return false, err
	}
	var grossValueIsZero bool
	grossValueIsZero, err = supportmath.IsZeroPointer(grossValue)
	if err != nil {
		return false, err
	}
	var feeAmountIsZero bool
	feeAmountIsZero, err = supportmath.IsZeroPointer(feeAmount)
	if err != nil {
		return false, err
	}

	return unitPriceIsZero && grossValueIsZero && feeAmountIsZero, nil
}
