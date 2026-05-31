// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
)

// setupSavedMsg reports the result of an application-layer setup save request.
// Authored by: OpenCode
type setupSavedMsg struct {
	Result runtime.SaveSetupResult
	Err    error
}

// syncFinishedMsg reports the result of an asynchronous sync run.
// Authored by: OpenCode
type syncFinishedMsg struct {
	Outcome runtime.SyncOutcome
	Attempt string
}

// diagnosticReportFinishedMsg reports the result of an asynchronous
// diagnostic-report write request.
// Authored by: OpenCode
type diagnosticReportFinishedMsg struct {
	Path string
	Err  error
}

// reportFinishedMsg reports the result of one asynchronous report-generation
// run.
// Authored by: OpenCode
type reportFinishedMsg struct {
	Outcome runtime.ReportOutcome
	Attempt string
}

// quitCmd returns a Bubble Tea quit message.
// Authored by: OpenCode
func quitCmd() tea.Msg {
	return tea.Quit()
}

// saveSetupCmd delegates setup validation and persistence to the application service.
// Authored by: OpenCode
func (m *Model) saveSetupCmd(request runtime.SaveSetupRequest) tea.Cmd {
	return func() tea.Msg {
		var result, err = m.deps.SetupService.Save(context.Background(), request)
		return setupSavedMsg{Result: result, Err: err}
	}
}

// syncCmd delegates a single sync attempt to the application service.
// Authored by: OpenCode
func (m *Model) syncCmd(ctx context.Context, attemptID string, request runtime.SyncRequest) tea.Cmd {
	return func() tea.Msg {
		return syncFinishedMsg{
			Outcome: m.deps.SyncService.Run(ctx, request),
			Attempt: attemptID,
		}
	}
}

// reportCmd delegates one report-generation attempt to the runtime report
// service.
// Authored by: OpenCode
func (m *Model) reportCmd(ctx context.Context, attemptID string, request runtime.ReportGenerationRequest) tea.Cmd {
	return func() tea.Msg {
		return reportFinishedMsg{
			Outcome: m.deps.ReportService.Generate(ctx, request),
			Attempt: attemptID,
		}
	}
}

// generateDiagnosticReportCmd delegates one result-screen diagnostic-report
// write request to the runtime service.
// Authored by: OpenCode
func (m *Model) generateDiagnosticReportCmd(request runtime.DiagnosticReportRequest) tea.Cmd {
	return func() tea.Msg {
		path, err := m.deps.SyncService.GenerateDiagnosticReport(context.Background(), request)
		return diagnosticReportFinishedMsg{Path: path, Err: err}
	}
}

// nextAttemptID returns a process-local identifier for the next sync attempt.
// Authored by: OpenCode
func nextAttemptID() string {
	return fmt.Sprintf("attempt-%d", time.Now().UnixNano())
}
