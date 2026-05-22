// Package screen verifies screen-local render helpers.
// Authored by: OpenCode
package screen

import (
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// TestSyncEntryScreenViewHidesTokenInputInContextMode verifies the in-context
// `Sync Data` renderer contract that reuses the unlocked token without showing
// token input again.
// Authored by: OpenCode
func TestSyncEntryScreenViewHidesTokenInputInContextMode(t *testing.T) {
	t.Parallel()

	var content = SyncEntryScreenView(SyncEntryScreenParams{
		Theme:                   component.DefaultTheme(),
		Width:                   100,
		Height:                  32,
		ScreenTitle:             "Sync Data",
		ScreenSubtitle:          "Retrieve, validate, and securely store supported activity history.",
		UseContextToken:         true,
		ShowProtectedDataStatus: true,
		MenuItems:               []component.MenuItem{{Label: "Start Sync", Enabled: true}, {Label: "Back", Enabled: true}},
		SelectedIndex:           0,
		HelpText:                "help",
	})

	if strings.Contains(content, "Ghostfolio Security Token") {
		t.Fatalf("expected in-context sync view to hide token label, got %q", content)
	}
	if strings.Contains(content, "Enter Ghostfolio security token") {
		t.Fatalf("expected in-context sync view to hide token placeholder, got %q", content)
	}
	if strings.Contains(content, "existing Sync and Reports context token") {
		t.Fatalf("expected in-context sync view to omit redundant explanation, got %q", content)
	}
	if !strings.Contains(content, "Start Sync to obtain current available activity data on the Ghostfolio") || !strings.Contains(content, "server.") {
		t.Fatalf("expected in-context sync status text, got %q", content)
	}
	if !strings.Contains(content, "Start Sync") || !strings.Contains(content, "Back") {
		t.Fatalf("expected in-context sync actions, got %q", content)
	}
}
