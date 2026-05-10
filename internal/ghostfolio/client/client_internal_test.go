package client

import (
	"context"
	"errors"
	"net"
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

	var err = &RequestFailure{Category: FailureTimeout, Message: "timeout", Err: context.DeadlineExceeded}
	if err.Error() != "timeout" {
		t.Fatalf("unexpected error text: %q", err.Error())
	}
	if !errors.Is(err.Unwrap(), context.DeadlineExceeded) {
		t.Fatalf("unexpected unwrap value")
	}

	if got := (&RequestFailure{Category: FailureRejectedToken}).Error(); got != string(FailureRejectedToken) {
		t.Fatalf("unexpected default error text: %q", got)
	}
}

func TestNewUsesDefaultClientWhenNil(t *testing.T) {
	t.Parallel()

	if client := New(nil); client.httpClient == nil {
		t.Fatalf("expected default http client")
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

func TestFetchActivitiesProbeHandlesServerResponseClasses(t *testing.T) {
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

			_, err := New(server.Client()).FetchActivitiesProbe(context.Background(), server.URL, "jwt")
			var failure *RequestFailure
			if !errors.As(err, &failure) || failure.Category != testCase.expected {
				t.Fatalf("unexpected failure: %v", err)
			}
		})
	}
}

func TestFetchActivitiesProbeHandlesTransportAndBuildErrors(t *testing.T) {
	t.Parallel()

	t.Run("transport timeout", func(t *testing.T) {
		var client = New(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, timeoutError{}
		})})
		_, err := client.FetchActivitiesProbe(context.Background(), "https://ghostfol.io", "jwt")
		var failure *RequestFailure
		if !errors.As(err, &failure) || failure.Category != FailureTimeout {
			t.Fatalf("unexpected failure: %v", err)
		}
	})

	t.Run("invalid origin", func(t *testing.T) {
		_, err := New(nil).FetchActivitiesProbe(context.Background(), "://bad", "jwt")
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

		var response, err = New(server.Client()).FetchActivitiesProbe(context.Background(), server.URL, "jwt")
		if err != nil || response.Count != 0 {
			t.Fatalf("expected successful activities fetch, response=%#v err=%v", response, err)
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

	var listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_ = listener.Close()
	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
	}))
	defer server.Close()

	var response *http.Response
	response, err = server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	closeBody(response)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
