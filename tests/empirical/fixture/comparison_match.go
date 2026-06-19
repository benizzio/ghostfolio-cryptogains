package fixture

import (
	"fmt"
	"sort"
)

// matchDecimalComparison stores one match-row decimal field comparison.
// Authored by: OpenCode
type matchDecimalComparison struct {
	field         string
	expectedValue string
	actualValue   string
}

// canonicalComparableMatches canonicalizes oracle and project match evidence and
// verifies both sides contain the same number of comparable rows.
// Authored by: OpenCode
func canonicalComparableMatches(
	project ProjectCalculationOutput,
	oracle OracleOutput,
) ([]OracleMatchEvidence, []ProjectMatchEvidence, error) {
	var oracleMatches, err = canonicalOracleMatches(oracle.Matches)
	if err != nil {
		return nil, nil, fmt.Errorf("compare project output %s: canonicalize oracle matches: %w", oracle.CaseID, err)
	}
	var projectMatches []ProjectMatchEvidence
	projectMatches, err = canonicalProjectMatches(project.Matches)
	if err != nil {
		return nil, nil, fmt.Errorf("compare project output %s: canonicalize project matches: %w", oracle.CaseID, err)
	}

	if len(projectMatches) != len(oracleMatches) {
		return nil, nil, fmt.Errorf(
			"compare project output %s %s: match evidence count mismatch: expected %d got %d",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			len(oracleMatches),
			len(projectMatches),
		)
	}

	return oracleMatches, projectMatches, nil
}

// compareMatchEvidence compares every canonical match evidence row.
// Authored by: OpenCode
func compareMatchEvidence(
	oracle OracleOutput,
	oracleMatches []OracleMatchEvidence,
	projectMatches []ProjectMatchEvidence,
) ([]EmpiricalComparisonResult, error) {
	var results = make([]EmpiricalComparisonResult, 0, len(oracleMatches)*4)
	var matchIndex int

	for matchIndex = range oracleMatches {
		var rowResults, err = compareMatchEvidenceRow(
			matchIndex,
			oracle,
			oracleMatches[matchIndex],
			projectMatches[matchIndex],
		)
		if err != nil {
			return nil, err
		}

		results = append(results, rowResults...)
	}

	return results, nil
}

// compareMatchEvidenceRow compares one canonical match evidence row.
// Authored by: OpenCode
func compareMatchEvidenceRow(
	matchIndex int,
	oracle OracleOutput,
	expected OracleMatchEvidence,
	actual ProjectMatchEvidence,
) ([]EmpiricalComparisonResult, error) {
	var err = compareMatchMetadata(matchIndex, oracle, expected, actual)
	if err != nil {
		return nil, err
	}

	var relevantIDs = comparisonRelevantSourceIDs(
		expected.DisposedSourceID,
		expected.AcquisitionSourceID,
	)
	var results = make([]EmpiricalComparisonResult, 0, 4)

	var requiredComparisons = []matchDecimalComparison{
		{
			field:         fmt.Sprintf("matches[%d].matched_quantity", matchIndex),
			expectedValue: expected.MatchedQuantity,
			actualValue:   actual.MatchedQuantity,
		},
		{
			field:         fmt.Sprintf("matches[%d].matched_basis", matchIndex),
			expectedValue: expected.MatchedBasis,
			actualValue:   actual.MatchedBasis,
		},
	}
	var comparisonIndex int
	for comparisonIndex = range requiredComparisons {
		var result, err = compareDecimalField(
			oracle,
			requiredComparisons[comparisonIndex].field,
			requiredComparisons[comparisonIndex].expectedValue,
			requiredComparisons[comparisonIndex].actualValue,
			exactComparisonTolerance,
			relevantIDs,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	var optionalComparisons = []matchDecimalComparison{
		{
			field:         fmt.Sprintf("matches[%d].matched_proceeds", matchIndex),
			expectedValue: expected.MatchedProceeds,
			actualValue:   actual.MatchedProceeds,
		},
		{
			field:         fmt.Sprintf("matches[%d].matched_gain_or_loss", matchIndex),
			expectedValue: expected.MatchedGainOrLoss,
			actualValue:   actual.MatchedGainOrLoss,
		},
	}
	for comparisonIndex = range optionalComparisons {
		var result, err = compareOptionalMatchDecimalField(
			oracle,
			optionalComparisons[comparisonIndex].field,
			optionalComparisons[comparisonIndex].expectedValue,
			optionalComparisons[comparisonIndex].actualValue,
			relevantIDs,
		)
		if err != nil {
			return nil, err
		}
		if result.Field == "" {
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// compareMatchMetadata verifies exact comparable match metadata fields.
// Authored by: OpenCode
func compareMatchMetadata(
	index int,
	oracle OracleOutput,
	expected OracleMatchEvidence,
	actual ProjectMatchEvidence,
) error {
	if expected.DisposedSourceID != actual.DisposedSourceID {
		return fmt.Errorf(
			"compare project output %s %s: matches[%d].disposed_source_id mismatch: expected %s got %s",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			index,
			expected.DisposedSourceID,
			actual.DisposedSourceID,
		)
	}
	if expected.AcquisitionSourceID != actual.AcquisitionSourceID {
		return fmt.Errorf(
			"compare project output %s %s: matches[%d].acquisition_source_id mismatch: expected %s got %s",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			index,
			expected.AcquisitionSourceID,
			actual.AcquisitionSourceID,
		)
	}
	if expected.ScopeID != actual.ScopeID {
		return fmt.Errorf(
			"compare project output %s %s: matches[%d].scope_id mismatch: expected %s got %s",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			index,
			expected.ScopeID,
			actual.ScopeID,
		)
	}
	if expected.SupportLabel != actual.SupportLabel {
		return fmt.Errorf(
			"compare project output %s %s: matches[%d].support_label mismatch: expected %s got %s",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			index,
			expected.SupportLabel,
			actual.SupportLabel,
		)
	}
	if expected.CompositionRuleID != actual.CompositionRuleID {
		return fmt.Errorf(
			"compare project output %s %s: matches[%d].composition_rule_id mismatch: expected %s got %s",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			index,
			expected.CompositionRuleID,
			actual.CompositionRuleID,
		)
	}

	return nil
}

// canonicalProjectMatches canonicalizes and sorts project evidence rows.
// Authored by: OpenCode
func canonicalProjectMatches(matches []ProjectMatchEvidence) ([]ProjectMatchEvidence, error) {
	var canonical = make([]ProjectMatchEvidence, len(matches))
	copy(canonical, matches)

	var index int
	for index = range canonical {
		var err error
		canonical[index].MatchedQuantity, err = canonicalRequiredPersistedDecimal(canonical[index].MatchedQuantity)
		if err != nil {
			return nil, fmt.Errorf("canonicalize project match %d matched_quantity: %w", index, err)
		}
		canonical[index].MatchedBasis, err = canonicalRequiredPersistedDecimal(canonical[index].MatchedBasis)
		if err != nil {
			return nil, fmt.Errorf("canonicalize project match %d matched_basis: %w", index, err)
		}
		canonical[index].MatchedProceeds, err = canonicalOptionalPersistedDecimal(canonical[index].MatchedProceeds)
		if err != nil {
			return nil, fmt.Errorf("canonicalize project match %d matched_proceeds: %w", index, err)
		}
		canonical[index].MatchedGainOrLoss, err = canonicalOptionalPersistedDecimal(canonical[index].MatchedGainOrLoss)
		if err != nil {
			return nil, fmt.Errorf("canonicalize project match %d matched_gain_or_loss: %w", index, err)
		}
	}

	sort.Slice(canonical, func(left int, right int) bool {
		return projectMatchSortKey(canonical[left]) < projectMatchSortKey(canonical[right])
	})

	return canonical, nil
}

// comparisonRelevantSourceIDs returns the stable set of non-empty source IDs for
// one evidence comparison row.
// Authored by: OpenCode
func comparisonRelevantSourceIDs(values ...string) []string {
	return stableComparisonSourceIDs(values)
}
