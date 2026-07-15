// Package runtime defines the report runtime models shared by the application
// service and TUI workflow.
// Authored by: OpenCode
package runtime

import (
	"context"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// SyncReportsUnlockState identifies how one Sync and Reports unlock attempt
// completed before the context menu can be exposed.
// Authored by: OpenCode
type SyncReportsUnlockState string

const (
	// SyncReportsUnlockStateSnapshotUnlocked indicates that a selected-server
	// protected snapshot unlocked successfully.
	SyncReportsUnlockStateSnapshotUnlocked SyncReportsUnlockState = "snapshot_unlocked"

	// SyncReportsUnlockStateAuthenticatedNewContext indicates that no
	// selected-server snapshot unlocked, but Ghostfolio accepted the token as a
	// valid new isolated local-user context.
	SyncReportsUnlockStateAuthenticatedNewContext SyncReportsUnlockState = "authenticated_new_context"

	// SyncReportsUnlockStateRejectedToken indicates that no selected-server
	// snapshot unlocked and Ghostfolio rejected the supplied token.
	SyncReportsUnlockStateRejectedToken SyncReportsUnlockState = "rejected_token"
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
	Request                 reportmodel.ReportRequest
	AttemptID               string
	ServerOrigin            string
	ExplicitDevelopmentMode bool
}

// ReportOutcome stores the structured result of one completed report attempt.
// Authored by: OpenCode
type ReportOutcome struct {
	Success       bool
	Message       string
	FailureReason ReportFailureReason
	Attempt       SyncAttempt
	Request       reportmodel.ReportRequest
	OutputFormat  reportmodel.ReportOutputFormat
	OutputBundle  reportmodel.ReportOutputBundle
	OutputFile    reportmodel.ReportOutputFile
	Diagnostic    DiagnosticReportState
}

// ReportFailureDiagnosticCarrier exposes original persisted-record context for
// one report-generation failure when such context exists.
// Authored by: OpenCode
type ReportFailureDiagnosticCarrier interface {
	DiagnosticReportContext() syncmodel.DiagnosticContext
}

// SyncReportsContextResult stores the unlocked selected-server protected-data
// summary for the active Sync and Reports workflow.
// Authored by: OpenCode
type SyncReportsContextResult struct {
	UnlockState             SyncReportsUnlockState
	FailureReason           SyncFailureReason
	ProtectedData           ProtectedDataState
	ReportUnavailableReason ReportFailureReason
}

// ReportService runs report generation against the currently unlocked protected
// activity cache.
// Authored by: OpenCode
type ReportService interface {
	// Generate validates the request, calculates the report, renders Markdown,
	// writes the final file, and returns one transient user-visible outcome.
	//
	// Callers should pass the currently unlocked runtime context together with one
	// fully validated `ReportGenerationRequest` that identifies the selected year,
	// cost-basis method, attempt identifier, current server origin, and whether
	// explicit development-mode diagnostics are allowed. The returned
	// `ReportOutcome` is intended for immediate workflow rendering only. It does
	// not imply any retained in-memory report history beyond the current screen.
	//
	// Example:
	//
	//	request := runtime.ReportGenerationRequest{
	//		Request:   validatedRequest,
	//		AttemptID: "attempt-1",
	//	}
	//	outcome := reportService.Generate(context.Background(), request)
	//	_ = outcome.Success
	//
	// Authored by: OpenCode
	Generate(context.Context, ReportGenerationRequest) ReportOutcome
}
