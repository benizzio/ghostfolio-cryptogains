// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	tea "charm.land/bubbletea/v2"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

// View renders the current full-screen workflow state.
//
// Example:
//
//	view := model.View()
//	_ = view.Content
//
// Authored by: OpenCode
func (m *Model) View() tea.View {
	var rendered string
	switch m.active {
	case setupScreenKey:
		rendered = m.viewSetupScreen()
	case mainMenuScreenKey:
		rendered = m.viewMainMenuScreen()
	case syncReportsUnlockScreenKey:
		rendered = m.viewSyncReportsUnlockScreen()
	case syncScreenKey:
		rendered = m.viewSyncScreen()
	case syncReportsMenuScreenKey:
		rendered = m.viewSyncReportsMenuScreen()
	case reportSelectionScreenKey:
		rendered = m.viewReportSelectionScreen()
	case reportBusyScreenKey:
		rendered = m.viewReportBusyScreen()
	case reportResultScreenKey:
		rendered = m.viewReportResultScreen()
	case serverReplacementScreenKey:
		rendered = m.viewServerReplacementScreen()
	case syncResultScreenKey:
		rendered = m.viewSyncResultScreen()
	default:
		rendered = ""
	}

	var view = tea.NewView(rendered)
	view.AltScreen = true
	view.WindowTitle = "ghostfolio-cryptogains"
	return view
}

// viewSetupScreen renders setup-specific state.
// Authored by: OpenCode
func (m *Model) viewSetupScreen() string {
	return screen.SetupScreenView(screen.SetupScreenParams{
		Theme:               m.theme,
		Width:               m.width,
		Height:              m.height,
		MenuItems:           m.setupMenuItems(),
		SelectedIndex:       m.setup.MenuIndex,
		ShowOriginInput:     m.setup.SelectedMode == configmodel.ServerModeCustomOrigin,
		OriginInput:         m.setup.OriginInput.View(),
		InvalidSetupMessage: setupInvalidMessage(m.setup.StartupReason),
		ValidationMessage:   m.setup.ValidationMessage,
		HelpText:            m.setupHelpText(),
		CanSave:             m.setupCanSave(),
	})
}

// viewMainMenuScreen renders the top-level menu.
// Authored by: OpenCode
func (m *Model) viewMainMenuScreen() string {
	return screen.MainMenuScreenView(screen.MainMenuScreenParams{
		Theme:         m.theme,
		Width:         m.width,
		Height:        m.height,
		MenuItems:     m.mainMenuItems(),
		SelectedIndex: 0,
		ServerOrigin:  m.currentServerOrigin(),
		HelpText:      m.mainMenuHelpText(),
	})
}

// viewSyncReportsUnlockScreen renders the context unlock prompt.
// Authored by: OpenCode
func (m *Model) viewSyncReportsUnlockScreen() string {
	return screen.SyncEntryScreenView(screen.SyncEntryScreenParams{
		Theme:                   m.theme,
		Width:                   m.width,
		Height:                  m.height,
		ScreenTitle:             "Sync and Reports",
		ScreenSubtitle:          "Unlock the active sync and reporting context.",
		IntroText:               component.SyncReportsUnlockIntroText,
		IdleStatusText:          component.SyncReportsUnlockIdleStatusText,
		ValidationMessage:       m.syncReportsUnlockValidationMessage(),
		ShowProtectedDataStatus: false,
		MenuItems:               m.syncMenuItems(),
		SelectedIndex:           m.sync.MenuIndex,
		TokenInput:              m.sync.TokenInput.View(),
		HelpText:                m.syncHelpText(),
		Busy:                    m.sync.Busy,
		BusyText:                m.sync.BusyText,
		SpinnerFrame:            m.spinner.View(),
		ProtectedDataExists:     m.deps.SyncService.ProtectedDataState().HasReadableSnapshot,
	})
}

// viewSyncScreen renders standalone or context-token sync entry.
// Authored by: OpenCode
func (m *Model) viewSyncScreen() string {
	var syncCopy = component.DefaultSyncEntryCopy(m.sync.UseContextToken)
	return screen.SyncEntryScreenView(screen.SyncEntryScreenParams{
		Theme:                   m.theme,
		Width:                   m.width,
		Height:                  m.height,
		ScreenTitle:             "Sync Data",
		ScreenSubtitle:          "Retrieve, validate, and securely store supported activity history.",
		IntroText:               syncCopy.IntroText,
		IdleStatusText:          syncCopy.IdleStatusText,
		UseContextToken:         m.sync.UseContextToken,
		ShowProtectedDataStatus: true,
		MenuItems:               m.syncMenuItems(),
		SelectedIndex:           m.sync.MenuIndex,
		TokenInput:              m.sync.TokenInput.View(),
		ValidationMessage:       m.sync.ValidationMessage,
		HelpText:                m.syncHelpText(),
		Busy:                    m.sync.Busy,
		BusyText:                m.sync.BusyText,
		SpinnerFrame:            m.spinner.View(),
		ProtectedDataExists:     m.deps.SyncService.ProtectedDataState().HasReadableSnapshot,
	})
}

// viewSyncReportsMenuScreen renders the unlocked context menu.
// Authored by: OpenCode
func (m *Model) viewSyncReportsMenuScreen() string {
	return screen.SyncReportsScreenView(screen.SyncReportsScreenParams{
		Theme:              m.theme,
		Width:              m.width,
		Height:             m.height,
		ServerOrigin:       m.syncReports.SelectedServerOrigin,
		SelectedIndex:      m.sync.MenuIndex,
		MenuItems:          m.syncReportsMenuItems(),
		ProtectedDataState: m.syncReports.ProtectedData,
		SyncOutcome:        m.syncReports.SyncResult.Outcome,
		Busy:               m.syncReports.SyncResult.Busy,
		StatusMessage:      m.syncReports.SyncResult.StatusMessage,
		UnavailableMessage: string(m.syncReports.ReportUnavailable),
		HelpText:           m.syncReportsHelpText(),
	})
}

// viewReportSelectionScreen renders year, method, and base-currency selection.
// Authored by: OpenCode
func (m *Model) viewReportSelectionScreen() string {
	return screen.ReportSelectionScreenView(screen.ReportSelectionScreenParams{
		Theme:                     m.theme,
		Width:                     m.width,
		Height:                    m.height,
		AvailableYears:            m.syncReports.ProtectedData.AvailableReportYears,
		SelectedYearIndex:         m.report.YearIndex,
		MethodItems:               m.reportMethodItems(),
		SelectedMethod:            m.report.MethodIndex,
		BaseCurrencyItems:         m.reportBaseCurrencyItems(),
		SelectedBaseCurrencyIndex: m.report.BaseCurrencyIndex,
		MethodExplanation:         reportMethodForIndex(m.report.MethodIndex).Explanation(),
		MenuItems:                 m.reportSelectionMenuItems(),
		SelectedAction:            m.report.ActionIndex,
		HelpText:                  m.reportSelectionHelpText(),
	})
}

// viewReportBusyScreen renders report-generation progress.
// Authored by: OpenCode
func (m *Model) viewReportBusyScreen() string {
	return screen.ReportBusyScreenView(screen.ReportBusyScreenParams{
		Theme:              m.theme,
		Width:              m.width,
		Height:             m.height,
		SelectedYear:       m.report.SelectedYear,
		MethodLabel:        reportMethodForIndex(m.report.MethodIndex).Label(),
		ReportBaseCurrency: m.report.SelectedBaseCurrency,
		BusyText:           m.report.BusyText,
		SpinnerFrame:       m.spinner.View(),
		HelpText:           m.reportBusyHelpText(),
	})
}

// viewReportResultScreen renders completed report output state.
// Authored by: OpenCode
func (m *Model) viewReportResultScreen() string {
	return screen.ReportResultScreenView(screen.ReportResultScreenParams{
		Theme:         m.theme,
		Width:         m.width,
		Height:        m.height,
		Outcome:       m.syncReports.ReportResult,
		MethodLabel:   m.syncReports.ReportResult.Request.CostBasisMethod.Label(),
		MenuItems:     m.reportResultMenuItems(),
		SelectedIndex: m.report.ActionIndex,
		HelpText:      m.reportResultHelpText(),
	})
}

// viewServerReplacementScreen renders server-replacement confirmation.
// Authored by: OpenCode
func (m *Model) viewServerReplacementScreen() string {
	return screen.ServerReplacementScreenView(screen.ServerReplacementScreenParams{
		Theme:         m.theme,
		Width:         m.width,
		Height:        m.height,
		MenuItems:     m.serverReplacementMenuItems(),
		SelectedIndex: m.replacement.MenuIndex,
		CurrentServer: m.replacement.CurrentServer,
		NewServer:     m.replacement.NewServer,
		HelpText:      m.serverReplacementHelpText(),
	})
}

// viewSyncResultScreen renders sync completion state.
// Authored by: OpenCode
func (m *Model) viewSyncResultScreen() string {
	return screen.SyncResultScreenView(screen.SyncResultScreenParams{
		Theme:         m.theme,
		Width:         m.width,
		Height:        m.height,
		MenuItems:     m.resultMenuItems(),
		SelectedIndex: m.result.MenuIndex,
		Outcome:       m.result.Outcome,
		Busy:          m.result.Busy,
		StatusMessage: m.result.StatusMessage,
		HelpText:      m.resultHelpText(),
	})
}
