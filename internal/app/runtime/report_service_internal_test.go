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

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
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

		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
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

		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
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

		var request = reportRequestFixture(t, 2030, reportmodel.CostBasisMethodFIFO)
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

		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			render: func(reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
				return reportmodel.ReportDocument{}, errors.New(" render boom ")
			},
			write: func(reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error) {
				t.Fatalf("write should not be called after render failure")
				return reportmodel.ReportOutputFile{}, nil
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
		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
		var fixture = testutil.NewReportIOFixture(t)
		var opener = testutil.NewOpenPathSpy(nil)
		var savedPath string
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			render: func(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
				return reportDocumentFixture(t, report, "# Report\n"), nil
			},
			write: func(document reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error) {
				savedPath = filepath.Join(fixture.DocumentsDir, "ghostfolio-capital-gains-2024-fifo-2026-05-20_15-04-05.md")
				return reportmodel.NewReportOutputFile(fixture.DocumentsDir, filepath.Base(savedPath), savedPath, document.GeneratedAt, false, "")
			},
			open: opener.Open,
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if !outcome.Success || outcome.FailureReason != ReportFailureNone {
			t.Fatalf("expected successful outcome, got %#v", outcome)
		}
		if !outcome.OutputFile.OpenRequested || outcome.OutputFile.Path != savedPath {
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
		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
		var fixture = testutil.NewReportIOFixture(t)
		var opener = testutil.NewOpenPathSpy(errors.New("open boom"))
		var savedPath = filepath.Join(fixture.DocumentsDir, "report.md")
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			render: func(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
				return reportDocumentFixture(t, report, "# Report\n"), nil
			},
			write: func(document reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error) {
				return reportmodel.NewReportOutputFile(fixture.DocumentsDir, filepath.Base(savedPath), savedPath, document.GeneratedAt, false, "")
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
		if !outcome.OutputFile.OpenRequested || outcome.OutputFile.OpenError == "" {
			t.Fatalf("expected preserved saved file with open error, got %#v", outcome.OutputFile)
		}
		if opener.CallCount() != 1 {
			t.Fatalf("expected one opener request, got %d", opener.CallCount())
		}
	})

	t.Run("fails with warning reason when opener is unavailable", func(t *testing.T) {
		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
		var fixture = testutil.NewReportIOFixture(t)
		var savedPath = filepath.Join(fixture.DocumentsDir, "report.md")
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			render: func(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
				return reportDocumentFixture(t, report, "# Report\n"), nil
			},
			write: func(document reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error) {
				return reportmodel.NewReportOutputFile(fixture.DocumentsDir, filepath.Base(savedPath), savedPath, document.GeneratedAt, false, "")
			},
			open: nil,
		}

		var outcome = service.Generate(context.Background(), ReportGenerationRequest{Request: request})
		if outcome.Success {
			t.Fatalf("expected nil opener to return warning failure outcome, got %#v", outcome)
		}
		if outcome.FailureReason != ReportFailureAutomaticOpenFailedAfterSave {
			t.Fatalf("expected automatic-open warning reason, got %#v", outcome)
		}
		if !strings.Contains(outcome.Message, "automatic opening is unavailable in this runtime") {
			t.Fatalf("expected unavailable-opener detail, got %q", outcome.Message)
		}
	})

	t.Run("fails when saved output cannot be finalized before open request", func(t *testing.T) {
		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			render: func(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
				return reportDocumentFixture(t, report, "# Report\n"), nil
			},
			write: func(document reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error) {
				return reportmodel.ReportOutputFile{
					DocumentsDirectory: "/tmp/docs",
					Path:               "/tmp/docs/report.md",
					SavedAt:            document.GeneratedAt,
				}, nil
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

		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			render: func(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
				return reportDocumentFixture(t, report, "# Report\n"), nil
			},
			write: func(reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error) {
				return reportmodel.ReportOutputFile{}, errors.New("resolve user home directory: boom")
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

		var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
		var service = &reportService{
			snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache),
			calculate: func(request reportmodel.ReportRequest, _ syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return capitalGainsReportFixture(t, request), nil
			},
			render: func(report reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error) {
				return reportDocumentFixture(t, report, "# Report\n"), nil
			},
			write: func(reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error) {
				return reportmodel.ReportOutputFile{}, errors.New("write report file \"/tmp/report.md\": permission denied")
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
		var request = reportRequestFixture(t, 2025, reportmodel.CostBasisMethodFIFO)
		var offendingRecord = reportFailureActivityRecordFixture(t)
		var service = &reportService{
			snapshots:         reportSnapshotLifecycleWithCache(syncmodel.ProtectedActivityCache{ActivityCount: 1, AvailableReportYears: []int{2025}, Activities: []syncmodel.ActivityRecord{offendingRecord}}),
			diagnosticReports: newDiagnosticReportService(t.TempDir()),
			calculate: func(reportmodel.ReportRequest, syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return reportmodel.CapitalGainsReport{}, reportmodel.NewCalculationError(reportmodel.CalculationErrorKindActivityInput, "incomplete context", offendingRecord.SourceID, offendingRecord.AssetSymbol, nil).WithPersistedActivityRecord(&offendingRecord)
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
		var request = reportRequestFixture(t, 2025, reportmodel.CostBasisMethodFIFO)
		var offendingRecord = reportFailureActivityRecordFixture(t)
		var service = &reportService{
			snapshots:         reportSnapshotLifecycleWithCache(syncmodel.ProtectedActivityCache{ActivityCount: 1, AvailableReportYears: []int{2025}, Activities: []syncmodel.ActivityRecord{offendingRecord}}),
			allowDevHTTP:      true,
			diagnosticReports: newDiagnosticReportService(baseDir),
			calculate: func(reportmodel.ReportRequest, syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
				return reportmodel.CapitalGainsReport{}, reportmodel.NewCalculationError(reportmodel.CalculationErrorKindActivityInput, "incomplete context", offendingRecord.SourceID, offendingRecord.AssetSymbol, nil).WithPersistedActivityRecord(&offendingRecord)
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

	var request = reportRequestFixture(t, 2024, reportmodel.CostBasisMethodFIFO)
	var service = &reportService{snapshots: reportSnapshotLifecycleWithCache(testutil.DeterministicReportLedgerFixture().ProtectedActivityCache)}

	var cache, outcome, ok = service.readAvailableCache(request)
	if !ok || cache.ActivityCount == 0 || outcome.Success || outcome.Message != "" || outcome.FailureReason != ReportFailureNone || outcome.Request != (reportmodel.ReportRequest{}) || outcome.OutputFile != (reportmodel.ReportOutputFile{}) || outcome.Attempt != (SyncAttempt{}) || outcome.Diagnostic.Eligible || outcome.Diagnostic.Path != "" || outcome.Diagnostic.GenerationMessage != "" {
		t.Fatalf("expected available cache read to succeed, got ok=%v cache=%#v outcome=%#v", ok, cache, outcome)
	}

	var message = reportCalculationFailureMessage(request, errors.New(" calculation boom "))
	if !strings.Contains(message, "Could not generate the 2024 FIFO report: calculation boom") {
		t.Fatalf("expected trimmed calculation failure message, got %q", message)
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
	if reason := reportWriteFailureReason(errors.New("documents path: missing")); reason != ReportFailureDocumentsFolderUnavailable {
		t.Fatalf("expected documents-path text to classify as folder unavailable, got %q", reason)
	}
	if reason := reportWriteFailureReason(errors.New("permission denied")); reason != ReportFailureReportFileWriteFailed {
		t.Fatalf("expected generic write error to classify as final write failure, got %q", reason)
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
func reportRequestFixture(t *testing.T, year int, method reportmodel.CostBasisMethod) reportmodel.ReportRequest {
	t.Helper()

	var request, err = reportmodel.NewReportRequest(year, method, time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	return request
}

// capitalGainsReportFixture returns one minimal valid calculated report for
// runtime report-service tests.
// Authored by: OpenCode
func capitalGainsReportFixture(t *testing.T, request reportmodel.ReportRequest) reportmodel.CapitalGainsReport {
	t.Helper()

	var zero apd.Decimal
	var summaryEntry, err = reportmodel.NewAssetSummaryEntry("asset-btc-001", "Bitcoin", zero, "NOT APPLICABLE")
	if err != nil {
		t.Fatalf("new summary entry: %v", err)
	}
	var detailSection reportmodel.AssetDetailSection
	detailSection, err = reportmodel.NewAssetDetailSection("asset-btc-001", "Bitcoin", zero, zero, zero, zero, "NOT APPLICABLE", nil, nil)
	if err != nil {
		t.Fatalf("new detail section: %v", err)
	}

	var report reportmodel.CapitalGainsReport
	report, err = reportmodel.NewCapitalGainsReport(request, request.RequestedAt, "NOT APPLICABLE", []reportmodel.AssetSummaryEntry{summaryEntry}, zero, nil, []reportmodel.AssetDetailSection{detailSection})
	if err != nil {
		t.Fatalf("new capital gains report: %v", err)
	}

	return report
}

// reportDocumentFixture returns one valid rendered Markdown document for runtime
// report-service tests.
// Authored by: OpenCode
func reportDocumentFixture(t *testing.T, report reportmodel.CapitalGainsReport, content string) reportmodel.ReportDocument {
	t.Helper()

	var document, err = reportmodel.NewReportDocument(reportmodel.ReportDocumentTypeMarkdown, content, report.Year, report.CostBasisMethod, report.GeneratedAt)
	if err != nil {
		t.Fatalf("new report document: %v", err)
	}

	return document
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
