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
//
// Use this helper when a caller has already run project calculation for a wider
// dataset slice but needs comparison values constrained to one empirical case.
// The helper filters liquidation evidence to `empiricalCase.ActivitySourceIDs`,
// preserves the report method and year, and derives closing quantity and basis
// from the last relevant activity row for the selected asset. If no relevant
// rows exist for the case and asset, it returns deterministic zero comparable
// values so fixture-backed tests can compare reference-only or out-of-scope
// assets without special-case branching.
//
// Example:
//
//	output, err := fixture.NormalizeProjectCalculationOutputForCase(
//		"case-fifo-basic-2024",
//		empiricalCase,
//		report,
//		"asset-alpha",
//	)
//	if err != nil {
//		panic(err)
//	}
//	_ = output.Matches
//
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
	var normalizedCaseID string
	var normalizedAssetIdentityKey string
	var err error
	normalizedCaseID, normalizedAssetIdentityKey, err = normalizeProjectOutputInputs(caseID, assetIdentityKey)
	if err != nil {
		return ProjectCalculationOutput{}, err
	}

	var detailSection reportmodel.AssetDetailSection
	var hasDetailSection bool
	detailSection, hasDetailSection, err = findValidatedProjectDetailSection(report, normalizedCaseID, normalizedAssetIdentityKey)
	if err != nil {
		return ProjectCalculationOutput{}, err
	}

	var output = newProjectCalculationOutput(normalizedCaseID, report, normalizedAssetIdentityKey)

	if !hasDetailSection {
		output.Values = zeroProjectComparableOutputValues()
		return output, nil
	}

	var relevantRows = projectRelevantActivityRows(empiricalCase, detailSection.ActivityRows)
	if empiricalCase != nil && len(relevantRows) == 0 {
		output.Values = zeroProjectComparableOutputValues()
		return output, nil
	}

	var liquidations = projectLiquidationsForOutput(empiricalCase, detailSection, relevantRows)
	var realizedGainOrLoss apd.Decimal
	var allocatedBasis apd.Decimal
	realizedGainOrLoss, allocatedBasis, output.Matches, err = aggregateProjectLiquidations(
		normalizedCaseID,
		normalizedAssetIdentityKey,
		liquidations,
	)
	if err != nil {
		return ProjectCalculationOutput{}, err
	}

	var closingQuantity apd.Decimal
	var closingBasis apd.Decimal
	closingQuantity, closingBasis = projectClosingValues(empiricalCase, detailSection, relevantRows)
	output.Values, err = canonicalProjectComparableOutputValues(realizedGainOrLoss, allocatedBasis, closingQuantity, closingBasis)
	if err != nil {
		return ProjectCalculationOutput{}, err
	}
	output.Matches = normalizeProjectMatchesForMethod(report.CostBasisMethod, output.Matches)

	return output, nil
}

// normalizeProjectOutputInputs trims and validates the output lookup identity.
// Authored by: OpenCode
func normalizeProjectOutputInputs(caseID string, assetIdentityKey string) (string, string, error) {
	var normalizedCaseID = strings.TrimSpace(caseID)
	if normalizedCaseID == "" {
		return "", "", fmt.Errorf("project output case_id is required")
	}

	var normalizedAssetIdentityKey = strings.TrimSpace(assetIdentityKey)
	if normalizedAssetIdentityKey == "" {
		return "", "", fmt.Errorf("project output asset identity key is required")
	}

	return normalizedCaseID, normalizedAssetIdentityKey, nil
}

// findValidatedProjectDetailSection returns a matching detail section after
// validating the detail/reference consistency rules.
// Authored by: OpenCode
func findValidatedProjectDetailSection(
	report reportmodel.CapitalGainsReport,
	normalizedCaseID string,
	normalizedAssetIdentityKey string,
) (reportmodel.AssetDetailSection, bool, error) {
	var detailSection, hasDetailSection = findProjectDetailSection(report, normalizedAssetIdentityKey)
	var referenceEntry, hasReferenceEntry = findProjectReferenceEntry(report, normalizedAssetIdentityKey)

	if !hasDetailSection && !hasReferenceEntry {
		return reportmodel.AssetDetailSection{}, false, fmt.Errorf(
			"normalize project output %s %s: asset not found in report detail sections or reference entries",
			normalizedCaseID,
			normalizedAssetIdentityKey,
		)
	}
	if !hasDetailSection && hasReferenceEntry && referenceEntry.MainSectionStatus != reportmodel.ReferenceSectionStatusReferenceOnly {
		return reportmodel.AssetDetailSection{}, false, fmt.Errorf(
			"normalize project output %s %s: reference entry without detail section must be reference only",
			normalizedCaseID,
			normalizedAssetIdentityKey,
		)
	}

	return detailSection, hasDetailSection, nil
}

// newProjectCalculationOutput creates the stable output shell shared by all
// normalization paths.
// Authored by: OpenCode
func newProjectCalculationOutput(
	normalizedCaseID string,
	report reportmodel.CapitalGainsReport,
	normalizedAssetIdentityKey string,
) ProjectCalculationOutput {
	return ProjectCalculationOutput{
		CaseID:           normalizedCaseID,
		Method:           report.CostBasisMethod,
		Year:             report.Year,
		AssetIdentityKey: normalizedAssetIdentityKey,
		Matches:          make([]ProjectMatchEvidence, 0),
	}
}

// zeroProjectComparableOutputValues returns the deterministic zero value set for
// reference-only or case-filtered empty results.
// Authored by: OpenCode
func zeroProjectComparableOutputValues() ComparableOutputValues {
	return ComparableOutputValues{
		RealizedGainOrLoss: "0",
		AllocatedBasis:     "0",
		ClosingQuantity:    "0",
		ClosingBasis:       "0",
	}
}

// projectRelevantActivityRows returns report activity rows constrained to the
// selected empirical case when one is provided.
// Authored by: OpenCode
func projectRelevantActivityRows(
	empiricalCase *EmpiricalCase,
	rows []reportmodel.AssetActivityRow,
) []reportmodel.AssetActivityRow {
	if empiricalCase == nil {
		return filterRelevantActivityRows(rows, nil)
	}

	var caseSourceIDs = empiricalCaseSourceIDSet(*empiricalCase)
	return filterRelevantActivityRows(rows, caseSourceIDs)
}

// projectLiquidationsForOutput selects aggregate liquidation summaries or
// case-filtered row liquidations for comparison normalization.
// Authored by: OpenCode
func projectLiquidationsForOutput(
	empiricalCase *EmpiricalCase,
	detailSection reportmodel.AssetDetailSection,
	relevantRows []reportmodel.AssetActivityRow,
) []reportmodel.LiquidationCalculation {
	if empiricalCase == nil {
		return append([]reportmodel.LiquidationCalculation(nil), detailSection.LiquidationSummaries...)
	}

	var liquidations = make([]reportmodel.LiquidationCalculation, 0, len(relevantRows))
	var rowIndex int
	for rowIndex = range relevantRows {
		if relevantRows[rowIndex].LiquidationCalculation == nil {
			continue
		}
		liquidations = append(liquidations, *relevantRows[rowIndex].LiquidationCalculation)
	}

	return liquidations
}

// aggregateProjectLiquidations sums liquidation values and normalizes match
// evidence for comparison.
// Authored by: OpenCode
func aggregateProjectLiquidations(
	normalizedCaseID string,
	normalizedAssetIdentityKey string,
	liquidations []reportmodel.LiquidationCalculation,
) (apd.Decimal, apd.Decimal, []ProjectMatchEvidence, error) {
	var realizedGainOrLoss = apd.Decimal{}
	var allocatedBasis = apd.Decimal{}
	var matches = make([]ProjectMatchEvidence, 0)
	var liquidationIndex int

	for liquidationIndex = range liquidations {
		var liquidation = liquidations[liquidationIndex]
		var err error

		realizedGainOrLoss, err = supportmath.Add(realizedGainOrLoss, liquidation.GainOrLoss)
		if err != nil {
			return apd.Decimal{}, apd.Decimal{}, nil, fmt.Errorf(
				"normalize project output %s %s realized gain or loss: %w",
				normalizedCaseID,
				normalizedAssetIdentityKey,
				err,
			)
		}

		allocatedBasis, err = supportmath.Add(allocatedBasis, liquidation.AllocatedBasis)
		if err != nil {
			return apd.Decimal{}, apd.Decimal{}, nil, fmt.Errorf(
				"normalize project output %s %s allocated basis: %w",
				normalizedCaseID,
				normalizedAssetIdentityKey,
				err,
			)
		}

		var normalizedMatches []ProjectMatchEvidence
		normalizedMatches, err = normalizeProjectLiquidationMatches(liquidation)
		if err != nil {
			return apd.Decimal{}, apd.Decimal{}, nil, fmt.Errorf(
				"normalize project output %s %s liquidation %s matches: %w",
				normalizedCaseID,
				normalizedAssetIdentityKey,
				strings.TrimSpace(liquidation.SourceID),
				err,
			)
		}

		matches = append(matches, normalizedMatches...)
	}

	return realizedGainOrLoss, allocatedBasis, matches, nil
}

// projectClosingValues chooses the report-level or case-filtered closing values.
// Authored by: OpenCode
func projectClosingValues(
	empiricalCase *EmpiricalCase,
	detailSection reportmodel.AssetDetailSection,
	relevantRows []reportmodel.AssetActivityRow,
) (apd.Decimal, apd.Decimal) {
	if empiricalCase != nil && len(relevantRows) != 0 {
		var lastRelevantRow = relevantRows[len(relevantRows)-1]
		return lastRelevantRow.QuantityAfterRow, lastRelevantRow.BasisAfterRow
	}

	return detailSection.ClosingQuantity, detailSection.ClosingCostBasis
}

// canonicalProjectComparableOutputValues canonicalizes all comparable decimal
// fields for persisted fixture comparison.
// Authored by: OpenCode
func canonicalProjectComparableOutputValues(
	realizedGainOrLoss apd.Decimal,
	allocatedBasis apd.Decimal,
	closingQuantity apd.Decimal,
	closingBasis apd.Decimal,
) (ComparableOutputValues, error) {
	var values ComparableOutputValues
	var err error
	values.RealizedGainOrLoss, err = CanonicalDecimalString(realizedGainOrLoss)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("normalize project output realized_gain_or_loss: %w", err)
	}
	values.AllocatedBasis, err = CanonicalDecimalString(allocatedBasis)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("normalize project output allocated_basis: %w", err)
	}
	values.ClosingQuantity, err = CanonicalDecimalString(closingQuantity)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("normalize project output closing_quantity: %w", err)
	}
	values.ClosingBasis, err = CanonicalDecimalString(closingBasis)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("normalize project output closing_basis: %w", err)
	}

	return values, nil
}

// normalizeProjectMatchesForMethod sorts match evidence and suppresses matches
// for Average Cost, which has no per-acquisition match comparison contract.
// Authored by: OpenCode
func normalizeProjectMatchesForMethod(
	method reportmodel.CostBasisMethod,
	matches []ProjectMatchEvidence,
) []ProjectMatchEvidence {
	sort.Slice(matches, func(left int, right int) bool {
		return projectMatchSortKey(matches[left]) < projectMatchSortKey(matches[right])
	})
	if method == reportmodel.CostBasisMethodAverageCost {
		return nil
	}

	return matches
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
			SupportLabel:        EvidenceSupportLabelRotkiBacked,
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
