// Package calculate defines asset grouping and per-asset orchestration for
// yearly gains-and-losses report calculation.
// Authored by: OpenCode
package calculate

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// assetInputGroup stores one asset's selected calculation inputs in synced
// deterministic replay order.
// Authored by: OpenCode
type assetInputGroup struct {
	AssetIdentityKey string
	DisplayLabel     string
	Inputs           []reportmodel.ActivityCalculationInput
}

// assetCalculationResult stores one asset's calculated report contributions.
// Authored by: OpenCode
type assetCalculationResult struct {
	IncludeInMain  bool
	SummaryEntry   reportmodel.AssetSummaryEntry
	ReferenceEntry *reportmodel.ReferenceLiquidationEntry
	DetailSection  reportmodel.AssetDetailSection
	AuditSection   reportmodel.PerAssetAuditSection
	IncludeInAudit bool
	YearlyNet      apd.Decimal
}

// selectAssetInputGroupsThroughYear converts selected-year-relevant protected
// activity rows into grouped calculation inputs while preserving synced replay
// order.
// Authored by: OpenCode
func selectAssetInputGroupsThroughYear(records []syncmodel.ActivityRecord, selectedYear int) ([]assetInputGroup, error) {
	var orderedKeys []string
	var groupsByKey = make(map[string]*assetInputGroup)

	for _, record := range records {
		var occurredAt, err = parseActivityOccurredAt(record)
		if err != nil {
			return nil, newRecordCalculationError(
				reportmodel.CalculationErrorKindActivityInput,
				record,
				"could not read the activity timestamp",
				err,
			)
		}
		if occurredAt.Year() > selectedYear {
			continue
		}
		if strings.TrimSpace(record.AssetIdentityKey) == "" {
			return nil, newRecordCalculationError(
				reportmodel.CalculationErrorKindActivityInput,
				record,
				"activity is missing the stored asset identity key required for reporting",
				nil,
			)
		}

		var input reportmodel.ActivityCalculationInput
		input, err = SelectActivityCalculationInput(record)
		if err != nil {
			return nil, newRecordCalculationError(
				reportmodel.CalculationErrorKindActivityInput,
				record,
				err.Error(),
				err,
			)
		}

		var group = groupsByKey[input.AssetIdentityKey]
		if group == nil {
			group = &assetInputGroup{
				AssetIdentityKey: input.AssetIdentityKey,
				DisplayLabel:     input.DisplayLabel,
			}
			groupsByKey[input.AssetIdentityKey] = group
			orderedKeys = append(orderedKeys, input.AssetIdentityKey)
		}
		if group.DisplayLabel == "" && strings.TrimSpace(input.DisplayLabel) != "" {
			group.DisplayLabel = input.DisplayLabel
		}
		group.Inputs = append(group.Inputs, input)
	}

	var groups = make([]assetInputGroup, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		groups = append(groups, *groupsByKey[key])
	}

	return groups, nil
}

// calculateAssetGroup replays one grouped asset history through the selected
// year cutoff and derives its summary, reference, and detail contributions.
// Authored by: OpenCode
func calculateAssetGroup(method reportmodel.CostBasisMethod, selectedYear int, group assetInputGroup) (assetCalculationResult, error) {
	var basisState, err = newAssetBasisStateFunc(method)
	if err != nil {
		return assetCalculationResult{}, newGroupCalculationError(reportmodel.CalculationErrorKindUnsupportedCostBasisMethod, group, err.Error(), err)
	}

	var scopedInputs []scopedActivityInput
	scopedInputs, err = resolveScopedInputsFunc(method, group)
	if err != nil {
		return assetCalculationResult{}, err
	}

	var replayState assetReplayState
	replayState, err = replayAssetGroup(basisState, scopedInputs, selectedYear)
	if err != nil {
		return assetCalculationResult{}, wrapAssetGroupReplayError(group, err)
	}

	return buildAssetCalculationResult(group, replayState)
}
