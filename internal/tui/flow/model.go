// Package flow owns the Bubble Tea root model and workflow routing for this
// validation-only slice.
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
	syncValidationScreenKey    activeScreen = "sync_validation"
	serverReplacementScreenKey activeScreen = "server_replacement"
	validationResultScreenKey  activeScreen = "validation_result"

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

// validationFinishedMsg reports the result of an asynchronous validation run.
// Authored by: OpenCode
type validationFinishedMsg struct {
	Outcome runtime.ValidationOutcome
	Attempt string
}

// Dependencies contains the runtime services required by the root Bubble Tea
// model.
//
// Authored by: OpenCode
type Dependencies struct {
	Options      bootstrap.Options
	Startup      bootstrap.StartupState
	SetupService runtime.SetupService
	SyncService  runtime.SyncService
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
	ValidationMessage string
	Busy              bool
	BusyText          string
	AttemptID         string
	Cancel            context.CancelFunc
}

// serverReplacementState holds transient UI state for server-mismatch confirmation.
// Authored by: OpenCode
type serverReplacementState struct {
	MenuIndex     int
	PendingToken  string
	CurrentServer string
	NewServer     string
}

// resultState holds transient UI state for the validation-result screen.
// Authored by: OpenCode
type resultState struct {
	MenuIndex int
	Outcome   runtime.ValidationOutcome
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
	replacement   serverReplacementState
	result        resultState
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
			m.cancelActiveValidation()
			m.sync.TokenInput.Reset()
			return m, quitCmd
		}
	}

	switch m.active {
	case setupScreenKey:
		return m.updateSetup(message)
	case mainMenuScreenKey:
		return m.updateMainMenu(message)
	case syncValidationScreenKey:
		return m.updateSyncValidation(message)
	case serverReplacementScreenKey:
		return m.updateServerReplacement(message)
	case validationResultScreenKey:
		return m.updateValidationResult(message)
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
				Theme:               m.theme,
				Width:               m.width,
				Height:              m.height,
				MenuItems:           m.mainMenuItems(),
				SelectedIndex:       0,
				ServerOrigin:        m.currentServerOrigin(),
				ProtectedDataExists: m.deps.SyncService.ProtectedDataState().HasReadableSnapshot,
				HelpText:            m.mainMenuHelpText(),
			},
		)
	case syncValidationScreenKey:
		rendered = screen.SyncValidationScreenView(
			screen.SyncValidationScreenParams{
				Theme:               m.theme,
				Width:               m.width,
				Height:              m.height,
				MenuItems:           m.syncMenuItems(),
				SelectedIndex:       m.sync.MenuIndex,
				TokenInput:          m.sync.TokenInput.View(),
				ValidationMessage:   m.sync.ValidationMessage,
				HelpText:            m.syncHelpText(),
				Busy:                m.sync.Busy,
				BusyText:            m.sync.BusyText,
				SpinnerFrame:        m.spinner.View(),
				ProtectedDataExists: m.deps.SyncService.ProtectedDataState().HasReadableSnapshot,
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
	case validationResultScreenKey:
		rendered = screen.ValidationResultScreenView(
			screen.ValidationResultScreenParams{
				Theme:         m.theme,
				Width:         m.width,
				Height:        m.height,
				MenuItems:     m.resultMenuItems(),
				SelectedIndex: m.result.MenuIndex,
				Outcome:       m.result.Outcome,
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
		return "The saved server selection is no longer valid. Complete setup again before sync validation can run."
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
	return []component.MenuItem{{Label: "Sync Data", Enabled: true}}
}

// syncMenuItems builds the primary sync-entry actions for the current render.
// Authored by: OpenCode
func (m *Model) syncMenuItems() []component.MenuItem {
	if m.sync.Busy {
		return nil
	}
	return []component.MenuItem{
		{Label: "Start Sync", Enabled: true},
		{Label: "Back", Enabled: true},
	}
}

// resultMenuItems builds the primary validation-result actions for the current render.
// Authored by: OpenCode
func (m *Model) resultMenuItems() []component.MenuItem {
	return []component.MenuItem{
		{Label: "Sync Again", Enabled: true},
		{Label: "Back To Main Menu", Enabled: true},
	}
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
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{
			Short: []key.Binding{
				upBinding(),
				downBinding(),
				enterBinding(),
				focusBinding(),
				quitBinding(),
			},
		},
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

// resultHelpText renders the visible hotkeys for the validation-result screen.
//
// Authored by: OpenCode
func (m *Model) resultHelpText() string {
	return component.RenderHelp(
		component.ContentWidthForScreen(m.width),
		component.Bindings{Short: []key.Binding{upBinding(), downBinding(), enterBinding(), quitBinding()}},
	)
}

// cancelActiveValidation aborts the active validation request when one exists.
// Authored by: OpenCode
func (m *Model) cancelActiveValidation() {
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
	m.result.MenuIndex = 0
	m.sync = newSyncState()
	m.sync.InputFocused = false
	m.setup.ValidationMessage = ""
	m.setup.StartupReason = bootstrap.SetupRequirementNone
}

// enterSyncValidation routes the application to the sync-validation entry screen.
// Authored by: OpenCode
func (m *Model) enterSyncValidation() tea.Cmd {
	m.active = syncValidationScreenKey
	m.sync = newSyncState()
	return m.sync.TokenInput.Focus()
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

// enterValidationResult routes the application to the validation result screen.
// Authored by: OpenCode
func (m *Model) enterValidationResult(outcome runtime.ValidationOutcome) {
	m.active = validationResultScreenKey
	m.result = resultState{Outcome: outcome}
	if outcome.Success {
		m.result.MenuIndex = 1
	}
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

// validationCmd delegates a single sync-validation attempt to the application service.
// Authored by: OpenCode
func (m *Model) validationCmd(ctx context.Context, attemptID string, request runtime.ValidateRequest) tea.Cmd {
	return func() tea.Msg {
		return validationFinishedMsg{
			Outcome: m.deps.SyncService.Validate(ctx, request),
			Attempt: attemptID,
		}
	}
}

// nextAttemptID returns a process-local identifier for the next validation attempt.
// Authored by: OpenCode
func nextAttemptID() string {
	return fmt.Sprintf("attempt-%d", time.Now().UnixNano())
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
