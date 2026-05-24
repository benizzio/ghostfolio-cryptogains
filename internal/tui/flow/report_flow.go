// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// reportBusyStatusText is the shared busy-state message for one report run.
// Authored by: OpenCode
const reportBusyStatusText = "Generating capital gains report..."

// updateReport handles report selection, busy-state completion, and result
// navigation.
// Authored by: OpenCode
func (m *Model) updateReport(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMessage := message.(type) {
	case reportFinishedMsg:
		return m.handleReportFinished(typedMessage)
	case diagnosticReportFinishedMsg:
		return m.handleDiagnosticReportFinished(typedMessage)
	case spinner.TickMsg:
		if m.active != reportBusyScreenKey || !m.report.Busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(typedMessage)
		return m, cmd
	}

	var keyMessage, ok = message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch m.active {
	case reportSelectionScreenKey:
		return m.handleReportSelectionKey(keyMessage)
	case reportBusyScreenKey:
		return m, nil
	case reportResultScreenKey:
		return m.handleReportResultKey(keyMessage)
	default:
		return m, nil
	}
}

// handleReportSelectionKey routes report-selection navigation.
// Authored by: OpenCode
func (m *Model) handleReportSelectionKey(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var yearCount = len(m.syncReports.ProtectedData.AvailableReportYears)
	var methodCount = len(m.reportMethodItems())
	var actionCount = len(m.reportSelectionActions())

	switch {
	case key.Matches(message, focusBinding()):
		m.advanceReportSelectionFocus()
	case key.Matches(message, upBinding()):
		m.moveReportSelection(-1, yearCount, methodCount, actionCount)
	case key.Matches(message, downBinding()):
		m.moveReportSelection(1, yearCount, methodCount, actionCount)
	case key.Matches(message, enterBinding()):
		return m.activateReportSelection()
	}

	m.syncSelectedReportYear(yearCount)
	return m, nil
}

// advanceReportSelectionFocus cycles focus across year, method, and action panes.
// Authored by: OpenCode
func (m *Model) advanceReportSelectionFocus() {
	if m.report.FocusArea < 2 {
		m.report.FocusArea++
		return
	}

	m.report.FocusArea = 0
}

// moveReportSelection advances the currently focused report-selection index by one step.
// Authored by: OpenCode
func (m *Model) moveReportSelection(step int, yearCount int, methodCount int, actionCount int) {
	switch m.report.FocusArea {
	case 0:
		m.report.YearIndex = boundedMenuIndex(m.report.YearIndex, step, yearCount)
	case 1:
		m.report.MethodIndex = boundedMenuIndex(m.report.MethodIndex, step, methodCount)
	case 2:
		m.report.ActionIndex = boundedMenuIndex(m.report.ActionIndex, step, actionCount)
	}
}

// activateReportSelection runs the selected report-selection action or advances focus.
// Authored by: OpenCode
func (m *Model) activateReportSelection() (tea.Model, tea.Cmd) {
	if m.report.FocusArea != 2 {
		if m.report.FocusArea < 2 {
			m.report.FocusArea++
		}
		return m, nil
	}

	switch m.selectedReportSelectionAction() {
	case reportSelectionActionGenerateReport:
		return m.startReportGeneration()
	case reportSelectionActionBack:
		m.clearTransientReportState()
		m.active = syncReportsMenuScreenKey
		m.sync.MenuIndex = m.syncReportsReportActionIndex()
		return m, nil
	default:
		return m, nil
	}
}

// syncSelectedReportYear applies the current year selection to the transient report state.
// Authored by: OpenCode
func (m *Model) syncSelectedReportYear(yearCount int) {
	if yearCount > 0 && m.report.YearIndex >= 0 && m.report.YearIndex < yearCount {
		m.report.SelectedYear = m.syncReports.ProtectedData.AvailableReportYears[m.report.YearIndex]
	}
}

// startReportGeneration validates report prerequisites and starts one async
// generation request.
// Authored by: OpenCode
func (m *Model) startReportGeneration() (tea.Model, tea.Cmd) {
	var request = reportmodel.ReportRequest{
		Year:            m.report.SelectedYear,
		CostBasisMethod: reportMethodForIndex(m.report.MethodIndex),
		RequestedAt:     time.Now(),
	}

	if m.deps.ReportService == nil {
		m.enterReportResult(runtime.ReportOutcome{
			Success:       false,
			FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
			Message:       "Report generation is unavailable because no runtime report service is configured.",
			Request:       request,
		})
		return m, nil
	}

	var validatedRequest, err = reportmodel.NewReportRequest(m.report.SelectedYear, reportMethodForIndex(m.report.MethodIndex), time.Now())
	if err != nil {
		m.enterReportResult(runtime.ReportOutcome{
			Success:       false,
			FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
			Message:       err.Error(),
			Request:       request,
		})
		return m, nil
	}

	m.active = reportBusyScreenKey
	m.report.Busy = true
	m.report.BusyText = reportBusyStatusText
	m.report.AttemptID = nextAttemptID()
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Line))
	return m, tea.Batch(
		m.spinner.Tick,
		m.reportCmd(context.Background(), m.report.AttemptID, runtime.ReportGenerationRequest{
			Request:                 validatedRequest,
			AttemptID:               m.report.AttemptID,
			ServerOrigin:            m.currentServerOrigin(),
			ExplicitDevelopmentMode: m.deps.Options.AllowDevHTTP,
		}),
	)
}

// handleReportFinished applies one completed asynchronous report-generation
// outcome.
// Authored by: OpenCode
func (m *Model) handleReportFinished(message reportFinishedMsg) (tea.Model, tea.Cmd) {
	if message.Attempt != m.report.AttemptID {
		return m, nil
	}

	m.enterReportResult(message.Outcome)
	return m, nil
}

// handleReportResultKey routes completed report-result navigation.
// Authored by: OpenCode
func (m *Model) handleReportResultKey(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var items = m.reportResultActions()

	switch {
	case key.Matches(message, upBinding()):
		if m.report.ActionIndex > 0 {
			m.report.ActionIndex--
		}
	case key.Matches(message, downBinding()):
		if m.report.ActionIndex < len(items)-1 {
			m.report.ActionIndex++
		}
	case key.Matches(message, enterBinding()):
		return m.activateReportResultSelection()
	}

	return m, nil
}

// activateReportResultSelection runs the selected report-result action.
// Authored by: OpenCode
func (m *Model) activateReportResultSelection() (tea.Model, tea.Cmd) {
	switch m.selectedReportResultAction() {
	case reportResultActionGenerateDiagnostic:
		return m.generateDiagnosticReport()
	case reportResultActionBackToSyncReports:
		m.clearTransientReportState()
		m.active = syncReportsMenuScreenKey
		m.sync.MenuIndex = m.syncReportsReportActionIndex()
		return m, nil
	case reportResultActionGenerateAnother:
		m.clearTransientReportState()
		m.enterReportSelection()
		return m, nil
	default:
		return m, nil
	}
}

// boundedMenuIndex applies one movement step while preserving menu bounds.
// Authored by: OpenCode
func boundedMenuIndex(current int, step int, count int) int {
	var next = current + step
	if count <= 0 || next < 0 || next >= count {
		return current
	}

	return next
}

// clearTransientReportState removes transient in-memory report output and result
// state so the application keeps no report history.
// Authored by: OpenCode
func (m *Model) clearTransientReportState() {
	m.syncReports.ReportResult = runtime.ReportOutcome{}
	m.report = newReportState(m.syncReports.ProtectedData.AvailableReportYears)
}
