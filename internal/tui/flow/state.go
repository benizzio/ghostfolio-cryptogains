// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"context"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

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
	UnlockFailure        runtime.SyncFailureReason
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
	FocusArea            int
	YearIndex            int
	MethodIndex          int
	BaseCurrencyIndex    int
	OutputFormatIndex    int
	ActionIndex          int
	Busy                 bool
	BusyText             string
	AttemptID            string
	SelectedYear         int
	SelectedBaseCurrency reportmodel.ReportBaseCurrency
	SelectedOutputFormat reportmodel.ReportOutputFormat
	ResultViewport       viewport.Model
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
	var resultViewport = viewport.New()
	resultViewport.SoftWrap = true
	resultViewport.FillHeight = true
	var state = reportState{FocusArea: 0, MethodIndex: 0, BaseCurrencyIndex: 0, OutputFormatIndex: 0, SelectedBaseCurrency: reportBaseCurrencyForIndex(0), SelectedOutputFormat: reportOutputFormatForIndex(0), ResultViewport: resultViewport}
	if len(years) > 0 {
		state.SelectedYear = years[0]
	}
	return state
}
