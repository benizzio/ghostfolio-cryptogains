package main

import (
	"sort"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// buildUnsupportedSegments derives the explicit unsupported fixture segments for
// one case, method, and target asset.
// Authored by: OpenCode
func buildUnsupportedSegments(
	dataset fixture.EmpiricalDataset,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	assetIdentityKey string,
	generationNotes []string,
	matches []oracleMatchEvidenceInput,
) []unsupportedOracleSegmentInput {
	var segments = make([]unsupportedOracleSegmentInput, 0, 2)
	var omittedZeroPricedSourceIDs = omittedZeroPricedReductionSourceIDs(dataset, empiricalCase, method, assetIdentityKey, generationNotes)
	var hledgerCoveredSourceIDs = hledgerBackedSourceIDSet(matches)

	if empiricalCase.OracleSupport != fixture.OracleSupportSupported {
		var assetSourceIDs = filterUncoveredSourceIDs(caseAssetSourceIDs(dataset, empiricalCase, assetIdentityKey), hledgerCoveredSourceIDs, omittedZeroPricedSourceIDs)
		if len(assetSourceIDs) != 0 {
			segments = append(segments, unsupportedOracleSegmentInput{
				CaseID:            strings.TrimSpace(empiricalCase.CaseID),
				Method:            method,
				ActivitySourceIDs: assetSourceIDs,
				Reason:            strings.TrimSpace(empiricalCase.UnsupportedReason),
				ComparisonPolicy:  fixture.ComparisonPolicyProjectCompositionOnly,
			})
		}
	}

	omittedZeroPricedSourceIDs = filterUncoveredSourceIDs(omittedZeroPricedSourceIDs, hledgerCoveredSourceIDs, nil)
	if len(omittedZeroPricedSourceIDs) != 0 {
		segments = append(segments, unsupportedOracleSegmentInput{
			CaseID:            strings.TrimSpace(empiricalCase.CaseID),
			Method:            method,
			ActivitySourceIDs: omittedZeroPricedSourceIDs,
			Reason:            omittedZeroPricedReductionReason(empiricalCase, method, omittedZeroPricedSourceIDs),
			ComparisonPolicy:  fixture.ComparisonPolicyProjectCompositionOnly,
		})
	}

	return segments
}

// hledgerBackedSourceIDSet returns the source identifiers already covered by hledger-backed match evidence.
// Authored by: OpenCode
func hledgerBackedSourceIDSet(matches []oracleMatchEvidenceInput) map[string]struct{} {
	var coveredSourceIDs = make(map[string]struct{})
	var match oracleMatchEvidenceInput
	for _, match = range matches {
		if match.SupportLabel != fixture.EvidenceSupportLabelHledgerBacked {
			continue
		}
		if strings.TrimSpace(match.DisposedSourceID) != "" {
			coveredSourceIDs[strings.TrimSpace(match.DisposedSourceID)] = struct{}{}
		}
		if strings.TrimSpace(match.AcquisitionSourceID) != "" {
			coveredSourceIDs[strings.TrimSpace(match.AcquisitionSourceID)] = struct{}{}
		}
	}

	return coveredSourceIDs
}

// filterUncoveredSourceIDs removes already-covered or separately-classified source identifiers while preserving stable sorted order.
// Authored by: OpenCode
func filterUncoveredSourceIDs(sourceIDs []string, coveredSourceIDs map[string]struct{}, excludedSourceIDs []string) []string {
	var excludedSourceIDSet = make(map[string]struct{}, len(excludedSourceIDs))
	var excludedSourceID string
	for _, excludedSourceID = range excludedSourceIDs {
		excludedSourceIDSet[strings.TrimSpace(excludedSourceID)] = struct{}{}
	}

	var filtered = make([]string, 0, len(sourceIDs))
	var sourceID string
	for _, sourceID = range sourceIDs {
		var trimmedSourceID = strings.TrimSpace(sourceID)
		if _, covered := coveredSourceIDs[trimmedSourceID]; covered {
			continue
		}
		if _, excluded := excludedSourceIDSet[trimmedSourceID]; excluded {
			continue
		}

		filtered = append(filtered, trimmedSourceID)
	}

	return filtered
}

// caseAssetSourceIDs returns the case source identifiers filtered to the target
// asset identity key.
// Authored by: OpenCode
func caseAssetSourceIDs(dataset fixture.EmpiricalDataset, empiricalCase fixture.EmpiricalCase, assetIdentityKey string) []string {
	var activitiesBySourceID = datasetActivitiesBySourceID(dataset)
	var sourceIDs = make([]string, 0)
	var sourceIndex int

	for sourceIndex = range empiricalCase.ActivitySourceIDs {
		var sourceID = strings.TrimSpace(empiricalCase.ActivitySourceIDs[sourceIndex])
		var activity, found = activitiesBySourceID[sourceID]
		if !found {
			continue
		}
		if strings.TrimSpace(activity.AssetIdentityKey) != strings.TrimSpace(assetIdentityKey) {
			continue
		}

		sourceIDs = append(sourceIDs, sourceID)
	}

	sort.Strings(sourceIDs)
	return sourceIDs
}

// omittedZeroPricedReductionSourceIDs returns the zero-priced reduction rows for
// one asset that the rendered journal explicitly omitted.
// Authored by: OpenCode
func omittedZeroPricedReductionSourceIDs(
	dataset fixture.EmpiricalDataset,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	assetIdentityKey string,
	generationNotes []string,
) []string {
	var noteSet = make(map[string]struct{}, len(generationNotes))
	var noteIndex int
	for noteIndex = range generationNotes {
		noteSet[strings.TrimSpace(generationNotes[noteIndex])] = struct{}{}
	}

	var activitiesBySourceID = datasetActivitiesBySourceID(dataset)
	var sourceIDs = make([]string, 0)
	var sourceIndex int
	for sourceIndex = range empiricalCase.ActivitySourceIDs {
		var sourceID = strings.TrimSpace(empiricalCase.ActivitySourceIDs[sourceIndex])
		var activity, found = activitiesBySourceID[sourceID]
		if !found {
			continue
		}
		if strings.TrimSpace(activity.AssetIdentityKey) != strings.TrimSpace(assetIdentityKey) {
			continue
		}
		if !isZeroPricedReduction(activity) {
			continue
		}

		if _, omitted := noteSet[zeroPricedReductionOmissionNote(activity, empiricalCase, method)]; !omitted {
			continue
		}

		sourceIDs = append(sourceIDs, sourceID)
	}

	sort.Strings(sourceIDs)
	return sourceIDs
}

// omittedZeroPricedReductionReason returns the deterministic unsupported reason
// recorded when a non-native lot mode omits one or more zero-priced reductions.
// Authored by: OpenCode
func omittedZeroPricedReductionReason(empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod, sourceIDs []string) string {
	return "journal omitted zero-priced reduction handling for " + strings.Join(sourceIDs, ", ") +
		" because lot mode " + journalLotMode(method, empiricalCase.CaseID) + " does not support native zero-priced handling"
}

// datasetActivitiesBySourceID builds one deterministic lookup map for dataset
// activity rows keyed by their stable source identifiers.
// Authored by: OpenCode
func datasetActivitiesBySourceID(dataset fixture.EmpiricalDataset) map[string]fixture.EmpiricalActivity {
	var activitiesBySourceID = make(map[string]fixture.EmpiricalActivity, len(dataset.Activities))
	var activityIndex int
	for activityIndex = range dataset.Activities {
		activitiesBySourceID[strings.TrimSpace(dataset.Activities[activityIndex].SourceID)] = dataset.Activities[activityIndex]
	}

	return activitiesBySourceID
}
