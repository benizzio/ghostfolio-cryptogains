package screen

import (
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
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
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MenuItems:     []component.MenuItem{{Label: "Sync and Reports", Enabled: true}},
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
	if !strings.Contains(content, "Sync and Reports") {
		t.Fatalf("expected Sync and Reports menu label, got %q", content)
	}
	if strings.Contains(content, "Protected Data:") || strings.Contains(content, "Last Successful Sync") || strings.Contains(content, "Available Report Years") {
		t.Fatalf("expected no protected metadata on the main menu before unlock, got %q", content)
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
		Theme:                   component.DefaultTheme(),
		Width:                   80,
		Height:                  24,
		ScreenTitle:             "Sync Data",
		ScreenSubtitle:          "Retrieve, validate, and securely store supported activity history.",
		IntroText:               "The application will authenticate, retrieve activity history, validate it, and store it securely for future use only.",
		IdleStatusText:          "Enter the Ghostfolio security token only when starting Sync Data.",
		ShowProtectedDataStatus: true,
		MenuItems:               []component.MenuItem{{Label: "Start Sync", Enabled: true}, {Label: "Back", Enabled: true}},
		SelectedIndex:           0,
		TokenInput:              "***",
		HelpText:                "help",
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
		IntroText:         "The application will authenticate, retrieve activity history, validate it, and store it securely for future use only.",
		IdleStatusText:    "Enter the Ghostfolio security token only when starting Sync Data.",
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

	var busyContent = SyncResultScreenView(SyncResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		Busy:          true,
		StatusMessage: "Generating diagnostic report...",
		MenuItems:     []component.MenuItem{{Label: "Generate Diagnostic Report", Enabled: false}, {Label: "Sync Again", Enabled: false}, {Label: "Back To Main Menu", Enabled: false}},
		Outcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true},
		},
	})
	if !strings.Contains(busyContent, "Generating a local synced-data diagnostic report") || !strings.Contains(busyContent, "Generating diagnostic report...") {
		t.Fatalf("expected busy diagnostic-report branch, got %q", busyContent)
	}
}

func TestReportScreenViewsCoverSelectionBusyAndResultBranches(t *testing.T) {
	t.Parallel()

	var selection = ReportSelectionScreenView(ReportSelectionScreenParams{
		Theme:             component.DefaultTheme(),
		Width:             80,
		Height:            24,
		AvailableYears:    []int{2024, 2025},
		SelectedYearIndex: 0,
		MethodItems:       []component.MenuItem{{Label: "FIFO", Enabled: true}, {Label: "LIFO", Enabled: true}, {Label: "HIFO", Enabled: true}, {Label: "Average Cost Basis", Enabled: true}, {Label: "Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order", Enabled: true}},
		SelectedMethod:    0,
		MethodExplanation: reportmodel.CostBasisMethodFIFO.Explanation(),
		MenuItems:         []component.MenuItem{{Label: "Generate Report", Enabled: true}, {Label: "Back", Enabled: true}},
		SelectedAction:    0,
		HelpText:          "help",
	})
	if !strings.Contains(selection, "Generate Capital Gains Report") || !strings.Contains(selection, "2024") || !strings.Contains(selection, "FIFO") || !strings.Contains(selection, reportmodel.CostBasisMethodFIFO.Explanation()) || !strings.Contains(selection, "Generate Report") {
		t.Fatalf("expected report selection content, got %q", selection)
	}

	var busy = ReportBusyScreenView(ReportBusyScreenParams{
		Theme:        component.DefaultTheme(),
		Width:        80,
		Height:       24,
		SelectedYear: 2024,
		MethodLabel:  "FIFO",
		BusyText:     "Generating capital gains report...",
		SpinnerFrame: "*",
		HelpText:     "help",
	})
	if !strings.Contains(busy, "Generating capital gains report") || !strings.Contains(busy, "Selected Year: 2024") || !strings.Contains(busy, "Cost Basis Method: FIFO") {
		t.Fatalf("expected report busy content, got %q", busy)
	}

	request, err := reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	outputFile, err := reportmodel.NewReportOutputFile("/tmp/Documents", "ghostfolio-capital-gains-2024-fifo.md", "/tmp/report.md", time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC), true, "")
	if err != nil {
		t.Fatalf("new report output file: %v", err)
	}

	var result = ReportResultScreenView(ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MethodLabel:   "FIFO",
		MenuItems:     []component.MenuItem{{Label: "Back To Sync and Reports", Enabled: true}, {Label: "Generate Another Report", Enabled: true}},
		SelectedIndex: 0,
		HelpText:      "help",
		Outcome: runtime.ReportOutcome{
			Success:    true,
			Message:    "Saved the report to \"/tmp/report.md\" and requested automatic opening.",
			Request:    request,
			OutputFile: outputFile,
		},
	})
	if !strings.Contains(result, "Saved Markdown Path: /tmp/report.md") || !strings.Contains(result, "Back To Sync and Reports") || !strings.Contains(result, "Generate Another Report") {
		t.Fatalf("expected report result content, got %q", result)
	}
}

// TestSyncFollowUpTextCoversRemainingFailureBranches verifies the
// explicit follow-up guidance branches that are not covered by the render tests.
// Authored by: OpenCode
func TestSyncFollowUpTextCoversRemainingFailureBranches(t *testing.T) {
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
			if got := syncFollowUpText(testCase.outcome, false); !strings.Contains(got, testCase.want) {
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
	if !strings.Contains(content, "Return to Sync and Reports") || !strings.Contains(content, "generate a capital gains report") || !strings.Contains(content, "newly stored protected data") {
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

// TestReportScreenHelperBranches verifies report-result warning and failure
// render variants plus direct helper fallbacks.
// Authored by: OpenCode
func TestReportScreenHelperBranches(t *testing.T) {
	t.Parallel()

	var request, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	var outputFile, outputErr = reportmodel.NewReportOutputFile("/tmp/Documents", "ghostfolio-capital-gains-2024-fifo.md", "/tmp/report.md", time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC), true, "open boom")
	if outputErr != nil {
		t.Fatalf("new report output file: %v", outputErr)
	}

	var warning = ReportResultScreenView(ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MethodLabel:   "FIFO",
		MenuItems:     []component.MenuItem{{Label: "Back To Sync and Reports", Enabled: true}},
		SelectedIndex: 0,
		HelpText:      "help",
		Outcome: runtime.ReportOutcome{
			Success:       true,
			FailureReason: runtime.ReportFailureAutomaticOpenFailedAfterSave,
			Message:       "Saved the report to \"/tmp/report.md\", but automatic opening failed.",
			Request:       request,
			OutputFile:    outputFile,
		},
	})
	if !strings.Contains(warning, "Success With Warning: automatic open failed after save") {
		t.Fatalf("expected report warning status, got %q", warning)
	}

	var failure = ReportResultScreenView(ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MethodLabel:   "FIFO",
		MenuItems:     []component.MenuItem{{Label: "Back To Sync and Reports", Enabled: true}},
		SelectedIndex: 0,
		HelpText:      "help",
		Outcome: runtime.ReportOutcome{
			Success:       false,
			FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
			Message:       "Could not generate the report.",
			Request:       request,
		},
	})
	if !strings.Contains(failure, "Failure Category: unsupported report calculation") {
		t.Fatalf("expected report failure status, got %q", failure)
	}

	if got := reportMethodExplanation("   "); !strings.Contains(got, reportmodel.CostBasisMethod("").Explanation()) {
		t.Fatalf("expected empty explanation to fall back, got %q", got)
	}
	if got := reportResultSummary(runtime.ReportOutcome{Success: true, Message: "saved without path"}); got != "saved without path" {
		t.Fatalf("expected success without path to return plain message, got %q", got)
	}
	if got := reportResultSummary(runtime.ReportOutcome{Success: false, Message: "failure detail"}); got != "failure detail" {
		t.Fatalf("expected failure summary to return failure message, got %q", got)
	}
}
