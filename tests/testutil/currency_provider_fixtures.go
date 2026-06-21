package testutil

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

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
