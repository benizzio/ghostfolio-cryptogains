// Package currency owns Federal Reserve H.10 HTTP integration for USD report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultFederalReserveH10BaseURL     = "https://www.federalreserve.gov"
	defaultFederalReserveH10Dataset     = "Federal Reserve H.10/Data Download Program"
	federalReserveH10DDPPackageSeriesID = "60f32914ab61dfab590e0e470153e3ae"
)

// federalReserveH10Client resolves USD-base rates from Federal Reserve H.10 CSV
// data using the fixed official Data Download Program endpoint family.
// Authored by: OpenCode
type federalReserveH10Client struct {
	baseURL    string
	datasetID  string
	httpClient *http.Client
}

// baseCurrency returns USD for the Federal Reserve H.10 provider.
// Authored by: OpenCode
func (client *federalReserveH10Client) baseCurrency() string {
	return BaseCurrencyUSD
}

// providerCategory returns the canonical provider category for Federal Reserve H.10.
// Authored by: OpenCode
func (client *federalReserveH10Client) providerCategory() ProviderID {
	return ProviderIDFederalReserveH10
}

// lookupRate requests Federal Reserve H.10 CSV data and maps it to canonical evidence.
// Authored by: OpenCode
func (client *federalReserveH10Client) lookupRate(ctx context.Context, request RateLookupRequest) (ExchangeRateEvidence, error) {
	return client.LookupRate(ctx, request)
}

// LookupRate requests Federal Reserve H.10 CSV data and maps it to canonical evidence.
// Authored by: OpenCode
func (client *federalReserveH10Client) LookupRate(ctx context.Context, request RateLookupRequest) (ExchangeRateEvidence, error) {
	if err := validateSupportedFederalReserveSourceCurrency(request.SourceCurrency); err != nil {
		return ExchangeRateEvidence{}, err
	}

	var endpoint, err = client.federalReserveURL(request)
	if err != nil {
		return ExchangeRateEvidence{}, err
	}
	var payload, fetchErr = fetchProviderPayload(ctx, client.httpClient, endpoint)
	if fetchErr != nil {
		return ExchangeRateEvidence{}, fetchErr
	}

	return MapFederalReserveH10CSVToEvidence(request, payload, client.datasetID)
}

// NewFederalReserveH10ClientForTesting creates a Federal Reserve H.10 provider
// client for deterministic package tests without exposing fixture URLs through
// production construction.
// Authored by: OpenCode
func NewFederalReserveH10ClientForTesting(baseURL string, httpClient *http.Client) *federalReserveH10Client {
	return newFederalReserveH10Client(baseURL, defaultFederalReserveH10Dataset, httpClient)
}

// newFederalReserveH10Client creates one Federal Reserve provider client.
// Authored by: OpenCode
func newFederalReserveH10Client(baseURL string, datasetID string, httpClient *http.Client) *federalReserveH10Client {
	return &federalReserveH10Client{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), datasetID: strings.TrimSpace(datasetID), httpClient: httpClient}
}

// federalReserveURL builds the fixed Federal Reserve H.10 request URL for one lookup.
// Authored by: OpenCode
func (client *federalReserveH10Client) federalReserveURL(request RateLookupRequest) (string, error) {
	var parsed, err = url.Parse(client.baseURL + "/datadownload/Output.aspx")
	if err != nil {
		return "", fmt.Errorf("build Federal Reserve H.10 URL: %w", err)
	}
	var query = parsed.Query()
	query.Set("rel", "H10")
	query.Set("series", federalReserveH10DDPPackageSeriesID)
	query.Set("lastobs", "")
	query.Set("from", formatDate(request.ActivityDate.AddDate(0, 0, -providerLookbackDays)))
	query.Set("to", formatDate(request.ActivityDate))
	query.Set("filetype", "csv")
	query.Set("label", "include")
	query.Set("layout", "seriesrow")
	query.Set("type", "package")
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}
