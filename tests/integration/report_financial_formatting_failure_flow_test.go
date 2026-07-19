// Package integration verifies renderer-scoped financial-formatting failures
// through the runtime output boundary.
// Authored by: OpenCode
package integration

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cockroachdb/apd/v3"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportFinancialFormattingFailuresStopBeforeOutputAndRetry verifies every
// FR-004a rejection class through each selected production renderer, with no
// alternate format attempt, and then completes one same-format retry.
// Authored by: OpenCode
func TestReportFinancialFormattingFailuresStopBeforeOutputAndRetry(t *testing.T) {
	var failureCases = []struct {
		name              string
		value             func() apd.Decimal
		injectedPrecision bool
		wantContext       string
	}{
		{name: "adjusted exponent below lower bound", value: func() apd.Decimal { return financialFormattingDecimalWithExponent(-100001) }, wantContext: "adjusted exponent is out of range"},
		{name: "adjusted exponent above upper bound", value: func() apd.Decimal { return financialFormattingDecimalWithExponent(100001) }, wantContext: "adjusted exponent is out of range"},
		{name: "upper bound carry", value: financialFormattingUpperBoundCarry, wantContext: "adjusted exponent exceeds range after rounding"},
		{name: "required precision above apd limit", value: func() apd.Decimal { return *apd.New(123, -2) }, injectedPrecision: true, wantContext: "required precision 2147383650 exceeds apd operational limit"},
	}

	for _, failureCase := range failureCases {
		var failureCase = failureCase
		for _, outputFormat := range []reportmodel.ReportOutputFormat{reportmodel.ReportOutputFormatMarkdown, reportmodel.ReportOutputFormatPDF} {
			var outputFormat = outputFormat
			t.Run(failureCase.name+"/"+string(outputFormat), func(t *testing.T) {
				var reportIO = testutil.NewReportIOFixture(t)
				var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
				var markdownCalls int
				var pdfCalls int
				var transformCalls int
				var pipelineOptions = runtime.ReportPipelineOptions{
					CalculatedReportTransform: func(report reportmodel.CapitalGainsReport) reportmodel.CapitalGainsReport {
						transformCalls++
						if transformCalls == 1 {
							report.YearlyNetTotal = failureCase.value()
						}
						return report
					},
					MarkdownRenderObserver: func() { markdownCalls++ },
					PDFRenderObserver:      func() { pdfCalls++ },
				}
				if failureCase.injectedPrecision {
					if outputFormat == reportmodel.ReportOutputFormatMarkdown {
						pipelineOptions.MarkdownFinancialFormatting = runtimeflow.RequiredPrecisionFailureOptions(t)
					} else {
						pipelineOptions.PDFFinancialFormatting = runtimeflow.RequiredPrecisionFailureOptions(t)
					}
				}
				var harness = runtimeflow.NewRuntimeBackedFlowHarnessWithCurrencyRateServiceAndReportPipelineOptions(
					t,
					t.TempDir(),
					runtimeflow.MustCloudSetupConfig(t),
					false,
					runtimeflow.DeterministicCurrencyRates{},
					pipelineOptions,
				)
				var fixture = testutil.DeterministicReportLedgerFixture()
				var token = "financial-formatting-" + strings.ReplaceAll(string(outputFormat), " ", "-")
				runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)
				var unlockResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
				if !unlockResult.ProtectedData.HasReadableSnapshot {
					t.Fatalf("expected readable snapshot, got %#v", unlockResult)
				}

				var request = runtimeflow.MustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, outputFormat)
				var calculator = reportcalculate.NewCalculator(runtimeflow.DeterministicCurrencyRates{})
				var baseline, err = calculator.Calculate(context.Background(), request, fixture.ProtectedActivityCache)
				if err != nil {
					t.Fatalf("calculate baseline: %v", err)
				}

				var failed = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{
					Request:      request,
					AttemptID:    "financial-formatting-failed-attempt",
					ServerOrigin: harness.Config.ServerOrigin,
				})
				if failed.Success || failed.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
					t.Fatalf("expected selected %s formatting failure, got %#v", outputFormat, failed)
				}
				if !strings.Contains(failed.Message, failureCase.wantContext) || strings.Contains(failed.Message, "synthetic-financial-format-secret") {
					t.Fatalf("expected contextual redacted formatting failure, got %q", failed.Message)
				}
				if !failureCase.injectedPrecision && !strings.Contains(failed.Message, "render yearly net total") {
					t.Fatalf("expected calculated yearly total to reach selected renderer, got %q", failed.Message)
				}
				if failureCase.injectedPrecision && !strings.Contains(failed.Message, "Bearer [REDACTED]") {
					t.Fatalf("expected injected precision secret to be redacted, got %q", failed.Message)
				}
				assertFinancialFormattingRendererCalls(t, outputFormat, markdownCalls, pdfCalls, 1)
				if strings.Contains(failed.Message, component.ReportCleartextExportDisclosureText) || strings.Contains(failed.Message, component.ReportCleartextExportDeletionGuidanceText) {
					t.Fatalf("expected TUI-owned disclosure to stay out of failure message, got %q", failed.Message)
				}
				if failed.OutputFormat != "" || failed.OutputFile != (reportmodel.ReportOutputFile{}) || failed.OutputBundle.OutputFormat != "" || len(failed.OutputBundle.Files) != 0 || !failed.OutputBundle.SavedAt.IsZero() || failed.OutputBundle.OpenRequested || failed.OutputBundle.OpenError != "" {
					t.Fatalf("expected no document, bundle, saved path, or output metadata, got %#v", failed)
				}
				var diagnosticText = failed.Diagnostic.Request.Context.FailureDetail + " " + strings.Join(failed.Diagnostic.Request.Context.FailureCauseChain, " ")
				if !strings.Contains(diagnosticText, failureCase.wantContext) || strings.Contains(diagnosticText, "synthetic-financial-format-secret") {
					t.Fatalf("expected redacted diagnostic formatting context, got %#v", failed.Diagnostic.Request.Context)
				}
				if !failureCase.injectedPrecision && !strings.Contains(diagnosticText, "render yearly net total") {
					t.Fatalf("expected calculated yearly total diagnostic context, got %#v", failed.Diagnostic.Request.Context)
				}
				if failureCase.injectedPrecision && !strings.Contains(diagnosticText, "Bearer [REDACTED]") {
					t.Fatalf("expected redacted injected precision diagnostic, got %#v", failed.Diagnostic.Request.Context)
				}
				if files, readErr := os.ReadDir(reportIO.DocumentsDir); readErr != nil || len(files) != 0 {
					t.Fatalf("expected no writer reservation or file after formatting failure, files=%#v err=%v", files, readErr)
				}
				if requests := runtimeflow.ReadOpenCommandRequests(t, openLogPath); len(requests) != 0 {
					t.Fatalf("expected no opener request after formatting failure, got %#v", requests)
				}
				runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)

				var retried = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{
					Request:      request,
					AttemptID:    "financial-formatting-successful-retry",
					ServerOrigin: harness.Config.ServerOrigin,
				})
				if !retried.Success || retried.OutputFormat != outputFormat || retried.FailureReason != runtime.ReportFailureNone {
					t.Fatalf("expected successful same-format retry, got %#v", retried)
				}
				assertFinancialFormattingRendererCalls(t, outputFormat, markdownCalls, pdfCalls, 2)
				var after, calculateErr = calculator.Calculate(context.Background(), request, fixture.ProtectedActivityCache)
				if calculateErr != nil {
					t.Fatalf("calculate post-retry model: %v", calculateErr)
				}
				assertCalculatedReportUnchanged(t, outputFormat, baseline, after)
				var savedPaths = runtimeflow.ReportOutputPaths(t, reportIO.DocumentsDir, outputFormat)
				if len(savedPaths) != expectedFinancialFormattingOutputCount(outputFormat) {
					t.Fatalf("expected selected-format output only, got %#v", savedPaths)
				}
				if requests := runtimeflow.ReadOpenCommandRequests(t, openLogPath); len(requests) != 1 {
					t.Fatalf("expected one opener request only after successful retry, got %#v", requests)
				}
				runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
			})
		}
	}
}

// financialFormattingDecimalWithExponent constructs a finite one-digit value
// that reaches the production adjusted-exponent guard without large allocation.
// Authored by: OpenCode
func financialFormattingDecimalWithExponent(exponent int32) apd.Decimal {
	var value apd.Decimal
	value.Form = apd.Finite
	value.Coeff.SetInt64(1)
	value.Exponent = exponent
	return value
}

// financialFormattingUpperBoundCarry constructs the accepted upper-bound value
// whose concrete HALF UP quantization carries into adjusted exponent 100001.
// Authored by: OpenCode
func financialFormattingUpperBoundCarry() apd.Decimal {
	var value apd.Decimal
	if _, _, err := value.SetString(strings.Repeat("9", 100001) + ".995"); err != nil {
		panic(err)
	}
	return value
}

// assertFinancialFormattingRendererCalls proves that only the selected concrete
// renderer was entered for the expected number of failure and retry attempts.
// Authored by: OpenCode
func assertFinancialFormattingRendererCalls(t testing.TB, outputFormat reportmodel.ReportOutputFormat, markdownCalls int, pdfCalls int, wantSelected int) {
	t.Helper()
	if outputFormat == reportmodel.ReportOutputFormatMarkdown {
		if markdownCalls != wantSelected || pdfCalls != 0 {
			t.Fatalf("renderer calls Markdown=%d PDF=%d, want Markdown=%d PDF=0", markdownCalls, pdfCalls, wantSelected)
		}
		return
	}
	if pdfCalls != wantSelected || markdownCalls != 0 {
		t.Fatalf("renderer calls Markdown=%d PDF=%d, want Markdown=0 PDF=%d", markdownCalls, pdfCalls, wantSelected)
	}
}

// expectedFinancialFormattingOutputCount returns the inherited selected-format
// bundle cardinality used to prove no alternate renderer was invoked.
// Authored by: OpenCode
func expectedFinancialFormattingOutputCount(outputFormat reportmodel.ReportOutputFormat) int {
	if outputFormat == reportmodel.ReportOutputFormatMarkdown {
		return 2
	}
	return 1
}
