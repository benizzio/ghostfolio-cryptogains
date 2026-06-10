package fixture

import (
	"fmt"
	"sort"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// ValidateDatasetCoverage enforces the explicit method and edge-case coverage
// tag contract for one empirical dataset.
//
// Example:
//
//	err := fixture.ValidateDatasetCoverage(dataset)
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateDatasetCoverage(dataset EmpiricalDataset) error {
	var missingTags = make([]string, 0)
	var tag string

	for _, tag = range requiredMethodCoverageTagSet() {
		if caseLevelMethodCoveragePresent(dataset, tag) {
			continue
		}

		missingTags = append(missingTags, tag)
	}
	for _, tag = range requiredEdgeCaseCoverageTagSet() {
		if caseLevelCoverageTagPresent(dataset, tag) {
			continue
		}

		missingTags = append(missingTags, tag)
	}

	if len(missingTags) == 0 {
		return nil
	}

	sort.Strings(missingTags)
	return fmt.Errorf("dataset coverage validation failed: missing required coverage tag(s): %s", strings.Join(missingTags, ", "))
}

// caseLevelMethodCoveragePresent reports whether one case explicitly covers the required method tag and applies that method.
// Authored by: OpenCode
func caseLevelMethodCoveragePresent(dataset EmpiricalDataset, tag string) bool {
	var caseRecord EmpiricalCase

	for _, caseRecord = range dataset.Cases {
		if !caseHasCoverageTag(caseRecord, tag) {
			continue
		}
		if !datasetCaseHasMethod(caseRecord, reportmodel.CostBasisMethod(tag)) {
			continue
		}

		return true
	}

	return false
}

// datasetCaseHasMethod reports whether one case declares the required supported method.
// Authored by: OpenCode
func datasetCaseHasMethod(caseRecord EmpiricalCase, method reportmodel.CostBasisMethod) bool {
	var declaredMethod reportmodel.CostBasisMethod
	for _, declaredMethod = range caseRecord.Methods {
		if declaredMethod == method {
			return true
		}
	}

	return false
}

// caseLevelCoverageTagPresent reports whether one case explicitly declares the required edge-case coverage tag.
// Authored by: OpenCode
func caseLevelCoverageTagPresent(dataset EmpiricalDataset, tag string) bool {
	var caseRecord EmpiricalCase

	for _, caseRecord = range dataset.Cases {
		if caseHasCoverageTag(caseRecord, tag) {
			return true
		}
	}

	return false
}

// requiredMethodCoverageTagSet returns the stable required method coverage tag identifiers.
// Authored by: OpenCode
func requiredMethodCoverageTagSet() []string {
	var tags = make([]string, 0, len(reportmodel.SupportedCostBasisMethods()))
	var method reportmodel.CostBasisMethod

	for _, method = range reportmodel.SupportedCostBasisMethods() {
		tags = append(tags, string(method))
	}

	return tags
}

// requiredEdgeCaseCoverageTagSet returns the explicit required edge-case coverage tags.
// Authored by: OpenCode
func requiredEdgeCaseCoverageTagSet() []string {
	var tags = make([]string, len(requiredDatasetEdgeCaseCoverageTags))
	copy(tags, requiredDatasetEdgeCaseCoverageTags)
	return tags
}

// allRequiredDatasetCoverageTags returns the full required coverage tag set in stable order.
// Authored by: OpenCode
func allRequiredDatasetCoverageTags() []string {
	var methodTags = requiredMethodCoverageTagSet()
	var tags = make([]string, 0, len(methodTags)+len(requiredDatasetEdgeCaseCoverageTags))

	tags = append(tags, methodTags...)
	tags = append(tags, requiredDatasetEdgeCaseCoverageTags...)

	return tags
}

var requiredDatasetEdgeCaseCoverageTags = []string{
	"acquisitions",
	"partial_liquidations",
	"full_liquidations",
	"gain_cases",
	"loss_cases",
	"zero_result_liquidations",
	"fees_on_priced_activity",
	"same_source_calendar_date_ordering",
	"pre_year_opening_positions",
	"multi_year_opening_history",
	"selected_year_in_year_activity",
	"post_selected_year_ignored_activity",
	"full_liquidation_followed_by_reacquisition",
	"excluded_assets_from_selected_year_main_results",
	"selected_year_single_lot_liquidation",
	"selected_year_multi_lot_liquidation",
	"hifo_deterministic_tie_breaking",
	"average_cost_multiple_acquisitions",
	"average_cost_partial_disposal",
	"average_cost_full_disposal",
	"average_cost_pool_reset_after_zero",
	"average_cost_reacquisition_after_zero",
	"scope_local_reliable_activity",
	"scope_local_narrowing",
	"scope_local_unreliable_or_unavailable_activity",
	"scope_local_broadening",
	"scope_local_fallback_activation",
	"scope_local_fallback_carry_forward_until_zero",
	"scope_local_same_scope_reset_after_zero",
	"scope_local_independent_other_scope_state",
	"zero_priced_holding_reduction_explicit_zero_fields",
	"zero_priced_holding_reduction_missing_optional_fields",
	"rounded_internal_division_or_allocation",
	"negative_yearly_totals",
}
