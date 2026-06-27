// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"errors"
	"fmt"
	"strings"
	"time"

	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// reportCurrencyRateProviderCategory is the optional report-rate metadata seam
// used by runtime diagnostics when the configured rate service exposes it.
// Authored by: OpenCode
type reportCurrencyRateProviderCategory interface {
	ProviderCategoryForBaseCurrency(string) string
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
	parsed.activityDate = datesupport.LeadingCalendarDate(activityDate)
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
