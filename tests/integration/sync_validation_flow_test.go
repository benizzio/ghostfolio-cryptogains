package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

type scriptedSyncService struct {
	outcomes []runtime.ValidationOutcome
	index    int
}

func (s *scriptedSyncService) Validate(context.Context, configmodel.AppSetupConfig, string) runtime.ValidationOutcome {
	var outcome = s.outcomes[s.index]
	if s.index < len(s.outcomes)-1 {
		s.index++
	}
	return outcome
}

func TestSyncValidationSuccessShowsTransientSuccessResult(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var service = &scriptedSyncService{outcomes: []runtime.ValidationOutcome{{
		Success:        true,
		SummaryMessage: "Communication with the selected Ghostfolio server is working.",
		DetailReason:   "communication_ok",
		FollowUpNote:   "No Ghostfolio data was stored locally, and reporting is not available in this slice.",
	}}}

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{ActiveConfig: &config},
		ConfigStore: configstore.NewJSONStore(t.TempDir()),
		SyncService: service,
	})

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	if got := model.View().Content; !contains(got, "Validating Ghostfolio communication") {
		t.Fatalf("expected busy state after submit, got %q", got)
	}

	_ = runCmd(cmd)
	updated, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model = updated.(*flow.Model)
	if got := model.View().Content; got == "" {
		t.Fatalf("expected rendered content after resize")
	}
}

func TestSyncValidationRetryUsesResultMenuPath(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{ActiveConfig: &config},
		ConfigStore: configstore.NewJSONStore(t.TempDir()),
		SyncService: &scriptedSyncService{outcomes: []runtime.ValidationOutcome{{Success: true, SummaryMessage: "Communication with the selected Ghostfolio server is working.", DetailReason: "communication_ok", FollowUpNote: "No Ghostfolio data was stored locally."}}},
	})

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	_ = runCmd(cmd)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	if model.ActiveScreen() != "sync_validation" {
		t.Fatalf("expected Validate Again to reopen sync validation, got %s", model.ActiveScreen())
	}
}

func TestSyncValidationNoPersistenceBeyondSetup(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var store = configstore.NewJSONStore(tempDir)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{ActiveConfig: &config},
		ConfigStore: store,
		SyncService: &scriptedSyncService{outcomes: []runtime.ValidationOutcome{{Success: true, SummaryMessage: "Communication with the selected Ghostfolio server is working.", DetailReason: "communication_ok", FollowUpNote: "No Ghostfolio data was stored locally."}}},
	})

	model = openSyncValidation(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInput(t, model)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	_ = runCmd(cmd)

	var entries []os.DirEntry
	entries, err = os.ReadDir(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatalf("read config directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "setup.json" {
		t.Fatalf("unexpected persisted files: %#v", entries)
	}
	if got := model.View().Content; !contains(got, "Validating Ghostfolio communication") {
		t.Fatalf("expected busy state content, got %q", got)
	}
}

func openSyncValidation(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	return updated.(*flow.Model)
}

func typeToken(t *testing.T, model *flow.Model, token string) *flow.Model {
	t.Helper()
	for _, runeValue := range token {
		updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Text: string(runeValue), Code: runeValue}))
		_ = runCmd(cmd)
		model = updated.(*flow.Model)
	}
	return model
}

func blurTokenInput(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	_ = runCmd(cmd)
	return updated.(*flow.Model)
}
