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
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
)

type failingStore struct {
	saveErr error
}

func (f failingStore) Load(context.Context) (configmodel.AppSetupConfig, error) {
	return configmodel.AppSetupConfig{}, configstore.ErrNotFound
}

func (f failingStore) Save(context.Context, configmodel.AppSetupConfig) error {
	return f.saveErr
}

func (failingStore) Delete(context.Context) error { return nil }

func (failingStore) Path() string { return "" }

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
	if app.SetupService == nil {
		t.Fatalf("expected setup service")
	}
}

func TestSetupServiceSaveReturnsValidationError(t *testing.T) {
	t.Parallel()

	var service = NewSetupService(failingStore{}, false)
	_, err := service.Save(context.Background(), SaveSetupRequest{
		ServerMode:   configmodel.ServerModeCustomOrigin,
		ServerOrigin: "http://localhost:8080",
		SavedAt:      time.Now(),
	})
	if !errors.Is(err, configmodel.ErrDisallowedTransport) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestSetupServiceSaveReturnsStoreError(t *testing.T) {
	t.Parallel()

	var expected = errors.New("save boom")
	var service = NewSetupService(failingStore{saveErr: expected}, false)
	_, err := service.Save(context.Background(), SaveSetupRequest{
		ServerMode:   configmodel.ServerModeGhostfolioCloud,
		ServerOrigin: configmodel.GhostfolioCloudOrigin,
		SavedAt:      time.Now(),
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped store error, got %v", err)
	}
}

func TestSetupServiceSaveReturnsPersistedConfig(t *testing.T) {
	t.Parallel()

	var service = NewSetupService(failingStore{}, true)
	var result, err = service.Save(context.Background(), SaveSetupRequest{
		ServerMode:   configmodel.ServerModeCustomOrigin,
		ServerOrigin: "http://localhost:8080",
		SavedAt:      time.Now(),
	})
	if err != nil {
		t.Fatalf("expected successful save result, got %v", err)
	}
	if result.Config.ServerOrigin != "http://localhost:8080" || result.Config.ServerMode != configmodel.ServerModeCustomOrigin {
		t.Fatalf("unexpected persisted config: %#v", result.Config)
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

	var outcome = service.Validate(context.Background(), ValidateRequest{Config: config, SecurityToken: "token"})
	if outcome.FailureReason != ValidationFailureIncompatibleServerContract {
		t.Fatalf("expected incompatible-server outcome, got %#v", outcome)
	}
}

func TestFinalizeFailureFallsBackToConnectivityCategory(t *testing.T) {
	t.Parallel()

	var outcome = finalizeFailure(&GhostfolioSession{}, &SyncValidationAttempt{}, errors.New("boom"))
	if outcome.FailureReason != ValidationFailureConnectivityProblem {
		t.Fatalf("expected connectivity fallback, got %#v", outcome)
	}
}

func TestValidationFailureReasonFromBoundaryCoversAllCategories(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name     string
		category ghostfolioclient.FailureCategory
		want     ValidationFailureReason
	}{
		{name: "rejected token", category: ghostfolioclient.FailureRejectedToken, want: ValidationFailureRejectedToken},
		{name: "timeout", category: ghostfolioclient.FailureTimeout, want: ValidationFailureTimeout},
		{name: "connectivity", category: ghostfolioclient.FailureConnectivityProblem, want: ValidationFailureConnectivityProblem},
		{name: "unsuccessful response", category: ghostfolioclient.FailureUnsuccessfulServerResponse, want: ValidationFailureUnsuccessfulServerResponse},
		{name: "incompatible contract", category: ghostfolioclient.FailureIncompatibleServerContract, want: ValidationFailureIncompatibleServerContract},
		{name: "unknown", category: ghostfolioclient.FailureCategory("unknown"), want: ValidationFailureConnectivityProblem},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := validationFailureReasonFromBoundary(testCase.category); got != testCase.want {
				t.Fatalf("validation failure reason mismatch: got %q want %q", got, testCase.want)
			}
		})
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

	var outcome = service.Validate(context.Background(), ValidateRequest{Config: config, SecurityToken: "token"})
	if outcome.FailureReason != ValidationFailureUnsuccessfulServerResponse {
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

	var outcome = service.Validate(context.Background(), ValidateRequest{Config: config, SecurityToken: "token"})
	if outcome.FailureReason != ValidationFailureIncompatibleServerContract {
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

	var outcome = service.Validate(context.Background(), ValidateRequest{Config: config, SecurityToken: "token"})
	if !outcome.Success || outcome.Attempt.Status != AttemptStatusSuccess || outcome.Attempt.AttemptID == "" || outcome.DetailReason != "communication_ok" || outcome.FailureReason != ValidationFailureNone {
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

	var outcome = service.Validate(context.Background(), ValidateRequest{Config: config, SecurityToken: "token"})
	if outcome.FailureReason != ValidationFailureTimeout || outcome.Attempt.Status != AttemptStatusFailure {
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
	if outcome.FailureReason != ValidationFailureTimeout {
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
