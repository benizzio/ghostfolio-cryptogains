// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"strings"
)

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

// PerAssetAuditSection stores one asset subsection in Annex 1's Detailed
// Per-Asset Audit Report.
// Authored by: OpenCode
type PerAssetAuditSection struct {
	AssetIdentityKey string
	DisplayLabel     string
	Entries          []AuditActivityEntry
}

// NewPerAssetAuditSection creates one validated Annex 1 per-asset audit
// section.
//
// Example:
//
//	section, err := model.NewPerAssetAuditSection("asset-btc", "BTC", entries)
//	if err != nil {
//		panic(err)
//	}
//	_ = section.DisplayLabel
//
// Authored by: OpenCode
func NewPerAssetAuditSection(assetIdentityKey string, displayLabel string, entries []AuditActivityEntry) (PerAssetAuditSection, error) {
	var section = PerAssetAuditSection{
		AssetIdentityKey: strings.TrimSpace(assetIdentityKey),
		DisplayLabel:     strings.TrimSpace(displayLabel),
		Entries:          cloneAuditActivityEntries(entries),
	}
	if err := section.Validate(); err != nil {
		return PerAssetAuditSection{}, err
	}

	return section, nil
}

// Validate verifies one Annex 1 per-asset section and every entry in report
// replay order. For example, call `err := section.Validate()` before assigning
// section to AuditAnnex.PerAssetAuditSections.
// Authored by: OpenCode
func (section PerAssetAuditSection) Validate() error {
	if strings.TrimSpace(section.AssetIdentityKey) == "" {
		return fmt.Errorf("per-asset audit section asset identity key is required")
	}
	if strings.TrimSpace(section.DisplayLabel) == "" {
		return fmt.Errorf("per-asset audit section display label is required")
	}

	for index, entry := range section.Entries {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("per-asset audit section entry %d: %w", index, err)
		}
	}

	return nil
}

// AuditAnnex stores the Annex 1 shell and audit evidence attached to every
// calculated report.
// Authored by: OpenCode
type AuditAnnex struct {
	Title                  string
	SectionOrder           []AuditAnnexSection
	PerAssetAuditSections  []PerAssetAuditSection
	ConversionAuditEntries []ConversionAuditEntry
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
// calculated report. For example, use `annex := DefaultAuditAnnex()` before
// attaching calculated audit evidence.
// Authored by: OpenCode
func DefaultAuditAnnex() AuditAnnex {
	return AuditAnnex{
		Title:        AuditAnnexTitle(),
		SectionOrder: RequiredAuditAnnexSectionOrder(),
	}
}

// AuditAnnexTitle returns the required Annex 1 title. For example, use
// `title := AuditAnnexTitle()` when constructing a validated AuditAnnex.
// Authored by: OpenCode
func AuditAnnexTitle() string {
	return auditAnnexTitle
}

// RequiredAuditAnnexSectionOrder returns an independent copy of the required
// top-level Annex 1 section order. For example, use
// `order := RequiredAuditAnnexSectionOrder()` when constructing an AuditAnnex.
// Authored by: OpenCode
func RequiredAuditAnnexSectionOrder() []AuditAnnexSection {
	return []AuditAnnexSection{
		AuditAnnexSectionPerAssetReport,
		AuditAnnexSectionCurrencyConversionAudit,
	}
}

// Validate verifies the required Annex 1 title, section order, and nested audit
// evidence. For example, call `err := annex.Validate()` before rendering the
// annex in either output format.
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
	for index, section := range annex.PerAssetAuditSections {
		if err := section.Validate(); err != nil {
			return fmt.Errorf("audit annex per-asset section %d: %w", index, err)
		}
	}
	for index, entry := range annex.ConversionAuditEntries {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("audit annex conversion audit entry %d: %w", index, err)
		}
	}

	return nil
}

// NewDetailedAuditAnnex creates one validated Annex 1 model with detailed audit
// evidence.
//
// Example:
//
//	annex, err := model.NewDetailedAuditAnnex(sections, conversions)
//	if err != nil {
//		panic(err)
//	}
//	_ = annex.PerAssetAuditSections
//
// Authored by: OpenCode
func NewDetailedAuditAnnex(sections []PerAssetAuditSection, conversions []ConversionAuditEntry) (AuditAnnex, error) {
	var annex = DefaultAuditAnnex()
	annex.PerAssetAuditSections = clonePerAssetAuditSections(sections)
	annex.ConversionAuditEntries = cloneConversionAuditEntries(conversions)
	if err := annex.Validate(); err != nil {
		return AuditAnnex{}, err
	}

	return annex, nil
}

// clonePerAssetAuditSections returns a defensive copy of per-asset audit
// sections.
// Authored by: OpenCode
func clonePerAssetAuditSections(sections []PerAssetAuditSection) []PerAssetAuditSection {
	var cloned = make([]PerAssetAuditSection, 0, len(sections))
	for _, section := range sections {
		var sectionCopy = section
		sectionCopy.Entries = cloneAuditActivityEntries(section.Entries)
		cloned = append(cloned, sectionCopy)
	}

	return cloned
}
