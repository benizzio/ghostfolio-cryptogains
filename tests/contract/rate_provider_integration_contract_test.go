// Package contract verifies externally visible workflow and storage contracts.
// Authored by: OpenCode
package contract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
)

// TestOfficialRateProviderContractResolvesDeterministicFixtures verifies the
// default contract path for supported official-provider source currencies.
// Authored by: OpenCode
func TestOfficialRateProviderContractResolvesDeterministicFixtures(t *testing.T) {
	var ecbServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.Contains(request.URL.Path, "/service/data/EXR/D.USD.EUR.SP00.A") {
			t.Fatalf("unexpected ECB path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("endPeriod") != "2024-01-06" {
			t.Fatalf("unexpected ECB endPeriod: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "text/csv")
		_, _ = writer.Write([]byte("TIME_PERIOD,OBS_VALUE\n2024-01-05,1.0921\n"))
	}))
	defer ecbServer.Close()

	var fedServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/datadownload/Output.aspx" {
			t.Fatalf("unexpected Federal Reserve path: %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "text/csv")
		_, _ = writer.Write([]byte("\"Descriptions:\",\"Unit:\",\"Multiplier:\",\"Currency:\",\"Unique Identifier:\",\"Series Name:\",2024-01-05,2024-01-06\n\"Euro-Area Euro\",\"Currency\",\"1\",\"EUR\",H10/H10/RXI$US_N.B.EU,RXI$US_N.B.EU,1.0957,ND\n\"Mexican Peso\",\"Currency\",\"1\",\"MXN\",H10/H10/RXI_N.B.MX,RXI_N.B.MX,16.9141,ND\n"))
	}))
	defer fedServer.Close()

	var service = testutil.NewOfficialCurrencyRateServiceFixture(t, testutil.OfficialCurrencyRateServiceFixtureEndpoints{
		ECBEXRBaseURL:            ecbServer.URL,
		FederalReserveH10BaseURL: fedServer.URL,
	})

	var ecbRequest = mustContractRateLookupRequest(t, "USD", currency.BaseCurrencyEUR, "2024-01-06")
	var ecbEvidence, ecbErr = service.LookupRate(context.Background(), ecbRequest)
	if ecbErr != nil {
		t.Fatalf("lookup ECB fixture evidence: %v", ecbErr)
	}
	assertContractEvidence(t, ecbEvidence, currency.ProviderIDECBEXR, currency.RateAuthorityEuropeanCentralBank, currency.QuoteDirectionSourcePerBase, "2024-01-05", "1.0921")

	var fedRequest = mustContractRateLookupRequest(t, "EUR", currency.BaseCurrencyUSD, "2024-01-06")
	var fedEvidence, fedErr = service.LookupRate(context.Background(), fedRequest)
	if fedErr != nil {
		t.Fatalf("lookup Federal Reserve fixture evidence: %v", fedErr)
	}
	assertContractEvidence(t, fedEvidence, currency.ProviderIDFederalReserveH10, currency.RateAuthorityFederalReserve, currency.QuoteDirectionBasePerSource, "2024-01-05", "1.0957")
}

// TestOfficialRateProviderContractRejectsUnsupportedSourceCurrencies verifies
// the contract failure path for suspended, absent, or unmapped currencies.
// Authored by: OpenCode
func TestOfficialRateProviderContractRejectsUnsupportedSourceCurrencies(t *testing.T) {
	var providerCalls int
	var fixtureServer = httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		providerCalls++
	}))
	defer fixtureServer.Close()

	var service = testutil.NewOfficialCurrencyRateServiceFixture(t, testutil.OfficialCurrencyRateServiceFixtureEndpoints{
		ECBEXRBaseURL:            fixtureServer.URL,
		FederalReserveH10BaseURL: fixtureServer.URL,
	})

	var testCases = []struct {
		name           string
		sourceCurrency string
		baseCurrency   string
		want           string
	}{
		{name: "ECB suspended RUB", sourceCurrency: "RUB", baseCurrency: currency.BaseCurrencyEUR, want: "unsupported_currency"},
		{name: "Federal Reserve unmapped VES", sourceCurrency: "VES", baseCurrency: currency.BaseCurrencyUSD, want: "unsupported_currency"},
		{name: "malformed currency", sourceCurrency: "X", baseCurrency: currency.BaseCurrencyEUR, want: "three-letter uppercase currency code"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var request, err = currency.NewRateLookupRequest(testCase.sourceCurrency, testCase.baseCurrency, mustContractDate(t, "2024-01-06"))
			if err == nil {
				_, err = service.LookupRate(context.Background(), request)
			}
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}

	if providerCalls != 0 {
		t.Fatalf("unsupported currencies must fail before provider calls, got %d calls", providerCalls)
	}
}

// mustContractRateLookupRequest creates one contract lookup request.
// Authored by: OpenCode
func mustContractRateLookupRequest(t *testing.T, sourceCurrency string, baseCurrency string, rawDate string) currency.RateLookupRequest {
	t.Helper()

	var request, err = currency.NewRateLookupRequest(sourceCurrency, baseCurrency, mustContractDate(t, rawDate))
	if err != nil {
		t.Fatalf("create rate lookup request: %v", err)
	}

	return request
}

// mustContractDate parses a source-calendar date for contract tests.
// Authored by: OpenCode
func mustContractDate(t *testing.T, rawDate string) time.Time {
	t.Helper()

	var value, err = time.Parse(time.DateOnly, rawDate)
	if err != nil {
		t.Fatalf("parse date %q: %v", rawDate, err)
	}

	return value
}

// assertContractEvidence verifies canonical provider evidence fields.
// Authored by: OpenCode
func assertContractEvidence(t *testing.T, evidence currency.ExchangeRateEvidence, providerID currency.ProviderID, authority currency.RateAuthority, quoteDirection currency.QuoteDirection, rateDate string, rateValue string) {
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
	assertContractDecimalString(t, evidence.RateValue, rateValue)
}

// assertContractDecimalString verifies one exact decimal rendering.
// Authored by: OpenCode
func assertContractDecimalString(t *testing.T, value apd.Decimal, expected string) {
	t.Helper()

	var actual, err = decimalsupport.CanonicalString(value)
	if err != nil {
		t.Fatalf("format decimal: %v", err)
	}
	if actual != expected {
		t.Fatalf("unexpected decimal: got %s want %s", actual, expected)
	}
}
