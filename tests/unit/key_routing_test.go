package unit

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

type fakeSyncService struct{}

func (fakeSyncService) Validate(context.Context, runtime.ValidateRequest) runtime.ValidationOutcome {
	return runtime.ValidationOutcome{Success: true, DetailReason: "communication_ok"}
}

func TestTokenInputConsumesPlainCharactersWithoutTriggeringActions(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, fakeSyncService{}))

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = updated.(*flow.Model)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync_validation" {
		t.Fatalf("expected to remain on sync validation screen")
	}
	if view := model.View().Content; !testutil.Contains(view, "*") {
		t.Fatalf("expected masked token input after typing, got %q", view)
	}
}

func TestFocusedInputsEnterReleaseToPrimaryMenusAndPasteSafely(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var setupModel = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{NeedsSetup: true, SetupRequirementReason: bootstrap.SetupRequirementMissing}, false, fakeSyncService{}))

	updated, _ := setupModel.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	setupModel = updated.(*flow.Model)
	updated, cmd := setupModel.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	setupModel = updated.(*flow.Model)

	updated, _ = setupModel.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	setupModel = updated.(*flow.Model)
	if got := setupModel.View().Content; !testutil.Contains(got, "> Save And Continue") {
		t.Fatalf("expected setup enter to return to save menu path, got %q", got)
	}

	var setupPasteModel = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{NeedsSetup: true, SetupRequirementReason: bootstrap.SetupRequirementMissing}, false, fakeSyncService{}))

	updated, _ = setupPasteModel.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	setupPasteModel = updated.(*flow.Model)
	updated, cmd = setupPasteModel.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	setupPasteModel = updated.(*flow.Model)
	updated, _ = setupPasteModel.Update(tea.PasteMsg{Content: "https://example.com"})
	setupPasteModel = updated.(*flow.Model)
	if setupPasteModel.ActiveScreen() != "setup" {
		t.Fatalf("expected setup paste to stay in setup workflow")
	}
	if got := setupPasteModel.View().Content; !testutil.Contains(got, "https://example.com") {
		t.Fatalf("expected pasted setup origin to remain in the input, got %q", got)
	}

	var syncModel = flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, fakeSyncService{}))

	updated, cmd = syncModel.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	syncModel = updated.(*flow.Model)
	updated, _ = syncModel.Update(tea.PasteMsg{Content: "token-123"})
	syncModel = updated.(*flow.Model)
	updated, _ = syncModel.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	syncModel = updated.(*flow.Model)
	if got := syncModel.View().Content; !testutil.Contains(got, "> Validate Communication") {
		t.Fatalf("expected sync enter to return to validation menu path, got %q", got)
	}
	if view := syncModel.View().Content; !testutil.Contains(view, "*********") {
		t.Fatalf("expected masked pasted token after blur, got %q", view)
	}
}
