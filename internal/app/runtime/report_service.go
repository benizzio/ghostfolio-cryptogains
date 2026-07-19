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
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// reportPDFRenderOptions supplies application-owned embedded font bytes for the
// local PDF renderer.
// Authored by: OpenCode
var reportPDFRenderOptions = func() reportpdf.RenderOptions {
	return reportpdf.RenderOptions{Fonts: reportpdf.FontData{Regular: goregular.TTF, Bold: gobold.TTF}}
}

// newReportDocumentForRuntime keeps the defensive runtime finalization branch
// testable after renderer validation has guaranteed normal document inputs.
// Authored by: OpenCode
var newReportDocumentForRuntime = reportmodel.NewReportDocument

// reportCalculator defines the calculation seam used by the runtime report
// service.
// Authored by: OpenCode
type reportCalculator func(context.Context, reportmodel.ReportRequest, syncmodel.ProtectedActivityCache) (
	reportmodel.CapitalGainsReport,
	error,
)

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
	pipelineOptions ReportPipelineOptions,
) ReportService {
	var calculator = reportcalculate.NewCalculator(currencyRates)
	var calculate reportCalculator = calculator.Calculate
	if pipelineOptions.CalculatedReportTransform != nil {
		calculate = func(ctx context.Context, request reportmodel.ReportRequest, cache syncmodel.ProtectedActivityCache) (reportmodel.CapitalGainsReport, error) {
			var report, err = calculator.Calculate(ctx, request, cache)
			if err != nil {
				return reportmodel.CapitalGainsReport{}, err
			}
			return pipelineOptions.CalculatedReportTransform(report), nil
		}
	}

	var service = &reportService{
		snapshots:         snapshots,
		allowDevHTTP:      allowDevHTTP,
		diagnosticReports: newDiagnosticReportService(baseConfigDir),
		currencyRates:     currencyRates,
		calculate:         calculate,
		renderBundle: func(outputFormat reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
			return renderReportOutputBundleWithOptions(outputFormat, report, pipelineOptions)
		},
	}
	service.writeBundle = reportoutput.WriteReportOutputBundle
	service.open = reportoutput.OpenPath
	return service
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
				strings.TrimSpace(redact.ErrorText(err)),
			),
			reportDiagnosticContextFromError(wrappedErr),
		)
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

// renderReportDocuments renders one report through the selected-format bundle seam.
// Authored by: OpenCode
func (s *reportService) renderReportDocuments(outputFormat reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
	if s.renderBundle == nil {
		return nil, fmt.Errorf("report renderer is unavailable")
	}
	return s.renderBundle(outputFormat, report)
}

// writeReportDocuments writes rendered documents through the bundle seam.
// Authored by: OpenCode
func (s *reportService) writeReportDocuments(outputFormat reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
	if s.writeBundle == nil {
		return reportmodel.ReportOutputBundle{}, fmt.Errorf("report writer is unavailable")
	}
	return s.writeBundle(outputFormat, documents)
}

// renderReportOutputBundle selects the local renderer for the requested output
// format and returns the rendered documents to save.
// Authored by: OpenCode
func renderReportOutputBundle(outputFormat reportmodel.ReportOutputFormat, report reportmodel.CapitalGainsReport) ([]reportmodel.ReportDocument, error) {
	return renderReportOutputBundleWithOptions(outputFormat, report, ReportPipelineOptions{})
}

// renderReportOutputBundleWithOptions selects the local renderer with one
// immutable report-pipeline option set.
// Authored by: OpenCode
func renderReportOutputBundleWithOptions(
	outputFormat reportmodel.ReportOutputFormat,
	report reportmodel.CapitalGainsReport,
	pipelineOptions ReportPipelineOptions,
) ([]reportmodel.ReportDocument, error) {
	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		var renderer = reportmarkdown.NewRenderer(reportmarkdown.RenderOptions{FinancialFormatting: pipelineOptions.MarkdownFinancialFormatting})
		if pipelineOptions.MarkdownRenderObserver != nil {
			pipelineOptions.MarkdownRenderObserver()
		}
		return renderer.RenderDocuments(report)
	case reportmodel.ReportOutputFormatPDF:
		var renderOptions = reportPDFRenderOptions()
		renderOptions.FinancialFormatting = pipelineOptions.PDFFinancialFormatting
		renderOptions.ByteFinalizer = pipelineOptions.PDFByteFinalizer
		var renderer, err = reportpdf.NewRenderer(renderOptions)
		if err != nil {
			return nil, err
		}
		if pipelineOptions.PDFRenderObserver != nil {
			pipelineOptions.PDFRenderObserver()
		}
		var payload []byte
		payload, err = renderer.Render(report)
		if err != nil {
			return nil, err
		}
		var document reportmodel.ReportDocument
		document, err = newReportDocumentForRuntime(reportmodel.ReportDocumentTypePDF, reportmodel.ReportDocumentRoleCombined, payload, report.Year, report.CostBasisMethod, report.GeneratedAt)
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
				strings.TrimSpace(redact.ErrorText(err)),
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
	var detail = strings.TrimSpace(redact.ErrorText(err))
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
