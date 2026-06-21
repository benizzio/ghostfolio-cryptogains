// Package currency verifies foundational currency integration contracts.
// Authored by: OpenCode
package currency

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// testRateProvider returns deterministic evidence for service scaffold tests.
// Authored by: OpenCode
type testRateProvider struct {
	baseCurrencyValue string
	evidence          ExchangeRateEvidence
	err               error
	calls             int
}

// baseCurrency returns the configured provider base currency.
// Authored by: OpenCode
func (provider *testRateProvider) baseCurrency() string {
	return provider.baseCurrencyValue
}

// lookupRate returns deterministic evidence and tracks provider calls.
// Authored by: OpenCode
func (provider *testRateProvider) lookupRate(context.Context, RateLookupRequest) (ExchangeRateEvidence, error) {
	provider.calls++
	if provider.err != nil {
		return ExchangeRateEvidence{}, provider.err
	}

	return provider.evidence, nil
}

// TestRateLookupRequestValidation verifies canonical public lookup request rules.
// Authored by: OpenCode
func TestRateLookupRequestValidation(t *testing.T) {
	t.Parallel()

	var sourceDate = time.Date(2024, time.January, 2, 15, 30, 0, 0, time.FixedZone("source", 3600))
	var request, err = NewRateLookupRequest(" USD ", BaseCurrencyEUR, sourceDate)
	if err != nil {
		t.Fatalf("expected trimmed uppercase request to validate: %v", err)
	}
	if request.SourceCurrency != "USD" || request.BaseCurrency != BaseCurrencyEUR {
		t.Fatalf("unexpected normalized currencies: %#v", request)
	}
	if request.ActivityDate != time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("unexpected canonical activity date: %s", request.ActivityDate)
	}

	var testCases = []struct {
		name    string
		request RateLookupRequest
		want    string
	}{
		{name: "missing source", request: RateLookupRequest{BaseCurrency: BaseCurrencyUSD, ActivityDate: sourceDate}, want: "source currency is required"},
		{name: "lowercase source", request: RateLookupRequest{SourceCurrency: "usd", BaseCurrency: BaseCurrencyEUR, ActivityDate: sourceDate}, want: "three-letter uppercase currency code"},
		{name: "bad source length", request: RateLookupRequest{SourceCurrency: "US", BaseCurrency: BaseCurrencyEUR, ActivityDate: sourceDate}, want: "three-letter uppercase currency code"},
		{name: "unsupported base", request: RateLookupRequest{SourceCurrency: "USD", BaseCurrency: "GBP", ActivityDate: sourceDate}, want: "unsupported base currency"},
		{name: "same currency", request: RateLookupRequest{SourceCurrency: BaseCurrencyUSD, BaseCurrency: BaseCurrencyUSD, ActivityDate: sourceDate}, want: "must differ"},
		{name: "missing activity date", request: RateLookupRequest{SourceCurrency: "USD", BaseCurrency: BaseCurrencyEUR}, want: "activity date is required"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var err = testCase.request.Validate()
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

// TestExchangeRateEvidenceValidation verifies canonical evidence consistency rules.
// Authored by: OpenCode
func TestExchangeRateEvidenceValidation(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var rateValue = mustCurrencyDecimal(t, "1.09")
	var evidence, err = NewExchangeRateEvidence(
		request,
		request.ActivityDate,
		RateAuthorityEuropeanCentralBank,
		ProviderIDECBEXR,
		RateKindECBEXRDailyReference,
		QuoteDirectionSourcePerBase,
		rateValue,
		"EXR/D.USD.EUR.SP00.A",
	)
	if err != nil {
		t.Fatalf("expected evidence to validate: %v", err)
	}
	if err = evidence.Validate(); err != nil {
		t.Fatalf("expected validated evidence to remain valid: %v", err)
	}

	evidence.ProviderID = ProviderIDFederalReserveH10
	if err = evidence.Validate(); err == nil || !strings.Contains(err.Error(), "requires provider") {
		t.Fatalf("expected provider/base mismatch, got %v", err)
	}

	evidence, err = NewExchangeRateEvidence(
		request,
		request.ActivityDate.AddDate(0, 0, 1),
		RateAuthorityEuropeanCentralBank,
		ProviderIDECBEXR,
		RateKindECBEXRDailyReference,
		QuoteDirectionSourcePerBase,
		rateValue,
		"EXR/D.USD.EUR.SP00.A",
	)
	if err == nil || !strings.Contains(err.Error(), "must not be after activity date") {
		t.Fatalf("expected future rate-date rejection, got evidence=%#v err=%v", evidence, err)
	}

	var zeroRate = *apd.New(0, 0)
	_, err = NewExchangeRateEvidence(
		request,
		request.ActivityDate,
		RateAuthorityEuropeanCentralBank,
		ProviderIDECBEXR,
		RateKindECBEXRDailyReference,
		QuoteDirectionSourcePerBase,
		zeroRate,
		"EXR/D.USD.EUR.SP00.A",
	)
	if err == nil || !strings.Contains(err.Error(), "rate value") {
		t.Fatalf("expected non-positive rate rejection, got %v", err)
	}
}

// TestCurrencyRateServiceUsesCacheBeforeProvider verifies cache-hit behavior without provider fallback.
// Authored by: OpenCode
func TestCurrencyRateServiceUsesCacheBeforeProvider(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var evidence = mustExchangeRateEvidence(t, request, "1.09")
	var provider = &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, err: errors.New("provider should not be called")}
	var cache = NewCurrencyRateSessionCache()
	if err := cache.Store(request, evidence); err != nil {
		t.Fatalf("store cached evidence: %v", err)
	}

	var service, err = newCurrencyRateService(cache, provider)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	var got, lookupErr = service.LookupRate(context.Background(), request)
	if lookupErr != nil {
		t.Fatalf("expected cache hit: %v", lookupErr)
	}
	if provider.calls != 0 {
		t.Fatalf("expected cache hit to avoid provider call, got %d calls", provider.calls)
	}
	assertCurrencyDecimalString(t, got.RateValue, "1.09")
}

// TestCurrencyRateServiceCachesProviderEvidence verifies provider lookup writes to the session cache.
// Authored by: OpenCode
func TestCurrencyRateServiceCachesProviderEvidence(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var evidence = mustExchangeRateEvidence(t, request, "1.09")
	var provider = &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, evidence: evidence}
	var service, err = newCurrencyRateService(NewCurrencyRateSessionCache(), provider)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	var first, lookupErr = service.LookupRate(context.Background(), request)
	if lookupErr != nil {
		t.Fatalf("expected provider lookup: %v", lookupErr)
	}
	assertCurrencyDecimalString(t, first.RateValue, "1.09")

	provider.evidence.RateValue = mustCurrencyDecimal(t, "2.00")
	var second, secondErr = service.LookupRate(context.Background(), request)
	if secondErr != nil {
		t.Fatalf("expected cache hit after provider lookup: %v", secondErr)
	}
	if provider.calls != 1 {
		t.Fatalf("expected one provider call, got %d", provider.calls)
	}
	assertCurrencyDecimalString(t, second.RateValue, "1.09")
}

// mustRateLookupRequest creates one valid request for tests.
// Authored by: OpenCode
func mustRateLookupRequest(t *testing.T, sourceCurrency string, baseCurrency string) RateLookupRequest {
	t.Helper()

	var request, err = NewRateLookupRequest(sourceCurrency, baseCurrency, time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create rate lookup request: %v", err)
	}

	return request
}

// mustExchangeRateEvidence creates valid ECB evidence for tests.
// Authored by: OpenCode
func mustExchangeRateEvidence(t *testing.T, request RateLookupRequest, rawRate string) ExchangeRateEvidence {
	t.Helper()

	var evidence, err = NewExchangeRateEvidence(
		request,
		request.ActivityDate,
		RateAuthorityEuropeanCentralBank,
		ProviderIDECBEXR,
		RateKindECBEXRDailyReference,
		QuoteDirectionSourcePerBase,
		mustCurrencyDecimal(t, rawRate),
		"EXR/D."+request.SourceCurrency+".EUR.SP00.A",
	)
	if err != nil {
		t.Fatalf("create exchange rate evidence: %v", err)
	}

	return evidence
}

// mustCurrencyDecimal parses one exact decimal for tests.
// Authored by: OpenCode
func mustCurrencyDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", raw, err)
	}

	return value
}

// assertCurrencyDecimalString verifies one canonical decimal string.
// Authored by: OpenCode
func assertCurrencyDecimalString(t *testing.T, value apd.Decimal, expected string) {
	t.Helper()

	var actual, err = decimalsupport.CanonicalString(value)
	if err != nil {
		t.Fatalf("format decimal: %v", err)
	}
	if actual != expected {
		t.Fatalf("unexpected decimal: got %s want %s", actual, expected)
	}
}
