// Package flow owns the Bubble Tea root model and workflow routing for this
// sync-and-storage slice.
// Authored by: OpenCode
package flow

import (
	"strings"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

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

// syncReportsUnlockValidationMessage returns the current unlock-screen guidance,
// including the blocked rejected-token state for the current unlock instance.
// Authored by: OpenCode
func (m *Model) syncReportsUnlockValidationMessage() string {
	if m.sync.ValidationMessage != "" {
		return m.sync.ValidationMessage
	}
	if m.syncReports.UnlockFailure == runtime.SyncFailureRejectedToken {
		return "access denied"
	}
	return ""
}

// reportOutcomeHasPendingDiagnostic reports whether the active report result may
// still generate a diagnostics artifact explicitly.
// Authored by: OpenCode
func (m *Model) reportOutcomeHasPendingDiagnostic() bool {
	return m.syncReports.ReportResult.Diagnostic.Eligible && m.syncReports.ReportResult.Diagnostic.Path == ""
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
	return menuIndexForAction(m.syncReportsMenuActions(), m.syncReportsDefaultMenuAction())
}

// cancelActiveSync aborts the active sync request when one exists.
// Authored by: OpenCode
func (m *Model) cancelActiveSync() {
	if m.sync.Cancel != nil {
		m.sync.Cancel()
		m.sync.Cancel = nil
	}
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
