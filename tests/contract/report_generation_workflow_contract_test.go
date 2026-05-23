// Package contract verifies rendered workflow and Ghostfolio-boundary contracts
// for the sync-and-storage slice.
// Authored by: OpenCode
package contract

import (
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

// TestReportGenerationWorkflowContract verifies the visible report selection,
// busy, and result workflow contract.
// Authored by: OpenCode
func TestReportGenerationWorkflowContract(t *testing.T) {
	t.Parallel()

	var selection = screen.ReportSelectionScreenView(screen.ReportSelectionScreenParams{
		Theme:             component.DefaultTheme(),
		Width:             100,
		Height:            32,
		AvailableYears:    []int{2024, 2025},
		SelectedYearIndex: 0,
		MethodItems: []component.MenuItem{
			{Label: "FIFO", Enabled: true},
			{Label: "LIFO", Enabled: true},
			{Label: "HIFO", Enabled: true},
			{Label: "Average Cost Basis", Enabled: true},
			{Label: "Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order", Enabled: true},
		},
		SelectedMethod:    0,
		MethodExplanation: reportmodel.CostBasisMethodFIFO.Explanation(),
		MenuItems:         []component.MenuItem{{Label: "Generate Report", Enabled: true}, {Label: "Back", Enabled: true}},
	})
	assertContains(t, selection, "Generate Capital Gains Report")
	assertContains(t, selection, "Available Years")
	assertContains(t, selection, "2024")
	assertContains(t, selection, "2025")
	assertContains(t, selection, "Cost Basis Methods")
	assertContains(t, selection, "FIFO")
	assertContains(t, selection, "LIFO")
	assertContains(t, selection, "HIFO")
	assertContains(t, selection, "Average Cost Basis")
	assertContains(t, selection, "Scope-Local Exact Unit Matching")
	assertContains(t, selection, "Oldest-Acquired")
	assertContains(t, selection, "Method Explanation")
	assertContains(t, selection, reportmodel.CostBasisMethodFIFO.Explanation())
	assertContains(t, selection, "Generate Report")
	assertContains(t, selection, "Back")

	var busy = screen.ReportBusyScreenView(screen.ReportBusyScreenParams{
		Theme:        component.DefaultTheme(),
		Width:        100,
		Height:       32,
		SelectedYear: 2024,
		MethodLabel:  "FIFO",
		BusyText:     "Generating capital gains report...",
		SpinnerFrame: "*",
	})
	assertContains(t, busy, "Report Generation")
	assertContains(t, busy, "Generating capital gains report")
	assertContains(t, busy, "Selected Year: 2024")
	assertContains(t, busy, "Cost Basis Method: FIFO")
	assertNotContains(t, busy, "# Ghostfolio Capital Gains And Losses Report")

	request, err := reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	outputFile, err := reportmodel.NewReportOutputFile("/tmp/Documents", "ghostfolio-capital-gains-2024-fifo.md", "/tmp/Documents/ghostfolio-capital-gains-2024-fifo.md", time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC), true, "")
	if err != nil {
		t.Fatalf("new report output file: %v", err)
	}

	var result = screen.ReportResultScreenView(screen.ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		MethodLabel:   "FIFO",
		MenuItems:     []component.MenuItem{{Label: "Back To Sync and Reports", Enabled: true}, {Label: "Generate Another Report", Enabled: true}},
		SelectedIndex: 0,
		Outcome: runtime.ReportOutcome{
			Success:    true,
			Message:    "Saved the report to \"/tmp/Documents/ghostfolio-capital-gains-2024-fifo.md\" and requested automatic opening.",
			Request:    request,
			OutputFile: outputFile,
		},
	})
	assertContains(t, result, "Report Result")
	assertContains(t, result, "Saved Markdown Path: /tmp/Documents/ghostfolio-capital-gains-2024-fifo.md")
	assertContains(t, result, "Selected Year: 2024")
	assertContains(t, result, "Cost Basis Method: FIFO")
	assertContains(t, result, "Back To Sync and Reports")
	assertContains(t, result, "Generate Another Report")

	var failure = screen.ReportResultScreenView(screen.ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		MethodLabel:   "HIFO",
		MenuItems:     []component.MenuItem{{Label: "Generate Diagnostic Report", Enabled: true}, {Label: "Back To Sync and Reports", Enabled: true}, {Label: "Generate Another Report", Enabled: true}},
		SelectedIndex: 0,
		Outcome: runtime.ReportOutcome{
			Success:       false,
			FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
			Message:       "Could not generate the selected report because the synced activity history is not supported for safe calculation.",
			Diagnostic:    runtime.DiagnosticReportState{Eligible: true},
			Request: reportmodel.ReportRequest{
				Year:            2025,
				CostBasisMethod: reportmodel.CostBasisMethodHIFO,
				RequestedAt:     time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC),
			},
		},
	})
	assertContains(t, failure, "Failure Category: unsupported report calculation")
	assertContains(t, failure, "Selected Year: 2025")
	assertContains(t, failure, "Cost Basis Method: HIFO")
	assertContains(t, failure, "Generate Diagnostic Report")
	assertContains(t, failure, "Generate Diagnostic Report is available for this failure from this screen.")
	assertContains(t, failure, "Back To Sync and Reports")

	var devFailure = screen.ReportResultScreenView(screen.ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		MethodLabel:   "FIFO",
		MenuItems:     []component.MenuItem{{Label: "Back To Sync and Reports", Enabled: true}, {Label: "Generate Another Report", Enabled: true}},
		SelectedIndex: 0,
		Outcome: runtime.ReportOutcome{
			Success:       false,
			FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
			Message:       "Could not generate the selected report because one activity is incomplete.",
			Diagnostic: runtime.DiagnosticReportState{
				Eligible:          true,
				GenerationMessage: "Diagnostic report generated successfully.",
				Path:              "/tmp/example.diagnostic.json",
			},
			Request: request,
		},
	})
	assertContains(t, devFailure, "Diagnostic report generated successfully.")
	assertContains(t, devFailure, "Diagnostic Report Path: /tmp/example.diagnostic.json")
}
