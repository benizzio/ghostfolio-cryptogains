// Package currency verifies ECB EXR provider client and mapping behavior.
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

// TestECBEXRClientBuildsFixedSeriesRequest verifies the ECB client request shape
// and canonical same-day evidence mapping from deterministic CSV.
// Authored by: OpenCode
func TestECBEXRClientBuildsFixedSeriesRequest(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/service/data/EXR/D.USD.EUR.SP00.A" {
			t.Fatalf("unexpected ECB path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("startPeriod") != "2023-12-07" {
			t.Fatalf("unexpected startPeriod: %s", request.URL.RawQuery)
		}
		if request.URL.Query().Get("endPeriod") != "2024-01-06" {
			t.Fatalf("unexpected endPeriod: %s", request.URL.RawQuery)
		}
		if request.URL.Query().Get("detail") != "dataonly" {
			t.Fatalf("unexpected detail query: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "text/csv")
		_, _ = writer.Write([]byte("TIME_PERIOD,OBS_VALUE\n2024-01-06,1.0921\n"))
	}))
	defer server.Close()

	var client = NewECBEXRClientForTesting(server.URL, http.DefaultClient)
	var request = mustRateLookupRequestOnDate(t, "USD", BaseCurrencyEUR, "2024-01-06")
	var evidence, err = client.LookupRate(context.Background(), request)
	if err != nil {
		t.Fatalf("lookup ECB rate: %v", err)
	}

	assertECBEvidence(t, evidence, request, "2024-01-06", "1.0921")
}

// TestECBEXRMapperSelectsPreviousAvailableObservation verifies TARGET closing
// day and weekend fallback to the latest prior observation.
// Authored by: OpenCode
func TestECBEXRMapperSelectsPreviousAvailableObservation(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequestOnDate(t, "GBP", BaseCurrencyEUR, "2024-01-06")
	var evidence, err = MapECBEXRCSVToEvidence(request, []byte("TIME_PERIOD,OBS_VALUE\n2024-01-04,0.86120\n2024-01-05,0.86010\n"), "EXR/D.GBP.EUR.SP00.A")
	if err != nil {
		t.Fatalf("map ECB evidence: %v", err)
	}

	assertECBEvidence(t, evidence, request, "2024-01-05", "0.8601")
}

// TestECBEXRMapperRejectsUnsupportedAndMalformedObservations verifies ECB
// failure classifications for unsupported or non-defensible provider data.
// Authored by: OpenCode
func TestECBEXRMapperRejectsUnsupportedAndMalformedObservations(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name           string
		sourceCurrency string
		payload        string
		want           string
	}{
		{name: "suspended RUB", sourceCurrency: "RUB", payload: "TIME_PERIOD,OBS_VALUE\n2024-01-05,99.99\n", want: "unsupported source currency RUB"},
		{name: "missing observation", sourceCurrency: "USD", payload: "TIME_PERIOD,OBS_VALUE\n", want: "no current or prior available observation"},
		{name: "ND observation", sourceCurrency: "USD", payload: "TIME_PERIOD,OBS_VALUE\n2024-01-05,ND\n", want: "invalid ECB observation"},
		{name: "malformed decimal", sourceCurrency: "USD", payload: "TIME_PERIOD,OBS_VALUE\n2024-01-05,not-a-decimal\n", want: "invalid ECB observation"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var request = mustRateLookupRequestOnDate(t, testCase.sourceCurrency, BaseCurrencyEUR, "2024-01-06")
			var _, err = MapECBEXRCSVToEvidence(request, []byte(testCase.payload), "EXR/D."+testCase.sourceCurrency+".EUR.SP00.A")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

// assertECBEvidence verifies canonical ECB evidence.
// Authored by: OpenCode
func assertECBEvidence(t *testing.T, evidence ExchangeRateEvidence, request RateLookupRequest, rateDate string, rateValue string) {
	t.Helper()

	if !evidence.matchesRequest(request) {
		t.Fatalf("evidence does not match request: %#v %#v", evidence, request)
	}
	if evidence.ProviderID != ProviderIDECBEXR || evidence.Authority != RateAuthorityEuropeanCentralBank {
		t.Fatalf("unexpected ECB provider identity: %#v", evidence)
	}
	if evidence.QuoteDirection != QuoteDirectionSourcePerBase {
		t.Fatalf("unexpected ECB quote direction: %s", evidence.QuoteDirection)
	}
	if evidence.RateDate != mustDateOnly(t, rateDate) {
		t.Fatalf("unexpected ECB rate date: got %s want %s", evidence.RateDate.Format(time.DateOnly), rateDate)
	}
	assertCurrencyDecimalString(t, evidence.RateValue, rateValue)
}

// mustRateLookupRequestOnDate creates a lookup request for a fixed date.
// Authored by: OpenCode
func mustRateLookupRequestOnDate(t *testing.T, sourceCurrency string, baseCurrency string, rawDate string) RateLookupRequest {
	t.Helper()

	var request, err = NewRateLookupRequest(sourceCurrency, baseCurrency, mustDateOnly(t, rawDate))
	if err != nil {
		t.Fatalf("create rate lookup request: %v", err)
	}

	return request
}

// mustDateOnly parses a YYYY-MM-DD date in UTC.
// Authored by: OpenCode
func mustDateOnly(t *testing.T, rawDate string) time.Time {
	t.Helper()

	var value, err = time.Parse(time.DateOnly, rawDate)
	if err != nil {
		t.Fatalf("parse date %q: %v", rawDate, err)
	}

	return value
}
