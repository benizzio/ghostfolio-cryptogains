// Package currency owns conversion failure shaping for official exchange-rate
// integration.
// Authored by: OpenCode
package currency

import (
	"errors"
	"fmt"
	"strings"
	"time"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
)

// ConversionFailureReason identifies why official conversion evidence could not
// be defensibly used by report generation.
// Authored by: OpenCode
type ConversionFailureReason string

const (
	// ConversionFailureReasonUnsupportedCurrency means the selected provider does not support the requested currency.
	ConversionFailureReasonUnsupportedCurrency ConversionFailureReason = "unsupported_currency"

	// ConversionFailureReasonMissingRate means no current or prior official observation exists for the request.
	ConversionFailureReasonMissingRate ConversionFailureReason = "missing_rate"

	// ConversionFailureReasonProviderUnavailable means the official provider could not be reached or returned failure status.
	ConversionFailureReasonProviderUnavailable ConversionFailureReason = "provider_unavailable"

	// ConversionFailureReasonMalformedRate means provider evidence was malformed or not a positive exact decimal.
	ConversionFailureReasonMalformedRate ConversionFailureReason = "malformed_rate"

	// ConversionFailureReasonAmbiguousQuote means provider quote direction could not be mapped unambiguously.
	ConversionFailureReasonAmbiguousQuote ConversionFailureReason = "ambiguous_quote"

	// ConversionFailureReasonInvalidActivityCurrency means the activity currency identity is missing or malformed.
	ConversionFailureReasonInvalidActivityCurrency ConversionFailureReason = "invalid_activity_currency"

	// ConversionFailureReasonAuthorityMismatch means evidence identity or authority does not match the lookup request.
	ConversionFailureReasonAuthorityMismatch ConversionFailureReason = "authority_mismatch"
)

// ConversionFailure is the public non-secret error shape for report conversion
// failures. It preserves request identity and provider category when known while
// keeping raw provider details out of SafeMessage output.
// Authored by: OpenCode
type ConversionFailure struct {
	SourceID           string
	SourceCurrency     string
	ReportBaseCurrency string
	ActivityDate       time.Time
	ProviderID         ProviderID
	Reason             ConversionFailureReason
	cause              error
}

// NewConversionFailure creates a classified conversion failure for one rate
// lookup request.
//
// Example:
//
//	request, _ := currency.NewRateLookupRequest("GBP", currency.BaseCurrencyUSD, time.Now())
//	err := currency.NewConversionFailure(request, currency.ProviderIDFederalReserveH10, currency.ConversionFailureReasonMissingRate, "no observation")
//	_ = err
//
// Authored by: OpenCode
func NewConversionFailure(request RateLookupRequest, providerID ProviderID, reason ConversionFailureReason, detail string) error {
	var failure = &ConversionFailure{
		SourceCurrency:     request.SourceCurrency,
		ReportBaseCurrency: request.BaseCurrency,
		ActivityDate:       datesupport.CalendarDate(request.ActivityDate),
		ProviderID:         providerID,
		Reason:             reason,
	}
	if strings.TrimSpace(detail) != "" {
		failure.cause = errors.New(detail)
	}

	return failure
}

// Error returns the same redaction-safe text as SafeMessage.
// Authored by: OpenCode
func (failure *ConversionFailure) Error() string {
	if failure == nil {
		return "conversion failure"
	}

	return failure.SafeMessage()
}

// Unwrap returns the underlying non-public detail for error-chain inspection.
// Authored by: OpenCode
func (failure *ConversionFailure) Unwrap() error {
	if failure == nil {
		return nil
	}

	return failure.cause
}

// SafeMessage returns a user-visible conversion failure message without raw
// provider payloads, tokens, authentication material, or financial amounts.
// Authored by: OpenCode
func (failure *ConversionFailure) SafeMessage() string {
	if failure == nil {
		return "conversion failed"
	}

	var sourceCurrency = safeMessageValue(failure.SourceCurrency)
	var baseCurrency = safeMessageValue(failure.ReportBaseCurrency)
	var activityDate = "unknown"
	if !failure.ActivityDate.IsZero() {
		activityDate = datesupport.FormatCalendarDate(failure.ActivityDate)
	}
	var provider = "unknown"
	if failure.ProviderID != "" {
		provider = string(failure.ProviderID)
	}

	return fmt.Sprintf(
		"conversion failed: reason=%s source_currency=%s report_base_currency=%s activity_date=%s provider=%s",
		failure.Reason,
		sourceCurrency,
		baseCurrency,
		activityDate,
		provider,
	)
}

// ConversionFailureReasonOf extracts a conversion failure reason from an error chain.
// Authored by: OpenCode
func ConversionFailureReasonOf(err error) (ConversionFailureReason, bool) {
	var failure *ConversionFailure
	if !errors.As(err, &failure) || failure == nil {
		return "", false
	}

	return failure.Reason, true
}

// safeMessageValue normalizes empty safe-message fields without adding raw detail.
// Authored by: OpenCode
func safeMessageValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}

	return value
}
