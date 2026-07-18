// Package integration verifies runtime-backed report presentation parity and
// pre/post-render model integrity for the current slice.
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

// TestReportValuePresentationPreservesAUD001AcrossMarkdownAndPDF verifies that
// both runtime-selected renderers consume the same mixed-currency cache without
// changing the complete calculated report model or its output bundle shape.
// Authored by: OpenCode
func TestReportValuePresentationPreservesAUD001AcrossMarkdownAndPDF(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var cache = runtimeflow.MixedCurrencyConversionProtectedActivityCache(t, 6)
	var harness = runtimeflow.NewRuntimeBackedFlowHarness(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false)
	var token = "report-value-presentation-token"

	runtimeflow.SeedProtectedSnapshot(t, harness, token, cache)
	var contextResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !contextResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", contextResult)
	}

	var calculator = reportcalculate.NewCalculator(runtimeflow.DeterministicCurrencyRates{})
	var formats = []reportmodel.ReportOutputFormat{
		reportmodel.ReportOutputFormatMarkdown,
		reportmodel.ReportOutputFormatPDF,
	}
	for _, outputFormat := range formats {
		var request = runtimeflow.MustIntegrationReportRequestForFormat(t, 2024, outputFormat)
		var before, err = calculator.Calculate(context.Background(), request, cache)
		if err != nil {
			t.Fatalf("calculate AUD-001 baseline for %s: %v", outputFormat, err)
		}
		if len(before.RateSources) == 0 || len(before.AuditAnnex.ConversionAuditEntries) == 0 {
			t.Fatalf("expected mixed-currency baseline to retain rate metadata and conversion entries, got %#v", before)
		}

		var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{Request: request})
		if !outcome.Success {
			t.Fatalf("expected %s report generation success, got %#v", outputFormat, outcome)
		}

		var after reportmodel.CapitalGainsReport
		after, err = calculator.Calculate(context.Background(), request, cache)
		if err != nil {
			t.Fatalf("calculate post-render AUD-001 model for %s: %v", outputFormat, err)
		}
		assertAUD001ReportEqual(t, outputFormat, before, after)
		assertReportValuePresentationBundle(t, reportIO.DocumentsDir, outputFormat, outcome)
	}

	var openerRequests = runtimeflow.ReadOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 2 {
		t.Fatalf("expected one opener request for each selected format, got %#v", openerRequests)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// assertAUD001ReportEqual compares the complete calculated report model so all
// exact values, quantities, rates, rate metadata, currencies, and inclusion
// states remain tied to the pre-presentation baseline.
// Authored by: OpenCode
func assertAUD001ReportEqual(t *testing.T, outputFormat reportmodel.ReportOutputFormat, before reportmodel.CapitalGainsReport, after reportmodel.CapitalGainsReport) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("AUD-001 report model changed after %s rendering: before=%#v after=%#v", outputFormat, before, after)
	}
}

// assertReportValuePresentationBundle verifies format-specific output cardinality
// and selected release-ready visible values for one successful runtime attempt.
// Authored by: OpenCode
func assertReportValuePresentationBundle(t *testing.T, documentsDir string, outputFormat reportmodel.ReportOutputFormat, outcome runtime.ReportOutcome) {
	t.Helper()
	if outcome.OutputFormat != outputFormat {
		t.Fatalf("expected outcome format %q, got %q", outputFormat, outcome.OutputFormat)
	}
	if err := outcome.OutputBundle.Validate(); err != nil {
		t.Fatalf("validate %s output bundle: %v", outputFormat, err)
	}
	if len(outcome.OutputBundle.Files) == 0 || outcome.OutputFile.Path != outcome.OutputBundle.Files[0].Path {
		t.Fatalf("expected primary output file to match the bundle, got %#v", outcome)
	}

	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		if len(outcome.OutputBundle.Files) != 2 {
			t.Fatalf("expected Markdown main-plus-annex bundle, got %#v", outcome.OutputBundle.Files)
		}
		var files = runtimeflow.AllMarkdownFiles(t, documentsDir)
		if len(files) != 2 {
			t.Fatalf("expected two Markdown files, got %#v", files)
		}
		var reportPath, annexPath = runtimeflow.MarkdownBundlePaths(t, files)
		assertReportOutputFilePaths(t, outcome, documentsDir, reportmodel.ReportDocumentRoleMain, reportmodel.ReportMediaTypeMarkdown)
		assertReportOutputFilePaths(t, outcome, documentsDir, reportmodel.ReportDocumentRoleAnnex, reportmodel.ReportMediaTypeMarkdown)

		// #nosec G304 -- paths are created in the test-owned Documents fixture.
		var rawReport, err = os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read generated Markdown report %q: %v", reportPath, err)
		}
		var reportText = string(rawReport)
		if !strings.Contains(reportText, "| mixed-usd-buy-2024-000 | BUY | 1 | 10.00 | 10.00 | 1.00 |") {
			t.Fatalf("expected US1 two-place Markdown values, got %q", reportText)
		}
		if strings.Count(reportText, testutil.ReportPresentationLegalWarningText) != 1 {
			t.Fatalf("expected one legal-use warning in Markdown main report, got %q", reportText)
		}
		// #nosec G304 -- annexPath is created in the test-owned Documents fixture.
		var rawAnnex, annexErr = os.ReadFile(annexPath)
		if annexErr != nil {
			t.Fatalf("read generated Markdown annex %q: %v", annexPath, annexErr)
		}
		if strings.Contains(string(rawAnnex), testutil.ReportPresentationLegalWarningText) {
			t.Fatalf("expected legal-use warning to be excluded from Markdown Annex 1")
		}
	case reportmodel.ReportOutputFormatPDF:
		if len(outcome.OutputBundle.Files) != 1 {
			t.Fatalf("expected one combined PDF bundle, got %#v", outcome.OutputBundle.Files)
		}
		var files = runtimeflow.PDFFiles(t, documentsDir)
		if len(files) != 1 {
			t.Fatalf("expected one PDF file, got %#v", files)
		}
		assertReportOutputFilePaths(t, outcome, documentsDir, reportmodel.ReportDocumentRoleCombined, reportmodel.ReportMediaTypePDF)
		// #nosec G304 -- PDF path is created in the test-owned Documents fixture.
		var rawPDF, err = os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("read generated PDF %q: %v", files[0], err)
		}
		var inspection, inspectErr = testutil.InspectGeneratedPDF(rawPDF)
		if inspectErr != nil {
			t.Fatalf("inspect generated PDF: %v", inspectErr)
		}
		if !inspection.ContainsSearchableText("10.00") || !inspection.ContainsSearchableText(testutil.ReportPresentationLegalWarningText) {
			t.Fatalf("expected US1 two-place value and legal-use warning in PDF, got %q", inspection.SearchableText)
		}
	default:
		t.Fatalf("unsupported report output format %q", outputFormat)
	}
}

// assertReportOutputFilePaths verifies one role and media type in a successful
// output bundle remain inside the test-owned Documents directory.
// Authored by: OpenCode
func assertReportOutputFilePaths(t *testing.T, outcome runtime.ReportOutcome, documentsDir string, role reportmodel.ReportDocumentRole, mediaType string) {
	t.Helper()
	for _, file := range outcome.OutputBundle.Files {
		if file.Role != role {
			continue
		}
		if file.MediaType != mediaType {
			t.Fatalf("expected %s output role %q to use media type %q, got %#v", outcome.OutputFormat, role, mediaType, file)
		}
		testutil.AssertPathWithin(t, file.Path, documentsDir)
		testutil.AssertRegularFile(t, file.Path)
		return
	}
	t.Fatalf("expected %s output bundle to contain role %q, got %#v", outcome.OutputFormat, role, outcome.OutputBundle.Files)
}
