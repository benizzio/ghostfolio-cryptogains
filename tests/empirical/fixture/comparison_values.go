package fixture

// aggregateValueComparison stores one aggregate decimal comparison requested by
// the oracle/project comparator.
// Authored by: OpenCode
type aggregateValueComparison struct {
	field         string
	expectedValue string
	actualValue   string
	tolerance     string
	relevantIDs   []string
}

// compareAggregateValues compares aggregate oracle and project output values.
// Authored by: OpenCode
func compareAggregateValues(
	oracle OracleOutput,
	project ProjectCalculationOutput,
) ([]EmpiricalComparisonResult, error) {
	var valueComparisons = aggregateValueComparisons(oracle, project)
	var results = make([]EmpiricalComparisonResult, 0, len(valueComparisons))
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
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

// aggregateValueComparisons returns the fixed aggregate field comparisons for
// one oracle/project output pair.
// Authored by: OpenCode
func aggregateValueComparisons(
	oracle OracleOutput,
	project ProjectCalculationOutput,
) []aggregateValueComparison {
	return []aggregateValueComparison{
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
}
