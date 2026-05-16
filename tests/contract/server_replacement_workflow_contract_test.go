package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestServerReplacementWorkflowContract(t *testing.T) {
	t.Parallel()

	content := screen.ServerReplacementScreenView(screen.ServerReplacementScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		MenuItems:     []component.MenuItem{{Label: "Continue And Replace", Enabled: true}, {Label: "Cancel", Enabled: true}},
		SelectedIndex: 0,
		CurrentServer: "https://old.example",
		NewServer:     "https://new.example",
		HelpText:      "help",
	})

	assertContains(t, content, "Continue And Replace")
	assertContains(t, content, "Cancel")
	assertContains(t, content, "replace the current protected data tied to that token and server")
	assertContains(t, content, "https://old.example")
	assertContains(t, content, "https://new.example")

	resultContent := screen.SyncResultScreenView(screen.SyncResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     100,
		Height:    32,
		MenuItems: []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureServerReplacementCancelled,
		},
	})
	assertContains(t, resultContent, "server replacement cancelled")
	assertContains(t, resultContent, "The existing protected data was left unchanged")
}
