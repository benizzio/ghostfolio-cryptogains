// Package integration verifies black-box workflow behavior for the current
// slice, including the unlocked Sync and Reports context entry behavior.
// Authored by: OpenCode
package integration

import (
	"context"
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

// syncReportsContextService is a deterministic SyncService test double for the
// unlocked Sync and Reports context integration coverage.
// Authored by: OpenCode
type syncReportsContextService struct {
	unlockResult runtime.SyncReportsContextResult
	runOutcome   runtime.SyncOutcome
}

// Run implements runtime.SyncService for context-oriented integration tests.
// Authored by: OpenCode
func (s *syncReportsContextService) Run(context.Context, runtime.SyncRequest) runtime.SyncOutcome {
	return s.runOutcome
}

func (s *syncReportsContextService) GenerateDiagnosticReport(context.Context, runtime.DiagnosticReportRequest) (string, error) {
	return "", nil
}

func (s *syncReportsContextService) ProtectedDataState() runtime.ProtectedDataState {
	if s.runOutcome.Success {
		return s.unlockResult.ProtectedData
	}
	return runtime.ProtectedDataState{}
}

func (s *syncReportsContextService) UnlockSelectedServerSnapshot(context.Context, configmodel.AppSetupConfig, string) runtime.SyncReportsContextResult {
	return s.unlockResult
}

func (s *syncReportsContextService) CheckServerReplacement(configmodel.AppSetupConfig) runtime.ServerReplacementCheck {
	return runtime.ServerReplacementCheck{}
}

func TestSyncReportsContextUnlockShowsNoDataReadiness(t *testing.T) {
	t.Parallel()

	var config = mustCloudSetupConfig(t)
	var service = &syncReportsContextService{
		unlockResult: runtime.SyncReportsContextResult{ReportUnavailableReason: runtime.ReportFailureNoSyncedDataAvailable},
	}
	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, service))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.PasteMsg{Content: "token-123"})
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected sync and reports menu, got %s", model.ActiveScreen())
	}
	var content = model.View().Content
	if !strings.Contains(content, "Sync Data") || !strings.Contains(content, "Generate Capital Gains Report") {
		t.Fatalf("expected context actions, got %q", content)
	}
	if !strings.Contains(content, "Sync Data: no synced data available") {
		t.Fatalf("expected no-data readiness, got %q", content)
	}
	if !strings.Contains(content, "Generate Capital Gains Report: unavailable") {
		t.Fatalf("expected report action to remain unavailable, got %q", content)
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected Back To Main Menu to leave context, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; strings.Contains(got, "Last Successful Sync") || strings.Contains(got, "Available Report Years") {
		t.Fatalf("expected protected readiness data to clear on main menu return, got %q", got)
	}
}

func TestSyncReportsContextUnlockShowsExistingDataReadinessAndReusesTokenForSync(t *testing.T) {
	t.Parallel()

	var syncedAt = time.Date(2026, time.May, 20, 13, 30, 0, 0, time.UTC)
	var config = mustCloudSetupConfig(t)
	var service = &syncReportsContextService{
		unlockResult: runtime.SyncReportsContextResult{
			ProtectedData: runtime.ProtectedDataState{
				HasReadableSnapshot:  true,
				ActivityCount:        4,
				LastSuccessfulSyncAt: syncedAt,
				AvailableReportYears: []int{2024, 2025},
			},
			ReportUnavailableReason: runtime.ReportFailureNone,
		},
		runOutcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureTimeout, DetailReason: string(runtime.SyncFailureTimeout)},
	}
	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, service))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.PasteMsg{Content: "token-123"})
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected sync and reports menu, got %s", model.ActiveScreen())
	}
	var content = model.View().Content
	if !strings.Contains(content, "Protected Activity Count: 4") {
		t.Fatalf("expected activity count, got %q", content)
	}
	if !strings.Contains(content, "Available Report Years: 2024, 2025") {
		t.Fatalf("expected report years, got %q", content)
	}
	if !strings.Contains(content, "Generate Capital Gains Report: available") {
		t.Fatalf("expected available report action, got %q", content)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync" {
		t.Fatalf("expected Sync Data to route to sync screen, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !strings.Contains(got, "*********") {
		t.Fatalf("expected stored token to stay masked on sync screen, got %q", got)
	}

	updated, syncCmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	model = applySyncBatch(t, model, syncCmd)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected sync completion from context to return to sync and reports menu, got %s", model.ActiveScreen())
	}
	content = model.View().Content
	if !strings.Contains(content, "Sync Data: no synced data available") {
		t.Fatalf("expected sync return to refresh unlocked readiness from runtime protected state, got %q", content)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected Back To Main Menu to leave context, got %s", model.ActiveScreen())
	}
	updated, unlockCmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(unlockCmd)
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_unlock" {
		t.Fatalf("expected re-entering workflow to require unlock again, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; strings.Contains(got, "*********") {
		t.Fatalf("expected leaving context to clear stored runtime token, got %q", got)
	}
}

// TestSyncReportsContextUnlockUsesSelectedServerSnapshotAndReusesTokenWithProductionRuntime
// verifies selected-server snapshot discovery, unlock, readiness metadata, and
// same-context token reuse through the production runtime service.
// Authored by: OpenCode
func TestSyncReportsContextUnlockUsesSelectedServerSnapshotAndReusesTokenWithProductionRuntime(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var store = configstore.NewJSONStore(tempDir)
	var server = newGhostfolioStorageServer(t, []storagePageFixture{{
		Count:          2,
		ActivitiesJSON: `[{"id":"activity-buy","date":"2024-12-31T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD","symbolProfileId":"asset-btc-context-001"}}]`,
	}, {
		Count:          2,
		ActivitiesJSON: `[{"id":"activity-sell","date":"2025-05-20T13:30:00Z","type":"SELL","quantity":0.25,"valueInBaseCurrency":35,"unitPriceInAssetProfileCurrency":140,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD","symbolProfileId":"asset-btc-context-001"}}]`,
	}})
	var config = mustCustomSetupConfig(t, server.URL)
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save config: %v", err)
	}
	var service = runtime.NewSyncService(
		ghostfolioclient.New(server.Client()),
		time.Second,
		tempDir,
		true,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(tempDir, nil),
	)
	if outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token-123"}); !outcome.Success {
		t.Fatalf("expected preload sync success, got %#v", outcome)
	}

	var model = flow.NewModel(newFlowDependenciesWithStore(t, bootstrap.StartupState{ActiveConfig: &config}, true, service, store))
	model = openSyncEntry(t, model)
	model = typeToken(t, model, "token-123")
	model = blurTokenInputFromSyncEntry(t, model)

	var updated, unlockCmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(unlockCmd)
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected sync and reports menu after snapshot unlock, got %s", model.ActiveScreen())
	}

	var content = model.View().Content
	if !strings.Contains(content, "Sync Data: last successful sync") {
		t.Fatalf("expected unlocked last-sync readiness, got %q", content)
	}
	if !strings.Contains(content, "Protected Activity Count: 2") {
		t.Fatalf("expected unlocked activity count from snapshot, got %q", content)
	}
	if !strings.Contains(content, "Available Report Years: 2024, 2025") {
		t.Fatalf("expected unlocked report years from snapshot, got %q", content)
	}
	if !strings.Contains(content, "Generate Capital Gains Report: available") {
		t.Fatalf("expected report availability after unlock, got %q", content)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync" {
		t.Fatalf("expected Sync Data action to route to sync screen, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !strings.Contains(got, "*********") {
		t.Fatalf("expected runtime token reuse to prefill masked sync token, got %q", got)
	}

	updated, syncCmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	model = applySyncBatch(t, model, syncCmd)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected post-sync return to unlocked context, got %s", model.ActiveScreen())
	}

	content = model.View().Content
	if !strings.Contains(content, "Sync Data: last successful sync") {
		t.Fatalf("expected readiness to remain unlocked after sync completion, got %q", content)
	}
	if !strings.Contains(content, "Protected Activity Count: 2") {
		t.Fatalf("expected refreshed protected-data summary after sync completion, got %q", content)
	}
	if !strings.Contains(content, "Available Report Years: 2024, 2025") {
		t.Fatalf("expected refreshed report years after sync completion, got %q", content)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected Back To Main Menu to leave context, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; strings.Contains(got, "Last Successful Sync") || strings.Contains(got, "Available Report Years") {
		t.Fatalf("expected context readiness data to clear on exit, got %q", got)
	}

	updated, unlockCmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(unlockCmd)
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_unlock" {
		t.Fatalf("expected re-entry to require unlock, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; strings.Contains(got, "*********") {
		t.Fatalf("expected context exit to clear stored runtime token, got %q", got)
	}

	candidates, err := snapshotstore.DiscoverServerCandidates(context.Background(), snapshotstore.NewEncryptedStore(tempDir, nil), config.ServerOrigin)
	if err != nil {
		t.Fatalf("discover selected-server snapshots: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one selected-server snapshot candidate, got %d", len(candidates))
	}
	payload, err := snapshotstore.NewEncryptedStore(tempDir, nil).Read(context.Background(), snapshotstore.ReadRequest{
		Candidate:     candidates[0],
		SecurityToken: "token-123",
	})
	if err != nil {
		t.Fatalf("read selected-server snapshot: %v", err)
	}
	if payload.SetupProfile.ServerOrigin != config.ServerOrigin {
		t.Fatalf("expected selected-server payload origin %q, got %q", config.ServerOrigin, payload.SetupProfile.ServerOrigin)
	}
	if got := len(payload.ProtectedActivityCache.Activities); got != 2 {
		t.Fatalf("expected unlocked snapshot activity count 2, got %d", got)
	}
}
