// Package fixture provides empirical fixture loading and validation helpers.
// Authored by: OpenCode
package fixture

import (
	"fmt"
	"strconv"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// validateMatches enforces the comparable match-evidence schema rules.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateMatches() {
	if validator.output.Matches == nil {
		return
	}

	var index int
	for index = range validator.output.Matches {
		validator.validateMatch(index, validator.output.Matches[index])
	}
}

// validateMatch enforces one comparable match-evidence row.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateMatch(index int, match OracleMatchEvidence) {
	var referenceValue = strconv.Itoa(index)

	if strings.TrimSpace(match.DisposedSourceID) == "" {
		validator.addIssue("match_index", referenceValue, "disposed_source_id", "required_field", "disposed_source_id must be non-empty")
	}

	validator.validateRequiredCanonicalDecimal("match_index", referenceValue, "matched_quantity", match.MatchedQuantity)
	validator.validateRequiredCanonicalDecimal("match_index", referenceValue, "matched_basis", match.MatchedBasis)
	validator.validateOptionalCanonicalDecimal("match_index", referenceValue, "matched_proceeds", match.MatchedProceeds)
	validator.validateOptionalCanonicalDecimal("match_index", referenceValue, "matched_gain_or_loss", match.MatchedGainOrLoss)

	switch match.SupportLabel {
	case "", EvidenceSupportLabelRotkiBacked:
	case EvidenceSupportLabelProjectCompositionRule:
		if strings.TrimSpace(match.CompositionRuleID) == "" {
			validator.addIssue("match_index", referenceValue, "composition_rule_id", "composition_rule", "project_composition_rule evidence requires composition_rule_id")
		}
	default:
		validator.addIssue("match_index", referenceValue, "support_label", "support_label", fmt.Sprintf("unsupported support label %s", match.SupportLabel))
	}

	if validator.output.Method == reportmodel.CostBasisMethodScopeLocalHybrid && match.SupportLabel == "" {
		validator.addIssue("match_index", referenceValue, "support_label", "support_label", "scope_local_hybrid matches must declare support_label")
	}
	if match.SupportLabel != EvidenceSupportLabelProjectCompositionRule && strings.TrimSpace(match.CompositionRuleID) != "" {
		validator.addIssue("match_index", referenceValue, "composition_rule_id", "composition_rule", "composition_rule_id is allowed only for project_composition_rule evidence")
	}
}

// validateUnsupportedSegments enforces the explicit unsupported-segment rules.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateUnsupportedSegments() {
	if validator.output.UnsupportedSegments == nil {
		return
	}

	var index int
	for index = range validator.output.UnsupportedSegments {
		validator.validateUnsupportedSegment(index, validator.output.UnsupportedSegments[index])
	}
}

// validateUnsupportedSegment enforces one explicit unsupported segment.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateUnsupportedSegment(index int, segment UnsupportedOracleSegment) {
	var referenceValue = strconv.Itoa(index)

	if strings.TrimSpace(segment.CaseID) == "" {
		validator.addIssue("unsupported_index", referenceValue, "case_id", "required_field", "case_id must be non-empty")
	} else if segment.CaseID != validator.output.CaseID {
		validator.addIssue("unsupported_index", referenceValue, "case_id", "case_id", fmt.Sprintf("unsupported segment case_id %s must match oracle output case_id %s", segment.CaseID, validator.output.CaseID))
	}

	if !isSupportedCostBasisMethod(segment.Method) {
		validator.addIssue("unsupported_index", referenceValue, "method", "supported_method", fmt.Sprintf("unsupported cost basis method %s", segment.Method))
	} else if segment.Method != validator.output.Method {
		validator.addIssue("unsupported_index", referenceValue, "method", "method", fmt.Sprintf("unsupported segment method %s must match oracle output method %s", segment.Method, validator.output.Method))
	}

	if len(segment.ActivitySourceIDs) == 0 {
		validator.addIssue("unsupported_index", referenceValue, "activity_source_ids", "required_field", "activity_source_ids must contain at least one source_id")
	}

	var sourceIndex int
	for sourceIndex = range segment.ActivitySourceIDs {
		if strings.TrimSpace(segment.ActivitySourceIDs[sourceIndex]) != "" {
			continue
		}

		validator.addIssue("unsupported_index", referenceValue, "activity_source_ids", "activity_source_ids", "activity_source_ids must not contain blank values")
		break
	}

	if strings.TrimSpace(segment.Reason) == "" {
		validator.addIssue("unsupported_index", referenceValue, "reason", "required_field", "reason must be non-empty")
	}

	switch segment.ComparisonPolicy {
	case ComparisonPolicySkipExternalOracle, ComparisonPolicyProjectCompositionOnly, ComparisonPolicyFailIfSelected:
	default:
		validator.addIssue("unsupported_index", referenceValue, "comparison_policy", "comparison_policy", fmt.Sprintf("unsupported comparison_policy %s", segment.ComparisonPolicy))
	}
}
