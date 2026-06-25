package testutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
)

const (
	officialECBEXRHost            = "data-api.ecb.europa.eu"
	officialFederalReserveH10Host = "www.federalreserve.gov"
)

var officialCurrencyRateServiceFixtureTransportMu sync.Mutex

// CurrencyProviderFixtureBuilder stores deterministic HTTP responses for a
// local exchange-rate provider test double.
// Authored by: OpenCode
type CurrencyProviderFixtureBuilder struct {
	responses []CurrencyProviderFixtureResponse
}

// CurrencyProviderFixtureResponse describes one deterministic provider response.
//
// Body is stored as a string so future rate fixtures can preserve exact decimal
// text without introducing floating-point values.
// Authored by: OpenCode
type CurrencyProviderFixtureResponse struct {
	Method     string
	Path       string
	StatusCode int
	Headers    map[string]string
	Body       string
}

// CurrencyProviderFixtureRequest records one provider request without retaining
// secret header values.
// Authored by: OpenCode
type CurrencyProviderFixtureRequest struct {
	Method                 string
	Path                   string
	RawQuery               string
	HasAuthorizationHeader bool
}

// CurrencyProviderFixture wraps one httptest server and the requests observed by
// that server.
// Authored by: OpenCode
type CurrencyProviderFixture struct {
	server    *httptest.Server
	responses []CurrencyProviderFixtureResponse
	mu        sync.Mutex
	requests  []CurrencyProviderFixtureRequest
}

// OfficialCurrencyRateServiceFixtureEndpoints configures local provider servers
// for contract tests that exercise the public currency rate service.
// Authored by: OpenCode
type OfficialCurrencyRateServiceFixtureEndpoints struct {
	ECBEXRBaseURL            string
	FederalReserveH10BaseURL string
}

// officialCurrencyRateProviderFixtureTransport routes production official-rate
// provider requests to deterministic local servers for one contract test.
// Authored by: OpenCode
type officialCurrencyRateProviderFixtureTransport struct {
	base                  http.RoundTripper
	ecbEXRBaseURL         *url.URL
	federalReserveBaseURL *url.URL
}

// NewCurrencyProviderFixtureBuilder creates an empty deterministic provider
// fixture builder.
//
// Example usage:
//
//	builder := testutil.NewCurrencyProviderFixtureBuilder().WithResponse(testutil.CurrencyProviderFixtureResponse{
//		Path: "/rates",
//		Body: `date,rate\n2024-01-02,1.2345\n`,
//	})
//
// Authored by: OpenCode
func NewCurrencyProviderFixtureBuilder() CurrencyProviderFixtureBuilder {
	return CurrencyProviderFixtureBuilder{}
}

// WithResponse appends one deterministic provider response to the builder.
//
// Empty Method defaults to GET, empty Path defaults to /, and zero StatusCode
// defaults to 200.
//
// Example usage:
//
//	builder := testutil.NewCurrencyProviderFixtureBuilder().WithResponse(testutil.CurrencyProviderFixtureResponse{
//		Method: "GET",
//		Path:   "/service/data/EXR/D.USD.EUR.SP00.A",
//		Body:   "fixture body",
//	})
//
// Authored by: OpenCode
func (builder CurrencyProviderFixtureBuilder) WithResponse(response CurrencyProviderFixtureResponse) CurrencyProviderFixtureBuilder {
	builder.responses = append(builder.responses, normalizeCurrencyProviderFixtureResponse(response))
	return builder
}

// NewCurrencyProviderFixture starts one local deterministic provider fixture for
// the current test and registers server cleanup with testing.T.
//
// Example usage:
//
//	fixture := testutil.NewCurrencyProviderFixture(t, builder)
//	baseURL := fixture.URL()
//
// Authored by: OpenCode
func NewCurrencyProviderFixture(t *testing.T, builder CurrencyProviderFixtureBuilder) *CurrencyProviderFixture {
	t.Helper()

	var fixture = &CurrencyProviderFixture{
		responses: builder.responses,
	}
	fixture.server = httptest.NewServer(fixture.handler())
	t.Cleanup(fixture.server.Close)

	return fixture
}

// NewOfficialCurrencyRateServiceFixture creates the public currency rate service
// with its official provider HTTP requests routed to local fixture servers.
//
// The production service keeps fixed official provider origins. This helper does
// not expose fixture endpoints through production code. Instead, it temporarily
// replaces http.DefaultClient.Transport for the current test process and restores
// it during test cleanup. Tests using this helper must not call t.Parallel.
//
// Example usage:
//
//	service := testutil.NewOfficialCurrencyRateServiceFixture(t, testutil.OfficialCurrencyRateServiceFixtureEndpoints{
//		ECBEXRBaseURL: "http://127.0.0.1:1234",
//		FederalReserveH10BaseURL: "http://127.0.0.1:5678",
//	})
//	_ = service.SupportedBaseCurrencies()
//
// Authored by: OpenCode
func NewOfficialCurrencyRateServiceFixture(t *testing.T, endpoints OfficialCurrencyRateServiceFixtureEndpoints) currency.CurrencyRateService {
	t.Helper()

	var ecbEXRBaseURL = mustParseFixtureBaseURL(t, endpoints.ECBEXRBaseURL, "ECB EXR base URL")
	var federalReserveBaseURL = mustParseFixtureBaseURL(t, endpoints.FederalReserveH10BaseURL, "Federal Reserve H.10 base URL")

	officialCurrencyRateServiceFixtureTransportMu.Lock()
	var previousTransport = http.DefaultClient.Transport
	var baseTransport http.RoundTripper = previousTransport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	http.DefaultClient.Transport = officialCurrencyRateProviderFixtureTransport{
		base:                  baseTransport,
		ecbEXRBaseURL:         ecbEXRBaseURL,
		federalReserveBaseURL: federalReserveBaseURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = previousTransport
		officialCurrencyRateServiceFixtureTransportMu.Unlock()
	})

	return currency.NewCurrencyRateService(currency.NewCurrencyRateSessionCache())
}

// RoundTrip rewrites official provider hosts to the configured local fixture hosts.
// Authored by: OpenCode
func (transport officialCurrencyRateProviderFixtureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	var target = transport.targetForRequest(request)
	if target == nil {
		return transport.base.RoundTrip(request)
	}

	var routedRequest = request.Clone(request.Context())
	var routedURL = *request.URL
	routedURL.Scheme = target.Scheme
	routedURL.Host = target.Host
	routedURL.Path = strings.TrimRight(target.Path, "/") + request.URL.Path
	routedRequest.URL = &routedURL
	routedRequest.Host = ""

	return transport.base.RoundTrip(routedRequest)
}

// targetForRequest returns the local provider URL for one official request host.
// Authored by: OpenCode
func (transport officialCurrencyRateProviderFixtureTransport) targetForRequest(request *http.Request) *url.URL {
	if request == nil || request.URL == nil {
		return nil
	}
	switch request.URL.Host {
	case officialECBEXRHost:
		return transport.ecbEXRBaseURL
	case officialFederalReserveH10Host:
		return transport.federalReserveBaseURL
	default:
		return nil
	}
}

// URL returns the local provider server base URL for clients under test.
//
// Example usage:
//
//	fixture := testutil.NewCurrencyProviderFixture(t, builder)
//	_ = fixture.URL()
//
// Authored by: OpenCode
func (fixture *CurrencyProviderFixture) URL() string {
	return fixture.server.URL
}

// Requests returns a copy of the provider requests observed by the fixture.
//
// Example usage:
//
//	requests := fixture.Requests()
//	if len(requests) != 1 {
//		t.Fatalf("unexpected request count: %d", len(requests))
//	}
//
// Authored by: OpenCode
func (fixture *CurrencyProviderFixture) Requests() []CurrencyProviderFixtureRequest {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()

	var requests = make([]CurrencyProviderFixtureRequest, len(fixture.requests))
	copy(requests, fixture.requests)
	return requests
}

// mustParseFixtureBaseURL parses and validates one local fixture base URL.
// Authored by: OpenCode
func mustParseFixtureBaseURL(t *testing.T, rawURL string, label string) *url.URL {
	t.Helper()

	var parsed, err = url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		t.Fatalf("parse %s: %v", label, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		t.Fatalf("%s must include scheme and host", label)
	}

	return parsed
}

// handler returns the HTTP handler used by the local provider server.
// Authored by: OpenCode
func (fixture *CurrencyProviderFixture) handler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fixture.recordRequest(request)

		for _, response := range fixture.responses {
			if response.Method == request.Method && response.Path == request.URL.Path {
				writeCurrencyProviderFixtureResponse(writer, response)
				return
			}
		}

		http.NotFound(writer, request)
	})
}

// recordRequest stores request metadata without retaining secret header values.
// Authored by: OpenCode
func (fixture *CurrencyProviderFixture) recordRequest(request *http.Request) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()

	fixture.requests = append(fixture.requests, CurrencyProviderFixtureRequest{
		Method:                 request.Method,
		Path:                   request.URL.Path,
		RawQuery:               request.URL.RawQuery,
		HasAuthorizationHeader: request.Header.Get("Authorization") != "",
	})
}

// normalizeCurrencyProviderFixtureResponse applies deterministic defaults to one
// response definition.
// Authored by: OpenCode
func normalizeCurrencyProviderFixtureResponse(response CurrencyProviderFixtureResponse) CurrencyProviderFixtureResponse {
	if response.Method == "" {
		response.Method = http.MethodGet
	}
	if response.Path == "" {
		response.Path = "/"
	}
	if response.StatusCode == 0 {
		response.StatusCode = http.StatusOK
	}
	response.Headers = cloneCurrencyProviderFixtureHeaders(response.Headers)
	return response
}

// cloneCurrencyProviderFixtureHeaders returns one independent header map for the
// fixture response.
// Authored by: OpenCode
func cloneCurrencyProviderFixtureHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	var cloned = make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

// writeCurrencyProviderFixtureResponse writes one deterministic response to the
// local provider request.
// Authored by: OpenCode
func writeCurrencyProviderFixtureResponse(writer http.ResponseWriter, response CurrencyProviderFixtureResponse) {
	for key, value := range response.Headers {
		writer.Header().Set(key, value)
	}
	writer.WriteHeader(response.StatusCode)
	_, _ = writer.Write([]byte(response.Body))
}
