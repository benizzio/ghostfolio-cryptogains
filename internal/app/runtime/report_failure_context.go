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
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// reportCurrencyRateProviderCategory is the optional report-rate metadata seam
// used by runtime diagnostics when the configured rate service exposes it.
// Authored by: OpenCode
type reportCurrencyRateProviderCategory interface {
	ProviderCategoryForBaseCurrency(string) string
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

// reportConversionFailureDetail stores safe fields parsed from report-owned
// calculation error copy.
// Authored by: OpenCode
type reportConversionFailureDetail struct {
	sourceCurrency     string
	reportBaseCurrency string
	activityDate       string
	reason             string
	provider           string
}

// parseReportConversionFailureDetail extracts stable conversion context from
// safe report-calculation copy without using raw provider detail.
// Authored by: OpenCode
func parseReportConversionFailureDetail(detail string) reportConversionFailureDetail {
	var parsed reportConversionFailureDetail
	var normalized = strings.TrimSpace(detail)
	parsed.reason = reportValueAfterToken(normalized, "reason=")
	parsed.provider = reportValueAfterToken(normalized, "provider=")
	parsed.sourceCurrency = reportValueAfterToken(normalized, "source_currency=")
	parsed.reportBaseCurrency = reportValueAfterToken(normalized, "report_base_currency=")
	parsed.activityDate = reportValueAfterToken(normalized, "activity_date=")
	if parsed.sourceCurrency != "" && parsed.reportBaseCurrency != "" && parsed.activityDate != "" {
		return parsed
	}

	var beforeDate, activityDate, ok = strings.Cut(normalized, " on ")
	if !ok {
		return parsed
	}
	var beforeSource, reportBaseCurrency, hasCurrencyPair = strings.Cut(beforeDate, " to ")
	if !hasCurrencyPair {
		return parsed
	}
	var sourceIndex = strings.LastIndex(beforeSource, " from ")
	if sourceIndex < 0 {
		return parsed
	}

	parsed.sourceCurrency = strings.TrimSpace(beforeSource[sourceIndex+len(" from "):])
	parsed.reportBaseCurrency = strings.TrimSpace(reportBaseCurrency)
	parsed.activityDate = reportLeadingDate(activityDate)
	return parsed
}

// reportValueAfterToken reads one unquoted token value from safe diagnostic
// copy.
// Authored by: OpenCode
func reportValueAfterToken(detail string, token string) string {
	var index = strings.Index(detail, token)
	if index < 0 {
		return ""
	}
	var value = detail[index+len(token):]
	if separator := strings.IndexAny(value, " \n\t)"); separator >= 0 {
		value = value[:separator]
	}
	if value == "unknown" {
		return ""
	}
	return strings.TrimSpace(value)
}

// reportLeadingDate returns the leading YYYY-MM-DD token when available.
// Authored by: OpenCode
func reportLeadingDate(detail string) string {
	var value = strings.TrimSpace(detail)
	if len(value) < len(time.DateOnly) {
		return ""
	}
	return value[:len(time.DateOnly)]
}

// reportConversionProviderCategory asks the configured rate service for the
// provider category associated with one report base currency.
// Authored by: OpenCode
func reportConversionProviderCategory(currencyRates reportcalculate.CurrencyRateService, reportBaseCurrency string) string {
	var metadata, ok = currencyRates.(reportCurrencyRateProviderCategory)
	if !ok {
		return ""
	}

	return strings.TrimSpace(metadata.ProviderCategoryForBaseCurrency(strings.TrimSpace(reportBaseCurrency)))
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
