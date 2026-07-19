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

const (
	reportSelectionFocusYear = iota
	reportSelectionFocusMethod
	reportSelectionFocusBaseCurrency
	reportSelectionFocusOutputFormat
	reportSelectionFocusAction
	reportSelectionFocusCount
)

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
	var baseCurrencyCount = len(m.reportBaseCurrencyItems())
	var outputFormatCount = len(m.reportOutputFormatItems())
	var actionCount = len(m.reportSelectionActions())

	switch {
	case key.Matches(message, focusBinding()):
		m.advanceReportSelectionFocus()
	case key.Matches(message, upBinding()):
		m.moveReportSelection(-1, yearCount, methodCount, baseCurrencyCount, outputFormatCount, actionCount)
	case key.Matches(message, downBinding()):
		m.moveReportSelection(1, yearCount, methodCount, baseCurrencyCount, outputFormatCount, actionCount)
	case key.Matches(message, enterBinding()):
		return m.activateReportSelection()
	}

	m.syncSelectedReportYear(yearCount)
	return m, nil
}

// advanceReportSelectionFocus cycles focus across year, method, base-currency,
// and action panes.
// Authored by: OpenCode
func (m *Model) advanceReportSelectionFocus() {
	if m.report.FocusArea < reportSelectionFocusCount-1 {
		m.report.FocusArea++
		return
	}

	m.report.FocusArea = 0
}

// moveReportSelection advances the currently focused report-selection index by one step.
// Authored by: OpenCode
func (m *Model) moveReportSelection(step int, yearCount int, methodCount int, baseCurrencyCount int, outputFormatCount int, actionCount int) {
	switch m.report.FocusArea {
	case reportSelectionFocusYear:
		m.report.YearIndex = boundedMenuIndex(m.report.YearIndex, step, yearCount)
	case reportSelectionFocusMethod:
		m.report.MethodIndex = boundedMenuIndex(m.report.MethodIndex, step, methodCount)
	case reportSelectionFocusBaseCurrency:
		m.report.BaseCurrencyIndex = boundedMenuIndex(m.report.BaseCurrencyIndex, step, baseCurrencyCount)
		m.selectReportBaseCurrencyAtCurrentIndex()
	case reportSelectionFocusOutputFormat:
		m.report.OutputFormatIndex = boundedMenuIndex(m.report.OutputFormatIndex, step, outputFormatCount)
		m.selectReportOutputFormatAtCurrentIndex()
	case reportSelectionFocusAction:
		m.report.ActionIndex = boundedMenuIndex(m.report.ActionIndex, step, actionCount)
	}
}

// activateReportSelection runs the selected report-selection action or advances focus.
// Authored by: OpenCode
func (m *Model) activateReportSelection() (tea.Model, tea.Cmd) {
	if m.report.FocusArea == reportSelectionFocusBaseCurrency {
		if !m.selectReportBaseCurrencyAtCurrentIndex() {
			m.report.BaseCurrencyIndex = 0
			m.selectReportBaseCurrencyAtCurrentIndex()
		}
		m.report.FocusArea = reportSelectionFocusOutputFormat
		return m, nil
	}

	if m.report.FocusArea == reportSelectionFocusOutputFormat {
		if !m.selectReportOutputFormatAtCurrentIndex() {
			m.report.OutputFormatIndex = 0
			m.selectReportOutputFormatAtCurrentIndex()
		}
		m.report.FocusArea = reportSelectionFocusAction
		return m, nil
	}

	if m.report.FocusArea != reportSelectionFocusAction {
		m.advanceReportSelectionFocus()
		return m, nil
	}

	switch m.selectedReportSelectionAction() {
	case reportSelectionActionGenerateReport:
		if !m.reportCanGenerate() {
			return m, nil
		}
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

// selectReportOutputFormatAtCurrentIndex applies the highlighted output format
// as the selected value for the pending report request.
// Authored by: OpenCode
func (m *Model) selectReportOutputFormatAtCurrentIndex() bool {
	var outputFormat = reportOutputFormatForIndex(m.report.OutputFormatIndex)
	if outputFormat == "" {
		return false
	}

	m.report.SelectedOutputFormat = outputFormat
	return true
}

// reportOutputFormatForIndex returns the supported report output format at one
// stable UI index.
// Authored by: OpenCode
func reportOutputFormatForIndex(index int) reportmodel.ReportOutputFormat {
	var formats = reportmodel.SupportedReportOutputFormats()
	if index < 0 || index >= len(formats) {
		return ""
	}
	return formats[index]
}

// syncSelectedReportYear applies the current year selection to the transient report state.
// Authored by: OpenCode
func (m *Model) syncSelectedReportYear(yearCount int) {
	if yearCount > 0 && m.report.YearIndex >= 0 && m.report.YearIndex < yearCount {
		m.report.SelectedYear = m.syncReports.ProtectedData.AvailableReportYears[m.report.YearIndex]
	}
}

// selectReportBaseCurrencyAtCurrentIndex applies the highlighted base currency
// as the selected value for the pending report request.
// Authored by: OpenCode
func (m *Model) selectReportBaseCurrencyAtCurrentIndex() bool {
	var currency = reportBaseCurrencyForIndex(m.report.BaseCurrencyIndex)
	if currency == "" {
		return false
	}

	m.report.SelectedBaseCurrency = currency
	return true
}

// reportBaseCurrencyForIndex returns the supported report base currency at one
// stable UI index.
// Authored by: OpenCode
func reportBaseCurrencyForIndex(index int) reportmodel.ReportBaseCurrency {
	var currencies = reportmodel.SupportedReportBaseCurrencies()
	if index < 0 || index >= len(currencies) {
		return ""
	}
	return currencies[index]
}

// startReportGeneration validates report prerequisites and starts one async
// generation request.
// Authored by: OpenCode
func (m *Model) startReportGeneration() (tea.Model, tea.Cmd) {
	if m.report.SelectedBaseCurrency == "" || m.report.SelectedOutputFormat == "" {
		return m, nil
	}

	var requestedAt = time.Now()
	var request = reportmodel.ReportRequest{
		Year:               m.report.SelectedYear,
		CostBasisMethod:    reportMethodForIndex(m.report.MethodIndex),
		ReportBaseCurrency: m.report.SelectedBaseCurrency,
		OutputFormat:       m.report.SelectedOutputFormat,
		RequestedAt:        requestedAt,
	}

	var validatedRequest, err = reportmodel.NewReportRequest(m.report.SelectedYear, reportMethodForIndex(m.report.MethodIndex), m.report.SelectedBaseCurrency, m.report.SelectedOutputFormat, requestedAt)
	if err != nil {
		m.enterReportResult(runtime.ReportOutcome{
			Success:       false,
			FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
			Message:       err.Error(),
			Request:       request,
		})
		return m, nil
	}

	if m.deps.ReportService == nil {
		m.enterReportResult(runtime.ReportOutcome{
			Success:       false,
			FailureReason: runtime.ReportFailureUnsupportedReportCalculation,
			Message:       "Report generation is unavailable because no runtime report service is configured.",
			Request:       validatedRequest,
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
	if key.Matches(message, pageUpBinding()) {
		m.report.ResultViewport.PageUp()
		return m, nil
	}
	if key.Matches(message, pageDownBinding()) {
		m.report.ResultViewport.PageDown()
		return m, nil
	}
	if m.report.Busy {
		return m, nil
	}

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

	m.refreshReportResultViewport(false)
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
