// Package component contains shared TUI styling and rendering helpers.
// Authored by: OpenCode
package component

// SyncEntryCopy stores the shared `Sync Data` screen copy for one render mode.
//
// Use `IntroText` for the optional explanatory body copy above the primary
// content and `IdleStatusText` for the footer guidance shown before the sync
// starts.
//
// Authored by: OpenCode
type SyncEntryCopy struct {
	IntroText      string
	IdleStatusText string
}

const (
	syncEntryStandaloneIntroText      = "The application will authenticate, retrieve activity history, validate it, and store it securely for future use only."
	syncEntryStandaloneIdleStatusText = "Enter the Ghostfolio security token only when starting Sync Data."
	syncEntryContextIdleStatusText    = "Start Sync to obtain current available activity data on the Ghostfolio server."
)

// DefaultSyncEntryCopy returns the shared `Sync Data` screen copy for
// standalone token entry or unlocked-context token reuse.
//
// Example:
//
//	copy := component.DefaultSyncEntryCopy(true)
//	_, _ = copy.IntroText, copy.IdleStatusText
//
// When `useContextToken` is true, the returned copy omits the intro text and
// keeps only the in-context start prompt. Otherwise it returns the standalone
// token-entry guidance used by the regular sync screen.
//
// Authored by: OpenCode
func DefaultSyncEntryCopy(useContextToken bool) SyncEntryCopy {
	if useContextToken {
		return SyncEntryCopy{IdleStatusText: syncEntryContextIdleStatusText}
	}

	return SyncEntryCopy{
		IntroText:      syncEntryStandaloneIntroText,
		IdleStatusText: syncEntryStandaloneIdleStatusText,
	}
}
