package main

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

const (
	oracleDecimalPolicy      = "scale=16,rounding=half_up"
	reliableHybridFIFOCaseID = "case-scope-local-reliable-epsilon-2024"
)

// caseHasMethod reports whether one case explicitly requests the provided
// cost-basis method.
// Authored by: OpenCode
func caseHasMethod(empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod) bool {
	var methodIndex int

	for methodIndex = range empiricalCase.Methods {
		if empiricalCase.Methods[methodIndex] == method {
			return true
		}
	}

	return false
}

// zeroPricedReductionOmissionNote records a deterministic generation note when
// active external-oracle generation omits a zero-priced reduction.
// Authored by: OpenCode
func zeroPricedReductionOmissionNote(activity fixture.EmpiricalActivity, empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod) string {
	return "omitted zero-priced reduction " + strings.TrimSpace(activity.SourceID) +
		" from " + strings.TrimSpace(empiricalCase.CaseID) +
		" because lot mode " + journalLotMode(method, empiricalCase.CaseID) +
		" does not support native zero-priced handling"
}

// journalLotMode returns the lot-mode label used by external-oracle
// compatibility decisions.
// Authored by: OpenCode
func journalLotMode(method reportmodel.CostBasisMethod, caseID string) string {
	switch method {
	case reportmodel.CostBasisMethodFIFO:
		return "FIFO"
	case reportmodel.CostBasisMethodLIFO:
		return "LIFO"
	case reportmodel.CostBasisMethodHIFO:
		return "HIFO"
	case reportmodel.CostBasisMethodAverageCost:
		return "AVERAGE"
	case reportmodel.CostBasisMethodScopeLocalHybrid:
		if strings.TrimSpace(caseID) == reliableHybridFIFOCaseID {
			return "FIFO"
		}

		return "AVERAGE"
	default:
		return strings.ToUpper(strings.TrimSpace(string(method)))
	}
}

// isZeroPricedReduction reports whether one SELL row represents a basis-only
// reduction with no priced proceeds.
// Authored by: OpenCode
func isZeroPricedReduction(activity fixture.EmpiricalActivity) bool {
	if strings.TrimSpace(activity.ZeroPricedReductionExplanation) != "" {
		return true
	}

	return activity.ActivityType == syncmodel.ActivityTypeSell &&
		strings.TrimSpace(activity.GrossValue) == "" &&
		strings.TrimSpace(activity.UnitPrice) == "" &&
		strings.TrimSpace(activity.Currency) == ""
}

// compareJournalActivities returns the deterministic empirical activity ordering
// shared by oracle-input selection.
// Authored by: OpenCode
func compareJournalActivities(left fixture.EmpiricalActivity, right fixture.EmpiricalActivity) int {
	var leftOccurredAt = strings.TrimSpace(left.OccurredAt)
	var rightOccurredAt = strings.TrimSpace(right.OccurredAt)
	if leftOccurredAt < rightOccurredAt {
		return -1
	}
	if leftOccurredAt > rightOccurredAt {
		return 1
	}

	var leftAssetKey = strings.TrimSpace(left.AssetIdentityKey)
	var rightAssetKey = strings.TrimSpace(right.AssetIdentityKey)
	if leftAssetKey < rightAssetKey {
		return -1
	}
	if leftAssetKey > rightAssetKey {
		return 1
	}

	if left.DeterministicOrder < right.DeterministicOrder {
		return -1
	}
	if left.DeterministicOrder > right.DeterministicOrder {
		return 1
	}

	var leftSourceID = strings.TrimSpace(left.SourceID)
	var rightSourceID = strings.TrimSpace(right.SourceID)
	if leftSourceID < rightSourceID {
		return -1
	}
	if leftSourceID > rightSourceID {
		return 1
	}

	return 0
}
