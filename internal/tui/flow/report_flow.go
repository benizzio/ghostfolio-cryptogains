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
	var actionCount = len(m.reportSelectionMenuItems())

	switch {
	case key.Matches(message, focusBinding()):
		if m.report.FocusArea < 2 {
			m.report.FocusArea++
		} else {
			m.report.FocusArea = 0
		}
	case key.Matches(message, upBinding()):
		switch m.report.FocusArea {
		case 0:
			if m.report.YearIndex > 0 {
				m.report.YearIndex--
			}
		case 1:
			if m.report.MethodIndex > 0 {
				m.report.MethodIndex--
			}
		case 2:
			if m.report.ActionIndex > 0 {
				m.report.ActionIndex--
			}
		}
	case key.Matches(message, downBinding()):
		switch m.report.FocusArea {
		case 0:
			if m.report.YearIndex < yearCount-1 {
				m.report.YearIndex++
			}
		case 1:
			if m.report.MethodIndex < methodCount-1 {
				m.report.MethodIndex++
			}
		case 2:
			if m.report.ActionIndex < actionCount-1 {
				m.report.ActionIndex++
			}
		}
	case key.Matches(message, enterBinding()):
		if m.report.FocusArea != 2 {
			if m.report.FocusArea < 2 {
				m.report.FocusArea++
			}
			break
		}
		if m.report.ActionIndex == 0 {
			return m.startReportGeneration()
		}
		m.clearTransientReportState()
		m.active = syncReportsMenuScreenKey
		m.sync.MenuIndex = 1
		return m, nil
	}

	if yearCount > 0 && m.report.YearIndex >= 0 && m.report.YearIndex < yearCount {
		m.report.SelectedYear = m.syncReports.ProtectedData.AvailableReportYears[m.report.YearIndex]
	}
	return m, nil
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
	var items = m.reportResultMenuItems()

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
		if m.reportOutcomeHasPendingDiagnostic() {
			switch m.report.ActionIndex {
			case 0:
				return m.generateDiagnosticReport()
			case 1:
				m.clearTransientReportState()
				m.active = syncReportsMenuScreenKey
				m.sync.MenuIndex = 1
				return m, nil
			default:
				m.clearTransientReportState()
				m.enterReportSelection()
				return m, nil
			}
		}

		if m.report.ActionIndex == 0 {
			m.clearTransientReportState()
			m.active = syncReportsMenuScreenKey
			m.sync.MenuIndex = 1
			return m, nil
		}
		m.clearTransientReportState()
		m.enterReportSelection()
		return m, nil
	}

	return m, nil
}

// clearTransientReportState removes transient in-memory report output and result
// state so the application keeps no report history.
// Authored by: OpenCode
func (m *Model) clearTransientReportState() {
	m.syncReports.ReportResult = runtime.ReportOutcome{}
	m.report = newReportState(m.syncReports.ProtectedData.AvailableReportYears)
}
