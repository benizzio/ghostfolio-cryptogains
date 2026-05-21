// Package screen verifies screen-local render helpers.
// Authored by: OpenCode
package screen

import (
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// TestSyncReportsUnlockScreenViewCoversUnlockRenderState verifies the unlock
// screen copy used before the context becomes active.
// Authored by: OpenCode
func TestSyncReportsUnlockScreenViewCoversUnlockRenderState(t *testing.T) {
	t.Parallel()

	var content = SyncEntryScreenView(SyncEntryScreenParams{
		Theme:                   component.DefaultTheme(),
		Width:                   80,
		Height:                  24,
		ScreenTitle:             "Sync and Reports",
		ScreenSubtitle:          "Unlock the active sync and reporting context.",
		IntroText:               "Enter the Ghostfolio security token once to unlock Sync Data and future reporting actions for this run.",
		IdleStatusText:          "Enter the Ghostfolio security token to unlock Sync and Reports for this run.",
		ShowProtectedDataStatus: false,
		MenuItems:               []component.MenuItem{{Label: "Unlock", Enabled: true}, {Label: "Back", Enabled: true}},
		SelectedIndex:           0,
		TokenInput:              "***",
		HelpText:                "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
	if !strings.Contains(content, "Sync and Reports") || !strings.Contains(content, "Unlock") {
		t.Fatalf("expected unlock-screen labels, got %q", content)
	}
	if strings.Contains(content, "Protected Data:") || strings.Contains(content, "Last Successful Sync") {
		t.Fatalf("expected unlock screen to hide protected-data readiness, got %q", content)
	}
}

// TestSyncReportsScreenViewCoversNoSyncedDataBranch verifies the visible
// renderer shape for the initial no-data Sync and Reports state.
// Authored by: OpenCode
func TestSyncReportsScreenViewCoversNoSyncedDataBranch(t *testing.T) {
	t.Parallel()

	var content = SyncReportsScreenView(SyncReportsScreenParams{
		Theme:              component.DefaultTheme(),
		Width:              80,
		Height:             24,
		ServerOrigin:       "https://ghostfol.io",
		SelectedIndex:      0,
		MenuItems:          []component.MenuItem{{Label: "Sync Data", Enabled: true}, {Label: "Generate Capital Gains Report", Enabled: false}, {Label: "Back To Main Menu", Enabled: true}},
		UnavailableMessage: "no synced data available",
		HelpText:           "help",
	})
	if content == "" {
		t.Fatalf("expected rendered content")
	}
	if !strings.Contains(content, "ghostfolio-cryptogains") || !strings.Contains(content, "[Ghostfolio]") {
		t.Fatalf("expected persistent application identity header, got %q", content)
	}
	if !strings.Contains(content, "Sync Data") || !strings.Contains(content, "Generate Capital Gains Report") {
		t.Fatalf("expected Sync and Reports actions, got %q", content)
	}
	if !strings.Contains(content, "Back To Main Menu") {
		t.Fatalf("expected context exit action, got %q", content)
	}
	if !strings.Contains(content, "no synced data available") {
		t.Fatalf("expected no-data readiness message, got %q", content)
	}
}

// TestSyncReportsScreenViewCoversReportableDataBranch verifies readiness details for readable protected data.
// Authored by: OpenCode
func TestSyncReportsScreenViewCoversReportableDataBranch(t *testing.T) {
	t.Parallel()

	var content = SyncReportsScreenView(SyncReportsScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		ServerOrigin:  "https://ghostfol.io",
		SelectedIndex: 1,
		MenuItems:     []component.MenuItem{{Label: "Sync Data", Enabled: true}, {Label: "Generate Capital Gains Report", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		ProtectedDataState: runtime.ProtectedDataState{
			HasReadableSnapshot:  true,
			ActivityCount:        3,
			LastSuccessfulSyncAt: time.Date(2026, time.May, 20, 13, 30, 0, 0, time.UTC),
			AvailableReportYears: []int{2024, 2025},
		},
		HelpText: "help",
	})
	if !strings.Contains(content, "Protected Activity Count: 3") {
		t.Fatalf("expected activity count, got %q", content)
	}
	if !strings.Contains(content, "Available Report Years: 2024, 2025") {
		t.Fatalf("expected available report years, got %q", content)
	}
	if !strings.Contains(content, "Sync Data: last successful sync") {
		t.Fatalf("expected last successful sync summary, got %q", content)
	}
	if !strings.Contains(content, "Generate Capital Gains Report: available") {
		t.Fatalf("expected available report action, got %q", content)
	}
}

// TestSyncReportsScreenViewCoversDiagnosticPromptAndWrittenPath verifies the
// context-based synced-data diagnostic prompt and written-path disclosure.
// Authored by: OpenCode
func TestSyncReportsScreenViewCoversDiagnosticPromptAndWrittenPath(t *testing.T) {
	t.Parallel()

	var prompt = SyncReportsScreenView(SyncReportsScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		ServerOrigin:  "https://ghostfol.io",
		SelectedIndex: 2,
		MenuItems: []component.MenuItem{
			{Label: "Sync Data", Enabled: true},
			{Label: "Generate Capital Gains Report", Enabled: false},
			{Label: "Generate Diagnostic Report", Enabled: true},
			{Label: "Back To Main Menu", Enabled: true},
		},
		SyncOutcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true},
		},
		UnavailableMessage: "no synced data available",
		HelpText:           "help",
	})
	if !strings.Contains(prompt, "Failure Category: unsupported activity history") {
		t.Fatalf("expected latest sync failure category, got %q", prompt)
	}
	if !strings.Contains(prompt, "Generate Diagnostic Report is available for this failure from this context.") {
		t.Fatalf("expected diagnostic prompt text, got %q", prompt)
	}

	var written = SyncReportsScreenView(SyncReportsScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		ServerOrigin:  "https://ghostfol.io",
		SelectedIndex: 0,
		MenuItems: []component.MenuItem{
			{Label: "Sync Data", Enabled: true},
			{Label: "Generate Capital Gains Report", Enabled: false},
			{Label: "Back To Main Menu", Enabled: true},
		},
		SyncOutcome: runtime.SyncOutcome{
			Success:       false,
			FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true, Path: "/tmp/example.diagnostic.json"},
		},
		StatusMessage:      "Diagnostic report generated successfully.",
		UnavailableMessage: "no synced data available",
		HelpText:           "help",
	})
	if !strings.Contains(written, "/tmp/example.diagnostic.json") {
		t.Fatalf("expected written diagnostic path, got %q", written)
	}
	if strings.Contains(written, "Generate Diagnostic Report") && strings.Contains(written, "> Generate Diagnostic Report") {
		t.Fatalf("expected written-path state to clear the extra diagnostic action, got %q", written)
	}
}

// TestSyncReportsHelperBranches verifies direct readiness, status, and fallback
// helper branches that are awkward to reach through full-screen rendering alone.
// Authored by: OpenCode
func TestSyncReportsHelperBranches(t *testing.T) {
	t.Parallel()

	var state = runtime.ProtectedDataState{HasReadableSnapshot: true, ActivityCount: 2}
	if got := syncReportsReadinessSummary(state, "report generation blocked"); !strings.Contains(got, "Generate Capital Gains Report: unavailable - report generation blocked") {
		t.Fatalf("expected readable state without years to surface unavailable reason, got %q", got)
	}
	if got := syncReportsStatusText(runtime.ProtectedDataState{}, runtime.SyncOutcome{}, false, "", ""); !strings.Contains(got, "Generate Capital Gains Report stays unavailable until no synced data available") {
		t.Fatalf("expected no-synced-data status text, got %q", got)
	}
	if got := syncReportsStatusText(runtime.ProtectedDataState{HasReadableSnapshot: true}, runtime.SyncOutcome{}, false, "", "reportable years become available"); !strings.Contains(got, "report generation stays unavailable until reportable years become available") {
		t.Fatalf("expected no-reportable-years status text, got %q", got)
	}
	if got := syncReportsStatusText(runtime.ProtectedDataState{HasReadableSnapshot: true, AvailableReportYears: []int{2024}}, runtime.SyncOutcome{}, false, "", ""); !strings.Contains(got, "report generation is available from this context") {
		t.Fatalf("expected ready status text, got %q", got)
	}
	if got := syncReportsStatusText(runtime.ProtectedDataState{}, runtime.SyncOutcome{}, true, "", ""); !strings.Contains(got, "Generating a local synced-data diagnostic report") {
		t.Fatalf("expected busy diagnostic-report status text, got %q", got)
	}
	if got := syncReportsUnavailableReason(""); got != "no synced data available" {
		t.Fatalf("expected unavailable-reason fallback, got %q", got)
	}
	if got := formatSyncReportsLastSuccessfulSync(time.Time{}); got != "unknown" {
		t.Fatalf("expected zero sync time fallback, got %q", got)
	}
	if got := syncReportsSyncFeedback(runtime.SyncOutcome{Success: true, Attempt: runtime.SyncAttempt{AttemptID: "attempt-1"}}, false); !strings.Contains(got, "Activity data was stored securely for future use.") {
		t.Fatalf("expected success sync feedback, got %q", got)
	}
	if got := syncReportsSyncFeedback(runtime.SyncOutcome{Success: false, FailureReason: runtime.SyncFailureTimeout}, false); !strings.Contains(got, "Failure Category: timeout") {
		t.Fatalf("expected failure sync feedback, got %q", got)
	}
}
