// Package externalintegration contains opt-in live checks for official external
// provider clients.
// Authored by: OpenCode
package externalintegration

import (
	"context"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// TestLiveOfficialCurrencyProvidersResolveFixedHistoricalObservations verifies
// one committed historical observation for each live official provider client.
// Authored by: OpenCode
func TestLiveOfficialCurrencyProvidersResolveFixedHistoricalObservations(t *testing.T) {
	requireExternalIntegration(t)

	var service = currency.NewCurrencyRateService(currency.NewCurrencyRateSessionCache())
	var testCases = []struct {
		name           string
		sourceCurrency string
		baseCurrency   string
		activityDate   string
		rateDate       string
		rateValue      string
		providerID     currency.ProviderID
		authority      currency.RateAuthority
		quoteDirection currency.QuoteDirection
	}{
		{
			name:           "ECB EXR USD to EUR",
			sourceCurrency: "USD",
			baseCurrency:   currency.BaseCurrencyEUR,
			activityDate:   "2024-01-06",
			rateDate:       "2024-01-05",
			rateValue:      "1.0921",
			providerID:     currency.ProviderIDECBEXR,
			authority:      currency.RateAuthorityEuropeanCentralBank,
			quoteDirection: currency.QuoteDirectionSourcePerBase,
		},
		{
			name:           "Federal Reserve H10 EUR to USD",
			sourceCurrency: "EUR",
			baseCurrency:   currency.BaseCurrencyUSD,
			activityDate:   "2024-01-06",
			rateDate:       "2024-01-05",
			rateValue:      "1.0946",
			providerID:     currency.ProviderIDFederalReserveH10,
			authority:      currency.RateAuthorityFederalReserve,
			quoteDirection: currency.QuoteDirectionBasePerSource,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var request = mustLiveRateLookupRequest(t, testCase.sourceCurrency, testCase.baseCurrency, testCase.activityDate)
			var evidence, err = service.LookupRate(context.Background(), request)
			if err != nil {
				t.Fatalf("lookup live provider evidence: %v", err)
			}

			assertLiveProviderEvidence(t, evidence, testCase.providerID, testCase.authority, testCase.quoteDirection, testCase.rateDate, testCase.rateValue)
		})
	}
}

// mustLiveRateLookupRequest creates one external integration lookup request.
// Authored by: OpenCode
func mustLiveRateLookupRequest(t *testing.T, sourceCurrency string, baseCurrency string, rawDate string) currency.RateLookupRequest {
	t.Helper()

	var activityDate, dateErr = time.Parse(time.DateOnly, rawDate)
	if dateErr != nil {
		t.Fatalf("parse activity date %q: %v", rawDate, dateErr)
	}
	var request, requestErr = currency.NewRateLookupRequest(sourceCurrency, baseCurrency, activityDate)
	if requestErr != nil {
		t.Fatalf("create rate lookup request: %v", requestErr)
	}

	return request
}

// assertLiveProviderEvidence verifies canonical live provider evidence fields.
// Authored by: OpenCode
func assertLiveProviderEvidence(t *testing.T, evidence currency.ExchangeRateEvidence, providerID currency.ProviderID, authority currency.RateAuthority, quoteDirection currency.QuoteDirection, rateDate string, rateValue string) {
	t.Helper()

	if evidence.ProviderID != providerID {
		t.Fatalf("unexpected provider ID: got %s want %s", evidence.ProviderID, providerID)
	}
	if evidence.Authority != authority {
		t.Fatalf("unexpected authority: got %s want %s", evidence.Authority, authority)
	}
	if evidence.QuoteDirection != quoteDirection {
		t.Fatalf("unexpected quote direction: got %s want %s", evidence.QuoteDirection, quoteDirection)
	}
	if evidence.RateDate.Format(time.DateOnly) != rateDate {
		t.Fatalf("unexpected rate date: got %s want %s", evidence.RateDate.Format(time.DateOnly), rateDate)
	}
	assertLiveDecimalString(t, evidence.RateValue, rateValue)
}

// assertLiveDecimalString verifies one exact decimal rendering.
// Authored by: OpenCode
func assertLiveDecimalString(t *testing.T, value apd.Decimal, expected string) {
	t.Helper()

	var actual, err = decimalsupport.CanonicalString(value)
	if err != nil {
		t.Fatalf("format decimal: %v", err)
	}
	if actual != expected {
		t.Fatalf("unexpected decimal: got %s want %s", actual, expected)
	}
}
