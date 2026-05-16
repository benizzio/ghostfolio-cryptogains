package screen

import (
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

func TestSyncEntryScreenViewCoversBusyBranch(t *testing.T) {
	t.Parallel()

	var content = SyncEntryScreenView(SyncEntryScreenParams{Theme: component.DefaultTheme(), Width: 80, Height: 24, Busy: true, BusyText: "Working", SpinnerFrame: "*", TokenInput: "***"})
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

func TestSyncEntryScreenViewCoversIdleBranch(t *testing.T) {
	t.Parallel()

	var content = SyncEntryScreenView(SyncEntryScreenParams{
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

func TestSyncEntryScreenViewUsesValidationMessageOverride(t *testing.T) {
	t.Parallel()

	var content = SyncEntryScreenView(SyncEntryScreenParams{
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

// TestSyncProtectedDataStatusLabelCoversBothBranches verifies the visible sync
// protected-data status helper.
// Authored by: OpenCode
func TestSyncProtectedDataStatusLabelCoversBothBranches(t *testing.T) {
	t.Parallel()

	if got := syncProtectedDataStatusLabel(true); got != "yes" {
		t.Fatalf("expected yes label, got %q", got)
	}
	if got := syncProtectedDataStatusLabel(false); got != "no" {
		t.Fatalf("expected no label, got %q", got)
	}
	if got := protectedDataStatusLabel(false); got != "none loaded for this run" {
		t.Fatalf("expected no protected-data label, got %q", got)
	}
}

func TestSyncResultScreenViewCoversFailureBranch(t *testing.T) {
	t.Parallel()

	var content = SyncResultScreenView(SyncResultScreenParams{Theme: component.DefaultTheme(), Width: 80, Height: 24, Outcome: runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureTimeout}, MenuItems: []component.MenuItem{{Label: "Back", Enabled: true}}})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
}

func TestSyncResultScreenViewCoversDiagnosticBranches(t *testing.T) {
	t.Parallel()

	var promptContent = SyncResultScreenView(SyncResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     80,
		Height:    24,
		MenuItems: []component.MenuItem{{Label: "Generate Diagnostic Report", Enabled: true}, {Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true},
		},
	})
	if !strings.Contains(promptContent, "Generate Diagnostic Report") || !strings.Contains(promptContent, "You can generate a synced-data diagnostic report") {
		t.Fatalf("expected diagnostic prompt branch, got %q", promptContent)
	}

	var writtenContent = SyncResultScreenView(SyncResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     80,
		Height:    24,
		MenuItems: []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureIncompatibleNewSyncData,
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true, Path: "/tmp/report.diagnostic.json"},
		},
	})
	if !strings.Contains(writtenContent, "/tmp/report.diagnostic.json") {
		t.Fatalf("expected generated-report path disclosure, got %q", writtenContent)
	}
}

// TestValidationFollowUpTextCoversRemainingFailureBranches verifies the
// explicit follow-up guidance branches that are not covered by the render tests.
// Authored by: OpenCode
func TestValidationFollowUpTextCoversRemainingFailureBranches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		outcome runtime.SyncOutcome
		want    string
	}{
		{name: "replacement cancelled", outcome: runtime.SyncOutcome{FailureReason: runtime.SyncFailureServerReplacementCancelled}, want: "server replacement was cancelled"},
		{name: "rejected token", outcome: runtime.SyncOutcome{FailureReason: runtime.SyncFailureRejectedToken}, want: "token was rejected"},
		{name: "unsupported stored-data version", outcome: runtime.SyncOutcome{FailureReason: runtime.SyncFailureUnsupportedStoredDataVersion}, want: "unsupported stored-data version"},
		{name: "incompatible new sync data", outcome: runtime.SyncOutcome{FailureReason: runtime.SyncFailureIncompatibleNewSyncData}, want: "could not be stored safely"},
		{name: "unsupported activity history", outcome: runtime.SyncOutcome{FailureReason: runtime.SyncFailureUnsupportedActivityHistory}, want: "activity history is not supported safely"},
		{name: "default failure", outcome: runtime.SyncOutcome{FailureReason: runtime.SyncFailureTimeout}, want: "Sync again or return to the main menu"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := validationFollowUpText(testCase.outcome); !strings.Contains(got, testCase.want) {
				t.Fatalf("expected follow-up text %q to contain %q", got, testCase.want)
			}
		})
	}
}

// TestSyncResultScreenViewCoversSuccessBranch exercises the successful
// sync render path.
// Authored by: OpenCode
func TestSyncResultScreenViewCoversSuccessBranch(t *testing.T) {
	t.Parallel()

	var content = SyncResultScreenView(SyncResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Main Menu", Enabled: true}},
		SelectedIndex: 0,
		Outcome:       runtime.SyncOutcome{Success: true},
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

// TestSyncResultScreenViewCoversIncompatibleContractBranch exercises the
// unsupported-server guidance branch.
// Authored by: OpenCode
func TestSyncResultScreenViewCoversIncompatibleContractBranch(t *testing.T) {
	t.Parallel()

	var content = SyncResultScreenView(SyncResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		SelectedIndex: 0,
		Outcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureIncompatibleServerContract,
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
