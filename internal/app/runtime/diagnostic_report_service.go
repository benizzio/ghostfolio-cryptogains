// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import "context"

// diagnosticReportService coordinates local synced-data diagnostic-report
// creation for runtime failures.
// Authored by: OpenCode
type diagnosticReportService struct {
	baseConfigDir string
}

// newDiagnosticReportService creates the runtime diagnostic-report writer.
// Authored by: OpenCode
func newDiagnosticReportService(baseConfigDir string) diagnosticReportService {
	return diagnosticReportService{baseConfigDir: baseConfigDir}
}

// Write writes one local diagnostic report for an eligible sync failure.
// Authored by: OpenCode
func (s diagnosticReportService) Write(ctx context.Context, request DiagnosticReportRequest) (string, error) {
	var baseConfigDir, err = resolveBaseConfigDir(s.baseConfigDir)
	if err != nil {
		return "", err
	}

	return writeDiagnosticReport(ctx, baseConfigDir, request)
}

// PrepareState builds the user-visible diagnostic-report state and auto-writes
// the report when explicit development mode is active.
// Authored by: OpenCode
func (s diagnosticReportService) PrepareState(request DiagnosticReportRequest) DiagnosticReportState {
	var state = DiagnosticReportState{
		Eligible: true,
		Request:  request,
	}
	if !request.ExplicitDevelopmentMode {
		return state
	}

	path, err := s.Write(context.Background(), request)
	if err == nil {
		state.Path = path
	}

	return state
}
