// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// reportCalculator defines the calculation seam used by the runtime report
// service.
// Authored by: OpenCode
type reportCalculator func(reportmodel.ReportRequest, syncmodel.ProtectedActivityCache) (
	reportmodel.CapitalGainsReport,
	error,
)

// reportRenderer defines the Markdown rendering seam used by the runtime report
// service.
// Authored by: OpenCode
type reportRenderer func(reportmodel.CapitalGainsReport) (reportmodel.ReportDocument, error)

// reportDocumentWriter defines the final file-save seam used by the runtime
// report service.
// Authored by: OpenCode
type reportDocumentWriter func(reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error)

// reportPathOpener defines the post-save opener seam used by the runtime report
// service.
// Authored by: OpenCode
type reportPathOpener func(string) error

// reportService coordinates report generation against the currently unlocked
// protected activity cache.
// Authored by: OpenCode
type reportService struct {
	snapshots         *snapshotLifecycle
	allowDevHTTP      bool
	diagnosticReports diagnosticReportService
	calculate         reportCalculator
	render            reportRenderer
	write             reportDocumentWriter
	open              reportPathOpener
}

// newReportService creates the runtime report service backed by the shared
// readable protected-snapshot lifecycle.
// Authored by: OpenCode
func newReportService(snapshots *snapshotLifecycle, baseConfigDir string, allowDevHTTP bool) ReportService {
	return &reportService{
		snapshots:         snapshots,
		allowDevHTTP:      allowDevHTTP,
		diagnosticReports: newDiagnosticReportService(baseConfigDir),
		calculate:         reportcalculate.Calculate,
		render:            reportmarkdown.Render,
		write:             reportoutput.WriteReportDocument,
		open:              reportoutput.OpenPath,
	}
}

// Generate validates report availability, calculates the report, renders
// Markdown, writes the final file, and performs one post-save open request.
// Authored by: OpenCode
func (s *reportService) Generate(ctx context.Context, request ReportGenerationRequest) ReportOutcome {
	_ = ctx

	var outcomeAttempt = reportAttempt(request)

	var cache, outcome, ok = s.readAvailableCache(request.Request)
	if !ok {
		outcome.Attempt = outcomeAttempt
		return outcome
	}

	var report, err = s.calculate(request.Request, cache)
	if err != nil {
		return s.reportFailureOutcome(
			ctx,
			request,
			outcomeAttempt,
			ReportFailureUnsupportedReportCalculation,
			reportCalculationFailureMessage(request.Request, err),
			reportDiagnosticContextFromError(err),
		)
	}

	var document reportmodel.ReportDocument
	document, err = s.render(report)
	if err != nil {
		var wrappedErr = reportRenderDiagnosticError(request.Request, err)
		return s.reportFailureOutcome(
			ctx,
			request,
			outcomeAttempt,
			ReportFailureUnsupportedReportCalculation,
			fmt.Sprintf(
				"Could not render the %d %s report: %s. No report file was saved.",
				request.Request.Year,
				request.Request.CostBasisMethod.Label(),
				strings.TrimSpace(err.Error()),
			),
			reportDiagnosticContextFromError(wrappedErr),
		)
	}

	var outputFile reportmodel.ReportOutputFile
	outputFile, err = s.write(document)
	if err != nil {
		var reason = reportWriteFailureReason(err)
		var wrappedErr = reportWriteDiagnosticError(reason, err)
		return s.reportFailureOutcome(
			ctx,
			request,
			outcomeAttempt,
			reason,
			reportWriteFailureMessage(reason, err),
			reportDiagnosticContextFromError(wrappedErr),
		)
	}

	var openedOutputFile, openedOutcome = requestAutomaticOpen(request.Request, outputFile, s.open)
	if openedOutcome != nil {
		openedOutcome.Attempt = outcomeAttempt
		return *openedOutcome
	}

	return ReportOutcome{
		Success:       true,
		Message:       reportSuccessMessage(openedOutputFile.Path),
		FailureReason: ReportFailureNone,
		Attempt:       outcomeAttempt,
		Request:       request.Request,
		OutputFile:    openedOutputFile,
	}
}

// readAvailableCache validates that the shared unlocked snapshot can satisfy
// the selected report request.
// Authored by: OpenCode
func (s *reportService) readAvailableCache(request reportmodel.ReportRequest) (
	syncmodel.ProtectedActivityCache,
	ReportOutcome,
	bool,
) {
	if s == nil || s.snapshots == nil {
		return syncmodel.ProtectedActivityCache{}, reportFailureOutcome(
			request,
			ReportFailureNoSyncedDataAvailable,
			"Report generation is unavailable because no synced data is currently unlocked. Return to Sync and Reports and unlock or sync data first.",
		), false
	}

	var cache, ok = s.snapshots.ReadableProtectedActivityCache()
	if !ok {
		return syncmodel.ProtectedActivityCache{}, reportFailureOutcome(
			request,
			ReportFailureNoSyncedDataAvailable,
			"Report generation is unavailable because no synced data is currently unlocked. Return to Sync and Reports and unlock or sync data first.",
		), false
	}
	if len(cache.AvailableReportYears) == 0 {
		return syncmodel.ProtectedActivityCache{}, reportFailureOutcome(
			request,
			ReportFailureNoReportableYearsAvailable,
			"Report generation is unavailable because the currently unlocked synced data has no reportable years. Run Sync Data first if you expect reportable activity.",
		), false
	}
	if err := request.Validate(); err != nil {
		return syncmodel.ProtectedActivityCache{}, reportFailureOutcome(
			request,
			ReportFailureUnsupportedReportCalculation,
			fmt.Sprintf(
				"Could not generate the report request: %s. Choose one of the available report years: %s.",
				strings.TrimSpace(err.Error()),
				joinAvailableYears(cache.AvailableReportYears),
			),
		), false
	}
	if !containsReportYear(cache.AvailableReportYears, request.Year) {
		return syncmodel.ProtectedActivityCache{}, reportFailureOutcome(
			request,
			ReportFailureUnsupportedReportCalculation,
			fmt.Sprintf(
				"Report year %d is not available in the currently unlocked synced data. Choose one of the available report years: %s.",
				request.Year,
				joinAvailableYears(cache.AvailableReportYears),
			),
		), false
	}

	return cache, ReportOutcome{}, true
}

// requestAutomaticOpen performs the single post-save opener request and keeps
// the saved file when the opener fails.
// Authored by: OpenCode
func requestAutomaticOpen(
	request reportmodel.ReportRequest,
	outputFile reportmodel.ReportOutputFile,
	open reportPathOpener,
) (reportmodel.ReportOutputFile, *ReportOutcome) {
	var updatedOutputFile, updateErr = reportmodel.NewReportOutputFile(
		outputFile.DocumentsDirectory,
		outputFile.Filename,
		outputFile.Path,
		outputFile.SavedAt,
		true,
		"",
	)
	if updateErr != nil {
		return reportmodel.ReportOutputFile{}, pointerToReportOutcome(
			reportFailureOutcome(
				request,
				ReportFailureReportFileWriteFailed,
				fmt.Sprintf(
					"Could not finalize the saved report result for %q: %s. The saved file may still exist at %q.",
					outputFile.Filename,
					strings.TrimSpace(updateErr.Error()),
					outputFile.Path,
				),
			),
		)
	}
	if open == nil {
		return updatedOutputFile, pointerToReportOutcome(
			reportFailureOutcome(
				request,
				ReportFailureAutomaticOpenFailedAfterSave,
				reportOpenFailureMessage(outputFile.Path, "automatic opening is unavailable in this runtime"),
			),
		)
	}

	var err = open(outputFile.Path)
	if err == nil {
		return updatedOutputFile, nil
	}

	updatedOutputFile, _ = reportmodel.NewReportOutputFile(
		outputFile.DocumentsDirectory,
		outputFile.Filename,
		outputFile.Path,
		outputFile.SavedAt,
		true,
		strings.TrimSpace(err.Error()),
	)

	return updatedOutputFile, pointerToReportOutcome(
		ReportOutcome{
			Success:       true,
			Message:       reportOpenFailureMessage(outputFile.Path, strings.TrimSpace(err.Error())),
			FailureReason: ReportFailureAutomaticOpenFailedAfterSave,
			Request:       request,
			OutputFile:    updatedOutputFile,
		},
	)
}

// reportFailureOutcome builds one runtime report outcome for an ineligible
// failed attempt.
// Authored by: OpenCode
func (s *reportService) reportFailureOutcome(
	ctx context.Context,
	request ReportGenerationRequest,
	attempt SyncAttempt,
	reason ReportFailureReason,
	message string,
	diagnosticContext syncmodel.DiagnosticContext,
) ReportOutcome {
	var outcome = reportFailureOutcome(request.Request, reason, message)
	outcome.Attempt = attempt
	if !reportDiagnosticEligible(reason) {
		return outcome
	}

	var diagnosticRequest = DiagnosticReportRequest{
		FailureCategory:         reason,
		ServerOrigin:            request.ServerOrigin,
		Attempt:                 attempt,
		Context:                 diagnosticContext,
		RedactFinancialValues:   !request.ExplicitDevelopmentMode,
		ExplicitDevelopmentMode: request.ExplicitDevelopmentMode,
	}
	outcome.Diagnostic = s.diagnosticReports.PrepareState(ctx, diagnosticRequest)
	return outcome
}

// reportFailureOutcome builds one runtime report outcome for a failed attempt.
// Authored by: OpenCode
func reportFailureOutcome(request reportmodel.ReportRequest, reason ReportFailureReason, message string) ReportOutcome {
	return ReportOutcome{
		Success:       false,
		Message:       message,
		FailureReason: reason,
		Request:       request,
	}
}

// reportAttempt derives one diagnostic/report attempt envelope from the runtime
// report-generation request.
// Authored by: OpenCode
func reportAttempt(request ReportGenerationRequest) SyncAttempt {
	var startedAt = request.Request.RequestedAt.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	return SyncAttempt{
		AttemptID:   strings.TrimSpace(request.AttemptID),
		Status:      AttemptStatusFailed,
		StartedAt:   startedAt,
		CompletedAt: time.Now().UTC(),
	}
}

// reportDiagnosticEligible reports whether one report failure may generate a
// local diagnostics artifact.
// Authored by: OpenCode
func reportDiagnosticEligible(reason ReportFailureReason) bool {
	switch reason {
	case ReportFailureUnsupportedReportCalculation, ReportFailureDocumentsFolderUnavailable, ReportFailureReportFileWriteFailed:
		return true
	default:
		return false
	}
}

// reportDiagnosticContextFromError extracts source-faithful troubleshooting
// context for one eligible report failure.
// Authored by: OpenCode
func reportDiagnosticContextFromError(err error) syncmodel.DiagnosticContext {
	var carrier ReportFailureDiagnosticCarrier
	if errors.As(err, &carrier) {
		var diagnosticReportContext = carrier.DiagnosticReportContext()
		if diagnosticReportContext.FailureDetail == "" {
			diagnosticReportContext.FailureDetail = diagnosticFailureDetail(err)
		}
		if len(diagnosticReportContext.FailureCauseChain) == 0 {
			diagnosticReportContext.FailureCauseChain = diagnosticCauseChainFromError(err)
		}
		return diagnosticReportContext
	}

	return diagnosticContextFromError(err, "")
}

// reportCalculationFailureMessage formats one actionable calculation failure.
// Authored by: OpenCode
func reportCalculationFailureMessage(request reportmodel.ReportRequest, err error) string {
	return fmt.Sprintf(
		"Could not generate the %d %s report: %s. Review the referenced synced activity data and try again. No report file was saved.",
		request.Year,
		request.CostBasisMethod.Label(),
		strings.TrimSpace(err.Error()),
	)
}

// reportRenderDiagnosticError wraps one renderer failure with report-level
// context so diagnostics can preserve both the actionable outer failure and the
// deeper wrapped cause chain.
// Authored by: OpenCode
func reportRenderDiagnosticError(request reportmodel.ReportRequest, err error) error {
	return fmt.Errorf(
		"could not render the %d %s report: %w",
		request.Year,
		request.CostBasisMethod.Label(),
		err,
	)
}

// reportWriteFailureReason classifies one save failure into the supported
// runtime taxonomy.
// Authored by: OpenCode
func reportWriteFailureReason(err error) ReportFailureReason {
	var category, ok = reportoutput.FailureCategoryOf(err)
	if ok {
		switch category {
		case reportoutput.FailureCategoryDocumentsDirectoryUnavailable:
			return ReportFailureDocumentsFolderUnavailable
		case reportoutput.FailureCategoryReportFileWriteFailed:
			return ReportFailureReportFileWriteFailed
		}
	}

	return ReportFailureReportFileWriteFailed
}

// reportWriteFailureMessage formats one actionable save failure.
// Authored by: OpenCode
func reportWriteFailureMessage(reason ReportFailureReason, err error) string {
	var detail = strings.TrimSpace(err.Error())
	if reason == ReportFailureDocumentsFolderUnavailable {
		return fmt.Sprintf(
			"Could not save the report because the Documents folder is unavailable: %s. Ensure the folder exists and is writable, then try again. No report file was saved.",
			detail,
		)
	}

	return fmt.Sprintf(
		"Could not save the report file: %s. Check write permissions and free space in the Documents folder, then try again. Any partial file created during this attempt was removed.",
		detail,
	)
}

// reportWriteDiagnosticError wraps one output-preparation failure with a stable
// report-level summary for diagnostics.
// Authored by: OpenCode
func reportWriteDiagnosticError(reason ReportFailureReason, err error) error {
	if reason == ReportFailureDocumentsFolderUnavailable {
		return fmt.Errorf("could not save the report because the Documents folder is unavailable: %w", err)
	}

	return fmt.Errorf("could not save the report file: %w", err)
}

// reportOpenFailureMessage formats one non-fatal automatic-open warning.
// Authored by: OpenCode
func reportOpenFailureMessage(path string, detail string) string {
	return fmt.Sprintf(
		"Saved the report to %q, but automatic opening failed: %s. Open the file manually. To remove this cleartext report later, delete %q.",
		path,
		detail,
		path,
	)
}

// reportSuccessMessage formats one successful report outcome.
// Authored by: OpenCode
func reportSuccessMessage(path string) string {
	return fmt.Sprintf(
		"Saved the report to %q and requested automatic opening. To remove this cleartext report later, delete %q.",
		path,
		path,
	)
}

// joinAvailableYears formats one readable available-year list.
// Authored by: OpenCode
func joinAvailableYears(years []int) string {
	var parts = make([]string, 0, len(years))
	for _, year := range years {
		parts = append(parts, fmt.Sprintf("%d", year))
	}

	return strings.Join(parts, ", ")
}

// containsReportYear reports whether the selected year exists in the unlocked
// cache metadata.
// Authored by: OpenCode
func containsReportYear(years []int, selectedYear int) bool {
	for _, year := range years {
		if year == selectedYear {
			return true
		}
	}

	return false
}

// pointerToReportOutcome returns the address of one local report outcome value.
// Authored by: OpenCode
func pointerToReportOutcome(outcome ReportOutcome) *ReportOutcome {
	return &outcome
}
