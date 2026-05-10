package component

import (
	"strings"
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

	var got = RenderScreen(DefaultTheme(), 0, 0, "Title", "Subtitle", "Body", "Status", "Footer")
	if got == "" {
		t.Fatalf("expected rendered screen")
	}
	if !strings.Contains(got, ApplicationIdentityName) || !strings.Contains(got, ApplicationIdentityCue) {
		t.Fatalf("expected persistent application identity header, got %q", got)
	}
}

func TestRenderScreenClampsNarrowPositiveWidth(t *testing.T) {
	t.Parallel()

	var got = RenderScreen(DefaultTheme(), 3, 10, "Title", "Subtitle", "Body", "Status", "Footer")
	if got == "" {
		t.Fatalf("expected rendered screen")
	}
	if !strings.Contains(got, ApplicationIdentityName) {
		t.Fatalf("expected persistent application identity header, got %q", got)
	}
}

func quitBindingForTest() key.Binding {
	return key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit"))
}
