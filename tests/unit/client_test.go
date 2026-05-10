package unit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
)

func TestClientAuthenticateSuccess(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/auth/anonymous" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
	}))
	defer server.Close()

	var client = ghostfolioclient.New(server.Client())
	var response, err = client.Authenticate(context.Background(), server.URL, "token")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if response.AuthToken != "jwt" {
		t.Fatalf("unexpected auth token: %q", response.AuthToken)
	}
}

func TestClientAuthenticateRejectedToken(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	var client = ghostfolioclient.New(server.Client())
	_, err := client.Authenticate(context.Background(), server.URL, "token")
	var failure *ghostfolioclient.RequestFailure
	if !errors.As(err, &failure) || failure.Category != ghostfolioclient.FailureRejectedToken {
		t.Fatalf("expected rejected token failure, got %v", err)
	}
}

func TestClientFetchActivitiesProbeContractFailure(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `{"activities":[{"id":"id","date":"2026-01-31T10:00:00Z","type":"BUY"}],"count":0}`)
	}))
	defer server.Close()

	var client = ghostfolioclient.New(server.Client())
	var response, err = client.FetchActivitiesProbe(context.Background(), server.URL, "jwt")
	if err != nil {
		t.Fatalf("fetch probe: %v", err)
	}
	if response.Count != 0 {
		t.Fatalf("unexpected count: %d", response.Count)
	}
}
