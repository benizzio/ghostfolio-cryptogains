// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
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
		if m.active == reportResultScreenKey {
			m.refreshReportResultViewport(false)
		}
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
