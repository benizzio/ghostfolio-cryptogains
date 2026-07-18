// Package screen verifies report workflow rendering contracts.
// Authored by: OpenCode
package screen

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// TestReportSelectionScreenViewRendersBaseCurrencyMenu verifies the report
// selection screen exposes the required USD/EUR base-currency menu.
// Authored by: OpenCode
func TestReportSelectionScreenViewRendersBaseCurrencyMenu(t *testing.T) {
	t.Parallel()

	var content = ReportSelectionScreenView(ReportSelectionScreenParams{
		Theme:             component.DefaultTheme(),
		Width:             80,
		Height:            24,
		AvailableYears:    []int{2024, 2025},
		SelectedYearIndex: 0,
		MethodItems:       []component.MenuItem{{Label: "FIFO", Enabled: true}, {Label: "LIFO", Enabled: true}},
		SelectedMethod:    0,
		MethodExplanation: reportmodel.CostBasisMethodFIFO.Explanation(),
		MenuItems:         []component.MenuItem{{Label: component.GenerateReportActionLabel, Enabled: false}, {Label: component.BackActionLabel, Enabled: true}},
		SelectedAction:    0,
		HelpText:          "help",
	})

	assertReportScreenContainsAll(t, content, []string{
		"Report Base Currency",
		"USD",
		"EUR",
		"all monetary",
		"report calculations and totals will use the selected base currency",
	})
	if strings.Contains(content, "GBP") {
		t.Fatalf("expected base-currency menu to be limited to USD and EUR, got %q", content)
	}
}

// TestReportOutputFormatExplanationRejectsUnsupportedFormats verifies the
// selection screen provides corrective copy before report generation starts.
// Authored by: OpenCode
func TestReportOutputFormatExplanationRejectsUnsupportedFormats(t *testing.T) {
	t.Parallel()

	if explanation := reportOutputFormatExplanation(reportmodel.ReportOutputFormatPDF); !strings.Contains(explanation, "one local A4 text PDF") {
		t.Fatalf("unexpected PDF explanation: %q", explanation)
	}
	if explanation := reportOutputFormatExplanation(reportmodel.ReportOutputFormatMarkdown); !strings.Contains(explanation, "separate Annex 1 Markdown") {
		t.Fatalf("unexpected Markdown explanation: %q", explanation)
	}
	var explanation = reportOutputFormatExplanation(reportmodel.ReportOutputFormat("html"))
	if !strings.Contains(explanation, "Choose Markdown or PDF") {
		t.Fatalf("unexpected unsupported-format explanation: %q", explanation)
	}
}

// TestReportBusyScreenViewRendersSelectedBaseCurrency verifies the busy screen
// keeps the selected report base currency visible during asynchronous work.
// Authored by: OpenCode
func TestReportBusyScreenViewRendersSelectedBaseCurrency(t *testing.T) {
	t.Parallel()

	var params = ReportBusyScreenParams{
		Theme:        component.DefaultTheme(),
		Width:        80,
		Height:       24,
		SelectedYear: 2024,
		MethodLabel:  reportmodel.CostBasisMethodFIFO.Label(),
		BusyText:     "Generating capital gains report...",
		SpinnerFrame: "*",
		HelpText:     "help",
	}
	setReportBusyBaseCurrencyForTest(t, &params, reportmodel.ReportBaseCurrencyEUR)

	var content = ReportBusyScreenView(params)
	assertReportScreenContainsAll(t, content, []string{"Selected Year: 2024", "Cost Basis Method: FIFO", "Report Base Currency: EUR"})
}

// TestReportResultScreenViewRendersSelectedBaseCurrency verifies completed
// report outcomes disclose the base currency used for the report request.
// Authored by: OpenCode
func TestReportResultScreenViewRendersSelectedBaseCurrency(t *testing.T) {
	t.Parallel()

	var request, err = reportmodel.NewReportRequest(
		2024,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyEUR,
		reportmodel.ReportOutputFormatMarkdown,
		time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	var outputFile, outputErr = reportmodel.NewReportOutputFile(
		"/tmp/Documents",
		"ghostfolio-capital-gains-2024-fifo.md",
		"/tmp/Documents/ghostfolio-capital-gains-2024-fifo.md",
		reportmodel.ReportDocumentRoleMain,
		reportmodel.ReportMediaTypeMarkdown,
		time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC),
	)
	if outputErr != nil {
		t.Fatalf("new report output file: %v", outputErr)
	}

	var content = ReportResultScreenView(ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MethodLabel:   reportmodel.CostBasisMethodFIFO.Label(),
		MenuItems:     []component.MenuItem{{Label: component.BackToSyncReportsActionLabel, Enabled: true}, {Label: component.GenerateAnotherReportActionLabel, Enabled: true}},
		SelectedIndex: 0,
		HelpText:      "help",
		Outcome: runtime.ReportOutcome{
			Success:    true,
			Message:    "Saved the report to \"/tmp/report.md\" and requested automatic opening.",
			Request:    request,
			OutputFile: outputFile,
		},
	})

	assertReportScreenContainsAll(t, content, []string{"Selected Year: 2024", "Cost Basis Method: FIFO", "Report Base Currency: EUR"})
}

// TestReportResultScreenViewRendersFailureBaseCurrency verifies failed report
// outcomes keep the selected report base currency visible.
// Authored by: OpenCode
func TestReportResultScreenViewRendersFailureBaseCurrency(t *testing.T) {
	t.Parallel()

	var request, err = reportmodel.NewReportRequest(
		2024,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatMarkdown,
		time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	var content = ReportResultScreenView(ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MethodLabel:   reportmodel.CostBasisMethodFIFO.Label(),
		MenuItems:     []component.MenuItem{{Label: component.BackToSyncReportsActionLabel, Enabled: true}},
		SelectedIndex: 0,
		HelpText:      "help",
		Outcome: runtime.ReportOutcome{
			Success:       false,
			FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
			Message:       "Could not generate the report. No report file was saved.",
			Request:       request,
		},
	})

	assertReportScreenContainsAll(t, content, []string{"Failure Category: unsupported report calculation", "Report Base Currency: USD"})
}

// TestReportSelectionScreenViewRendersUnselectedOutputFormatGuidance verifies
// stale output-format selections render actionable fallback copy.
// Authored by: OpenCode
func TestReportSelectionScreenViewRendersUnselectedOutputFormatGuidance(t *testing.T) {
	t.Parallel()

	var content = ReportSelectionScreenView(ReportSelectionScreenParams{
		Theme:                     component.DefaultTheme(),
		Width:                     80,
		Height:                    24,
		AvailableYears:            []int{2024},
		SelectedYearIndex:         0,
		MethodItems:               []component.MenuItem{{Label: "FIFO", Enabled: true}},
		SelectedMethod:            0,
		SelectedOutputFormatIndex: 99,
		MethodExplanation:         reportmodel.CostBasisMethodFIFO.Explanation(),
		MenuItems:                 []component.MenuItem{{Label: component.GenerateReportActionLabel, Enabled: false}},
		HelpText:                  "help",
	})

	assertReportScreenContainsAll(t, content, []string{"Output Format Explanation", "Choose Markdown or PDF before generation starts."})
}

// TestReportResultScreenViewRendersPDFBundlePath verifies combined PDF bundle
// output uses the PDF saved-path label.
// Authored by: OpenCode
func TestReportResultScreenViewRendersPDFBundlePath(t *testing.T) {
	t.Parallel()

	var request, err = reportmodel.NewReportRequest(
		2024,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatPDF,
		time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	var savedAt = time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC)
	var pdfFile, outputErr = reportmodel.NewReportOutputFile(
		"/tmp/Documents",
		"ghostfolio-capital-gains-2024-fifo.pdf",
		"/tmp/Documents/ghostfolio-capital-gains-2024-fifo.pdf",
		reportmodel.ReportDocumentRoleCombined,
		reportmodel.ReportMediaTypePDF,
		savedAt,
	)
	if outputErr != nil {
		t.Fatalf("new PDF output file: %v", outputErr)
	}
	var bundle, bundleErr = reportmodel.NewReportOutputBundle(reportmodel.ReportOutputFormatPDF, []reportmodel.ReportOutputFile{pdfFile}, savedAt, true, "")
	if bundleErr != nil {
		t.Fatalf("new PDF output bundle: %v", bundleErr)
	}

	var content = ReportResultScreenView(ReportResultScreenParams{
		Theme:         component.DefaultTheme(),
		Width:         80,
		Height:        24,
		MethodLabel:   reportmodel.CostBasisMethodFIFO.Label(),
		MenuItems:     []component.MenuItem{{Label: component.BackToSyncReportsActionLabel, Enabled: true}},
		SelectedIndex: 0,
		HelpText:      "help",
		Outcome: runtime.ReportOutcome{
			Success:      true,
			Message:      "Saved the report.",
			Request:      request,
			OutputBundle: bundle,
			OutputFile:   pdfFile,
		},
	})

	assertReportScreenContainsAll(t, content, []string{"Output Format: PDF", "Saved PDF Path: /tmp/Documents/ghostfolio-capital-gains-2024-fifo.pdf"})
}

// TestReportResultScreenViewDisclosesCleartextFilesAndDeletionGuidance verifies
// normal Markdown and opener-warning PDF results disclose every saved path.
// Authored by: OpenCode
func TestReportResultScreenViewDisclosesCleartextFilesAndDeletionGuidance(t *testing.T) {
	t.Parallel()

	var savedAt = time.Date(2026, time.May, 21, 11, 0, 1, 0, time.UTC)
	testCases := []struct {
		name          string
		requestFormat reportmodel.ReportOutputFormat
		failureReason runtime.ReportFailureReason
		message       string
		files         []reportmodel.ReportOutputFile
	}{
		{
			name:          "normal Markdown success",
			requestFormat: reportmodel.ReportOutputFormatMarkdown,
			message:       "Saved both Markdown report files and requested automatic opening.",
			files: []reportmodel.ReportOutputFile{
				{DocumentsDirectory: "/tmp/Documents", Filename: "synthetic-report.md", Path: "/tmp/Documents/synthetic-report.md", Role: reportmodel.ReportDocumentRoleMain, MediaType: reportmodel.ReportMediaTypeMarkdown, SavedAt: savedAt},
				{DocumentsDirectory: "/tmp/Documents", Filename: "synthetic-report-annex-1.md", Path: "/tmp/Documents/synthetic-report-annex-1.md", Role: reportmodel.ReportDocumentRoleAnnex, MediaType: reportmodel.ReportMediaTypeMarkdown, SavedAt: savedAt},
			},
		},
		{
			name:          "opener-warning PDF success",
			requestFormat: reportmodel.ReportOutputFormatPDF,
			failureReason: runtime.ReportFailureAutomaticOpenFailedAfterSave,
			message:       "Saved the PDF report, but automatic opening failed. Open the file manually.",
			files: []reportmodel.ReportOutputFile{
				{DocumentsDirectory: "/tmp/Documents", Filename: "synthetic-report.pdf", Path: "/tmp/Documents/synthetic-report.pdf", Role: reportmodel.ReportDocumentRoleCombined, MediaType: reportmodel.ReportMediaTypePDF, SavedAt: savedAt},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var request, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportmodel.ReportBaseCurrencyUSD, testCase.requestFormat, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatalf("new report request: %v", err)
			}
			var bundle, bundleErr = reportmodel.NewReportOutputBundle(testCase.requestFormat, testCase.files, savedAt, true, "")
			if bundleErr != nil {
				t.Fatalf("new report output bundle: %v", bundleErr)
			}

			var content = ReportResultScreenView(ReportResultScreenParams{
				Theme:       component.DefaultTheme(),
				Width:       100,
				Height:      32,
				MethodLabel: reportmodel.CostBasisMethodFIFO.Label(),
				MenuItems:   []component.MenuItem{{Label: component.BackToSyncReportsActionLabel, Enabled: true}},
				Outcome: runtime.ReportOutcome{
					Success:       true,
					FailureReason: testCase.failureReason,
					Message:       testCase.message,
					Request:       request,
					OutputFormat:  testCase.requestFormat,
					OutputFile:    testCase.files[0],
					OutputBundle:  bundle,
				},
			})

			assertReportScreenContainsAll(t, content, []string{
				component.ReportCleartextExportDisclosureText,
				component.ReportCleartextExportDeletionGuidanceText,
			})
			for _, file := range testCase.files {
				if strings.Count(content, file.Path) != 1 {
					t.Fatalf("expected saved path %q, got %q", file.Path, content)
				}
			}
			if strings.Count(content, component.ReportCleartextExportDisclosureText) != 1 || strings.Count(content, component.ReportCleartextExportDeletionGuidanceText) != 1 {
				t.Fatalf("expected cleartext disclosure and deletion guidance once, got %q", content)
			}
		})
	}
}

// assertReportScreenContainsAll verifies that rendered report content includes
// every expected plain-text fragment.
// Authored by: OpenCode
func assertReportScreenContainsAll(t *testing.T, content string, expected []string) {
	t.Helper()

	for _, fragment := range expected {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected rendered content to contain %q, got %q", fragment, content)
		}
	}
}

// setReportBusyBaseCurrencyForTest assigns the selected base currency through
// the expected busy-screen render parameter once the production API exists.
// Authored by: OpenCode
func setReportBusyBaseCurrencyForTest(t *testing.T, params *ReportBusyScreenParams, currency reportmodel.ReportBaseCurrency) {
	t.Helper()

	var currencyValue = reflect.ValueOf(currency)
	var paramsValue = reflect.ValueOf(params).Elem()
	for _, fieldName := range []string{"ReportBaseCurrency", "SelectedBaseCurrency", "BaseCurrency"} {
		var field = paramsValue.FieldByName(fieldName)
		if !field.IsValid() || !field.CanSet() {
			continue
		}
		if field.Type() == currencyValue.Type() {
			field.Set(currencyValue)
			return
		}
		if field.Kind() == reflect.String {
			field.SetString(currency.Label())
			return
		}
	}

	t.Fatalf("ReportBusyScreenParams must expose the selected report base currency for rendering")
}
