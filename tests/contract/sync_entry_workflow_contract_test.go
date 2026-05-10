package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestSyncEntryWorkflowContract(t *testing.T) {
	t.Parallel()

	var content = screen.SyncValidationScreenView(screen.SyncValidationScreenParams{
		Theme:      component.DefaultTheme(),
		Width:      100,
		Height:     32,
		TokenInput: "******",
		MenuItems:  []component.MenuItem{{Label: "Validate Communication", Enabled: true}, {Label: "Back", Enabled: true}},
	})

	assertContains(t, content, "Ghostfolio Security Token")
	assertContains(t, content, "Validate Communication")
	assertContains(t, content, "Back")
	assertContains(t, content, "communication only")
}
