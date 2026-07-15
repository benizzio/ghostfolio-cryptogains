// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import "fmt"

// ReportOutputFormat identifies the user-selected output format for one report
// generation run. Use SupportedReportOutputFormats to render the complete set
// of selectable formats and Validate before starting report calculation.
//
// Example:
//
//	for _, format := range model.SupportedReportOutputFormats() {
//		fmt.Println(format.Label())
//	}
//
// Authored by: OpenCode
type ReportOutputFormat string

const (
	// ReportOutputFormatMarkdown identifies the Markdown main-plus-annex output.
	ReportOutputFormatMarkdown ReportOutputFormat = "markdown"

	// ReportOutputFormatPDF identifies the combined PDF output.
	ReportOutputFormatPDF ReportOutputFormat = "pdf"
)

// Label returns the user-facing label for a supported report output format.
// Unsupported values return an empty label so callers can fail validation before
// rendering user-visible output.
// Authored by: OpenCode
func (format ReportOutputFormat) Label() string {
	switch format {
	case ReportOutputFormatMarkdown:
		return "Markdown"
	case ReportOutputFormatPDF:
		return "PDF"
	default:
		return ""
	}
}

// Validate verifies that the report output format is one of the supported
// transient generation choices.
// Authored by: OpenCode
func (format ReportOutputFormat) Validate() error {
	return validateReportOutputFormat(format)
}

// SupportedReportOutputFormats returns the complete supported output-format
// list in stable UI order.
//
// Example:
//
//	formats := model.SupportedReportOutputFormats()
//	_ = formats[0].Label()
//
// Authored by: OpenCode
func SupportedReportOutputFormats() []ReportOutputFormat {
	return []ReportOutputFormat{
		ReportOutputFormatMarkdown,
		ReportOutputFormatPDF,
	}
}

// validateReportOutputFormat rejects missing or unsupported report output
// format selections.
// Authored by: OpenCode
func validateReportOutputFormat(format ReportOutputFormat) error {
	switch format {
	case ReportOutputFormatMarkdown, ReportOutputFormatPDF:
		return nil
	case "":
		return fmt.Errorf("report output format is required")
	default:
		return fmt.Errorf("unsupported report output format %q", format)
	}
}
