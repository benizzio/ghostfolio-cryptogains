package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

// TestMainMenuWorkflowContract verifies the visible main-menu contract for the
// first Sync and Reports slice.
// Authored by: OpenCode
func TestMainMenuWorkflowContract(t *testing.T) {
	t.Parallel()

	var content = screen.MainMenuScreenView(screen.MainMenuScreenParams{
		Theme:        component.DefaultTheme(),
		Width:        100,
		Height:       32,
		ServerOrigin: "https://ghostfol.io",
		MenuItems:    []component.MenuItem{{Label: "Sync and Reports", Enabled: true}},
	})

	assertContains(t, content, "Sync and Reports")
	assertContains(t, content, "Selected Server")
	assertContains(t, content, "ghostfolio-cryptogains")
	assertContains(t, content, "[Ghostfolio]")
	assertNotContains(t, content, "Protected Data:")
	assertNotContains(t, content, "Last Successful Sync")
	assertNotContains(t, content, "Available Report Years")
	assertNotContains(t, content, "Sync Data")
}
