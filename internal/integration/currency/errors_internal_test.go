// Package currency verifies package-local provider failure classification for
// report base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestConversionFailureClassification verifies that provider and request errors
// expose the US3 conversion-failure taxonomy without leaking unsafe details.
// Authored by: OpenCode
func TestConversionFailureClassification(t *testing.T) {
	var activityDate = time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	var request = RateLookupRequest{SourceCurrency: "EUR", BaseCurrency: BaseCurrencyUSD, ActivityDate: activityDate}
	var cases = []struct {
		name     string
		reason   ConversionFailureReason
		provider ProviderID
		detail   string
	}{
		{name: "unsupported currency", reason: ConversionFailureReasonUnsupportedCurrency, provider: ProviderIDFederalReserveH10, detail: "RUB is unsupported"},
		{name: "missing rate", reason: ConversionFailureReasonMissingRate, provider: ProviderIDECBEXR, detail: "no current or prior observation"},
		{name: "provider unavailable", reason: ConversionFailureReasonProviderUnavailable, provider: ProviderIDFederalReserveH10, detail: "Bearer jwt-secret token token-123 upstream timeout"},
		{name: "malformed rate", reason: ConversionFailureReasonMalformedRate, provider: ProviderIDECBEXR, detail: "amount 1000.25 is not parseable"},
		{name: "ambiguous quote", reason: ConversionFailureReasonAmbiguousQuote, provider: ProviderIDFederalReserveH10, detail: "quote direction is ambiguous"},
		{name: "invalid activity currency", reason: ConversionFailureReasonInvalidActivityCurrency, provider: "", detail: "EU is malformed"},
		{name: "authority mismatch", reason: ConversionFailureReasonAuthorityMismatch, provider: ProviderIDECBEXR, detail: "provider returned USD for EUR request"},
	}

	for _, testCase := range cases {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var err = NewConversionFailure(request, testCase.provider, testCase.reason, testCase.detail)
			var failure *ConversionFailure
			if !errors.As(err, &failure) {
				t.Fatalf("expected conversion failure error, got %T", err)
			}
			if failure.Reason != testCase.reason || failure.SourceCurrency != request.SourceCurrency || failure.ReportBaseCurrency != request.BaseCurrency || !failure.ActivityDate.Equal(activityDate) || failure.ProviderID != testCase.provider {
				t.Fatalf("unexpected failure classification: %#v", failure)
			}
			var safeMessage = failure.SafeMessage()
			for _, expected := range []string{string(testCase.reason), request.SourceCurrency, request.BaseCurrency, "2024-01-15"} {
				if !strings.Contains(safeMessage, expected) {
					t.Fatalf("expected safe message to contain %q, got %q", expected, safeMessage)
				}
			}
			for _, forbidden := range []string{"token-123", "jwt-secret", "Bearer jwt-secret", "1000.25"} {
				if strings.Contains(safeMessage, forbidden) {
					t.Fatalf("expected safe message to exclude %q, got %q", forbidden, safeMessage)
				}
			}
		})
	}
}

// TestConversionFailureReasonOf verifies reason extraction through wrapped
// provider errors.
// Authored by: OpenCode
func TestConversionFailureReasonOf(t *testing.T) {
	var request = RateLookupRequest{SourceCurrency: "GBP", BaseCurrency: BaseCurrencyUSD, ActivityDate: time.Date(2024, time.March, 4, 0, 0, 0, 0, time.UTC)}
	var err = errors.New("plain error")
	if _, ok := ConversionFailureReasonOf(err); ok {
		t.Fatalf("expected plain error to have no conversion failure reason")
	}

	err = errors.Join(err, NewConversionFailure(request, ProviderIDFederalReserveH10, ConversionFailureReasonProviderUnavailable, "upstream timeout"))
	var reason, ok = ConversionFailureReasonOf(err)
	if !ok || reason != ConversionFailureReasonProviderUnavailable {
		t.Fatalf("expected provider unavailable reason through wrapped error, got %q %v", reason, ok)
	}
}
