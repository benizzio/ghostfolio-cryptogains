package unit

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

type fakeSyncService struct{}

func (fakeSyncService) Validate(context.Context, configmodel.AppSetupConfig, string) runtime.ValidationOutcome {
	return runtime.ValidationOutcome{Success: true, SummaryMessage: "ok", DetailReason: "communication_ok", FollowUpNote: "No Ghostfolio data was stored locally."}
}

func TestTokenInputConsumesPlainCharactersWithoutTriggeringActions(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var model = flow.NewModel(flow.Dependencies{
		Options:     bootstrap.DefaultOptions(),
		Startup:     bootstrap.StartupState{ActiveConfig: &config},
		ConfigStore: configstore.NewJSONStore(t.TempDir()),
		SyncService: fakeSyncService{},
	})

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = runCmd(cmd)
	model = updated.(*flow.Model)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	model = updated.(*flow.Model)

	if model.ActiveScreen() != "sync_validation" {
		t.Fatalf("expected to remain on sync validation screen")
	}
	if view := model.View().Content; !contains(view, "*") {
		t.Fatalf("expected masked token input after typing, got %q", view)
	}
}
