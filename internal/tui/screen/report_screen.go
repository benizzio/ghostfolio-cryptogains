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
	Theme                     component.Theme
	Width                     int
	Height                    int
	AvailableYears            []int
	SelectedYearIndex         int
	MethodItems               []component.MenuItem
	SelectedMethod            int
	BaseCurrencyItems         []component.MenuItem
	SelectedBaseCurrencyIndex int
	OutputFormatItems         []component.MenuItem
	SelectedOutputFormatIndex int
	SelectedOutputFormat      reportmodel.ReportOutputFormat
	MethodExplanation         string
	MenuItems                 []component.MenuItem
	SelectedAction            int
	HelpText                  string
}

// ReportBusyScreenParams contains the render state for one report-generation
// busy screen.
// Authored by: OpenCode
type ReportBusyScreenParams struct {
	Theme              component.Theme
	Width              int
	Height             int
	SelectedYear       int
	MethodLabel        string
	ReportBaseCurrency reportmodel.ReportBaseCurrency
	OutputFormat       reportmodel.ReportOutputFormat
	BusyText           string
	SpinnerFrame       string
	HelpText           string
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

// ReportSelectionScreenView renders the year, method, and base-currency
// selection workflow.
//
// Example:
//
//	view := screen.ReportSelectionScreenView(params)
//	_ = view
//
// Authored by: OpenCode
func ReportSelectionScreenView(params ReportSelectionScreenParams) string {
	var body = fmt.Sprintf(
		"Select one report year, one cost basis method, one report base currency, and one output format.\n\nAvailable Years\n%s\n\nCost Basis Methods\n%s\n\nReport Base Currency\n%s\n\nOutput Format\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
		renderReportYears(params.Theme, params.AvailableYears, params.SelectedYearIndex),
		component.RenderMenu(params.Theme, params.MethodItems, params.SelectedMethod),
		component.RenderMenu(params.Theme, reportBaseCurrencyItems(params.BaseCurrencyItems), params.SelectedBaseCurrencyIndex),
		component.RenderMenu(params.Theme, params.OutputFormatItems, params.SelectedOutputFormatIndex),
		reportMethodExplanation(params.MethodExplanation),
		reportBaseCurrencyExplanation(),
		reportOutputFormatExplanation(params.SelectedOutputFormat),
		component.RenderMenu(params.Theme, params.MenuItems, params.SelectedAction),
	)

	return component.RenderScreen(
		params.Theme,
		params.Width,
		params.Height,
		"Generate Capital Gains Report",
		"Choose one year, one supported cost basis method, one report base currency, and one output format.",
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
		"%s %s\n\nSelected Year: %d\nCost Basis Method: %s\nReport Base Currency: %s\nOutput Format: %s",
		params.SpinnerFrame,
		params.BusyText,
		params.SelectedYear,
		params.MethodLabel,
		reportBusyBaseCurrencyLabel(params.ReportBaseCurrency),
		reportOutputFormatLabel(params.OutputFormat),
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
		"%s\n\nSelected Year: %d\nCost Basis Method: %s\nReport Base Currency: %s\nOutput Format: %s\n\n%s\n\n%s",
		resultLine,
		params.Outcome.Request.Year,
		params.MethodLabel,
		reportResultBaseCurrencyLabel(params.Outcome),
		reportOutputFormatLabel(params.Outcome.Request.OutputFormat),
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

// reportBaseCurrencyItems returns the explicit base-currency menu or the
// default USD/EUR report base-currency menu.
// Authored by: OpenCode
func reportBaseCurrencyItems(items []component.MenuItem) []component.MenuItem {
	if len(items) > 0 {
		return items
	}

	var currencies = reportmodel.SupportedReportBaseCurrencies()
	var defaultItems = make([]component.MenuItem, 0, len(currencies))
	for _, currency := range currencies {
		defaultItems = append(defaultItems, component.MenuItem{Label: currency.Label(), Enabled: true})
	}
	return defaultItems
}

// reportBaseCurrencyExplanation formats the static base-currency explanation.
// Authored by: OpenCode
func reportBaseCurrencyExplanation() string {
	return "Base Currency Explanation\nThe generated report uses the selected base currency; all monetary report calculations and totals will use the selected base currency."
}

// reportOutputFormatExplanation formats the selected output-format explanation.
// Authored by: OpenCode
func reportOutputFormatExplanation(outputFormat reportmodel.ReportOutputFormat) string {
	switch outputFormat {
	case reportmodel.ReportOutputFormatPDF:
		return "Output Format Explanation\nPDF creates one local A4 text PDF containing the main report and Annex 1."
	case reportmodel.ReportOutputFormatMarkdown:
		return "Output Format Explanation\nMarkdown creates one main report file and one separate Annex 1 Markdown file."
	default:
		return "Output Format Explanation\nChoose Markdown or PDF before generation starts."
	}
}

// reportBusyBaseCurrencyLabel formats the selected base currency for busy-state
// rendering.
// Authored by: OpenCode
func reportBusyBaseCurrencyLabel(currency reportmodel.ReportBaseCurrency) string {
	var label = strings.TrimSpace(currency.Label())
	if label != "" {
		return label
	}
	return "not selected"
}

// reportOutputFormatLabel formats the selected output format for busy and
// result rendering.
// Authored by: OpenCode
func reportOutputFormatLabel(outputFormat reportmodel.ReportOutputFormat) string {
	var label = strings.TrimSpace(outputFormat.Label())
	if label != "" {
		return label
	}
	return "not selected"
}

// reportResultBaseCurrencyLabel formats the selected base currency for both
// successful and failed report-result screens.
// Authored by: OpenCode
func reportResultBaseCurrencyLabel(outcome runtime.ReportOutcome) string {
	var label = strings.TrimSpace(outcome.Request.ReportBaseCurrency.Label())
	if label != "" {
		return label
	}
	return "not selected"
}

// reportResultSummary formats the user-visible report result message.
// Authored by: OpenCode
func reportResultSummary(outcome runtime.ReportOutcome) string {
	if outcome.Success {
		if len(outcome.OutputBundle.Files) > 0 {
			return reportOutputBundleSummary(outcome)
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

// reportOutputBundleSummary formats every saved path in a successful output
// bundle.
// Authored by: OpenCode
func reportOutputBundleSummary(outcome runtime.ReportOutcome) string {
	var lines []string
	for _, file := range outcome.OutputBundle.Files {
		var label string
		switch file.Role {
		case reportmodel.ReportDocumentRoleAnnex:
			label = "Saved Annex 1 Markdown Path"
		case reportmodel.ReportDocumentRoleCombined:
			label = "Saved PDF Path"
		default:
			label = "Saved Markdown Path"
		}
		lines = append(lines, fmt.Sprintf("%s: %s", label, file.Path))
	}
	lines = append(lines, outcome.Message)
	return strings.Join(lines, "\n\n")
}
