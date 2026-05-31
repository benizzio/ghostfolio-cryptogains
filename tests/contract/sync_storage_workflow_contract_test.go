package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestSyncStorageWorkflowContract(t *testing.T) {
	t.Parallel()

	var busy = screen.SyncEntryScreenView(screen.SyncEntryScreenParams{
		Theme:        component.DefaultTheme(),
		Width:        100,
		Height:       32,
		TokenInput:   "******",
		Busy:         true,
		BusyText:     "Syncing and storing activity history...",
		SpinnerFrame: "*",
	})
	assertContains(t, busy, "securely for future use only")
	assertContains(t, busy, "Syncing and storing activity history")

	var success = screen.SyncResultScreenView(screen.SyncResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     100,
		Height:    32,
		MenuItems: []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome:   runtime.SyncOutcome{Success: true, DetailReason: "activity_data_stored"},
	})
	assertContains(t, success, "stored securely for future use")
	assertContains(t, success, "Return to Sync and Reports to generate a capital gains report")
	assertNotContains(t, success, "No report-generation, report-preview")

	var failure = screen.SyncReportsScreenView(screen.SyncReportsScreenParams{
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
			DetailReason:  string(runtime.SyncFailureUnsupportedActivityHistory),
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true},
		},
		UnavailableMessage: "no synced data available",
	})
	assertContains(t, failure, "Failure Category: unsupported activity history")
	assertContains(t, failure, "Generate Diagnostic Report")
	assertContains(t, failure, "Generate Diagnostic Report is available for this failure from this context.")
}
