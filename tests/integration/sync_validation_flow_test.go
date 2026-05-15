// Package integration verifies black-box workflow behavior for the current
// slice, including sync-validation journeys that execute the production
// Ghostfolio runtime path against mocked HTTP servers.
// Authored by: OpenCode
package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// ghostfolioScenario describes the mocked Ghostfolio HTTP behavior that one
// production-path sync-validation integration test should exercise.
// Authored by: OpenCode
type ghostfolioScenario struct {
	authStatus            int
	authContentType       string
	authBody              string
	authDelay             time.Duration
	activitiesStatus      int
	activitiesContentType string
	activitiesBody        string
	activitiesDelay       time.Duration
}

// syncValidationFixture wires a remembered config to the production sync
// service that the integration workflow should execute.
// Authored by: OpenCode
type syncValidationFixture struct {
	config  configmodel.AppSetupConfig
	service runtime.SyncService
}

func TestSyncValidationSuccessUsesProductionRuntimePath(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var server = newGhostfolioScenarioServer(t, ghostfolioScenario{
		activitiesBody: `{"activities":[{"id":"activity-1","date":"2026-01-31T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":10,"unitPriceInAssetProfileCurrency":10,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}],"count":1}`,
	})
	var fixture = syncValidationFixture{
		config: mustCustomSetupConfig(t, server.URL),
		service: runtime.NewSyncService(
			ghostfolioclient.New(server.Client()),
			time.Second,
			tempDir,
			true,
			decimalsupport.NewService(),
			syncnormalize.NewNormalizer(),
			syncvalidate.NewValidator(),
			snapshotstore.NewEncryptedStore(tempDir, nil),
		),
	}
	var model = newSyncValidationModel(t, fixture)

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)

	model, cmd := startSyncValidationAttempt(t, model)
	model = applyValidationBatch(t, model, cmd)

	if model.ActiveScreen() != "validation_result" {
		t.Fatalf("expected validation result screen, got %s", model.ActiveScreen())
	}

	var content = model.View().Content
	if !strings.Contains(content, "Activity data was stored securely for future use.") {
		t.Fatalf("expected successful sync summary, got %q", content)
	}
	if !strings.Contains(content, "No report-generation") || !strings.Contains(content, "cached-data browsing workflow") {
		t.Fatalf("expected success follow-up text, got %q", content)
	}
	if strings.Contains(content, "abc123") || strings.Contains(content, "jwt") {
		t.Fatalf("expected transient secrets to stay out of the rendered result, got %q", content)
	}
}

func TestSyncValidationFailureCategoriesUseProductionRuntimePath(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name           string
		buildFixture   func(*testing.T) syncValidationFixture
		wantCategory   runtime.ValidationFailureReason
		wantFollowUp   string
		wantSecretSafe bool
	}{
		{
			name: "rejected token",
			buildFixture: func(t *testing.T) syncValidationFixture {
				t.Helper()
				var server = newGhostfolioScenarioServer(t, ghostfolioScenario{authStatus: http.StatusForbidden})
				return newSyncValidationFixture(t, server.Client(), server.URL, time.Second)
			},
			wantCategory:   runtime.ValidationFailureRejectedToken,
			wantFollowUp:   "The supplied token was rejected. Try again with a valid Ghostfolio security token.",
			wantSecretSafe: true,
		},
		{
			name: "timeout",
			buildFixture: func(t *testing.T) syncValidationFixture {
				t.Helper()
				var server = newGhostfolioScenarioServer(t, ghostfolioScenario{activitiesDelay: 200 * time.Millisecond})
				return newSyncValidationFixture(t, server.Client(), server.URL, 20*time.Millisecond)
			},
			wantCategory:   runtime.ValidationFailureTimeout,
			wantFollowUp:   "Sync again or return to the main menu. No protected activity data was stored.",
			wantSecretSafe: true,
		},
		{
			name: "connectivity problem",
			buildFixture: func(t *testing.T) syncValidationFixture {
				t.Helper()
				var server = httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
				var origin = server.URL
				var client = server.Client()
				server.Close()
				return newSyncValidationFixture(t, client, origin, time.Second)
			},
			wantCategory:   runtime.ValidationFailureConnectivityProblem,
			wantFollowUp:   "Sync again or return to the main menu. No protected activity data was stored.",
			wantSecretSafe: true,
		},
		{
			name: "unsuccessful server response",
			buildFixture: func(t *testing.T) syncValidationFixture {
				t.Helper()
				var server = newGhostfolioScenarioServer(t, ghostfolioScenario{activitiesStatus: http.StatusUnauthorized})
				return newSyncValidationFixture(t, server.Client(), server.URL, time.Second)
			},
			wantCategory:   runtime.ValidationFailureUnsuccessfulServerResponse,
			wantFollowUp:   "Sync again or return to the main menu. No protected activity data was stored.",
			wantSecretSafe: true,
		},
		{
			name: "incompatible server contract",
			buildFixture: func(t *testing.T) syncValidationFixture {
				t.Helper()
				var server = newGhostfolioScenarioServer(t, ghostfolioScenario{activitiesStatus: http.StatusBadRequest})
				return newSyncValidationFixture(t, server.Client(), server.URL, time.Second)
			},
			wantCategory:   runtime.ValidationFailureIncompatibleServerContract,
			wantFollowUp:   "The selected server responded, but it did not satisfy the supported contract",
			wantSecretSafe: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var fixture = testCase.buildFixture(t)
			var model = newSyncValidationModel(t, fixture)

			model = openSyncValidation(t, model)
			model = typeToken(t, model, "abc123")
			model = blurTokenInput(t, model)

			model, cmd := startSyncValidationAttempt(t, model)
			model = applyValidationBatch(t, model, cmd)

			if model.ActiveScreen() != "validation_result" {
				t.Fatalf("expected validation result screen, got %s", model.ActiveScreen())
			}

			var content = model.View().Content
			var expectedCategory = fmt.Sprintf("Failure Category: %s", testCase.wantCategory)
			if !strings.Contains(content, expectedCategory) {
				t.Fatalf("expected failure category %q, got %q", expectedCategory, content)
			}
			if !strings.Contains(content, testCase.wantFollowUp) {
				t.Fatalf("expected follow-up text %q, got %q", testCase.wantFollowUp, content)
			}
			if testCase.wantSecretSafe && (strings.Contains(content, "abc123") || strings.Contains(content, "jwt")) {
				t.Fatalf("expected transient secrets to stay out of the rendered result, got %q", content)
			}
		})
	}
}

func TestFailedValidationDefaultActionRetriesValidation(t *testing.T) {
	t.Parallel()

	var server = newGhostfolioScenarioServer(t, ghostfolioScenario{authStatus: http.StatusForbidden})
	var fixture = newSyncValidationFixture(t, server.Client(), server.URL, time.Second)
	var model = newSyncValidationModel(t, fixture)

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)
	model, cmd := startSyncValidationAttempt(t, model)
	model = applyValidationBatch(t, model, cmd)

	var updated, focusCmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(focusCmd)
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "sync_validation" {
		t.Fatalf("expected Sync Again to reopen sync validation, got %s", model.ActiveScreen())
	}
}

func TestSuccessfulValidationDefaultActionReturnsToMainMenu(t *testing.T) {
	t.Parallel()

	var server = newGhostfolioScenarioServer(t, ghostfolioScenario{})
	var fixture = newSyncValidationFixture(t, server.Client(), server.URL, time.Second)
	var model = newSyncValidationModel(t, fixture)

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)
	model, cmd := startSyncValidationAttempt(t, model)
	model = applyValidationBatch(t, model, cmd)

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected successful validation to return to the main menu by default, got %s", model.ActiveScreen())
	}
}

func TestSyncValidationBusyStateStillHandlesResizeBeforeCompletion(t *testing.T) {
	t.Parallel()

	var server = newGhostfolioScenarioServer(t, ghostfolioScenario{})
	var fixture = newSyncValidationFixture(t, server.Client(), server.URL, time.Second)
	var model = newSyncValidationModel(t, fixture)

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)

	model, cmd := startSyncValidationAttempt(t, model)

	var updated, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model = assertFlowModel(t, updated)
	if got := model.View().Content; got == "" {
		t.Fatalf("expected rendered content after resize")
	}

	model = applyValidationBatch(t, model, cmd)
	if model.ActiveScreen() != "validation_result" {
		t.Fatalf("expected validation result screen after the delayed batch completed, got %s", model.ActiveScreen())
	}
}

func TestSyncValidationNoPersistenceBeyondSetup(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var store = configstore.NewJSONStore(tempDir)
	var server = newGhostfolioScenarioServer(t, ghostfolioScenario{})
	var fixture = syncValidationFixture{
		config: mustCustomSetupConfig(t, server.URL),
		service: runtime.NewSyncService(
			ghostfolioclient.New(server.Client()),
			time.Second,
			tempDir,
			true,
			decimalsupport.NewService(),
			syncnormalize.NewNormalizer(),
			syncvalidate.NewValidator(),
			snapshotstore.NewEncryptedStore(tempDir, nil),
		),
	}
	if err := store.Save(context.Background(), fixture.config); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var model = flow.NewModel(newFlowDependenciesWithStore(t, bootstrap.StartupState{ActiveConfig: &fixture.config}, true, fixture.service, store))

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)
	model, cmd := startSyncValidationAttempt(t, model)
	model = applyValidationBatch(t, model, cmd)

	var entries []os.DirEntry
	var err error
	entries, err = os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("read config directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "ghostfolio-cryptogains" {
		t.Fatalf("unexpected persisted files: %#v", entries)
	}

	var appEntries, readErr = os.ReadDir(filepath.Join(tempDir, "ghostfolio-cryptogains"))
	if readErr != nil {
		t.Fatalf("read app directory: %v", readErr)
	}
	if len(appEntries) != 2 {
		t.Fatalf("expected setup plus snapshots directory, got %#v", appEntries)
	}
	if got := model.View().Content; strings.Contains(got, "abc123") || strings.Contains(got, "jwt") {
		t.Fatalf("expected result screen to remain secret-safe, got %q", got)
	}
}

// newGhostfolioScenarioServer creates one deterministic httptest server for the
// sync-validation runtime path.
// Authored by: OpenCode
func newGhostfolioScenarioServer(t *testing.T, scenario ghostfolioScenario) *httptest.Server {
	t.Helper()

	if scenario.authStatus == 0 {
		scenario.authStatus = http.StatusOK
	}
	if scenario.authContentType == "" && scenario.authStatus >= http.StatusOK && scenario.authStatus < http.StatusMultipleChoices {
		scenario.authContentType = "application/json"
	}
	if scenario.authBody == "" {
		scenario.authBody = `{"authToken":"jwt"}`
	}
	if scenario.activitiesStatus == 0 {
		scenario.activitiesStatus = http.StatusOK
	}
	if scenario.activitiesContentType == "" && scenario.activitiesStatus >= http.StatusOK && scenario.activitiesStatus < http.StatusMultipleChoices {
		scenario.activitiesContentType = "application/json"
	}
	if scenario.activitiesBody == "" {
		scenario.activitiesBody = `{"activities":[],"count":0}`
	}

	var server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			if scenario.authDelay > 0 {
				time.Sleep(scenario.authDelay)
			}
			if scenario.authContentType != "" {
				writer.Header().Set("Content-Type", scenario.authContentType)
			}
			writer.WriteHeader(scenario.authStatus)
			_, _ = writer.Write([]byte(scenario.authBody))
		case "/api/v1/activities":
			if scenario.activitiesDelay > 0 {
				time.Sleep(scenario.activitiesDelay)
			}
			if scenario.activitiesContentType != "" {
				writer.Header().Set("Content-Type", scenario.activitiesContentType)
			}
			writer.WriteHeader(scenario.activitiesStatus)
			_, _ = writer.Write([]byte(scenario.activitiesBody))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

// newSyncValidationFixture constructs a production runtime sync service and a
// remembered custom-origin config for one integration test.
// Authored by: OpenCode
func newSyncValidationFixture(t *testing.T, client *http.Client, origin string, requestTimeout time.Duration) syncValidationFixture {
	t.Helper()

	return syncValidationFixture{
		config: mustCustomSetupConfig(t, origin),
		service: func() runtime.SyncService {
			var tempDir = t.TempDir()
			return runtime.NewSyncService(ghostfolioclient.New(client), requestTimeout, tempDir, true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), snapshotstore.NewEncryptedStore(tempDir, nil))
		}(),
	}
}

// newSyncValidationModel constructs the root flow model for sync-validation
// integration tests that should execute the production runtime path.
// Authored by: OpenCode
func newSyncValidationModel(t *testing.T, fixture syncValidationFixture) *flow.Model {
	t.Helper()

	return flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &fixture.config}, fixture.config.AllowDevHTTP, fixture.service))
}

// startSyncValidationAttempt submits the current token value and returns the
// batch command that will deliver the async validation messages.
// Authored by: OpenCode
func startSyncValidationAttempt(t *testing.T, model *flow.Model) (*flow.Model, tea.Cmd) {
	t.Helper()

	var updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if cmd == nil {
		t.Fatalf("expected validation batch command")
	}
	if got := model.View().Content; !strings.Contains(got, "Syncing and storing activity history") {
		t.Fatalf("expected busy state after submit, got %q", got)
	}

	return model, cmd
}

// applyValidationBatch runs the asynchronous validation batch to completion and
// applies each resulting message to the flow model.
// Authored by: OpenCode
func applyValidationBatch(t *testing.T, model *flow.Model, cmd tea.Cmd) *flow.Model {
	t.Helper()

	var message = testutil.RunCmd(cmd)
	var batch, ok = message.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected validation command to return tea.BatchMsg, got %T", message)
	}

	for _, batchCmd := range batch {
		var batchMessage = testutil.RunCmd(batchCmd)
		if batchMessage == nil {
			continue
		}

		var updated, _ = model.Update(batchMessage)
		model = assertFlowModel(t, updated)
	}

	return model
}

// mustCustomSetupConfig returns a valid remembered custom-origin setup for
// integration tests that need a self-hosted Ghostfolio origin.
// Authored by: OpenCode
func mustCustomSetupConfig(t *testing.T, origin string) configmodel.AppSetupConfig {
	t.Helper()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, origin, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	return config
}

// openSyncValidation enters the sync-validation workflow from the main menu.
// Authored by: OpenCode
func openSyncValidation(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	return updated.(*flow.Model)
}

// typeToken types one token value into the focused Ghostfolio security-token
// field.
// Authored by: OpenCode
func typeToken(t *testing.T, model *flow.Model, token string) *flow.Model {
	t.Helper()
	for _, runeValue := range token {
		updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Text: string(runeValue), Code: runeValue}))
		_ = testutil.RunCmd(cmd)
		model = updated.(*flow.Model)
	}
	return model
}

// blurTokenInput returns focus from the token input to the sync-validation menu.
// Authored by: OpenCode
func blurTokenInput(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	_ = testutil.RunCmd(cmd)
	return updated.(*flow.Model)
}
