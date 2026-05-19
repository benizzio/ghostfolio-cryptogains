package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestSyncEntryWorkflowContract(t *testing.T) {
	t.Parallel()

	var content = screen.SyncEntryScreenView(screen.SyncEntryScreenParams{
		Theme:      component.DefaultTheme(),
		Width:      100,
		Height:     32,
		TokenInput: "******",
		MenuItems:  []component.MenuItem{{Label: "Start Sync", Enabled: true}, {Label: "Back", Enabled: true}},
	})

	assertContains(t, content, "Ghostfolio Security Token")
	assertContains(t, content, "Start Sync")
	assertContains(t, content, "Back")
	assertContains(t, content, "securely for future use only")
	assertContains(t, content, "ghostfolio-cryptogains")
	assertContains(t, content, "[Ghostfolio]")
}
