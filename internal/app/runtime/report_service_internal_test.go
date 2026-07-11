// Package runtime verifies package-local report-service guardrails and failure
// classification.
// Authored by: OpenCode
package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
)

// TestReportServiceGenerateCoversAvailabilityAndPersistenceOutcomes verifies
// runtime report-service classification, success, and opener-warning behavior.
// Authored by: OpenCode
func TestReportServiceGenerateCoversAvailabilityAndPersistenceOutcomes(t *testing.T) {
	t.Run("fails when no readable cache is unlocked", func(t *testing.T) {
		t.Parallel()

		var request = reportRequestFixture(t, 2024)
		var service = &reportService{}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.Success {
			t.Fatalf("expected failure outcome, got %#v", outcome)
		}
		if outcome.FailureReason != ReportFailureNoSyncedDataAvailable {
			t.Fatalf("expected no synced data failure, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "unlock or sync data first") {
			t.Fatalf("expected actionable message, got %q", outcome.Message)
		}
	})

	t.Run("fails when readable cache has no reportable years", func(t *testing.T) {
		t.Parallel()

		var request = reportRequestFixture(t, 2024)
		var snapshots = reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache)
		var cache, _ = snapshots.ReadableProtectedActivityCache()
		cache.AvailableReportYears = nil
		snapshots.SetActiveSnapshot(snapshotstore.Candidate{SnapshotID: "snapshot-1"}, snapshotmodel.Payload{ProtectedActivityCache: cache})

		var service = &reportService{snapshots: snapshots}
		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.FailureReason != ReportFailureNoReportableYearsAvailable {
			t.Fatalf("expected no reportable years failure, got %#v", outcome)
		}
	})

	t.Run("returns unsupported calculation when selected year is unavailable", func(t *testing.T) {
		t.Parallel()

		var request = reportRequestFixture(t, 2030)
		var service = &reportService{snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache)}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.FailureReason != ReportFailureUnsupportedReportCalculation {
			t.Fatalf("expected unsupported calculation failure, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "available report years") {
			t.Fatalf("expected available-year guidance, got %q", outcome.Message)
		}
	})

	t.Run("returns unsupported calculation when request validation fails", func(t *testing.T) {
		t.Parallel()

		var request = reportmodel.ReportRequest{
			Year:            2024,
			CostBasisMethod: reportmodel.CostBasisMethod("unsupported"),
			RequestedAt:     time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC),
		}
		var service = &reportService{snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache)}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.FailureReason != ReportFailureUnsupportedReportCalculation {
			t.Fatalf("expected unsupported calculation failure, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "Could not generate the report request") {
			t.Fatalf("expected request-validation guidance, got %q", outcome.Message)
		}
	})

	t.Run("returns unsupported calculation when rendering fails", func(t *testing.T) {
		t.Parallel()

		var request = reportRequestFixture(t, 2024)
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(_ context.Context, request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			renderBundle: func(reportmodel.ReportOutputFormat, reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
				return nil, errors.New(" render boom ")
			},
			writeBundle: func(reportmodel.ReportOutputFormat, []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
				t.Fatalf("write should not be called after render failure")
				return reportmodel.ReportOutputBundle{}, nil
			},
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.FailureReason != ReportFailureUnsupportedReportCalculation {
			t.Fatalf("expected render failure to map to unsupported calculation, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "Could not render the 2024 FIFO report: render boom") {
			t.Fatalf("expected trimmed render failure message, got %q", outcome.Message)
		}
		if !outcome.Diagnostic.Eligible || len(outcome.Diagnostic.Request.Context.FailureCauseChain) != 2 {
			t.Fatalf("expected wrapped render diagnostic chain, got %#v", outcome.Diagnostic)
		}
		if !strings.Contains(outcome.Diagnostic.Request.Context.FailureCauseChain[0], "could not render the 2024 FIFO report") || outcome.Diagnostic.Request.Context.FailureCauseChain[1] != "render boom" {
			t.Fatalf("expected ordered render failure cause chain, got %#v", outcome.Diagnostic.Request.Context.FailureCauseChain)
		}
	})

	t.Run("saves and requests automatic opening on success", func(t *testing.T) {
		var request = reportRequestFixture(t, 2024)
		var fixture = testutil.NewReportIOFixture(t)
		var opener = testutil.NewOpenPathSpy(nil)
		var savedPath string
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(_ context.Context, request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			renderBundle: func(_ reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
				return reportDocumentBundleFixture(t, report), nil
			},
			writeBundle: func(_ reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
				savedPath = filepath.Join(fixture.DocumentsDir, "ghostfolio-capital-gains-2024-fifo-2026-05-20_15-04-05.md")
				return reportOutputBundleFixture(t, fixture.DocumentsDir, savedPath, documents[0].GeneratedAt), nil
			},
			open: opener.Open,
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if !outcome.Success || outcome.FailureReason != ReportFailureNone {
			t.Fatalf("expected successful outcome, got %#v", outcome)
		}
		if outcome.OutputFile.Path != savedPath {
			t.Fatalf("expected saved output with open request, got %#v", outcome.OutputFile)
		}
		if opener.CallCount() != 1 || len(opener.Paths()) != 1 || opener.Paths()[0] != savedPath {
			t.Fatalf("expected one opener request for %q, got %#v", savedPath, opener.Paths())
		}
		if !strings.Contains(outcome.Message, "delete") || !strings.Contains(outcome.Message, savedPath) {
			t.Fatalf("expected saved-path removal guidance, got %q", outcome.Message)
		}
	})

	t.Run("preserves success with warning when automatic open fails", func(t *testing.T) {
		var request = reportRequestFixture(t, 2024)
		var fixture = testutil.NewReportIOFixture(t)
		var opener = testutil.NewOpenPathSpy(errors.New("open boom"))
		var savedPath = filepath.Join(fixture.DocumentsDir, "report.md")
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(_ context.Context, request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			renderBundle: func(_ reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
				return reportDocumentBundleFixture(t, report), nil
			},
			writeBundle: func(_ reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
				return reportOutputBundleFixture(t, fixture.DocumentsDir, savedPath, documents[0].GeneratedAt), nil
			},
			open: opener.Open,
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if !outcome.Success {
			t.Fatalf("expected non-fatal warning outcome, got %#v", outcome)
		}
		if outcome.FailureReason != ReportFailureAutomaticOpenFailedAfterSave {
			t.Fatalf("expected automatic-open warning, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "open boom") {
			t.Fatalf("expected preserved saved file with open error, got %#v", outcome.OutputFile)
		}
		if opener.CallCount() != 1 {
			t.Fatalf("expected one opener request, got %d", opener.CallCount())
		}
	})

	t.Run("fails with warning reason when opener is unavailable", func(t *testing.T) {
		var request = reportRequestFixture(t, 2024)
		var fixture = testutil.NewReportIOFixture(t)
		var savedPath = filepath.Join(fixture.DocumentsDir, "report.md")
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(_ context.Context, request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			renderBundle: func(_ reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
				return reportDocumentBundleFixture(t, report), nil
			},
			writeBundle: func(_ reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
				return reportOutputBundleFixture(t, fixture.DocumentsDir, savedPath, documents[0].GeneratedAt), nil
			},
			open: nil,
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if !outcome.Success {
			t.Fatalf("expected nil opener to preserve saved-output success, got %#v", outcome)
		}
		if outcome.FailureReason != ReportFailureAutomaticOpenFailedAfterSave {
			t.Fatalf("expected automatic-open warning reason, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "automatic opening is unavailable in this runtime") {
			t.Fatalf("expected unavailable-opener detail, got %q", outcome.Message)
		}
	})

	t.Run("fails when saved output cannot be finalized before open request", func(t *testing.T) {
		var request = reportRequestFixture(t, 2024)
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(_ context.Context, request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			renderBundle: func(_ reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
				return reportDocumentBundleFixture(t, report), nil
			},
			writeBundle: func(_ reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
				return reportmodel.ReportOutputBundle{OutputFormat: reportmodel.ReportOutputFormatMarkdown, Files: []reportmodel.ReportOutputFile{{DocumentsDirectory: "/tmp/docs", Path: "/tmp/docs/report.md", Role: reportmodel.ReportDocumentRoleMain, MediaType: reportmodel.ReportMediaTypeMarkdown, SavedAt: documents[0].GeneratedAt}}, SavedAt: documents[0].GeneratedAt}, nil
			},
			open: func(string) error { return nil },
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.Success {
			t.Fatalf("expected output finalization failure, got %#v", outcome)
		}
		if outcome.FailureReason != ReportFailureReportFileWriteFailed {
			t.Fatalf("expected write-failure classification, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "Could not finalize the saved report result") {
			t.Fatalf("expected finalization failure detail, got %q", outcome.Message)
		}
	})

	t.Run("classifies documents resolution failure", func(t *testing.T) {
		t.Parallel()

		var request = reportRequestFixture(t, 2024)
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(_ context.Context, request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			renderBundle: func(_ reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
				return reportDocumentBundleFixture(t, report), nil
			},
			writeBundle: func(reportmodel.ReportOutputFormat, []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
				return reportmodel.ReportOutputBundle{}, reportoutputFailure(
					reportoutput.FailureCategoryDocumentsDirectoryUnavailable,
					"resolve user home directory: boom",
				)
			},
			open: func(string) error { return nil },
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.FailureReason != ReportFailureDocumentsFolderUnavailable {
			t.Fatalf("expected documents folder failure, got %#v", outcome)
		}
	})

	t.Run("classifies final write failure", func(t *testing.T) {
		t.Parallel()

		var request = reportRequestFixture(t, 2024)
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(_ context.Context, request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			renderBundle: func(_ reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
				return reportDocumentBundleFixture(t, report), nil
			},
			writeBundle: func(reportmodel.ReportOutputFormat, []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
				return reportmodel.ReportOutputBundle{}, reportoutputFailure(
					reportoutput.FailureCategoryReportFileWriteFailed,
					"write report file \"/tmp/report.md\": permission denied",
				)
			},
			open: func(string) error { return nil },
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.FailureReason != ReportFailureReportFileWriteFailed {
			t.Fatalf("expected report file write failure, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "partial file") {
			t.Fatalf("expected cleanup guidance, got %q", outcome.Message)
		}
		if !outcome.Diagnostic.Eligible || len(outcome.Diagnostic.Request.Context.FailureCauseChain) != 2 {
			t.Fatalf("expected wrapped write diagnostic chain, got %#v", outcome.Diagnostic)
		}
		if outcome.Diagnostic.Request.Context.FailureCauseChain[0] != "could not save the report file" || !strings.Contains(outcome.Diagnostic.Request.Context.FailureCauseChain[1], "write report file") || !strings.Contains(outcome.Diagnostic.Request.Context.FailureCauseChain[1], "permission denied") {
			t.Fatalf("expected ordered write failure cause chain, got %#v", outcome.Diagnostic.Request.Context.FailureCauseChain)
		}
	})

	t.Run("production calculation failure exposes pending diagnostics with original persisted record", func(t *testing.T) {
		var request = reportRequestFixture(t, 2025)
		var offendingRecord = reportFailureActivityRecordFixture(t)
		var service = &reportService{
			snapshots:         reportSnapshotLifecycleWithCache(syncmodel.ProtectedActivityCache{ActivityCount: 1, AvailableReportYears: []int{2025}, Activities: []syncmodel.ActivityRecord{offendingRecord}}),
			diagnosticReports: newDiagnosticReportService(t.TempDir()),
			calculate: func(context.Context, reportmodel.ReportRequest, syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return reportmodel.CapitalGainsReport{}, runtimeDiagnosticCarrierError{
					context: syncmodel.DiagnosticContext{
						FailureDetail:           "incomplete context (asset \"BTC\", source \"report-failure-buy-1\")",
						FailureCauseChain:       []string{"incomplete context (asset \"BTC\", source \"report-failure-buy-1\")"},
						OffendingActivityRecord: &offendingRecord,
					},
				}
			},
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request, ServerOrigin: "https://ghostfol.io"})
		if outcome.FailureReason != ReportFailureUnsupportedReportCalculation || !outcome.Diagnostic.Eligible {
			t.Fatalf("expected diagnostics-eligible calculation failure, got %#v", outcome)
		}
		if outcome.Diagnostic.Path != "" {
			t.Fatalf("expected production mode to defer diagnostic creation, got %#v", outcome.Diagnostic)
		}
		if outcome.Diagnostic.Request.Context.OffendingActivityRecord == nil || outcome.Diagnostic.Request.Context.OffendingActivityRecord.SourceID != offendingRecord.SourceID {
			t.Fatalf("expected original persisted record in diagnostics request, got %#v", outcome.Diagnostic.Request.Context)
		}
	})

	t.Run("explicit development mode auto-writes report diagnostics with explicit null fields", func(t *testing.T) {
		var baseDir = t.TempDir()
		var request = reportRequestFixture(t, 2025)
		var offendingRecord = reportFailureActivityRecordFixture(t)
		var service = &reportService{
			snapshots:         reportSnapshotLifecycleWithCache(syncmodel.ProtectedActivityCache{ActivityCount: 1, AvailableReportYears: []int{2025}, Activities: []syncmodel.ActivityRecord{offendingRecord}}),
			allowDevHTTP:      true,
			diagnosticReports: newDiagnosticReportService(baseDir),
			calculate: func(context.Context, reportmodel.ReportRequest, syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return reportmodel.CapitalGainsReport{}, runtimeDiagnosticCarrierError{
					context: syncmodel.DiagnosticContext{
						FailureDetail:           "incomplete context (asset \"BTC\", source \"report-failure-buy-1\")",
						FailureCauseChain:       []string{"incomplete context (asset \"BTC\", source \"report-failure-buy-1\")"},
						OffendingActivityRecord: &offendingRecord,
					},
				}
			},
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request, ServerOrigin: "https://ghostfol.io", ExplicitDevelopmentMode: true, AttemptID: "attempt-report-1"})
		if outcome.Diagnostic.Path == "" {
			t.Fatalf("expected explicit development mode to auto-write diagnostics, got %#v", outcome)
		}
		var raw, err = os.ReadFile(outcome.Diagnostic.Path)
		if err != nil {
			t.Fatalf("read report diagnostic artifact: %v", err)
		}
		var text = string(raw)
		if !strings.Contains(text, `"failure_category": "unsupported report calculation"`) {
			t.Fatalf("expected report failure category in artifact, got %q", text)
		}
		if !strings.Contains(text, `"offending_activity_record"`) || !strings.Contains(text, `"order_currency": null`) || !strings.Contains(text, `"asset_profile_currency": null`) || !strings.Contains(text, `"source_scope": null`) {
			t.Fatalf("expected original persisted record with explicit nulls, got %q", text)
		}
		if strings.Contains(text, `"selected_currency_context"`) || strings.Contains(text, `"activity_currency"`) {
			t.Fatalf("expected no derived report-input fields in artifact, got %q", text)
		}
	})
}

// TestReportServiceHelperFunctionsCoverRemainingBranches verifies direct helper
// formatting and cache-read branches not reached through the full service flow.
// Authored by: OpenCode
func TestReportServiceHelperFunctionsCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	var request = reportRequestFixture(t, 2024)
	var service = &reportService{
		snapshots:     reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
		currencyRates: reportProviderCategoryRateService{},
	}

	var cache, outcome, ok = service.readAvailableCache(request)
	if !ok || cache.ActivityCount == 0 || outcome.Success || outcome.Message != "" || outcome.FailureReason != ReportFailureNone || outcome.Request != (reportmodel.ReportRequest{}) || outcome.OutputFile != (reportmodel.ReportOutputFile{}) || outcome.Attempt != (SyncAttempt{}) || outcome.Diagnostic.Eligible || outcome.Diagnostic.Path != "" || outcome.Diagnostic.GenerationMessage != "" {
		t.Fatalf("expected available cache read to succeed, got ok=%v cache=%#v outcome=%#v", ok, cache, outcome)
	}

	var message = service.reportCalculationFailureMessage(request, errors.New(" calculation boom "))
	if !strings.Contains(message, "Could not generate the 2024 FIFO report: calculation boom") {
		t.Fatalf("expected trimmed calculation failure message, got %q", message)
	}
	var conversionErr = reportmodel.NewCalculationError(
		reportmodel.CalculationErrorKindActivityInput,
		"could not resolve currency conversion rate",
		"late-gbp-buy",
		"FAIL",
		testConversionFailureContextCause{context: reportcalculate.ConversionFailureContext{
			SourceID:           "late-gbp-buy",
			SourceCurrency:     "GBP",
			ReportBaseCurrency: "USD",
			ActivityDate:       time.Date(2024, time.June, 13, 0, 0, 0, 0, time.UTC),
			Reason:             "missing_rate",
		}},
	)
	var conversionMessage = service.reportCalculationFailureMessage(request, conversionErr)
	for _, expected := range []string{"Conversion Failure Context", "Source ID: late-gbp-buy", "Source Currency: GBP", "Report Base Currency: USD", "Activity Date: 2024-06-13", "Failure Reason: missing_rate", "Provider Category: federal_reserve_h10"} {
		if !strings.Contains(conversionMessage, expected) {
			t.Fatalf("expected conversion failure message to contain %q, got %q", expected, conversionMessage)
		}
	}
	if !reportDiagnosticEligible(ReportFailureUnsupportedReportCalculation) {
		t.Fatalf("expected unsupported calculation to remain diagnostic eligible")
	}
	if !reportDiagnosticEligible(ReportFailureDocumentsFolderUnavailable) {
		t.Fatalf("expected documents-folder failure to remain diagnostic eligible")
	}
	if !reportDiagnosticEligible(ReportFailureReportFileWriteFailed) {
		t.Fatalf("expected write failure to remain diagnostic eligible")
	}
	if reportDiagnosticEligible(ReportFailureAutomaticOpenFailedAfterSave) {
		t.Fatalf("expected automatic-open warning to be diagnostic ineligible")
	}

	var zeroAttempt = reportAttempt(ReportGenerationRequest{Request: reportmodel.ReportRequest{}})
	if zeroAttempt.StartedAt.IsZero() || zeroAttempt.CompletedAt.IsZero() {
		t.Fatalf("expected zero request time to fall back to current timestamps, got %#v", zeroAttempt)
	}

	var carrierContext = syncmodel.DiagnosticContext{FailureDetail: "carrier detail", FailureCauseChain: []string{"carrier detail", "carrier inner detail"}}
	var fromCarrier = reportDiagnosticContextFromError(runtimeDiagnosticCarrierError{context: carrierContext})
	if fromCarrier.FailureStage != carrierContext.FailureStage || fromCarrier.FailureDetail != carrierContext.FailureDetail || len(fromCarrier.FailureCauseChain) != len(carrierContext.FailureCauseChain) || len(fromCarrier.Records) != len(carrierContext.Records) || fromCarrier.OffendingActivityRecord != carrierContext.OffendingActivityRecord {
		t.Fatalf("expected carrier context to be preserved, got %#v", fromCarrier)
	}

	var fallbackCarrier = reportDiagnosticContextFromError(runtimeDiagnosticCarrierError{context: syncmodel.DiagnosticContext{}})
	if fallbackCarrier.FailureDetail != "carrier boom" {
		t.Fatalf("expected empty carrier detail to fall back to error text, got %#v", fallbackCarrier)
	}
	if len(fallbackCarrier.FailureCauseChain) != 1 || fallbackCarrier.FailureCauseChain[0] != "carrier boom" {
		t.Fatalf("expected empty carrier cause chain to fall back to error chain, got %#v", fallbackCarrier)
	}

	var fromPlain = reportDiagnosticContextFromError(fmt.Errorf("plain detail: %w", errors.New("lower detail")))
	if fromPlain.FailureDetail != "plain detail" {
		t.Fatalf("expected plain error detail fallback, got %#v", fromPlain)
	}
	if len(fromPlain.FailureCauseChain) != 2 || fromPlain.FailureCauseChain[0] != "plain detail" || fromPlain.FailureCauseChain[1] != "lower detail" {
		t.Fatalf("expected plain wrapped cause chain fallback, got %#v", fromPlain)
	}

	var nonEligibleOutcome = service.reportFailureOutcome(
		context.Background(),
		ReportGenerationRequest{Request: request, ServerOrigin: "https://ghostfol.io"},
		SyncAttempt{AttemptID: "attempt-non-eligible"},
		ReportFailureAutomaticOpenFailedAfterSave,
		"warning",
		syncmodel.DiagnosticContext{FailureDetail: "warning"},
	)
	if nonEligibleOutcome.Diagnostic.Eligible || nonEligibleOutcome.Diagnostic.Path != "" || nonEligibleOutcome.Diagnostic.Request.FailureReason != SyncFailureNone || nonEligibleOutcome.Diagnostic.Request.FailureCategory != ReportFailureNone || nonEligibleOutcome.Diagnostic.Request.ServerOrigin != "" || nonEligibleOutcome.Diagnostic.Request.Attempt != (SyncAttempt{}) || nonEligibleOutcome.Diagnostic.Request.Context.FailureStage != "" || nonEligibleOutcome.Diagnostic.Request.Context.FailureDetail != "" || len(nonEligibleOutcome.Diagnostic.Request.Context.FailureCauseChain) != 0 || len(nonEligibleOutcome.Diagnostic.Request.Context.Records) != 0 || nonEligibleOutcome.Diagnostic.Request.Context.OffendingActivityRecord != nil || nonEligibleOutcome.Diagnostic.Request.RedactFinancialValues || nonEligibleOutcome.Diagnostic.Request.ExplicitDevelopmentMode {
		t.Fatalf("expected non-eligible report failure to omit diagnostic state, got %#v", nonEligibleOutcome.Diagnostic)
	}

	var eligibleOutcome = service.reportFailureOutcome(
		context.Background(),
		ReportGenerationRequest{Request: request, ServerOrigin: "https://ghostfol.io", ExplicitDevelopmentMode: true},
		SyncAttempt{AttemptID: "attempt-eligible"},
		ReportFailureReportFileWriteFailed,
		"write failed",
		syncmodel.DiagnosticContext{FailureDetail: "write failed"},
	)
	if !eligibleOutcome.Diagnostic.Eligible {
		t.Fatalf("expected eligible report failure to expose diagnostic state")
	}
	if eligibleOutcome.Diagnostic.Request.FailureCategory != ReportFailureReportFileWriteFailed {
		t.Fatalf("expected diagnostic request to preserve failure category, got %#v", eligibleOutcome.Diagnostic.Request)
	}
	if !eligibleOutcome.Diagnostic.Request.ExplicitDevelopmentMode {
		t.Fatalf("expected explicit development mode to flow into diagnostic request")
	}
	if eligibleOutcome.Diagnostic.GenerationMessage == "" {
		t.Fatalf("expected explicit development mode to attempt immediate diagnostic generation, got %#v", eligibleOutcome.Diagnostic)
	}
	if reason := reportWriteFailureReason(reportoutputFailure(reportoutput.FailureCategoryDocumentsDirectoryUnavailable, "documents path: missing")); reason != ReportFailureDocumentsFolderUnavailable {
		t.Fatalf("expected documents-path text to classify as folder unavailable, got %q", reason)
	}
	if reason := reportWriteFailureReason(reportoutputFailure(reportoutput.FailureCategoryReportFileWriteFailed, "permission denied")); reason != ReportFailureReportFileWriteFailed {
		t.Fatalf("expected generic write error to classify as final write failure, got %q", reason)
	}
	if reason := reportWriteFailureReason(reportoutputFailure(reportoutput.FailureCategory("unexpected"), "permission denied")); reason != ReportFailureReportFileWriteFailed {
		t.Fatalf("expected unknown typed write error to fall back to final write failure, got %q", reason)
	}
	if reason := reportWriteFailureReason(errors.New("plain write error")); reason != ReportFailureReportFileWriteFailed {
		t.Fatalf("expected untyped write error to fall back to final write failure, got %q", reason)
	}
	if got := joinAvailableYears(nil); got != "" {
		t.Fatalf("expected nil available years to join empty string, got %q", got)
	}
	if containsReportYear([]int{2024, 2025}, 2026) {
		t.Fatalf("expected selected year to be absent")
	}
	var pointed = pointerToReportOutcome(ReportOutcome{FailureReason: ReportFailureReportFileWriteFailed})
	if pointed == nil || pointed.FailureReason != ReportFailureReportFileWriteFailed {
		t.Fatalf("expected pointer helper to preserve report outcome, got %#v", pointed)
	}

	t.Run("fails when snapshot lifecycle has no active readable cache", func(t *testing.T) {
		t.Parallel()

		var emptyLifecycle = newSnapshotLifecycle(runtimeSnapshotStore{}, newActiveSnapshotState(), protectedPayloadBuilder{})
		var unavailableService = &reportService{snapshots: emptyLifecycle}

		var _, unavailableOutcome, ok = unavailableService.readAvailableCache(request)
		if ok {
			t.Fatalf("expected unreadable snapshot lifecycle to fail cache read")
		}
		if unavailableOutcome.FailureReason != ReportFailureNoSyncedDataAvailable {
			t.Fatalf("expected no synced data failure, got %#v", unavailableOutcome)
		}
		if !strings.Contains(unavailableOutcome.Message, "unlock or sync data first") {
			t.Fatalf("expected unlocked-cache guidance, got %q", unavailableOutcome.Message)
		}
	})
}

// TestRenderReportOutputBundleCoversPDFSelectionFailures verifies unsupported
// format and PDF renderer option failures before runtime save logic begins.
// Authored by: OpenCode
func TestRenderReportOutputBundleCoversPDFSelectionFailures(t *testing.T) {
	var request = reportRequestFixture(t, 2024)
	var report = capitalGainsReportFixture(t, request)

	if _, err := renderReportOutputBundle(reportmodel.ReportOutputFormat("html"), report); err == nil || !strings.Contains(err.Error(), "unsupported report output format") {
		t.Fatalf("expected unsupported output format failure, got %v", err)
	}

	var previousOptions = reportPDFRenderOptions
	defer func() {
		reportPDFRenderOptions = previousOptions
	}()
	reportPDFRenderOptions = func() reportpdf.RenderOptions {
		return reportpdf.RenderOptions{Fonts: reportpdf.FontData{}}
	}

	if _, err := renderReportOutputBundle(reportmodel.ReportOutputFormatPDF, report); err == nil || !strings.Contains(err.Error(), "font data") {
		t.Fatalf("expected PDF renderer option failure, got %v", err)
	}

	reportPDFRenderOptions = previousOptions
	var previousPDFDocumentConstructor = newPDFReportDocumentForRuntime
	defer func() {
		newPDFReportDocumentForRuntime = previousPDFDocumentConstructor
	}()
	newPDFReportDocumentForRuntime = func(reportmodel.ReportDocumentRole, []byte, int, reportmodel.CostBasisMethod, time.Time) (reportmodel.ReportDocument, error) {
		return reportmodel.ReportDocument{}, errors.New("pdf document finalization boom")
	}
	if _, err := renderReportOutputBundle(reportmodel.ReportOutputFormatPDF, report); err == nil || !strings.Contains(err.Error(), "pdf document finalization boom") {
		t.Fatalf("expected PDF document finalization failure, got %v", err)
	}
}

// TestReportServiceWriteReportOutputBundleBranches verifies bundle-writer seams.
// Authored by: OpenCode
func TestReportServiceWriteReportOutputBundleBranches(t *testing.T) {
	var request = reportRequestFixture(t, 2024)
	var report = capitalGainsReportFixture(t, request)
	var documents = reportDocumentBundleFixture(t, report)
	var savedAt = time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC)

	var unavailableService = &reportService{}
	if _, err := unavailableService.writeReportDocuments(reportmodel.ReportOutputFormatMarkdown, documents); err == nil || !strings.Contains(err.Error(), "report writer is unavailable") {
		t.Fatalf("expected unavailable bundle writer failure, got %v", err)
	}

	var writeFailureService = &reportService{
		writeBundle: func(reportmodel.ReportOutputFormat, []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
			return reportmodel.ReportOutputBundle{}, errors.New("bundle write boom")
		},
	}
	if _, err := writeFailureService.writeReportDocuments(reportmodel.ReportOutputFormatMarkdown, documents); err == nil || !strings.Contains(err.Error(), "bundle write boom") {
		t.Fatalf("expected bundle writer error to be returned, got %v", err)
	}

	var successService = &reportService{
		writeBundle: func(reportmodel.ReportOutputFormat, []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
			return reportOutputBundleFixture(t, "/tmp", "/tmp/report.md", savedAt), nil
		},
	}
	var bundle, err = successService.writeReportDocuments(reportmodel.ReportOutputFormatMarkdown, documents)
	if err != nil || len(bundle.Files) != 2 {
		t.Fatalf("expected valid bundle writer result, got bundle=%#v err=%v", bundle, err)
	}
}

// TestRequestAutomaticOpenBundleAdditionalBranches verifies bundle finalization
// and unavailable-opener behavior for multi-file report output.
// Authored by: OpenCode
func TestRequestAutomaticOpenBundleAdditionalBranches(t *testing.T) {
	var request = reportRequestFixture(t, 2024)
	var savedAt = time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC)
	var mainFile, err = reportmodel.NewReportOutputFile("/tmp", "main.md", "/tmp/main.md", reportmodel.ReportDocumentRoleMain, reportmodel.ReportMediaTypeMarkdown, savedAt)
	if err != nil {
		t.Fatalf("new main output file: %v", err)
	}
	var annexFile reportmodel.ReportOutputFile
	annexFile, err = reportmodel.NewReportOutputFile("/tmp", "annex.md", "/tmp/annex.md", reportmodel.ReportDocumentRoleAnnex, reportmodel.ReportMediaTypeMarkdown, savedAt)
	if err != nil {
		t.Fatalf("new annex output file: %v", err)
	}

	_, outcome := requestAutomaticOpenBundle(request, reportmodel.ReportOutputBundle{}, func(string) error { return nil })
	if outcome == nil || outcome.FailureReason != ReportFailureReportFileWriteFailed || !strings.Contains(outcome.Message, "Could not finalize") {
		t.Fatalf("expected invalid bundle finalization failure, got %#v", outcome)
	}

	var bundle reportmodel.ReportOutputBundle
	bundle, err = reportmodel.NewReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, []reportmodel.ReportOutputFile{mainFile, annexFile}, savedAt, false, "")
	if err != nil {
		t.Fatalf("new output bundle: %v", err)
	}
	var openedBundle reportmodel.ReportOutputBundle
	openedBundle, outcome = requestAutomaticOpenBundle(request, bundle, nil)
	if outcome == nil || !outcome.Success || outcome.FailureReason != ReportFailureAutomaticOpenFailedAfterSave {
		t.Fatalf("expected unavailable opener warning success, got bundle=%#v outcome=%#v", openedBundle, outcome)
	}
	if !openedBundle.OpenRequested || !strings.Contains(openedBundle.OpenError, "automatic opening is unavailable") {
		t.Fatalf("expected unavailable opener metadata, got %#v", openedBundle)
	}

	if got := reportOutputBundleForOpen(reportmodel.ReportOutputFormatMarkdown, nil, ""); got.OutputFormat != "" || len(got.Files) != 0 || !got.SavedAt.IsZero() {
		t.Fatalf("expected empty files to produce empty open bundle, got %#v", got)
	}
	if got := reportOutputBundleForOpen(reportmodel.ReportOutputFormatPDF, []reportmodel.ReportOutputFile{mainFile}, ""); got.OutputFormat != "" || len(got.Files) != 0 || !got.SavedAt.IsZero() {
		t.Fatalf("expected invalid file shape to produce empty open bundle, got %#v", got)
	}
}

// TestReportServiceRenderFallbackBranches verifies unavailable bundle seams.
// Authored by: OpenCode
func TestReportServiceLegacyAndRenderFallbackBranches(t *testing.T) {
	var request = reportRequestFixture(t, 2024)
	var service = &reportService{
		snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
		calculate: func(_ context.Context, request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
			return capitalGainsReportFixture(t, request), nil
		},
		renderBundle: func(_ reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
			return reportDocumentBundleFixture(t, report), nil
		},
		writeBundle: nil,
	}

	var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
	if outcome.FailureReason != ReportFailureReportFileWriteFailed || !strings.Contains(outcome.Message, "report writer is unavailable") {
		t.Fatalf("expected legacy unavailable writer failure, got %#v", outcome)
	}

	var renderUnavailableService = &reportService{}
	if _, err := renderUnavailableService.renderReportDocuments(reportmodel.ReportOutputFormatMarkdown, reportmodel.CapitalGainsReport{}); err == nil || !strings.Contains(err.Error(), "report renderer is unavailable") {
		t.Fatalf("expected unavailable renderer failure, got %v", err)
	}
}

// TestRenderReportOutputBundleCoversPDFRenderFailure verifies PDF render errors
// are returned before output writing begins.
// Authored by: OpenCode
func TestRenderReportOutputBundleCoversPDFRenderFailure(t *testing.T) {
	if _, err := renderReportOutputBundle(reportmodel.ReportOutputFormatPDF, reportmodel.CapitalGainsReport{}); err == nil {
		t.Fatalf("expected invalid report to fail PDF rendering")
	}
}

// reportSnapshotLifecycleWithCache returns one shared readable snapshot
// lifecycle seeded with the provided protected cache.
// Authored by: OpenCode
func reportSnapshotLifecycleWithCache(cache syncmodel.ProtectedActivityCache) *snapshotLifecycle {
	var lifecycle = newSnapshotLifecycle(runtimeSnapshotStore{}, newActiveSnapshotState(), protectedPayloadBuilder{})
	lifecycle.SetActiveSnapshot(snapshotstore.Candidate{SnapshotID: "snapshot-1"}, snapshotmodel.Payload{ProtectedActivityCache: cache})
	return lifecycle
}

// reportRequestFixture returns one valid runtime report request for internal
// report-service tests.
// Authored by: OpenCode
func reportRequestFixture(t *testing.T, year int) reportmodel.ReportRequest {
	t.Helper()

	var request, err = reportmodel.NewReportRequest(
		year,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatMarkdown,
		time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	return request
}

// reportoutputFailure returns one typed report-output failure for runtime tests.
// Authored by: OpenCode
func reportoutputFailure(category reportoutput.FailureCategory, message string) error {
	return reportoutput.NewFailure(category, errors.New(message))
}

// reportProviderCategoryRateService exposes deterministic provider metadata for
// runtime report failure tests.
// Authored by: OpenCode
type reportProviderCategoryRateService struct{}

// LookupRate satisfies the report currency-rate service seam.
// Authored by: OpenCode
func (service reportProviderCategoryRateService) LookupRate(context.Context, currencyintegration.RateLookupRequest) (currencyintegration.ExchangeRateEvidence, error) {
	return currencyintegration.ExchangeRateEvidence{}, errors.New("lookup is unused in provider-category tests")
}

// ProviderCategoryForBaseCurrency returns deterministic provider categories.
// Authored by: OpenCode
func (service reportProviderCategoryRateService) ProviderCategoryForBaseCurrency(baseCurrency string) string {
	switch baseCurrency {
	case currencyintegration.BaseCurrencyUSD:
		return string(currencyintegration.ProviderIDFederalReserveH10)
	case currencyintegration.BaseCurrencyEUR:
		return string(currencyintegration.ProviderIDECBEXR)
	default:
		return ""
	}
}

// capitalGainsReportFixture returns one minimal valid calculated report for
// runtime report-service tests.
// Authored by: OpenCode
func capitalGainsReportFixture(t *testing.T, request reportmodel.ReportRequest) reportmodel.CapitalGainsReport {
	t.Helper()

	var zero apd.Decimal
	var reportCalculationCurrency = request.ReportBaseCurrency.Label()
	var summaryEntry, err = reportmodel.NewAssetSummaryEntry("asset-btc-001", "Bitcoin", zero, reportCalculationCurrency)
	if err != nil {
		t.Fatalf("new summary entry: %v", err)
	}
	var detailSection reportmodel.AssetDetailSection
	detailSection, err = reportmodel.NewAssetDetailSection("asset-btc-001", "Bitcoin", zero, zero, zero, zero, reportCalculationCurrency, nil, nil)
	if err != nil {
		t.Fatalf("new detail section: %v", err)
	}

	var report reportmodel.CapitalGainsReport
	report, err = reportmodel.NewCapitalGainsReport(request, request.RequestedAt, reportCalculationCurrency, []reportmodel.AssetSummaryEntry{summaryEntry}, zero, nil, []reportmodel.AssetDetailSection{detailSection})
	if err != nil {
		t.Fatalf("new capital gains report: %v", err)
	}

	return report
}

// reportDocumentFixture returns one valid rendered Markdown document for runtime
// report-service tests.
// Authored by: OpenCode
func reportDocumentFixture(t *testing.T, report reportmodel.CapitalGainsReport) reportmodel.ReportDocument {
	t.Helper()

	var document, err = reportmodel.NewReportDocument(reportmodel.ReportDocumentTypeMarkdown, reportmodel.ReportDocumentRoleMain, "# Report\n", report.Year, report.CostBasisMethod, report.GeneratedAt)
	if err != nil {
		t.Fatalf("new report document: %v", err)
	}

	return document
}

// reportDocumentBundleFixture returns the valid Markdown main-plus-annex bundle
// required by runtime report-service tests.
// Authored by: OpenCode
func reportDocumentBundleFixture(t *testing.T, report reportmodel.CapitalGainsReport) []reportmodel.ReportDocument {
	var main = reportDocumentFixture(t, report)
	var annex, err = reportmodel.NewReportDocument(reportmodel.ReportDocumentTypeMarkdown, reportmodel.ReportDocumentRoleAnnex, "# Annex 1 - Audit\n", report.Year, report.CostBasisMethod, report.GeneratedAt)
	if err != nil {
		t.Fatalf("new annex report document: %v", err)
	}
	return []reportmodel.ReportDocument{main, annex}
}

// reportOutputBundleFixture returns valid saved metadata for a Markdown bundle.
// Authored by: OpenCode
func reportOutputBundleFixture(t *testing.T, directory string, mainPath string, savedAt time.Time) reportmodel.ReportOutputBundle {
	t.Helper()
	var main, err = reportmodel.NewReportOutputFile(directory, filepath.Base(mainPath), mainPath, reportmodel.ReportDocumentRoleMain, reportmodel.ReportMediaTypeMarkdown, savedAt)
	if err != nil {
		t.Fatalf("new main report output file: %v", err)
	}
	var annexPath = strings.TrimSuffix(mainPath, ".md") + "-annex-1.md"
	var annex reportmodel.ReportOutputFile
	annex, err = reportmodel.NewReportOutputFile(directory, filepath.Base(annexPath), annexPath, reportmodel.ReportDocumentRoleAnnex, reportmodel.ReportMediaTypeMarkdown, savedAt)
	if err != nil {
		t.Fatalf("new annex report output file: %v", err)
	}
	var bundle reportmodel.ReportOutputBundle
	bundle, err = reportmodel.NewReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, []reportmodel.ReportOutputFile{main, annex}, savedAt, false, "")
	if err != nil {
		t.Fatalf("new report output bundle: %v", err)
	}
	return bundle
}

// reportFailureActivityRecordFixture returns one persisted activity record with
// nullable source fields used for report-diagnostics coverage.
// Authored by: OpenCode
func reportFailureActivityRecordFixture(t *testing.T) syncmodel.ActivityRecord {
	var quantity, _, err = decimalsupport.ParseString("1")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}

	return syncmodel.ActivityRecord{
		SourceID:         "doge-buy-2025-incomplete-001",
		OccurredAt:       "2025-02-01T10:00:00Z",
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-doge-001",
		AssetSymbol:      "DOGE",
		Quantity:         quantity,
		RawHash:          "doge-buy-2025-incomplete-001",
	}
}

// TestReportConversionFailureContextFormatsTypedDetails verifies non-secret
// conversion context formatting from typed calculation context.
// Authored by: OpenCode
func TestReportConversionFailureContextFormatsTypedDetails(t *testing.T) {
	t.Parallel()

	var service = &reportService{currencyRates: reportProviderCategoryRateService{}}
	var fallbackErr = reportmodel.NewCalculationError(
		reportmodel.CalculationErrorKindActivityInput,
		"could not prepare currency conversion",
		"bad-currency",
		"BTC",
		testConversionFailureContextCause{context: reportcalculate.ConversionFailureContext{
			SourceID:           "bad-currency",
			SourceCurrency:     "usd",
			ReportBaseCurrency: "EUR",
			ActivityDate:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			Reason:             "invalid_activity_currency",
		}},
	)
	var fallbackContext = service.reportConversionFailureContext(fallbackErr)
	for _, expected := range []string{
		"Conversion Failure Context",
		"Source ID: bad-currency",
		"Source Currency: usd",
		"Report Base Currency: EUR",
		"Activity Date: 2024-01-02",
		"Failure Reason: invalid_activity_currency",
	} {
		if !strings.Contains(fallbackContext, expected) {
			t.Fatalf("expected fallback context to contain %q, got %q", expected, fallbackContext)
		}
	}
	if strings.Contains(fallbackContext, "Provider Category") {
		t.Fatalf("expected invalid activity currency context to suppress provider category, got %q", fallbackContext)
	}

	var providerFallbackErr = reportmodel.NewCalculationError(
		reportmodel.CalculationErrorKindActivityInput,
		"could not resolve currency conversion rate",
		"",
		"",
		testConversionFailureContextCause{context: reportcalculate.ConversionFailureContext{
			SourceCurrency:     "GBP",
			ReportBaseCurrency: "USD",
			ActivityDate:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		}},
	)
	var providerFallbackContext = service.reportConversionFailureContext(providerFallbackErr)
	for _, expected := range []string{
		"Source Currency: GBP",
		"Report Base Currency: USD",
		"Activity Date: 2024-01-03",
		"Provider Category: federal_reserve_h10",
	} {
		if !strings.Contains(providerFallbackContext, expected) {
			t.Fatalf("expected provider fallback context to contain %q, got %q", expected, providerFallbackContext)
		}
	}
}

// TestReportConversionFailureContextRejectsIncompleteDetails verifies helpers do
// not manufacture conversion context from unrelated or incomplete errors.
// Authored by: OpenCode
func TestReportConversionFailureContextRejectsIncompleteDetails(t *testing.T) {
	t.Parallel()

	var service = &reportService{currencyRates: reportProviderCategoryRateService{}}
	if got := service.reportConversionFailureContext(errors.New("plain failure")); got != "" {
		t.Fatalf("expected non-calculation error to produce no context, got %q", got)
	}

	var incompleteErr = reportmodel.NewCalculationError(reportmodel.CalculationErrorKindActivityInput, "could not resolve currency conversion rate", "", "", testConversionFailureContextCause{})
	if got := service.reportConversionFailureContext(incompleteErr); got != "" {
		t.Fatalf("expected incomplete conversion detail to produce no context, got %q", got)
	}
	if got := reportConversionProviderCategory(reportProviderCategoryRateService{}, "GBP"); got != "" {
		t.Fatalf("expected unsupported base currency to have no provider category, got %q", got)
	}
	if got := reportConversionProviderCategory(nil, "USD"); got != "" {
		t.Fatalf("expected unavailable provider metadata to have no provider category, got %q", got)
	}
}

// testConversionFailureContextCause carries typed conversion context in runtime
// package tests.
// Authored by: OpenCode
type testConversionFailureContextCause struct {
	context reportcalculate.ConversionFailureContext
}

// Error returns a safe test error message.
// Authored by: OpenCode
func (cause testConversionFailureContextCause) Error() string {
	return "conversion failed"
}

// ReportConversionFailureContext returns typed test conversion context.
// Authored by: OpenCode
func (cause testConversionFailureContextCause) ReportConversionFailureContext() reportcalculate.ConversionFailureContext {
	return cause.context
}
