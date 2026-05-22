// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

// activeScreen identifies the currently rendered workflow screen.
// Authored by: OpenCode
type activeScreen string

// Screen keys identify the workflow views that the root model can render.
// Authored by: OpenCode
const (
	setupScreenKey             activeScreen = "setup"
	mainMenuScreenKey          activeScreen = "main_menu"
	syncReportsUnlockScreenKey activeScreen = "sync_reports_unlock"
	syncReportsMenuScreenKey   activeScreen = "sync_reports_menu"
	reportSelectionScreenKey   activeScreen = "report_selection"
	reportBusyScreenKey        activeScreen = "report_busy"
	reportResultScreenKey      activeScreen = "report_result"
	syncScreenKey              activeScreen = "sync"
	serverReplacementScreenKey activeScreen = "server_replacement"
	syncResultScreenKey        activeScreen = "sync_result"

	setupMenuGhostfolioCloudIndex = 0
	setupMenuCustomOriginIndex    = 1
	setupMenuSavePathIndex        = 2
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

// Dependencies contains the runtime services required by the root Bubble Tea
// model.
//
// Authored by: OpenCode
type Dependencies struct {
	Options       bootstrap.Options
	Startup       bootstrap.StartupState
	SetupService  runtime.SetupService
	SyncService   runtime.SyncService
	ReportService runtime.ReportService
}

// setupState holds transient UI state for the setup workflow.
// Authored by: OpenCode
type setupState struct {
	SelectedMode      string
	MenuIndex         int
	InputFocused      bool
	OriginInput       textinput.Model
	ValidationMessage string
	StartupReason     bootstrap.SetupRequirementReason
}

// syncState holds transient UI state for the sync-entry workflow.
// Authored by: OpenCode
type syncState struct {
	MenuIndex         int
	InputFocused      bool
	TokenInput        textinput.Model
	UseContextToken   bool
	ValidationMessage string
	Busy              bool
	BusyText          string
	AttemptID         string
	Cancel            context.CancelFunc
}

// syncReportsContextState holds the active unlock-context shell that later
// slices will reuse across sync and report actions.
// Authored by: OpenCode
type syncReportsContextState struct {
	Active               bool
	RuntimeToken         string
	SelectedServerOrigin string
	ProtectedData        runtime.ProtectedDataState
	SyncResult           syncContextResultState
	ReportUnavailable    runtime.ReportFailureReason
	ReportResult         runtime.ReportOutcome
}

// syncContextResultState holds transient sync-failure feedback rendered inside
// the active `Sync and Reports` context.
// Authored by: OpenCode
type syncContextResultState struct {
	Outcome       runtime.SyncOutcome
	Busy          bool
	StatusMessage string
}

// reportState holds transient UI state for report selection, busy execution,
// and result routing.
// Authored by: OpenCode
type reportState struct {
	FocusArea    int
	YearIndex    int
	MethodIndex  int
	ActionIndex  int
	Busy         bool
	BusyText     string
	AttemptID    string
	SelectedYear int
}

// serverReplacementState holds transient UI state for server-mismatch confirmation.
// Authored by: OpenCode
type serverReplacementState struct {
	MenuIndex     int
	PendingToken  string
	CurrentServer string
	NewServer     string
}

// resultState holds transient UI state for the sync-result screen.
// Authored by: OpenCode
type resultState struct {
	MenuIndex     int
	Outcome       runtime.SyncOutcome
	Busy          bool
	StatusMessage string
}

// Model is the root Bubble Tea model for the application workflow.
//
// Authored by: OpenCode
type Model struct {
	deps          Dependencies
	width         int
	height        int
	theme         component.Theme
	active        activeScreen
	currentConfig *configmodel.AppSetupConfig
	setup         setupState
	sync          syncState
	syncReports   syncReportsContextState
	replacement   serverReplacementState
	result        resultState
	report        reportState
	spinner       spinner.Model
}

// NewModel creates the root Bubble Tea model for the current slice.
//
// Example:
//
//	model := flow.NewModel(deps)
//	_ = model
//
// Authored by: OpenCode
func NewModel(deps Dependencies) *Model {
	var model = &Model{
		deps:    deps,
		width:   deps.Options.InitialWindowWidth,
		height:  deps.Options.InitialWindowHeight,
		theme:   component.DefaultTheme(),
		spinner: spinner.New(spinner.WithSpinner(spinner.Line)),
	}
	model.setup = newSetupState(nil, deps.Startup.SetupRequirementReason)
	model.sync = newSyncState()

	if deps.Startup.ActiveConfig != nil {
		var config = *deps.Startup.ActiveConfig
		model.currentConfig = &config
		model.active = mainMenuScreenKey
	} else {
		model.active = setupScreenKey
	}
	model.syncReports = newSyncReportsContextState(model.currentServerOrigin(), deps.SyncService.ProtectedDataState())
	model.report = newReportState(model.syncReports.ProtectedData.AvailableReportYears)

	return model
}

// Init initializes the Bubble Tea model.
//
// Example:
//
//	cmd := model.Init()
//	_ = cmd
//
// Authored by: OpenCode
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update applies the next Bubble Tea message to the root model.
//
// Example:
//
//	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
//	_, _ = updated, cmd
//
// Authored by: OpenCode
func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMessage := message.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMessage.Width
		m.height = typedMessage.Height
		return m, nil
	case tea.KeyPressMsg:
		if typedMessage.String() == "ctrl+c" {
			m.cancelActiveSync()
			m.sync.TokenInput.Reset()
			m.clearSyncReportsRuntimeState()
			return m, quitCmd
		}
	}

	switch m.active {
	case setupScreenKey:
		return m.updateSetup(message)
	case mainMenuScreenKey:
		return m.updateMainMenu(message)
	case syncReportsUnlockScreenKey:
		return m.updateSync(message)
	case syncReportsMenuScreenKey:
		return m.updateSyncReportsMenu(message)
	case reportSelectionScreenKey, reportBusyScreenKey, reportResultScreenKey:
		return m.updateReport(message)
	case syncScreenKey:
		return m.updateSync(message)
	case serverReplacementScreenKey:
		return m.updateServerReplacement(message)
	case syncResultScreenKey:
		return m.updateSyncResult(message)
	default:
		return m, nil
	}
}

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
		rendered = screen.SetupScreenView(
			screen.SetupScreenParams{
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
			},
		)
	case mainMenuScreenKey:
		rendered = screen.MainMenuScreenView(
			screen.MainMenuScreenParams{
				Theme:         m.theme,
				Width:         m.width,
				Height:        m.height,
				MenuItems:     m.mainMenuItems(),
				SelectedIndex: 0,
				ServerOrigin:  m.currentServerOrigin(),
				HelpText:      m.mainMenuHelpText(),
			},
		)
	case syncReportsUnlockScreenKey:
		rendered = screen.SyncEntryScreenView(
			screen.SyncEntryScreenParams{
				Theme:                   m.theme,
				Width:                   m.width,
				Height:                  m.height,
				ScreenTitle:             "Sync and Reports",
				ScreenSubtitle:          "Unlock the active sync and reporting context.",
				IntroText:               "Enter the Ghostfolio security token once to unlock Sync Data and future reporting actions for this run.",
				IdleStatusText:          "Enter the Ghostfolio security token to unlock Sync and Reports for this run.",
				ShowProtectedDataStatus: false,
				MenuItems:               m.syncMenuItems(),
				SelectedIndex:           m.sync.MenuIndex,
				TokenInput:              m.sync.TokenInput.View(),
				ValidationMessage:       m.sync.ValidationMessage,
				HelpText:                m.syncHelpText(),
				Busy:                    m.sync.Busy,
				BusyText:                m.sync.BusyText,
				SpinnerFrame:            m.spinner.View(),
				ProtectedDataExists:     m.deps.SyncService.ProtectedDataState().HasReadableSnapshot,
			},
		)
	case syncScreenKey:
		var syncCopy = component.DefaultSyncEntryCopy(m.sync.UseContextToken)
		rendered = screen.SyncEntryScreenView(
			screen.SyncEntryScreenParams{
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
			},
		)
	case syncReportsMenuScreenKey:
		rendered = screen.SyncReportsScreenView(
			screen.SyncReportsScreenParams{
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
			},
		)
	case reportSelectionScreenKey:
		rendered = screen.ReportSelectionScreenView(
			screen.ReportSelectionScreenParams{
				Theme:             m.theme,
				Width:             m.width,
				Height:            m.height,
				AvailableYears:    m.syncReports.ProtectedData.AvailableReportYears,
				SelectedYearIndex: m.report.YearIndex,
				MethodItems:       m.reportMethodItems(),
				SelectedMethod:    m.report.MethodIndex,
				MethodExplanation: reportMethodForIndex(m.report.MethodIndex).Explanation(),
				MenuItems:         m.reportSelectionMenuItems(),
				SelectedAction:    m.report.ActionIndex,
				HelpText:          m.reportSelectionHelpText(),
			},
		)
	case reportBusyScreenKey:
		rendered = screen.ReportBusyScreenView(
			screen.ReportBusyScreenParams{
				Theme:        m.theme,
				Width:        m.width,
				Height:       m.height,
				SelectedYear: m.report.SelectedYear,
				MethodLabel:  reportMethodForIndex(m.report.MethodIndex).Label(),
				BusyText:     m.report.BusyText,
				SpinnerFrame: m.spinner.View(),
				HelpText:     m.reportBusyHelpText(),
			},
		)
	case reportResultScreenKey:
		rendered = screen.ReportResultScreenView(
			screen.ReportResultScreenParams{
				Theme:         m.theme,
				Width:         m.width,
				Height:        m.height,
				Outcome:       m.syncReports.ReportResult,
				MethodLabel:   m.syncReports.ReportResult.Request.CostBasisMethod.Label(),
				MenuItems:     m.reportResultMenuItems(),
				SelectedIndex: m.report.ActionIndex,
				HelpText:      m.reportResultHelpText(),
			},
		)
	case serverReplacementScreenKey:
		rendered = screen.ServerReplacementScreenView(
			screen.ServerReplacementScreenParams{
				Theme:         m.theme,
				Width:         m.width,
				Height:        m.height,
				MenuItems:     m.serverReplacementMenuItems(),
				SelectedIndex: m.replacement.MenuIndex,
				CurrentServer: m.replacement.CurrentServer,
				NewServer:     m.replacement.NewServer,
				HelpText:      m.serverReplacementHelpText(),
			},
		)
	case syncResultScreenKey:
		rendered = screen.SyncResultScreenView(
			screen.SyncResultScreenParams{
				Theme:         m.theme,
				Width:         m.width,
				Height:        m.height,
				MenuItems:     m.resultMenuItems(),
				SelectedIndex: m.result.MenuIndex,
				Outcome:       m.result.Outcome,
				Busy:          m.result.Busy,
				StatusMessage: m.result.StatusMessage,
				HelpText:      m.resultHelpText(),
			},
		)
	default:
		rendered = ""
	}

	var view = tea.NewView(rendered)
	view.AltScreen = true
	view.WindowTitle = "ghostfolio-cryptogains"
	return view
}

// ActiveScreen returns the current workflow screen identifier.
//
// Example:
//
//	name := model.ActiveScreen()
//	_ = name
//
// Authored by: OpenCode
func (m *Model) ActiveScreen() string {
	return string(m.active)
}

// currentServerOrigin returns the active origin summary for the current run.
// Authored by: OpenCode
func (m *Model) currentServerOrigin() string {
	if m.currentConfig == nil {
		return configmodel.GhostfolioCloudOrigin
	}
	return m.currentConfig.ServerOrigin
}

// setupInvalidMessage maps structured startup state into setup-screen wording.
// Authored by: OpenCode
func setupInvalidMessage(reason bootstrap.SetupRequirementReason) string {
	if reason == bootstrap.SetupRequirementInvalidRememberedSetup {
		return "The saved server selection is no longer valid. Complete setup again before Sync Data can run."
	}
	return ""
}

// setupMenuItems builds the primary setup actions for the current render.
// Authored by: OpenCode
func (m *Model) setupMenuItems() []component.MenuItem {
	return []component.MenuItem{
		{Label: "Use Ghostfolio Cloud", Enabled: true},
		{Label: "Use Custom Server", Enabled: true},
		{Label: "Save And Continue", Enabled: m.setupCanSave()},
	}
}

// mainMenuItems builds the primary main-menu actions for the current render.
// Authored by: OpenCode
func (m *Model) mainMenuItems() []component.MenuItem {
	return []component.MenuItem{{Label: "Sync and Reports", Enabled: true}}
}

// syncMenuItems builds the primary sync-entry actions for the current render.
// Authored by: OpenCode
func (m *Model) syncMenuItems() []component.MenuItem {
	if m.sync.Busy {
		return nil
	}
	if m.active == syncReportsUnlockScreenKey {
		return []component.MenuItem{
			{Label: "Unlock", Enabled: true},
			{Label: "Back", Enabled: true},
		}
	}
	return []component.MenuItem{
		{Label: "Start Sync", Enabled: true},
		{Label: "Back", Enabled: true},
	}
}

// resultMenuItems builds the primary sync-result actions for the current render.
// Authored by: OpenCode
func (m *Model) resultMenuItems() []component.MenuItem {
	if m.result.Busy {
		return []component.MenuItem{
			{Label: "Generate Diagnostic Report", Enabled: false},
			{Label: "Sync Again", Enabled: false},
			{Label: "Back To Main Menu", Enabled: false},
		}
	}
	if m.result.Outcome.Diagnostic.Eligible && m.result.Outcome.Diagnostic.Path == "" {
		return []component.MenuItem{
			{Label: "Generate Diagnostic Report", Enabled: true},
			{Label: "Sync Again", Enabled: true},
			{Label: "Back To Main Menu", Enabled: true},
		}
	}

	return []component.MenuItem{
		{Label: "Sync Again", Enabled: true},
		{Label: "Back To Main Menu", Enabled: true},
	}
}

// syncReportsMenuItems builds the primary unlocked-context actions.
// Authored by: OpenCode
func (m *Model) syncReportsMenuItems() []component.MenuItem {
	var reportEnabled = m.syncReports.ProtectedData.HasReadableSnapshot && len(m.syncReports.ProtectedData.AvailableReportYears) > 0
	var contextBusy = m.syncReports.SyncResult.Busy
	var items = []component.MenuItem{
		{Label: "Sync Data", Enabled: !contextBusy},
		{Label: "Generate Capital Gains Report", Enabled: reportEnabled && !contextBusy},
	}
	if m.syncReportsHasPendingDiagnostic() {
		items = append(items, component.MenuItem{Label: "Generate Diagnostic Report", Enabled: !contextBusy})
	}
	items = append(items, component.MenuItem{Label: "Back To Main Menu", Enabled: !contextBusy})
	return items
}

// reportMethodItems builds the supported report method menu.
// Authored by: OpenCode
func (m *Model) reportMethodItems() []component.MenuItem {
	var methods = reportmodel.SupportedCostBasisMethods()
	var items = make([]component.MenuItem, 0, len(methods))
	for _, method := range methods {
		items = append(items, component.MenuItem{Label: method.Label(), Enabled: true})
	}
	return items
}

// reportSelectionMenuItems builds the report-selection action menu.
// Authored by: OpenCode
func (m *Model) reportSelectionMenuItems() []component.MenuItem {
	return []component.MenuItem{{Label: "Generate Report", Enabled: true}, {Label: "Back", Enabled: true}}
}

// reportResultMenuItems builds the completed report-result action menu.
// Authored by: OpenCode
func (m *Model) reportResultMenuItems() []component.MenuItem {
	var items = []component.MenuItem{{Label: "Back To Sync and Reports", Enabled: true}}
	if m.syncReports.ProtectedData.HasReadableSnapshot && len(m.syncReports.ProtectedData.AvailableReportYears) > 0 {
		items = append(items, component.MenuItem{Label: "Generate Another Report", Enabled: true})
	}
	return items
}

// serverReplacementMenuItems builds the primary server-replacement confirmation actions.
// Authored by: OpenCode
func (m *Model) serverReplacementMenuItems() []component.MenuItem {
	return []component.MenuItem{{Label: "Continue And Replace", Enabled: true}, {Label: "Cancel", Enabled: true}}
}

// setupHelpText renders the visible hotkeys for the setup screen.
// Authored by: OpenCode
func (m *Model) setupHelpText() string {
	var bindings = []key.Binding{upBinding(), downBinding(), enterBinding(), focusBinding(), quitBinding()}
	if m.currentConfig != nil {
		bindings = append(bindings, cancelBinding())
	}
	return component.RenderHelp(component.ContentWidthForScreen(m.width), component.Bindings{Short: bindings})
}

// mainMenuHelpText renders the visible hotkeys for the main menu.
//
// Authored by: OpenCode
func (m *Model) mainMenuHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{enterBinding(), editSetupBinding(), quitBinding()}},
	)
}

// syncHelpText renders the visible hotkeys for the sync screen.
//
// Authored by: OpenCode
func (m *Model) syncHelpText() string {
	var bindings = []key.Binding{
		upBinding(),
		downBinding(),
		enterBinding(),
		quitBinding(),
	}
	if !m.sync.UseContextToken {
		bindings = append(bindings, focusBinding())
	}

	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{
			Short: bindings,
		},
	)
}

// syncReportsHelpText renders the visible hotkeys for the unlocked Sync and Reports menu.
// Authored by: OpenCode
func (m *Model) syncReportsHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

// reportSelectionHelpText renders the visible hotkeys for report selection.
// Authored by: OpenCode
func (m *Model) reportSelectionHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), focusBinding(), quitBinding()}},
	)
}

// reportBusyHelpText renders the visible hotkeys for report busy state.
// Authored by: OpenCode
func (m *Model) reportBusyHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{quitBinding()}},
	)
}

// reportResultHelpText renders the visible hotkeys for report result navigation.
// Authored by: OpenCode
func (m *Model) reportResultHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

// serverReplacementHelpText renders the visible hotkeys for the server-replacement screen.
// Authored by: OpenCode
func (m *Model) serverReplacementHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

// resultHelpText renders the visible hotkeys for the sync-result screen.
//
// Authored by: OpenCode
func (m *Model) resultHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

// cancelActiveSync aborts the active sync request when one exists.
// Authored by: OpenCode
func (m *Model) cancelActiveSync() {
	if m.sync.Cancel != nil {
		m.sync.Cancel()
		m.sync.Cancel = nil
	}
}

// newSetupState creates the initial setup workflow state.
// Authored by: OpenCode
func newSetupState(config *configmodel.AppSetupConfig, startupReason bootstrap.SetupRequirementReason) setupState {
	var input = textinput.New()
	input.SetWidth(48)
	input.Prompt = ""
	input.Placeholder = "https://your-ghostfolio.example"

	var state = setupState{
		SelectedMode:  configmodel.ServerModeGhostfolioCloud,
		OriginInput:   input,
		StartupReason: startupReason,
	}
	state.OriginInput.SetValue(configmodel.GhostfolioCloudOrigin)

	if config != nil {
		state.SelectedMode = config.ServerMode
		state.OriginInput.SetValue(config.ServerOrigin)
		if config.ServerMode == configmodel.ServerModeCustomOrigin {
			state.MenuIndex = setupMenuCustomOriginIndex
		}
	}

	return state
}

// newSyncState creates the initial sync-entry workflow state.
// Authored by: OpenCode
func newSyncState() syncState {
	var input = textinput.New()
	input.SetWidth(48)
	input.Prompt = ""
	input.Placeholder = "Enter Ghostfolio security token"
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '*'
	return syncState{InputFocused: true, TokenInput: input}
}

// newSyncReportsContextState creates the initial `Sync and Reports` context
// shell for the currently selected server.
// Authored by: OpenCode
func newSyncReportsContextState(serverOrigin string, protectedData runtime.ProtectedDataState) syncReportsContextState {
	return syncReportsContextState{
		SelectedServerOrigin: serverOrigin,
		ProtectedData:        protectedData,
		ReportUnavailable:    runtime.ReportFailureNoSyncedDataAvailable,
	}
}

// newReportState creates the initial report workflow state.
// Authored by: OpenCode
func newReportState(years []int) reportState {
	var state = reportState{FocusArea: 0, MethodIndex: 0}
	if len(years) > 0 {
		state.SelectedYear = years[0]
	}
	return state
}

// clearSyncReportsRuntimeState scrubs token and transient report state for the
// active `Sync and Reports` context.
// Authored by: OpenCode
func (m *Model) clearSyncReportsRuntimeState() {
	m.syncReports = newSyncReportsContextState(m.currentServerOrigin(), m.deps.SyncService.ProtectedDataState())
	m.report = newReportState(m.syncReports.ProtectedData.AvailableReportYears)
}

// enterSetup routes the application to the setup workflow.
// Authored by: OpenCode
func (m *Model) enterSetup(message string, startupReason bootstrap.SetupRequirementReason) tea.Cmd {
	m.active = setupScreenKey
	m.setup = newSetupState(m.currentConfig, startupReason)
	m.setup.ValidationMessage = message
	return nil
}

// enterMainMenu routes the application back to the main menu.
// Authored by: OpenCode
func (m *Model) enterMainMenu() {
	m.active = mainMenuScreenKey
	m.result = resultState{}
	m.sync = newSyncState()
	m.clearSyncReportsRuntimeState()
	m.sync.InputFocused = false
	m.setup.ValidationMessage = ""
	m.setup.StartupReason = bootstrap.SetupRequirementNone
}

// enterSyncReportsUnlock routes the application to the token-unlock entry
// screen for the active `Sync and Reports` context.
// Authored by: OpenCode
func (m *Model) enterSyncReportsUnlock() tea.Cmd {
	m.active = syncReportsUnlockScreenKey
	m.sync = newSyncState()
	m.clearSyncReportsRuntimeState()
	return m.sync.TokenInput.Focus()
}

// enterSyncReportsMenu routes the application to the unlocked Sync and Reports context menu.
// Authored by: OpenCode
func (m *Model) enterSyncReportsMenu(unlocked runtime.SyncReportsContextResult, token string) tea.Cmd {
	m.active = syncReportsMenuScreenKey
	m.sync = newSyncState()
	m.sync.InputFocused = false
	m.sync.TokenInput.Blur()
	m.sync.MenuIndex = 0
	m.syncReports.Active = true
	m.syncReports.RuntimeToken = token
	m.syncReports.SelectedServerOrigin = m.currentServerOrigin()
	m.syncReports.ProtectedData = unlocked.ProtectedData
	m.syncReports.ReportUnavailable = unlocked.ReportUnavailableReason
	m.report = newReportState(unlocked.ProtectedData.AvailableReportYears)
	return nil
}

// enterReportSelection routes the application to the report-selection screen.
// Authored by: OpenCode
func (m *Model) enterReportSelection() {
	m.active = reportSelectionScreenKey
	m.report = newReportState(m.syncReports.ProtectedData.AvailableReportYears)
	m.report.ActionIndex = 0
}

// enterReportResult routes the application to the report result screen.
// Authored by: OpenCode
func (m *Model) enterReportResult(outcome runtime.ReportOutcome) {
	m.active = reportResultScreenKey
	m.report.Busy = false
	m.report.BusyText = ""
	m.report.AttemptID = ""
	m.report.ActionIndex = 0
	m.syncReports.ReportResult = outcome
}

// enterSync routes the application to the sync entry screen.
// Authored by: OpenCode
func (m *Model) enterSync() tea.Cmd {
	m.active = syncScreenKey
	m.sync = newSyncState()
	return m.sync.TokenInput.Focus()
}

// enterSyncWithContextToken routes the application to the sync entry screen in
// token-free context mode, reusing the active `Sync and Reports` token without
// exposing it in the renderer or input state.
// Authored by: OpenCode
func (m *Model) enterSyncWithContextToken() tea.Cmd {
	m.active = syncScreenKey
	m.sync = newSyncState()
	m.sync.UseContextToken = true
	m.sync.InputFocused = false
	m.sync.TokenInput.Blur()
	m.sync.MenuIndex = 0
	return nil
}

// enterServerReplacement routes the application to the server-mismatch confirmation screen.
// Authored by: OpenCode
func (m *Model) enterServerReplacement(check runtime.ServerReplacementCheck, pendingToken string) {
	m.active = serverReplacementScreenKey
	m.replacement = serverReplacementState{
		PendingToken:  pendingToken,
		CurrentServer: check.ActiveServerOrigin,
		NewServer:     check.SelectedServerOrigin,
	}
}

// enterSyncResult routes the application to the sync result screen.
// Authored by: OpenCode
func (m *Model) enterSyncResult(outcome runtime.SyncOutcome) {
	m.active = syncResultScreenKey
	m.result = resultState{Outcome: outcome}
	if outcome.Success {
		m.result.MenuIndex = 1
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

// syncReportsHasPendingDiagnostic reports whether the active Sync and Reports
// context should offer explicit synced-data diagnostic generation.
// Authored by: OpenCode
func (m *Model) syncReportsHasPendingDiagnostic() bool {
	return m.syncReports.SyncResult.Outcome.Diagnostic.Eligible && m.syncReports.SyncResult.Outcome.Diagnostic.Path == ""
}

// syncReportsDefaultMenuIndex returns the preferred unlocked-context selection
// after one sync attempt completes.
// Authored by: OpenCode
func (m *Model) syncReportsDefaultMenuIndex() int {
	if m.syncReportsHasPendingDiagnostic() {
		return 2
	}

	return 0
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

// nextAttemptID returns a process-local identifier for the next sync attempt.
// Authored by: OpenCode
func nextAttemptID() string {
	return fmt.Sprintf("attempt-%d", time.Now().UnixNano())
}

// reportMethodForIndex returns the supported method for one stable menu index.
// Authored by: OpenCode
func reportMethodForIndex(index int) reportmodel.CostBasisMethod {
	var methods = reportmodel.SupportedCostBasisMethods()
	if index < 0 || index >= len(methods) {
		return ""
	}
	return methods[index]
}

// selectedSetupOrigin returns the currently selected setup origin.
// Authored by: OpenCode
func (m *Model) selectedSetupOrigin() string {
	if m.setup.SelectedMode == configmodel.ServerModeGhostfolioCloud {
		return configmodel.GhostfolioCloudOrigin
	}
	return strings.TrimSpace(m.setup.OriginInput.Value())
}

// setupCanSave reports whether the current setup selection is valid for persistence.
// Authored by: OpenCode
func (m *Model) setupCanSave() bool {
	var _, err = configmodel.NormalizeOrigin(m.selectedSetupOrigin(), m.deps.Options.AllowDevHTTP)
	return err == nil
}
