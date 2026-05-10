package screen

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

func TestSyncValidationScreenViewCoversBusyBranch(t *testing.T) {
	t.Parallel()

	var content = SyncValidationScreenView(SyncValidationScreenParams{Theme: component.DefaultTheme(), Width: 80, Height: 24, Busy: true, BusyText: "Working", SpinnerFrame: "*", TokenInput: "***"})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
}

func TestSetupScreenViewCoversVisibleBranches(t *testing.T) {
	t.Parallel()

	var content = SetupScreenView(SetupScreenParams{
		Theme:               component.DefaultTheme(),
		Width:               80,
		Height:              24,
		MenuItems:           []component.MenuItem{{Label: "Use Ghostfolio Cloud", Enabled: true}, {Label: "Use Custom Server", Enabled: true}, {Label: "Save And Continue", Enabled: false}},
		SelectedIndex:       1,
		ShowOriginInput:     true,
		OriginInput:         "http://localhost:8080",
		InvalidSetupMessage: "invalid remembered setup",
		ValidationMessage:   "validation error",
		HelpText:            "help",
		CanSave:             false,
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
}

func TestMainMenuScreenViewCoversRenderPath(t *testing.T) {
	t.Parallel()

	var content = MainMenuScreenView(MainMenuScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Sync Data", Enabled: true}},
		SelectedIndex: 0,
		ServerOrigin:  "https://ghostfol.io",
		HelpText:      "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
}

func TestSyncValidationScreenViewCoversIdleBranch(t *testing.T) {
	t.Parallel()

	var content = SyncValidationScreenView(SyncValidationScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Validate Communication", Enabled: true}, {Label: "Back", Enabled: true}},
		SelectedIndex: 0,
		TokenInput:    "***",
		HelpText:      "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
}

func TestSyncValidationScreenViewUsesValidationMessageOverride(t *testing.T) {
	t.Parallel()

	var content = SyncValidationScreenView(SyncValidationScreenParams{
		Theme:             component.DefaultTheme(),
		Width:             80,
		Height:            24,
		TokenInput:        "***",
		ValidationMessage: "validation failed",
		HelpText:          "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
}

func TestValidationResultScreenViewCoversFailureBranch(t *testing.T) {
	t.Parallel()

	var content = ValidationResultScreenView(ValidationResultScreenParams{Theme: component.DefaultTheme(), Width: 80, Height: 24, Outcome: runtime.ValidationOutcome{Success: false, FailureCategory: ghostfolioclient.FailureTimeout}, MenuItems: []component.MenuItem{{Label: "Back", Enabled: true}}})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
}
