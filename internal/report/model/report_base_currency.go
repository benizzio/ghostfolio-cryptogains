// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
)

// ReportBaseCurrency identifies the selected fiat currency used for every
// monetary calculation in one report-generation run.
// Authored by: OpenCode
type ReportBaseCurrency string

const (
	// ReportBaseCurrencyUSD identifies a report calculated in United States dollars.
	ReportBaseCurrencyUSD ReportBaseCurrency = "USD"

	// ReportBaseCurrencyEUR identifies a report calculated in euros.
	ReportBaseCurrencyEUR ReportBaseCurrency = "EUR"
)

var supportedReportBaseCurrencies = []ReportBaseCurrency{
	ReportBaseCurrencyUSD,
	ReportBaseCurrencyEUR,
}

// SupportedReportBaseCurrencies returns the exact report base currencies
// supported by the conversion feature in stable UI selection order.
//
// Example:
//
//	currencies := model.SupportedReportBaseCurrencies()
//	_ = currencies[0]
//
// Authored by: OpenCode
func SupportedReportBaseCurrencies() []ReportBaseCurrency {
	var currencies = make([]ReportBaseCurrency, len(supportedReportBaseCurrencies))
	copy(currencies, supportedReportBaseCurrencies)
	return currencies
}

// Label returns the user-visible label for one report base currency.
//
// Example:
//
//	label := model.ReportBaseCurrencyUSD.Label()
//	_ = label
//
// Authored by: OpenCode
func (currency ReportBaseCurrency) Label() string {
	switch currency {
	case ReportBaseCurrencyUSD:
		return "USD"
	case ReportBaseCurrencyEUR:
		return "EUR"
	default:
		return strings.TrimSpace(string(currency))
	}
}

// validateReportBaseCurrency rejects missing or unsupported report base
// currency values.
// Authored by: OpenCode
func validateReportBaseCurrency(currency ReportBaseCurrency) error {
	if strings.TrimSpace(string(currency)) == "" {
		return fmt.Errorf("report base currency is required")
	}

	for _, supportedCurrency := range SupportedReportBaseCurrencies() {
		if currency == supportedCurrency {
			return nil
		}
	}

	return fmt.Errorf("unsupported report base currency %q", currency)
}
