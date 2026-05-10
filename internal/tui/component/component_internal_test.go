package component

import (
	"testing"

	"charm.land/bubbles/v2/key"
)

func TestBindingsExposeShortAndFullHelp(t *testing.T) {
	t.Parallel()

	var bindings = Bindings{Short: []key.Binding{quitBindingForTest()}, Full: [][]key.Binding{{quitBindingForTest()}}}
	if len(bindings.ShortHelp()) != 1 || len(bindings.FullHelp()) != 1 {
		t.Fatalf("unexpected bindings")
	}
}

func TestRenderHelpUsesWidthFallback(t *testing.T) {
	t.Parallel()

	if got := RenderHelp(0, Bindings{Short: []key.Binding{quitBindingForTest()}}); got == "" {
		t.Fatalf("expected rendered help")
	}
}

func TestRenderMenuCoversSelectedDisabledAndDescriptions(t *testing.T) {
	t.Parallel()

	var menu = RenderMenu(DefaultTheme(), []MenuItem{{Label: "One", Enabled: true, Description: "desc"}, {Label: "Two", Enabled: false}}, 0)
	if menu == "" {
		t.Fatalf("expected rendered menu")
	}
}

func TestRenderScreenUsesDefaultDimensions(t *testing.T) {
	t.Parallel()

	if got := RenderScreen(DefaultTheme(), 0, 0, "Title", "Subtitle", "Body", "Status", "Footer"); got == "" {
		t.Fatalf("expected rendered screen")
	}
}

func quitBindingForTest() key.Binding {
	return key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit"))
}
