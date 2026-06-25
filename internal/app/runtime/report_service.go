// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// reportCalculator defines the calculation seam used by the runtime report
// service.
// Authored by: OpenCode
type reportCalculator func(context.Context, reportmodel.ReportRequest, syncmodel.ProtectedActivityCache) (
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
	currencyRates     reportcalculate.CurrencyRateService
	calculate         reportCalculator
	render            reportRenderer
	write             reportDocumentWriter
	open              reportPathOpener
}

// newReportService creates the runtime report service backed by the shared
// readable protected-snapshot lifecycle.
// Authored by: OpenCode
func newReportService(
	snapshots *snapshotLifecycle,
	baseConfigDir string,
	allowDevHTTP bool,
	currencyRates reportcalculate.CurrencyRateService,
) ReportService {
	var calculator = reportcalculate.NewCalculator(currencyRates)

	return &reportService{
		snapshots:         snapshots,
		allowDevHTTP:      allowDevHTTP,
		diagnosticReports: newDiagnosticReportService(baseConfigDir),
		currencyRates:     currencyRates,
		calculate:         calculator.Calculate,
		render:            reportmarkdown.Render,
		write:             reportoutput.WriteReportDocument,
		open:              reportoutput.OpenPath,
	}
}

// Generate validates report availability, calculates the report, renders
// Markdown, writes the final file, and performs one post-save open request.
// Authored by: OpenCode
func (s *reportService) Generate(ctx context.Context, request ReportGenerationRequest) ReportOutcome {
	var outcomeAttempt = reportAttempt(request)

	var cache, outcome, ok = s.readAvailableCache(request.Request)
	if !ok {
		outcome.Attempt = outcomeAttempt
		return outcome
	}

	var report, err = s.calculate(ctx, request.Request, cache)
	if err != nil {
		return s.reportFailureOutcome(
			ctx,
			request,
			outcomeAttempt,
			ReportFailureUnsupportedReportCalculation,
			s.reportCalculationFailureMessage(request.Request, err),
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

// reportCalculationFailureMessage formats one actionable calculation failure.
// Authored by: OpenCode
func (s *reportService) reportCalculationFailureMessage(request reportmodel.ReportRequest, err error) string {
	var detail = strings.TrimSpace(err.Error())
	var conversionContext = s.reportConversionFailureContext(request, err)
	if conversionContext != "" {
		detail += "\n\n" + conversionContext
	}

	return fmt.Sprintf(
		"Could not generate the %d %s report: %s. Review the referenced synced activity data and try again. No report file was saved.",
		request.Year,
		request.CostBasisMethod.Label(),
		detail,
	)
}

// reportConversionFailureContext formats known non-secret conversion lookup
// context as short lines that survive terminal wrapping.
// Authored by: OpenCode
func (s *reportService) reportConversionFailureContext(request reportmodel.ReportRequest, err error) string {
	var calculationError *reportmodel.CalculationError
	if !errors.As(err, &calculationError) || calculationError == nil {
		return ""
	}

	var parsed = parseReportConversionFailureDetail(calculationError.Error())
	if parsed.sourceCurrency == "" || parsed.activityDate == "" {
		return ""
	}
	if parsed.reportBaseCurrency == "" {
		parsed.reportBaseCurrency = request.ReportBaseCurrency.Label()
	}
	if parsed.reason == "" && strings.Contains(calculationError.Error(), "could not prepare currency conversion") {
		parsed.reason = "invalid_activity_currency"
	}
	if parsed.provider == "" {
		parsed.provider = reportConversionProviderCategory(s.currencyRates, parsed.reportBaseCurrency)
	}

	var lines = []string{"Conversion Failure Context"}
	if sourceID := strings.TrimSpace(calculationError.SourceID()); sourceID != "" {
		lines = append(lines, "Source ID: "+sourceID)
	}
	lines = append(lines,
		"Source Currency: "+parsed.sourceCurrency,
		"Report Base Currency: "+parsed.reportBaseCurrency,
		"Activity Date: "+parsed.activityDate,
	)
	if parsed.reason != "" {
		lines = append(lines, "Failure Reason: "+parsed.reason)
	}
	if parsed.provider != "" && parsed.reason != "invalid_activity_currency" {
		lines = append(lines, "Provider Category: "+parsed.provider)
	}

	return strings.Join(lines, "\n")
}
