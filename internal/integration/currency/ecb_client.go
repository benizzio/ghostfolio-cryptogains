// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
)

// ecbEXRClient resolves EUR-base rates from ECB EXR CSV data.
// Authored by: OpenCode
type ecbEXRClient struct {
	baseURL    string
	httpClient *http.Client
}

// baseCurrency returns EUR for the ECB EXR provider.
// Authored by: OpenCode
func (client *ecbEXRClient) baseCurrency() string {
	return BaseCurrencyEUR
}

// providerCategory returns the canonical provider category for ECB EXR.
// Authored by: OpenCode
func (client *ecbEXRClient) providerCategory() ProviderID {
	return ProviderIDECBEXR
}

// lookupRate requests ECB EXR CSV data and maps it to canonical evidence.
// Authored by: OpenCode
func (client *ecbEXRClient) lookupRate(ctx context.Context, request RateLookupRequest) (ExchangeRateEvidence, error) {
	return client.LookupRate(ctx, request)
}

// LookupRate requests ECB EXR CSV data and maps it to canonical evidence.
// Authored by: OpenCode
func (client *ecbEXRClient) LookupRate(ctx context.Context, request RateLookupRequest) (ExchangeRateEvidence, error) {
	if err := validateSupportedECBSourceCurrency(request.SourceCurrency); err != nil {
		return ExchangeRateEvidence{}, err
	}

	var endpoint, datasetReference, err = client.ecbURL(request)
	if err != nil {
		return ExchangeRateEvidence{}, err
	}
	var payload, fetchErr = fetchProviderPayload(ctx, client.httpClient, endpoint)
	if fetchErr != nil {
		return ExchangeRateEvidence{}, fetchErr
	}

	return MapECBEXRCSVToEvidence(request, payload, datasetReference)
}

// newECBEXRClient creates one ECB provider client.
// Authored by: OpenCode
func newECBEXRClient(baseURL string, httpClient *http.Client) *ecbEXRClient {
	return &ecbEXRClient{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), httpClient: httpClient}
}

// ecbURL builds the fixed ECB EXR request URL for one lookup.
// Authored by: OpenCode
func (client *ecbEXRClient) ecbURL(request RateLookupRequest) (string, string, error) {
	var datasetReference = "EXR/D." + request.SourceCurrency + ".EUR.SP00.A"
	var parsed, err = url.Parse(client.baseURL + "/service/data/" + datasetReference)
	if err != nil {
		return "", "", fmt.Errorf("build ECB EXR URL: %w", err)
	}
	var query = parsed.Query()
	query.Set("startPeriod", datesupport.FormatCalendarDate(request.ActivityDate.AddDate(0, 0, -providerLookbackDays)))
	query.Set("endPeriod", datesupport.FormatCalendarDate(request.ActivityDate))
	query.Set("detail", "dataonly")
	query.Set("format", "csvdata")
	parsed.RawQuery = query.Encode()

	return parsed.String(), datasetReference, nil
}

// validateSupportedECBSourceCurrency rejects absent or suspended ECB source currencies.
// Authored by: OpenCode
func validateSupportedECBSourceCurrency(sourceCurrency string) error {
	if !supportedECBSourceCurrencies[sourceCurrency] {
		return fmt.Errorf("unsupported source currency %s for ECB EXR", sourceCurrency)
	}

	return nil
}

var supportedECBSourceCurrencies = map[string]bool{
	"AUD": true, "BRL": true, "CAD": true, "CHF": true, "CNY": true, "CZK": true, "DKK": true, "GBP": true, "HKD": true, "HUF": true,
	"IDR": true, "ILS": true, "INR": true, "ISK": true, "JPY": true, "KRW": true, "MXN": true, "MYR": true, "NOK": true, "NZD": true,
	"PHP": true, "PLN": true, "RON": true, "SEK": true, "SGD": true, "THB": true, "TRY": true, "USD": true, "ZAR": true,
}
