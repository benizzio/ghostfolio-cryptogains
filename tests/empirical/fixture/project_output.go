package fixture

import (
	"fmt"
	"sort"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// NormalizeProjectCalculationOutput converts one calculated capital-gains report
// segment into the stable empirical comparison shape used by oracle-backed
// fixture tests.
//
// Example:
//
//	output, err := fixture.NormalizeProjectCalculationOutput("case-fifo-2024", report, "asset-alpha")
//	if err != nil {
//		panic(err)
//	}
//	_ = output.Values
//
// The function uses only calculated report data. If the asset is present only in
// the report reference section, it emits deterministic zero comparable values and
// no match evidence for the selected year.
// Authored by: OpenCode
func NormalizeProjectCalculationOutput(
	caseID string,
	report reportmodel.CapitalGainsReport,
	assetIdentityKey string,
) (ProjectCalculationOutput, error) {
	return normalizeProjectCalculationOutput(caseID, nil, report, assetIdentityKey)
}

// NormalizeProjectCalculationOutputForCase converts one calculated capital-gains
// report segment into the stable empirical comparison shape for one specific
// empirical case source-id slice.
// Authored by: OpenCode
func NormalizeProjectCalculationOutputForCase(
	caseID string,
	empiricalCase EmpiricalCase,
	report reportmodel.CapitalGainsReport,
	assetIdentityKey string,
) (ProjectCalculationOutput, error) {
	return normalizeProjectCalculationOutput(caseID, &empiricalCase, report, assetIdentityKey)
}

// normalizeProjectCalculationOutput implements both the asset-wide and
// case-filtered normalization paths.
// Authored by: OpenCode
func normalizeProjectCalculationOutput(
	caseID string,
	empiricalCase *EmpiricalCase,
	report reportmodel.CapitalGainsReport,
	assetIdentityKey string,
) (ProjectCalculationOutput, error) {
	var normalizedCaseID = strings.TrimSpace(caseID)
	if normalizedCaseID == "" {
		return ProjectCalculationOutput{}, fmt.Errorf("project output case_id is required")
	}

	var normalizedAssetIdentityKey = strings.TrimSpace(assetIdentityKey)
	if normalizedAssetIdentityKey == "" {
		return ProjectCalculationOutput{}, fmt.Errorf("project output asset identity key is required")
	}

	var detailSection, hasDetailSection = findProjectDetailSection(report, normalizedAssetIdentityKey)
	var referenceEntry, hasReferenceEntry = findProjectReferenceEntry(report, normalizedAssetIdentityKey)

	if !hasDetailSection && !hasReferenceEntry {
		return ProjectCalculationOutput{}, fmt.Errorf(
			"normalize project output %s %s: asset not found in report detail sections or reference entries",
			normalizedCaseID,
			normalizedAssetIdentityKey,
		)
	}
	if !hasDetailSection && hasReferenceEntry && referenceEntry.MainSectionStatus != reportmodel.ReferenceSectionStatusReferenceOnly {
		return ProjectCalculationOutput{}, fmt.Errorf(
			"normalize project output %s %s: reference entry without detail section must be reference only",
			normalizedCaseID,
			normalizedAssetIdentityKey,
		)
	}

	var output = ProjectCalculationOutput{
		CaseID:           normalizedCaseID,
		Method:           report.CostBasisMethod,
		Year:             report.Year,
		AssetIdentityKey: normalizedAssetIdentityKey,
		Matches:          make([]ProjectMatchEvidence, 0),
	}

	if !hasDetailSection {
		output.Values = ComparableOutputValues{
			RealizedGainOrLoss: "0",
			AllocatedBasis:     "0",
			ClosingQuantity:    "0",
			ClosingBasis:       "0",
		}
		return output, nil
	}

	var caseSourceIDs map[string]struct{}
	if empiricalCase != nil {
		caseSourceIDs = empiricalCaseSourceIDSet(*empiricalCase)
	}

	var relevantRows = filterRelevantActivityRows(detailSection.ActivityRows, caseSourceIDs)
	if empiricalCase != nil && len(relevantRows) == 0 {
		output.Values = ComparableOutputValues{
			RealizedGainOrLoss: "0",
			AllocatedBasis:     "0",
			ClosingQuantity:    "0",
			ClosingBasis:       "0",
		}
		return output, nil
	}

	var realizedGainOrLoss = apd.Decimal{}
	var allocatedBasis = apd.Decimal{}
	var rowIndex int
	var liquidations []reportmodel.LiquidationCalculation
	if empiricalCase == nil {
		liquidations = append([]reportmodel.LiquidationCalculation(nil), detailSection.LiquidationSummaries...)
	} else {
		liquidations = make([]reportmodel.LiquidationCalculation, 0, len(relevantRows))
		for rowIndex = range relevantRows {
			if relevantRows[rowIndex].LiquidationCalculation == nil {
				continue
			}
			liquidations = append(liquidations, *relevantRows[rowIndex].LiquidationCalculation)
		}
	}

	for rowIndex = range liquidations {
		var liquidation = liquidations[rowIndex]
		var err error

		realizedGainOrLoss, err = supportmath.Add(realizedGainOrLoss, liquidation.GainOrLoss)
		if err != nil {
			return ProjectCalculationOutput{}, fmt.Errorf(
				"normalize project output %s %s realized gain or loss: %w",
				normalizedCaseID,
				normalizedAssetIdentityKey,
				err,
			)
		}

		allocatedBasis, err = supportmath.Add(allocatedBasis, liquidation.AllocatedBasis)
		if err != nil {
			return ProjectCalculationOutput{}, fmt.Errorf(
				"normalize project output %s %s allocated basis: %w",
				normalizedCaseID,
				normalizedAssetIdentityKey,
				err,
			)
		}

		var normalizedMatches []ProjectMatchEvidence
			normalizedMatches, err = normalizeProjectLiquidationMatches(liquidation)
		if err != nil {
			return ProjectCalculationOutput{}, fmt.Errorf(
				"normalize project output %s %s liquidation %s matches: %w",
				normalizedCaseID,
				normalizedAssetIdentityKey,
				strings.TrimSpace(liquidation.SourceID),
				err,
			)
		}

		output.Matches = append(output.Matches, normalizedMatches...)
	}

	var closingQuantity = detailSection.ClosingQuantity
	var closingBasis = detailSection.ClosingCostBasis
	if empiricalCase != nil && len(relevantRows) != 0 {
		closingQuantity = relevantRows[len(relevantRows)-1].QuantityAfterRow
		closingBasis = relevantRows[len(relevantRows)-1].BasisAfterRow
	}

	var err error
	output.Values.RealizedGainOrLoss, err = CanonicalDecimalString(realizedGainOrLoss)
	if err != nil {
		return ProjectCalculationOutput{}, fmt.Errorf("normalize project output realized_gain_or_loss: %w", err)
	}
	output.Values.AllocatedBasis, err = CanonicalDecimalString(allocatedBasis)
	if err != nil {
		return ProjectCalculationOutput{}, fmt.Errorf("normalize project output allocated_basis: %w", err)
	}
	output.Values.ClosingQuantity, err = CanonicalDecimalString(closingQuantity)
	if err != nil {
		return ProjectCalculationOutput{}, fmt.Errorf("normalize project output closing_quantity: %w", err)
	}
	output.Values.ClosingBasis, err = CanonicalDecimalString(closingBasis)
	if err != nil {
		return ProjectCalculationOutput{}, fmt.Errorf("normalize project output closing_basis: %w", err)
	}

	sort.Slice(output.Matches, func(left int, right int) bool {
		return projectMatchSortKey(output.Matches[left]) < projectMatchSortKey(output.Matches[right])
	})

	return output, nil
}

// findProjectDetailSection returns one matching detail section when present.
// Authored by: OpenCode
func findProjectDetailSection(report reportmodel.CapitalGainsReport, assetIdentityKey string) (reportmodel.AssetDetailSection, bool) {
	var index int

	for index = range report.DetailSections {
		if report.DetailSections[index].AssetIdentityKey == assetIdentityKey {
			return report.DetailSections[index], true
		}
	}

	return reportmodel.AssetDetailSection{}, false
}

// findProjectReferenceEntry returns one matching reference entry when present.
// Authored by: OpenCode
func findProjectReferenceEntry(report reportmodel.CapitalGainsReport, assetIdentityKey string) (reportmodel.ReferenceLiquidationEntry, bool) {
	var index int

	for index = range report.ReferenceEntries {
		if report.ReferenceEntries[index].AssetIdentityKey == assetIdentityKey {
			return report.ReferenceEntries[index], true
		}
	}

	return reportmodel.ReferenceLiquidationEntry{}, false
}

// normalizeProjectLiquidationMatches converts one liquidation's basis-match
// evidence into the stable empirical comparison shape.
// Authored by: OpenCode
func normalizeProjectLiquidationMatches(
	liquidation reportmodel.LiquidationCalculation,
) ([]ProjectMatchEvidence, error) {
	var normalized = make([]ProjectMatchEvidence, 0, len(liquidation.Matches))
	var matchIndex int

	for matchIndex = range liquidation.Matches {
		var match = liquidation.Matches[matchIndex]
		var normalizedMatch = ProjectMatchEvidence{
			DisposedSourceID:    strings.TrimSpace(liquidation.SourceID),
			AcquisitionSourceID: strings.TrimSpace(match.AcquisitionSourceID),
			SupportLabel:        EvidenceSupportLabelHledgerBacked,
		}

		var err error
		normalizedMatch.MatchedQuantity, err = CanonicalDecimalString(match.MatchedQuantity)
		if err != nil {
			return nil, fmt.Errorf("match %d matched_quantity: %w", matchIndex, err)
		}
		normalizedMatch.MatchedBasis, err = CanonicalDecimalString(match.MatchedBasis)
		if err != nil {
			return nil, fmt.Errorf("match %d matched_basis: %w", matchIndex, err)
		}
		normalizedMatch.MatchedProceeds, err = canonicalOptionalProjectDecimal(match.MatchedProceeds)
		if err != nil {
			return nil, fmt.Errorf("match %d matched_proceeds: %w", matchIndex, err)
		}
		normalizedMatch.MatchedGainOrLoss, err = canonicalOptionalProjectDecimal(match.MatchedGainOrLoss)
		if err != nil {
			return nil, fmt.Errorf("match %d matched_gain_or_loss: %w", matchIndex, err)
		}

		normalized = append(normalized, normalizedMatch)
	}

	return normalized, nil
}

// canonicalOptionalProjectDecimal converts one optional decimal pointer into the
// canonical persisted comparison text.
// Authored by: OpenCode
func canonicalOptionalProjectDecimal(value *apd.Decimal) (string, error) {
	if value == nil {
		return "", nil
	}

	return CanonicalDecimalString(*value)
}

// projectMatchSortKey returns the stable lexical sort key for one normalized
// project evidence row.
// Authored by: OpenCode
func projectMatchSortKey(match ProjectMatchEvidence) string {
	return strings.Join([]string{
		match.DisposedSourceID,
		match.AcquisitionSourceID,
		match.ScopeID,
		match.MatchedQuantity,
		match.MatchedBasis,
		match.MatchedProceeds,
		match.MatchedGainOrLoss,
		string(match.SupportLabel),
		match.CompositionRuleID,
	}, "\x00")
}

// empiricalCaseSourceIDSet returns the stable case source-id lookup set.
// Authored by: OpenCode
func empiricalCaseSourceIDSet(empiricalCase EmpiricalCase) map[string]struct{} {
	var sourceIDs = make(map[string]struct{}, len(empiricalCase.ActivitySourceIDs))
	var index int
	for index = range empiricalCase.ActivitySourceIDs {
		sourceIDs[strings.TrimSpace(empiricalCase.ActivitySourceIDs[index])] = struct{}{}
	}

	return sourceIDs
}

// filterRelevantActivityRows returns the in-year activity rows that belong to
// the selected empirical case source-id slice.
// Authored by: OpenCode
func filterRelevantActivityRows(rows []reportmodel.AssetActivityRow, sourceIDs map[string]struct{}) []reportmodel.AssetActivityRow {
	if sourceIDs == nil {
		return append([]reportmodel.AssetActivityRow(nil), rows...)
	}

	var filtered = make([]reportmodel.AssetActivityRow, 0, len(rows))
	var index int
	for index = range rows {
		if _, ok := sourceIDs[strings.TrimSpace(rows[index].SourceID)]; !ok {
			continue
		}

		filtered = append(filtered, rows[index])
	}

	return filtered
}
