// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
)

// cloneAssetActivityRows returns a defensive copy of one activity-row slice.
// Authored by: OpenCode
func cloneAssetActivityRows(rows []AssetActivityRow) []AssetActivityRow {
	var cloned = make([]AssetActivityRow, 0, len(rows))
	for _, row := range rows {
		var rowCopy = row
		rowCopy.UnitPrice = decimalsupport.ClonePointer(row.UnitPrice)
		rowCopy.GrossValue = decimalsupport.ClonePointer(row.GrossValue)
		rowCopy.FeeAmount = decimalsupport.ClonePointer(row.FeeAmount)
		if row.LiquidationCalculation != nil {
			var liquidationCopy = cloneLiquidationCalculation(*row.LiquidationCalculation)
			rowCopy.LiquidationCalculation = &liquidationCopy
		}
		cloned = append(cloned, rowCopy)
	}

	return cloned
}

// cloneAssetDetailSections returns a defensive copy of one detail-section slice.
// Authored by: OpenCode
func cloneAssetDetailSections(sections []AssetDetailSection) []AssetDetailSection {
	var cloned = make([]AssetDetailSection, 0, len(sections))
	for _, section := range sections {
		var sectionCopy = section
		sectionCopy.ActivityRows = cloneAssetActivityRows(section.ActivityRows)
		sectionCopy.LiquidationSummaries = cloneLiquidationCalculations(section.LiquidationSummaries)
		cloned = append(cloned, sectionCopy)
	}

	return cloned
}

// cloneLiquidationCalculations returns a defensive copy of one liquidation slice.
// Authored by: OpenCode
func cloneLiquidationCalculations(calculations []LiquidationCalculation) []LiquidationCalculation {
	var cloned = make([]LiquidationCalculation, 0, len(calculations))
	for _, calculation := range calculations {
		cloned = append(cloned, cloneLiquidationCalculation(calculation))
	}

	return cloned
}

// cloneLiquidationCalculation returns a defensive copy of one liquidation.
// Authored by: OpenCode
func cloneLiquidationCalculation(calculation LiquidationCalculation) LiquidationCalculation {
	var calculationCopy = calculation
	calculationCopy.Matches = cloneBasisMatches(calculation.Matches)
	return calculationCopy
}

// cloneBasisMatches returns a defensive copy of one basis-match slice.
// Authored by: OpenCode
func cloneBasisMatches(matches []BasisMatch) []BasisMatch {
	var cloned = make([]BasisMatch, 0, len(matches))
	for _, match := range matches {
		var matchCopy = match
		matchCopy.MatchedProceeds = decimalsupport.ClonePointer(match.MatchedProceeds)
		matchCopy.MatchedGainOrLoss = decimalsupport.ClonePointer(match.MatchedGainOrLoss)
		cloned = append(cloned, matchCopy)
	}

	return cloned
}

// cloneAuditAnnex returns a defensive copy of the audit annex.
// Authored by: OpenCode
func cloneAuditAnnex(annex AuditAnnex) AuditAnnex {
	var cloned = annex
	cloned.SectionOrder = append([]AuditAnnexSection(nil), annex.SectionOrder...)
	cloned.PerAssetAuditSections = clonePerAssetAuditSections(annex.PerAssetAuditSections)
	cloned.ConversionAuditEntries = cloneConversionAuditEntries(annex.ConversionAuditEntries)
	return cloned
}
