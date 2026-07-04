// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"
)

// ReportOutputBundle stores the saved-file outcome for one successful report
// generation run. A Markdown bundle must contain main and annex files. A PDF
// bundle must contain the single combined file.
// Authored by: OpenCode
type ReportOutputBundle struct {
	OutputFormat  ReportOutputFormat
	Files         []ReportOutputFile
	SavedAt       time.Time
	OpenRequested bool
	OpenError     string
}

// NewReportOutputBundle creates one validated report output bundle.
//
// Example:
//
//	bundle, err := model.NewReportOutputBundle(model.ReportOutputFormatPDF, []model.ReportOutputFile{file}, time.Now(), false, "")
//	if err != nil {
//		panic(err)
//	}
//	_ = bundle.OutputFormat
//
// Authored by: OpenCode
func NewReportOutputBundle(
	outputFormat ReportOutputFormat,
	files []ReportOutputFile,
	savedAt time.Time,
	openRequested bool,
	openError string,
) (ReportOutputBundle, error) {
	var bundle = ReportOutputBundle{
		OutputFormat:  outputFormat,
		Files:         append([]ReportOutputFile(nil), files...),
		SavedAt:       savedAt,
		OpenRequested: openRequested,
		OpenError:     strings.TrimSpace(openError),
	}

	if err := bundle.Validate(); err != nil {
		return ReportOutputBundle{}, err
	}

	return bundle, nil
}

// Validate verifies one successful output bundle shape before runtime reports
// saved paths to the user.
// Authored by: OpenCode
func (bundle ReportOutputBundle) Validate() error {
	if err := validateReportOutputFormat(bundle.OutputFormat); err != nil {
		return fmt.Errorf("report output bundle format: %w", err)
	}
	if bundle.SavedAt.IsZero() {
		return fmt.Errorf("report output bundle saved-at timestamp is required")
	}
	if !bundle.OpenRequested && bundle.OpenError != "" {
		return fmt.Errorf("report output bundle open error requires an open request")
	}

	for index, file := range bundle.Files {
		if err := file.Validate(); err != nil {
			return fmt.Errorf("report output bundle file %d: %w", index, err)
		}
	}

	return bundle.validateFileShape()
}

// validateFileShape verifies the selected format's required persisted files.
// Authored by: OpenCode
func (bundle ReportOutputBundle) validateFileShape() error {
	switch bundle.OutputFormat {
	case ReportOutputFormatMarkdown:
		return bundle.validateMarkdownFiles()
	case ReportOutputFormatPDF:
		return bundle.validatePDFFiles()
	default:
		return fmt.Errorf("report output bundle format: unsupported report output format %q", bundle.OutputFormat)
	}
}

// validateMarkdownFiles verifies the two-file Markdown main-plus-annex bundle.
// Authored by: OpenCode
func (bundle ReportOutputBundle) validateMarkdownFiles() error {
	if len(bundle.Files) != 2 {
		return fmt.Errorf("markdown report output bundle requires exactly two files")
	}
	if bundle.Files[0].Role != ReportDocumentRoleMain {
		return fmt.Errorf("markdown report output bundle file 0 must be main")
	}
	if bundle.Files[1].Role != ReportDocumentRoleAnnex {
		return fmt.Errorf("markdown report output bundle file 1 must be annex")
	}
	for index, file := range bundle.Files {
		if file.MediaType != ReportMediaTypeMarkdown {
			return fmt.Errorf("markdown report output bundle file %d must use media type %q", index, ReportMediaTypeMarkdown)
		}
	}

	return nil
}

// validatePDFFiles verifies the one-file combined PDF bundle.
// Authored by: OpenCode
func (bundle ReportOutputBundle) validatePDFFiles() error {
	if len(bundle.Files) != 1 {
		return fmt.Errorf("pdf report output bundle requires exactly one file")
	}
	if bundle.Files[0].Role != ReportDocumentRoleCombined {
		return fmt.Errorf("pdf report output bundle file must be combined")
	}
	if bundle.Files[0].MediaType != ReportMediaTypePDF {
		return fmt.Errorf("pdf report output bundle file must use media type %q", ReportMediaTypePDF)
	}

	return nil
}
