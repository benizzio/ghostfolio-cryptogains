// Package contract verifies rendered workflow and Ghostfolio-boundary contracts
// for the sync-and-storage slice.
// Authored by: OpenCode
package contract

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

// reportGenerationWorkflowSyncService supplies an unlocked reportable context
// for root-flow contract navigation.
//
// Authored by: OpenCode
type reportGenerationWorkflowSyncService struct{}

// Run is unused by this report-selection contract.
// Authored by: OpenCode
func (reportGenerationWorkflowSyncService) Run(context.Context, runtime.SyncRequest) runtime.SyncOutcome {
	return runtime.SyncOutcome{}
}

// GenerateDiagnosticReport is unused by this report-selection contract.
// Authored by: OpenCode
func (reportGenerationWorkflowSyncService) GenerateDiagnosticReport(context.Context, runtime.DiagnosticReportRequest) (string, error) {
	return "", nil
}

// ProtectedDataState reports no startup snapshot before the test unlock action.
// Authored by: OpenCode
func (reportGenerationWorkflowSyncService) ProtectedDataState() runtime.ProtectedDataState {
	return runtime.ProtectedDataState{}
}

// UnlockSelectedServerSnapshot provides one reportable year for the root-flow
// contract navigation.
// Authored by: OpenCode
func (reportGenerationWorkflowSyncService) UnlockSelectedServerSnapshot(context.Context, configmodel.AppSetupConfig, string) runtime.SyncReportsContextResult {
	return runtime.SyncReportsContextResult{
		UnlockState: runtime.SyncReportsUnlockStateAuthenticatedNewContext,
		ProtectedData: runtime.ProtectedDataState{
			HasReadableSnapshot:  true,
			AvailableReportYears: []int{2024},
		},
		ReportUnavailableReason: runtime.ReportFailureNone,
	}
}

// CheckServerReplacement is unused because this contract does not start sync.
// Authored by: OpenCode
func (reportGenerationWorkflowSyncService) CheckServerReplacement(configmodel.AppSetupConfig) runtime.ServerReplacementCheck {
	return runtime.ServerReplacementCheck{}
}

// reportGenerationWorkflowReportService records whether the asynchronous
// report command has been executed.
//
// Authored by: OpenCode
type reportGenerationWorkflowReportService struct {
	called bool
}

// Generate records execution so the transition can prove it did not run
// synchronously in the Bubble Tea update.
// Authored by: OpenCode
func (service *reportGenerationWorkflowReportService) Generate(context.Context, runtime.ReportGenerationRequest) runtime.ReportOutcome {
	service.called = true
	return runtime.ReportOutcome{}
}

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
		OutputFormatItems: []component.MenuItem{
			{Label: reportmodel.ReportOutputFormatMarkdown.Label(), Enabled: true},
			{Label: reportmodel.ReportOutputFormatPDF.Label(), Enabled: true},
		},
		SelectedOutputFormatIndex: 1,
		MenuItems:                 []component.MenuItem{{Label: "Generate Report", Enabled: true}, {Label: "Back", Enabled: true}},
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
	assertContains(t, selection, "Report Base Currency")
	assertContains(t, selection, "USD")
	assertContains(t, selection, "EUR")
	assertNotContains(t, selection, "GBP")
	assertContains(t, selection, "Output Format")
	assertContains(t, selection, "Markdown")
	assertContains(t, selection, "PDF")
	assertContains(t, selection, "Method Explanation")
	assertContains(t, selection, reportmodel.CostBasisMethodFIFO.Explanation())
	assertContains(t, selection, "Generate Report")
	assertContains(t, selection, "Back")

	var busy = screen.ReportBusyScreenView(screen.ReportBusyScreenParams{
		Theme:              component.DefaultTheme(),
		Width:              100,
		Height:             32,
		SelectedYear:       2024,
		MethodLabel:        "FIFO",
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		OutputFormat:       reportmodel.ReportOutputFormatPDF,
		BusyText:           "Generating capital gains report...",
		SpinnerFrame:       "*",
	})
	assertContains(t, busy, "Report Generation")
	assertContains(t, busy, "Generating capital gains report")
	assertContains(t, busy, "Selected Year: 2024")
	assertContains(t, busy, "Cost Basis Method: FIFO")
	assertContains(t, busy, "Report Base Currency: USD")
	assertContains(t, busy, "Output Format: PDF")
	assertNotContains(t, busy, "# Ghostfolio Capital Gains And Losses Report")

	request, err := reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportmodel.ReportBaseCurrencyUSD, reportmodel.ReportOutputFormatPDF, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	outputFile, err := reportmodel.NewReportOutputFile("/tmp/Documents", "ghostfolio-capital-gains-2024-fifo.pdf", "/tmp/Documents/ghostfolio-capital-gains-2024-fifo.pdf", reportmodel.ReportDocumentRoleCombined, reportmodel.ReportMediaTypePDF, time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC))
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
			Success:      true,
			Message:      "Saved the report to \"/tmp/Documents/ghostfolio-capital-gains-2024-fifo.pdf\" and requested automatic opening.",
			Request:      request,
			OutputFormat: reportmodel.ReportOutputFormatPDF,
			OutputFile:   outputFile,
			OutputBundle: reportmodel.ReportOutputBundle{OutputFormat: reportmodel.ReportOutputFormatPDF, Files: []reportmodel.ReportOutputFile{outputFile}},
		},
	})
	assertContains(t, result, "Report Result")
	assertContains(t, result, component.ReportCleartextExportDisclosureText)
	assertContains(t, result, "Saved PDF Path: /tmp/Documents/ghostfolio-capital-gains-2024-fifo.pdf")
	assertContains(t, result, component.ReportCleartextExportDeletionGuidanceText)
	assertContains(t, result, "Selected Year: 2024")
	assertContains(t, result, "Cost Basis Method: FIFO")
	assertContains(t, result, "Report Base Currency: USD")
	assertContains(t, result, "Output Format: PDF")
	assertContains(t, result, "Back To Sync and Reports")
	assertContains(t, result, "Generate Another Report")

	annexFile, err := reportmodel.NewReportOutputFile("/tmp/Documents", "ghostfolio-capital-gains-2024-fifo-annex-1.md", "/tmp/Documents/ghostfolio-capital-gains-2024-fifo-annex-1.md", reportmodel.ReportDocumentRoleAnnex, reportmodel.ReportMediaTypeMarkdown, time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatalf("new annex report output file: %v", err)
	}
	markdownBundle, err := reportmodel.NewReportOutputBundle(
		reportmodel.ReportOutputFormatMarkdown,
		[]reportmodel.ReportOutputFile{
			{
				DocumentsDirectory: "/tmp/Documents",
				Filename:           "ghostfolio-capital-gains-2024-fifo.md",
				Path:               "/tmp/Documents/ghostfolio-capital-gains-2024-fifo.md",
				Role:               reportmodel.ReportDocumentRoleMain,
				MediaType:          reportmodel.ReportMediaTypeMarkdown,
				SavedAt:            time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC),
			},
			annexFile,
		},
		time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC),
		true,
		"",
	)
	if err != nil {
		t.Fatalf("new Markdown report output bundle: %v", err)
	}
	var markdownResult = screen.ReportResultScreenView(screen.ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		MethodLabel:   "FIFO",
		MenuItems:     []component.MenuItem{{Label: "Back To Sync and Reports", Enabled: true}, {Label: "Generate Another Report", Enabled: true}},
		SelectedIndex: 0,
		Outcome: runtime.ReportOutcome{
			Success:      true,
			Message:      "Saved both Markdown report files and requested automatic opening.",
			Request:      request,
			OutputFormat: reportmodel.ReportOutputFormatMarkdown,
			OutputFile:   markdownBundle.Files[0],
			OutputBundle: markdownBundle,
		},
	})
	assertContains(t, markdownResult, "Saved Markdown Path: /tmp/Documents/ghostfolio-capital-gains-2024-fifo.md")
	assertContains(t, markdownResult, "Saved Annex 1 Markdown Path: /tmp/Documents/ghostfolio-capital-gains-2024-fifo-annex-1.md")
	assertContains(t, markdownResult, component.ReportCleartextExportDisclosureText)
	assertContains(t, markdownResult, component.ReportCleartextExportDeletionGuidanceText)

	var warningFile, warningErr = reportmodel.NewReportOutputFile("/tmp/Documents", "synthetic-warning-report.pdf", "/tmp/Documents/synthetic-warning-report.pdf", reportmodel.ReportDocumentRoleCombined, reportmodel.ReportMediaTypePDF, time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC))
	if warningErr != nil {
		t.Fatalf("new warning report output file: %v", warningErr)
	}
	var warningBundle, warningBundleErr = reportmodel.NewReportOutputBundle(reportmodel.ReportOutputFormatPDF, []reportmodel.ReportOutputFile{warningFile}, time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC), true, "synthetic opener warning")
	if warningBundleErr != nil {
		t.Fatalf("new warning report output bundle: %v", warningBundleErr)
	}
	var warningResult = screen.ReportResultScreenView(screen.ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         100,
		Height:        32,
		MethodLabel:   "FIFO",
		MenuItems:     []component.MenuItem{{Label: "Back To Sync and Reports", Enabled: true}},
		SelectedIndex: 0,
		Outcome: runtime.ReportOutcome{
			Success:       true,
			FailureReason: runtime.ReportFailureAutomaticOpenFailedAfterSave,
			Message:       "Saved the synthetic PDF, but automatic opening failed. Open the file manually.",
			Request:       request,
			OutputFormat:  reportmodel.ReportOutputFormatPDF,
			OutputFile:    warningFile,
			OutputBundle:  warningBundle,
		},
	})
	assertContains(t, warningResult, "Success With Warning: automatic open failed after save")
	assertContains(t, warningResult, component.ReportCleartextExportDisclosureText)
	assertContains(t, warningResult, "Saved PDF Path: /tmp/Documents/synthetic-warning-report.pdf")
	assertContains(t, warningResult, component.ReportCleartextExportDeletionGuidanceText)

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
				Year:               2025,
				CostBasisMethod:    reportmodel.CostBasisMethodHIFO,
				ReportBaseCurrency: reportmodel.ReportBaseCurrencyEUR,
				RequestedAt:        time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC),
			},
		},
	})
	assertContains(t, failure, "Failure Category: unsupported report calculation")
	assertContains(t, failure, "Selected Year: 2025")
	assertContains(t, failure, "Cost Basis Method: HIFO")
	assertContains(t, failure, "Report Base Currency: EUR")
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

// TestReportBaseCurrencyChoiceContract verifies the supported report base
// currency set used by report generation requests.
// Authored by: OpenCode
func TestReportBaseCurrencyChoiceContract(t *testing.T) {
	t.Parallel()

	var currencies = reportmodel.SupportedReportBaseCurrencies()
	if len(currencies) != 2 {
		t.Fatalf("expected exactly two report base currencies, got %#v", currencies)
	}
	if currencies[0] != reportmodel.ReportBaseCurrencyUSD || currencies[1] != reportmodel.ReportBaseCurrencyEUR {
		t.Fatalf("expected USD and EUR report base currencies in stable order, got %#v", currencies)
	}

	var err error
	for _, unsupported := range []reportmodel.ReportBaseCurrency{"", "GBP"} {
		_, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, unsupported, reportmodel.ReportOutputFormatMarkdown, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
		if err == nil {
			t.Fatalf("expected unsupported report base currency %q to fail request validation", unsupported)
		}
	}
}

// TestReportGenerationSelectionToStartWorkflowBoundContract verifies that the
// root Bubble Tea workflow selects PDF and enters its busy state before the
// asynchronous report-generation command executes.
// Authored by: OpenCode
func TestReportGenerationSelectionToStartWorkflowBoundContract(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(
		configmodel.ServerModeGhostfolioCloud,
		configmodel.GhostfolioCloudOrigin,
		false,
		time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var reportService = &reportGenerationWorkflowReportService{}
	var model = flow.NewModel(flow.Dependencies{
		Options:       bootstrap.DefaultOptions(),
		Startup:       bootstrap.StartupState{ActiveConfig: &config},
		SyncService:   reportGenerationWorkflowSyncService{},
		ReportService: reportService,
	})

	var updated tea.Model
	var cmd tea.Cmd
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	if cmd != nil {
		updated, _ = model.Update(cmd())
		model = updated.(*flow.Model)
	}
	updated, _ = model.Update(tea.PasteMsg{Content: "contract-token"})
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected unlocked Sync and Reports menu, got %s", model.ActiveScreen())
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*flow.Model)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	if model.ActiveScreen() != "report_selection" {
		t.Fatalf("expected report selection screen, got %s", model.ActiveScreen())
	}

	for range 3 {
		updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		model = updated.(*flow.Model)
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(*flow.Model)
	assertContains(t, model.View().Content, "> PDF")
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)

	var startedAt = time.Now()
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = updated.(*flow.Model)
	var elapsed = time.Since(startedAt)
	if elapsed > 30*time.Second {
		t.Fatalf("expected PDF selection-to-busy transition within the SC-001 30-second bound, took %s", elapsed)
	}
	if cmd == nil {
		t.Fatal("expected Generate Report to return an asynchronous command")
	}
	if reportService.called {
		t.Fatal("expected report generation to remain outside the synchronous Bubble Tea transition")
	}

	if model.ActiveScreen() != "report_busy" {
		t.Fatalf("expected immediate report busy screen, got %s", model.ActiveScreen())
	}
	var busy = model.View().Content
	assertContains(t, busy, "Report Generation")
	assertContains(t, busy, "Selected Year: 2024")
	assertContains(t, busy, "Cost Basis Method: FIFO")
	assertContains(t, busy, "Report Base Currency: USD")
	assertContains(t, busy, "Output Format: PDF")
	assertNotContains(t, busy, "Saved PDF Path")
	assertNotContains(t, busy, "Saved Markdown Path")
	assertNotContains(t, busy, "/tmp/")
}
