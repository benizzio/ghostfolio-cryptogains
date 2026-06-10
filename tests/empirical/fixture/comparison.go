package fixture

import (
	"fmt"
	"sort"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

var exactComparisonTolerance = "0"

// EmpiricalComparisonOutcome stores one complete comparison pass between one
// normalized project output segment and one oracle fixture segment.
// Authored by: OpenCode
type EmpiricalComparisonOutcome struct {
	Results []EmpiricalComparisonResult
	Skips   []EmpiricalComparisonSkip
}

// EmpiricalComparisonSkip stores one unsupported external-oracle assertion that
// is recorded as informational skip context rather than compared as project
// output evidence.
// Authored by: OpenCode
type EmpiricalComparisonSkip struct {
	CaseID            string
	Method            reportmodel.CostBasisMethod
	Year              int
	AssetIdentityKey  string
	ComparisonPolicy  ComparisonPolicy
	Reason            string
	RelevantSourceIDs []string
	DiagnosticContext string
}

// CompareProjectCalculationOutput compares one normalized project calculation
// segment against one oracle fixture segment using exact decimal arithmetic and
// the fixture-declared financial tolerances.
//
// Example:
//
//	outcome, err := fixture.CompareProjectCalculationOutput(projectOutput, oracleOutput)
//	if err != nil {
//		panic(err)
//	}
//	_ = outcome.Results
//
// Unsupported oracle segments are returned as informational skips. Recorded
// oracle values and recorded match evidence are still compared directly.
// Authored by: OpenCode
func CompareProjectCalculationOutput(
	project ProjectCalculationOutput,
	oracle OracleOutput,
) (EmpiricalComparisonOutcome, error) {
	var outcome = EmpiricalComparisonOutcome{
		Results: make([]EmpiricalComparisonResult, 0),
		Skips:   buildEmpiricalComparisonSkips(oracle),
	}

	if err := validateComparableSegmentIdentity(project, oracle); err != nil {
		return EmpiricalComparisonOutcome{}, err
	}
	if _, err := parseOracleDecimalPolicy(oracle.Metadata.DecimalPolicy); err != nil {
		return EmpiricalComparisonOutcome{}, fmt.Errorf("compare project output %s: invalid decimal policy: %w", oracle.CaseID, err)
	}

	var valueComparisons = []struct {
		field         string
		expectedValue string
		actualValue   string
		tolerance     string
		relevantIDs   []string
	}{
		{
			field:         "values.realized_gain_or_loss",
			expectedValue: oracle.Values.RealizedGainOrLoss,
			actualValue:   project.Values.RealizedGainOrLoss,
			tolerance:     oracle.Metadata.FinancialTolerances["realized_gain_or_loss"],
		},
		{
			field:         "values.allocated_basis",
			expectedValue: oracle.Values.AllocatedBasis,
			actualValue:   project.Values.AllocatedBasis,
			tolerance:     oracle.Metadata.FinancialTolerances["allocated_basis"],
		},
		{
			field:         "values.closing_quantity",
			expectedValue: oracle.Values.ClosingQuantity,
			actualValue:   project.Values.ClosingQuantity,
			tolerance:     exactComparisonTolerance,
		},
		{
			field:         "values.closing_basis",
			expectedValue: oracle.Values.ClosingBasis,
			actualValue:   project.Values.ClosingBasis,
			tolerance:     oracle.Metadata.FinancialTolerances["closing_basis"],
		},
	}

	var comparisonIndex int
	for comparisonIndex = range valueComparisons {
		var result, err = compareDecimalField(
			oracle,
			valueComparisons[comparisonIndex].field,
			valueComparisons[comparisonIndex].expectedValue,
			valueComparisons[comparisonIndex].actualValue,
			valueComparisons[comparisonIndex].tolerance,
			valueComparisons[comparisonIndex].relevantIDs,
		)
		if err != nil {
			return EmpiricalComparisonOutcome{}, err
		}

		outcome.Results = append(outcome.Results, result)
	}

	var oracleMatches, err = canonicalOracleMatches(oracle.Matches)
	if err != nil {
		return EmpiricalComparisonOutcome{}, fmt.Errorf("compare project output %s: canonicalize oracle matches: %w", oracle.CaseID, err)
	}
	var projectMatches []ProjectMatchEvidence
	projectMatches, err = canonicalProjectMatches(project.Matches)
	if err != nil {
		return EmpiricalComparisonOutcome{}, fmt.Errorf("compare project output %s: canonicalize project matches: %w", oracle.CaseID, err)
	}

	if len(projectMatches) != len(oracleMatches) {
		return EmpiricalComparisonOutcome{}, fmt.Errorf(
			"compare project output %s %s: match evidence count mismatch: expected %d got %d",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			len(oracleMatches),
			len(projectMatches),
		)
	}

	var matchIndex int
	for matchIndex = range oracleMatches {
		if err := compareMatchMetadata(matchIndex, oracle, oracleMatches[matchIndex], projectMatches[matchIndex]); err != nil {
			return EmpiricalComparisonOutcome{}, err
		}

		var relevantIDs = comparisonRelevantSourceIDs(
			oracleMatches[matchIndex].DisposedSourceID,
			oracleMatches[matchIndex].AcquisitionSourceID,
		)

		var result EmpiricalComparisonResult
		result, err = compareDecimalField(
			oracle,
			fmt.Sprintf("matches[%d].matched_quantity", matchIndex),
			oracleMatches[matchIndex].MatchedQuantity,
			projectMatches[matchIndex].MatchedQuantity,
			exactComparisonTolerance,
			relevantIDs,
		)
		if err != nil {
			return EmpiricalComparisonOutcome{}, err
		}
		outcome.Results = append(outcome.Results, result)

		result, err = compareDecimalField(
			oracle,
			fmt.Sprintf("matches[%d].matched_basis", matchIndex),
			oracleMatches[matchIndex].MatchedBasis,
			projectMatches[matchIndex].MatchedBasis,
			exactComparisonTolerance,
			relevantIDs,
		)
		if err != nil {
			return EmpiricalComparisonOutcome{}, err
		}
		outcome.Results = append(outcome.Results, result)

		result, err = compareOptionalMatchDecimalField(
			oracle,
			fmt.Sprintf("matches[%d].matched_proceeds", matchIndex),
			oracleMatches[matchIndex].MatchedProceeds,
			projectMatches[matchIndex].MatchedProceeds,
			relevantIDs,
		)
		if err != nil {
			return EmpiricalComparisonOutcome{}, err
		}
		if result.Field != "" {
			outcome.Results = append(outcome.Results, result)
		}

		result, err = compareOptionalMatchDecimalField(
			oracle,
			fmt.Sprintf("matches[%d].matched_gain_or_loss", matchIndex),
			oracleMatches[matchIndex].MatchedGainOrLoss,
			projectMatches[matchIndex].MatchedGainOrLoss,
			relevantIDs,
		)
		if err != nil {
			return EmpiricalComparisonOutcome{}, err
		}
		if result.Field != "" {
			outcome.Results = append(outcome.Results, result)
		}
	}

	return outcome, nil
}

// FormatEmpiricalComparisonFailures renders one stable multi-line summary of the
// failed comparison results only.
//
// Example:
//
//	text := fixture.FormatEmpiricalComparisonFailures(outcome.Results)
//	_ = text
//
// Authored by: OpenCode
func FormatEmpiricalComparisonFailures(results []EmpiricalComparisonResult) string {
	var failed = make([]EmpiricalComparisonResult, 0)
	var index int

	for index = range results {
		if !results[index].Passed {
			failed = append(failed, results[index])
		}
	}

	if len(failed) == 0 {
		return ""
	}

	sort.Slice(failed, func(left int, right int) bool {
		return empiricalComparisonSortKey(failed[left]) < empiricalComparisonSortKey(failed[right])
	})

	var builder strings.Builder
	for index = range failed {
		if index > 0 {
			builder.WriteByte('\n')
		}

		builder.WriteString("case=")
		builder.WriteString(failed[index].CaseID)
		builder.WriteString(" method=")
		builder.WriteString(string(failed[index].Method))
		builder.WriteString(" year=")
		builder.WriteString(fmt.Sprintf("%d", failed[index].Year))
		builder.WriteString(" asset=")
		builder.WriteString(failed[index].AssetIdentityKey)
		builder.WriteString(" field=")
		builder.WriteString(failed[index].Field)
		builder.WriteString(" expected=")
		builder.WriteString(failed[index].ExpectedValue)
		builder.WriteString(" actual=")
		builder.WriteString(failed[index].ActualValue)
		builder.WriteString(" difference=")
		builder.WriteString(failed[index].Difference)
		builder.WriteString(" tolerance=")
		builder.WriteString(failed[index].Tolerance)
		builder.WriteString(" decimal_policy=")
		builder.WriteString(failed[index].DecimalPolicy)
		if len(failed[index].RelevantSourceIDs) != 0 {
			builder.WriteString(" source_ids=")
			builder.WriteString(strings.Join(failed[index].RelevantSourceIDs, ","))
		}
		if strings.TrimSpace(failed[index].DiagnosticContext) != "" {
			builder.WriteString(" context=")
			builder.WriteString(failed[index].DiagnosticContext)
		}
	}

	return builder.String()
}

// buildEmpiricalComparisonSkips converts unsupported oracle segments into
// deterministic informational skip records.
// Authored by: OpenCode
func buildEmpiricalComparisonSkips(oracle OracleOutput) []EmpiricalComparisonSkip {
	var skips = make([]EmpiricalComparisonSkip, 0, len(oracle.UnsupportedSegments))
	var segments, err = canonicalUnsupportedOracleSegments(oracle.UnsupportedSegments)
	if err != nil {
		segments = append([]UnsupportedOracleSegment(nil), oracle.UnsupportedSegments...)
	}

	var index int
	for index = range segments {
		var relevantIDs = copyStringSlice(segments[index].ActivitySourceIDs)
		sort.Strings(relevantIDs)
		skips = append(skips, EmpiricalComparisonSkip{
			CaseID:            oracle.CaseID,
			Method:            oracle.Method,
			Year:              oracle.Year,
			AssetIdentityKey:  oracle.AssetIdentityKey,
			ComparisonPolicy:  segments[index].ComparisonPolicy,
			Reason:            strings.TrimSpace(segments[index].Reason),
			RelevantSourceIDs: relevantIDs,
			DiagnosticContext: fmt.Sprintf(
				"unsupported external-oracle segment policy=%s reason=%s source_ids=%s",
				segments[index].ComparisonPolicy,
				strings.TrimSpace(segments[index].Reason),
				strings.Join(relevantIDs, ","),
			),
		})
	}

	return skips
}

// validateComparableSegmentIdentity verifies that the compared project and
// oracle segments describe the same empirical slice.
// Authored by: OpenCode
func validateComparableSegmentIdentity(project ProjectCalculationOutput, oracle OracleOutput) error {
	if project.CaseID != oracle.CaseID {
		return fmt.Errorf("compare project output: case_id mismatch: expected %s got %s", oracle.CaseID, project.CaseID)
	}
	if project.Method != oracle.Method {
		return fmt.Errorf("compare project output %s: method mismatch: expected %s got %s", oracle.CaseID, oracle.Method, project.Method)
	}
	if project.Year != oracle.Year {
		return fmt.Errorf("compare project output %s: year mismatch: expected %d got %d", oracle.CaseID, oracle.Year, project.Year)
	}
	if project.AssetIdentityKey != oracle.AssetIdentityKey {
		return fmt.Errorf(
			"compare project output %s: asset_identity_key mismatch: expected %s got %s",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			project.AssetIdentityKey,
		)
	}

	return nil
}

// compareDecimalField compares one required decimal field using exact quantity
// equality or fixture-declared financial tolerance.
// Authored by: OpenCode
func compareDecimalField(
	oracle OracleOutput,
	field string,
	expectedValue string,
	actualValue string,
	toleranceValue string,
	relevantSourceIDs []string,
) (EmpiricalComparisonResult, error) {
	var expectedDecimal, canonicalExpectedValue, err = ParseDecimalString(expectedValue)
	if err != nil {
		return EmpiricalComparisonResult{}, fmt.Errorf("compare project output %s %s: expected %s: %w", oracle.CaseID, oracle.AssetIdentityKey, field, err)
	}
	var actualDecimal apd.Decimal
	var canonicalActualValue string
	actualDecimal, canonicalActualValue, err = ParseDecimalString(actualValue)
	if err != nil {
		return EmpiricalComparisonResult{}, fmt.Errorf("compare project output %s %s: actual %s: %w", oracle.CaseID, oracle.AssetIdentityKey, field, err)
	}
	var toleranceDecimal apd.Decimal
	var canonicalToleranceValue string
	toleranceDecimal, canonicalToleranceValue, err = ParseDecimalString(toleranceValue)
	if err != nil {
		return EmpiricalComparisonResult{}, fmt.Errorf("compare project output %s %s: tolerance %s: %w", oracle.CaseID, oracle.AssetIdentityKey, field, err)
	}
	if !isQuantityComparisonField(field) && toleranceDecimal.Sign() == 0 {
		var impliedTolerance apd.Decimal
		var impliedToleranceText string
		impliedTolerance, impliedToleranceText, err = impliedPersistedFinancialTolerance(expectedValue)
		if err != nil {
			return EmpiricalComparisonResult{}, fmt.Errorf("compare project output %s %s: implied tolerance %s: %w", oracle.CaseID, oracle.AssetIdentityKey, field, err)
		}
		if impliedTolerance.Sign() > 0 {
			toleranceDecimal = impliedTolerance
			canonicalToleranceValue = impliedToleranceText
		}
	}

	if isQuantityComparisonField(field) && toleranceDecimal.Sign() != 0 {
		return EmpiricalComparisonResult{}, fmt.Errorf("compare project output %s %s: quantity field %s must use tolerance 0", oracle.CaseID, oracle.AssetIdentityKey, field)
	}

	var difference apd.Decimal
	difference, err = absoluteDecimalDifference(expectedDecimal, actualDecimal)
	if err != nil {
		return EmpiricalComparisonResult{}, fmt.Errorf("compare project output %s %s: difference %s: %w", oracle.CaseID, oracle.AssetIdentityKey, field, err)
	}
	var canonicalDifference string
	canonicalDifference, err = CanonicalDecimalString(difference)
	if err != nil {
		return EmpiricalComparisonResult{}, fmt.Errorf("compare project output %s %s: difference %s: %w", oracle.CaseID, oracle.AssetIdentityKey, field, err)
	}

	var comparison = EmpiricalComparisonResult{
		CaseID:            oracle.CaseID,
		Method:            oracle.Method,
		Year:              oracle.Year,
		AssetIdentityKey:  oracle.AssetIdentityKey,
		Field:             field,
		ExpectedValue:     canonicalExpectedValue,
		ActualValue:       canonicalActualValue,
		Difference:        canonicalDifference,
		DecimalPolicy:     oracle.Metadata.DecimalPolicy,
		Tolerance:         canonicalToleranceValue,
		Passed:            difference.Cmp(&toleranceDecimal) <= 0,
		RelevantSourceIDs: stableComparisonSourceIDs(relevantSourceIDs),
	}
	comparison.DiagnosticContext = buildComparisonDiagnosticContext(comparison)

	return comparison, nil
}

// compareOptionalMatchDecimalField compares one optional match-level decimal
// field when both sides provide a value.
// Authored by: OpenCode
func compareOptionalMatchDecimalField(
	oracle OracleOutput,
	field string,
	expectedValue string,
	actualValue string,
	relevantSourceIDs []string,
) (EmpiricalComparisonResult, error) {
	if expectedValue == "" && actualValue == "" {
		return EmpiricalComparisonResult{}, nil
	}
	if expectedValue == "" || actualValue == "" {
		return EmpiricalComparisonResult{}, fmt.Errorf(
			"compare project output %s %s: %s presence mismatch: expected %q actual %q",
			oracle.CaseID,
			oracle.AssetIdentityKey,
			field,
			expectedValue,
			actualValue,
		)
	}

	return compareDecimalField(oracle, field, expectedValue, actualValue, exactComparisonTolerance, relevantSourceIDs)
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

// absoluteDecimalDifference returns the absolute exact difference between two
// finite decimal values.
// Authored by: OpenCode
func absoluteDecimalDifference(expected apd.Decimal, actual apd.Decimal) (apd.Decimal, error) {
	var difference, err = supportmath.Subtract(actual, expected)
	if err != nil {
		return apd.Decimal{}, err
	}
	if difference.Sign() >= 0 {
		return difference, nil
	}

	var zero = apd.Decimal{}
	return supportmath.Subtract(zero, difference)
}

// isQuantityComparisonField reports whether one comparison field must use exact
// quantity equality with zero tolerance.
// Authored by: OpenCode
func isQuantityComparisonField(field string) bool {
	return strings.Contains(field, "quantity")
}

// comparisonRelevantSourceIDs returns the stable set of non-empty source IDs for
// one evidence comparison row.
// Authored by: OpenCode
func comparisonRelevantSourceIDs(values ...string) []string {
	return stableComparisonSourceIDs(values)
}

// stableComparisonSourceIDs sorts and de-duplicates one source-ID slice.
// Authored by: OpenCode
func stableComparisonSourceIDs(values []string) []string {
	var unique = make(map[string]struct{}, len(values))
	var normalized = make([]string, 0, len(values))
	var index int

	for index = range values {
		var value = strings.TrimSpace(values[index])
		if value == "" {
			continue
		}
		if _, seen := unique[value]; seen {
			continue
		}

		unique[value] = struct{}{}
		normalized = append(normalized, value)
	}

	sort.Strings(normalized)
	return normalized
}

// impliedPersistedFinancialTolerance returns one implicit one-unit tolerance at
// the persisted expected-value scale when that value uses fractional precision.
// Authored by: OpenCode
func impliedPersistedFinancialTolerance(expectedValue string) (apd.Decimal, string, error) {
	var trimmed = strings.TrimSpace(expectedValue)
	var decimalIndex = strings.IndexByte(trimmed, '.')
	if decimalIndex < 0 {
		return ParseDecimalString("0")
	}

	var scale = len(trimmed) - decimalIndex - 1
	if scale <= 0 {
		return ParseDecimalString("0")
	}

	var builder strings.Builder
	builder.WriteString("0.")
	if scale > 1 {
		builder.WriteString(strings.Repeat("0", scale-1))
	}
	builder.WriteByte('1')

	return ParseDecimalString(builder.String())
}

// buildComparisonDiagnosticContext renders one non-secret deterministic context
// string for a single comparison result.
// Authored by: OpenCode
func buildComparisonDiagnosticContext(result EmpiricalComparisonResult) string {
	var builder strings.Builder

	builder.WriteString("case=")
	builder.WriteString(result.CaseID)
	builder.WriteString(" method=")
	builder.WriteString(string(result.Method))
	builder.WriteString(" year=")
	builder.WriteString(fmt.Sprintf("%d", result.Year))
	builder.WriteString(" asset=")
	builder.WriteString(result.AssetIdentityKey)
	builder.WriteString(" field=")
	builder.WriteString(result.Field)
	builder.WriteString(" expected=")
	builder.WriteString(result.ExpectedValue)
	builder.WriteString(" actual=")
	builder.WriteString(result.ActualValue)
	builder.WriteString(" difference=")
	builder.WriteString(result.Difference)
	builder.WriteString(" tolerance=")
	builder.WriteString(result.Tolerance)
	builder.WriteString(" decimal_policy=")
	builder.WriteString(result.DecimalPolicy)
	if len(result.RelevantSourceIDs) != 0 {
		builder.WriteString(" source_ids=")
		builder.WriteString(strings.Join(result.RelevantSourceIDs, ","))
	}

	return builder.String()
}

// empiricalComparisonSortKey returns the stable lexical sort key for one
// comparison result.
// Authored by: OpenCode
func empiricalComparisonSortKey(result EmpiricalComparisonResult) string {
	return strings.Join([]string{
		result.CaseID,
		string(result.Method),
		fmt.Sprintf("%09d", result.Year),
		result.AssetIdentityKey,
		result.Field,
		strings.Join(result.RelevantSourceIDs, "\x01"),
	}, "\x00")
}
