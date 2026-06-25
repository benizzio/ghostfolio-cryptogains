// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"fmt"
	"strings"
	"time"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// ProviderID identifies one fixed official exchange-rate provider behind the
// currency integration boundary.
// Authored by: OpenCode
type ProviderID string

const (
	// ProviderIDECBEXR identifies the European Central Bank EXR daily reference-rate provider.
	ProviderIDECBEXR ProviderID = "ecb_exr"

	// ProviderIDFederalReserveH10 identifies the Federal Reserve H.10 provider.
	ProviderIDFederalReserveH10 ProviderID = "federal_reserve_h10"
)

// RateAuthority identifies the official authority behind one canonical rate.
// Authored by: OpenCode
type RateAuthority string

const (
	// RateAuthorityEuropeanCentralBank identifies the European Central Bank authority.
	RateAuthorityEuropeanCentralBank RateAuthority = "european_central_bank"

	// RateAuthorityFederalReserve identifies the Federal Reserve authority.
	RateAuthorityFederalReserve RateAuthority = "federal_reserve"
)

// QuoteDirection identifies how a provider-published rate is quoted before the
// report calculator applies conversion formulas.
// Authored by: OpenCode
type QuoteDirection string

const (
	// QuoteDirectionSourcePerBase means the rate is source currency units per one base unit.
	QuoteDirectionSourcePerBase QuoteDirection = "source_per_base"

	// QuoteDirectionBasePerSource means the rate is base currency units per one source unit.
	QuoteDirectionBasePerSource QuoteDirection = "base_per_source"
)

const (
	// RateKindECBEXRDailyReference describes ECB EXR daily euro reference-rate observations.
	RateKindECBEXRDailyReference = "daily euro foreign exchange reference rate"

	// RateKindFederalReserveH10NoonBuying describes Federal Reserve H.10 noon buying-rate observations.
	RateKindFederalReserveH10NoonBuying = "daily noon buying rate"
)

// ExchangeRateEvidence stores canonical authority-backed evidence for one
// source-to-base conversion rate. Provider DTOs must be mapped into this model
// before report calculation consumes rate data.
//
// Example:
//
//	request, err := currency.NewRateLookupRequest("USD", currency.BaseCurrencyEUR, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	rate := *apd.New(109, -2)
//	evidence, err := currency.NewExchangeRateEvidence(
//		request,
//		request.ActivityDate,
//		currency.RateAuthorityEuropeanCentralBank,
//		currency.ProviderIDECBEXR,
//		currency.RateKindECBEXRDailyReference,
//		currency.QuoteDirectionSourcePerBase,
//		rate,
//		"EXR/D.USD.EUR.SP00.A",
//	)
//	if err != nil {
//		panic(err)
//	}
//	_ = evidence.RateValue
//
// Authored by: OpenCode
type ExchangeRateEvidence struct {
	SourceCurrency   string
	BaseCurrency     string
	ActivityDate     time.Time
	RateDate         time.Time
	Authority        RateAuthority
	ProviderID       ProviderID
	RateKind         string
	QuoteDirection   QuoteDirection
	RateValue        apd.Decimal
	DatasetReference string
}

// NewExchangeRateEvidence creates validated canonical rate evidence from one
// lookup request and one provider observation.
//
// Example:
//
//	request, _ := currency.NewRateLookupRequest("USD", currency.BaseCurrencyEUR, time.Now())
//	rate := *apd.New(109, -2)
//	evidence, err := currency.NewExchangeRateEvidence(request, request.ActivityDate, currency.RateAuthorityEuropeanCentralBank, currency.ProviderIDECBEXR, currency.RateKindECBEXRDailyReference, currency.QuoteDirectionSourcePerBase, rate, "EXR/D.USD.EUR.SP00.A")
//	if err != nil {
//		panic(err)
//	}
//	_ = evidence.ProviderID
//
// Authored by: OpenCode
func NewExchangeRateEvidence(
	request RateLookupRequest,
	rateDate time.Time,
	authority RateAuthority,
	providerID ProviderID,
	rateKind string,
	quoteDirection QuoteDirection,
	rateValue apd.Decimal,
	datasetReference string,
) (ExchangeRateEvidence, error) {
	request.ActivityDate = canonicalDate(request.ActivityDate)
	var evidence = ExchangeRateEvidence{
		SourceCurrency:   request.SourceCurrency,
		BaseCurrency:     request.BaseCurrency,
		ActivityDate:     request.ActivityDate,
		RateDate:         canonicalDate(rateDate),
		Authority:        authority,
		ProviderID:       providerID,
		RateKind:         strings.TrimSpace(rateKind),
		QuoteDirection:   quoteDirection,
		RateValue:        decimalsupport.Clone(rateValue),
		DatasetReference: strings.TrimSpace(datasetReference),
	}

	if err := evidence.Validate(); err != nil {
		return ExchangeRateEvidence{}, err
	}

	return evidence, nil
}

// Validate verifies that one canonical rate evidence value is internally
// consistent and defensible for source-to-base conversion.
// Authored by: OpenCode
func (evidence ExchangeRateEvidence) Validate() error {
	var request = RateLookupRequest{
		SourceCurrency: evidence.SourceCurrency,
		BaseCurrency:   evidence.BaseCurrency,
		ActivityDate:   evidence.ActivityDate,
	}
	if err := request.Validate(); err != nil {
		return fmt.Errorf("exchange rate evidence lookup identity: %w", err)
	}
	if evidence.RateDate.IsZero() {
		return fmt.Errorf("exchange rate evidence rate date is required")
	}

	var activityDate = canonicalDate(evidence.ActivityDate)
	var rateDate = canonicalDate(evidence.RateDate)
	if rateDate.After(activityDate) {
		return fmt.Errorf("exchange rate evidence rate date %s must not be after activity date %s", formatDate(rateDate), formatDate(activityDate))
	}
	if err := validateRateAuthority(evidence.Authority); err != nil {
		return fmt.Errorf("exchange rate evidence authority: %w", err)
	}
	if err := validateProviderID(evidence.ProviderID); err != nil {
		return fmt.Errorf("exchange rate evidence provider: %w", err)
	}
	if err := validateProviderForBaseCurrency(evidence.BaseCurrency, evidence.ProviderID, evidence.Authority); err != nil {
		return fmt.Errorf("exchange rate evidence provider selection: %w", err)
	}
	if strings.TrimSpace(evidence.RateKind) == "" {
		return fmt.Errorf("exchange rate evidence rate kind is required")
	}
	if err := validateQuoteDirection(evidence.QuoteDirection); err != nil {
		return fmt.Errorf("exchange rate evidence quote direction: %w", err)
	}
	if err := supportmath.RequirePositive(evidence.RateValue); err != nil {
		return fmt.Errorf("exchange rate evidence rate value: %w", err)
	}
	if strings.TrimSpace(evidence.DatasetReference) == "" {
		return fmt.Errorf("exchange rate evidence dataset reference is required")
	}

	return nil
}

// matchesRequest verifies that one evidence value resolves the requested public
// lookup key.
// Authored by: OpenCode
func (evidence ExchangeRateEvidence) matchesRequest(request RateLookupRequest) bool {
	return evidence.SourceCurrency == request.SourceCurrency &&
		evidence.BaseCurrency == request.BaseCurrency &&
		canonicalDate(evidence.ActivityDate).Equal(canonicalDate(request.ActivityDate))
}

// cloneExchangeRateEvidence returns a defensive copy of canonical rate evidence.
// Authored by: OpenCode
func cloneExchangeRateEvidence(evidence ExchangeRateEvidence) ExchangeRateEvidence {
	var cloned = evidence
	cloned.ActivityDate = canonicalDate(evidence.ActivityDate)
	cloned.RateDate = canonicalDate(evidence.RateDate)
	cloned.RateValue = decimalsupport.Clone(evidence.RateValue)
	return cloned
}

// ValidateProviderID verifies that a provider identifier belongs to the
// supported official provider set.
//
// Example:
//
//	if err := currency.ValidateProviderID(currency.ProviderIDECBEXR); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateProviderID(providerID ProviderID) error {
	return validateProviderID(providerID)
}

// ValidateRateAuthority verifies that a rate authority belongs to the supported
// official authority set.
//
// Example:
//
//	if err := currency.ValidateRateAuthority(currency.RateAuthorityEuropeanCentralBank); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateRateAuthority(authority RateAuthority) error {
	return validateRateAuthority(authority)
}

// ValidateQuoteDirection verifies that a quote direction is supported by the
// report conversion formulas.
//
// Example:
//
//	if err := currency.ValidateQuoteDirection(currency.QuoteDirectionSourcePerBase); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateQuoteDirection(direction QuoteDirection) error {
	return validateQuoteDirection(direction)
}

// ValidateProviderForBaseCurrency verifies that the provider and authority match
// the fixed official provider selected for a report base currency.
//
// Example:
//
//	if err := currency.ValidateProviderForBaseCurrency(currency.BaseCurrencyEUR, currency.ProviderIDECBEXR, currency.RateAuthorityEuropeanCentralBank); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateProviderForBaseCurrency(baseCurrency string, providerID ProviderID, authority RateAuthority) error {
	return validateProviderForBaseCurrency(baseCurrency, providerID, authority)
}

// RateAuthorityDisplayLabel returns the report-facing label for a canonical rate
// authority. Unknown values are returned unchanged so callers can still disclose
// validated or diagnostic evidence safely after applying their own escaping.
//
// Example:
//
//	label := currency.RateAuthorityDisplayLabel(currency.RateAuthorityEuropeanCentralBank)
//	_ = label
//
// Authored by: OpenCode
func RateAuthorityDisplayLabel(authority RateAuthority) string {
	switch authority {
	case RateAuthorityEuropeanCentralBank:
		return "European Central Bank"
	case RateAuthorityFederalReserve:
		return "Federal Reserve"
	default:
		return string(authority)
	}
}

// ProviderDisplayLabel returns the report-facing label for a canonical provider
// identifier. Unknown values are returned unchanged so callers can still
// disclose diagnostic evidence safely after applying their own escaping.
//
// Example:
//
//	label := currency.ProviderDisplayLabel(currency.ProviderIDECBEXR)
//	_ = label
//
// Authored by: OpenCode
func ProviderDisplayLabel(provider ProviderID) string {
	switch provider {
	case ProviderIDECBEXR:
		return "ECB Data Portal `EXR`"
	case ProviderIDFederalReserveH10:
		return "Federal Reserve Board H.10/Data Download Program"
	default:
		return string(provider)
	}
}

// ProviderUnavailableDateRule returns the report-facing fallback-observation
// rule for a canonical provider identifier.
//
// Example:
//
//	rule := currency.ProviderUnavailableDateRule(currency.ProviderIDFederalReserveH10)
//	_ = rule
//
// Authored by: OpenCode
func ProviderUnavailableDateRule(provider ProviderID) string {
	switch provider {
	case ProviderIDECBEXR:
		return "most recent previous available ECB observation"
	case ProviderIDFederalReserveH10:
		return "most recent previous available H.10 observation"
	default:
		return "most recent previous available official observation"
	}
}

// validateProviderID rejects provider identifiers outside the supported feature set.
// Authored by: OpenCode
func validateProviderID(providerID ProviderID) error {
	switch providerID {
	case ProviderIDECBEXR, ProviderIDFederalReserveH10:
		return nil
	default:
		return fmt.Errorf("unsupported provider ID %q", providerID)
	}
}

// validateRateAuthority rejects authority identifiers outside the supported feature set.
// Authored by: OpenCode
func validateRateAuthority(authority RateAuthority) error {
	switch authority {
	case RateAuthorityEuropeanCentralBank, RateAuthorityFederalReserve:
		return nil
	default:
		return fmt.Errorf("unsupported rate authority %q", authority)
	}
}

// validateQuoteDirection rejects ambiguous or unsupported quote directions.
// Authored by: OpenCode
func validateQuoteDirection(direction QuoteDirection) error {
	switch direction {
	case QuoteDirectionSourcePerBase, QuoteDirectionBasePerSource:
		return nil
	default:
		return fmt.Errorf("unsupported quote direction %q", direction)
	}
}

// validateProviderForBaseCurrency verifies fixed base-currency provider selection.
// Authored by: OpenCode
func validateProviderForBaseCurrency(baseCurrency string, providerID ProviderID, authority RateAuthority) error {
	var expectedProviderID, expectedAuthority, ok = expectedProviderForBaseCurrency(baseCurrency)
	if !ok {
		return fmt.Errorf("unsupported base currency %q", baseCurrency)
	}
	if providerID != expectedProviderID {
		return fmt.Errorf("base currency %s requires provider %s", baseCurrency, expectedProviderID)
	}
	if authority != expectedAuthority {
		return fmt.Errorf("base currency %s requires authority %s", baseCurrency, expectedAuthority)
	}

	return nil
}

// expectedProviderForBaseCurrency returns the fixed official provider for a base currency.
// Authored by: OpenCode
func expectedProviderForBaseCurrency(baseCurrency string) (ProviderID, RateAuthority, bool) {
	switch baseCurrency {
	case BaseCurrencyEUR:
		return ProviderIDECBEXR, RateAuthorityEuropeanCentralBank, true
	case BaseCurrencyUSD:
		return ProviderIDFederalReserveH10, RateAuthorityFederalReserve, true
	default:
		return "", "", false
	}
}
