// Package integration verifies the concrete PDF finalization boundary through
// the runtime report service.
// Authored by: OpenCode
package integration

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

// TestReportPDFFinalizationFailureLeavesOutputBoundaryAndRetries verifies that
// a concrete PDF byte-finalization fault survives through runtime, discards
// partial bytes before output, redacts its secret-bearing cause, and succeeds on
// one later PDF attempt through the same report service.
// Authored by: OpenCode
func TestReportPDFFinalizationFailureLeavesOutputBoundaryAndRetries(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = runtimeflow.InstallOpenCommandRecorder(t, 0)
	var finalizationCause = errors.New("Bearer synthetic-pdf-finalization-secret")
	var finalizerCalls int
	var fallbackCalls int
	var finalizer = func(defaultFinalizer func() ([]byte, error)) ([]byte, error) {
		finalizerCalls++
		if finalizerCalls == 1 {
			return []byte("%PDF-partial-secret-bearing-attempt"), finalizationCause
		}
		fallbackCalls++
		return defaultFinalizer()
	}
	var harness = runtimeflow.NewRuntimeBackedFlowHarnessWithCurrencyRateServiceAndPDFByteFinalizer(
		t,
		t.TempDir(),
		runtimeflow.MustCloudSetupConfig(t),
		false,
		runtimeflow.DeterministicCurrencyRates{},
		finalizer,
	)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var token = strings.Join([]string{"pdf-finalization", "retry", "token"}, "-")
	runtimeflow.SeedProtectedSnapshot(t, harness, token, fixture.ProtectedActivityCache)
	var unlockResult = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !unlockResult.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot for PDF finalization journey, got %#v", unlockResult)
	}

	var request = runtimeflow.MustIntegrationReportRequestForFormat(t, fixture.PrimaryReportYear, reportmodel.ReportOutputFormatPDF)
	var failed = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{
		Request:      request,
		AttemptID:    "pdf-finalization-failed-attempt",
		ServerOrigin: harness.Config.ServerOrigin,
	})
	if failed.Success || failed.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
		t.Fatalf("expected concrete PDF finalization failure, got %#v", failed)
	}
	if finalizerCalls != 1 || fallbackCalls != 0 {
		t.Fatalf("expected one failed finalization without fallback, finalizer calls=%d fallback calls=%d", finalizerCalls, fallbackCalls)
	}
	if failed.OutputFormat != "" || failed.OutputFile != (reportmodel.ReportOutputFile{}) || failed.OutputBundle.OutputFormat != "" || len(failed.OutputBundle.Files) != 0 || !failed.OutputBundle.SavedAt.IsZero() || failed.OutputBundle.OpenRequested || failed.OutputBundle.OpenError != "" {
		t.Fatalf("expected failed finalization to return no output metadata or path, got %#v", failed)
	}
	if !strings.Contains(failed.Message, "PDF byte finalization failed") || !strings.Contains(failed.Message, "Bearer [REDACTED]") || strings.Contains(failed.Message, finalizationCause.Error()) {
		t.Fatalf("expected redacted PDF finalization context, got %q", failed.Message)
	}
	var diagnosticContext = failed.Diagnostic.Request.Context
	if !failed.Diagnostic.Eligible || !strings.Contains(diagnosticContext.FailureDetail, "could not render") || !strings.Contains(strings.Join(diagnosticContext.FailureCauseChain, " | "), "PDF byte finalization failed") || !strings.Contains(strings.Join(diagnosticContext.FailureCauseChain, " | "), "Bearer [REDACTED]") || strings.Contains(diagnosticContext.FailureDetail, finalizationCause.Error()) || strings.Contains(strings.Join(diagnosticContext.FailureCauseChain, " | "), finalizationCause.Error()) {
		t.Fatalf("expected redacted diagnostic finalization context, got %#v", diagnosticContext)
	}
	if strings.Contains(failed.Message, component.ReportCleartextExportDisclosureText) || strings.Contains(failed.Message, component.ReportCleartextExportDeletionGuidanceText) {
		t.Fatalf("expected report service failure to leave cleartext disclosure ownership to the TUI, got %q", failed.Message)
	}

	var entries, err = os.ReadDir(reportIO.DocumentsDir)
	if err != nil {
		t.Fatalf("read Documents directory after failed finalization: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no reservation, writer artifact, or alternate-format file after failed finalization, got %#v", entries)
	}
	if openerRequests := runtimeflow.ReadOpenCommandRequests(t, openLogPath); len(openerRequests) != 0 {
		t.Fatalf("expected no opener request after failed finalization, got %#v", openerRequests)
	}
	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)

	var retried = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{
		Request:      request,
		AttemptID:    "pdf-finalization-successful-retry",
		ServerOrigin: harness.Config.ServerOrigin,
	})
	if !retried.Success || retried.FailureReason != runtime.ReportFailureNone || retried.OutputFormat != reportmodel.ReportOutputFormatPDF {
		t.Fatalf("expected successful PDF retry through the same service, got %#v", retried)
	}
	if finalizerCalls != 2 || fallbackCalls != 1 {
		t.Fatalf("expected one failed and one concrete successful finalization, finalizer calls=%d fallback calls=%d", finalizerCalls, fallbackCalls)
	}
	if len(retried.OutputBundle.Files) != 1 || retried.OutputFile.Path == "" {
		t.Fatalf("expected one PDF output metadata record after retry, got %#v", retried)
	}
	var info, statErr = os.Stat(retried.OutputFile.Path)
	if statErr != nil {
		t.Fatalf("stat successful PDF retry %q: %v", retried.OutputFile.Path, statErr)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("successful PDF retry mode = %#o, want 0600", info.Mode().Perm())
	}
	// #nosec G304 -- the successful path is returned by the controlled output fixture.
	var payload, readErr = os.ReadFile(retried.OutputFile.Path)
	if readErr != nil {
		t.Fatalf("read successful PDF retry %q: %v", retried.OutputFile.Path, readErr)
	}
	var inspection, inspectErr = testutil.InspectGeneratedPDF(payload)
	if inspectErr != nil {
		t.Fatalf("inspect successful PDF retry: %v", inspectErr)
	}
	if !inspection.ContainsSearchableText("Gains-And-Losses Summary") {
		t.Fatalf("expected successful retry to contain the report summary")
	}
	if markdownFiles := runtimeflow.AllMarkdownFiles(t, reportIO.DocumentsDir); len(markdownFiles) != 0 {
		t.Fatalf("expected PDF-only retry without alternate Markdown renderer, got %#v", markdownFiles)
	}
	if pdfFiles := runtimeflow.PDFFiles(t, reportIO.DocumentsDir); len(pdfFiles) != 1 || pdfFiles[0] != retried.OutputFile.Path {
		t.Fatalf("expected one successful PDF path %q, got %#v", retried.OutputFile.Path, pdfFiles)
	}
	if openerRequests := runtimeflow.ReadOpenCommandRequests(t, openLogPath); len(openerRequests) != 1 || openerRequests[0] != retried.OutputFile.Path {
		t.Fatalf("expected one opener request for successful retry %q, got %#v", retried.OutputFile.Path, openerRequests)
	}
	runtimeflow.AssertNoCleartextReportInAppStorage(t, harness.BaseDir)
}
