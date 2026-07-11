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

// Validate verifies one successful persisted output bundle shape before runtime
// reports saved paths to the user. For example, call `err := bundle.Validate()`
// after a writer has recorded its files and optional open warning.
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

// ValidateRenderedDocuments validates the complete rendered bundle before any
// filesystem reservation occurs. For example, Markdown requires main then Annex
// documents with shared report metadata, while PDF requires one combined PDF.
// Authored by: OpenCode
func ValidateRenderedDocuments(outputFormat ReportOutputFormat, documents []ReportDocument) error {
	if err := validateReportOutputFormat(outputFormat); err != nil {
		return err
	}
	if err := validateRenderedDocumentContents(documents); err != nil {
		return err
	}
	if err := validateRenderedDocumentShape(outputFormat, documents); err != nil {
		return err
	}
	return validateRenderedDocumentMetadata(documents)
}

// validateRenderedDocumentContents verifies every rendered payload before its
// format-specific bundle shape is evaluated.
// Authored by: OpenCode
func validateRenderedDocumentContents(documents []ReportDocument) error {
	for index, document := range documents {
		if err := document.Validate(); err != nil {
			return fmt.Errorf("report document %d: %w", index, err)
		}
	}
	return nil
}

// validateRenderedDocumentShape verifies the selected output format has its
// required main, annex, or combined document roles.
// Authored by: OpenCode
func validateRenderedDocumentShape(outputFormat ReportOutputFormat, documents []ReportDocument) error {
	switch outputFormat {
	case ReportOutputFormatMarkdown:
		if len(documents) != 2 {
			return fmt.Errorf("markdown report output requires exactly two documents")
		}
		if documents[0].DocumentType != ReportDocumentTypeMarkdown || documents[0].Role != ReportDocumentRoleMain {
			return fmt.Errorf("markdown report output document 0 must be the main Markdown document")
		}
		if documents[1].DocumentType != ReportDocumentTypeMarkdown || documents[1].Role != ReportDocumentRoleAnnex {
			return fmt.Errorf("markdown report output document 1 must be the Annex 1 Markdown document")
		}
	case ReportOutputFormatPDF:
		if len(documents) != 1 {
			return fmt.Errorf("pdf report output requires exactly one document")
		}
		if documents[0].DocumentType != ReportDocumentTypePDF || documents[0].Role != ReportDocumentRoleCombined {
			return fmt.Errorf("pdf report output document must be the combined PDF document")
		}
	}
	return nil
}

// validateRenderedDocumentMetadata verifies documents in one bundle describe
// the same calculated report.
// Authored by: OpenCode
func validateRenderedDocumentMetadata(documents []ReportDocument) error {
	var first = documents[0]
	for index := 1; index < len(documents); index++ {
		if documents[index].Year != first.Year {
			return fmt.Errorf("report document %d year does not match the first document", index)
		}
		if documents[index].CostBasisMethod != first.CostBasisMethod {
			return fmt.Errorf("report document %d cost basis method does not match the first document", index)
		}
		if !documents[index].GeneratedAt.Equal(first.GeneratedAt) {
			return fmt.Errorf("report document %d generated-at timestamp does not match the first document", index)
		}
	}
	return nil
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
