package fixture

import (
	"fmt"
	"sort"
	"strings"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

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
