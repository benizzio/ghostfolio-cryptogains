// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
	"time"
)

// NewReportDocument creates one validated rendered report document.
//
// Example:
//
//	document, err := model.NewReportDocument(model.ReportDocumentTypeMarkdown, "# Report\n", 2024, model.CostBasisMethodFIFO, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = document.Content
//
// Authored by: OpenCode
func NewReportDocument(
	documentType ReportDocumentType,
	content string,
	year int,
	method CostBasisMethod,
	generatedAt time.Time,
) (ReportDocument, error) {
	var document = ReportDocument{
		DocumentType:    documentType,
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

// Validate verifies one rendered report document before save.
// Authored by: OpenCode
func (document ReportDocument) Validate() error {
	if err := validateReportDocumentType(document.DocumentType); err != nil {
		return fmt.Errorf("report document type: %w", err)
	}
	if strings.TrimSpace(document.Content) == "" {
		return fmt.Errorf("report document content is required")
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
