package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
)

func TestSyncServiceSuccess(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/activities":
			_, _ = writer.Write([]byte(`{"activities":[{"id":"id","date":"2026-01-31T10:00:00Z","type":"BUY"}],"count":1}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var client = ghostfolioclient.New(server.Client())
	var service = runtime.NewSyncService(client, 5*time.Second)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var outcome = service.Validate(context.Background(), config, "token")
	if !outcome.Success {
		t.Fatalf("expected success outcome: %#v", outcome)
	}
}

func TestSyncServiceFailureCategoryPassThrough(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	var client = ghostfolioclient.New(server.Client())
	var service = runtime.NewSyncService(client, 5*time.Second)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var outcome = service.Validate(context.Background(), config, "token")
	if outcome.FailureCategory != ghostfolioclient.FailureRejectedToken {
		t.Fatalf("expected rejected token category, got %#v", outcome)
	}
}
