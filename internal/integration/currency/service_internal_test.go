// Package currency verifies foundational currency integration contracts.
// Authored by: OpenCode
package currency

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// testRateProvider returns deterministic evidence for service scaffold tests.
// Authored by: OpenCode
type testRateProvider struct {
	baseCurrencyValue     string
	providerCategoryValue ProviderID
	evidence              ExchangeRateEvidence
	err                   error
	calls                 int
}

// errorReadCloser fails reads for provider payload read-error coverage.
// Authored by: OpenCode
type errorReadCloser struct{}

// Read always fails to simulate a broken provider body.
// Authored by: OpenCode
func (errorReadCloser) Read([]byte) (int, error) { return 0, errors.New("read boom") }

// Close completes the broken provider body contract.
// Authored by: OpenCode
func (errorReadCloser) Close() error { return nil }

// roundTripFunc adapts a function to http.RoundTripper for provider tests.
// Authored by: OpenCode
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip executes the configured transport callback.
// Authored by: OpenCode
func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

// baseCurrency returns the configured provider base currency.
// Authored by: OpenCode
func (provider *testRateProvider) baseCurrency() string {
	return provider.baseCurrencyValue
}

// providerCategory returns the configured provider category.
// Authored by: OpenCode
func (provider *testRateProvider) providerCategory() ProviderID {
	if provider.providerCategoryValue != "" {
		return provider.providerCategoryValue
	}

	return providerIDForBaseCurrency(provider.baseCurrencyValue)
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

// TestOfficialProviderIdentity verifies concrete provider identity metadata used
// by diagnostics and provider routing.
// Authored by: OpenCode
func TestOfficialProviderIdentity(t *testing.T) {
	t.Parallel()

	var ecbProvider = newECBEXRClient("https://example.test", nil)
	if got := ecbProvider.providerCategory(); got != ProviderIDECBEXR {
		t.Fatalf("expected ECB provider category %q, got %q", ProviderIDECBEXR, got)
	}

	var federalReserveProvider = newFederalReserveH10Client("https://example.test", "dataset", nil)
	if got := federalReserveProvider.providerCategory(); got != ProviderIDFederalReserveH10 {
		t.Fatalf("expected Federal Reserve provider category %q, got %q", ProviderIDFederalReserveH10, got)
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

// TestExchangeRateEvidenceValidationRejectsRemainingMalformedFields verifies
// validator branches that are not exercised by provider mapper fixtures.
// Authored by: OpenCode
func TestExchangeRateEvidenceValidationRejectsRemainingMalformedFields(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var valid = mustExchangeRateEvidence(t, request, "1.09")
	var testCases = []struct {
		name   string
		mutate func(*ExchangeRateEvidence)
		want   string
	}{
		{name: "missing rate date", mutate: func(evidence *ExchangeRateEvidence) { evidence.RateDate = time.Time{} }, want: "rate date is required"},
		{name: "unsupported authority", mutate: func(evidence *ExchangeRateEvidence) { evidence.Authority = RateAuthority("market") }, want: "unsupported rate authority"},
		{name: "unsupported provider", mutate: func(evidence *ExchangeRateEvidence) { evidence.ProviderID = ProviderID("market") }, want: "unsupported provider ID"},
		{name: "authority mismatch", mutate: func(evidence *ExchangeRateEvidence) { evidence.Authority = RateAuthorityFederalReserve }, want: "requires authority"},
		{name: "missing rate kind", mutate: func(evidence *ExchangeRateEvidence) { evidence.RateKind = " \t" }, want: "rate kind is required"},
		{name: "missing dataset reference", mutate: func(evidence *ExchangeRateEvidence) { evidence.DatasetReference = " \t" }, want: "dataset reference is required"},
		{name: "unsupported base identity", mutate: func(evidence *ExchangeRateEvidence) { evidence.BaseCurrency = "GBP" }, want: "unsupported base currency"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var evidence = valid
			testCase.mutate(&evidence)
			var err = evidence.Validate()
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
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

// TestCurrencyRateServiceDefensiveBranches verifies defensive constructor,
// lookup, and classification branches not reached by provider success fixtures.
// Authored by: OpenCode
func TestCurrencyRateServiceDefensiveBranches(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var evidence = mustExchangeRateEvidence(t, request, "1.09")
	var service, err = newCurrencyRateService(nil, &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, evidence: evidence})
	if err != nil {
		t.Fatalf("expected nil cache to be defaulted: %v", err)
	}
	if got := service.SupportedBaseCurrencies(); len(got) != 2 || got[0] != BaseCurrencyUSD || got[1] != BaseCurrencyEUR {
		t.Fatalf("unexpected service base currencies: %#v", got)
	}
	if got := service.ProviderCategoryForBaseCurrency(BaseCurrencyEUR); got != string(ProviderIDECBEXR) {
		t.Fatalf("unexpected provider category for EUR: %q", got)
	}
	if got := SupportedBaseCurrencies(); len(got) != 2 || got[0] != BaseCurrencyUSD || got[1] != BaseCurrencyEUR {
		t.Fatalf("unexpected package base currencies: %#v", got)
	}
	if !IsSupportedBaseCurrency(BaseCurrencyUSD) || IsSupportedBaseCurrency("GBP") {
		t.Fatalf("unexpected supported-base-currency result")
	}

	var nilService *currencyRateService
	if _, err = nilService.LookupRate(context.Background(), request); err == nil || !strings.Contains(err.Error(), "currency rate service is required") {
		t.Fatalf("expected nil service failure, got %v", err)
	}
	if got := nilService.ProviderCategoryForBaseCurrency(BaseCurrencyEUR); got != "" {
		t.Fatalf("expected nil service to return no provider category, got %q", got)
	}
	if _, err = service.LookupRate(nil, request); err == nil || !strings.Contains(err.Error(), "rate lookup context is required") {
		t.Fatalf("expected nil context failure, got %v", err)
	}
	if _, err = service.LookupRate(context.Background(), RateLookupRequest{SourceCurrency: "usd", BaseCurrency: BaseCurrencyEUR, ActivityDate: request.ActivityDate}); err == nil {
		t.Fatalf("expected invalid request to become conversion failure")
	}
	var reason ConversionFailureReason
	var ok bool
	reason, ok = ConversionFailureReasonOf(err)
	if !ok || reason != ConversionFailureReasonInvalidActivityCurrency {
		t.Fatalf("expected invalid activity currency reason, got reason=%q ok=%v err=%v", reason, ok, err)
	}

	var noProviderService, noProviderErr = newCurrencyRateService(NewCurrencyRateSessionCache())
	if noProviderErr != nil {
		t.Fatalf("create no-provider service: %v", noProviderErr)
	}
	_, err = noProviderService.LookupRate(context.Background(), request)
	reason, ok = ConversionFailureReasonOf(err)
	if !ok || reason != ConversionFailureReasonProviderUnavailable {
		t.Fatalf("expected no-provider unavailable reason, got reason=%q ok=%v err=%v", reason, ok, err)
	}
	if got := noProviderService.ProviderCategoryForBaseCurrency(BaseCurrencyEUR); got != "" {
		t.Fatalf("expected missing provider to return no category, got %q", got)
	}
	var nilProviderCategoryService = &currencyRateService{providers: map[string]officialRateProvider{BaseCurrencyEUR: nil}}
	if got := nilProviderCategoryService.ProviderCategoryForBaseCurrency(BaseCurrencyEUR); got != "" {
		t.Fatalf("expected nil provider to return no category, got %q", got)
	}

	_, err = newCurrencyRateService(NewCurrencyRateSessionCache(), nil)
	if err == nil || !strings.Contains(err.Error(), "official rate provider is required") {
		t.Fatalf("expected nil provider rejection, got %v", err)
	}
	_, err = newCurrencyRateService(NewCurrencyRateSessionCache(), &testRateProvider{baseCurrencyValue: "GBP"})
	if err == nil || !strings.Contains(err.Error(), "unsupported base currency") {
		t.Fatalf("expected provider base currency rejection, got %v", err)
	}
	_, err = newCurrencyRateService(NewCurrencyRateSessionCache(), &testRateProvider{baseCurrencyValue: BaseCurrencyEUR}, &testRateProvider{baseCurrencyValue: BaseCurrencyEUR})
	if err == nil || !strings.Contains(err.Error(), "already configured") {
		t.Fatalf("expected duplicate provider rejection, got %v", err)
	}

	if got := providerIDForBaseCurrency("GBP"); got != "" {
		t.Fatalf("expected unsupported base currency to have no provider ID, got %q", got)
	}
	if err = validateCurrencyCode(" USD", "source currency"); err == nil || !strings.Contains(err.Error(), "surrounding whitespace") {
		t.Fatalf("expected whitespace validation failure, got %v", err)
	}
	if !datesupport.CalendarDate(time.Time{}).IsZero() {
		t.Fatalf("expected zero canonical date to stay zero")
	}
	if _, err = parsePositiveRate("0"); err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("expected non-positive rate rejection, got %v", err)
	}
	if _, err = NewRateLookupRequest("usd", BaseCurrencyEUR, request.ActivityDate); err == nil || !strings.Contains(err.Error(), "uppercase currency code") {
		t.Fatalf("expected constructor validation failure, got %v", err)
	}
	if err = validateBaseCurrency("usd"); err == nil || !strings.Contains(err.Error(), "uppercase currency code") {
		t.Fatalf("expected base-currency code validation failure, got %v", err)
	}
	if _, _, err = ecbEXRColumnIndexes([]string{"DATE", "OTHER"}); err == nil || !strings.Contains(err.Error(), "required columns") {
		t.Fatalf("expected missing CSV column rejection, got %v", err)
	}
	if err = validateProviderForBaseCurrency("GBP", ProviderIDECBEXR, RateAuthorityEuropeanCentralBank); err == nil || !strings.Contains(err.Error(), "unsupported base currency") {
		t.Fatalf("expected unsupported provider base-currency validation failure, got %v", err)
	}
	if NewCurrencyRateService(nil) == nil {
		t.Fatalf("expected production currency rate service to be constructed")
	}
}

// TestCurrencyRateServiceFailureClassificationBranches verifies provider and
// evidence failures are mapped to the public conversion-failure taxonomy.
// Authored by: OpenCode
func TestCurrencyRateServiceFailureClassificationBranches(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var providerErrors = []struct {
		name   string
		detail string
		reason ConversionFailureReason
	}{
		{name: "wrapped conversion failure", detail: "", reason: ConversionFailureReasonMissingRate},
		{name: "missing observation", detail: "no current or prior available observation", reason: ConversionFailureReasonMissingRate},
		{name: "ambiguous quote", detail: "ambiguous quote direction", reason: ConversionFailureReasonAmbiguousQuote},
		{name: "http status", detail: "provider returned HTTP status 500", reason: ConversionFailureReasonProviderUnavailable},
		{name: "context failure", detail: "provider context canceled", reason: ConversionFailureReasonProviderUnavailable},
		{name: "malformed fallback", detail: "csv payload cannot be parsed", reason: ConversionFailureReasonMalformedRate},
		{name: "unsupported base", detail: "unsupported base currency GBP", reason: ConversionFailureReasonUnsupportedCurrency},
	}
	for _, testCase := range providerErrors {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var providerErr error
			if testCase.name == "wrapped conversion failure" {
				providerErr = NewConversionFailure(request, ProviderIDECBEXR, testCase.reason, "already classified")
			} else {
				providerErr = errors.New(testCase.detail)
			}
			var service, err = newCurrencyRateService(NewCurrencyRateSessionCache(), &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, err: providerErr})
			if err != nil {
				t.Fatalf("create service: %v", err)
			}
			_, err = service.LookupRate(context.Background(), request)
			var reason, ok = ConversionFailureReasonOf(err)
			if !ok || reason != testCase.reason {
				t.Fatalf("expected reason %q, got reason=%q ok=%v err=%v", testCase.reason, reason, ok, err)
			}
		})
	}

	var evidenceFailures = []struct {
		name        string
		mutate      func(*ExchangeRateEvidence)
		wantReason  ConversionFailureReason
		wantMessage string
	}{
		{name: "wrapped conversion failure", mutate: func(evidence *ExchangeRateEvidence) {}, wantReason: ConversionFailureReasonAuthorityMismatch, wantMessage: "already classified"},
		{name: "quote direction", mutate: func(evidence *ExchangeRateEvidence) { evidence.QuoteDirection = QuoteDirection("") }, wantReason: ConversionFailureReasonAmbiguousQuote, wantMessage: "quote direction"},
		{name: "provider selection", mutate: func(evidence *ExchangeRateEvidence) { evidence.ProviderID = ProviderIDFederalReserveH10 }, wantReason: ConversionFailureReasonAuthorityMismatch, wantMessage: "provider selection"},
		{name: "malformed rate", mutate: func(evidence *ExchangeRateEvidence) { evidence.RateValue = *apd.New(0, 0) }, wantReason: ConversionFailureReasonMalformedRate, wantMessage: "rate value"},
		{name: "lookup identity mismatch", mutate: func(evidence *ExchangeRateEvidence) { evidence.SourceCurrency = "GBP" }, wantReason: ConversionFailureReasonAuthorityMismatch, wantMessage: "does not match"},
	}
	for _, testCase := range evidenceFailures {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var evidence = mustExchangeRateEvidence(t, request, "1.09")
			var provider officialRateProvider
			if testCase.name == "wrapped conversion failure" {
				provider = &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, err: NewConversionFailure(request, ProviderIDECBEXR, testCase.wantReason, testCase.wantMessage)}
			} else {
				testCase.mutate(&evidence)
				provider = &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, evidence: evidence}
			}
			var service, err = newCurrencyRateService(NewCurrencyRateSessionCache(), provider)
			if err != nil {
				t.Fatalf("create service: %v", err)
			}
			_, err = service.LookupRate(context.Background(), request)
			var reason, ok = ConversionFailureReasonOf(err)
			if !ok || reason != testCase.wantReason || !strings.Contains(err.Error(), string(testCase.wantReason)) {
				t.Fatalf("expected reason %q, got reason=%q ok=%v err=%v", testCase.wantReason, reason, ok, err)
			}
		})
	}
}

// TestCurrencyRateServiceRemainingFailureBranches verifies conversion-failure
// passthrough and cache-store failure classification branches.
// Authored by: OpenCode
func TestCurrencyRateServiceRemainingFailureBranches(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var existing = NewConversionFailure(request, ProviderIDECBEXR, ConversionFailureReasonMissingRate, "safe")
	if got := classifyEvidenceFailure(request, ProviderIDECBEXR, existing); got != existing {
		t.Fatalf("expected existing conversion failure passthrough")
	}

	var evidence = mustExchangeRateEvidence(t, request, "1.09")
	var provider = &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, evidence: evidence}
	var service = &currencyRateService{cache: nil, providers: map[string]officialRateProvider{BaseCurrencyEUR: provider}}
	var _, err = service.LookupRate(context.Background(), request)
	var reason, ok = ConversionFailureReasonOf(err)
	if !ok || reason != ConversionFailureReasonMalformedRate {
		t.Fatalf("expected nil-cache store failure to classify as malformed rate, got reason=%q ok=%v err=%v", reason, ok, err)
	}

	_, err = service.LookupRate(context.Background(), RateLookupRequest{SourceCurrency: "USD", BaseCurrency: "GBP", ActivityDate: request.ActivityDate})
	reason, ok = ConversionFailureReasonOf(err)
	if !ok || reason != ConversionFailureReasonUnsupportedCurrency {
		t.Fatalf("expected unsupported base request reason, got reason=%q ok=%v err=%v", reason, ok, err)
	}
}

// TestFetchProviderPayloadDefensiveBranches verifies HTTP transport failures
// are converted into stable provider errors for later classification.
// Authored by: OpenCode
func TestFetchProviderPayloadDefensiveBranches(t *testing.T) {
	t.Parallel()

	if _, err := fetchProviderPayload(context.Background(), http.DefaultClient, "%"); err == nil || !strings.Contains(err.Error(), "build provider request") {
		t.Fatalf("expected request build failure, got %v", err)
	}

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte("provider down"))
	}))
	defer server.Close()
	if _, err := fetchProviderPayload(context.Background(), nil, server.URL); err == nil || !strings.Contains(err.Error(), "provider returned HTTP status 502") {
		t.Fatalf("expected status failure, got %v", err)
	}

	var requestErrClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial boom")
	})}
	if _, err := fetchProviderPayload(context.Background(), requestErrClient, "http://example.invalid"); err == nil || !strings.Contains(err.Error(), "request provider evidence") {
		t.Fatalf("expected request failure, got %v", err)
	}

	var readErrClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: errorReadCloser{}, Header: make(http.Header)}, nil
	})}
	if _, err := fetchProviderPayload(context.Background(), readErrClient, "http://example.invalid"); err == nil || !strings.Contains(err.Error(), "read provider evidence") {
		t.Fatalf("expected read failure, got %v", err)
	}
}

// TestFetchProviderPayloadAddsDeadlineForUnboundedContext verifies provider IO
// is bounded when callers pass a context without a deadline.
// Authored by: OpenCode
func TestFetchProviderPayloadAddsDeadlineForUnboundedContext(t *testing.T) {
	t.Parallel()

	var deadlineSeen bool
	var client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var _, ok = request.Context().Deadline()
		deadlineSeen = ok
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
	})}
	if _, err := fetchProviderPayload(context.Background(), client, "http://example.invalid"); err != nil {
		t.Fatalf("expected bounded provider request to succeed: %v", err)
	}
	if !deadlineSeen {
		t.Fatalf("expected provider request context to have a deadline")
	}
}

// TestFetchProviderPayloadPreservesCallerDeadline verifies provider IO does not
// replace an existing caller deadline.
// Authored by: OpenCode
func TestFetchProviderPayloadPreservesCallerDeadline(t *testing.T) {
	t.Parallel()

	var callerDeadline = time.Now().Add(time.Hour).UTC()
	var ctx, cancel = context.WithDeadline(context.Background(), callerDeadline)
	defer cancel()

	var seenDeadline time.Time
	var client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var ok bool
		seenDeadline, ok = request.Context().Deadline()
		if !ok {
			t.Fatalf("expected provider request context to keep caller deadline")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
	})}
	if _, err := fetchProviderPayload(ctx, client, "http://example.invalid"); err != nil {
		t.Fatalf("expected provider request with caller deadline to succeed: %v", err)
	}
	if !seenDeadline.Equal(callerDeadline) {
		t.Fatalf("expected caller deadline %s, got %s", callerDeadline, seenDeadline)
	}
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
