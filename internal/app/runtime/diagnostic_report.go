// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

const (
	applicationDirectoryName = "ghostfolio-cryptogains"
	diagnosticsDirectoryName = "diagnostics"
	diagnosticReportVersion  = 1
)

// diagnosticReportDocument is the structured local troubleshooting artifact
// written for eligible synced-data failures.
// Authored by: OpenCode
type diagnosticReportDocument struct {
	SchemaVersion           int                              `json:"schema_version"`
	GeneratedAt             time.Time                        `json:"generated_at"`
	FailureReason           SyncFailureReason                `json:"failure_reason"`
	ServerOrigin            string                           `json:"server_origin"`
	ExplicitDevelopmentMode bool                             `json:"explicit_development_mode"`
	FinancialValuesRedacted bool                             `json:"financial_values_redacted"`
	Attempt                 diagnosticAttemptDocument        `json:"attempt"`
	FailureStage            syncmodel.DiagnosticFailureStage `json:"failure_stage,omitempty"`
	FailureDetail           string                           `json:"failure_detail,omitempty"`
	Records                 []syncmodel.DiagnosticRecord     `json:"records,omitempty"`
}

// diagnosticAttemptDocument stores the structured attempt lifecycle context
// persisted with one report.
// Authored by: OpenCode
type diagnosticAttemptDocument struct {
	AttemptID               string        `json:"attempt_id,omitempty"`
	Status                  AttemptStatus `json:"status,omitempty"`
	StartedAt               time.Time     `json:"started_at"`
	CompletedAt             time.Time     `json:"completed_at"`
	ServerMismatchConfirmed bool          `json:"server_mismatch_confirmed"`
}

// writeDiagnosticReport persists one structured local troubleshooting artifact.
// Authored by: OpenCode
func writeDiagnosticReport(
	ctx context.Context,
	baseConfigDir string,
	request DiagnosticReportRequest,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if request.FailureReason == SyncFailureNone {
		return "", fmt.Errorf("diagnostic report requires a failure reason")
	}
	if request.ServerOrigin == "" {
		return "", fmt.Errorf("diagnostic report requires a server origin")
	}

	var reportID, err = randomIdentifier(8)
	if err != nil {
		return "", err
	}

	var timestamp = time.Now().UTC()
	var fileName = fmt.Sprintf("%s-%s.diagnostic.json", timestamp.Format("20060102T150405.000000000Z"), reportID)
	var path = filepath.Join(baseConfigDir, applicationDirectoryName, diagnosticsDirectoryName, fileName)

	var document = buildDiagnosticReportDocument(request, timestamp)
	var contents []byte
	contents, err = json.MarshalIndent(document, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode diagnostic report: %w", err)
	}
	contents = append(contents, '\n')

	if err := snapshotstore.ReplaceFileAtomically(path, contents); err != nil {
		return "", fmt.Errorf("write diagnostic report: %w", err)
	}

	return path, nil
}

// buildDiagnosticReportDocument converts one report request into the structured
// persisted document.
// Authored by: OpenCode
func buildDiagnosticReportDocument(
	request DiagnosticReportRequest,
	generatedAt time.Time,
) diagnosticReportDocument {
	var context = request.Context
	if request.RedactFinancialValues {
		context = redactDiagnosticContext(context)
	}

	return diagnosticReportDocument{
		SchemaVersion:           diagnosticReportVersion,
		GeneratedAt:             generatedAt,
		FailureReason:           request.FailureReason,
		ServerOrigin:            request.ServerOrigin,
		ExplicitDevelopmentMode: request.ExplicitDevelopmentMode,
		FinancialValuesRedacted: request.RedactFinancialValues,
		Attempt: diagnosticAttemptDocument{
			AttemptID:               request.Attempt.AttemptID,
			Status:                  request.Attempt.Status,
			StartedAt:               request.Attempt.StartedAt,
			CompletedAt:             request.Attempt.CompletedAt,
			ServerMismatchConfirmed: request.Attempt.ServerMismatchConfirmed,
		},
		FailureStage:  context.FailureStage,
		FailureDetail: context.FailureDetail,
		Records:       context.Records,
	}
}

// redactDiagnosticContext removes financial-value fields from offending-record
// context for non-development diagnostic reports.
// Authored by: OpenCode
func redactDiagnosticContext(context syncmodel.DiagnosticContext) syncmodel.DiagnosticContext {
	var records = make([]syncmodel.DiagnosticRecord, 0, len(context.Records))
	for _, record := range context.Records {
		record.Quantity = ""
		record.UnitPrice = ""
		record.GrossValue = ""
		record.FeeAmount = ""
		records = append(records, record)
	}

	context.Records = records
	return context
}

// resolveBaseConfigDir returns the configured application-data root used for
// local stores and troubleshooting artifacts.
// Authored by: OpenCode
func resolveBaseConfigDir(baseConfigDir string) (string, error) {
	if baseConfigDir != "" {
		return baseConfigDir, nil
	}

	var userConfigDir, err = os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}

	return userConfigDir, nil
}
