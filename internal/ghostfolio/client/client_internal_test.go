package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return false }

func TestRequestFailureHelpers(t *testing.T) {
	t.Parallel()

	var err = &RequestFailure{Category: FailureTimeout, Detail: "ghostfolio request deadline exceeded", Err: context.DeadlineExceeded}
	if err.Error() != "ghostfolio request deadline exceeded" {
		t.Fatalf("unexpected error text: %q", err.Error())
	}
	if !errors.Is(err.Unwrap(), context.DeadlineExceeded) {
		t.Fatalf("unexpected unwrap value")
	}

	if got := (&RequestFailure{Category: FailureRejectedToken, Operation: "anonymous auth", StatusCode: http.StatusForbidden}).Error(); got != "anonymous auth returned HTTP 403" {
		t.Fatalf("unexpected default error text: %q", got)
	}
	if got := (*RequestFailure)(nil).Error(); got != "" {
		t.Fatalf("unexpected nil error text: %q", got)
	}
}

func TestNewUsesDefaultClientWhenNil(t *testing.T) {
	t.Parallel()

	var client = New(nil)
	if client.httpClient == nil {
		t.Fatalf("expected default http client")
	}
	if client.httpClient.Timeout != defaultHTTPClientTimeout {
		t.Fatalf("unexpected default timeout: got %v want %v", client.httpClient.Timeout, defaultHTTPClientTimeout)
	}
}

func TestClassifyTransportFailureTimeout(t *testing.T) {
	t.Parallel()

	var failure *RequestFailure
	if !errors.As(classifyTransportFailure(timeoutError{}), &failure) || failure.Category != FailureTimeout {
		t.Fatalf("expected timeout failure")
	}
}

func TestClassifyTransportFailureConnectivity(t *testing.T) {
	t.Parallel()

	var failure *RequestFailure
	if !errors.As(classifyTransportFailure(errors.New("boom")), &failure) || failure.Category != FailureConnectivityProblem {
		t.Fatalf("expected connectivity failure")
	}
}

func TestAuthenticateHandlesServerResponseClasses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		status      int
		contentType string
		body        string
		expected    FailureCategory
	}{
		{name: "unsuccessful response", status: http.StatusInternalServerError, expected: FailureUnsuccessfulServerResponse},
		{name: "invalid content type", status: http.StatusOK, contentType: "text/plain", body: "ok", expected: FailureIncompatibleServerContract},
		{name: "invalid json", status: http.StatusOK, contentType: "application/json", body: "{", expected: FailureIncompatibleServerContract},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if testCase.contentType != "" {
					writer.Header().Set("Content-Type", testCase.contentType)
				}
				writer.WriteHeader(testCase.status)
				_, _ = writer.Write([]byte(testCase.body))
			}))
			defer server.Close()

			_, err := New(server.Client()).Authenticate(context.Background(), server.URL, "token")
			var failure *RequestFailure
			if !errors.As(err, &failure) || failure.Category != testCase.expected {
				t.Fatalf("unexpected failure: %v", err)
			}
		})
	}
}

func TestAuthenticateHandlesTransportErrors(t *testing.T) {
	t.Parallel()

	var client = New(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, timeoutError{}
	})})
	_, err := client.Authenticate(context.Background(), "https://ghostfol.io", "token")
	var failure *RequestFailure
	if !errors.As(err, &failure) || failure.Category != FailureTimeout {
		t.Fatalf("unexpected failure: %v", err)
	}
}

func TestAuthenticateAcceptsJSONContentTypesAndBuildErrors(t *testing.T) {
	t.Parallel()

	t.Run("json with suffix", func(t *testing.T) {
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/problem+json")
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		}))
		defer server.Close()

		var response, err = New(server.Client()).Authenticate(context.Background(), server.URL, "token")
		if err != nil || response.AuthToken != "jwt" {
			t.Fatalf("expected suffix json content-type to work, response=%#v err=%v", response, err)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		_, err := New(server.Client()).Authenticate(context.Background(), server.URL, "token")
		var failure *RequestFailure
		if !errors.As(err, &failure) || failure.Category != FailureRejectedToken {
			t.Fatalf("expected rejected token failure, got %v", err)
		}
	})

	t.Run("invalid origin", func(t *testing.T) {
		_, err := New(nil).Authenticate(context.Background(), "://bad", "token")
		if err == nil {
			t.Fatalf("expected request build error")
		}
	})
}

func TestFetchSingleActivitiesPageHandlesServerResponseClasses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		status      int
		contentType string
		body        string
		expected    FailureCategory
	}{
		{name: "bad request", status: http.StatusBadRequest, expected: FailureIncompatibleServerContract},
		{name: "forbidden", status: http.StatusForbidden, expected: FailureUnsuccessfulServerResponse},
		{name: "unauthorized", status: http.StatusUnauthorized, expected: FailureUnsuccessfulServerResponse},
		{name: "invalid content type", status: http.StatusOK, contentType: "text/plain", body: "ok", expected: FailureIncompatibleServerContract},
		{name: "invalid json", status: http.StatusOK, contentType: "application/json", body: "{", expected: FailureIncompatibleServerContract},
		{name: "server error", status: http.StatusInternalServerError, expected: FailureUnsuccessfulServerResponse},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if testCase.contentType != "" {
					writer.Header().Set("Content-Type", testCase.contentType)
				}
				writer.WriteHeader(testCase.status)
				_, _ = writer.Write([]byte(testCase.body))
			}))
			defer server.Close()

			_, err := New(server.Client()).FetchActivitiesHistory(context.Background(), server.URL, "jwt")
			var failure *RequestFailure
			if !errors.As(err, &failure) || failure.Category != testCase.expected {
				t.Fatalf("unexpected failure: %v", err)
			}
		})
	}
}

func TestFetchActivitiesHistoryHandlesTransportAndBuildErrors(t *testing.T) {
	t.Parallel()

	t.Run("transport timeout", func(t *testing.T) {
		var client = New(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, timeoutError{}
		})})
		_, err := client.FetchActivitiesHistory(context.Background(), "https://ghostfol.io", "jwt")
		var failure *RequestFailure
		if !errors.As(err, &failure) || failure.Category != FailureTimeout {
			t.Fatalf("unexpected failure: %v", err)
		}
	})

	t.Run("invalid origin", func(t *testing.T) {
		_, err := New(nil).FetchActivitiesHistory(context.Background(), "://bad", "jwt")
		if err == nil {
			t.Fatalf("expected request build error")
		}
	})

	t.Run("success", func(t *testing.T) {
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"activities":[],"count":0}`))
		}))
		defer server.Close()

		var response, err = New(server.Client()).FetchActivitiesHistory(context.Background(), server.URL, "jwt")
		if err != nil || response.Count != 0 {
			t.Fatalf("expected successful activities fetch, response=%#v err=%v", response, err)
		}
	})
}

// TestFetchActivitiesHistoryCoversPaginationBranches verifies paginated history
// retrieval across success and incompatible-pagination branches.
// Authored by: OpenCode
func TestFetchActivitiesHistoryCoversPaginationBranches(t *testing.T) {
	t.Parallel()

	t.Run("multi page success", func(t *testing.T) {
		var requestCount int
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestCount++
			writer.Header().Set("Content-Type", "application/json")
			if request.URL.Query().Get("take") != "250" || request.URL.Query().Get("sortColumn") != "date" || request.URL.Query().Get("sortDirection") != "asc" {
				t.Fatalf("unexpected pagination query: %s", request.URL.RawQuery)
			}
			if request.Header.Get("Authorization") != "Bearer jwt" {
				t.Fatalf("unexpected auth header: %q", request.Header.Get("Authorization"))
			}

			switch request.URL.Query().Get("skip") {
			case "0":
				_, _ = writer.Write([]byte(`{"activities":[{"id":"a"},{"id":"b"}],"count":3}`))
			case "2":
				_, _ = writer.Write([]byte(`{"activities":[{"id":"c"}],"count":3}`))
			default:
				writer.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		var response, err = New(server.Client()).FetchActivitiesHistory(context.Background(), server.URL, "jwt")
		if err != nil {
			t.Fatalf("fetch activities history: %v", err)
		}
		if response.Count != 3 || len(response.Activities) != 3 {
			t.Fatalf("unexpected paginated response: %#v", response)
		}
		if requestCount != 2 {
			t.Fatalf("expected two paginated requests, got %d", requestCount)
		}
	})

	t.Run("zero count success", func(t *testing.T) {
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"activities":[],"count":0}`))
		}))
		defer server.Close()

		var response, err = New(server.Client()).FetchActivitiesHistory(context.Background(), server.URL, "jwt")
		if err != nil {
			t.Fatalf("fetch zero-count history: %v", err)
		}
		if response.Count != 0 || len(response.Activities) != 0 {
			t.Fatalf("unexpected zero-count response: %#v", response)
		}
	})

	t.Run("count changes during retrieval", func(t *testing.T) {
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			switch request.URL.Query().Get("skip") {
			case "0":
				_, _ = writer.Write([]byte(`{"activities":[{"id":"a"}],"count":2}`))
			case "1":
				_, _ = writer.Write([]byte(`{"activities":[{"id":"b"}],"count":3}`))
			default:
				writer.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		_, err := New(server.Client()).FetchActivitiesHistory(context.Background(), server.URL, "jwt")
		var failure *RequestFailure
		if !errors.As(err, &failure) || failure.Category != FailureIncompatibleServerContract {
			t.Fatalf("expected incompatible-contract failure, got %v", err)
		}
	})

	t.Run("zero count with activities fails", func(t *testing.T) {
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"activities":[{"id":"a"}],"count":0}`))
		}))
		defer server.Close()

		_, err := New(server.Client()).FetchActivitiesHistory(context.Background(), server.URL, "jwt")
		var failure *RequestFailure
		if !errors.As(err, &failure) || failure.Category != FailureIncompatibleServerContract {
			t.Fatalf("expected incompatible-contract failure, got %v", err)
		}
	})

	t.Run("pagination ends early", func(t *testing.T) {
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			switch request.URL.Query().Get("skip") {
			case "0":
				_, _ = writer.Write([]byte(`{"activities":[{"id":"a"}],"count":2}`))
			case "1":
				_, _ = writer.Write([]byte(`{"activities":[],"count":2}`))
			default:
				writer.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		_, err := New(server.Client()).FetchActivitiesHistory(context.Background(), server.URL, "jwt")
		var failure *RequestFailure
		if !errors.As(err, &failure) || failure.Category != FailureIncompatibleServerContract {
			t.Fatalf("expected incompatible-contract failure, got %v", err)
		}
	})

	t.Run("pagination exceeds reported count", func(t *testing.T) {
		var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"activities":[{"id":"a"},{"id":"b"}],"count":1}`))
		}))
		defer server.Close()

		_, err := New(server.Client()).FetchActivitiesHistory(context.Background(), server.URL, "jwt")
		var failure *RequestFailure
		if !errors.As(err, &failure) || failure.Category != FailureIncompatibleServerContract {
			t.Fatalf("expected incompatible-contract failure, got %v", err)
		}
	})

	t.Run("invalid origin", func(t *testing.T) {
		_, err := New(nil).FetchActivitiesHistory(context.Background(), "://bad", "jwt")
		if err == nil {
			t.Fatalf("expected invalid origin to fail")
		}
	})
}

func TestClassifyTransportFailureDeadlineExceeded(t *testing.T) {
	t.Parallel()

	var failure *RequestFailure
	if !errors.As(classifyTransportFailure(context.DeadlineExceeded), &failure) || failure.Category != FailureTimeout {
		t.Fatalf("expected deadline timeout failure")
	}
}

func TestRequireJSONContentTypeRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	if err := requireJSONContentType("bad;content;type"); err == nil {
		t.Fatalf("expected content-type parse error")
	}
}

func TestCloseBodyClosesResponseBody(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
	}))
	defer server.Close()

	var response, err = server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	closeBody(response)
}

// roundTripFunc is a test-only http.RoundTripper that delegates response
// behavior to the wrapped function.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper by delegating to the wrapped function.
func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
