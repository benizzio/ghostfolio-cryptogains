package runtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
)

func TestNewUsesExplicitConfigDirectory(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var app, err = New(bootstrap.Options{ConfigDir: tempDir, RequestTimeout: time.Second})
	if err != nil {
		t.Fatalf("new runtime app: %v", err)
	}
	if app.ConfigStore.Path() == "" {
		t.Fatalf("expected config store path")
	}
}

func TestNewUsesUserConfigDirWhenUnset(t *testing.T) {
	var tempDir = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	var app, err = New(bootstrap.Options{RequestTimeout: time.Second})
	if err != nil {
		t.Fatalf("new runtime app: %v", err)
	}
	if app.ConfigStore.Path() == "" {
		t.Fatalf("expected config store path")
	}
}

func TestNewFailsWithoutResolvableUserConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	_, err := New(bootstrap.Options{RequestTimeout: time.Second})
	if err == nil {
		t.Fatalf("expected config directory resolution error")
	}
}

func TestValidateCoversInvalidAuthPayload(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"authToken":""}`))
	}))
	defer server.Close()

	var service = NewSyncService(ghostfolioclient.New(server.Client()), time.Second)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var outcome = service.Validate(context.Background(), config, "token")
	if outcome.FailureCategory != ghostfolioclient.FailureIncompatibleServerContract {
		t.Fatalf("expected incompatible-server outcome, got %#v", outcome)
	}
}

func TestFinalizeFailureFallsBackToConnectivityCategory(t *testing.T) {
	t.Parallel()

	var outcome = finalizeFailure(&GhostfolioSession{}, &SyncValidationAttempt{}, errors.New("boom"))
	if outcome.FailureCategory != ghostfolioclient.FailureConnectivityProblem {
		t.Fatalf("expected connectivity fallback, got %#v", outcome)
	}
}

func TestValidateHandlesActivitiesTransportFailure(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/activities":
			writer.WriteHeader(http.StatusUnauthorized)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var service = NewSyncService(ghostfolioclient.New(server.Client()), time.Second)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var outcome = service.Validate(context.Background(), config, "token")
	if outcome.FailureCategory != ghostfolioclient.FailureUnsuccessfulServerResponse {
		t.Fatalf("expected unsuccessful-response outcome, got %#v", outcome)
	}
}

func TestValidateHandlesActivitiesPayloadValidationFailure(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/activities":
			_, _ = writer.Write([]byte(`{"activities":[{"id":"","date":"2026-01-31T10:00:00Z","type":"BUY"}],"count":1}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var service = NewSyncService(ghostfolioclient.New(server.Client()), time.Second)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var outcome = service.Validate(context.Background(), config, "token")
	if outcome.FailureCategory != ghostfolioclient.FailureIncompatibleServerContract {
		t.Fatalf("expected incompatible-contract outcome, got %#v", outcome)
	}
}

func TestValidateSuccessOutcomeIncludesAttemptAndMessages(t *testing.T) {
	t.Parallel()

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/activities":
			_, _ = writer.Write([]byte(`{"activities":[],"count":0}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var service = NewSyncService(ghostfolioclient.New(server.Client()), time.Second)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var outcome = service.Validate(context.Background(), config, "token")
	if !outcome.Success || outcome.Attempt.Status != AttemptStatusSuccess || outcome.Attempt.AttemptID == "" || outcome.DetailReason != "communication_ok" || outcome.FollowUpNote == "" {
		t.Fatalf("unexpected success outcome: %#v", outcome)
	}
}

func TestValidateHandlesAuthTransportFailure(t *testing.T) {
	t.Parallel()

	var client = ghostfolioclient.New(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})})
	var service = NewSyncService(client, time.Second)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var outcome = service.Validate(context.Background(), config, "token")
	if outcome.FailureCategory != ghostfolioclient.FailureTimeout || outcome.Attempt.Status != AttemptStatusFailure {
		t.Fatalf("unexpected auth failure outcome: %#v", outcome)
	}
}

func TestFinalizeFailureUsesCategorizedRequestFailure(t *testing.T) {
	t.Parallel()

	var outcome = finalizeFailure(
		&GhostfolioSession{SecurityToken: "token", AuthToken: "jwt"},
		&SyncValidationAttempt{},
		&ghostfolioclient.RequestFailure{Category: ghostfolioclient.FailureTimeout, Message: "timeout"},
	)
	if outcome.FailureCategory != ghostfolioclient.FailureTimeout {
		t.Fatalf("expected timeout category, got %#v", outcome)
	}
}

func TestClearSessionSecrets(t *testing.T) {
	t.Parallel()

	var session = GhostfolioSession{SecurityToken: "token", AuthToken: "jwt"}
	clearSessionSecrets(&session)
	if session.SecurityToken != "" || session.AuthToken != "" {
		t.Fatalf("expected secrets to be cleared: %#v", session)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
