// Package runtime defines the sync runtime models shared by the application
// service and later workflow phases.
// Authored by: OpenCode
package runtime

import (
	"time"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// SyncFailureReason identifies one supported user-visible sync outcome category.
// Authored by: OpenCode
type SyncFailureReason string

// ValidationFailureReason preserves the existing validation-only type name while
// the sync slice expands the broader failure taxonomy.
// Authored by: OpenCode
type ValidationFailureReason = SyncFailureReason

const (
	// SyncFailureNone indicates that the sync completed successfully.
	SyncFailureNone SyncFailureReason = ""

	// SyncFailureRejectedToken indicates that Ghostfolio rejected the supplied token.
	SyncFailureRejectedToken SyncFailureReason = "rejected token"

	// SyncFailureTimeout indicates that the sync exceeded the allowed runtime deadline.
	SyncFailureTimeout SyncFailureReason = "timeout"

	// SyncFailureConnectivityProblem indicates a transport-level reachability problem.
	SyncFailureConnectivityProblem SyncFailureReason = "connectivity problem"

	// SyncFailureUnsuccessfulServerResponse indicates a reachable server that returned an unsuccessful response.
	SyncFailureUnsuccessfulServerResponse SyncFailureReason = "unsuccessful server response"

	// SyncFailureIncompatibleServerContract indicates that the upstream response contract is incompatible.
	SyncFailureIncompatibleServerContract SyncFailureReason = "incompatible server contract"

	// SyncFailureUnsupportedActivityHistory indicates that retrieved activity data could not be supported safely.
	SyncFailureUnsupportedActivityHistory SyncFailureReason = "unsupported activity history"

	// SyncFailureUnsupportedStoredDataVersion indicates that local protected data could not be read safely.
	SyncFailureUnsupportedStoredDataVersion SyncFailureReason = "unsupported stored-data version"

	// SyncFailureIncompatibleNewSyncData indicates that newly retrieved data could not be persisted safely.
	SyncFailureIncompatibleNewSyncData SyncFailureReason = "incompatible new sync data"

	// SyncFailureServerReplacementCancelled indicates that the user declined a server-replacement sync.
	SyncFailureServerReplacementCancelled SyncFailureReason = "server replacement cancelled"
)

const (
	// ValidationFailureNone preserves the existing validation-only success constant.
	ValidationFailureNone ValidationFailureReason = SyncFailureNone

	// ValidationFailureRejectedToken preserves the existing validation-only failure constant.
	ValidationFailureRejectedToken ValidationFailureReason = SyncFailureRejectedToken

	// ValidationFailureTimeout preserves the existing validation-only failure constant.
	ValidationFailureTimeout ValidationFailureReason = SyncFailureTimeout

	// ValidationFailureConnectivityProblem preserves the existing validation-only failure constant.
	ValidationFailureConnectivityProblem ValidationFailureReason = SyncFailureConnectivityProblem

	// ValidationFailureUnsuccessfulServerResponse preserves the existing validation-only failure constant.
	ValidationFailureUnsuccessfulServerResponse ValidationFailureReason = SyncFailureUnsuccessfulServerResponse

	// ValidationFailureIncompatibleServerContract preserves the existing validation-only failure constant.
	ValidationFailureIncompatibleServerContract ValidationFailureReason = SyncFailureIncompatibleServerContract
)

// AttemptStatus identifies the current phase of one sync attempt.
// Authored by: OpenCode
type AttemptStatus string

const (
	// AttemptStatusIdle indicates that no sync attempt is currently running.
	AttemptStatusIdle AttemptStatus = "idle"

	// AttemptStatusStarted indicates that a sync attempt has started.
	AttemptStatusStarted AttemptStatus = "started"

	// AttemptStatusDiscoveringSnapshot indicates that local snapshot discovery is in flight.
	AttemptStatusDiscoveringSnapshot AttemptStatus = "discovering_snapshot"

	// AttemptStatusUnlockingSnapshot indicates that a selected-server snapshot unlock attempt is in flight.
	AttemptStatusUnlockingSnapshot AttemptStatus = "unlocking_snapshot"

	// AttemptStatusAuthenticating indicates that anonymous auth is in flight.
	AttemptStatusAuthenticating AttemptStatus = "authenticating"

	// AttemptStatusRetrievingHistory indicates that activity retrieval is in flight.
	AttemptStatusRetrievingHistory AttemptStatus = "retrieving_history"

	// AttemptStatusNormalizing indicates that activity normalization is in flight.
	AttemptStatusNormalizing AttemptStatus = "normalizing"

	// AttemptStatusValidating indicates that activity validation is in flight.
	AttemptStatusValidating AttemptStatus = "validating"

	// AttemptStatusPersisting indicates that protected snapshot persistence is in flight.
	AttemptStatusPersisting AttemptStatus = "persisting"

	// AttemptStatusSuccess indicates that a sync attempt completed successfully.
	AttemptStatusSuccess AttemptStatus = "success"

	// AttemptStatusFailed indicates that a sync attempt completed with failure.
	AttemptStatusFailed AttemptStatus = "failed"

	// AttemptStatusAborted indicates that a sync attempt was aborted before retrieval completed.
	AttemptStatusAborted AttemptStatus = "aborted"
)

const (
	// AttemptStatusRequestingActivities preserves the existing validation-only lifecycle label.
	AttemptStatusRequestingActivities AttemptStatus = AttemptStatusRetrievingHistory

	// AttemptStatusValidatingPayload preserves the existing validation-only lifecycle label.
	AttemptStatusValidatingPayload AttemptStatus = AttemptStatusValidating

	// AttemptStatusFailure preserves the existing validation-only lifecycle label.
	AttemptStatusFailure AttemptStatus = AttemptStatusFailed
)

// GhostfolioSession is the transient authenticated runtime state for one sync attempt.
// Authored by: OpenCode
type GhostfolioSession struct {
	ServerOrigin    string
	SecurityToken   string
	AuthToken       string
	StartedAt       time.Time
	AuthenticatedAt time.Time
}

// SyncAttempt is the transient workflow state for one sync-data execution.
// Authored by: OpenCode
type SyncAttempt struct {
	AttemptID               string
	Status                  AttemptStatus
	FailureReason           SyncFailureReason
	StartedAt               time.Time
	CompletedAt             time.Time
	ServerMismatchConfirmed bool
}

// SyncValidationAttempt preserves the existing validation-only attempt type name.
// Authored by: OpenCode
type SyncValidationAttempt = SyncAttempt

// SyncOutcome is the structured result of a completed sync attempt.
// Authored by: OpenCode
type SyncOutcome struct {
	Success       bool
	DetailReason  string
	FailureReason SyncFailureReason
	Attempt       SyncAttempt
	Diagnostic    DiagnosticReportState
}

// ValidationOutcome preserves the existing validation-only outcome type name.
// Authored by: OpenCode
type ValidationOutcome = SyncOutcome

// DiagnosticReportRequest stores the structured data needed to write one local
// synced-data diagnostic report.
// Authored by: OpenCode
type DiagnosticReportRequest struct {
	FailureReason           SyncFailureReason
	ServerOrigin            string
	Attempt                 SyncAttempt
	Context                 syncmodel.DiagnosticContext
	RedactFinancialValues   bool
	ExplicitDevelopmentMode bool
}

// DiagnosticReportState tracks whether a local synced-data diagnostic report is
// available for the current failure outcome and where it was written.
// Authored by: OpenCode
type DiagnosticReportState struct {
	Eligible bool
	Path     string
	Request  DiagnosticReportRequest
}
