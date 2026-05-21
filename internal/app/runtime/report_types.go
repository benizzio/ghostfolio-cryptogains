// Package runtime defines the report runtime models shared by the application
// service and TUI workflow.
// Authored by: OpenCode
package runtime

import (
	"context"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// ReportFailureReason identifies one supported user-visible report result
// category, including the non-fatal automatic-open warning after a successful
// save.
// Authored by: OpenCode
type ReportFailureReason string

const (
	// ReportFailureNone indicates that report generation completed without a
	// failure or warning category.
	ReportFailureNone ReportFailureReason = ""

	// ReportFailureNoSyncedDataAvailable indicates that no protected synced data
	// was available for report generation.
	ReportFailureNoSyncedDataAvailable ReportFailureReason = "no synced data available"

	// ReportFailureNoReportableYearsAvailable indicates that synced data exists
	// but no reportable years were available.
	ReportFailureNoReportableYearsAvailable ReportFailureReason = "no reportable years available"

	// ReportFailureUnsupportedStoredDataVersion indicates that stored protected
	// data could not be read safely for reporting.
	ReportFailureUnsupportedStoredDataVersion ReportFailureReason = "unsupported stored-data version"

	// ReportFailureUnsupportedReportCalculation indicates that the selected data
	// or activity history could not support one safe report calculation.
	ReportFailureUnsupportedReportCalculation ReportFailureReason = "unsupported report calculation"

	// ReportFailureDocumentsFolderUnavailable indicates that the user's
	// Documents folder could not be resolved or used safely.
	ReportFailureDocumentsFolderUnavailable ReportFailureReason = "documents folder unavailable"

	// ReportFailureReportFileWriteFailed indicates that the final report file
	// could not be written successfully.
	ReportFailureReportFileWriteFailed ReportFailureReason = "report file write failed"

	// ReportFailureAutomaticOpenFailedAfterSave indicates that the report saved
	// successfully but the post-save open request failed.
	ReportFailureAutomaticOpenFailedAfterSave ReportFailureReason = "automatic open failed after save"
)

// ReportGenerationRequest stores the runtime inputs for one report-generation
// attempt.
// Authored by: OpenCode
type ReportGenerationRequest struct {
	Request reportmodel.ReportRequest
}

// ReportOutcome stores the structured result of one completed report attempt.
// Authored by: OpenCode
type ReportOutcome struct {
	Success       bool
	Message       string
	FailureReason ReportFailureReason
	Request       reportmodel.ReportRequest
	OutputFile    reportmodel.ReportOutputFile
}

// SyncReportsContextResult stores the unlocked selected-server protected-data
// summary for the active Sync and Reports workflow.
// Authored by: OpenCode
type SyncReportsContextResult struct {
	ProtectedData           ProtectedDataState
	ReportUnavailableReason ReportFailureReason
}

// ReportService runs report generation against the currently unlocked protected
// activity cache.
// Authored by: OpenCode
type ReportService interface {
	// Generate validates the request, calculates the report, renders Markdown,
	// writes the final file, and returns one transient user-visible outcome.
	// Authored by: OpenCode
	Generate(context.Context, ReportGenerationRequest) ReportOutcome
}
