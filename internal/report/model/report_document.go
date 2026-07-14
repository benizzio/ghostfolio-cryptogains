// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"bytes"
	"fmt"
	"time"
)

// ReportDocument stores the rendered report content before the final save.
// Authored by: OpenCode
type ReportDocument struct {
	DocumentType    ReportDocumentType
	Role            ReportDocumentRole
	Content         []byte
	Year            int
	CostBasisMethod CostBasisMethod
	GeneratedAt     time.Time
}

// NewReportDocument creates one validated rendered report document.
//
// Example:
//
//	document, err := model.NewReportDocument(model.ReportDocumentTypeMarkdown, model.ReportDocumentRoleMain, []byte("# Report\n"), 2024, model.CostBasisMethodFIFO, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = document.Content
//
// Authored by: OpenCode
func NewReportDocument(
	documentType ReportDocumentType,
	role ReportDocumentRole,
	content []byte,
	year int,
	method CostBasisMethod,
	generatedAt time.Time,
) (ReportDocument, error) {
	var document = ReportDocument{
		DocumentType:    documentType,
		Role:            role,
		Content:         append([]byte(nil), content...),
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
	if err := validateReportDocumentRole(document.Role); err != nil {
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
		if len(bytes.TrimSpace(document.Content)) == 0 {
			return fmt.Errorf("report document content is required")
		}
	case ReportDocumentTypePDF:
		if len(bytes.TrimSpace(document.Content)) == 0 {
			return fmt.Errorf("report document PDF content is required")
		}
	}

	return nil
}

// validateRoleCompatibility verifies document type and role combinations.
// Authored by: OpenCode
func (document ReportDocument) validateRoleCompatibility() error {
	switch document.DocumentType {
	case ReportDocumentTypeMarkdown:
		if document.Role != ReportDocumentRoleMain && document.Role != ReportDocumentRoleAnnex {
			return fmt.Errorf("markdown report document role must be main or annex")
		}
	case ReportDocumentTypePDF:
		if document.Role != ReportDocumentRoleCombined {
			return fmt.Errorf("pdf report document role must be combined")
		}
	}

	return nil
}
