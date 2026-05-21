// Package runtime verifies package-local report-service guardrails and failure
// classification.
// Authored by: OpenCode
package runtime

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
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
	if !ok || cache.ActivityCount == 0 || outcome != (ReportOutcome{}) {
		t.Fatalf("expected available cache read to succeed, got ok=%v cache=%#v outcome=%#v", ok, cache, outcome)
	}

	var message = reportCalculationFailureMessage(request, errors.New(" calculation boom "))
	if !strings.Contains(message, "Could not generate the 2024 FIFO report: calculation boom") {
		t.Fatalf("expected trimmed calculation failure message, got %q", message)
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
