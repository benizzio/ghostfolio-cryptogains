// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import "fmt"

const auditAnnexTitle = "Annex 1 - Audit"

// AuditAnnexSection identifies one top-level Annex 1 section in its required
// rendering order.
// Authored by: OpenCode
type AuditAnnexSection string

const (
	// AuditAnnexSectionPerAssetReport identifies the detailed per-asset audit
	// report section.
	AuditAnnexSectionPerAssetReport AuditAnnexSection = "detailed_per_asset_audit_report"

	// AuditAnnexSectionCurrencyConversionAudit identifies the Currency Conversion
	// Audit section.
	AuditAnnexSectionCurrencyConversionAudit AuditAnnexSection = "currency_conversion_audit"
)

// AuditAnnex stores the minimal Annex 1 shell attached to every calculated
// report. Later audit-evidence tasks extend this model with section content.
// Authored by: OpenCode
type AuditAnnex struct {
	Title        string
	SectionOrder []AuditAnnexSection
}

// NewAuditAnnex creates one validated Annex 1 shell.
//
// Example:
//
//	annex, err := model.NewAuditAnnex(model.AuditAnnexTitle(), model.RequiredAuditAnnexSectionOrder())
//	if err != nil {
//		panic(err)
//	}
//	_ = annex.Title
//
// Authored by: OpenCode
func NewAuditAnnex(title string, sectionOrder []AuditAnnexSection) (AuditAnnex, error) {
	var annex = AuditAnnex{
		Title:        title,
		SectionOrder: append([]AuditAnnexSection(nil), sectionOrder...),
	}

	if err := annex.Validate(); err != nil {
		return AuditAnnex{}, err
	}

	return annex, nil
}

// DefaultAuditAnnex creates the required empty Annex 1 shell for a newly
// calculated report.
// Authored by: OpenCode
func DefaultAuditAnnex() AuditAnnex {
	return AuditAnnex{
		Title:        AuditAnnexTitle(),
		SectionOrder: RequiredAuditAnnexSectionOrder(),
	}
}

// AuditAnnexTitle returns the required Annex 1 title.
// Authored by: OpenCode
func AuditAnnexTitle() string {
	return auditAnnexTitle
}

// RequiredAuditAnnexSectionOrder returns the required top-level Annex 1 section
// order.
// Authored by: OpenCode
func RequiredAuditAnnexSectionOrder() []AuditAnnexSection {
	return []AuditAnnexSection{
		AuditAnnexSectionPerAssetReport,
		AuditAnnexSectionCurrencyConversionAudit,
	}
}

// Validate verifies the required Annex 1 title and top-level section order.
// Authored by: OpenCode
func (annex AuditAnnex) Validate() error {
	if annex.Title != auditAnnexTitle {
		return fmt.Errorf("audit annex title must be %q", auditAnnexTitle)
	}

	var requiredOrder = RequiredAuditAnnexSectionOrder()
	if len(annex.SectionOrder) != len(requiredOrder) {
		return fmt.Errorf("audit annex section order must contain %d sections", len(requiredOrder))
	}
	for index, section := range requiredOrder {
		if annex.SectionOrder[index] != section {
			return fmt.Errorf("audit annex section order %d must be %q", index, section)
		}
	}

	return nil
}
