package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestSyncStorageWorkflowContract(t *testing.T) {
	t.Parallel()

	var busy = screen.SyncValidationScreenView(screen.SyncValidationScreenParams{
		Theme:        component.DefaultTheme(),
		Width:        100,
		Height:       32,
		TokenInput:   "******",
		Busy:         true,
		BusyText:     "Syncing and storing activity history...",
		SpinnerFrame: "*",
	})
	assertContains(t, busy, "securely for future use only")
	assertContains(t, busy, "Syncing and storing activity history")

	var success = screen.ValidationResultScreenView(screen.ValidationResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     100,
		Height:    32,
		MenuItems: []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome:   runtime.ValidationOutcome{Success: true, DetailReason: "activity_data_stored"},
	})
	assertContains(t, success, "stored securely for future use")
	assertContains(t, success, "cached-data browsing workflow")

	var failure = screen.ValidationResultScreenView(screen.ValidationResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     100,
		Height:    32,
		MenuItems: []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome: runtime.ValidationOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
			DetailReason:  string(runtime.SyncFailureUnsupportedActivityHistory),
		},
	})
	assertContains(t, failure, "Failure Category: unsupported activity history")
}
