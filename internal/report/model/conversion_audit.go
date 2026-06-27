// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
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

// ConversionAuditEntry stores report-visible conversion evidence for one priced
// activity that required exchange-rate conversion.
// Authored by: OpenCode
type ConversionAuditEntry struct {
	SourceID           string
	AssetLabel         string
	ActivityDate       time.Time
	SourceCurrency     string
	ReportBaseCurrency ReportBaseCurrency
	RateDate           time.Time
	RateAuthority      RateAuthority
	RateKind           string
	RateValue          apd.Decimal
	QuoteDirection     QuoteDirection
	Amounts            []ConvertedActivityAmount
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

// Validate verifies one converted activity amount and its rate evidence
// relationship.
// Authored by: OpenCode
func (amount ConvertedActivityAmount) Validate() error {
	if strings.TrimSpace(amount.SourceID) == "" {
		return fmt.Errorf("converted activity amount source ID is required")
	}
	if err := validateConvertedAmountKind(amount.AmountKind); err != nil {
		return fmt.Errorf("converted activity amount kind: %w", err)
	}
	if strings.TrimSpace(amount.OriginalCurrency) == "" {
		return fmt.Errorf("converted activity amount original currency is required")
	}
	if err := validateReportBaseCurrency(amount.ReportBaseCurrency); err != nil {
		return fmt.Errorf("converted activity amount report base currency: %w", err)
	}
	if err := validateNonNegativeDecimal(amount.OriginalAmount, "converted activity amount original amount"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(amount.ConvertedAmount, "converted activity amount converted amount"); err != nil {
		return err
	}
	if err := validateConversionStatus(amount.ConversionStatus); err != nil {
		return fmt.Errorf("converted activity amount status: %w", err)
	}

	if amount.ConversionStatus == ConversionStatusSameCurrency {
		return validateSameCurrencyAmount(amount)
	}

	return validateConvertedAmount(amount)
}

// Validate verifies one report-visible conversion audit entry and all converted
// amounts attached to it.
// Authored by: OpenCode
func (entry ConversionAuditEntry) Validate() error {
	if err := entry.validateIdentity(); err != nil {
		return err
	}
	if err := entry.validateRateEvidence(); err != nil {
		return err
	}
	if len(entry.Amounts) == 0 {
		return fmt.Errorf("conversion audit entry amounts are required")
	}

	for index, amount := range entry.Amounts {
		if err := entry.validateAmount(index, amount); err != nil {
			return err
		}
	}

	return nil
}

// validateIdentity verifies the source activity identity and currency fields for
// one conversion audit entry.
// Authored by: OpenCode
func (entry ConversionAuditEntry) validateIdentity() error {
	if strings.TrimSpace(entry.SourceID) == "" {
		return fmt.Errorf("conversion audit entry source ID is required")
	}
	if strings.TrimSpace(entry.AssetLabel) == "" {
		return fmt.Errorf("conversion audit entry asset label is required")
	}
	if entry.ActivityDate.IsZero() {
		return fmt.Errorf("conversion audit entry activity date is required")
	}
	if strings.TrimSpace(entry.SourceCurrency) == "" {
		return fmt.Errorf("conversion audit entry source currency is required")
	}
	if err := validateReportBaseCurrency(entry.ReportBaseCurrency); err != nil {
		return fmt.Errorf("conversion audit entry report base currency: %w", err)
	}
	if strings.TrimSpace(entry.SourceCurrency) == entry.ReportBaseCurrency.Label() {
		return fmt.Errorf("conversion audit entry source currency must differ from report base currency")
	}

	return nil
}

// validateRateEvidence verifies the provider-level evidence retained by one
// conversion audit entry.
// Authored by: OpenCode
func (entry ConversionAuditEntry) validateRateEvidence() error {
	if entry.RateDate.IsZero() {
		return fmt.Errorf("conversion audit entry rate date is required")
	}
	if entry.RateDate.After(entry.ActivityDate) {
		return fmt.Errorf("conversion audit entry rate date must not be after activity date")
	}
	if err := validateRateAuthority(entry.RateAuthority); err != nil {
		return fmt.Errorf("conversion audit entry rate authority: %w", err)
	}
	if strings.TrimSpace(entry.RateKind) == "" {
		return fmt.Errorf("conversion audit entry rate kind is required")
	}
	if err := validatePositiveDecimal(entry.RateValue, "conversion audit entry rate value"); err != nil {
		return err
	}
	if err := validateQuoteDirection(entry.QuoteDirection); err != nil {
		return fmt.Errorf("conversion audit entry quote direction: %w", err)
	}

	return nil
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

// RateAuthorityDisplayLabel returns the report-facing label for a retained rate
// authority value.
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

// RateProviderDisplayLabel returns the report-facing label for retained provider
// evidence.
// Authored by: OpenCode
func RateProviderDisplayLabel(providerID RateProviderID) string {
	switch providerID {
	case RateProviderIDECBEXR:
		return "ECB Data Portal `EXR`"
	case RateProviderIDFederalReserveH10:
		return "Federal Reserve Board H.10/Data Download Program"
	default:
		return string(providerID)
	}
}

// RateProviderUnavailableDateRule returns the report-facing prior-observation
// disclosure for retained provider evidence.
// Authored by: OpenCode
func RateProviderUnavailableDateRule(providerID RateProviderID) string {
	switch providerID {
	case RateProviderIDECBEXR:
		return "most recent previous available ECB observation"
	case RateProviderIDFederalReserveH10:
		return "most recent previous available H.10 observation"
	default:
		return "most recent previous available official observation"
	}
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
	if err := amount.ExchangeRateEvidence.Validate(); err != nil {
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

// validateAmount verifies one audit amount and checks it against the entry-level
// conversion evidence disclosed in the report.
// Authored by: OpenCode
func (entry ConversionAuditEntry) validateAmount(index int, amount ConvertedActivityAmount) error {
	if err := amount.Validate(); err != nil {
		return fmt.Errorf("conversion audit entry amount %d: %w", index, err)
	}
	if amount.ConversionStatus != ConversionStatusConverted {
		return fmt.Errorf("conversion audit entry amount %d: amount must be converted", index)
	}
	if strings.TrimSpace(amount.SourceID) != strings.TrimSpace(entry.SourceID) {
		return fmt.Errorf("conversion audit entry amount %d: source ID mismatch", index)
	}
	if strings.TrimSpace(amount.OriginalCurrency) != strings.TrimSpace(entry.SourceCurrency) {
		return fmt.Errorf("conversion audit entry amount %d: source currency mismatch", index)
	}
	if amount.ReportBaseCurrency != entry.ReportBaseCurrency {
		return fmt.Errorf("conversion audit entry amount %d: report base currency mismatch", index)
	}
	if amount.ExchangeRateEvidence == nil || !entry.matchesExchangeRateEvidence(*amount.ExchangeRateEvidence) {
		return fmt.Errorf("conversion audit entry amount %d: exchange-rate evidence mismatch", index)
	}

	return nil
}

// matchesExchangeRateEvidence reports whether one amount-level evidence record
// matches the entry-level audit evidence.
// Authored by: OpenCode
func (entry ConversionAuditEntry) matchesExchangeRateEvidence(evidence ExchangeRateEvidence) bool {
	return strings.TrimSpace(evidence.SourceCurrency) == strings.TrimSpace(entry.SourceCurrency) &&
		evidence.BaseCurrency == entry.ReportBaseCurrency &&
		datesupport.CalendarDate(evidence.ActivityDate).Equal(datesupport.CalendarDate(entry.ActivityDate)) &&
		datesupport.CalendarDate(evidence.RateDate).Equal(datesupport.CalendarDate(entry.RateDate)) &&
		evidence.Authority == entry.RateAuthority &&
		strings.TrimSpace(evidence.RateKind) == strings.TrimSpace(entry.RateKind) &&
		evidence.QuoteDirection == entry.QuoteDirection &&
		evidence.RateValue.Cmp(&entry.RateValue) == 0
}
