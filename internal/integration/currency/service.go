// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

const (
	// BaseCurrencyUSD identifies USD as a supported report base currency.
	BaseCurrencyUSD = "USD"

	// BaseCurrencyEUR identifies EUR as a supported report base currency.
	BaseCurrencyEUR = "EUR"
)

const (
	defaultECBEXRBaseURL = "https://data-api.ecb.europa.eu"
	providerLookbackDays = 30
)

// RateLookupRequest is the public canonical request for one required official
// source-to-base conversion rate. It intentionally excludes provider IDs and
// provider URLs so provider selection remains inside this package.
//
// Example:
//
//	request, err := currency.NewRateLookupRequest("USD", currency.BaseCurrencyEUR, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = request.ActivityDate
//
// Authored by: OpenCode
type RateLookupRequest struct {
	SourceCurrency string
	BaseCurrency   string
	ActivityDate   time.Time
}

// CurrencyRateService resolves canonical exchange-rate evidence for report
// calculation. Implementations must use fixed official providers selected from
// the validated base currency and must not persist evidence.
//
// Example:
//
//	service := currency.NewCurrencyRateService(currency.NewCurrencyRateSessionCache())
//	request, _ := currency.NewRateLookupRequest("USD", currency.BaseCurrencyEUR, time.Now())
//	_, _ = service.LookupRate(context.Background(), request)
//
// Authored by: OpenCode
type CurrencyRateService interface {
	LookupRate(context.Context, RateLookupRequest) (ExchangeRateEvidence, error)
	SupportedBaseCurrencies() []string
	ProviderCategoryForBaseCurrency(string) string
}

// currencyRateService is the cache-aware service scaffold used by later fixed
// official provider adapters.
// Authored by: OpenCode
type currencyRateService struct {
	cache     *CurrencyRateSessionCache
	providers map[string]officialRateProvider
}

// officialRateProvider resolves one base currency through a fixed official provider.
// Authored by: OpenCode
type officialRateProvider interface {
	baseCurrency() string
	providerCategory() ProviderID
	lookupRate(context.Context, RateLookupRequest) (ExchangeRateEvidence, error)
}

// NewRateLookupRequest creates one validated public rate lookup request.
//
// Example:
//
//	request, err := currency.NewRateLookupRequest("GBP", currency.BaseCurrencyUSD, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = request.SourceCurrency
//
// Authored by: OpenCode
func NewRateLookupRequest(sourceCurrency string, baseCurrency string, activityDate time.Time) (RateLookupRequest, error) {
	var request = RateLookupRequest{
		SourceCurrency: trimCurrencyCode(sourceCurrency),
		BaseCurrency:   trimCurrencyCode(baseCurrency),
		ActivityDate:   canonicalDate(activityDate),
	}

	if err := request.Validate(); err != nil {
		return RateLookupRequest{}, err
	}

	return request, nil
}

// Validate verifies that a public lookup request is complete and limited to the
// canonical source/base/date identity required by the rate service.
// Authored by: OpenCode
func (request RateLookupRequest) Validate() error {
	if err := validateCurrencyCode(request.SourceCurrency, "source currency"); err != nil {
		return fmt.Errorf("rate lookup request: %w", err)
	}
	if err := validateBaseCurrency(request.BaseCurrency); err != nil {
		return fmt.Errorf("rate lookup request: %w", err)
	}
	if request.SourceCurrency == request.BaseCurrency {
		return fmt.Errorf("rate lookup request source currency and base currency must differ")
	}
	if request.ActivityDate.IsZero() {
		return fmt.Errorf("rate lookup request activity date is required")
	}

	return nil
}

// NewCurrencyRateService creates the cache-aware public rate service with fixed
// official provider origins selected internally by base currency.
//
// Example:
//
//	service := currency.NewCurrencyRateService(currency.NewCurrencyRateSessionCache())
//	_ = service.SupportedBaseCurrencies()
//
// Authored by: OpenCode
func NewCurrencyRateService(cache *CurrencyRateSessionCache) CurrencyRateService {
	var service, _ = newCurrencyRateService(
		cache,
		newECBEXRClient(defaultECBEXRBaseURL, http.DefaultClient),
		newFederalReserveH10Client(defaultFederalReserveH10BaseURL, defaultFederalReserveH10Dataset, http.DefaultClient),
	)
	return service
}

// SupportedBaseCurrencies returns the supported report base currencies in UI order.
//
// Example:
//
//	baseCurrencies := currency.SupportedBaseCurrencies()
//	_ = baseCurrencies[0]
//
// Authored by: OpenCode
func SupportedBaseCurrencies() []string {
	return []string{BaseCurrencyUSD, BaseCurrencyEUR}
}

// IsSupportedBaseCurrency reports whether the currency can be selected as a report base.
//
// Example:
//
//	if !currency.IsSupportedBaseCurrency(currency.BaseCurrencyUSD) {
//		panic("unsupported base currency")
//	}
//
// Authored by: OpenCode
func IsSupportedBaseCurrency(baseCurrency string) bool {
	return validateBaseCurrency(baseCurrency) == nil
}

// LookupRate resolves canonical rate evidence from the session cache or the
// fixed official provider selected by base currency.
// Authored by: OpenCode
func (service *currencyRateService) LookupRate(ctx context.Context, request RateLookupRequest) (ExchangeRateEvidence, error) {
	if service == nil {
		return ExchangeRateEvidence{}, fmt.Errorf("currency rate service is required")
	}
	if ctx == nil {
		return ExchangeRateEvidence{}, fmt.Errorf("rate lookup context is required")
	}

	request.ActivityDate = canonicalDate(request.ActivityDate)
	if err := request.Validate(); err != nil {
		return ExchangeRateEvidence{}, NewConversionFailure(request, providerIDForBaseCurrency(request.BaseCurrency), conversionFailureReasonForRequestError(err), err.Error())
	}
	if evidence, ok := service.cache.Get(request); ok {
		return evidence, nil
	}

	var provider, ok = service.providers[request.BaseCurrency]
	if !ok {
		return ExchangeRateEvidence{}, NewConversionFailure(request, providerIDForBaseCurrency(request.BaseCurrency), ConversionFailureReasonProviderUnavailable, "official rate provider is not configured")
	}

	var evidence, err = provider.lookupRate(ctx, request)
	if err != nil {
		return ExchangeRateEvidence{}, classifyLookupFailure(request, providerIDForBaseCurrency(request.BaseCurrency), err)
	}
	evidence = cloneExchangeRateEvidence(evidence)
	if err = evidence.Validate(); err != nil {
		return ExchangeRateEvidence{}, classifyEvidenceFailure(request, providerIDForBaseCurrency(request.BaseCurrency), err)
	}
	if !evidence.matchesRequest(request) {
		return ExchangeRateEvidence{}, NewConversionFailure(request, providerIDForBaseCurrency(request.BaseCurrency), ConversionFailureReasonAuthorityMismatch, "rate provider evidence does not match lookup request")
	}
	if err = service.cache.Store(request, evidence); err != nil {
		return ExchangeRateEvidence{}, classifyEvidenceFailure(request, providerIDForBaseCurrency(request.BaseCurrency), err)
	}

	return cloneExchangeRateEvidence(evidence), nil
}

// SupportedBaseCurrencies returns the base currencies supported by this service.
// Authored by: OpenCode
func (service *currencyRateService) SupportedBaseCurrencies() []string {
	return SupportedBaseCurrencies()
}

// ProviderCategoryForBaseCurrency returns the provider category configured for
// one supported report base currency.
// Authored by: OpenCode
func (service *currencyRateService) ProviderCategoryForBaseCurrency(baseCurrency string) string {
	if service == nil {
		return ""
	}

	var provider, ok = service.providers[trimCurrencyCode(baseCurrency)]
	if !ok || provider == nil {
		return ""
	}

	return string(provider.providerCategory())
}

// classifyLookupFailure converts provider and mapper failures into the public
// conversion-failure contract.
// Authored by: OpenCode
func classifyLookupFailure(request RateLookupRequest, providerID ProviderID, err error) error {
	var failure *ConversionFailure
	if errors.As(err, &failure) {
		return err
	}

	return NewConversionFailure(request, providerID, conversionFailureReasonForProviderError(err), err.Error())
}

// classifyEvidenceFailure converts malformed or mismatched canonical evidence
// into the public conversion-failure contract.
// Authored by: OpenCode
func classifyEvidenceFailure(request RateLookupRequest, providerID ProviderID, err error) error {
	var failure *ConversionFailure
	if errors.As(err, &failure) {
		return err
	}
	var reason = ConversionFailureReasonMalformedRate
	var message = strings.ToLower(err.Error())
	if strings.Contains(message, "quote direction") {
		reason = ConversionFailureReasonAmbiguousQuote
	} else if strings.Contains(message, "provider selection") || strings.Contains(message, "requires provider") || strings.Contains(message, "requires authority") || strings.Contains(message, "lookup identity") || strings.Contains(message, "does not match") {
		reason = ConversionFailureReasonAuthorityMismatch
	}

	return NewConversionFailure(request, providerID, reason, err.Error())
}

// conversionFailureReasonForRequestError classifies malformed public lookup
// requests before provider IO is attempted.
// Authored by: OpenCode
func conversionFailureReasonForRequestError(err error) ConversionFailureReason {
	var message = strings.ToLower(err.Error())
	if strings.Contains(message, "unsupported base currency") {
		return ConversionFailureReasonUnsupportedCurrency
	}

	return ConversionFailureReasonInvalidActivityCurrency
}

// conversionFailureReasonForProviderError classifies provider lookup and mapper
// errors without exposing raw provider details to callers.
// Authored by: OpenCode
func conversionFailureReasonForProviderError(err error) ConversionFailureReason {
	var message = strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unsupported source currency") || strings.Contains(message, "unsupported base currency"):
		return ConversionFailureReasonUnsupportedCurrency
	case strings.Contains(message, "no current or prior available observation"):
		return ConversionFailureReasonMissingRate
	case strings.Contains(message, "ambiguous quote direction"):
		return ConversionFailureReasonAmbiguousQuote
	case strings.Contains(message, "provider returned http status") || strings.Contains(message, "request provider evidence") || strings.Contains(message, "read provider evidence") || strings.Contains(message, "build provider request"):
		return ConversionFailureReasonProviderUnavailable
	case strings.Contains(message, "provider") && strings.Contains(message, "context"):
		return ConversionFailureReasonProviderUnavailable
	default:
		return ConversionFailureReasonMalformedRate
	}
}

// providerIDForBaseCurrency returns the internally selected provider category
// when the base currency is supported.
// Authored by: OpenCode
func providerIDForBaseCurrency(baseCurrency string) ProviderID {
	var providerID, _, ok = expectedProviderForBaseCurrency(baseCurrency)
	if !ok {
		return ""
	}

	return providerID
}

// newCurrencyRateService creates one service with package-local provider injection.
// Authored by: OpenCode
func newCurrencyRateService(cache *CurrencyRateSessionCache, providers ...officialRateProvider) (*currencyRateService, error) {
	if cache == nil {
		cache = NewCurrencyRateSessionCache()
	}

	var service = &currencyRateService{
		cache:     cache,
		providers: map[string]officialRateProvider{},
	}
	for _, provider := range providers {
		if provider == nil {
			return nil, fmt.Errorf("official rate provider is required")
		}

		var baseCurrency = provider.baseCurrency()
		if err := validateBaseCurrency(baseCurrency); err != nil {
			return nil, fmt.Errorf("official rate provider base currency: %w", err)
		}
		if _, exists := service.providers[baseCurrency]; exists {
			return nil, fmt.Errorf("official rate provider for base currency %q is already configured", baseCurrency)
		}
		service.providers[baseCurrency] = provider
	}

	return service, nil
}

// validateBaseCurrency rejects report base currencies outside the supported set.
// Authored by: OpenCode
func validateBaseCurrency(baseCurrency string) error {
	if err := validateCurrencyCode(baseCurrency, "base currency"); err != nil {
		return err
	}
	switch baseCurrency {
	case BaseCurrencyUSD, BaseCurrencyEUR:
		return nil
	default:
		return fmt.Errorf("unsupported base currency %q", baseCurrency)
	}
}

// validateCurrencyCode verifies one uppercase three-letter currency identity.
// Authored by: OpenCode
func validateCurrencyCode(currencyCode string, label string) error {
	if currencyCode == "" {
		return fmt.Errorf("%s is required", label)
	}
	if trimCurrencyCode(currencyCode) != currencyCode {
		return fmt.Errorf("%s must not contain surrounding whitespace", label)
	}
	if len(currencyCode) != 3 {
		return fmt.Errorf("%s %q must be a three-letter uppercase currency code", label, currencyCode)
	}
	for _, character := range currencyCode {
		if character < 'A' || character > 'Z' {
			return fmt.Errorf("%s %q must be a three-letter uppercase currency code", label, currencyCode)
		}
	}

	return nil
}

// trimCurrencyCode removes non-semantic surrounding whitespace from constructor input.
// Authored by: OpenCode
func trimCurrencyCode(currencyCode string) string {
	return strings.TrimSpace(currencyCode)
}

// canonicalDate strips clock and location fields from a source-calendar date.
// Authored by: OpenCode
func canonicalDate(value time.Time) time.Time {
	return datesupport.CalendarDate(value)
}

// formatDate returns the canonical YYYY-MM-DD rendering for diagnostics.
// Authored by: OpenCode
func formatDate(value time.Time) string {
	return datesupport.FormatCalendarDate(value)
}

// fetchProviderPayload performs one fixed-provider HTTP GET and returns the body.
// Authored by: OpenCode
func fetchProviderPayload(ctx context.Context, httpClient *http.Client, endpoint string) ([]byte, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	var request, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build provider request: %w", err)
	}
	var response, doErr = httpClient.Do(request)
	if doErr != nil {
		return nil, fmt.Errorf("request provider evidence: %w", doErr)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil, fmt.Errorf("provider returned HTTP status %d", response.StatusCode)
	}
	var payload, readErr = io.ReadAll(response.Body)
	if readErr != nil {
		return nil, fmt.Errorf("read provider evidence: %w", readErr)
	}

	return payload, nil
}

// parsePositiveRate parses one positive exact provider rate without float math.
// Authored by: OpenCode
func parsePositiveRate(rawRate string) (apd.Decimal, error) {
	var rate, _, err = decimalsupport.ParseString(rawRate)
	if err != nil {
		return apd.Decimal{}, err
	}
	if err = supportmath.RequirePositive(rate); err != nil {
		return apd.Decimal{}, err
	}

	return rate, nil
}
