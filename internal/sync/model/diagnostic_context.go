// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

// DiagnosticFailureStage identifies the sync stage that produced diagnostic
// troubleshooting context.
// Authored by: OpenCode
type DiagnosticFailureStage string

const (
	// DiagnosticFailureStageMapping identifies Ghostfolio activity mapping failures.
	DiagnosticFailureStageMapping DiagnosticFailureStage = "mapping"

	// DiagnosticFailureStageNormalization identifies normalized-history ordering or duplicate-handling failures.
	DiagnosticFailureStageNormalization DiagnosticFailureStage = "normalization"

	// DiagnosticFailureStageValidation identifies normalized-history rule failures.
	DiagnosticFailureStageValidation DiagnosticFailureStage = "validation"

	// DiagnosticFailureStageStoredDataCompatibility identifies local stored-data compatibility failures.
	DiagnosticFailureStageStoredDataCompatibility DiagnosticFailureStage = "stored_data_compatibility"

	// DiagnosticFailureStageProtectedPersistence identifies protected-storage write and local artifact failures.
	DiagnosticFailureStageProtectedPersistence DiagnosticFailureStage = "protected_persistence"
)

// DiagnosticContext stores the structured troubleshooting context attached to a
// synced-data failure.
// Authored by: OpenCode
type DiagnosticContext struct {
	FailureStage            DiagnosticFailureStage `json:"failure_stage,omitempty"`
	FailureDetail           string                 `json:"failure_detail,omitempty"`
	FailureCauseChain       []string               `json:"failure_cause_chain,omitempty"`
	Records                 []DiagnosticRecord     `json:"records,omitempty"`
	OffendingActivityRecord *ActivityRecord        `json:"-"`
}

// DiagnosticContextCarrier exposes structured troubleshooting context from
// lower-level sync failures.
// Authored by: OpenCode
type DiagnosticContextCarrier interface {
	DiagnosticContext() DiagnosticContext
}
