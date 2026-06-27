// Package flow tests the root Bubble Tea workflow model, including internal
// helper behavior used to drive setup, sync, and result navigation.
// Authored by: OpenCode
package flow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

type testSyncService struct {
	outcome            runtime.SyncOutcome
	protectedDataState runtime.ProtectedDataState
	unlockResult       runtime.SyncReportsContextResult
	replacementCheck   runtime.ServerReplacementCheck
	diagnosticPath     string
	diagnosticErr      error
}

type testReportService struct {
	outcome runtime.ReportOutcome
	request runtime.ReportGenerationRequest
	called  bool
}

func (s testSyncService) Run(context.Context, runtime.SyncRequest) runtime.SyncOutcome {
	return s.outcome
}

func (s testSyncService) GenerateDiagnosticReport(context.Context, runtime.DiagnosticReportRequest) (string, error) {
	if s.diagnosticErr != nil {
		return "", s.diagnosticErr
	}
	if s.diagnosticPath != "" {
		return s.diagnosticPath, nil
	}
	return "/tmp/report.diagnostic.json", nil
}

func (s testSyncService) ProtectedDataState() runtime.ProtectedDataState {
	return s.protectedDataState
}

func (s testSyncService) UnlockSelectedServerSnapshot(context.Context, configmodel.AppSetupConfig, string) runtime.SyncReportsContextResult {
	if s.unlockResult.ProtectedData.HasReadableSnapshot || len(s.unlockResult.ProtectedData.AvailableReportYears) > 0 || s.unlockResult.ReportUnavailableReason != "" {
		return s.unlockResult
	}
	return runtime.SyncReportsContextResult{
		UnlockState:             runtime.SyncReportsUnlockStateAuthenticatedNewContext,
		ProtectedData:           s.protectedDataState,
		ReportUnavailableReason: runtime.ReportFailureNoSyncedDataAvailable,
	}
}

func (s testSyncService) CheckServerReplacement(configmodel.AppSetupConfig) runtime.ServerReplacementCheck {
	return s.replacementCheck
}

func (s *testReportService) Generate(_ context.Context, request runtime.ReportGenerationRequest) runtime.ReportOutcome {
	s.called = true
	s.request = request
	if s.outcome.Request == (reportmodel.ReportRequest{}) {
		s.outcome.Request = request.Request
	}
	return s.outcome
}

type cancellingSyncService struct {
	called bool
	ctxErr error
}

func (s *cancellingSyncService) Run(ctx context.Context, _ runtime.SyncRequest) runtime.SyncOutcome {
	s.called = true
	<-ctx.Done()
	s.ctxErr = ctx.Err()
	return runtime.SyncOutcome{Success: false, DetailReason: string(runtime.SyncFailureTimeout), FailureReason: runtime.SyncFailureTimeout}
}

func (*cancellingSyncService) GenerateDiagnosticReport(context.Context, runtime.DiagnosticReportRequest) (string, error) {
	return "", nil
}

func (*cancellingSyncService) ProtectedDataState() runtime.ProtectedDataState {
	return runtime.ProtectedDataState{}
}

func (*cancellingSyncService) UnlockSelectedServerSnapshot(context.Context, configmodel.AppSetupConfig, string) runtime.SyncReportsContextResult {
	return runtime.SyncReportsContextResult{
		UnlockState:             runtime.SyncReportsUnlockStateAuthenticatedNewContext,
		ReportUnavailableReason: runtime.ReportFailureNoSyncedDataAvailable,
	}
}

func (*cancellingSyncService) CheckServerReplacement(configmodel.AppSetupConfig) runtime.ServerReplacementCheck {
	return runtime.ServerReplacementCheck{}
}

// assertUpdatedModel converts an updated Bubble Tea model into the concrete
// flow model type for deterministic test assertions.
// Authored by: OpenCode
func assertUpdatedModel(t *testing.T, updated tea.Model) *Model {
	t.Helper()

	var model, ok = updated.(*Model)
	if !ok {
		t.Fatalf("expected updated model to be *Model, got %T", updated)
	}

	return model
}

func TestModelInitAndHelpers(t *testing.T) {
	t.Parallel()

	var model = newTestModel(t, nil)
	if model.Init() != nil {
		t.Fatalf("expected nil init command")
	}
	if model.ActiveScreen() != "setup" {
		t.Fatalf("expected setup screen")
	}
	if model.setup.StartupReason != bootstrap.SetupRequirementMissing {
		t.Fatalf("expected missing-setup startup reason")
	}
	if model.currentServerOrigin() != configmodel.GhostfolioCloudOrigin {
		t.Fatalf("unexpected default origin")
	}
	_ = model.setupMenuItems()
	_ = model.mainMenuItems()
	_ = model.syncMenuItems()
	_ = model.resultMenuItems()
	_ = model.setupHelpText()
	_ = model.mainMenuHelpText()
	_ = model.syncHelpText()
	_ = model.serverReplacementHelpText()
	_ = model.resultHelpText()
	_ = model.reportSelectionHelpText()
	_ = model.reportBusyHelpText()
	_ = model.reportResultHelpText()
	_ = model.View()
	model.active = activeScreen("unknown")
	_ = model.View()
	model.active = setupScreenKey
	_ = nextAttemptID()
	_ = quitCmd()
	model.cancelActiveSync()
	if got := setupInvalidMessage(bootstrap.SetupRequirementInvalidRememberedSetup); got == "" {
		t.Fatalf("expected invalid-remembered-setup message")
	}
	var config = mustSetupConfig(t)
	model.currentConfig = &config
	_ = model.currentServerOrigin()
	model.currentConfig = nil
	model.sync.Busy = true
	if model.syncMenuItems() != nil {
		t.Fatalf("expected nil sync menu while busy")
	}
	model.sync.Busy = false
	model.currentConfig = &config
	if got := model.setupHelpText(); got == "" {
		t.Fatalf("expected setup help text with current config")
	}
	model.active = mainMenuScreenKey
	_ = model.View()
	model.active = syncReportsUnlockScreenKey
	_ = model.View()
	model.active = reportSelectionScreenKey
	_ = model.View()
	model.active = reportBusyScreenKey
	_ = model.View()
	model.active = reportResultScreenKey
	_ = model.View()
	model.active = syncScreenKey
	_ = model.View()
	model.active = serverReplacementScreenKey
	_ = model.View()
	model.active = syncResultScreenKey
	_ = model.View()
	model.active = activeScreen("unknown")
	var updated tea.Model
	var cmd tea.Cmd
	updated, cmd = model.Update(struct{}{})
	if cmd != nil || assertUpdatedModel(t, updated).active != activeScreen("unknown") {
		t.Fatalf("expected unknown active screen to ignore messages")
	}
	model.active = setupScreenKey
	model.enterSyncResult(runtime.SyncOutcome{Success: false})
	model.enterSyncResult(runtime.SyncOutcome{Success: true})
	if model.result.MenuIndex != 1 {
		t.Fatalf("expected success result to default to main menu option")
	}
	model.enterMainMenu()
	if model.setup.StartupReason != bootstrap.SetupRequirementNone {
		t.Fatalf("expected setup reason to clear on main menu entry")
	}
	if model.syncReports.Active {
		t.Fatalf("expected main menu entry to clear active sync and reports context")
	}
	if model.syncReports.RuntimeToken != "" {
		t.Fatalf("expected main menu entry to clear sync and reports runtime token")
	}
	if model.syncReports.ReportResult.Success || model.syncReports.ReportResult.Message != "" || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureNone || model.syncReports.ReportResult.Request != (reportmodel.ReportRequest{}) || model.syncReports.ReportResult.OutputFile != (reportmodel.ReportOutputFile{}) || model.syncReports.ReportResult.Attempt != (runtime.SyncAttempt{}) || model.syncReports.ReportResult.Diagnostic.Eligible || model.syncReports.ReportResult.Diagnostic.Path != "" || model.syncReports.ReportResult.Diagnostic.GenerationMessage != "" {
		t.Fatalf("expected main menu entry to clear sync and reports report scratch state")
	}
	_ = model.enterSetup("invalid", bootstrap.SetupRequirementNone)
	_ = model.enterSyncReportsUnlock()
	_ = model.enterSync()
	_ = model.enterSyncWithContextToken()
	_ = model.selectedSetupOrigin()
	_ = model.setupCanSave()
	if reportMethodForIndex(-1) != "" {
		t.Fatalf("expected invalid report method index to return empty method")
	}
}

func TestUpdateHandlesWindowResizeAndQuit(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Fatalf("expected no command on resize")
	}
	model = updated.(*Model)
	if model.width != 120 || model.height != 40 {
		t.Fatalf("unexpected size: %dx%d", model.width, model.height)
	}

	model.active = syncScreenKey
	model.sync.TokenInput.SetValue("token")
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'c'}))
	if _, ok := runCmdFlow(cmd).(tea.QuitMsg); !ok {
		t.Fatalf("expected quit command")
	}
	model = assertUpdatedModel(t, updated)
	if model.sync.TokenInput.Value() != "" {
		t.Fatalf("expected token input reset")
	}
	if model.syncReports.RuntimeToken != "" {
		t.Fatalf("expected quit to clear sync and reports runtime token")
	}
}

func TestUpdateSetupCoversSaveSuccessAndError(t *testing.T) {
	t.Parallel()

	var model = newTestModel(t, nil)
	updated, _ := model.Update(setupSavedMsg{Err: context.DeadlineExceeded})
	model = updated.(*Model)
	if model.setup.ValidationMessage == "" {
		t.Fatalf("expected save error message")
	}

	var config = mustSetupConfig(t)
	updated, _ = model.Update(setupSavedMsg{Result: runtime.SaveSetupResult{Config: config}})
	model = updated.(*Model)
	if model.active != mainMenuScreenKey || model.currentConfig == nil {
		t.Fatalf("expected main menu after save")
	}
}

func TestUpdateSetupCoversNavigationAndInput(t *testing.T) {
	t.Parallel()

	var model = newTestModel(t, nil)
	model.setup.InputFocused = true
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	_ = runCmdFlow(cmd)
	model = updated.(*Model)
	if model.setup.OriginInput.Value() == "" {
		t.Fatalf("expected input value")
	}

	model.setup.ValidationMessage = "stale"
	updated, cmd = model.Update(tea.PasteMsg{Content: "https://paste.example"})
	_ = runCmdFlow(cmd)
	model = updated.(*Model)
	if model.setup.ValidationMessage != "" {
		t.Fatalf("expected paste to clear setup validation message")
	}
	if model.setup.OriginInput.Value() == "" {
		t.Fatalf("expected setup paste to update origin input")
	}

	model.setup.InputFocused = false
	updated, cmd = model.Update(tea.PasteStartMsg{})
	if cmd != nil || assertUpdatedModel(t, updated).active != setupScreenKey {
		t.Fatalf("expected unfocused setup paste start to be ignored")
	}
	updated, cmd = model.Update(tea.PasteEndMsg{})
	if cmd != nil || assertUpdatedModel(t, updated).active != setupScreenKey {
		t.Fatalf("expected unfocused setup paste end to be ignored")
	}
	model = updated.(*Model)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmdFlow(cmd)
	model = updated.(*Model)
	if !model.setup.InputFocused {
		t.Fatalf("expected custom-origin input focus")
	}

	model.setup.OriginInput.SetValue("http://localhost:8080")
	model.deps.Options.AllowDevHTTP = false
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.setup.ValidationMessage == "" {
		t.Fatalf("expected setup validation message")
	}

	model.deps.Options.AllowDevHTTP = true
	model.deps.SetupService = runtime.NewSetupService(configstore.NewJSONStore(t.TempDir()), true)
	_, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	msg := runCmdFlow(cmd)
	updated, _ = model.Update(msg)
	model = updated.(*Model)
	if model.active != mainMenuScreenKey {
		t.Fatalf("expected main menu after valid save")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'e'}))
	model = updated.(*Model)
	if model.active != setupScreenKey {
		t.Fatalf("expected edit setup to reopen setup")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	model = updated.(*Model)
	if model.active != mainMenuScreenKey {
		t.Fatalf("expected escape to cancel edit setup")
	}

	model.active = setupScreenKey
	model.setup.MenuIndex = 1
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	_ = runCmdFlow(cmd)
	if !updated.(*Model).setup.InputFocused {
		t.Fatalf("expected tab to focus custom-origin input")
	}

	updated, _ = model.Update(struct{}{})
	if updated.(*Model).active != setupScreenKey {
		t.Fatalf("expected unrelated message to be ignored")
	}

	model.setup.InputFocused = false
	model.setup.OriginInput.Blur()
	model.setup.MenuIndex = 1
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	model = updated.(*Model)
	if model.setup.MenuIndex != 0 {
		t.Fatalf("expected up to move setup menu selection")
	}

	model.setup.SelectedMode = configmodel.ServerModeCustomOrigin
	model.setup.OriginInput.SetValue("https://example.com")
	model.setup.InputFocused = true
	model.setup.OriginInput.Blur()
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	model.setup.ValidationMessage = "stale"
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.setup.SelectedMode != configmodel.ServerModeGhostfolioCloud || model.setup.InputFocused || model.setup.OriginInput.Value() != configmodel.GhostfolioCloudOrigin || model.setup.ValidationMessage != "" {
		t.Fatalf("expected cloud selection branch to reset setup state: %#v", model.setup)
	}

	model.setup.MenuIndex = 2
	model.setup.SelectedMode = "invalid"
	model.setup.OriginInput.SetValue(configmodel.GhostfolioCloudOrigin)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if updated.(*Model).setup.ValidationMessage == "" {
		t.Fatalf("expected invalid server mode save error")
	}

	model.setup.MenuIndex = 99
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if updated.(*Model).setup.MenuIndex != 99 {
		t.Fatalf("expected unsupported setup selection to be ignored")
	}

	model.setup.MenuIndex = 2
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	if updated.(*Model).active != setupScreenKey {
		t.Fatalf("expected unrelated setup-menu key to be ignored")
	}
}

func TestUpdateSyncCoversResultAndBusyBranches(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncScreenKey
	model.sync.Busy = true
	model.sync.AttemptID = "current"
	updated, _ := model.Update(syncFinishedMsg{Attempt: "other", Outcome: runtime.SyncOutcome{Success: true}})
	model = updated.(*Model)
	if model.active != syncScreenKey {
		t.Fatalf("expected mismatched attempt to be ignored")
	}

	updated, _ = model.Update(spinner.TickMsg{ID: model.spinner.ID()})
	model = updated.(*Model)
	updated, _ = model.Update(syncFinishedMsg{Attempt: "current", Outcome: runtime.SyncOutcome{Success: true}})
	model = updated.(*Model)
	if model.active != syncResultScreenKey {
		t.Fatalf("expected sync result screen")
	}

	model = newTestModel(t, &config)
	model.active = syncScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token-123"
	model.syncReports.ReportResult = runtime.ReportOutcome{Success: true, Message: "stale"}
	model.sync.Busy = true
	model.sync.AttemptID = "current"
	model.deps.SyncService = testSyncService{protectedDataState: runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2025}}}
	updated, _ = model.Update(syncFinishedMsg{Attempt: "current", Outcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureTimeout}})
	model = updated.(*Model)
	if model.active != syncReportsMenuScreenKey {
		t.Fatalf("expected active context sync to return to sync and reports menu, got %s", model.active)
	}
	if model.syncReports.SyncResult.Outcome.FailureReason != runtime.SyncFailureTimeout {
		t.Fatalf("expected active context sync failure to stay visible in context, got %#v", model.syncReports.SyncResult)
	}
	if model.syncReports.RuntimeToken != "token-123" {
		t.Fatalf("expected active context sync to keep runtime token, got %q", model.syncReports.RuntimeToken)
	}
	if !model.syncReports.ReportResult.Success || model.syncReports.ReportResult.Message != "stale" || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureNone || model.syncReports.ReportResult.Diagnostic.Eligible || model.syncReports.ReportResult.Diagnostic.Path != "" || model.syncReports.ReportResult.Diagnostic.GenerationMessage != "" {
		t.Fatalf("expected active context sync to preserve report scratch state until context exit, got %#v", model.syncReports.ReportResult)
	}
}

func TestUpdateSyncCoversInputValidationAndBack(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncScreenKey
	model.sync.InputFocused = true
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Text: "t", Code: 't'}))
	_ = runCmdFlow(cmd)
	model = updated.(*Model)
	model.sync.ValidationMessage = "stale"
	updated, cmd = model.Update(tea.PasteMsg{Content: "token"})
	_ = runCmdFlow(cmd)
	model = updated.(*Model)
	if model.sync.ValidationMessage != "" {
		t.Fatalf("expected paste to clear sync validation message")
	}
	if model.active != syncScreenKey {
		t.Fatalf("expected sync paste to remain in sync workflow")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	if model.sync.InputFocused {
		t.Fatalf("expected input blur")
	}

	updated, cmd = model.Update(tea.PasteStartMsg{})
	if cmd != nil || updated.(*Model).active != syncScreenKey {
		t.Fatalf("expected unfocused sync paste start to be ignored")
	}
	updated, cmd = model.Update(tea.PasteEndMsg{})
	if cmd != nil || updated.(*Model).active != syncScreenKey {
		t.Fatalf("expected unfocused sync paste end to be ignored")
	}
	model = updated.(*Model)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.sync.Busy {
		t.Fatalf("expected empty token to block validation")
	}
	if model.sync.ValidationMessage == "" {
		t.Fatalf("expected empty-token validation message")
	}

	model.sync.MenuIndex = 1
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != mainMenuScreenKey {
		t.Fatalf("expected Back to return main menu")
	}

	updated, _ = model.Update(struct{}{})
	if updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected unrelated sync message to be ignored")
	}
}

// TestUpdateSyncReportsUnlockCapturesContextToken verifies unlock-to-context
// behavior for the active `Sync and Reports` menu.
// Authored by: OpenCode
func TestUpdateSyncReportsUnlockCapturesContextToken(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{unlockResult: runtime.SyncReportsContextResult{UnlockState: runtime.SyncReportsUnlockStateAuthenticatedNewContext, ReportUnavailableReason: runtime.ReportFailureNoSyncedDataAvailable}}
	var updated tea.Model
	model.active = syncReportsUnlockScreenKey
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token-123")
	model.sync.TokenInput.Blur()
	model.sync.MenuIndex = 0

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != syncReportsMenuScreenKey {
		t.Fatalf("expected unlock to route to sync and reports menu, got %s", model.active)
	}
	if !model.syncReports.Active {
		t.Fatalf("expected sync and reports context to become active")
	}
	if model.syncReports.RuntimeToken != "token-123" {
		t.Fatalf("expected runtime token capture, got %q", model.syncReports.RuntimeToken)
	}
	if model.sync.TokenInput.Value() != "" {
		t.Fatalf("expected unlock to clear the input after entering context, got %q", model.sync.TokenInput.Value())
	}
	if got := model.View().Content; !strings.Contains(got, "Sync Data: no synced data available") {
		t.Fatalf("expected no-data readiness state after unlock, got %q", got)
	}
	if model.syncReports.UnlockFailure != runtime.SyncFailureNone {
		t.Fatalf("expected successful unlock to clear unlock failure state, got %q", model.syncReports.UnlockFailure)
	}

	model = newTestModel(t, &config)
	model.active = syncReportsUnlockScreenKey
	model.sync.InputFocused = false
	model.sync.MenuIndex = 1
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != mainMenuScreenKey {
		t.Fatalf("expected unlock Back to return main menu")
	}
}

func TestUpdateSyncReportsUnlockRejectedTokenRequiresBackBeforeRetry(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{unlockResult: runtime.SyncReportsContextResult{UnlockState: runtime.SyncReportsUnlockStateRejectedToken, FailureReason: runtime.SyncFailureRejectedToken, ReportUnavailableReason: runtime.ReportFailureNoSyncedDataAvailable}}
	model.active = syncReportsUnlockScreenKey
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token-123")
	model.sync.TokenInput.Blur()

	var updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected rejected-token unlock to stay synchronous")
	}
	model = updated.(*Model)
	if model.active != syncReportsUnlockScreenKey {
		t.Fatalf("expected rejected token to stay on unlock screen, got %s", model.active)
	}
	if model.syncReports.UnlockFailure != runtime.SyncFailureRejectedToken {
		t.Fatalf("expected rejected-token unlock failure state, got %q", model.syncReports.UnlockFailure)
	}
	if model.sync.MenuIndex != 1 {
		t.Fatalf("expected rejected-token branch to select Back, got %d", model.sync.MenuIndex)
	}
	if model.sync.TokenInput.Value() != "token-123" {
		t.Fatalf("expected rejected-token branch to preserve token input on failed screen instance, got %q", model.sync.TokenInput.Value())
	}
	if got := model.syncMenuItems(); len(got) != 2 || got[0].Enabled || !got[1].Enabled {
		t.Fatalf("expected rejected-token branch to disable Unlock and keep Back enabled, got %#v", got)
	}
	if got := model.syncReportsUnlockValidationMessage(); got != "access denied" {
		t.Fatalf("expected access-denied unlock validation message, got %q", got)
	}

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected blocked repeated unlock to remain synchronous")
	}
	model = updated.(*Model)
	if model.active != mainMenuScreenKey {
		t.Fatalf("expected selected Back action to return main menu after rejection, got %s", model.active)
	}
	if model.sync.TokenInput.Value() != "" {
		t.Fatalf("expected leaving rejected unlock screen to clear retained token field, got %q", model.sync.TokenInput.Value())
	}

	cmd = model.enterSyncReportsUnlock()
	_ = runCmdFlow(cmd)
	if model.active != syncReportsUnlockScreenKey {
		t.Fatalf("expected re-entry to return to unlock screen, got %s", model.active)
	}
	if model.sync.TokenInput.Value() != "" {
		t.Fatalf("expected re-entered unlock screen to start with cleared token input, got %q", model.sync.TokenInput.Value())
	}
	if model.syncReports.UnlockFailure != runtime.SyncFailureNone {
		t.Fatalf("expected re-entered unlock screen to clear rejected-token state, got %q", model.syncReports.UnlockFailure)
	}

	model.sync.ValidationMessage = "manual validation"
	model.syncReports.UnlockFailure = runtime.SyncFailureRejectedToken
	if got := model.syncReportsUnlockValidationMessage(); got != "manual validation" {
		t.Fatalf("expected explicit validation message to take precedence, got %q", got)
	}
}

func TestUpdateSyncReportsMenuUsesStoredTokenAndReadiness(t *testing.T) {
	t.Parallel()

	var syncedAt = time.Date(2026, time.May, 20, 13, 30, 0, 0, time.UTC)
	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{unlockResult: runtime.SyncReportsContextResult{
		ProtectedData: runtime.ProtectedDataState{
			HasReadableSnapshot:  true,
			ActivityCount:        3,
			LastSuccessfulSyncAt: syncedAt,
			AvailableReportYears: []int{2024, 2025},
		},
		ReportUnavailableReason: runtime.ReportFailureNone,
	}}
	model.active = syncReportsUnlockScreenKey
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token-123")
	model.sync.TokenInput.Blur()

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)

	if model.active != syncReportsMenuScreenKey {
		t.Fatalf("expected unlock to open sync and reports menu, got %s", model.active)
	}
	if got := model.View().Content; !strings.Contains(got, "Available Report Years: 2024, 2025") || !strings.Contains(got, "Generate Capital Gains Report: available") {
		t.Fatalf("expected reportable readiness details, got %q", got)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != syncScreenKey {
		t.Fatalf("expected Sync Data action to route to sync screen, got %s", model.active)
	}
	if !model.sync.UseContextToken {
		t.Fatalf("expected sync screen to enter context-token mode")
	}
	if model.sync.InputFocused {
		t.Fatalf("expected sync screen to start from the primary action with stored token")
	}
}

func TestUpdateSyncReportsMenuCoversFallbackAndBackBranches(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncReportsMenuScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token-123"
	model.syncReports.ProtectedData = runtime.ProtectedDataState{}
	model.sync.MenuIndex = 1

	var updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || updated.(*Model).active != syncReportsMenuScreenKey {
		t.Fatalf("expected unavailable report action to be ignored")
	}

	model = updated.(*Model)
	model.sync.MenuIndex = 2
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected plain Back action to return without command")
	}
	model = updated.(*Model)
	if model.active != mainMenuScreenKey {
		t.Fatalf("expected menu index 2 without pending diagnostic to return main menu, got %s", model.active)
	}

	model = newTestModel(t, &config)
	model.active = syncReportsMenuScreenKey
	model.syncReports.Active = true
	model.sync.MenuIndex = 3
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected explicit Back action to return without command")
	}
	if updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected menu index 3 to return main menu, got %s", updated.(*Model).active)
	}

	model = newTestModel(t, &config)
	model.active = syncReportsMenuScreenKey
	model.sync.MenuIndex = 99
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || updated.(*Model).active != syncReportsMenuScreenKey {
		t.Fatalf("expected unsupported sync reports selection to be ignored")
	}

	model = newTestModel(t, &config)
	model.active = syncReportsMenuScreenKey
	model.sync.MenuIndex = 1
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if cmd != nil {
		t.Fatalf("expected up navigation to stay synchronous")
	}
	if updated.(*Model).sync.MenuIndex != 0 {
		t.Fatalf("expected up to move sync reports selection to top, got %d", updated.(*Model).sync.MenuIndex)
	}
}

func TestUpdateSyncReportsMenuCoversDiagnosticPromptAndGeneration(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.currentConfig = &config
	model.active = syncReportsMenuScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token-123"
	model.syncReports.SyncResult = syncContextResultState{
		Outcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
			Attempt:       runtime.SyncAttempt{AttemptID: "attempt-1"},
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true, Request: runtime.DiagnosticReportRequest{}},
		},
	}
	model.sync.MenuIndex = 2

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if !model.syncReports.SyncResult.Busy {
		t.Fatalf("expected context diagnostic generation to enter busy state")
	}
	if model.sync.MenuIndex != 2 {
		t.Fatalf("expected diagnostic action to remain selected while busy, got %d", model.sync.MenuIndex)
	}
	updated, _ = model.Update(runCmdFlow(cmd))
	model = updated.(*Model)
	if model.syncReports.SyncResult.Outcome.Diagnostic.Path == "" {
		t.Fatalf("expected context diagnostic generation to populate written path")
	}
	if model.syncReportsHasPendingDiagnostic() {
		t.Fatalf("expected written path to clear pending diagnostic action")
	}
	if model.sync.MenuIndex != 0 {
		t.Fatalf("expected selection to return to Sync Data after written path, got %d", model.sync.MenuIndex)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.sync.MenuIndex != 2 {
		t.Fatalf("expected down to skip disabled report action after written path, got %d", model.sync.MenuIndex)
	}
	updated, _ = model.Update(struct{}{})
	if updated.(*Model).active != syncReportsMenuScreenKey {
		t.Fatalf("expected unrelated sync reports message to be ignored")
	}

	model = newTestModel(t, &config)
	model.currentConfig = &config
	model.active = syncReportsMenuScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token-123"
	model.syncReports.SyncResult = syncContextResultState{
		Outcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
			Attempt:       runtime.SyncAttempt{AttemptID: "attempt-2"},
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true},
		},
	}
	model.deps.SyncService = testSyncService{diagnosticErr: errors.New("report boom")}
	model.sync.MenuIndex = 2

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if !model.syncReports.SyncResult.Busy {
		t.Fatalf("expected context diagnostic generation error path to enter busy state")
	}
	updated, _ = model.Update(runCmdFlow(cmd))
	model = updated.(*Model)
	if model.syncReports.SyncResult.Outcome.Diagnostic.Path != "" {
		t.Fatalf("expected context diagnostic error path to keep written path empty, got %#v", model.syncReports.SyncResult.Outcome.Diagnostic)
	}
	if model.syncReports.SyncResult.StatusMessage == "" {
		t.Fatalf("expected context diagnostic write error to surface in status message")
	}
	if model.sync.MenuIndex != 2 {
		t.Fatalf("expected failed diagnostic action to remain selected, got %d", model.sync.MenuIndex)
	}

	model.syncReports.SyncResult.Busy = true
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || !updated.(*Model).syncReports.SyncResult.Busy || updated.(*Model).sync.MenuIndex != 2 {
		t.Fatalf("expected busy sync reports diagnostic state to ignore key input, got syncResult=%#v menu=%d cmd=%v", updated.(*Model).syncReports.SyncResult, updated.(*Model).sync.MenuIndex, cmd)
	}
	if items := model.syncReportsMenuItems(); len(items) != 4 || items[0].Enabled || items[1].Enabled || items[2].Enabled || items[3].Enabled {
		t.Fatalf("expected busy sync reports menu items to be present but disabled, got %#v", items)
	}
	updated, _ = model.Update(diagnosticReportFinishedMsg{Path: "/tmp/report.diagnostic.json"})
	if updated.(*Model).syncReports.SyncResult.Outcome.Diagnostic.Path != "/tmp/report.diagnostic.json" {
		t.Fatalf("expected direct diagnostic finished message to update context path, got %#v", updated.(*Model).syncReports.SyncResult.Outcome.Diagnostic)
	}
}

func TestUpdateSyncReportsMenuSkipsDisabledReportAction(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncReportsMenuScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token-123"
	model.syncReports.ProtectedData = runtime.ProtectedDataState{}
	model.sync.MenuIndex = 0

	var updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if cmd != nil {
		t.Fatalf("expected down navigation to remain synchronous")
	}
	model = updated.(*Model)
	if model.sync.MenuIndex != 2 {
		t.Fatalf("expected down to skip disabled report action and land on Back To Main Menu, got %d", model.sync.MenuIndex)
	}

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if cmd != nil {
		t.Fatalf("expected up navigation to remain synchronous")
	}
	model = updated.(*Model)
	if model.sync.MenuIndex != 0 {
		t.Fatalf("expected up to skip disabled report action and return to Sync Data, got %d", model.sync.MenuIndex)
	}

	model.sync.MenuIndex = 0
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if cmd != nil {
		t.Fatalf("expected up navigation at top to remain synchronous")
	}
	model = updated.(*Model)
	if model.sync.MenuIndex != 0 {
		t.Fatalf("expected up at top to stay on Sync Data, got %d", model.sync.MenuIndex)
	}

	model.sync.MenuIndex = 2
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if cmd != nil {
		t.Fatalf("expected down navigation at bottom to remain synchronous")
	}
	model = updated.(*Model)
	if model.sync.MenuIndex != 2 {
		t.Fatalf("expected down at bottom to stay on Back To Main Menu, got %d", model.sync.MenuIndex)
	}

	model.sync.MenuIndex = 1
	model.moveSyncReportsMenuSelection(0, model.syncReportsMenuItems())
	if model.sync.MenuIndex != 1 {
		t.Fatalf("expected zero-step sync reports navigation to keep selection, got %d", model.sync.MenuIndex)
	}
	model.moveSyncReportsMenuSelection(1, nil)
	if model.sync.MenuIndex != 1 {
		t.Fatalf("expected empty sync reports menu navigation to keep selection, got %d", model.sync.MenuIndex)
	}
}

func TestUpdateReportCoversSelectionBusyAndResultBranches(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var reportService = &testReportService{outcome: runtime.ReportOutcome{
		Success: true,
		Message: "Saved the report to \"/tmp/report.md\" and requested automatic opening.",
		OutputFile: reportmodel.ReportOutputFile{
			DocumentsDirectory: "/tmp",
			Filename:           "report.md",
			Path:               "/tmp/report.md",
			SavedAt:            time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
			OpenRequested:      true,
		},
	}}
	var model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{
		unlockResult: runtime.SyncReportsContextResult{
			ProtectedData: runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024, 2025}},
		},
	}
	model.deps.ReportService = reportService
	model.active = syncReportsMenuScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token-123"
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024, 2025}}

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != reportSelectionScreenKey {
		t.Fatalf("expected report selection screen, got %s", model.active)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.report.YearIndex != 1 || model.report.SelectedYear != 2025 {
		t.Fatalf("expected report year selection to move to 2025, got %#v", model.report)
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.report.MethodIndex != 1 {
		t.Fatalf("expected report method selection to move, got %#v", model.report)
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if cmd != nil || model.report.FocusArea != reportSelectionFocusAction {
		t.Fatalf("expected base-currency activation to advance to actions, got cmd=%v report=%#v", cmd, model.report)
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != reportBusyScreenKey || !model.report.Busy {
		t.Fatalf("expected report generation busy state, got active=%s report=%#v", model.active, model.report)
	}
	var reportBatch = runCmdFlow(cmd)
	var batch, ok = reportBatch.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected report command to return tea.BatchMsg, got %T", reportBatch)
	}
	for _, batchCmd := range batch {
		var batchMessage = runCmdFlow(batchCmd)
		if batchMessage == nil {
			continue
		}
		updated, _ = model.Update(batchMessage)
		model = updated.(*Model)
	}
	if model.active != reportResultScreenKey {
		t.Fatalf("expected report result screen, got %s", model.active)
	}
	if !reportService.called {
		t.Fatalf("expected report service Generate to be called")
	}
	if reportService.request.Request.Year != 2025 || reportService.request.Request.CostBasisMethod != reportmodel.CostBasisMethodLIFO || reportService.request.Request.ReportBaseCurrency != reportmodel.ReportBaseCurrencyUSD {
		t.Fatalf("expected report service request to use selected year and method, got %#v", reportService.request.Request)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != reportSelectionScreenKey {
		t.Fatalf("expected Generate Another Report to reopen selection, got %s", model.active)
	}
	if model.syncReports.ReportResult.Success || model.syncReports.ReportResult.Message != "" || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureNone || model.syncReports.ReportResult.Request != (reportmodel.ReportRequest{}) || model.syncReports.ReportResult.OutputFile != (reportmodel.ReportOutputFile{}) || model.syncReports.ReportResult.Attempt != (runtime.SyncAttempt{}) || model.syncReports.ReportResult.Diagnostic.Eligible || model.syncReports.ReportResult.Diagnostic.Path != "" || model.syncReports.ReportResult.Diagnostic.GenerationMessage != "" {
		t.Fatalf("expected Generate Another Report to clear transient report state, got %#v", model.syncReports)
	}

	model.enterReportResult(runtime.ReportOutcome{Success: false, FailureReason: runtime.ReportFailureUnsupportedReportCalculation, Request: reportmodel.ReportRequest{Year: 2024, CostBasisMethod: reportmodel.CostBasisMethodFIFO, RequestedAt: time.Now()}})
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != syncReportsMenuScreenKey {
		t.Fatalf("expected Back To Sync and Reports to return to context menu, got %s", model.active)
	}
	if model.syncReports.ReportResult.Success || model.syncReports.ReportResult.Message != "" || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureNone || model.syncReports.ReportResult.Request != (reportmodel.ReportRequest{}) || model.syncReports.ReportResult.OutputFile != (reportmodel.ReportOutputFile{}) || model.syncReports.ReportResult.Attempt != (runtime.SyncAttempt{}) || model.syncReports.ReportResult.Diagnostic.Eligible || model.syncReports.ReportResult.Diagnostic.Path != "" || model.syncReports.ReportResult.Diagnostic.GenerationMessage != "" {
		t.Fatalf("expected result dismissal to clear transient report state, got %#v", model.syncReports)
	}
	model.enterMainMenu()
	if model.syncReports.ReportResult.Success || model.syncReports.ReportResult.Message != "" || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureNone || model.syncReports.ReportResult.Request != (reportmodel.ReportRequest{}) || model.syncReports.ReportResult.OutputFile != (reportmodel.ReportOutputFile{}) || model.syncReports.ReportResult.Attempt != (runtime.SyncAttempt{}) || model.syncReports.ReportResult.Diagnostic.Eligible || model.syncReports.ReportResult.Diagnostic.Path != "" || model.syncReports.ReportResult.Diagnostic.GenerationMessage != "" {
		t.Fatalf("expected context exit to clear report history state, got %#v", model.syncReports)
	}
}

// TestReportSelectionFocusIncludesBaseCurrencyPane verifies keyboard focus
// moves through year, method, base currency, and action panes before wrapping.
// Authored by: OpenCode
func TestReportSelectionFocusIncludesBaseCurrencyPane(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)

	model.advanceReportSelectionFocus()
	if model.report.FocusArea != 1 {
		t.Fatalf("expected first focus move to reach method pane, got %d", model.report.FocusArea)
	}
	model.advanceReportSelectionFocus()
	if model.report.FocusArea != 2 {
		t.Fatalf("expected second focus move to reach base-currency pane, got %d", model.report.FocusArea)
	}
	model.advanceReportSelectionFocus()
	if model.report.FocusArea != 3 {
		t.Fatalf("expected third focus move to reach action pane after base-currency pane, got %d", model.report.FocusArea)
	}
	model.advanceReportSelectionFocus()
	if model.report.FocusArea != 0 {
		t.Fatalf("expected focus to wrap to year pane after actions, got %d", model.report.FocusArea)
	}
}

// TestReportSelectionDefaultsBaseCurrencyAndCanGenerate verifies a report can
// start from the initial selection state because a base currency is preselected.
// Authored by: OpenCode
func TestReportSelectionDefaultsBaseCurrencyAndCanGenerate(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var reportService = &testReportService{}
	var model = newTestModel(t, &config)
	model.deps.ReportService = reportService
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	if model.report.BaseCurrencyIndex != 0 || model.report.SelectedBaseCurrency != reportmodel.ReportBaseCurrencyUSD {
		t.Fatalf("expected initial report state to select USD at index 0, got %#v", model.report)
	}
	if !model.reportCanGenerate() {
		t.Fatalf("expected initial report state to be ready to generate, got %#v", model.report)
	}

	var items = model.reportSelectionMenuItems()
	if len(items) == 0 || items[0].Label != component.GenerateReportActionLabel {
		t.Fatalf("expected Generate Report to be the first report-selection action, got %#v", items)
	}
	if !items[0].Enabled {
		t.Fatalf("expected Generate Report to be enabled with default base currency, got %#v", items[0])
	}

	model.report.FocusArea = reportSelectionFocusAction
	model.report.ActionIndex = 0
	var updated, cmd = model.activateReportSelection()
	if cmd == nil {
		t.Fatalf("expected default generation activation to start asynchronous report generation")
	}
	model = updated.(*Model)
	if model.active != reportBusyScreenKey || !model.report.Busy {
		t.Fatalf("expected default generation activation to enter report busy state, got active=%s report=%#v", model.active, model.report)
	}

	model = newTestModel(t, &config)
	model.deps.ReportService = reportService
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)

	model.report.FocusArea = reportSelectionFocusAction
	model.report.SelectedBaseCurrency = ""
	updated, cmd = model.activateReportSelection()
	if cmd != nil {
		t.Fatalf("expected disabled activation to stay synchronous")
	}
	model = assertUpdatedModel(t, updated)
	if model.active != reportSelectionScreenKey || reportService.called {
		t.Fatalf("expected disabled activation to remain on selection without report service call, active=%s called=%v", model.active, reportService.called)
	}
}

// TestReportSelectionActivationFallsBackFromInvalidBaseCurrencyIndex verifies
// activating a stale base-currency index restores the first supported currency.
// Authored by: OpenCode
func TestReportSelectionActivationFallsBackFromInvalidBaseCurrencyIndex(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	model.report.FocusArea = reportSelectionFocusBaseCurrency
	model.report.BaseCurrencyIndex = 99
	model.report.SelectedBaseCurrency = ""

	var updated, cmd = model.activateReportSelection()
	if cmd != nil {
		t.Fatalf("expected base-currency fallback activation to stay synchronous")
	}
	model = assertUpdatedModel(t, updated)
	if model.report.BaseCurrencyIndex != 0 || model.report.SelectedBaseCurrency != reportmodel.ReportBaseCurrencyUSD || model.report.FocusArea != reportSelectionFocusAction {
		t.Fatalf("expected invalid base-currency index to fall back to USD and actions, got %#v", model.report)
	}
}

func TestUpdateReportCoversIgnoredAndFallbackBranches(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024, 2025}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)

	var updated, cmd = model.Update(struct{}{})
	if cmd != nil || updated.(*Model).active != reportSelectionScreenKey {
		t.Fatalf("expected non-key report message to be ignored on selection screen")
	}

	model = updated.(*Model)
	model.report.FocusArea = reportSelectionFocusAction
	model.report.ActionIndex = 1
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected Back action to remain synchronous")
	}
	model = updated.(*Model)
	if model.active != syncReportsMenuScreenKey || model.sync.MenuIndex != 1 {
		t.Fatalf("expected selection Back to return to report menu item, got active=%s index=%d", model.active, model.sync.MenuIndex)
	}

	model = newTestModel(t, &config)
	model.sync.InputFocused = false
	model.sync.UseContextToken = true
	updated, cmd = model.focusSyncTokenInput()
	if cmd != nil || updated.(*Model).sync.InputFocused {
		t.Fatalf("expected focusSyncTokenInput to ignore context-token mode")
	}

	model = newTestModel(t, &config)
	model.active = syncScreenKey
	model.sync.UseContextToken = true
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = ""
	updated, cmd = model.startSync()
	if cmd != nil {
		t.Fatalf("expected missing context token to fail synchronously")
	}
	if updated.(*Model).sync.ValidationMessage == "" {
		t.Fatalf("expected missing context token to surface validation guidance")
	}

	model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024, 2025}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	model.report.FocusArea = reportSelectionFocusAction
	model.report.ActionIndex = 0
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if cmd != nil {
		t.Fatalf("expected report selection up to remain synchronous")
	}
	model = updated.(*Model)
	if model.report.ActionIndex != 0 {
		t.Fatalf("expected action index at top to stay in place, got %d", model.report.ActionIndex)
	}

	model.report.FocusArea = 0
	model.report.YearIndex = 1
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	model = updated.(*Model)
	if model.report.YearIndex != 0 || model.report.SelectedYear != 2024 {
		t.Fatalf("expected year selection to move upward, got %#v", model.report)
	}

	model.report.FocusArea = 1
	model.report.MethodIndex = 1
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	model = updated.(*Model)
	if model.report.MethodIndex != 0 {
		t.Fatalf("expected method selection to move upward, got %#v", model.report)
	}

	model.report.FocusArea = reportSelectionFocusAction
	model.report.ActionIndex = 0
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.report.ActionIndex != 1 {
		t.Fatalf("expected action selection to move downward, got %#v", model.report)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	if model.report.FocusArea != 0 {
		t.Fatalf("expected focus toggle to wrap to first area, got %d", model.report.FocusArea)
	}

	model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	model.report.MethodIndex = 99
	model.report.FocusArea = reportSelectionFocusAction
	model.report.SelectedBaseCurrency = reportmodel.ReportBaseCurrencyUSD
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected invalid request failure to stay synchronous")
	}
	model = updated.(*Model)
	if model.active != reportResultScreenKey || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
		t.Fatalf("expected invalid request to route to report result failure, got active=%s outcome=%#v", model.active, model.syncReports.ReportResult)
	}

	model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	model.report.FocusArea = reportSelectionFocusAction
	model.report.SelectedBaseCurrency = reportmodel.ReportBaseCurrencyUSD
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected missing report service failure to stay synchronous")
	}
	model = updated.(*Model)
	if model.active != reportResultScreenKey || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
		t.Fatalf("expected missing report service to route to report result failure, got active=%s outcome=%#v", model.active, model.syncReports.ReportResult)
	}

	model = newTestModel(t, &config)
	model.active = reportBusyScreenKey
	model.report.Busy = false
	updated, cmd = model.Update(spinner.TickMsg{ID: model.spinner.ID()})
	if cmd != nil || updated.(*Model).active != reportBusyScreenKey {
		t.Fatalf("expected idle report-busy spinner tick to be ignored")
	}

	model = updated.(*Model)
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || updated.(*Model).active != reportBusyScreenKey {
		t.Fatalf("expected report busy screen to ignore key input")
	}

	model = newTestModel(t, &config)
	model.active = reportResultScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{}
	model.syncReports.ReportResult = runtime.ReportOutcome{FailureReason: runtime.ReportFailureUnsupportedReportCalculation}
	model.report.ActionIndex = 0
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if cmd != nil {
		t.Fatalf("expected report result up to remain synchronous")
	}
	model = updated.(*Model)
	if model.report.ActionIndex != 0 {
		t.Fatalf("expected report result selection at top to stay in place, got %d", model.report.ActionIndex)
	}

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if cmd != nil {
		t.Fatalf("expected report result down to remain synchronous")
	}
	if updated.(*Model).report.ActionIndex != 0 {
		t.Fatalf("expected report result with one item to ignore down, got %d", updated.(*Model).report.ActionIndex)
	}

	model = newTestModel(t, &config)
	model.active = reportResultScreenKey
	model.report.Busy = true
	if items := model.reportResultMenuItems(); len(items) != 2 || items[0].Label != "Generate Diagnostic Report" || items[1].Label != "Back To Sync and Reports" || items[0].Enabled || items[1].Enabled {
		t.Fatalf("expected busy report-result menu to disable visible actions, got %#v", items)
	}
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.syncReports.ReportResult = runtime.ReportOutcome{FailureReason: runtime.ReportFailureUnsupportedReportCalculation}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if cmd != nil || model.report.ActionIndex != 0 || model.active != reportResultScreenKey {
		t.Fatalf("expected busy report-result navigation to be ignored, got active=%s index=%d cmd=%v", model.active, model.report.ActionIndex, cmd)
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if cmd != nil || model.active != reportResultScreenKey || !model.report.Busy || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
		t.Fatalf("expected busy report-result action to be ignored, got active=%s busy=%t outcome=%#v cmd=%v", model.active, model.report.Busy, model.syncReports.ReportResult, cmd)
	}

	model = newTestModel(t, &config)
	model.active = reportResultScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.syncReports.ReportResult = runtime.ReportOutcome{
		FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
		Attempt:       runtime.SyncAttempt{AttemptID: "report-attempt-1"},
		Diagnostic: runtime.DiagnosticReportState{
			Eligible: true,
			Request:  runtime.DiagnosticReportRequest{},
		},
	}
	if items := model.reportResultMenuItems(); len(items) != 3 || items[0].Label != "Generate Diagnostic Report" || items[1].Label != "Back To Sync and Reports" || items[2].Label != "Generate Another Report" {
		t.Fatalf("expected report result diagnostic menu items, got %#v", items)
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if !model.report.Busy {
		t.Fatalf("expected report diagnostic generation to enter busy state")
	}
	updated, _ = model.Update(runCmdFlow(cmd))
	model = updated.(*Model)
	if model.syncReports.ReportResult.Diagnostic.Path == "" {
		t.Fatalf("expected report diagnostic path after generation, got %#v", model.syncReports.ReportResult.Diagnostic)
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.report.ActionIndex != 1 {
		t.Fatalf("expected report result menu to move to Back after diagnostics, got %d", model.report.ActionIndex)
	}

	model = newTestModel(t, &config)
	model.active = reportResultScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.syncReports.ReportResult = runtime.ReportOutcome{
		FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
		Diagnostic: runtime.DiagnosticReportState{
			Eligible: true,
		},
	}
	model.report.ActionIndex = 1
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected pending-diagnostic Back action to remain synchronous")
	}
	model = updated.(*Model)
	if model.active != syncReportsMenuScreenKey || model.sync.MenuIndex != 1 {
		t.Fatalf("expected pending-diagnostic Back action to return to sync and reports, got active=%s index=%d", model.active, model.sync.MenuIndex)
	}

	model = newTestModel(t, &config)
	model.active = reportResultScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.syncReports.ReportResult = runtime.ReportOutcome{
		FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
		Diagnostic: runtime.DiagnosticReportState{
			Eligible: true,
		},
	}
	model.report.ActionIndex = 2
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected pending-diagnostic default action to remain synchronous")
	}
	model = updated.(*Model)
	if model.active != reportSelectionScreenKey {
		t.Fatalf("expected pending-diagnostic default action to reopen report selection, got %s", model.active)
	}

	model = newTestModel(t, &config)
	model.active = activeScreen("report_unknown")
	updated, cmd = model.updateReport(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || updated.(*Model).active != activeScreen("report_unknown") {
		t.Fatalf("expected unknown report screen to ignore key input")
	}

	model = newTestModel(t, &config)
	model.report.AttemptID = "current"
	updated, cmd = model.Update(reportFinishedMsg{Attempt: "other", Outcome: runtime.ReportOutcome{Success: true}})
	if cmd != nil || updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected mismatched report attempt to be ignored")
	}
}

func TestFocusedInputEnterReturnsToPrimaryMenus(t *testing.T) {
	t.Parallel()

	var model = newTestModel(t, nil)
	model.setup.InputFocused = true
	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.setup.InputFocused || model.setup.MenuIndex != 2 {
		t.Fatalf("expected enter to return focused setup input to save menu path")
	}

	var config = mustSetupConfig(t)
	model = newTestModel(t, &config)
	model.active = syncScreenKey
	model.sync.InputFocused = true
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.sync.InputFocused || model.sync.MenuIndex != 0 {
		t.Fatalf("expected enter to return focused sync input to sync menu path")
	}
}

func TestUpdateSyncCoversBusyIgnoreAndSetupRedirect(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncScreenKey
	model.sync.Busy = true
	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if !model.sync.Busy {
		t.Fatalf("expected busy state to ignore key input")
	}
	updated, cmd := model.Update(tea.PasteMsg{Content: "token"})
	if cmd != nil || !updated.(*Model).sync.Busy {
		t.Fatalf("expected busy sync paste to be ignored")
	}
	model = updated.(*Model)

	model.sync.Busy = false
	model.currentConfig = nil
	model.sync.TokenInput.SetValue("token")
	model.sync.InputFocused = false
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmdFlow(cmd)
	if updated.(*Model).active != setupScreenKey {
		t.Fatalf("expected setup redirect when config is missing")
	}

	model = newTestModel(t, &config)
	model.active = syncScreenKey
	model.sync.InputFocused = false
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if cmd != nil || updated.(*Model).sync.MenuIndex != 0 {
		t.Fatalf("expected up at top to be ignored")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.sync.MenuIndex != 1 {
		t.Fatalf("expected down to move menu index")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	model = updated.(*Model)
	if model.sync.MenuIndex != 0 {
		t.Fatalf("expected up to move sync menu index")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.sync.MenuIndex != 1 {
		t.Fatalf("expected down to move back to lower sync item")
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if cmd != nil || updated.(*Model).sync.MenuIndex != 1 {
		t.Fatalf("expected down at bottom to stay in place")
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if runCmdFlow(cmd) == nil || !updated.(*Model).sync.InputFocused {
		t.Fatalf("expected tab to focus token input")
	}
	model = updated.(*Model)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	if model.sync.InputFocused {
		t.Fatalf("expected tab while focused to blur token input")
	}

	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	if cmd != nil || updated.(*Model).sync.MenuIndex != 1 {
		t.Fatalf("expected unrelated sync-menu key to be ignored")
	}

	model.sync.MenuIndex = 99
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || updated.(*Model).sync.MenuIndex != 99 {
		t.Fatalf("expected unsupported sync selection to be ignored")
	}

	model = newTestModel(t, &config)
	model.active = syncScreenKey
	updated, cmd = model.Update(spinner.TickMsg{ID: model.spinner.ID()})
	if cmd != nil || updated.(*Model).sync.Busy {
		t.Fatalf("expected idle spinner tick to be ignored")
	}

	updated, cmd = model.Update(struct{}{})
	if cmd != nil || updated.(*Model).active != syncScreenKey {
		t.Fatalf("expected unrelated sync message to be ignored")
	}
}

func TestUpdateSyncRoutesToServerReplacementWhenRequired(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{replacementCheck: runtime.ServerReplacementCheck{Required: true, ActiveServerOrigin: "https://old.example", SelectedServerOrigin: config.ServerOrigin}}
	model.active = syncScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token"
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token")

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected no async command before confirmation")
	}
	model = updated.(*Model)
	if model.active != serverReplacementScreenKey {
		t.Fatalf("expected server replacement screen, got %s", model.active)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != syncReportsMenuScreenKey {
		t.Fatalf("expected cancellation to return to sync and reports menu")
	}
	if model.replacement.PendingToken != "" {
		t.Fatalf("expected cancellation to scrub pending token")
	}
	if model.syncReports.RuntimeToken != "token" {
		t.Fatalf("expected active context cancellation to preserve runtime token, got %q", model.syncReports.RuntimeToken)
	}

	model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{replacementCheck: runtime.ServerReplacementCheck{Required: true, ActiveServerOrigin: "https://old.example", SelectedServerOrigin: config.ServerOrigin}}
	model.active = syncScreenKey
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token")
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatalf("expected confirmed replacement to start async sync")
	}
	model = updated.(*Model)
	if model.replacement.PendingToken != "" {
		t.Fatalf("expected confirmed replacement to scrub pending token")
	}
	if model.active != syncScreenKey || !model.sync.Busy {
		t.Fatalf("expected confirmed replacement to resume sync busy state")
	}
}

func TestCancelActiveSyncCancelsContextAndSyncCmdRuns(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var service = &cancellingSyncService{}
	var store = configstore.NewJSONStore(t.TempDir())
	var model = NewModel(Dependencies{
		Options:       bootstrap.DefaultOptions(),
		Startup:       bootstrap.StartupState{ActiveConfig: &config},
		SetupService:  runtime.NewSetupService(store, false),
		SyncService:   service,
		ReportService: nil,
	})

	model.active = syncScreenKey
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token")
	_, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatalf("expected sync command")
	}
	model.cancelActiveSync()
	msg := runCmdFlow(cmd)
	var batch, ok = msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected batch command message, got %T", msg)
	}

	var finished syncFinishedMsg
	var found bool
	for _, batchCmd := range batch {
		if candidate, ok := runCmdFlow(batchCmd).(syncFinishedMsg); ok {
			finished = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected sync finished message in batch")
	}
	if finished.Attempt == "" {
		t.Fatalf("expected sync attempt id")
	}
	if !service.called || service.ctxErr == nil {
		t.Fatalf("expected cancelled sync context, called=%v err=%v", service.called, service.ctxErr)
	}
}

func TestUpdateSyncResultCoversNavigation(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncResultScreenKey
	model.result = resultState{Outcome: runtime.SyncOutcome{Success: false}}
	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.result.MenuIndex != 1 {
		t.Fatalf("expected menu index to move down")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	model = updated.(*Model)
	if model.result.MenuIndex != 0 {
		t.Fatalf("expected menu index to move up")
	}
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmdFlow(cmd)
	model = updated.(*Model)
	if model.active != syncScreenKey {
		t.Fatalf("expected Sync Again to reopen sync")
	}

	model = newTestModel(t, &config)
	model.active = syncResultScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token-123"
	model.result = resultState{Outcome: runtime.SyncOutcome{Success: false}}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmdFlow(cmd)
	model = updated.(*Model)
	if model.active != syncScreenKey {
		t.Fatalf("expected context Sync Again to reopen sync")
	}
	if !model.sync.UseContextToken {
		t.Fatalf("expected context Sync Again to reuse the runtime token without exposing it")
	}

	model.active = syncResultScreenKey
	model.result = resultState{MenuIndex: 1, Outcome: runtime.SyncOutcome{Success: false}}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != mainMenuScreenKey {
		t.Fatalf("expected Back To Main Menu to return main menu")
	}

	updated, _ = model.Update(struct{}{})
	if updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected unrelated result message to be ignored")
	}

	model.active = syncResultScreenKey
	model.result = resultState{MenuIndex: 0, Outcome: runtime.SyncOutcome{Success: false}}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if updated.(*Model).result.MenuIndex != 0 {
		t.Fatalf("expected up at top to stay in place")
	}

	model.active = syncResultScreenKey
	model.result = resultState{MenuIndex: 1, Outcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureUnsupportedActivityHistory, Diagnostic: runtime.DiagnosticReportState{Eligible: true}}}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if updated.(*Model).active != syncScreenKey {
		t.Fatalf("expected diagnostic result second action to reopen sync")
	}

	model = newTestModel(t, &config)
	model.active = syncResultScreenKey
	model.result = resultState{MenuIndex: 2, Outcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureUnsupportedActivityHistory, Diagnostic: runtime.DiagnosticReportState{Eligible: true}}}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected diagnostic result default action to return main menu")
	}

	model = newTestModel(t, &config)
	model.active = syncResultScreenKey
	model.syncReports.Active = true
	model.syncReports.RuntimeToken = "token-456"
	model.result = resultState{MenuIndex: 1, Outcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureUnsupportedActivityHistory, Diagnostic: runtime.DiagnosticReportState{Eligible: true}}}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmdFlow(cmd)
	model = updated.(*Model)
	if model.active != syncScreenKey {
		t.Fatalf("expected diagnostic result sync-again action to reopen sync")
	}
	if !model.sync.UseContextToken {
		t.Fatalf("expected diagnostic result sync-again to reuse runtime token without exposing it")
	}

	model.active = syncResultScreenKey
	model.result = resultState{Outcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureUnsupportedActivityHistory, Diagnostic: runtime.DiagnosticReportState{Eligible: true}}}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if !model.result.Busy {
		t.Fatalf("expected diagnostic report generation to enter busy state")
	}
	updated, _ = model.Update(runCmdFlow(cmd))
	model = updated.(*Model)
	if model.result.Outcome.Diagnostic.Path == "" {
		t.Fatalf("expected diagnostic report generation to populate the written path")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.result.MenuIndex != 1 {
		t.Fatalf("expected diagnostic result menu to move to Sync Again")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	model = updated.(*Model)
	if model.result.MenuIndex != 0 {
		t.Fatalf("expected updated diagnostic result menu to return to Sync Again after report generation")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != syncScreenKey {
		t.Fatalf("expected diagnostic result Sync Again to reopen sync")
	}

	model.active = syncResultScreenKey
	updated, _ = model.Update(struct{}{})
	if updated.(*Model).active != syncResultScreenKey {
		t.Fatalf("expected non-key sync result message to be ignored")
	}

	model = newTestModel(t, &config)
	model.active = syncResultScreenKey
	model.result = resultState{Busy: true, Outcome: runtime.SyncOutcome{Success: false}}
	if items := model.resultMenuItems(); len(items) != 3 || items[0].Enabled || items[1].Enabled || items[2].Enabled {
		t.Fatalf("expected busy result menu items to be present but disabled, got %#v", items)
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || !updated.(*Model).result.Busy || updated.(*Model).result.MenuIndex != 0 {
		t.Fatalf("expected busy sync result to ignore key input, got model=%#v cmd=%v", updated.(*Model).result, cmd)
	}
}

func TestUpdateMainMenuCoversEnterAndDefaultKey(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = mainMenuScreenKey

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmdFlow(cmd)
	if updated.(*Model).active != syncReportsUnlockScreenKey {
		t.Fatalf("expected enter to open sync and reports unlock")
	}
	if updated.(*Model).syncReports.Active {
		t.Fatalf("expected main-menu entry not to activate context before unlock")
	}

	model = newTestModel(t, &config)
	model.active = mainMenuScreenKey
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	if cmd != nil || updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected unrelated main-menu key to be ignored")
	}

	updated, cmd = model.updateMainMenu(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil || updated.(*Model).active != syncReportsUnlockScreenKey {
		t.Fatalf("expected direct main-menu enter handler to route to unlock screen")
	}
}

// TestGenerateDiagnosticReportAndServerReplacementIgnoreBranches verifies the
// remaining result and replacement navigation ignore paths.
// Authored by: OpenCode
func TestGenerateDiagnosticReportAndServerReplacementIgnoreBranches(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{diagnosticErr: errors.New("report boom")}
	model.active = syncResultScreenKey
	model.result = resultState{Outcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureUnsupportedActivityHistory, Diagnostic: runtime.DiagnosticReportState{Eligible: true, Request: runtime.DiagnosticReportRequest{}}}}
	model.currentConfig = &config

	updated, cmd := model.generateDiagnosticReport()
	model = updated.(*Model)
	if !model.result.Busy || model.result.StatusMessage == "" {
		t.Fatalf("expected diagnostic generation to enter busy status, got %#v", model.result)
	}
	updated, _ = model.Update(runCmdFlow(cmd))
	model = updated.(*Model)
	if model.result.Outcome.Diagnostic.Path != "" {
		t.Fatalf("expected diagnostic path to stay empty on write error, got %#v", model.result.Outcome.Diagnostic)
	}
	if model.result.StatusMessage == "" {
		t.Fatalf("expected diagnostic write error to surface on result screen")
	}

	model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{diagnosticErr: errors.New("report boom")}
	model.active = reportResultScreenKey
	model.syncReports.ReportResult = runtime.ReportOutcome{
		FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
		Diagnostic:    runtime.DiagnosticReportState{Eligible: true, Request: runtime.DiagnosticReportRequest{}},
	}
	updated, cmd = model.generateDiagnosticReport()
	model = updated.(*Model)
	if !model.report.Busy || model.syncReports.ReportResult.Diagnostic.GenerationMessage != "Generating diagnostic report..." {
		t.Fatalf("expected report diagnostic generation to enter busy state, got %#v", model.syncReports.ReportResult.Diagnostic)
	}
	updated, _ = model.Update(runCmdFlow(cmd))
	model = updated.(*Model)
	if model.syncReports.ReportResult.Diagnostic.Path != "" {
		t.Fatalf("expected report diagnostic write error to keep empty path, got %#v", model.syncReports.ReportResult.Diagnostic)
	}
	if model.syncReports.ReportResult.Diagnostic.GenerationMessage != "Diagnostic report generation failed. Try again." {
		t.Fatalf("expected report diagnostic error message, got %#v", model.syncReports.ReportResult.Diagnostic)
	}

	model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{}
	model.active = syncResultScreenKey
	model.result = resultState{Outcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureUnsupportedActivityHistory, Diagnostic: runtime.DiagnosticReportState{Eligible: true, Request: runtime.DiagnosticReportRequest{}}}}
	model.currentConfig = &config
	updated, cmd = model.generateDiagnosticReport()
	model = updated.(*Model)
	if model.result.Outcome.Diagnostic.Request.ServerOrigin != config.ServerOrigin || model.result.Outcome.Diagnostic.Request.Attempt.AttemptID != model.result.Outcome.Attempt.AttemptID {
		t.Fatalf("expected diagnostic request defaults to be filled, got %#v", model.result.Outcome.Diagnostic.Request)
	}
	updated, _ = model.Update(runCmdFlow(cmd))
	model = updated.(*Model)
	if model.result.Outcome.Diagnostic.Path == "" {
		t.Fatalf("expected diagnostic path after successful write, got %#v", model.result.Outcome.Diagnostic)
	}

	updated, cmd = model.Update(struct{}{})
	if cmd != nil || updated.(*Model).active != syncResultScreenKey {
		t.Fatalf("expected non-key sync-result message to be ignored")
	}

	model = newTestModel(t, &config)
	model.active = syncReportsMenuScreenKey
	updated, cmd = model.activateSyncReportsGenerateDiagnostic()
	if cmd != nil || updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected sync-reports diagnostic fallback without pending diagnostic to return main menu")
	}

	model = newTestModel(t, &config)
	model.active = syncResultScreenKey
	model.moveSyncResultSelection(0)
	if model.result.MenuIndex != 0 {
		t.Fatalf("expected zero-step sync-result move to keep selection")
	}
	model.moveSyncResultSelection(-1)
	if model.result.MenuIndex != 0 {
		t.Fatalf("expected out-of-bounds sync-result move to keep selection")
	}
	updated, cmd = model.activateSyncResultSelection()
	if cmd == nil || updated.(*Model).active != syncScreenKey {
		t.Fatalf("expected default sync-result action to reopen sync")
	}

	model = newTestModel(t, &config)
	model.active = syncResultScreenKey
	model.result.MenuIndex = 99
	updated, cmd = model.activateSyncResultSelection()
	if cmd != nil || updated.(*Model).active != syncResultScreenKey {
		t.Fatalf("expected unsupported sync-result action to be ignored")
	}

	model = newTestModel(t, &config)
	model.result.Outcome = runtime.SyncOutcome{Success: false, Diagnostic: runtime.DiagnosticReportState{Eligible: true, Path: "/tmp/report.diagnostic.json"}}
	if items := model.resultMenuItems(); len(items) != 2 || items[0].Label != component.SyncAgainActionLabel || items[1].Label != component.BackToMainMenuActionLabel {
		t.Fatalf("expected settled sync-result menu items without diagnostic action, got %#v", items)
	}
	model.result.Outcome = runtime.SyncOutcome{Success: false, Diagnostic: runtime.DiagnosticReportState{Eligible: true}}
	if items := model.resultMenuItems(); len(items) != 3 || !items[0].Enabled || !items[1].Enabled || !items[2].Enabled {
		t.Fatalf("expected pending-diagnostic sync-result menu items to stay enabled, got %#v", items)
	}

	model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024, 2025}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	updated, cmd = model.activateReportSelection()
	if cmd != nil || updated.(*Model).report.FocusArea != 1 {
		t.Fatalf("expected report selection activation before actions to advance focus")
	}

	model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	model.report.FocusArea = reportSelectionFocusAction
	model.report.ActionIndex = 99
	updated, cmd = model.activateReportSelection()
	if cmd != nil || updated.(*Model).active != reportSelectionScreenKey {
		t.Fatalf("expected unsupported report-selection action to be ignored")
	}

	model = newTestModel(t, &config)
	model.active = reportResultScreenKey
	model.report.ActionIndex = 99
	updated, cmd = model.activateReportResultSelection()
	if cmd != nil || updated.(*Model).active != reportResultScreenKey {
		t.Fatalf("expected unsupported report-result action to be ignored")
	}

	model = newTestModel(t, &config)
	model.active = syncReportsMenuScreenKey
	updated, cmd = model.Update(struct{}{})
	if cmd != nil || updated.(*Model).active != syncReportsMenuScreenKey {
		t.Fatalf("expected non-key sync-reports message to be ignored")
	}

	model = newTestModel(t, &config)
	model.active = syncReportsUnlockScreenKey
	model.sync.MenuIndex = 0
	model.syncReports.UnlockFailure = runtime.SyncFailureRejectedToken
	updated, cmd = model.Update(tea.PasteMsg{Content: "blocked-token"})
	if cmd != nil || updated.(*Model).sync.TokenInput.Value() != "" {
		t.Fatalf("expected blocked rejected-token unlock paste to be ignored")
	}
	model = updated.(*Model)
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if cmd != nil {
		t.Fatalf("expected unlock-screen down navigation to remain synchronous")
	}
	model = updated.(*Model)
	if model.sync.MenuIndex != 1 {
		t.Fatalf("expected unlock-screen down navigation to skip disabled Unlock action, got %d", model.sync.MenuIndex)
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if cmd != nil || !updated.(*Model).sync.InputFocused {
		t.Fatalf("expected focus toggle to leave the blocked unlock input state unchanged")
	}

	model = newTestModel(t, &config)
	model.active = syncReportsUnlockScreenKey
	model.deps.SyncService = testSyncService{unlockResult: runtime.SyncReportsContextResult{ReportUnavailableReason: runtime.ReportFailureUnsupportedStoredDataVersion}}
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token-123")
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("expected unsupported stored-data unlock to remain synchronous")
	}
	model = updated.(*Model)
	if model.syncReports.UnlockFailure != runtime.SyncFailureUnsupportedStoredDataVersion || model.sync.ValidationMessage != "unsupported stored-data version" {
		t.Fatalf("expected unsupported stored-data unlock guidance, got failure=%q message=%q", model.syncReports.UnlockFailure, model.sync.ValidationMessage)
	}

	model.moveSyncMenuSelection(0, model.syncMenuItems())
	if model.sync.MenuIndex != 0 {
		t.Fatalf("expected zero-step sync menu move to leave selection unchanged")
	}

	model.active = serverReplacementScreenKey
	updated, cmd = model.Update(struct{}{})
	if cmd != nil || updated.(*Model).active != serverReplacementScreenKey {
		t.Fatalf("expected non-key replacement message to be ignored")
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if updated.(*Model).replacement.MenuIndex != 0 {
		t.Fatalf("expected replacement up at top to stay in place")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*Model)
	if model.replacement.MenuIndex != 1 {
		t.Fatalf("expected replacement down to move menu index")
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if cmd != nil || updated.(*Model).replacement.MenuIndex != 1 {
		t.Fatalf("expected replacement down at bottom to stay in place")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if updated.(*Model).replacement.MenuIndex != 0 {
		t.Fatalf("expected replacement up to move back to first option")
	}

	model = newTestModel(t, &config)
	model.active = serverReplacementScreenKey
	model.replacement.MenuIndex = 1
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != syncResultScreenKey {
		t.Fatalf("expected replacement cancellation outside sync reports to enter sync result")
	}
	if model.result.Outcome.FailureReason != runtime.SyncFailureServerReplacementCancelled {
		t.Fatalf("expected replacement cancellation outcome, got %#v", model.result.Outcome)
	}
}

// TestUpdateReportAndSyncHelperBranches verifies remaining direct report-flow and
// sync-flow branches not reached through broader workflow tests.
// Authored by: OpenCode
func TestUpdateReportAndSyncHelperBranches(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024, 2025}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	model.report.FocusArea = reportSelectionFocusAction
	model.report.ActionIndex = 1

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if cmd != nil || updated.(*Model).report.ActionIndex != 0 {
		t.Fatalf("expected report action up to move back to first action")
	}

	model = updated.(*Model)
	model.report.MethodIndex = 99
	model.report.SelectedBaseCurrency = reportmodel.ReportBaseCurrencyUSD

	updated, cmd = model.startReportGeneration()
	model = updated.(*Model)
	if cmd != nil {
		t.Fatalf("expected invalid report request to fail synchronously")
	}
	if model.active != reportResultScreenKey {
		t.Fatalf("expected invalid report request to enter report result")
	}
	if model.syncReports.ReportResult.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
		t.Fatalf("expected invalid request to map to unsupported calculation, got %#v", model.syncReports.ReportResult)
	}
	if model.syncReports.ReportResult.Request.CostBasisMethod != "" {
		t.Fatalf("expected invalid report method index to propagate empty method, got %#v", model.syncReports.ReportResult.Request)
	}

	model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.syncReports.ProtectedData = runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)
	model.report.FocusArea = reportSelectionFocusAction
	model.report.SelectedYear = 0
	model.report.SelectedBaseCurrency = reportmodel.ReportBaseCurrencyUSD
	model.deps.ReportService = &testReportService{}
	updated, cmd = model.startReportGeneration()
	if cmd != nil {
		t.Fatalf("expected invalid report request with configured service to fail synchronously")
	}
	model = updated.(*Model)
	if model.active != reportResultScreenKey || model.syncReports.ReportResult.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
		t.Fatalf("expected invalid request with configured service to route to report result failure, got active=%s outcome=%#v", model.active, model.syncReports.ReportResult)
	}

	model = newTestModel(t, &config)
	model.active = reportResultScreenKey
	model.report.ActionIndex = 1
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if cmd != nil || updated.(*Model).report.ActionIndex != 0 {
		t.Fatalf("expected report-result up to move to first action")
	}

	model = newTestModel(t, &config)
	model.active = reportSelectionScreenKey
	model.report.AttemptID = "attempt-current"
	updated, cmd = model.Update(reportFinishedMsg{Attempt: "attempt-stale", Outcome: runtime.ReportOutcome{Success: true}})
	if cmd != nil || updated.(*Model).active != reportSelectionScreenKey || updated.(*Model).report.AttemptID != "attempt-current" {
		t.Fatalf("expected stale report-finished message to be ignored")
	}

	model = newTestModel(t, nil)
	updated, cmd = model.unlockSyncReportsContext()
	if cmd != nil || updated.(*Model).active != setupScreenKey || !strings.Contains(updated.(*Model).setup.ValidationMessage, "Complete setup before Sync and Reports can run.") {
		t.Fatalf("expected unlock without setup to return to setup with guidance")
	}

	model = newTestModel(t, &config)
	updated, cmd = model.unlockSyncReportsContext()
	if cmd != nil || updated.(*Model).sync.ValidationMessage == "" {
		t.Fatalf("expected blank unlock token to surface validation message")
	}

	model.active = syncReportsUnlockScreenKey
	model.sync.MenuIndex = 99
	updated, cmd = model.activateSyncReportsUnlockSelection()
	if cmd != nil || updated.(*Model).active != syncReportsUnlockScreenKey || updated.(*Model).sync.MenuIndex != 99 {
		t.Fatalf("expected unsupported unlock selection index to be ignored")
	}

	model = newTestModel(t, &config)
	model.active = syncScreenKey
	model.syncReports.Active = true
	model.sync.TokenInput.SetValue("token-to-clear")
	updated, cmd = model.leaveSync()
	if cmd != nil {
		t.Fatalf("expected leaveSync in sync-reports context not to enqueue a command")
	}
	model = updated.(*Model)
	if model.active != syncReportsMenuScreenKey {
		t.Fatalf("expected leaveSync with active sync-reports context to return to context menu")
	}
	if model.sync.TokenInput.Value() != "" || model.sync.MenuIndex != 0 {
		t.Fatalf("expected leaveSync to reset token input and menu index, got %#v", model.sync)
	}
}

// newTestModel builds a root model with repository-default test dependencies.
// Authored by: OpenCode
func newTestModel(t *testing.T, config *configmodel.AppSetupConfig) *Model {
	t.Helper()
	var startup = bootstrap.StartupState{}
	if config != nil {
		startup.ActiveConfig = config
	} else {
		startup.NeedsSetup = true
		startup.SetupRequirementReason = bootstrap.SetupRequirementMissing
	}
	var store = configstore.NewJSONStore(t.TempDir())
	return NewModel(Dependencies{
		Options:       bootstrap.DefaultOptions(),
		Startup:       startup,
		SetupService:  runtime.NewSetupService(store, false),
		SyncService:   testSyncService{outcome: runtime.SyncOutcome{Success: true, DetailReason: "activity_data_stored"}},
		ReportService: nil,
	})
}

// mustSetupConfig returns a valid Ghostfolio Cloud setup configuration for
// model tests that need remembered startup state.
// Authored by: OpenCode
func mustSetupConfig(t *testing.T) configmodel.AppSetupConfig {
	t.Helper()
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	return config
}

// runCmdFlow executes one Bubble Tea command and returns its resulting message.
// Authored by: OpenCode
func runCmdFlow(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}
