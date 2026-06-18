// Package main contains the rotki-backed oracle boundary used to generate
// normalized empirical fixtures.
//
// Authored by: OpenCode
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
	"github.com/cockroachdb/apd/v3"
)

const rotkiOracleInputRootRepositoryPath = ".cache/empiricaloracle/oracle-inputs"

// rotkiOracleActivityInput stores one generated adapter activity row.
// Authored by: OpenCode
type rotkiOracleActivityInput struct {
	SourceID           string                  `json:"source_id"`
	OccurredAt         string                  `json:"occurred_at"`
	DeterministicOrder int                     `json:"deterministic_order"`
	ActivityType       string                  `json:"activity_type"`
	AssetIdentityKey   string                  `json:"asset_identity_key"`
	AssetSymbol        string                  `json:"asset_symbol"`
	Quantity           string                  `json:"quantity"`
	GrossValue         string                  `json:"gross_value,omitempty"`
	FeeAmount          string                  `json:"fee_amount,omitempty"`
	SourceScope        *fixture.EmpiricalScope `json:"source_scope,omitempty"`
}

// rotkiOracleInput stores one deterministic generated adapter input written to
// the repository-local untracked cache before rotki execution.
// Authored by: OpenCode
type rotkiOracleInput struct {
	CaseID                      string                      `json:"case_id"`
	Method                      reportmodel.CostBasisMethod `json:"method"`
	RotkiMethod                 string                      `json:"rotki_method"`
	Year                        int                         `json:"year"`
	AssetIdentityKey            string                      `json:"asset_identity_key"`
	ComparisonActivitySourceIDs []string                    `json:"comparison_activity_source_ids"`
	Activities                  []rotkiOracleActivityInput  `json:"activities"`
}

// rotkiOracleCapture stores the direct adapter JSON result before fixture
// normalization applies the shared validation contract.
// Authored by: OpenCode
type rotkiOracleCapture struct {
	Values  comparableOutputValuesInput `json:"values"`
	Matches []oracleMatchEvidenceInput  `json:"matches"`
}

// isRepositoryControlledBoundaryMethod reports whether fixture regeneration for
// the method uses the active rotki or composite-oracle boundary.
// Authored by: OpenCode
func isRepositoryControlledBoundaryMethod(method reportmodel.CostBasisMethod) bool {
	switch method {
	case reportmodel.CostBasisMethodFIFO,
		reportmodel.CostBasisMethodLIFO,
		reportmodel.CostBasisMethodHIFO,
		reportmodel.CostBasisMethodAverageCost,
		reportmodel.CostBasisMethodScopeLocalHybrid:
		return true
	default:
		return false
	}
}

// isRotkiPureMethod reports whether one method maps directly to a rotki
// cost-basis method.
// Authored by: OpenCode
func isRotkiPureMethod(method reportmodel.CostBasisMethod) bool {
	switch method {
	case reportmodel.CostBasisMethodFIFO,
		reportmodel.CostBasisMethodLIFO,
		reportmodel.CostBasisMethodHIFO,
		reportmodel.CostBasisMethodAverageCost:
		return true
	default:
		return false
	}
}

// buildRotkiOracleOutputForAsset generates one pure-method oracle fixture from
// deterministic adapter input plus verified pinned rotki source execution.
// Authored by: OpenCode
func buildRotkiOracleOutputForAsset(
	ctx context.Context,
	runtime rotkiSourceRuntime,
	repositoryRoot string,
	dataset fixture.EmpiricalDataset,
	datasetInputHash string,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	assetIdentityKey string,
) (fixture.OracleOutput, error) {
	var _, inputRelativePath, rawInput, err = buildRotkiOracleInput(empiricalCase, method, assetIdentityKey, selectRotkiOracleActivities(dataset, empiricalCase, assetIdentityKey), false)
	if err != nil {
		return fixture.OracleOutput{}, err
	}
	if _, err = writeArtifact(repositoryRoot, inputRelativePath, rawInput, true); err != nil {
		return fixture.OracleOutput{}, fmt.Errorf("write generated rotki adapter input %s: %w", inputRelativePath, err)
	}

	var capture rotkiOracleCapture
	var verifiedSource verifiedRotkiSource
	capture, _, verifiedSource, err = runtime.captureOracleOutput(ctx, inputRelativePath, rotkiMethodForCase(method))
	if err != nil {
		return fixture.OracleOutput{}, err
	}
	capture, err = roundRotkiOracleCapture(capture)
	if err != nil {
		return fixture.OracleOutput{}, fmt.Errorf("round rotki oracle capture for case %s method %s asset %s: %w", empiricalCase.CaseID, method, assetIdentityKey, err)
	}

	var matches = capture.Matches
	var adapterConstraints = []string{
		"Verified pinned rotki source archive execution from an untracked project-local cache",
		"Zero-priced holding reductions are excluded from external-oracle fixture generation",
	}
	var unsupportedSegments []unsupportedOracleSegmentInput
	if method == reportmodel.CostBasisMethodAverageCost {
		adapterConstraints = []string{
			"Verified pinned rotki source archive execution from an untracked project-local cache",
			"Average-cost comparison is limited to aggregate values until project-compatible pool provenance exists",
		}
		matches = nil
		unsupportedSegments = buildAverageCostUnsupportedSegments(dataset, empiricalCase, assetIdentityKey)
	}

	return normalizeOracleOutput(oracleOutputNormalizationInput{
		DatasetVersion:      strings.TrimSpace(dataset.DatasetVersion),
		CaseID:              strings.TrimSpace(empiricalCase.CaseID),
		Method:              method,
		Year:                empiricalCase.Year,
		AssetIdentityKey:    strings.TrimSpace(assetIdentityKey),
		Values:              capture.Values,
		Matches:             matches,
		UnsupportedSegments: unsupportedSegments,
		Metadata: oracleGenerationMetadataInput{
			OracleName:              defaultRotkiPureOracleName,
			SourceURL:               verifiedSource.SourceURL,
			SourceChecksum:          verifiedSource.SourceChecksum,
			VersionOrCommit:         verifiedSource.VersionOrCommit,
			AdapterArguments:        buildRotkiAdapterArguments(inputRelativePath, method, verifiedSource.SourceRootRelativePath),
			AdapterConstraints:      adapterConstraints,
			DatasetInputHash:        strings.TrimSpace(datasetInputHash),
			ExternalOracleInputHash: stablePrefixedSHA256Hash(rawInput),
			DecimalPolicy:           oracleDecimalPolicy,
			FinancialTolerances:     map[string]string{"realized_gain_or_loss": "0", "allocated_basis": "0", "closing_basis": "0"},
			ToleranceNotes:          map[string]string{},
		},
	})
}

// buildRotkiCompositeOracleOutputForAsset generates one scope-local hybrid
// composite fixture from rotki ACB arithmetic plus project-owned scope semantics.
// Authored by: OpenCode
func buildRotkiCompositeOracleOutputForAsset(
	ctx context.Context,
	runtime rotkiSourceRuntime,
	repositoryRoot string,
	dataset fixture.EmpiricalDataset,
	datasetInputHash string,
	empiricalCase fixture.EmpiricalCase,
	assetIdentityKey string,
) (fixture.OracleOutput, error) {
	var activities = selectRotkiOracleActivities(dataset, empiricalCase, assetIdentityKey)
	var _, inputRelativePath, rawInput, err = buildRotkiOracleInput(empiricalCase, reportmodel.CostBasisMethodScopeLocalHybrid, assetIdentityKey, activities, true)
	if err != nil {
		return fixture.OracleOutput{}, err
	}
	if _, err = writeArtifact(repositoryRoot, inputRelativePath, rawInput, true); err != nil {
		return fixture.OracleOutput{}, fmt.Errorf("write generated scope-local hybrid adapter input %s: %w", inputRelativePath, err)
	}

	var capture rotkiOracleCapture
	var verifiedSource verifiedRotkiSource
	capture, _, verifiedSource, err = runtime.captureOracleOutput(ctx, inputRelativePath, reportMethod("average_cost"))
	if err != nil {
		return fixture.OracleOutput{}, err
	}
	capture, err = roundRotkiOracleCapture(capture)
	if err != nil {
		return fixture.OracleOutput{}, fmt.Errorf("round rotki composite capture for case %s asset %s: %w", empiricalCase.CaseID, assetIdentityKey, err)
	}

	return normalizeOracleOutput(oracleOutputNormalizationInput{
		DatasetVersion:      strings.TrimSpace(dataset.DatasetVersion),
		CaseID:              strings.TrimSpace(empiricalCase.CaseID),
		Method:              reportmodel.CostBasisMethodScopeLocalHybrid,
		Year:                empiricalCase.Year,
		AssetIdentityKey:    strings.TrimSpace(assetIdentityKey),
		Values:              capture.Values,
		Matches:             capture.Matches,
		UnsupportedSegments: buildUnsupportedSegments(dataset, empiricalCase, reportmodel.CostBasisMethodScopeLocalHybrid, assetIdentityKey, nil, capture.Matches),
		Metadata: oracleGenerationMetadataInput{
			OracleName:              defaultRotkiHybridCompositeOracleName,
			SourceURL:               verifiedSource.SourceURL,
			SourceChecksum:          verifiedSource.SourceChecksum,
			VersionOrCommit:         verifiedSource.VersionOrCommit,
			AdapterArguments:        buildRotkiAdapterArguments(inputRelativePath, reportmodel.CostBasisMethodScopeLocalHybrid, verifiedSource.SourceRootRelativePath),
			AdapterConstraints:      []string{"Verified pinned rotki source archive execution from an untracked project-local cache", "Scope-local routing and lifecycle assertions remain documented project-owned composition rules"},
			DatasetInputHash:        strings.TrimSpace(datasetInputHash),
			ExternalOracleInputHash: stablePrefixedSHA256Hash(rawInput),
			DecimalPolicy:           oracleDecimalPolicy,
			CompositeRuleVersion:    defaultRotkiCompositeRuleVersion,
			FinancialTolerances:     map[string]string{"realized_gain_or_loss": "0", "allocated_basis": "0", "closing_basis": "0"},
			ToleranceNotes:          map[string]string{},
		},
	})
}

// buildRotkiOracleInput renders one deterministic untracked adapter input.
// Authored by: OpenCode
func buildRotkiOracleInput(
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	assetIdentityKey string,
	activities []fixture.EmpiricalActivity,
	composite bool,
) (rotkiOracleInput, string, []byte, error) {
	var activityInputs = make([]rotkiOracleActivityInput, 0, len(activities))
	var activityIndex int
	for activityIndex = range activities {
		activityInputs = append(activityInputs, rotkiOracleActivityInput{
			SourceID:           strings.TrimSpace(activities[activityIndex].SourceID),
			OccurredAt:         strings.TrimSpace(activities[activityIndex].OccurredAt),
			DeterministicOrder: activities[activityIndex].DeterministicOrder,
			ActivityType:       string(activities[activityIndex].ActivityType),
			AssetIdentityKey:   strings.TrimSpace(activities[activityIndex].AssetIdentityKey),
			AssetSymbol:        strings.TrimSpace(activities[activityIndex].AssetSymbol),
			Quantity:           strings.TrimSpace(activities[activityIndex].Quantity),
			GrossValue:         strings.TrimSpace(activities[activityIndex].GrossValue),
			FeeAmount:          strings.TrimSpace(activities[activityIndex].FeeAmount),
			SourceScope:        activities[activityIndex].SourceScope,
		})
	}

	var input = rotkiOracleInput{
		CaseID:                      strings.TrimSpace(empiricalCase.CaseID),
		Method:                      method,
		RotkiMethod:                 string(rotkiMethodForCase(method)),
		Year:                        empiricalCase.Year,
		AssetIdentityKey:            strings.TrimSpace(assetIdentityKey),
		ComparisonActivitySourceIDs: comparisonSourceIDsForCaseActivities(empiricalCase, assetIdentityKey, activities),
		Activities:                  activityInputs,
	}
	if composite {
		input.RotkiMethod = "average_cost"
	}

	var rawInput, err = json.MarshalIndent(input, "", "  ")
	if err != nil {
		return rotkiOracleInput{}, "", nil, fmt.Errorf("marshal generated rotki adapter input: %w", err)
	}

	return input, generatedRotkiOracleInputRelativePath(method, empiricalCase, assetIdentityKey), append(rawInput, '\n'), nil
}

// selectRotkiOracleActivities chooses the deterministic activity history that
// feeds one generated rotki adapter input.
// Authored by: OpenCode
func selectRotkiOracleActivities(dataset fixture.EmpiricalDataset, empiricalCase fixture.EmpiricalCase, assetIdentityKey string) []fixture.EmpiricalActivity {
	var latestSelectedYearActivity *fixture.EmpiricalActivity
	var activitiesBySourceID = datasetActivitiesBySourceID(dataset)
	var sourceIndex int
	for sourceIndex = range empiricalCase.ActivitySourceIDs {
		var activity, found = activitiesBySourceID[strings.TrimSpace(empiricalCase.ActivitySourceIDs[sourceIndex])]
		if !found {
			continue
		}
		if strings.TrimSpace(activity.AssetIdentityKey) != strings.TrimSpace(assetIdentityKey) {
			continue
		}
		if activityYear(activity) != empiricalCase.Year {
			continue
		}
		if latestSelectedYearActivity == nil || compareOracleInputActivities(activity, *latestSelectedYearActivity) > 0 {
			var copiedActivity = activity
			latestSelectedYearActivity = &copiedActivity
		}
	}

	var selected = make([]fixture.EmpiricalActivity, 0)
	var activityIndex int
	for activityIndex = range dataset.Activities {
		var activity = dataset.Activities[activityIndex]
		if strings.TrimSpace(activity.AssetIdentityKey) != strings.TrimSpace(assetIdentityKey) {
			continue
		}
		var year = activityYear(activity)
		if year < empiricalCase.Year {
			selected = append(selected, activity)
			continue
		}
		if year > empiricalCase.Year || latestSelectedYearActivity == nil {
			continue
		}
		if compareOracleInputActivities(activity, *latestSelectedYearActivity) <= 0 {
			selected = append(selected, activity)
		}
	}

	return selected
}

// comparisonSourceIDsForCaseActivities returns the selected-year comparison slice
// passed into the generated adapter input.
// Authored by: OpenCode
func comparisonSourceIDsForCaseActivities(empiricalCase fixture.EmpiricalCase, assetIdentityKey string, activities []fixture.EmpiricalActivity) []string {
	var selectedSourceIDs = make(map[string]struct{}, len(activities))
	var activityIndex int
	for activityIndex = range activities {
		selectedSourceIDs[strings.TrimSpace(activities[activityIndex].SourceID)] = struct{}{}
	}

	var comparisonSourceIDs = make([]string, 0)
	var sourceIndex int
	for sourceIndex = range empiricalCase.ActivitySourceIDs {
		var sourceID = strings.TrimSpace(empiricalCase.ActivitySourceIDs[sourceIndex])
		if sourceID == "" {
			continue
		}
		if _, found := selectedSourceIDs[sourceID]; !found {
			continue
		}
		comparisonSourceIDs = append(comparisonSourceIDs, sourceID)
	}

	return comparisonSourceIDs
}

// generatedRotkiOracleInputRelativePath returns one repository-relative cache
// path for the generated adapter input.
// Authored by: OpenCode
func generatedRotkiOracleInputRelativePath(empiricalCaseMethod reportmodel.CostBasisMethod, empiricalCase fixture.EmpiricalCase, assetIdentityKey string) string {
	var baseName = strings.TrimSpace(empiricalCase.CaseID)
	if len(empiricalCase.AssetIdentityKeys) > 1 {
		baseName += "--" + strings.TrimSpace(assetIdentityKey)
	}

	return path.Join(rotkiOracleInputRootRepositoryPath, empiricalCaseMethod.FilenameSlug(), baseName+".json")
}

// buildRotkiAdapterArguments records the deterministic adapter arguments used by
// one fixture regeneration run.
// Authored by: OpenCode
func buildRotkiAdapterArguments(inputRelativePath string, method reportmodel.CostBasisMethod, sourceRootRelativePath string) []string {
	var rotkiMethod = string(rotkiMethodForCase(method))
	if method == reportmodel.CostBasisMethodScopeLocalHybrid {
		rotkiMethod = "average_cost"
	}

	return []string{
		"--source-root",
		sourceRootRelativePath,
		"--input",
		inputRelativePath,
		"--rotki-method",
		rotkiMethod,
		"--method",
		string(method),
	}
}

// rotkiMethodForCase maps one project method to the adapter's direct rotki
// method selector.
// Authored by: OpenCode
func rotkiMethodForCase(method reportmodel.CostBasisMethod) reportMethod {
	switch method {
	case reportmodel.CostBasisMethodFIFO:
		return reportMethod("fifo")
	case reportmodel.CostBasisMethodLIFO:
		return reportMethod("lifo")
	case reportmodel.CostBasisMethodHIFO:
		return reportMethod("hifo")
	default:
		return reportMethod("average_cost")
	}
}

// buildAverageCostUnsupportedSegments records the aggregate-only unsupported
// pool provenance for average-cost fixtures.
// Authored by: OpenCode
func buildAverageCostUnsupportedSegments(
	dataset fixture.EmpiricalDataset,
	empiricalCase fixture.EmpiricalCase,
	assetIdentityKey string,
) []unsupportedOracleSegmentInput {
	var relevantSellSourceIDs = make([]string, 0)
	var activitiesBySourceID = datasetActivitiesBySourceID(dataset)
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
		if activityYear(activity) != empiricalCase.Year {
			continue
		}
		if strings.TrimSpace(string(activity.ActivityType)) != "SELL" {
			continue
		}
		relevantSellSourceIDs = append(relevantSellSourceIDs, sourceID)
	}

	if len(relevantSellSourceIDs) == 0 {
		return nil
	}

	return []unsupportedOracleSegmentInput{{
		CaseID:            strings.TrimSpace(empiricalCase.CaseID),
		Method:            reportmodel.CostBasisMethodAverageCost,
		ActivitySourceIDs: relevantSellSourceIDs,
		Reason:            "Average-cost pool provenance remains outside the verified rotki aggregate oracle boundary",
		ComparisonPolicy:  fixture.ComparisonPolicySkipExternalOracle,
	}}
}

// activityYear parses one empirical activity year from its persisted timestamp.
// Authored by: OpenCode
func activityYear(activity fixture.EmpiricalActivity) int {
	var occurredAt = strings.TrimSpace(activity.OccurredAt)
	if len(occurredAt) < 4 {
		return 0
	}

	var year int
	_, _ = fmt.Sscanf(occurredAt[:4], "%d", &year)
	return year
}

// roundRotkiOracleCapture aligns raw adapter decimals to the repository report
// decimal policy before fixture normalization persists them.
// Authored by: OpenCode
func roundRotkiOracleCapture(capture rotkiOracleCapture) (rotkiOracleCapture, error) {
	var err error
	capture.Values.RealizedGainOrLoss, err = roundRotkiDecimalString(capture.Values.RealizedGainOrLoss)
	if err != nil {
		return rotkiOracleCapture{}, fmt.Errorf("round realized_gain_or_loss: %w", err)
	}
	capture.Values.AllocatedBasis, err = roundRotkiDecimalString(capture.Values.AllocatedBasis)
	if err != nil {
		return rotkiOracleCapture{}, fmt.Errorf("round allocated_basis: %w", err)
	}
	capture.Values.ClosingQuantity, err = roundRotkiDecimalString(capture.Values.ClosingQuantity)
	if err != nil {
		return rotkiOracleCapture{}, fmt.Errorf("round closing_quantity: %w", err)
	}
	capture.Values.ClosingBasis, err = roundRotkiDecimalString(capture.Values.ClosingBasis)
	if err != nil {
		return rotkiOracleCapture{}, fmt.Errorf("round closing_basis: %w", err)
	}

	var matchIndex int
	for matchIndex = range capture.Matches {
		capture.Matches[matchIndex].MatchedQuantity, err = roundRotkiDecimalString(capture.Matches[matchIndex].MatchedQuantity)
		if err != nil {
			return rotkiOracleCapture{}, fmt.Errorf("round match %d matched_quantity: %w", matchIndex, err)
		}
		capture.Matches[matchIndex].MatchedBasis, err = roundRotkiDecimalString(capture.Matches[matchIndex].MatchedBasis)
		if err != nil {
			return rotkiOracleCapture{}, fmt.Errorf("round match %d matched_basis: %w", matchIndex, err)
		}
		capture.Matches[matchIndex].MatchedProceeds, err = roundRotkiDecimalString(capture.Matches[matchIndex].MatchedProceeds)
		if err != nil {
			return rotkiOracleCapture{}, fmt.Errorf("round match %d matched_proceeds: %w", matchIndex, err)
		}
		capture.Matches[matchIndex].MatchedGainOrLoss, err = roundRotkiDecimalString(capture.Matches[matchIndex].MatchedGainOrLoss)
		if err != nil {
			return rotkiOracleCapture{}, fmt.Errorf("round match %d matched_gain_or_loss: %w", matchIndex, err)
		}
	}

	return capture, nil
}

// roundRotkiDecimalString rounds one rotki decimal to the production 16-decimal
// half-up policy and returns its canonical persisted representation.
// Authored by: OpenCode
func roundRotkiDecimalString(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}

	var value apd.Decimal
	if _, _, err := value.SetString(strings.TrimSpace(raw)); err != nil {
		return "", fmt.Errorf("parse decimal value %q: %w", raw, err)
	}

	var rounded apd.Decimal
	var context = apd.BaseContext.WithPrecision(200)
	context.Rounding = apd.RoundHalfUp
	if _, err := context.Quantize(&rounded, &value, -16); err != nil {
		return "", fmt.Errorf("quantize decimal value %q to scale 16: %w", raw, err)
	}

	var canonical, err = fixture.CanonicalDecimalString(rounded)
	if err != nil {
		return "", fmt.Errorf("canonicalize rounded decimal value %q: %w", raw, err)
	}

	return canonical, nil
}
