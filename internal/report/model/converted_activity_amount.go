// Package model defines converted activity amount models used by report
// calculation and report audit rendering.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// ConvertedAmountKind identifies which selected activity monetary field was
// preserved or converted for report calculation and audit rendering.
// Authored by: OpenCode
type ConvertedAmountKind string

const (
	// ConvertedAmountKindUnitPrice identifies a selected unit-price amount.
	ConvertedAmountKindUnitPrice ConvertedAmountKind = "unit_price"

	// ConvertedAmountKindGrossValue identifies a selected gross-value amount.
	ConvertedAmountKindGrossValue ConvertedAmountKind = "gross_value"

	// ConvertedAmountKindFeeAmount identifies a selected fee amount.
	ConvertedAmountKindFeeAmount ConvertedAmountKind = "fee_amount"
)

// ConversionStatus identifies whether an activity amount was already in the
// report base currency or required exchange-rate conversion.
// Authored by: OpenCode
type ConversionStatus string

const (
	// ConversionStatusSameCurrency indicates no exchange-rate evidence was needed.
	ConversionStatusSameCurrency ConversionStatus = "same_currency"

	// ConversionStatusConverted indicates official rate evidence was applied.
	ConversionStatusConverted ConversionStatus = "converted"
)

// ConvertedActivityAmount stores one selected activity monetary value after the
// report conversion boundary has classified or converted it.
// Authored by: OpenCode
type ConvertedActivityAmount struct {
	SourceID             string
	AmountKind           ConvertedAmountKind
	OriginalCurrency     string
	OriginalAmount       apd.Decimal
	ReportBaseCurrency   ReportBaseCurrency
	ConvertedAmount      apd.Decimal
	ExchangeRateEvidence *ExchangeRateEvidence
	ConversionStatus     ConversionStatus
}

// Validate verifies one converted activity amount and its rate evidence
// relationship.
// Authored by: OpenCode
func (amount ConvertedActivityAmount) Validate() error {
	if strings.TrimSpace(amount.SourceID) == "" {
		return fmt.Errorf("converted activity amount source ID is required")
	}
	var err = validateConvertedAmountKind(amount.AmountKind)
	if err != nil {
		return fmt.Errorf("converted activity amount kind: %w", err)
	}
	if strings.TrimSpace(amount.OriginalCurrency) == "" {
		return fmt.Errorf("converted activity amount original currency is required")
	}
	err = validateReportBaseCurrency(amount.ReportBaseCurrency)
	if err != nil {
		return fmt.Errorf("converted activity amount report base currency: %w", err)
	}
	err = validateNonNegativeDecimal(amount.OriginalAmount, "converted activity amount original amount")
	if err != nil {
		return err
	}
	err = validateNonNegativeDecimal(amount.ConvertedAmount, "converted activity amount converted amount")
	if err != nil {
		return err
	}
	err = validateConversionStatus(amount.ConversionStatus)
	if err != nil {
		return fmt.Errorf("converted activity amount status: %w", err)
	}

	if amount.ConversionStatus == ConversionStatusSameCurrency {
		return validateSameCurrencyAmount(amount)
	}

	return validateConvertedAmount(amount)
}

// cloneConvertedActivityAmounts copies converted amount records and their
// optional exchange-rate evidence values.
// Authored by: OpenCode
func cloneConvertedActivityAmounts(amounts []ConvertedActivityAmount) []ConvertedActivityAmount {
	var cloned = append([]ConvertedActivityAmount(nil), amounts...)
	for index := range cloned {
		if cloned[index].ExchangeRateEvidence == nil {
			continue
		}
		var evidence = *cloned[index].ExchangeRateEvidence
		cloned[index].ExchangeRateEvidence = &evidence
	}

	return cloned
}

// validateConvertedAmountKind rejects unsupported converted amount kinds.
// Authored by: OpenCode
func validateConvertedAmountKind(kind ConvertedAmountKind) error {
	switch kind {
	case ConvertedAmountKindUnitPrice, ConvertedAmountKindGrossValue, ConvertedAmountKindFeeAmount:
		return nil
	default:
		return fmt.Errorf("unsupported converted amount kind %q", kind)
	}
}

// validateConversionStatus rejects unsupported conversion status values.
// Authored by: OpenCode
func validateConversionStatus(status ConversionStatus) error {
	switch status {
	case ConversionStatusSameCurrency, ConversionStatusConverted:
		return nil
	default:
		return fmt.Errorf("unsupported conversion status %q", status)
	}
}

// validateSameCurrencyAmount verifies no rate evidence is attached to an
// unchanged same-currency amount.
// Authored by: OpenCode
func validateSameCurrencyAmount(amount ConvertedActivityAmount) error {
	if strings.TrimSpace(amount.OriginalCurrency) != amount.ReportBaseCurrency.Label() {
		return fmt.Errorf("same-currency amount original currency must match report base currency")
	}
	if amount.ExchangeRateEvidence != nil {
		return fmt.Errorf("same-currency amount must not include exchange-rate evidence")
	}
	if amount.OriginalAmount.Cmp(&amount.ConvertedAmount) != 0 {
		return fmt.Errorf("same-currency amount converted amount must equal original amount")
	}

	return nil
}

// validateConvertedAmount verifies a converted amount has matching canonical
// exchange-rate evidence.
// Authored by: OpenCode
func validateConvertedAmount(amount ConvertedActivityAmount) error {
	if strings.TrimSpace(amount.OriginalCurrency) == amount.ReportBaseCurrency.Label() {
		return fmt.Errorf("converted amount original currency must differ from report base currency")
	}
	if amount.ExchangeRateEvidence == nil {
		return fmt.Errorf("converted amount exchange-rate evidence is required")
	}
	var err = amount.ExchangeRateEvidence.Validate()
	if err != nil {
		return fmt.Errorf("converted amount exchange-rate evidence: %w", err)
	}
	if strings.TrimSpace(amount.ExchangeRateEvidence.SourceCurrency) != strings.TrimSpace(amount.OriginalCurrency) {
		return fmt.Errorf("converted amount evidence source currency mismatch")
	}
	if amount.ExchangeRateEvidence.BaseCurrency != amount.ReportBaseCurrency {
		return fmt.Errorf("converted amount evidence base currency mismatch")
	}

	return nil
}
