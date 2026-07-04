// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// ReportDocument stores the rendered report content before the final save.
// Authored by: OpenCode
type ReportDocument struct {
	DocumentType    ReportDocumentType
	Role            ReportDocumentRole
	Content         string
	PDFContent      []byte
	Year            int
	CostBasisMethod CostBasisMethod
	GeneratedAt     time.Time
}

// NewReportDocument creates one validated rendered report document.
//
// Example:
//
//	document, err := model.NewReportDocument(model.ReportDocumentTypeMarkdown, model.ReportDocumentRoleMain, "# Report\n", 2024, model.CostBasisMethodFIFO, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = document.Content
//
// Authored by: OpenCode
func NewReportDocument(
	documentType ReportDocumentType,
	args ...any,
) (ReportDocument, error) {
	var role, content, year, method, generatedAt, argErr = parseMarkdownReportDocumentArgs(args)
	if argErr != nil {
		return ReportDocument{}, argErr
	}

	var document = ReportDocument{
		DocumentType:    documentType,
		Role:            role,
		Content:         content,
		Year:            year,
		CostBasisMethod: method,
		GeneratedAt:     generatedAt,
	}

	if err := document.Validate(); err != nil {
		return ReportDocument{}, err
	}

	return document, nil
}

// parseMarkdownReportDocumentArgs accepts current role-aware document calls and
// legacy Markdown-only calls that are migrated in later story tasks.
// Authored by: OpenCode
func parseMarkdownReportDocumentArgs(args []any) (ReportDocumentRole, string, int, CostBasisMethod, time.Time, error) {
	if len(args) == 4 {
		var content, contentOK = args[0].(string)
		var year, yearOK = args[1].(int)
		var method, methodOK = args[2].(CostBasisMethod)
		var generatedAt, timeOK = args[3].(time.Time)
		if !contentOK || !yearOK || !methodOK || !timeOK {
			return "", "", 0, "", time.Time{}, fmt.Errorf("report document content, year, method, and generated-at arguments are required")
		}
		return ReportDocumentRoleMain, content, year, method, generatedAt, nil
	}
	if len(args) == 5 {
		var role, roleOK = args[0].(ReportDocumentRole)
		var content, contentOK = args[1].(string)
		var year, yearOK = args[2].(int)
		var method, methodOK = args[3].(CostBasisMethod)
		var generatedAt, timeOK = args[4].(time.Time)
		if !roleOK || !contentOK || !yearOK || !methodOK || !timeOK {
			return "", "", 0, "", time.Time{}, fmt.Errorf("report document role, content, year, method, and generated-at arguments are required")
		}
		return role, content, year, method, generatedAt, nil
	}

	return "", "", 0, "", time.Time{}, fmt.Errorf("report document requires content, year, method, and generated-at arguments")
}

// NewPDFReportDocument creates one validated PDF report document from rendered
// bytes.
//
// Example:
//
//	document, err := model.NewPDFReportDocument(model.ReportDocumentRoleCombined, []byte("%PDF"), 2024, model.CostBasisMethodFIFO, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = document.PDFContent
//
// Authored by: OpenCode
func NewPDFReportDocument(
	role ReportDocumentRole,
	content []byte,
	year int,
	method CostBasisMethod,
	generatedAt time.Time,
) (ReportDocument, error) {
	var document = ReportDocument{
		DocumentType:    ReportDocumentTypePDF,
		Role:            role,
		PDFContent:      append([]byte(nil), content...),
		Year:            year,
		CostBasisMethod: method,
		GeneratedAt:     generatedAt,
	}

	if err := document.Validate(); err != nil {
		return ReportDocument{}, err
	}

	return document, nil
}

// Validate verifies one rendered report document before save.
// Authored by: OpenCode
func (document ReportDocument) Validate() error {
	if err := validateReportDocumentType(document.DocumentType); err != nil {
		return fmt.Errorf("report document type: %w", err)
	}
	var role = effectiveReportDocumentRole(document)
	if err := validateReportDocumentRole(role); err != nil {
		return fmt.Errorf("report document role: %w", err)
	}
	if err := document.validateContent(); err != nil {
		return err
	}
	if err := document.validateRoleCompatibility(); err != nil {
		return err
	}
	if document.Year <= 0 {
		return fmt.Errorf("report document year must be greater than zero")
	}
	if err := validateCostBasisMethod(document.CostBasisMethod); err != nil {
		return fmt.Errorf("report document cost basis method: %w", err)
	}
	if document.GeneratedAt.IsZero() {
		return fmt.Errorf("report document generated-at timestamp is required")
	}

	return nil
}

// validateContent verifies the rendered payload for the selected document type.
// Authored by: OpenCode
func (document ReportDocument) validateContent() error {
	switch document.DocumentType {
	case ReportDocumentTypeMarkdown:
		if strings.TrimSpace(document.Content) == "" {
			return fmt.Errorf("report document content is required")
		}
	case ReportDocumentTypePDF:
		if len(bytes.TrimSpace(document.PDFContent)) == 0 {
			return fmt.Errorf("report document PDF content is required")
		}
	}

	return nil
}

// validateRoleCompatibility verifies document type and role combinations.
// Authored by: OpenCode
func (document ReportDocument) validateRoleCompatibility() error {
	var role = effectiveReportDocumentRole(document)
	switch document.DocumentType {
	case ReportDocumentTypeMarkdown:
		if role != ReportDocumentRoleMain && role != ReportDocumentRoleAnnex {
			return fmt.Errorf("markdown report document role must be main or annex")
		}
	case ReportDocumentTypePDF:
		if role != ReportDocumentRoleCombined {
			return fmt.Errorf("pdf report document role must be combined")
		}
	}

	return nil
}

// effectiveReportDocumentRole returns the implicit main role for legacy
// Markdown document struct literals while keeping PDF role validation explicit.
// Authored by: OpenCode
func effectiveReportDocumentRole(document ReportDocument) ReportDocumentRole {
	if document.Role == "" && document.DocumentType == ReportDocumentTypeMarkdown {
		return ReportDocumentRoleMain
	}
	return document.Role
}
