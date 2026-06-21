// Package currency verifies Federal Reserve H.10 provider client and mapping behavior.
// Authored by: OpenCode
package currency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestFederalReserveH10ClientBuildsFixedRequest verifies the H.10 client request
// shape and canonical evidence mapping from deterministic CSV.
// Authored by: OpenCode
func TestFederalReserveH10ClientBuildsFixedRequest(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.Contains(request.URL.Path, "/datadownload/") {
			t.Fatalf("unexpected Federal Reserve path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("from") != "2023-12-07" {
			t.Fatalf("unexpected from query: %s", request.URL.RawQuery)
		}
		if request.URL.Query().Get("to") != "2024-01-06" {
			t.Fatalf("unexpected to query: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "text/csv")
		_, _ = writer.Write([]byte("Currency,Monetary unit,Quote direction,2024-01-05,2024-01-06\nMexico,MXN,currency units per USD,16.9140,ND\nEMU member countries,EUR,USD per currency unit,1.0946,ND\n"))
	}))
	defer server.Close()

	var client = NewFederalReserveH10ClientForTesting(server.URL, http.DefaultClient)
	var request = mustRateLookupRequestOnDate(t, "MXN", BaseCurrencyUSD, "2024-01-06")
	var evidence, err = client.LookupRate(context.Background(), request)
	if err != nil {
		t.Fatalf("lookup Federal Reserve rate: %v", err)
	}

	assertFederalReserveEvidence(t, evidence, request, QuoteDirectionSourcePerBase, "2024-01-05", "16.914")
}

// TestFederalReserveH10MapperPreservesQuoteDirection verifies unstarred and
// starred H.10 rows map to distinct canonical quote directions.
// Authored by: OpenCode
func TestFederalReserveH10MapperPreservesQuoteDirection(t *testing.T) {
	t.Parallel()

	var payload = []byte("Currency,Monetary unit,Quote direction,2024-01-05,2024-01-06\nMexico,MXN,currency units per USD,16.9140,ND\nEMU member countries,EUR,USD per currency unit,1.0946,ND\n")
	var mxnRequest = mustRateLookupRequestOnDate(t, "MXN", BaseCurrencyUSD, "2024-01-06")
	var mxnEvidence, mxnErr = MapFederalReserveH10CSVToEvidence(mxnRequest, payload, "H10 fixture")
	if mxnErr != nil {
		t.Fatalf("map MXN H.10 evidence: %v", mxnErr)
	}
	assertFederalReserveEvidence(t, mxnEvidence, mxnRequest, QuoteDirectionSourcePerBase, "2024-01-05", "16.914")

	var eurRequest = mustRateLookupRequestOnDate(t, "EUR", BaseCurrencyUSD, "2024-01-06")
	var eurEvidence, eurErr = MapFederalReserveH10CSVToEvidence(eurRequest, payload, "H10 fixture")
	if eurErr != nil {
		t.Fatalf("map EUR H.10 evidence: %v", eurErr)
	}
	assertFederalReserveEvidence(t, eurEvidence, eurRequest, QuoteDirectionBasePerSource, "2024-01-05", "1.0946")
}

// TestFederalReserveH10MapperRejectsUnsupportedAndMalformedObservations verifies
// failure behavior for unsupported, ND, missing, and ambiguous H.10 data.
// Authored by: OpenCode
func TestFederalReserveH10MapperRejectsUnsupportedAndMalformedObservations(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name           string
		sourceCurrency string
		payload        string
		want           string
	}{
		{name: "unsupported Venezuela row", sourceCurrency: "VES", payload: "Currency,Monetary unit,Quote direction,2024-01-05\nVenezuela,VES,currency units per USD,35.00\n", want: "unsupported source currency VES"},
		{name: "ND only", sourceCurrency: "MXN", payload: "Currency,Monetary unit,Quote direction,2024-01-05\nMexico,MXN,currency units per USD,ND\n", want: "no current or prior available observation"},
		{name: "malformed decimal", sourceCurrency: "MXN", payload: "Currency,Monetary unit,Quote direction,2024-01-05\nMexico,MXN,currency units per USD,not-a-decimal\n", want: "invalid Federal Reserve observation"},
		{name: "ambiguous direction", sourceCurrency: "MXN", payload: "Currency,Monetary unit,Quote direction,2024-01-05\nMexico,MXN,market rate,16.9140\n", want: "ambiguous quote direction"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var request = mustRateLookupRequestOnDate(t, testCase.sourceCurrency, BaseCurrencyUSD, "2024-01-06")
			var _, err = MapFederalReserveH10CSVToEvidence(request, []byte(testCase.payload), "H10 fixture")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

// TestFederalReserveH10ClientDefensiveBranches verifies provider-interface
// wrappers and request-building failures outside the successful fixture path.
// Authored by: OpenCode
func TestFederalReserveH10ClientDefensiveBranches(t *testing.T) {
	t.Parallel()

	var client = NewFederalReserveH10ClientForTesting("%", http.DefaultClient)
	if client.baseCurrency() != BaseCurrencyUSD {
		t.Fatalf("expected Federal Reserve provider to advertise USD base currency")
	}
	var request = mustRateLookupRequestOnDate(t, "EUR", BaseCurrencyUSD, "2024-01-06")
	if _, err := client.federalReserveURL(request); err == nil || !strings.Contains(err.Error(), "build Federal Reserve H.10 URL") {
		t.Fatalf("expected malformed Federal Reserve URL failure, got %v", err)
	}
	if _, err := client.LookupRate(context.Background(), request); err == nil || !strings.Contains(err.Error(), "build Federal Reserve H.10 URL") {
		t.Fatalf("expected malformed Federal Reserve lookup URL failure, got %v", err)
	}
	if _, err := client.LookupRate(context.Background(), mustRateLookupRequestOnDate(t, "VES", BaseCurrencyUSD, "2024-01-06")); err == nil || !strings.Contains(err.Error(), "unsupported source currency VES") {
		t.Fatalf("expected unsupported Federal Reserve source rejection, got %v", err)
	}

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	client = NewFederalReserveH10ClientForTesting(server.URL, http.DefaultClient)
	if _, err := client.LookupRate(context.Background(), request); err == nil || !strings.Contains(err.Error(), "provider returned HTTP status") {
		t.Fatalf("expected Federal Reserve provider fetch failure, got %v", err)
	}

	var malformedServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("Currency,Monetary unit,Quote direction,2024-01-05\n"))
	}))
	defer malformedServer.Close()
	client = NewFederalReserveH10ClientForTesting(malformedServer.URL, http.DefaultClient)
	if _, err := client.LookupRate(context.Background(), request); err == nil || !strings.Contains(err.Error(), "unsupported source currency EUR") {
		t.Fatalf("expected Federal Reserve mapper failure through client, got %v", err)
	}
}

// TestFederalReserveH10MapperRejectsCSVShapeAndDateFallbackGaps verifies
// malformed CSV envelopes and unavailable observations fail without fallback rates.
// Authored by: OpenCode
func TestFederalReserveH10MapperRejectsCSVShapeAndDateFallbackGaps(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequestOnDate(t, "MXN", BaseCurrencyUSD, "2024-01-06")
	var testCases = []struct {
		name    string
		payload string
		want    string
	}{
		{name: "empty payload", payload: "", want: "no current or prior available observation"},
		{name: "malformed quoted CSV", payload: "Currency,Monetary unit,Quote direction,2024-01-05\n\"Mexico,MXN,currency units per USD,16.9140\n", want: "parse Federal Reserve H.10 CSV"},
		{name: "missing source row", payload: "Currency,Monetary unit,Quote direction,2024-01-05\nEMU member countries,EUR,USD per currency unit,1.0946\n", want: "unsupported source currency MXN"},
		{name: "too few columns", payload: "Currency,Monetary unit,Quote direction\nMexico,MXN,currency units per USD\n", want: "date observations are required"},
		{name: "future observation only", payload: "Currency,Monetary unit,Quote direction,2024-01-07\nMexico,MXN,currency units per USD,16.9140\n", want: "no current or prior available observation"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var _, err = MapFederalReserveH10CSVToEvidence(request, []byte(testCase.payload), "H10 fixture")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

// assertFederalReserveEvidence verifies canonical H.10 evidence.
// Authored by: OpenCode
func assertFederalReserveEvidence(t *testing.T, evidence ExchangeRateEvidence, request RateLookupRequest, quoteDirection QuoteDirection, rateDate string, rateValue string) {
	t.Helper()

	if !evidence.matchesRequest(request) {
		t.Fatalf("evidence does not match request: %#v %#v", evidence, request)
	}
	if evidence.ProviderID != ProviderIDFederalReserveH10 || evidence.Authority != RateAuthorityFederalReserve {
		t.Fatalf("unexpected Federal Reserve provider identity: %#v", evidence)
	}
	if evidence.QuoteDirection != quoteDirection {
		t.Fatalf("unexpected Federal Reserve quote direction: got %s want %s", evidence.QuoteDirection, quoteDirection)
	}
	if evidence.RateDate != mustDateOnly(t, rateDate) {
		t.Fatalf("unexpected Federal Reserve rate date: got %s want %s", evidence.RateDate.Format(time.DateOnly), rateDate)
	}
	assertCurrencyDecimalString(t, evidence.RateValue, rateValue)
}
