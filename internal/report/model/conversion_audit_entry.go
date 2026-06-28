// Package model defines report-visible conversion audit entry models.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	"github.com/cockroachdb/apd/v3"
)

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

// cloneConversionAuditEntries copies conversion audit entries for report model
// construction.
// Authored by: OpenCode
func cloneConversionAuditEntries(entries []ConversionAuditEntry) []ConversionAuditEntry {
	var cloned = append([]ConversionAuditEntry(nil), entries...)
	for index := range cloned {
		cloned[index].Amounts = cloneConvertedActivityAmounts(cloned[index].Amounts)
	}

	return cloned
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
