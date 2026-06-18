package fixture

import (
	"fmt"
	"sort"
	"strings"
)

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
