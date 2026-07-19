// Package integration verifies runtime-backed Annex presentation behavior and
// AUD-001 model integrity.
// Authored by: OpenCode
package integration

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// Synthetic source IDs identify the inherited classified and unclassified Annex
// controls in the deterministic runtime fixture.
// Authored by: OpenCode
const (
	// auditPresentationClassifiedSourceID identifies the inherited classified Annex control.
	// Authored by: OpenCode
	auditPresentationClassifiedSourceID = "xrp-reduction-2024-001"
	// auditPresentationUnclassifiedSourceID identifies the unclassified Annex control.
	// Authored by: OpenCode
	auditPresentationUnclassifiedSourceID = "eth-sell-2023-001"
)

// auditPresentationRow stores the semantic Annex values needed for a
// cross-format comparison. Empty cells are retained in Markdown cells and
// omitted from NonEmptyCells so the suppression boundary is observable in both
// renderers.
// Authored by: OpenCode
type auditPresentationRow struct {
	OriginalCurrency    string
	CalculationCurrency string
	FullLiquidation     string
	NonEmptyCells       []string
}

// TestReportAuditPresentationPreservesAUD001AndMatchesFormats verifies that
// runtime-generated Markdown and PDF Annex rows suppress only the visible
// classified original currency while retaining calculation evidence and
// inherited classification in the calculated report model.
// Authored by: OpenCode
func TestReportAuditPresentationPreservesAUD001AndMatchesFormats(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)
	var token = "report-audit-presentation-token"

	runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)
	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}

	var calculator = reportcalculate.NewCalculator(runtimeflow.DeterministicCurrencyRates{})
	var baseline, err = calculator.Calculate(context.Background(), runtimeflow.MustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportOutputFormatMarkdown), fixture.ProtectedActivityCache)
	if err != nil {
		t.Fatalf("calculate AUD-001 Annex baseline: %v", err)
	}
	var classifiedBaseline = auditPresentationEntry(t, baseline, auditPresentationClassifiedSourceID)
	var unclassifiedBaseline = auditPresentationEntry(t, baseline, auditPresentationUnclassifiedSourceID)
	assertAuditPresentationBaseline(t, classifiedBaseline, true, "USD", "No")
	assertAuditPresentationBaseline(t, unclassifiedBaseline, false, "USD", "Yes")

	var classifiedObservations []auditPresentationRow
	var unclassifiedObservations []auditPresentationRow
	for _, outputFormat := range []reportmodel.ReportOutputFormat{
		reportmodel.ReportOutputFormatMarkdown,
		reportmodel.ReportOutputFormatPDF,
	} {
		var request = runtimeflow.MustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, outputFormat)
		var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
		if !outcome.Success {
			t.Fatalf("expected %s Annex report generation success, got %#v", outputFormat, outcome)
		}

		var after, calculateErr = calculator.Calculate(context.Background(), request, fixture.ProtectedActivityCache)
		if calculateErr != nil {
			t.Fatalf("calculate post-render AUD-001 Annex model for %s: %v", outputFormat, calculateErr)
		}
		assertCalculatedReportUnchanged(t, outputFormat, baseline, after)
		assertAuditPresentationEntryUnchanged(t, outputFormat, baseline, after, auditPresentationClassifiedSourceID)
		assertAuditPresentationEntryUnchanged(t, outputFormat, baseline, after, auditPresentationUnclassifiedSourceID)

		var observation auditPresentationRow
		observation, err = readAuditPresentationOutput(t, reportIO.DocumentsDir, outcome, auditPresentationClassifiedSourceID)
		if err != nil {
			t.Fatalf("read classified Annex row from %s output: %v", outputFormat, err)
		}
		var unclassifiedRow auditPresentationRow
		unclassifiedRow, err = readAuditPresentationOutput(t, reportIO.DocumentsDir, outcome, auditPresentationUnclassifiedSourceID)
		if err != nil {
			t.Fatalf("read unclassified Annex row from %s output: %v", outputFormat, err)
		}
		assertAuditPresentationVisibleValues(t, outputFormat, observation, unclassifiedRow)
		classifiedObservations = append(classifiedObservations, observation)
		unclassifiedObservations = append(unclassifiedObservations, unclassifiedRow)
	}

	if len(classifiedObservations) != 2 || !reflect.DeepEqual(classifiedObservations[0].NonEmptyCells, classifiedObservations[1].NonEmptyCells) {
		t.Fatalf("expected Markdown and PDF classified Annex rows to agree, got %#v", classifiedObservations)
	}
	if len(unclassifiedObservations) != 2 || !reflect.DeepEqual(unclassifiedObservations[0].NonEmptyCells, unclassifiedObservations[1].NonEmptyCells) {
		t.Fatalf("expected Markdown and PDF unclassified Annex rows to agree, got %#v", unclassifiedObservations)
	}
	var openerRequests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 2 {
		t.Fatalf("expected one opener request for each Annex presentation format, got %#v", openerRequests)
	}
}

// auditPresentationEntry locates one calculated Annex entry by its synthetic
// source ID.
// Authored by: OpenCode
func auditPresentationEntry(t *testing.T, report reportmodel.CapitalGainsReport, sourceID string) reportmodel.AuditActivityEntry {
	t.Helper()
	for _, section := range report.AuditAnnex.PerAssetAuditSections {
		for _, entry := range section.Entries {
			if entry.SourceID == sourceID {
				return entry
			}
		}
	}
	t.Fatalf("expected calculated Annex entry %q, got %#v", sourceID, report.AuditAnnex.PerAssetAuditSections)
	return reportmodel.AuditActivityEntry{}
}

// assertAuditPresentationBaseline verifies the inherited pre-format audit
// currency, classification, and boolean state used by the renderer assertions.
// Authored by: OpenCode
func assertAuditPresentationBaseline(t *testing.T, entry reportmodel.AuditActivityEntry, classified bool, activityCurrency string, fullLiquidation string) {
	t.Helper()
	if entry.IsZeroPricedHoldingReduction != classified {
		t.Fatalf("Annex entry %q classification = %t, want %t", entry.SourceID, entry.IsZeroPricedHoldingReduction, classified)
	}
	if entry.ActivityCurrency != activityCurrency || entry.CalculationCurrency != "USD" {
		t.Fatalf("Annex entry %q currencies = %q/%q, want %q/USD", entry.SourceID, entry.ActivityCurrency, entry.CalculationCurrency, activityCurrency)
	}
	if entry.FullLiquidationEvent != (fullLiquidation == "Yes") {
		t.Fatalf("Annex entry %q full liquidation = %t, want %s", entry.SourceID, entry.FullLiquidationEvent, fullLiquidation)
	}
}

// assertAuditPresentationEntryUnchanged proves the selected audit evidence is
// equal before and after each renderer, including its inherited classification.
// Authored by: OpenCode
func assertAuditPresentationEntryUnchanged(t *testing.T, outputFormat reportmodel.ReportOutputFormat, before reportmodel.CapitalGainsReport, after reportmodel.CapitalGainsReport, sourceID string) {
	t.Helper()
	var beforeEntry = auditPresentationEntry(t, before, sourceID)
	var afterEntry = auditPresentationEntry(t, after, sourceID)
	if !reflect.DeepEqual(beforeEntry, afterEntry) {
		t.Fatalf("AUD-001 Annex entry %q changed after %s rendering: before=%#v after=%#v", sourceID, outputFormat, beforeEntry, afterEntry)
	}
}

// assertAuditPresentationVisibleValues verifies only the classified row's
// original currency is blank and that both boolean and calculation-currency
// values remain visible for classified and unclassified controls.
// Authored by: OpenCode
func assertAuditPresentationVisibleValues(t *testing.T, outputFormat reportmodel.ReportOutputFormat, classified auditPresentationRow, unclassified auditPresentationRow) {
	t.Helper()
	if classified.OriginalCurrency != "" {
		t.Fatalf("classified original currency in %s = %q, want blank", outputFormat, classified.OriginalCurrency)
	}
	if classified.CalculationCurrency != "USD" || classified.FullLiquidation != "No" {
		t.Fatalf("classified Annex values in %s = currency %q, liquidation %q, want USD/No", outputFormat, classified.CalculationCurrency, classified.FullLiquidation)
	}
	if unclassified.OriginalCurrency != "USD" || unclassified.CalculationCurrency != "USD" || unclassified.FullLiquidation != "Yes" {
		t.Fatalf("unclassified Annex values in %s = currency %q/%q, liquidation %q, want USD/USD/Yes", outputFormat, unclassified.OriginalCurrency, unclassified.CalculationCurrency, unclassified.FullLiquidation)
	}
}

// readAuditPresentationOutput reads one semantic Annex row from the selected
// runtime output and normalizes it into a format-neutral observation.
// Authored by: OpenCode
func readAuditPresentationOutput(t *testing.T, documentsDir string, outcome runtime.ReportOutcome, sourceID string) (auditPresentationRow, error) {
	t.Helper()
	if err := outcome.OutputBundle.Validate(); err != nil {
		return auditPresentationRow{}, err
	}
	if len(outcome.OutputBundle.Files) == 0 {
		return auditPresentationRow{}, os.ErrNotExist
	}

	switch outcome.OutputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		var files = runtimeflow.AllMarkdownFiles(t, documentsDir)
		var _, annexPath = runtimeflow.MarkdownBundlePaths(t, files)
		// #nosec G304 -- the report path is created in the test-owned Documents fixture.
		var raw, err = os.ReadFile(annexPath)
		if err != nil {
			return auditPresentationRow{}, err
		}
		return parseMarkdownAuditPresentationRow(string(raw), sourceID)
	case reportmodel.ReportOutputFormatPDF:
		// #nosec G304 -- the report path is returned by the controlled runtime output fixture.
		var raw, err = os.ReadFile(outcome.OutputBundle.Files[0].Path)
		if err != nil {
			return auditPresentationRow{}, err
		}
		var inspection testutil.GeneratedPDF
		inspection, err = testutil.InspectGeneratedPDF(raw)
		if err != nil {
			return auditPresentationRow{}, err
		}
		return parsePDFAuditPresentationRow(inspection, sourceID)
	default:
		return auditPresentationRow{}, os.ErrInvalid
	}
}

// parseMarkdownAuditPresentationRow extracts one Annex pipe-table row and its
// fixed semantic column positions.
// Authored by: OpenCode
func parseMarkdownAuditPresentationRow(content string, sourceID string) (auditPresentationRow, error) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "|") || !strings.Contains(line, sourceID) {
			continue
		}
		var cells = strings.Split(line, "|")
		if len(cells) <= 12 {
			return auditPresentationRow{}, os.ErrInvalid
		}
		for index := range cells {
			cells[index] = strings.TrimSpace(cells[index])
		}
		return auditPresentationRow{
			OriginalCurrency:    cells[8],
			CalculationCurrency: cells[9],
			FullLiquidation:     cells[12],
			NonEmptyCells:       runtimeflow.NonEmptyPDFCells(cells[1 : len(cells)-1]),
		}, nil
	}
	return auditPresentationRow{}, os.ErrNotExist
}

// parsePDFAuditPresentationRow extracts one semantic Annex row from PDF text
// runs, retaining empty columns so classified rows do not shift positions.
// Authored by: OpenCode
func parsePDFAuditPresentationRow(inspection testutil.GeneratedPDF, sourceID string) (auditPresentationRow, error) {
	var annexPage int
	for _, run := range inspection.TextRuns {
		if strings.Contains(run.Text, "Annex 1 - Audit") {
			annexPage = run.Page
			break
		}
	}
	if annexPage == 0 {
		return auditPresentationRow{}, os.ErrNotExist
	}

	var sourceRuns, found = runtimeflow.FindAnnexPDFSourceRuns(inspection.TextRuns, annexPage, sourceID)
	if !found {
		return auditPresentationRow{}, os.ErrNotExist
	}

	var rowRuns = runtimeflow.AnnexPDFRowRuns(inspection.TextRuns, sourceRuns)
	var cells = runtimeflow.AnnexPDFSemanticCells(rowRuns)
	if len(cells) != runtimeflow.AnnexPDFColumnCount || runtimeflow.NormalizePDFSourceID(cells[1]) != runtimeflow.NormalizePDFSourceID(sourceID) {
		return auditPresentationRow{}, os.ErrNotExist
	}
	var row = auditPresentationRow{
		OriginalCurrency:    cells[7],
		CalculationCurrency: cells[8],
		FullLiquidation:     cells[11],
		NonEmptyCells:       runtimeflow.NonEmptyPDFCells(cells),
	}
	return row, nil
}
