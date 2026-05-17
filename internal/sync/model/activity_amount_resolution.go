// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// ResolvedActivityAmounts stores the transient current-slice money view derived
// from one persisted activity record.
//
// Example:
//
//	amounts, err := model.ResolveActivityAmounts(record)
//	if err != nil {
//		panic(err)
//	}
//	_ = amounts.UnitPrice
//
// The resolved values are not persisted. They exist only so validation and
// diagnostic-report code can apply the current slice's basis-input rules without
// forcing the mapper to store one selected cross-currency view.
// Authored by: OpenCode
type ResolvedActivityAmounts struct {
	UnitPrice          *apd.Decimal
	UnitPriceCurrency  string
	GrossValue         *apd.Decimal
	GrossValueCurrency string
	FeeAmount          *apd.Decimal
	FeeAmountCurrency  string
}

// ResolveActivityAmounts derives the transient unit-price, gross-value, and fee
// view required by the current slice from one explicit-currency activity record.
//
// Example:
//
//	amounts, err := model.ResolveActivityAmounts(record)
//	if err != nil {
//		panic(err)
//	}
//	_ = amounts.GrossValue
//
// The selection rules stay local to validation and diagnostics. The persisted
// activity record itself keeps only the explicit order-currency,
// asset-profile-currency, and base-currency groups that Ghostfolio exposes.
// Authored by: OpenCode
func ResolveActivityAmounts(record ActivityRecord) (ResolvedActivityAmounts, error) {
	grossValue, grossValueCurrency, err := resolveGrossValue(record)
	if err != nil {
		return ResolvedActivityAmounts{}, err
	}
	unitPrice, unitPriceCurrency, err := resolveUnitPrice(record, grossValue, grossValueCurrency)
	if err != nil {
		return ResolvedActivityAmounts{}, err
	}
	feeAmount, feeAmountCurrency := resolveFeeAmount(record)

	return ResolvedActivityAmounts{
		UnitPrice:          unitPrice,
		UnitPriceCurrency:  unitPriceCurrency,
		GrossValue:         grossValue,
		GrossValueCurrency: grossValueCurrency,
		FeeAmount:          feeAmount,
		FeeAmountCurrency:  feeAmountCurrency,
	}, nil
}

// resolveUnitPrice derives the current-slice unit price view without persisting
// it on the activity record.
// Authored by: OpenCode
func resolveUnitPrice(
	record ActivityRecord,
	grossValue *apd.Decimal,
	grossValueCurrency string,
) (*apd.Decimal, string, error) {
	if record.OrderUnitPrice != nil {
		return record.OrderUnitPrice, strings.TrimSpace(record.OrderCurrency), nil
	}
	if record.AssetProfileUnitPrice != nil {
		return record.AssetProfileUnitPrice, strings.TrimSpace(record.AssetProfileCurrency), nil
	}
	if grossValue == nil {
		return nil, "", fmt.Errorf("activity %q unit price basis input is required", strings.TrimSpace(record.SourceID))
	}

	unitPrice, _, err := decimalsupport.DivideExact(*grossValue, record.Quantity)
	if err != nil {
		return nil, "", fmt.Errorf("activity %q unit price basis input is not exact: %w", strings.TrimSpace(record.SourceID), err)
	}

	return &unitPrice, strings.TrimSpace(grossValueCurrency), nil
}

// resolveGrossValue derives the current-slice gross value view without
// persisting it on the activity record.
// Authored by: OpenCode
func resolveGrossValue(record ActivityRecord) (*apd.Decimal, string, error) {
	if record.OrderGrossValue != nil {
		return record.OrderGrossValue, strings.TrimSpace(record.OrderCurrency), nil
	}
	if record.BaseGrossValue != nil {
		return record.BaseGrossValue, strings.TrimSpace(record.BaseCurrency), nil
	}
	if record.AssetProfileUnitPrice != nil {
		grossValue, err := multiplyActivityAmount(record.Quantity, *record.AssetProfileUnitPrice)
		if err != nil {
			return nil, "", fmt.Errorf("activity %q gross value basis input is invalid: %w", strings.TrimSpace(record.SourceID), err)
		}
		return &grossValue, strings.TrimSpace(record.AssetProfileCurrency), nil
	}
	if record.OrderUnitPrice != nil {
		grossValue, err := multiplyActivityAmount(record.Quantity, *record.OrderUnitPrice)
		if err != nil {
			return nil, "", fmt.Errorf("activity %q gross value basis input is invalid: %w", strings.TrimSpace(record.SourceID), err)
		}
		return &grossValue, strings.TrimSpace(record.OrderCurrency), nil
	}

	return nil, "", fmt.Errorf("activity %q gross value basis input is required", strings.TrimSpace(record.SourceID))
}

// resolveFeeAmount selects the current-slice fee view without persisting it on
// the activity record.
// Authored by: OpenCode
func resolveFeeAmount(record ActivityRecord) (*apd.Decimal, string) {
	if record.OrderFeeAmount != nil {
		return record.OrderFeeAmount, strings.TrimSpace(record.OrderCurrency)
	}
	if record.AssetProfileFeeAmount != nil {
		return record.AssetProfileFeeAmount, strings.TrimSpace(record.AssetProfileCurrency)
	}
	if record.BaseFeeAmount != nil {
		return record.BaseFeeAmount, strings.TrimSpace(record.BaseCurrency)
	}

	return nil, ""
}

// multiplyActivityAmount preserves exact-decimal precision when one transient
// current-slice amount must be derived from quantity and unit price.
// Authored by: OpenCode
func multiplyActivityAmount(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	var product apd.Decimal
	if _, err := apd.BaseContext.Mul(&product, &left, &right); err != nil {
		return apd.Decimal{}, fmt.Errorf("derive activity amount from quantity and unit price: %w", err)
	}

	return product, nil
}
