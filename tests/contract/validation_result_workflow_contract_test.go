package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestValidationResultWorkflowContract(t *testing.T) {
	t.Parallel()

	var success = screen.ValidationResultScreenView(screen.ValidationResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     100,
		Height:    32,
		MenuItems: []component.MenuItem{{Label: "Validate Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome:   runtime.ValidationOutcome{Success: true, SummaryMessage: "Communication with the selected Ghostfolio server is working.", FollowUpNote: "No Ghostfolio data was stored locally, and reporting is not available in this slice."},
	})
	assertContains(t, success, "Validate Again")
	assertContains(t, success, "Back To Main Menu")
	assertContains(t, success, "No Ghostfolio data was stored locally")

	var failure = screen.ValidationResultScreenView(screen.ValidationResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     100,
		Height:    32,
		MenuItems: []component.MenuItem{{Label: "Validate Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome:   runtime.ValidationOutcome{Success: false, FailureCategory: ghostfolioclient.FailureTimeout, SummaryMessage: "Communication validation did not succeed.", FollowUpNote: "Validate again."},
	})
	assertContains(t, failure, "Failure Category: timeout")
}
