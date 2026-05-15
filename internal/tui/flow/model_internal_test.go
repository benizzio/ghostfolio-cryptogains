// Package flow tests the root Bubble Tea workflow model, including internal
// helper behavior used to drive setup, sync validation, and result navigation.
// Authored by: OpenCode
package flow

import (
	"context"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
)

type testSyncService struct {
	outcome            runtime.ValidationOutcome
	protectedDataState runtime.ProtectedDataState
	replacementCheck   runtime.ServerReplacementCheck
}

func (s testSyncService) Validate(context.Context, runtime.ValidateRequest) runtime.ValidationOutcome {
	return s.outcome
}

func (s testSyncService) ProtectedDataState() runtime.ProtectedDataState {
	return s.protectedDataState
}

func (s testSyncService) CheckServerReplacement(configmodel.AppSetupConfig) runtime.ServerReplacementCheck {
	return s.replacementCheck
}

type cancellingSyncService struct {
	called bool
	ctxErr error
}

func (s *cancellingSyncService) Validate(ctx context.Context, _ runtime.ValidateRequest) runtime.ValidationOutcome {
	s.called = true
	<-ctx.Done()
	s.ctxErr = ctx.Err()
	return runtime.ValidationOutcome{Success: false, DetailReason: string(runtime.ValidationFailureTimeout), FailureReason: runtime.ValidationFailureTimeout}
}

func (*cancellingSyncService) ProtectedDataState() runtime.ProtectedDataState {
	return runtime.ProtectedDataState{}
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
	_ = model.resultHelpText()
	_ = model.View()
	model.active = activeScreen("unknown")
	_ = model.View()
	model.active = setupScreenKey
	_ = nextAttemptID()
	_ = quitCmd()
	model.cancelActiveValidation()
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
	model.active = syncValidationScreenKey
	_ = model.View()
	model.active = validationResultScreenKey
	_ = model.View()
	model.active = activeScreen("unknown")
	updated, cmd := model.Update(struct{}{})
	if cmd != nil || assertUpdatedModel(t, updated).active != activeScreen("unknown") {
		t.Fatalf("expected unknown active screen to ignore messages")
	}
	model.active = setupScreenKey
	model.enterValidationResult(runtime.ValidationOutcome{Success: false})
	model.enterValidationResult(runtime.ValidationOutcome{Success: true})
	if model.result.MenuIndex != 1 {
		t.Fatalf("expected success result to default to main menu option")
	}
	model.enterMainMenu()
	if model.setup.StartupReason != bootstrap.SetupRequirementNone {
		t.Fatalf("expected setup reason to clear on main menu entry")
	}
	_ = model.enterSetup("invalid", bootstrap.SetupRequirementNone)
	_ = model.enterSyncValidation()
	_ = model.selectedSetupOrigin()
	_ = model.setupCanSave()
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

	model.active = syncValidationScreenKey
	model.sync.TokenInput.SetValue("token")
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'c'}))
	if _, ok := runCmdFlow(cmd).(tea.QuitMsg); !ok {
		t.Fatalf("expected quit command")
	}
	if updated.(*Model).sync.TokenInput.Value() != "" {
		t.Fatalf("expected token input reset")
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

func TestUpdateSyncValidationCoversResultAndBusyBranches(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncValidationScreenKey
	model.sync.Busy = true
	model.sync.AttemptID = "current"
	updated, _ := model.Update(validationFinishedMsg{Attempt: "other", Outcome: runtime.ValidationOutcome{Success: true}})
	model = updated.(*Model)
	if model.active != syncValidationScreenKey {
		t.Fatalf("expected mismatched attempt to be ignored")
	}

	updated, _ = model.Update(spinner.TickMsg{ID: model.spinner.ID()})
	model = updated.(*Model)
	updated, _ = model.Update(validationFinishedMsg{Attempt: "current", Outcome: runtime.ValidationOutcome{Success: true}})
	model = updated.(*Model)
	if model.active != validationResultScreenKey {
		t.Fatalf("expected validation result screen")
	}
}

func TestUpdateSyncValidationCoversInputValidationAndBack(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncValidationScreenKey
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
	if model.active != syncValidationScreenKey {
		t.Fatalf("expected sync paste to remain in sync validation workflow")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(*Model)
	if model.sync.InputFocused {
		t.Fatalf("expected input blur")
	}

	updated, cmd = model.Update(tea.PasteStartMsg{})
	if cmd != nil || updated.(*Model).active != syncValidationScreenKey {
		t.Fatalf("expected unfocused sync paste start to be ignored")
	}
	updated, cmd = model.Update(tea.PasteEndMsg{})
	if cmd != nil || updated.(*Model).active != syncValidationScreenKey {
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
	model.active = syncValidationScreenKey
	model.sync.InputFocused = true
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.sync.InputFocused || model.sync.MenuIndex != 0 {
		t.Fatalf("expected enter to return focused sync input to validation menu path")
	}
}

func TestUpdateSyncValidationCoversBusyIgnoreAndSetupRedirect(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = syncValidationScreenKey
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
	model.active = syncValidationScreenKey
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
	model.active = syncValidationScreenKey
	updated, cmd = model.Update(spinner.TickMsg{ID: model.spinner.ID()})
	if cmd != nil || updated.(*Model).sync.Busy {
		t.Fatalf("expected idle spinner tick to be ignored")
	}

	updated, cmd = model.Update(struct{}{})
	if cmd != nil || updated.(*Model).active != syncValidationScreenKey {
		t.Fatalf("expected unrelated sync message to be ignored")
	}
}

func TestUpdateSyncValidationRoutesToServerReplacementWhenRequired(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{replacementCheck: runtime.ServerReplacementCheck{Required: true, ActiveServerOrigin: "https://old.example", SelectedServerOrigin: config.ServerOrigin}}
	model.active = syncValidationScreenKey
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
	if model.active != validationResultScreenKey {
		t.Fatalf("expected cancellation to route to result screen")
	}
	if model.result.Outcome.FailureReason != runtime.SyncFailureServerReplacementCancelled {
		t.Fatalf("expected cancellation outcome, got %#v", model.result.Outcome)
	}

	model = newTestModel(t, &config)
	model.deps.SyncService = testSyncService{replacementCheck: runtime.ServerReplacementCheck{Required: true, ActiveServerOrigin: "https://old.example", SelectedServerOrigin: config.ServerOrigin}}
	model.active = syncValidationScreenKey
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token")
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatalf("expected confirmed replacement to start async sync")
	}
	model = updated.(*Model)
	if model.active != syncValidationScreenKey || !model.sync.Busy {
		t.Fatalf("expected confirmed replacement to resume sync busy state")
	}
}

func TestCancelActiveValidationCancelsContextAndValidationCmdRuns(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var service = &cancellingSyncService{}
	var store = configstore.NewJSONStore(t.TempDir())
	var model = NewModel(Dependencies{
		Options:      bootstrap.DefaultOptions(),
		Startup:      bootstrap.StartupState{ActiveConfig: &config},
		SetupService: runtime.NewSetupService(store, false),
		SyncService:  service,
	})

	model.active = syncValidationScreenKey
	model.sync.InputFocused = false
	model.sync.TokenInput.SetValue("token")
	_, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatalf("expected validation command")
	}
	model.cancelActiveValidation()
	msg := runCmdFlow(cmd)
	var batch, ok = msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected batch command message, got %T", msg)
	}

	var finished validationFinishedMsg
	var found bool
	for _, batchCmd := range batch {
		if candidate, ok := runCmdFlow(batchCmd).(validationFinishedMsg); ok {
			finished = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected validation finished message in batch")
	}
	if finished.Attempt == "" {
		t.Fatalf("expected validation attempt id")
	}
	if !service.called || service.ctxErr == nil {
		t.Fatalf("expected cancelled validation context, called=%v err=%v", service.called, service.ctxErr)
	}
}

func TestUpdateValidationResultCoversNavigation(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = validationResultScreenKey
	model.result = resultState{Outcome: runtime.ValidationOutcome{Success: false}}
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
	if model.active != syncValidationScreenKey {
		t.Fatalf("expected Validate Again to reopen sync validation")
	}

	model.active = validationResultScreenKey
	model.result = resultState{MenuIndex: 1, Outcome: runtime.ValidationOutcome{Success: false}}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*Model)
	if model.active != mainMenuScreenKey {
		t.Fatalf("expected Back To Main Menu to return main menu")
	}

	updated, _ = model.Update(struct{}{})
	if updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected unrelated result message to be ignored")
	}

	model.active = validationResultScreenKey
	model.result = resultState{MenuIndex: 0, Outcome: runtime.ValidationOutcome{Success: false}}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if updated.(*Model).result.MenuIndex != 0 {
		t.Fatalf("expected up at top to stay in place")
	}

	model.active = validationResultScreenKey
	updated, _ = model.Update(struct{}{})
	if updated.(*Model).active != validationResultScreenKey {
		t.Fatalf("expected non-key validation result message to be ignored")
	}
}

func TestUpdateMainMenuCoversEnterAndDefaultKey(t *testing.T) {
	t.Parallel()

	var config = mustSetupConfig(t)
	var model = newTestModel(t, &config)
	model.active = mainMenuScreenKey

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmdFlow(cmd)
	if updated.(*Model).active != syncValidationScreenKey {
		t.Fatalf("expected enter to open sync validation")
	}

	model = newTestModel(t, &config)
	model.active = mainMenuScreenKey
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	if cmd != nil || updated.(*Model).active != mainMenuScreenKey {
		t.Fatalf("expected unrelated main-menu key to be ignored")
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
		Options:      bootstrap.DefaultOptions(),
		Startup:      startup,
		SetupService: runtime.NewSetupService(store, false),
		SyncService:  testSyncService{outcome: runtime.ValidationOutcome{Success: true, DetailReason: "activity_data_stored"}},
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
