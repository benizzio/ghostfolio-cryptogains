// Package contract verifies rendered workflow and Ghostfolio-boundary contracts
// for the sync-and-storage slice.
// Authored by: OpenCode
package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

// TestSyncReportsWorkflowContract verifies the first visible Sync and Reports
// menu contract shape.
// Authored by: OpenCode
func TestSyncReportsWorkflowContract(t *testing.T) {
	t.Parallel()

	var content = screen.SyncReportsScreenView(screen.SyncReportsScreenParams{
		Theme:              component.DefaultTheme(),
		Width:              100,
		Height:             32,
		ServerOrigin:       "https://ghostfol.io",
		SelectedIndex:      0,
		MenuItems:          []component.MenuItem{{Label: "Sync Data", Enabled: true}, {Label: "Generate Capital Gains Report", Enabled: false}, {Label: "Back To Main Menu", Enabled: true}},
		UnavailableMessage: "no synced data available",
	})

	assertContains(t, content, "Sync and Reports")
	assertContains(t, content, "Selected Server")
	assertContains(t, content, "Sync Data")
	assertContains(t, content, "Generate Capital Gains Report")
	assertContains(t, content, "no synced data available")
	assertContains(t, content, "Back To Main Menu")
	assertContains(t, content, "ghostfolio-cryptogains")
	assertContains(t, content, "[Ghostfolio]")

	var diagnostic = screen.SyncReportsScreenView(screen.SyncReportsScreenParams{
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
	})
	assertContains(t, diagnostic, "Failure Category: unsupported activity history")
	assertContains(t, diagnostic, "Generate Diagnostic Report")
	assertContains(t, diagnostic, "Generate Diagnostic Report is available for this failure from this context.")
}
