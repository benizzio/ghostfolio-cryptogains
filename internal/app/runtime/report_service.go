// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
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

// reportBundleRenderer defines the selected-format rendering seam used by the
// runtime report service.
// Authored by: OpenCode
type reportBundleRenderer func(reportmodel.ReportOutputFormat, reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error)

// reportBundleWriter defines the output-bundle save seam used by the runtime
// report service.
// Authored by: OpenCode
type reportBundleWriter func(reportmodel.ReportOutputFormat, []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error)

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
	renderBundle      reportBundleRenderer
	writeBundle       reportBundleWriter
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
		renderBundle:      renderReportOutputBundle,
		writeBundle:       reportoutput.WriteReportOutputBundle,
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

	var documents []reportmodel.ReportDocument
	documents, err = s.renderReportDocuments(request.Request.OutputFormat, report)
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
	if s.writeBundle == nil {
		return s.generateLegacySingleDocumentOutcome(ctx, request, outcomeAttempt, documents)
	}

	var outputBundle reportmodel.ReportOutputBundle
	outputBundle, err = s.writeReportDocuments(request.Request.OutputFormat, documents)
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

	var openedOutputBundle, openedOutcome = requestAutomaticOpenBundle(request.Request, outputBundle, s.open)
	if openedOutcome != nil {
		openedOutcome.Attempt = outcomeAttempt
		return *openedOutcome
	}
	var primaryOutputFile = openedOutputBundle.Files[0]

	return ReportOutcome{
		Success:       true,
		Message:       reportBundleSuccessMessage(openedOutputBundle.Files),
		FailureReason: ReportFailureNone,
		Attempt:       outcomeAttempt,
		Request:       request.Request,
		OutputFormat:  request.Request.OutputFormat,
		OutputBundle:  openedOutputBundle,
		OutputFile:    primaryOutputFile,
	}
}

// generateLegacySingleDocumentOutcome preserves the older render/write seams
// used by runtime unit tests that predate output bundles.
// Authored by: OpenCode
func (s *reportService) generateLegacySingleDocumentOutcome(
	ctx context.Context,
	request ReportGenerationRequest,
	outcomeAttempt SyncAttempt,
	documents []reportmodel.ReportDocument,
) ReportOutcome {
	if s.write == nil || len(documents) != 1 {
		return s.reportFailureOutcome(
			ctx,
			request,
			outcomeAttempt,
			ReportFailureReportFileWriteFailed,
			"Could not save the report file: report writer is unavailable. No report file was saved.",
			syncmodel.DiagnosticContext{},
		)
	}

	var outputFile, err = s.write(documents[0])
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
		OutputFormat:  request.Request.OutputFormat,
		OutputFile:    openedOutputFile,
	}
}

// renderReportDocuments renders one report through the selected-format seam or
// legacy Markdown seam used by older tests.
// Authored by: OpenCode
func (s *reportService) renderReportDocuments(outputFormat reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
	if s.renderBundle != nil {
		return s.renderBundle(outputFormat, report)
	}
	if s.render == nil {
		return nil, fmt.Errorf("report renderer is unavailable")
	}
	var document, err = s.render(report)
	if err != nil {
		return nil, err
	}
	return []reportmodel.ReportDocument{document}, nil
}

// writeReportDocuments writes rendered documents through the bundle seam or
// legacy single-document seam used by older tests.
// Authored by: OpenCode
func (s *reportService) writeReportDocuments(outputFormat reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
	if s.writeBundle != nil {
		return s.writeBundle(outputFormat, documents)
	}
	if s.write == nil || len(documents) != 1 {
		return reportmodel.ReportOutputBundle{}, fmt.Errorf("report writer is unavailable")
	}
	var outputFile, err = s.write(documents[0])
	if err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}
	return reportmodel.NewReportOutputBundle(outputFormat, []reportmodel.ReportOutputFile{outputFile}, outputFile.SavedAt, false, "")
}

// renderReportOutputBundle selects the local renderer for the requested output
// format and returns the rendered documents to save.
// Authored by: OpenCode
func renderReportOutputBundle(outputFormat reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		return reportmarkdown.RenderDocuments(report)
	case reportmodel.ReportOutputFormatPDF:
		if detail := strings.TrimSpace(os.Getenv("GHOSTFOLIO_CRYPTOGAINS_PDF_RENDER_FAILURE")); detail != "" {
			return nil, fmt.Errorf("forced PDF render failure: %s", detail)
		}
		var renderer, err = reportpdf.NewRenderer(reportpdf.RenderOptions{Fonts: reportpdf.FontData{Regular: []byte("application regular font"), Bold: []byte("application bold font")}})
		if err != nil {
			return nil, err
		}
		var payload []byte
		payload, err = renderer.Render(report)
		if err != nil {
			return nil, err
		}
		var document reportmodel.ReportDocument
		document, err = reportmodel.NewPDFReportDocument(reportmodel.ReportDocumentRoleCombined, payload, report.Year, report.CostBasisMethod, report.GeneratedAt)
		if err != nil {
			return nil, err
		}
		return []reportmodel.ReportDocument{document}, nil
	default:
		return nil, fmt.Errorf("unsupported report output format %q", outputFormat)
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
	var conversionContext = s.reportConversionFailureContext(err)
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
func (s *reportService) reportConversionFailureContext(err error) string {
	var carrier reportcalculate.ConversionFailureContextCarrier
	if !errors.As(err, &carrier) {
		return ""
	}

	var context = carrier.ReportConversionFailureContext()
	if strings.TrimSpace(context.SourceCurrency) == "" || context.ActivityDate.IsZero() {
		return ""
	}
	if strings.TrimSpace(context.ProviderCategory) == "" {
		context.ProviderCategory = reportConversionProviderCategory(s.currencyRates, context.ReportBaseCurrency)
	}

	var lines = []string{"Conversion Failure Context"}
	if sourceID := strings.TrimSpace(context.SourceID); sourceID != "" {
		lines = append(lines, "Source ID: "+sourceID)
	}
	lines = append(lines,
		"Source Currency: "+strings.TrimSpace(context.SourceCurrency),
		"Report Base Currency: "+strings.TrimSpace(context.ReportBaseCurrency),
		"Activity Date: "+datesupport.FormatCalendarDate(context.ActivityDate),
	)
	if strings.TrimSpace(context.Reason) != "" {
		lines = append(lines, "Failure Reason: "+strings.TrimSpace(context.Reason))
	}
	if strings.TrimSpace(context.ProviderCategory) != "" && strings.TrimSpace(context.Reason) != "invalid_activity_currency" {
		lines = append(lines, "Provider Category: "+strings.TrimSpace(context.ProviderCategory))
	}

	return strings.Join(lines, "\n")
}
