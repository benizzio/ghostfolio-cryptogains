// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

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
	return []component.MenuItem{{Label: component.SyncAndReportsActionLabel, Enabled: true}}
}

// syncMenuItems builds the primary sync-entry actions for the current render.
// Authored by: OpenCode
func (m *Model) syncMenuItems() []component.MenuItem {
	if m.sync.Busy {
		return nil
	}
	if m.active == syncReportsUnlockScreenKey {
		var unlockEnabled = m.syncReports.UnlockFailure != runtime.SyncFailureRejectedToken
		return []component.MenuItem{
			{Label: component.UnlockActionLabel, Enabled: unlockEnabled},
			{Label: component.BackActionLabel, Enabled: true},
		}
	}
	return []component.MenuItem{
		{Label: component.StartSyncActionLabel, Enabled: true},
		{Label: component.BackActionLabel, Enabled: true},
	}
}

// resultMenuItems builds the primary sync-result actions for the current render.
// Authored by: OpenCode
func (m *Model) resultMenuItems() []component.MenuItem {
	if m.result.Busy {
		return []component.MenuItem{
			{Label: component.GenerateDiagnosticReportActionLabel, Enabled: false},
			{Label: component.SyncAgainActionLabel, Enabled: false},
			{Label: component.BackToMainMenuActionLabel, Enabled: false},
		}
	}
	if m.result.Outcome.Diagnostic.Eligible && m.result.Outcome.Diagnostic.Path == "" {
		return []component.MenuItem{
			{Label: component.GenerateDiagnosticReportActionLabel, Enabled: true},
			{Label: component.SyncAgainActionLabel, Enabled: true},
			{Label: component.BackToMainMenuActionLabel, Enabled: true},
		}
	}

	return []component.MenuItem{
		{Label: component.SyncAgainActionLabel, Enabled: true},
		{Label: component.BackToMainMenuActionLabel, Enabled: true},
	}
}

// syncReportsMenuItems builds the primary unlocked-context actions.
// Authored by: OpenCode
func (m *Model) syncReportsMenuItems() []component.MenuItem {
	var reportEnabled = m.syncReports.ProtectedData.HasReadableSnapshot && len(m.syncReports.ProtectedData.AvailableReportYears) > 0
	var contextBusy = m.syncReports.SyncResult.Busy
	var items = []component.MenuItem{
		{Label: component.SyncDataActionLabel, Enabled: !contextBusy},
		{Label: component.GenerateCapitalGainsReportActionLabel, Enabled: reportEnabled && !contextBusy},
	}
	if m.syncReportsHasPendingDiagnostic() {
		items = append(items, component.MenuItem{Label: component.GenerateDiagnosticReportActionLabel, Enabled: !contextBusy})
	}
	items = append(items, component.MenuItem{Label: component.BackToMainMenuActionLabel, Enabled: !contextBusy})
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

// reportBaseCurrencyItems builds the supported report base-currency menu.
// Authored by: OpenCode
func (m *Model) reportBaseCurrencyItems() []component.MenuItem {
	var currencies = reportmodel.SupportedReportBaseCurrencies()
	var items = make([]component.MenuItem, 0, len(currencies))
	for _, currency := range currencies {
		items = append(items, component.MenuItem{Label: currency.Label(), Enabled: true})
	}
	return items
}

// reportOutputFormatItems builds the supported report output-format menu.
// Authored by: OpenCode
func (m *Model) reportOutputFormatItems() []component.MenuItem {
	var formats = reportmodel.SupportedReportOutputFormats()
	var items = make([]component.MenuItem, 0, len(formats))
	for _, format := range formats {
		items = append(items, component.MenuItem{Label: format.Label(), Enabled: true})
	}
	return items
}

// reportSelectionMenuItems builds the report-selection action menu.
// Authored by: OpenCode
func (m *Model) reportSelectionMenuItems() []component.MenuItem {
	return []component.MenuItem{{Label: component.GenerateReportActionLabel, Enabled: m.reportCanGenerate()}, {Label: component.BackActionLabel, Enabled: true}}
}

// reportCanGenerate reports whether the selection screen has all required user
// choices for starting report generation.
// Authored by: OpenCode
func (m *Model) reportCanGenerate() bool {
	return m.report.SelectedYear > 0 && m.report.MethodIndex >= 0 && m.report.SelectedBaseCurrency != "" && m.report.SelectedOutputFormat != ""
}

// reportResultMenuItems builds the completed report-result action menu.
// Authored by: OpenCode
func (m *Model) reportResultMenuItems() []component.MenuItem {
	var items []component.MenuItem
	if m.report.Busy {
		items = append(items, component.MenuItem{Label: component.GenerateDiagnosticReportActionLabel, Enabled: false})
	}
	if m.reportOutcomeHasPendingDiagnostic() {
		items = append(items, component.MenuItem{Label: component.GenerateDiagnosticReportActionLabel, Enabled: true})
	}
	items = append(items, component.MenuItem{Label: component.BackToSyncReportsActionLabel, Enabled: !m.report.Busy})
	if m.syncReports.ProtectedData.HasReadableSnapshot && len(m.syncReports.ProtectedData.AvailableReportYears) > 0 {
		items = append(items, component.MenuItem{Label: component.GenerateAnotherReportActionLabel, Enabled: !m.report.Busy})
	}
	return items
}

// serverReplacementMenuItems builds the primary server-replacement confirmation actions.
// Authored by: OpenCode
func (m *Model) serverReplacementMenuItems() []component.MenuItem {
	return []component.MenuItem{{Label: component.ContinueAndReplaceActionLabel, Enabled: true}, {Label: component.CancelActionLabel, Enabled: true}}
}
