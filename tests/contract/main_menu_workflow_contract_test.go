package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestMainMenuWorkflowContract(t *testing.T) {
	t.Parallel()

	var content = screen.MainMenuScreenView(screen.MainMenuScreenParams{
		Theme:        component.DefaultTheme(),
		Width:        100,
		Height:       32,
		ServerOrigin: "https://ghostfol.io",
		MenuItems:    []component.MenuItem{{Label: "Sync Data", Enabled: true}},
	})

	assertContains(t, content, "Sync Data")
	assertContains(t, content, "Selected Server")
	assertNotContains(t, content, "Report")
}
