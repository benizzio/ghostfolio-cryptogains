package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

func TestValidationResultWorkflowContract(t *testing.T) {
	t.Parallel()

	var success = screen.ValidationResultScreenView(screen.ValidationResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     100,
		Height:    32,
		MenuItems: []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome:   runtime.ValidationOutcome{Success: true, DetailReason: "activity_data_stored"},
	})
	assertContains(t, success, "Sync Again")
	assertContains(t, success, "Back To Main Menu")
	assertContains(t, success, "stored securely for future use")

	var failure = screen.ValidationResultScreenView(screen.ValidationResultScreenParams{
		Theme:     component.DefaultTheme(),
		Width:     100,
		Height:    32,
		MenuItems: []component.MenuItem{{Label: "Sync Again", Enabled: true}, {Label: "Back To Main Menu", Enabled: true}},
		Outcome:   runtime.ValidationOutcome{Success: false, FailureReason: runtime.ValidationFailureTimeout, DetailReason: string(runtime.ValidationFailureTimeout)},
	})
	assertContains(t, failure, "Failure Category: timeout")
}
