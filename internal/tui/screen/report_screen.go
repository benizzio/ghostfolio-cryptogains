// Package screen renders full-screen workflow states for the terminal
// application.
// Authored by: OpenCode
package screen

import (
	"fmt"
	"strings"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
)

// ReportSelectionScreenParams contains the render state for the report
// selection workflow.
// Authored by: OpenCode
type ReportSelectionScreenParams struct {
	Theme             component.Theme
	Width             int
	Height            int
	AvailableYears    []int
	SelectedYearIndex int
	MethodItems       []component.MenuItem
	SelectedMethod    int
	MethodExplanation string
	MenuItems         []component.MenuItem
	SelectedAction    int
	HelpText          string
}

// ReportBusyScreenParams contains the render state for one report-generation
// busy screen.
// Authored by: OpenCode
type ReportBusyScreenParams struct {
	Theme        component.Theme
	Width        int
	Height       int
	SelectedYear int
	MethodLabel  string
	BusyText     string
	SpinnerFrame string
	HelpText     string
}

// ReportResultScreenParams contains the render state for the report result
// screen.
// Authored by: OpenCode
type ReportResultScreenParams struct {
	Theme         component.Theme
	Width         int
	Height        int
	Outcome       runtime.ReportOutcome
	MethodLabel   string
	MenuItems     []component.MenuItem
	SelectedIndex int
	HelpText      string
}

// ReportSelectionScreenView renders the year and method selection workflow.
//
// Example:
//
//	view := screen.ReportSelectionScreenView(params)
//	_ = view
//
// Authored by: OpenCode
func ReportSelectionScreenView(params ReportSelectionScreenParams) string {
	var body = fmt.Sprintf(
		"Select one report year and one cost basis method.\n\nAvailable Years\n%s\n\nCost Basis Methods\n%s\n\n%s\n\n%s",
		renderReportYears(params.Theme, params.AvailableYears, params.SelectedYearIndex),
		component.RenderMenu(params.Theme, params.MethodItems, params.SelectedMethod),
		reportMethodExplanation(params.MethodExplanation),
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedAction),
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Generate Capital Gains Report",
		"Choose one year and one supported cost basis method.",
		body,
		"Selections are transient. The report content is not previewed in the TUI before save.",
		params.HelpText,
	)
}

// ReportBusyScreenView renders the asynchronous report-generation busy state.
//
// Example:
//
//	view := screen.ReportBusyScreenView(params)
//	_ = view
//
// Authored by: OpenCode
func ReportBusyScreenView(params ReportBusyScreenParams) string {
	var body = fmt.Sprintf(
		"%s %s\n\nSelected Year: %d\nCost Basis Method: %s",
		params.SpinnerFrame,
		params.BusyText,
		params.SelectedYear,
		params.MethodLabel,
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Report Generation",
		"Calculating, rendering, saving, and requesting automatic opening.",
		body,
		"Report generation uses the unlocked protected cache and does not run a new sync.",
		params.HelpText,
	)
}

// ReportResultScreenView renders the result of one completed report-generation
// attempt.
//
// Example:
//
//	view := screen.ReportResultScreenView(params)
//	_ = view
//
// Authored by: OpenCode
func ReportResultScreenView(params ReportResultScreenParams) string {
	var resultLine = params.Theme.SuccessStatus.Render("Success")
	if !params.Outcome.Success {
		resultLine = params.Theme.FailureStatus.Render(fmt.Sprintf("Failure Category: %s", params.Outcome.FailureReason))
	}
	if params.Outcome.Success && params.Outcome.FailureReason == runtime.ReportFailureAutomaticOpenFailedAfterSave {
		resultLine = params.Theme.FailureStatus.Render("Success With Warning: automatic open failed after save")
	}

	var body = fmt.Sprintf(
		"%s\n\nSelected Year: %d\nCost Basis Method: %s\n\n%s\n\n%s",
		resultLine,
		params.Outcome.Request.Year,
		params.MethodLabel,
		reportResultSummary(params.Outcome),
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedIndex),
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Report Result",
		"Review the saved-path outcome and choose the next step.",
		body,
		component.ReportSavedPathsTransientStatusText,
		params.HelpText,
	)
}

// renderReportYears formats the selectable report-year list.
// Authored by: OpenCode
func renderReportYears(theme component.Theme, years []int, selected int) string {
	var lines = make([]string, 0, len(years))
	for index, year := range years {
		var prefix = "  "
		var style = theme.BodyText
		if index == selected {
			prefix = "> "
			style = theme.SelectedItem
		}
		lines = append(lines, style.Render(fmt.Sprintf("%s%d", prefix, year)))
	}
	return strings.Join(lines, "\n")
}

// reportMethodExplanation formats the highlighted method explanation block.
// Authored by: OpenCode
func reportMethodExplanation(explanation string) string {
	var normalized = strings.TrimSpace(explanation)
	if normalized == "" {
		normalized = reportmodel.CostBasisMethod("").Explanation()
	}
	return fmt.Sprintf("Method Explanation\n%s", normalized)
}

// reportResultSummary formats the user-visible report result message.
// Authored by: OpenCode
func reportResultSummary(outcome runtime.ReportOutcome) string {
	if outcome.Success {
		if strings.TrimSpace(outcome.OutputFile.Path) != "" {
			return fmt.Sprintf("Saved Markdown Path: %s\n\n%s", outcome.OutputFile.Path, outcome.Message)
		}
		return outcome.Message
	}

	var lines = []string{outcome.Message}
	if strings.TrimSpace(outcome.Diagnostic.GenerationMessage) != "" {
		lines = append(lines, outcome.Diagnostic.GenerationMessage)
	}
	if strings.TrimSpace(outcome.Diagnostic.Path) != "" {
		lines = append(lines, fmt.Sprintf("Diagnostic Report Path: %s", outcome.Diagnostic.Path))
	}
	if outcome.Diagnostic.Eligible && outcome.Diagnostic.Path == "" && strings.TrimSpace(outcome.Diagnostic.GenerationMessage) == "" {
		lines = append(lines, component.ReportDiagnosticAvailableFromScreenMessage)
	}

	return strings.Join(lines, "\n\n")
}
