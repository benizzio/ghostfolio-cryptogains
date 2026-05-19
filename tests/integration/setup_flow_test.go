// Package integration verifies black-box workflow behavior for the current
// slice, including the setup flow and its test-specific sync stub.
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
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// integrationSyncService returns a stable successful sync outcome for
// setup-centric integration tests.
//
// Authored by: OpenCode
type integrationSyncService struct{}

// Run implements runtime.SyncService for setup-centric integration tests.
// Authored by: OpenCode
func (integrationSyncService) Run(context.Context, runtime.SyncRequest) runtime.SyncOutcome {
	return runtime.SyncOutcome{Success: true, DetailReason: "activity_data_stored"}
}

func (integrationSyncService) GenerateDiagnosticReport(context.Context, runtime.DiagnosticReportRequest) (
	string,
	error,
) {
	return "", nil
}

func (integrationSyncService) ProtectedDataState() runtime.ProtectedDataState {
	return runtime.ProtectedDataState{}
}

func (integrationSyncService) CheckServerReplacement(configmodel.AppSetupConfig) runtime.ServerReplacementCheck {
	return runtime.ServerReplacementCheck{}
}

func TestFreshRunCompletesSetupAndReachesMainMenu(t *testing.T) {
	t.Parallel()

	var store = configstore.NewJSONStore(t.TempDir())
	var model = flow.NewModel(
		newFlowDependenciesWithStore(
			t,
			bootstrap.StartupState{NeedsSetup: true, SetupRequirementReason: bootstrap.SetupRequirementMissing},
			false,
			integrationSyncService{},
			store,
		),
	)

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	_, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	result := testutil.RunCmd(cmd)
	updated, _ = model.Update(result)
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected main menu, got %s", model.ActiveScreen())
	}

	var loaded, err = store.Load(context.Background())
	if err != nil {
		t.Fatalf("load remembered setup: %v", err)
	}
	if loaded.ServerOrigin != configmodel.GhostfolioCloudOrigin {
		t.Fatalf("remembered origin mismatch: %q", loaded.ServerOrigin)
	}
}

func TestStartupSkipsSetupWhenRememberedConfigExists(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(
		configmodel.ServerModeGhostfolioCloud,
		configmodel.GhostfolioCloudOrigin,
		false,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(
		newFlowDependencies(
			t,
			bootstrap.StartupState{ActiveConfig: &config},
			false,
			integrationSyncService{},
		),
	)

	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected main menu startup, got %s", model.ActiveScreen())
	}
}

func TestInvalidRememberedSetupFallsBackToSetup(t *testing.T) {
	t.Parallel()

	var model = flow.NewModel(
		newFlowDependencies(
			t,
			bootstrap.StartupState{
				NeedsSetup:             true,
				SetupRequirementReason: bootstrap.SetupRequirementInvalidRememberedSetup,
			},
			false,
			integrationSyncService{},
		),
	)

	if model.ActiveScreen() != "setup" {
		t.Fatalf("expected setup screen, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !strings.Contains(got, "saved server selection is no longer valid") {
		t.Fatalf("expected invalid remembered setup message, got %q", got)
	}
}

func TestSetupFileRemovalAfterStartupDoesNotBreakCurrentRun(t *testing.T) {
	t.Parallel()

	var store = configstore.NewJSONStore(t.TempDir())
	var config, err = configmodel.NewSetupConfig(
		configmodel.ServerModeGhostfolioCloud,
		configmodel.GhostfolioCloudOrigin,
		false,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var model = flow.NewModel(
		newFlowDependenciesWithStore(
			t,
			bootstrap.StartupState{ActiveConfig: &config},
			false,
			integrationSyncService{},
			store,
		),
	)

	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("delete setup file: %v", err)
	}

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "sync" {
		t.Fatalf("expected current run to keep working after setup file removal")
	}
}

func TestFocusedCustomOriginInputEnterReturnsToSavePath(t *testing.T) {
	t.Parallel()

	var model = flow.NewModel(
		newFlowDependencies(
			t,
			bootstrap.StartupState{NeedsSetup: true, SetupRequirementReason: bootstrap.SetupRequirementMissing},
			false,
			integrationSyncService{},
		),
	)

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	_, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	result := testutil.RunCmd(cmd)
	updated, _ = model.Update(result)
	model = assertFlowModel(t, updated)

	model = replaceSetupOriginInput(t, model, "https://example.com")
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)

	if got := model.View().Content; !strings.Contains(got, "> Save And Continue") {
		t.Fatalf("expected setup menu focus to return to Save And Continue, got %q", got)
	}

	_, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	result = testutil.RunCmd(cmd)
	updated, _ = model.Update(result)
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "main_menu" {
		t.Fatalf("expected save path to remain reachable, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !strings.Contains(got, "ghostfolio-cryptogains") {
		t.Fatalf("expected persistent header on main menu, got %q", got)
	}
}

func TestFocusedCustomOriginInputPasteDoesNotTriggerWorkflowNavigation(t *testing.T) {
	t.Parallel()

	var model = flow.NewModel(
		newFlowDependencies(
			t,
			bootstrap.StartupState{NeedsSetup: true, SetupRequirementReason: bootstrap.SetupRequirementMissing},
			false,
			integrationSyncService{},
		),
	)

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)

	updated, _ = model.Update(tea.PasteStartMsg{})
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.PasteMsg{Content: "https://localhost:8080"})
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.PasteEndMsg{})
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "setup" {
		t.Fatalf("expected setup screen to remain active during paste, got %s", model.ActiveScreen())
	}
	if got := model.View().Content; !strings.Contains(got, "https://localhost:8080") {
		t.Fatalf("expected pasted origin in setup input, got %q", got)
	}
	if got := model.View().Content; !strings.Contains(got, "Use Custom Server") {
		t.Fatalf("expected setup workflow to remain active after paste, got %q", got)
	}
}

// replaceSetupOriginInput clears the prefilled custom-origin value and replaces
// it with the provided origin while the setup input owns focus.
// Authored by: OpenCode
func replaceSetupOriginInput(t *testing.T, model *flow.Model, origin string) *flow.Model {
	t.Helper()

	for range configmodel.GhostfolioCloudOrigin {
		updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
		model = assertFlowModel(t, updated)
	}

	updated, _ := model.Update(tea.PasteMsg{Content: origin})
	return assertFlowModel(t, updated)
}

func TestReplaceSetupOriginInputReplacesPrefilledOrigin(t *testing.T) {
	t.Parallel()

	var model = flow.NewModel(
		newFlowDependencies(
			t,
			bootstrap.StartupState{NeedsSetup: true, SetupRequirementReason: bootstrap.SetupRequirementMissing},
			false,
			integrationSyncService{},
		),
	)

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)

	model = replaceSetupOriginInput(t, model, "https://example.com")

	var content = model.View().Content
	if !strings.Contains(content, "https://example.com") {
		t.Fatalf("expected replacement origin in view, got %q", content)
	}
	if strings.Contains(content, configmodel.GhostfolioCloudOrigin+"https://example.com") {
		t.Fatalf("expected prefilled origin to be replaced, got %q", content)
	}
}
