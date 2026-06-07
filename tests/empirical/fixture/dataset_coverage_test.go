package fixture

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// requiredMethodCoverageTags pins the stable method coverage tag identifiers in
// the same order as the supported report cost-basis methods.
// Authored by: OpenCode
var requiredMethodCoverageTags = requiredMethodCoverageTagSet()

// requiredEdgeCaseCoverageTags pins the stable edge-case coverage tag
// identifiers derived from the empirical dataset specification.
// Authored by: OpenCode
var requiredEdgeCaseCoverageTags = requiredEdgeCaseCoverageTagSet()

// coverageTagIdentifierPattern enforces the snake_case coverage tag identifier
// format used by the empirical dataset contract.
// Authored by: OpenCode
var coverageTagIdentifierPattern = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)

// TestValidateDatasetCoverageRequiredMethodTagsMatchSupportedMethods verifies
// the explicit required method coverage tags stay aligned with the supported
// application methods.
// Authored by: OpenCode
func TestValidateDatasetCoverageRequiredMethodTagsMatchSupportedMethods(t *testing.T) {
	t.Parallel()

	if !reflect.DeepEqual(requiredMethodCoverageTags, supportedMethodCoverageTags()) {
		t.Fatalf("required method coverage tags drifted from supported methods: got %v want %v", requiredMethodCoverageTags, supportedMethodCoverageTags())
	}
}

// TestValidateDatasetCoverageRequiredTagsUseStableIdentifiers verifies the
// explicit required coverage tag identifiers stay unique and use canonical
// snake_case names.
// Authored by: OpenCode
func TestValidateDatasetCoverageRequiredTagsUseStableIdentifiers(t *testing.T) {
	t.Parallel()

	var seen = make(map[string]struct{}, len(requiredMethodCoverageTags)+len(requiredEdgeCaseCoverageTags))

	for _, tag := range allRequiredDatasetCoverageTags() {
		if !coverageTagIdentifierPattern.MatchString(tag) {
			t.Fatalf("coverage tag %q must use lowercase snake_case", tag)
		}

		if _, ok := seen[tag]; ok {
			t.Fatalf("duplicate required coverage tag %q", tag)
		}

		seen[tag] = struct{}{}
	}
}

// TestValidateDatasetCoverageAcceptsCompleteRequiredMethodAndEdgeCaseTags
// verifies a minimal but internally consistent dataset fixture passes when all
// required method and edge-case coverage tags are present.
// Authored by: OpenCode
func TestValidateDatasetCoverageAcceptsCompleteRequiredMethodAndEdgeCaseTags(t *testing.T) {
	t.Parallel()

	var dataset = newDatasetCoverageFixture()

	if err := ValidateDatasetCoverage(dataset); err != nil {
		t.Fatalf("expected complete coverage fixture to validate, got %v", err)
	}
}

// TestValidateDatasetCoverageRejectsEachMissingMethodTag verifies every
// required method coverage tag is mandatory independent of the case method
// fields.
// Authored by: OpenCode
func TestValidateDatasetCoverageRejectsEachMissingMethodTag(t *testing.T) {
	t.Parallel()

	for _, missingTag := range requiredMethodCoverageTags {
		var missingTag = missingTag

		t.Run(missingTag, func(t *testing.T) {
			t.Parallel()

			var dataset = withoutCoverageTag(newDatasetCoverageFixture(), missingTag)
			var err = ValidateDatasetCoverage(dataset)

			assertMissingCoverageTagError(t, err, missingTag)
		})
	}
}

// TestValidateDatasetCoverageRejectsEachMissingEdgeCaseTag verifies every
// required edge-case coverage tag is mandatory.
// Authored by: OpenCode
func TestValidateDatasetCoverageRejectsEachMissingEdgeCaseTag(t *testing.T) {
	t.Parallel()

	for _, missingTag := range requiredEdgeCaseCoverageTags {
		var missingTag = missingTag

		t.Run(missingTag, func(t *testing.T) {
			t.Parallel()

			var dataset = withoutCoverageTag(newDatasetCoverageFixture(), missingTag)
			var err = ValidateDatasetCoverage(dataset)

			assertMissingCoverageTagError(t, err, missingTag)
		})
	}
}

// newDatasetCoverageFixture builds one minimal dataset fixture whose coverage
// fields intentionally satisfy the required method and edge-case tag contract.
// Authored by: OpenCode
func newDatasetCoverageFixture() EmpiricalDataset {
	var tags = allRequiredDatasetCoverageTags()
	var supportedMethods = reportmodel.SupportedCostBasisMethods()
	var activities = make([]EmpiricalActivity, 0, len(tags))
	var cases = make([]EmpiricalCase, 0, len(tags))

	for index, tag := range tags {
		var year = 2023 + index%3
		var sourceID = fmt.Sprintf("emp-act-%06d", index+1)
		var caseMethods = coverageCaseMethodsForTag(supportedMethods, index, tag)

		activities = append(activities, EmpiricalActivity{
			SourceID:           sourceID,
			OccurredAt:         fmt.Sprintf("%04d-01-02T09:00:00Z", year),
			DeterministicOrder: index + 1,
			ActivityType:       syncmodel.ActivityTypeBuy,
			AssetIdentityKey:   "asset-alpha",
			AssetSymbol:        "ALPHA",
			Quantity:           "1",
			GrossValue:         "10",
			UnitPrice:          "10",
			FeeAmount:          "0",
			Currency:           "USD",
			CoverageTags:       []string{tag},
		})

		cases = append(cases, EmpiricalCase{
			CaseID:            fmt.Sprintf("coverage-case-%02d", index+1),
			Description:       "Coverage contract fixture for " + tag,
			Methods:           caseMethods,
			Year:              2024,
			AssetIdentityKeys: []string{"asset-alpha"},
			ActivitySourceIDs: []string{sourceID},
			CoverageTags:      []string{tag},
			OracleSupport:     OracleSupportSupported,
		})
	}

	return EmpiricalDataset{
		DatasetVersion:   "1",
		Description:      "Synthetic empirical dataset coverage contract fixture",
		Currency:         "USD",
		SupportedYears:   []int{2023, 2024, 2025},
		SupportedMethods: supportedMethods,
		CoverageTags:     tags,
		Activities:       activities,
		Cases:            cases,
	}
}

// supportedMethodCoverageTags returns the supported application methods in the
// stable string form required for coverage tags.
// Authored by: OpenCode
func supportedMethodCoverageTags() []string {
	var methods = reportmodel.SupportedCostBasisMethods()
	var tags = make([]string, 0, len(methods))

	for _, method := range methods {
		tags = append(tags, string(method))
	}

	return tags
}

// allRequiredCoverageTags returns the full explicit required coverage tag set
// in stable order.
// Authored by: OpenCode
func allRequiredCoverageTags() []string {
	var tags = make([]string, 0, len(requiredMethodCoverageTags)+len(requiredEdgeCaseCoverageTags))

	tags = append(tags, requiredMethodCoverageTags...)
	tags = append(tags, requiredEdgeCaseCoverageTags...)

	return tags
}

// coverageCaseMethodsForTag selects one stable case method set for the given
// coverage tag while keeping method coverage tags independent from `methods`
// field coverage.
// Authored by: OpenCode
func coverageCaseMethodsForTag(supportedMethods []reportmodel.CostBasisMethod, index int, tag string) []reportmodel.CostBasisMethod {
	if isMethodCoverageTag(tag) {
		return []reportmodel.CostBasisMethod{reportmodel.CostBasisMethod(tag)}
	}

	return []reportmodel.CostBasisMethod{supportedMethods[index%len(supportedMethods)]}
}

// isMethodCoverageTag reports whether one coverage tag is one of the required
// method coverage identifiers.
// Authored by: OpenCode
func isMethodCoverageTag(tag string) bool {
	return slices.Contains(requiredMethodCoverageTags, tag)
}

// withoutCoverageTag removes one coverage tag from dataset-level, activity-level,
// and case-level coverage tag lists while leaving supported-method metadata
// intact.
// Authored by: OpenCode
func withoutCoverageTag(dataset EmpiricalDataset, missingTag string) EmpiricalDataset {
	dataset.CoverageTags = filterStringsWithoutTag(dataset.CoverageTags, missingTag)

	for index := range dataset.Activities {
		dataset.Activities[index].CoverageTags = filterStringsWithoutTag(dataset.Activities[index].CoverageTags, missingTag)
	}

	for index := range dataset.Cases {
		dataset.Cases[index].CoverageTags = filterStringsWithoutTag(dataset.Cases[index].CoverageTags, missingTag)
	}

	return dataset
}

// filterStringsWithoutTag removes one tag from a string slice while preserving
// the remaining order.
// Authored by: OpenCode
func filterStringsWithoutTag(values []string, missingTag string) []string {
	var filtered = make([]string, 0, len(values))

	for _, value := range values {
		if value == missingTag {
			continue
		}

		filtered = append(filtered, value)
	}

	return filtered
}

// assertMissingCoverageTagError verifies validation fails and reports the
// missing tag without requiring a fully pinned error sentence.
// Authored by: OpenCode
func assertMissingCoverageTagError(t *testing.T, err error, missingTag string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected missing coverage tag %q to fail validation", missingTag)
	}

	if !strings.Contains(err.Error(), "coverage") {
		t.Fatalf("expected coverage-oriented error for %q, got %v", missingTag, err)
	}

	if !strings.Contains(err.Error(), missingTag) {
		t.Fatalf("expected error for missing coverage tag %q, got %v", missingTag, err)
	}
}
