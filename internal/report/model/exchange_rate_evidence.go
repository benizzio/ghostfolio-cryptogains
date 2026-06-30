// Package model defines report-owned exchange-rate evidence models retained by
// calculated capital-gains reports.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/apd/v3"
)

// RateAuthority identifies the report-owned authority concept retained as
// conversion evidence without depending on integration-layer provider types.
// Authored by: OpenCode
type RateAuthority string

const (
	// RateAuthorityEuropeanCentralBank identifies ECB-authorized rate evidence.
	RateAuthorityEuropeanCentralBank RateAuthority = "european_central_bank"

	// RateAuthorityFederalReserve identifies Federal Reserve-authorized evidence.
	RateAuthorityFederalReserve RateAuthority = "federal_reserve"
)

// RateProviderID identifies the report-owned provider category retained for
// audit rendering and report validation.
// Authored by: OpenCode
type RateProviderID string

const (
	// RateProviderIDECBEXR identifies ECB Data Portal EXR evidence.
	RateProviderIDECBEXR RateProviderID = "ecb_exr"

	// RateProviderIDFederalReserveH10 identifies Federal Reserve H.10 evidence.
	RateProviderIDFederalReserveH10 RateProviderID = "federal_reserve_h10"
)

// QuoteDirection identifies how a provider-published rate converts source
// amounts into the report base currency.
// Authored by: OpenCode
type QuoteDirection string

const (
	// QuoteDirectionSourcePerBase divides source amounts by the rate.
	QuoteDirectionSourcePerBase QuoteDirection = "source_per_base"

	// QuoteDirectionBasePerSource multiplies source amounts by the rate.
	QuoteDirectionBasePerSource QuoteDirection = "base_per_source"
)

// ExchangeRateEvidence stores canonical authority-backed rate details used for
// one source-currency to report-base-currency conversion.
// Authored by: OpenCode
type ExchangeRateEvidence struct {
	SourceCurrency   string
	BaseCurrency     ReportBaseCurrency
	ActivityDate     time.Time
	RateDate         time.Time
	Authority        RateAuthority
	ProviderID       RateProviderID
	RateKind         string
	QuoteDirection   QuoteDirection
	RateValue        apd.Decimal
	DatasetReference string
}

// Validate verifies canonical rate evidence before it is used in report
// calculation or rendered as report audit data.
// Authored by: OpenCode
func (evidence ExchangeRateEvidence) Validate() error {
	if strings.TrimSpace(evidence.SourceCurrency) == "" {
		return fmt.Errorf("exchange rate evidence source currency is required")
	}
	if err := validateReportBaseCurrency(evidence.BaseCurrency); err != nil {
		return fmt.Errorf("exchange rate evidence base currency: %w", err)
	}
	if strings.TrimSpace(evidence.SourceCurrency) == evidence.BaseCurrency.Label() {
		return fmt.Errorf("exchange rate evidence source currency must differ from base currency")
	}
	if evidence.ActivityDate.IsZero() {
		return fmt.Errorf("exchange rate evidence activity date is required")
	}
	if evidence.RateDate.IsZero() {
		return fmt.Errorf("exchange rate evidence rate date is required")
	}
	if evidence.RateDate.After(evidence.ActivityDate) {
		return fmt.Errorf("exchange rate evidence rate date must not be after activity date")
	}
	if err := validateRateAuthority(evidence.Authority); err != nil {
		return fmt.Errorf("exchange rate evidence authority: %w", err)
	}
	if err := validateRateProviderID(evidence.ProviderID); err != nil {
		return fmt.Errorf("exchange rate evidence provider: %w", err)
	}
	if err := validateProviderForBaseCurrency(evidence.BaseCurrency.Label(), evidence.ProviderID, evidence.Authority); err != nil {
		return fmt.Errorf("exchange rate evidence provider does not match report base currency: %w", err)
	}
	if strings.TrimSpace(evidence.RateKind) == "" {
		return fmt.Errorf("exchange rate evidence rate kind is required")
	}
	if err := validateQuoteDirection(evidence.QuoteDirection); err != nil {
		return fmt.Errorf("exchange rate evidence quote direction: %w", err)
	}
	if err := validatePositiveDecimal(evidence.RateValue, "exchange rate evidence rate value"); err != nil {
		return err
	}
	if strings.TrimSpace(evidence.DatasetReference) == "" {
		return fmt.Errorf("exchange rate evidence dataset reference is required")
	}

	return nil
}

// cloneExchangeRateEvidence copies retained rate evidence for report model
// construction.
// Authored by: OpenCode
func cloneExchangeRateEvidence(sources []ExchangeRateEvidence) []ExchangeRateEvidence {
	var cloned = append([]ExchangeRateEvidence(nil), sources...)
	for index := range cloned {
		cloned[index].RateValue.Set(&sources[index].RateValue)
	}

	return cloned
}

// validateRateAuthority rejects unsupported report-owned provider authorities.
// Authored by: OpenCode
func validateRateAuthority(authority RateAuthority) error {
	switch authority {
	case RateAuthorityEuropeanCentralBank, RateAuthorityFederalReserve:
		return nil
	default:
		return fmt.Errorf("unsupported rate authority %q", authority)
	}
}

// validateRateProviderID rejects unsupported report-owned rate providers.
// Authored by: OpenCode
func validateRateProviderID(providerID RateProviderID) error {
	switch providerID {
	case RateProviderIDECBEXR, RateProviderIDFederalReserveH10:
		return nil
	default:
		return fmt.Errorf("unsupported rate provider %q", providerID)
	}
}

// validateQuoteDirection rejects ambiguous source-to-base rate semantics.
// Authored by: OpenCode
func validateQuoteDirection(direction QuoteDirection) error {
	switch direction {
	case QuoteDirectionSourcePerBase, QuoteDirectionBasePerSource:
		return nil
	default:
		return fmt.Errorf("unsupported quote direction %q", direction)
	}
}

// validateProviderForBaseCurrency verifies report-owned provider evidence is
// consistent with the selected report base currency.
// Authored by: OpenCode
func validateProviderForBaseCurrency(baseCurrency string, providerID RateProviderID, authority RateAuthority) error {
	switch strings.TrimSpace(baseCurrency) {
	case ReportBaseCurrencyEUR.Label():
		if providerID == RateProviderIDECBEXR && authority == RateAuthorityEuropeanCentralBank {
			return nil
		}
	case ReportBaseCurrencyUSD.Label():
		if providerID == RateProviderIDFederalReserveH10 && authority == RateAuthorityFederalReserve {
			return nil
		}
	}

	return fmt.Errorf("provider %q with authority %q is not valid for base currency %q", providerID, authority, baseCurrency)
}
