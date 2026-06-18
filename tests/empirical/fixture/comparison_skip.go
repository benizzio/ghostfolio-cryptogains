package fixture

import (
	"fmt"
	"sort"
	"strings"
)

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
