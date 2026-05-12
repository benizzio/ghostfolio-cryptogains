package screen

import (
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

func TestSyncValidationScreenViewCoversBusyBranch(t *testing.T) {
	t.Parallel()

	var content = SyncValidationScreenView(SyncValidationScreenParams{Theme: component.DefaultTheme(), Width: 80, Height: 24, Busy: true, BusyText: "Working", SpinnerFrame: "*", TokenInput: "***"})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
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
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
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
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
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
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
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
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
	}
}

func TestValidationResultScreenViewCoversFailureBranch(t *testing.T) {
	t.Parallel()

	var content = ValidationResultScreenView(ValidationResultScreenParams{Theme: component.DefaultTheme(), Width: 80, Height: 24, Outcome: runtime.ValidationOutcome{Success: false, FailureReason: runtime.ValidationFailureTimeout}, MenuItems: []component.MenuItem{{Label: "Back", Enabled: true}}})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
}

// TestValidationResultScreenViewCoversSuccessBranch exercises the successful
// validation render path.
// Authored by: OpenCode
func TestValidationResultScreenViewCoversSuccessBranch(t *testing.T) {
	t.Parallel()

	var content = ValidationResultScreenView(ValidationResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Main Menu", Enabled: true}},
		SelectedIndex: 0,
		Outcome:       runtime.ValidationOutcome{Success: true},
		HelpText:      "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
	if !strings.Contains(content, "Success") {
		t.Fatalf("expected success status in rendered content, got %q", content)
	}
	if !strings.Contains(content, "Communication with the selected Ghostfolio server is working.") {
		t.Fatalf("expected success summary text, got %q", content)
	}
	if !strings.Contains(content, "No Ghostfolio data was stored locally, and reporting is not available") || !strings.Contains(content, "in this slice.") {
		t.Fatalf("expected success follow-up text, got %q", content)
	}
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
	}
}

// TestValidationResultScreenViewCoversIncompatibleContractBranch exercises the
// unsupported-server guidance branch.
// Authored by: OpenCode
func TestValidationResultScreenViewCoversIncompatibleContractBranch(t *testing.T) {
	t.Parallel()

	var content = ValidationResultScreenView(ValidationResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Validate Again", Enabled: true}, {Label: "Main Menu", Enabled: true}},
		SelectedIndex: 0,
		Outcome: runtime.ValidationOutcome{
			Success:       false,
			FailureReason: runtime.ValidationFailureIncompatibleServerContract,
		},
		HelpText: "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
	if !strings.Contains(content, "Failure Category: incompatible server contract") {
		t.Fatalf("expected incompatible-contract failure status, got %q", content)
	}
	if !strings.Contains(content, "The selected server responded, but it did not satisfy the supported") || !strings.Contains(content, "contract for this slice.") {
		t.Fatalf("expected incompatible-contract guidance, got %q", content)
	}
	if strings.Contains(content, "Validate again or return to the main menu. No Ghostfolio data was stored locally.") {
		t.Fatalf("expected special incompatible-contract guidance instead of default failure guidance, got %q", content)
	}
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
	}
}
