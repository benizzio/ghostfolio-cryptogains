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
		Theme:               component.DefaultTheme(),
		Width:               80,
		Height:              24,
		MenuItems:           []component.MenuItem{{Label: "Sync Data", Enabled: true}},
		SelectedIndex:       0,
		ServerOrigin:        "https://ghostfol.io",
		ProtectedDataExists: true,
		HelpText:            "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
	}
}

func TestServerReplacementScreenViewCoversRenderPath(t *testing.T) {
	t.Parallel()

	var content = ServerReplacementScreenView(ServerReplacementScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Continue And Replace", Enabled: true}, {Label: "Cancel", Enabled: true}},
		SelectedIndex: 0,
		CurrentServer: "https://old.example",
		NewServer:     "https://new.example",
		HelpText:      "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
	if !strings.Contains(content, "replace the current protected data tied to that token") || !strings.Contains(content, "and server only after the replacement sync completes successfully") {
		t.Fatalf("expected replacement warning text, got %q", content)
	}
}

func TestSyncValidationScreenViewCoversIdleBranch(t *testing.T) {
	t.Parallel()

	var content = SyncValidationScreenView(SyncValidationScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Start Sync", Enabled: true}, {Label: "Back", Enabled: true}},
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

func TestValidationResultScreenViewCoversDiagnosticBranches(t *testing.T) {
	t.Parallel()

	var promptContent = ValidationResultScreenView(ValidationResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     80,
		Height:    24,
		MenuItems: []component.MenuItem{{Label: "Generate Diagnostic Report", Enabled: true}, {Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome: runtime.ValidationOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true},
		},
	})
	if !strings.Contains(promptContent, "Generate Diagnostic Report") || !strings.Contains(promptContent, "You can generate a synced-data diagnostic report") {
		t.Fatalf("expected diagnostic prompt branch, got %q", promptContent)
	}

	var writtenContent = ValidationResultScreenView(ValidationResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     80,
		Height:    24,
		MenuItems: []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome: runtime.ValidationOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureIncompatibleNewSyncData,
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true, Path: "/tmp/report.diagnostic.json"},
		},
	})
	if !strings.Contains(writtenContent, "/tmp/report.diagnostic.json") {
		t.Fatalf("expected generated-report path disclosure, got %q", writtenContent)
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
	if !strings.Contains(content, "Activity data was stored securely for future use.") {
		t.Fatalf("expected success summary text, got %q", content)
	}
	if !strings.Contains(content, "No report-generation") || !strings.Contains(content, "cached-data browsing workflow") || !strings.Contains(content, "in this slice.") {
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
	if strings.Contains(content, "Sync again or return to the main menu. No protected activity data was stored.") {
		t.Fatalf("expected special incompatible-contract guidance instead of default failure guidance, got %q", content)
	}
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
	}
}
