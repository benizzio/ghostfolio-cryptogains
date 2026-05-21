// Package integration verifies black-box workflow behavior for the current
// slice, including method-specific report outcomes.
// Authored by: OpenCode
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
)

// TestReportGenerationMatchesControlledLedgersAcrossCostBasisMethods verifies
// that each supported method produces the controlled expected report outcome for
// the deterministic multi-year fixture.
// Authored by: OpenCode
func TestReportGenerationMatchesControlledLedgersAcrossCostBasisMethods(t *testing.T) {
	var methods = []reportmodel.CostBasisMethod{
		reportmodel.CostBasisMethodFIFO,
		reportmodel.CostBasisMethodLIFO,
		reportmodel.CostBasisMethodHIFO,
		reportmodel.CostBasisMethodAverageCost,
		reportmodel.CostBasisMethodScopeLocalHybrid,
	}

	for _, method := range methods {
		var method = method
		t.Run(string(method), func(t *testing.T) {
			var reportIO = testutil.NewReportIOFixture(t)
			var openLogPath = installOpenCommandRecorder(t, 0)
			var fixture = testutil.DeterministicReportLedgerFixture()
			var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

			seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

			var model = unlockSyncReportsContext(t, harness.Model, "token-123")
			model = openReportSelectionFromContext(t, model)
			model = selectReportYear(t, model, fixture.PrimaryReportYear)
			model = selectReportMethod(t, model, method.Label())

			var selectionContent = normalizeRenderedText(model.View().Content)
			if !strings.Contains(compactWhitespace(selectionContent), compactWhitespace(method.Explanation())) {
				t.Fatalf("expected method explanation %q, got %q", method.Explanation(), selectionContent)
			}

			var expected = fixture.ExpectedReports[method]
			model, cmd := startReportGenerationFromSelection(t, model)
			model = applyBatchCmd(t, model, cmd)

			if model.ActiveScreen() != "report_result" {
				t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
			}

			var content = normalizeRenderedText(model.View().Content)
			if !strings.Contains(content, "Selected Year: 2024") || !strings.Contains(content, "Cost Basis Method: "+method.Label()) {
				t.Fatalf("expected selected year and method in result view, got %q", content)
			}

			var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
			if len(files) != 1 {
				t.Fatalf("expected one saved Markdown file, got %#v", files)
			}

			var reportPath = files[0]
			testutil.AssertPathWithin(t, reportPath, reportIO.DocumentsDir)
			testutil.AssertRegularFile(t, reportPath)
			if !strings.HasPrefix(filepath.Base(reportPath), "ghostfolio-capital-gains-2024-"+method.FilenameSlug()+"-") {
				t.Fatalf("expected filename slug %q, got %q", method.FilenameSlug(), filepath.Base(reportPath))
			}

			var openerRequests = readOpenCommandRequests(t, openLogPath)
			if len(openerRequests) != 1 || openerRequests[0] != reportPath {
				t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
			}

			var rawReport, err = os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("read saved report %q: %v", reportPath, err)
			}
			var reportText = string(rawReport)
			for _, required := range []string{
				"- Cost Basis Method: " + method.Label(),
				"- Report Calculation Currency: " + expected.ReportCalculationCurrency,
				"## Gains-And-Losses Summary",
				"## Reference Section",
			} {
				if !strings.Contains(reportText, required) {
					t.Fatalf("expected saved report to contain %q, got %q", required, reportText)
				}
			}

			assertExpectedReportLedger(t, reportText, expected)
			assertTextOmitted(t, reportText, "token-123", reportPath)
			assertNoCleartextReportInAppStorage(t, harness.BaseDir)

			var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			model = assertFlowModel(t, updated)
			if model.ActiveScreen() != "sync_reports_menu" {
				t.Fatalf("expected result dismissal to return to sync and reports menu, got %s", model.ActiveScreen())
			}
		})
	}
}

// assertExpectedReportLedger verifies the rendered Markdown against one
// controlled expected report ledger.
// Authored by: OpenCode
func assertExpectedReportLedger(t *testing.T, reportText string, expected testutil.ExpectedReportLedger) {
	t.Helper()

	for _, entry := range expected.SummaryByAsset {
		var row = "| " + entry.DisplayLabel + " | " + entry.NetGainOrLoss + " | " + expected.ReportCalculationCurrency + " |"
		if !strings.Contains(reportText, row) {
			t.Fatalf("expected summary row %q in report %q", row, reportText)
		}
	}

	var yearlyNetRow = "| Overall Yearly Net Total | " + expected.YearlyNetTotal + " | " + expected.ReportCalculationCurrency + " |"
	if !strings.Contains(reportText, yearlyNetRow) {
		t.Fatalf("expected yearly net total row %q in report %q", yearlyNetRow, reportText)
	}

	for _, entry := range expected.ReferenceByAsset {
		var row = "| " + entry.DisplayLabel + " | " + decimalInt(entry.FullLiquidationCountThroughYearEnd) + " | " + string(entry.MainSectionStatus) + " |"
		if !strings.Contains(reportText, row) {
			t.Fatalf("expected reference row %q in report %q", row, reportText)
		}
	}

	for _, detail := range expected.DetailByAsset {
		if !strings.Contains(reportText, "## Asset Detail: "+detail.DisplayLabel) {
			t.Fatalf("expected detail heading for %q in report %q", detail.DisplayLabel, reportText)
		}
		for _, row := range detail.ActivityRows {
			assertExpectedActivityRow(t, reportText, row)
		}
		for _, liquidation := range detail.LiquidationSummaries {
			var row = "| " + liquidation.SourceID + " |"
			if !strings.Contains(reportText, row) {
				row = "| " + liquidation.SourceID + " |"
			}
			for _, part := range []string{
				liquidation.SourceID,
				liquidation.DisposedQuantity,
				liquidation.ActivityCurrency,
				liquidation.AllocatedBasis,
				liquidation.NetLiquidationProceeds,
				liquidation.GainOrLoss,
				liquidation.CalculationCurrency,
			} {
				if !strings.Contains(reportText, part) {
					t.Fatalf("expected liquidation fragment %q in report %q", part, reportText)
				}
			}
		}
	}
}

// assertExpectedActivityRow verifies one expected detail activity row by its
// stable source identifier and rendered value fragments.
// Authored by: OpenCode
func assertExpectedActivityRow(t *testing.T, reportText string, expected testutil.ExpectedAssetActivityRow) {
	t.Helper()

	for _, part := range []string{
		expected.SourceID,
		string(expected.ActivityType),
		expected.Quantity,
		expected.BasisAfterRow,
		expected.CalculationCurrency,
		expected.QuantityAfterRow,
	} {
		if !strings.Contains(reportText, part) {
			t.Fatalf("expected activity-row fragment %q in report %q", part, reportText)
		}
	}

	if expected.GrossValue != "" && !strings.Contains(reportText, expected.GrossValue) {
		t.Fatalf("expected priced row gross value %q in report %q", expected.GrossValue, reportText)
	}
	if expected.FeeAmount != "" && !strings.Contains(reportText, expected.FeeAmount) {
		t.Fatalf("expected priced row fee %q in report %q", expected.FeeAmount, reportText)
	}
	if expected.ActivityCurrency != "" && !strings.Contains(reportText, expected.ActivityCurrency) {
		t.Fatalf("expected activity currency %q in report %q", expected.ActivityCurrency, reportText)
	}
	if expected.HoldingReductionExplanation != "" && !strings.Contains(reportText, expected.HoldingReductionExplanation) {
		t.Fatalf("expected holding reduction explanation %q in report %q", expected.HoldingReductionExplanation, reportText)
	}
}

// decimalInt formats one integration assertion integer without extra helpers.
// Authored by: OpenCode
func decimalInt(value int) string {
	return strings.TrimSpace(apd.New(int64(value), 0).String())
}

// compactWhitespace normalizes wrapped UI text for substring assertions.
// Authored by: OpenCode
func compactWhitespace(value string) string {
	var compact = strings.Join(strings.Fields(value), " ")
	compact = strings.ReplaceAll(compact, "- ", "-")
	return compact
}

// assertIntegrationDecimalString verifies one exact decimal using canonical
// formatting.
// Authored by: OpenCode
func assertIntegrationDecimalString(t *testing.T, value apd.Decimal, want string, label string) {
	t.Helper()

	var canonical, err = decimalsupport.CanonicalString(value)
	if err != nil {
		t.Fatalf("canonicalize %s: %v", label, err)
	}
	if canonical != want {
		t.Fatalf("unexpected %s: got %q want %q", label, canonical, want)
	}
}
