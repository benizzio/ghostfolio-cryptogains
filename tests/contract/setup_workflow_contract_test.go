package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestSetupWorkflowContract(t *testing.T) {
	t.Parallel()

	var content = screen.SetupScreenView(screen.SetupScreenParams{
		Theme:           component.DefaultTheme(),
		Width:           100,
		Height:          32,
		ShowOriginInput: true,
		CanSave:         false,
		MenuItems:       []component.MenuItem{{Label: "Use Ghostfolio Cloud", Enabled: true}, {Label: "Use Custom Server", Enabled: true}, {Label: "Save And Continue", Enabled: false}},
	})

	assertContains(t, content, "Use Ghostfolio Cloud")
	assertContains(t, content, "Use Custom Server")
	assertContains(t, content, "Save And Continue")
	assertContains(t, content, "Custom Server Origin")
	assertContains(t, content, "Production-like custom origins")
	assertContains(t, content, "ghostfolio-cryptogains")
	assertContains(t, content, "[Ghostfolio]")
}
