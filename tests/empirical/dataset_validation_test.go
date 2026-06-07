package empirical

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

const (
	minimumEmpiricalActivityCount = 150
	minimumEmpiricalYearSpan      = 3

	empiricalDatasetRepositoryPath = "testdata/empirical/financial-dataset.yaml"
)

// empiricalDatasetValidationHooks defines the future fixture helper entry
// points that this contract test expects once dataset parsing and validation are
// implemented.
// Authored by: OpenCode
var empiricalDatasetValidationHooks = datasetValidationHooks{
	load: fixture.LoadEmpiricalDataset,
	validate: func(path string, rawContent string, dataset fixture.EmpiricalDataset) error {
		return fixture.ValidateEmpiricalDataset(path, rawContent, dataset)
	},
}

// datasetValidationHooks stores the load and validate hooks used by this test
// file.
//
// The zero value is intentional while `tests/empirical/fixture` is still
// missing the parser and validator implementation promised by the active spec.
// Authored by: OpenCode
type datasetValidationHooks struct {
	load     func(path string) (fixture.EmpiricalDataset, string, error)
	validate func(path string, rawContent string, dataset fixture.EmpiricalDataset) error
}

// missingDatasetValidationImplementationError reports that the future fixture
// helper wiring is still absent from this contract test file.
// Authored by: OpenCode
type missingDatasetValidationImplementationError struct {
	Hint      string
	Operation string
	Path      string
}

// structuralDatasetValidationTestCase stores one failure-focused structural
// validation contract case.
// Authored by: OpenCode
type structuralDatasetValidationTestCase struct {
	mutate func(dataset *fixture.EmpiricalDataset)
	name   string
	path   string
	want   func(dataset fixture.EmpiricalDataset) []string
}

// TestEmpiricalDatasetValidation captures the dataset validation contract for
// inline synthetic fixtures now and for the repository-backed dataset loading
// path later.
//
// The structural subtests intentionally fail with actionable missing-
// implementation errors until the dataset parser and validator helpers are wired
// into `empiricalDatasetValidationHooks`.
// Authored by: OpenCode
func TestEmpiricalDatasetValidation(t *testing.T) {
	t.Parallel()

	var structuralCases = []structuralDatasetValidationTestCase{
		{
			name: "rejects_activity_count_below_minimum",
			path: inlineDatasetValidationPath("activity-count-below-minimum.yaml"),
			mutate: func(dataset *fixture.EmpiricalDataset) {
				dataset.Activities = dataset.Activities[:minimumEmpiricalActivityCount-1]
			},
			want: func(dataset fixture.EmpiricalDataset) []string {
				return []string{"activities", "activity_count", fmt.Sprintf("%d", minimumEmpiricalActivityCount)}
			},
		},
		{
			name: "rejects_year_span_below_minimum",
			path: inlineDatasetValidationPath("year-span-below-minimum.yaml"),
			mutate: func(dataset *fixture.EmpiricalDataset) {
				var index int
				for index = range dataset.Activities {
					var activity = &dataset.Activities[index]
					activity.OccurredAt = fmt.Sprintf("2024-01-%02dT09:00:00Z", (index%28)+1)
					activity.DeterministicOrder = (index % 3) + 1
				}
				dataset.SupportedYears = []int{2024}
			},
			want: func(dataset fixture.EmpiricalDataset) []string {
				return []string{"supported_years", "year_span", fmt.Sprintf("%d", minimumEmpiricalYearSpan)}
			},
		},
		{
			name: "rejects_missing_supported_methods",
			path: inlineDatasetValidationPath("missing-supported-methods.yaml"),
			mutate: func(dataset *fixture.EmpiricalDataset) {
				dataset.SupportedMethods = removeCostBasisMethod(dataset.SupportedMethods, reportmodel.CostBasisMethodScopeLocalHybrid)
				dataset.Cases = removeCasesForMethod(dataset.Cases, reportmodel.CostBasisMethodScopeLocalHybrid)
			},
			want: func(dataset fixture.EmpiricalDataset) []string {
				return []string{"supported_methods", "supported_methods", string(reportmodel.CostBasisMethodScopeLocalHybrid)}
			},
		},
		{
			name: "rejects_duplicate_deterministic_source_ids",
			path: inlineDatasetValidationPath("duplicate-source-ids.yaml"),
			mutate: func(dataset *fixture.EmpiricalDataset) {
				dataset.Activities[1].SourceID = dataset.Activities[0].SourceID
			},
			want: func(dataset fixture.EmpiricalDataset) []string {
				return []string{dataset.Activities[0].SourceID, "source_id", "deterministic_source_id"}
			},
		},
		{
			name: "rejects_missing_or_invalid_ordering_metadata",
			path: inlineDatasetValidationPath("invalid-ordering-metadata.yaml"),
			mutate: func(dataset *fixture.EmpiricalDataset) {
				dataset.Activities[0].DeterministicOrder = 0
			},
			want: func(dataset fixture.EmpiricalDataset) []string {
				return []string{dataset.Activities[0].SourceID, "deterministic_order", "ordering_metadata"}
			},
		},
		{
			name: "rejects_mixed_priced_row_currencies",
			path: inlineDatasetValidationPath("mixed-currencies.yaml"),
			mutate: func(dataset *fixture.EmpiricalDataset) {
				dataset.Activities[0].Currency = "EUR"
			},
			want: func(dataset fixture.EmpiricalDataset) []string {
				return []string{dataset.Activities[0].SourceID, "currency", "single_currency"}
			},
		},
	}

	var testCase structuralDatasetValidationTestCase
	for _, testCase = range structuralCases {
		var testCase = testCase

		t.Run(testCase.name, func(t *testing.T) {
			var dataset = newValidSyntheticEmpiricalDataset()
			testCase.mutate(&dataset)

			var rawContent = renderSyntheticDatasetContent(dataset)
			requireDatasetValidationFailure(t, empiricalDatasetValidationHooks, testCase.path, rawContent, dataset, testCase.want(dataset)...)
		})
	}

	t.Run("rejects_non_synthetic_content", func(t *testing.T) {
		var dataset = newValidSyntheticEmpiricalDataset()
		var datasetPath = inlineDatasetValidationPath("non-synthetic-content.yaml")
		var rawContent = renderSyntheticDatasetContent(
			dataset,
			`owner_name: "John Doe"`,
			`authorization: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJlbXBpcmljYWwtdXNlci0wMDEiLCJuYW1lIjoiSm9obiBEb2UifQ.signaturevalue123"`,
		)

		var err = fixture.ValidateSyntheticOnlyContent(datasetPath, rawContent)
		if err == nil {
			t.Fatal("expected synthetic-only content validation error, got nil")
		}

		assertErrorContainsAll(t, err, datasetPath, "field owner_name", "real_name_like_value", "field authorization", "bearer_token", "jwt_like_value")

		var message = err.Error()
		if strings.Contains(message, "John Doe") {
			t.Fatalf("expected non-secret validation message, got %q", message)
		}
		if strings.Contains(message, "Bearer eyJ") {
			t.Fatalf("expected non-secret validation message, got %q", message)
		}
	})

	t.Run("loads_and_validates_repository_dataset", func(t *testing.T) {
		requireRepositoryDatasetValidationSuccess(t, empiricalDatasetValidationHooks)
	})
}

// Error formats the actionable missing-implementation message.
// Authored by: OpenCode
func (implementationErr missingDatasetValidationImplementationError) Error() string {
	return fmt.Sprintf(
		"dataset validation %s helper is not wired for %s: %s",
		implementationErr.Operation,
		implementationErr.Path,
		implementationErr.Hint,
	)
}

// loadDataset loads one empirical dataset through the local hook.
// Authored by: OpenCode
func (hooks datasetValidationHooks) loadDataset(path string) (fixture.EmpiricalDataset, string, error) {
	if hooks.load == nil {
		return fixture.EmpiricalDataset{}, "", missingDatasetValidationImplementationError{
			Hint:      "assign empiricalDatasetValidationHooks.load to the future fixture dataset loader",
			Operation: "load",
			Path:      path,
		}
	}

	return hooks.load(path)
}

// validateDataset validates one already-loaded empirical dataset through the
// local hook.
// Authored by: OpenCode
func (hooks datasetValidationHooks) validateDataset(path string, rawContent string, dataset fixture.EmpiricalDataset) error {
	if hooks.validate == nil {
		return missingDatasetValidationImplementationError{
			Hint:      "assign empiricalDatasetValidationHooks.validate to the future fixture dataset validator",
			Operation: "validate",
			Path:      path,
		}
	}

	return hooks.validate(path, rawContent, dataset)
}

// newValidSyntheticEmpiricalDataset builds one fully synthetic dataset that
// satisfies the current structural contract targeted by this test file.
// Authored by: OpenCode
func newValidSyntheticEmpiricalDataset() fixture.EmpiricalDataset {
	var dataset = fixture.EmpiricalDataset{
		DatasetVersion:   "1",
		Description:      "Synthetic empirical financial validation dataset",
		Currency:         "USD",
		SupportedYears:   []int{2023, 2024, 2025},
		SupportedMethods: reportmodel.SupportedCostBasisMethods(),
		CoverageTags: []string{
			"fifo",
			"lifo",
			"hifo",
			"average_cost",
			"scope_local_hybrid",
			"multi_year_opening_history",
			"same_source_calendar_date_ordering",
			"single_currency",
			"zero_priced_reduction",
		},
	}

	var activityPattern = []syncmodel.ActivityType{
		syncmodel.ActivityTypeBuy,
		syncmodel.ActivityTypeBuy,
		syncmodel.ActivityTypeSell,
		syncmodel.ActivityTypeBuy,
		syncmodel.ActivityTypeSell,
		syncmodel.ActivityTypeBuy,
		syncmodel.ActivityTypeSell,
		syncmodel.ActivityTypeBuy,
		syncmodel.ActivityTypeSell,
		syncmodel.ActivityTypeBuy,
	}
	var assetIdentityKeys = []string{"asset-alpha", "asset-beta", "asset-gamma", "asset-delta", "asset-epsilon"}
	var assetSymbols = []string{"ALPHA", "BETA", "GAMMA", "DELTA", "EPS"}
	var years = []int{2023, 2024, 2025}
	var sourceCounter = 1
	var yearIndex int

	dataset.Activities = make([]fixture.EmpiricalActivity, 0, minimumEmpiricalActivityCount)

	for yearIndex = range years {
		var assetIndex int
		for assetIndex = range assetIdentityKeys {
			var sequenceIndex int
			for sequenceIndex = range activityPattern {
				var sequenceGroup = sequenceIndex / 3
				var unitPrice = 10 + (yearIndex * 10) + (assetIndex * 3) + sequenceIndex
				var activity = fixture.EmpiricalActivity{
					SourceID:           fmt.Sprintf("emp-act-%06d", sourceCounter),
					OccurredAt:         fmt.Sprintf("%04d-%02d-%02dT09:00:00Z", years[yearIndex], ((assetIndex*2)+sequenceGroup+yearIndex)%12+1, sequenceGroup+1),
					DeterministicOrder: (sequenceIndex % 3) + 1,
					ActivityType:       activityPattern[sequenceIndex],
					AssetIdentityKey:   assetIdentityKeys[assetIndex],
					AssetSymbol:        assetSymbols[assetIndex],
					Quantity:           "1",
					GrossValue:         fmt.Sprintf("%d", unitPrice),
					UnitPrice:          fmt.Sprintf("%d", unitPrice),
					FeeAmount:          "0",
					Currency:           dataset.Currency,
					SourceScope:        newSyntheticEmpiricalScope(assetIndex),
					CoverageTags: []string{
						string(dataset.SupportedMethods[assetIndex%len(dataset.SupportedMethods)]),
						"synthetic_dataset",
					},
				}

				dataset.Activities = append(dataset.Activities, activity)
				sourceCounter++
			}
		}
	}

	markSyntheticZeroPricedReduction(&dataset)

	dataset.Cases = []fixture.EmpiricalCase{
		{
			CaseID:            "case-fifo-basic-2024",
			Description:       "Synthetic FIFO validation slice",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodFIFO},
			Year:              2024,
			AssetIdentityKeys: []string{"asset-alpha"},
			ActivitySourceIDs: selectSyntheticActivitySourceIDs(dataset, 2024, "asset-alpha", 5),
			CoverageTags:      []string{"fifo"},
			OracleSupport:     fixture.OracleSupportSupported,
		},
		{
			CaseID:            "case-lifo-basic-2024",
			Description:       "Synthetic LIFO validation slice",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodLIFO},
			Year:              2024,
			AssetIdentityKeys: []string{"asset-beta"},
			ActivitySourceIDs: selectSyntheticActivitySourceIDs(dataset, 2024, "asset-beta", 5),
			CoverageTags:      []string{"lifo"},
			OracleSupport:     fixture.OracleSupportSupported,
		},
		{
			CaseID:            "case-hifo-basic-2024",
			Description:       "Synthetic HIFO validation slice",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodHIFO},
			Year:              2024,
			AssetIdentityKeys: []string{"asset-gamma"},
			ActivitySourceIDs: selectSyntheticActivitySourceIDs(dataset, 2024, "asset-gamma", 5),
			CoverageTags:      []string{"hifo"},
			OracleSupport:     fixture.OracleSupportSupported,
		},
		{
			CaseID:            "case-average-cost-basic-2024",
			Description:       "Synthetic average cost validation slice",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodAverageCost},
			Year:              2024,
			AssetIdentityKeys: []string{"asset-delta"},
			ActivitySourceIDs: selectSyntheticActivitySourceIDs(dataset, 2024, "asset-delta", 5),
			CoverageTags:      []string{"average_cost"},
			OracleSupport:     fixture.OracleSupportSupported,
		},
		{
			CaseID:            "case-scope-local-hybrid-basic-2024",
			Description:       "Synthetic scope-local hybrid validation slice",
			Methods:           []reportmodel.CostBasisMethod{reportmodel.CostBasisMethodScopeLocalHybrid},
			Year:              2024,
			AssetIdentityKeys: []string{"asset-epsilon"},
			ActivitySourceIDs: selectSyntheticActivitySourceIDs(dataset, 2024, "asset-epsilon", 5),
			CoverageTags:      []string{"scope_local_hybrid"},
			OracleSupport:     fixture.OracleSupportSupported,
		},
	}

	return dataset
}

// newSyntheticEmpiricalScope returns one clearly synthetic scope profile for a
// generated activity row.
// Authored by: OpenCode
func newSyntheticEmpiricalScope(assetIndex int) *fixture.EmpiricalScope {
	switch assetIndex {
	case 0:
		return &fixture.EmpiricalScope{
			ScopeID:     "wallet-alpha",
			ScopeKind:   syncmodel.SourceScopeKindWallet,
			Reliability: syncmodel.ScopeReliabilityReliable,
			DisplayName: "Synthetic Wallet Alpha",
		}
	case 1:
		return &fixture.EmpiricalScope{
			ScopeID:     "account-beta",
			ScopeKind:   syncmodel.SourceScopeKindAccount,
			Reliability: syncmodel.ScopeReliabilityReliable,
			DisplayName: "Synthetic Account Beta",
		}
	case 2:
		return &fixture.EmpiricalScope{
			ScopeID:     "wallet-gamma",
			ScopeKind:   syncmodel.SourceScopeKindWallet,
			Reliability: syncmodel.ScopeReliabilityPartial,
			DisplayName: "Synthetic Wallet Gamma",
		}
	case 3:
		return &fixture.EmpiricalScope{
			Reliability: syncmodel.ScopeReliabilityUnavailable,
		}
	default:
		return &fixture.EmpiricalScope{
			ScopeID:     "wallet-epsilon",
			ScopeKind:   syncmodel.SourceScopeKindWallet,
			Reliability: syncmodel.ScopeReliabilityReliable,
			DisplayName: "Synthetic Wallet Epsilon",
		}
	}
}

// markSyntheticZeroPricedReduction rewrites one generated row into a zero-priced
// holding reduction with the required explanation while keeping the dataset
// otherwise synthetic and structurally valid.
// Authored by: OpenCode
func markSyntheticZeroPricedReduction(dataset *fixture.EmpiricalDataset) {
	var lastIndex = len(dataset.Activities) - 1
	if lastIndex < 0 {
		return
	}

	var activity = &dataset.Activities[lastIndex]
	activity.ActivityType = syncmodel.ActivityTypeSell
	activity.GrossValue = ""
	activity.UnitPrice = ""
	activity.FeeAmount = ""
	activity.Currency = ""
	activity.ZeroPricedReductionExplanation = "Synthetic basis-only reduction without proceeds"
	activity.CoverageTags = append(activity.CoverageTags, "zero_priced_reduction")
}

// selectSyntheticActivitySourceIDs returns up to `limit` source IDs for one
// asset and source year from the generated synthetic dataset.
// Authored by: OpenCode
func selectSyntheticActivitySourceIDs(dataset fixture.EmpiricalDataset, year int, assetIdentityKey string, limit int) []string {
	var sourceIDs = make([]string, 0, limit)
	var activity fixture.EmpiricalActivity

	for _, activity = range dataset.Activities {
		if !strings.HasPrefix(activity.OccurredAt, fmt.Sprintf("%04d-", year)) {
			continue
		}
		if activity.AssetIdentityKey != assetIdentityKey {
			continue
		}

		sourceIDs = append(sourceIDs, activity.SourceID)
		if len(sourceIDs) == limit {
			break
		}
	}

	return sourceIDs
}

// renderSyntheticDatasetContent renders one YAML-like synthetic dataset text for
// raw-content validation checks.
// Authored by: OpenCode
func renderSyntheticDatasetContent(dataset fixture.EmpiricalDataset, extraLines ...string) string {
	var builder strings.Builder
	var year int
	var method reportmodel.CostBasisMethod
	var tag string
	var activity fixture.EmpiricalActivity
	var caseRecord fixture.EmpiricalCase

	builder.WriteString(fmt.Sprintf("dataset_version: %q\n", dataset.DatasetVersion))
	builder.WriteString(fmt.Sprintf("description: %q\n", dataset.Description))
	builder.WriteString(fmt.Sprintf("currency: %q\n", dataset.Currency))
	builder.WriteString("supported_years:\n")
	for _, year = range dataset.SupportedYears {
		builder.WriteString(fmt.Sprintf("  - %d\n", year))
	}
	builder.WriteString("supported_methods:\n")
	for _, method = range dataset.SupportedMethods {
		builder.WriteString(fmt.Sprintf("  - %s\n", method))
	}
	builder.WriteString("coverage_tags:\n")
	for _, tag = range dataset.CoverageTags {
		builder.WriteString(fmt.Sprintf("  - %s\n", tag))
	}
	builder.WriteString("activities:\n")
	for _, activity = range dataset.Activities {
		builder.WriteString(fmt.Sprintf("  - source_id: %s\n", activity.SourceID))
		builder.WriteString(fmt.Sprintf("    occurred_at: %q\n", activity.OccurredAt))
		builder.WriteString(fmt.Sprintf("    deterministic_order: %d\n", activity.DeterministicOrder))
		builder.WriteString(fmt.Sprintf("    activity_type: %s\n", activity.ActivityType))
		builder.WriteString(fmt.Sprintf("    asset_identity_key: %s\n", activity.AssetIdentityKey))
		builder.WriteString(fmt.Sprintf("    asset_symbol: %s\n", activity.AssetSymbol))
		builder.WriteString(fmt.Sprintf("    quantity: %q\n", activity.Quantity))
		if activity.GrossValue != "" {
			builder.WriteString(fmt.Sprintf("    gross_value: %q\n", activity.GrossValue))
		}
		if activity.UnitPrice != "" {
			builder.WriteString(fmt.Sprintf("    unit_price: %q\n", activity.UnitPrice))
		}
		if activity.FeeAmount != "" {
			builder.WriteString(fmt.Sprintf("    fee_amount: %q\n", activity.FeeAmount))
		}
		if activity.Currency != "" {
			builder.WriteString(fmt.Sprintf("    currency: %s\n", activity.Currency))
		}
		if activity.SourceScope != nil {
			builder.WriteString("    source_scope:\n")
			if activity.SourceScope.ScopeID != "" {
				builder.WriteString(fmt.Sprintf("      scope_id: %s\n", activity.SourceScope.ScopeID))
			}
			if activity.SourceScope.ScopeKind != "" {
				builder.WriteString(fmt.Sprintf("      scope_kind: %s\n", activity.SourceScope.ScopeKind))
			}
			builder.WriteString(fmt.Sprintf("      reliability: %s\n", activity.SourceScope.Reliability))
			if activity.SourceScope.DisplayName != "" {
				builder.WriteString(fmt.Sprintf("      display_name: %q\n", activity.SourceScope.DisplayName))
			}
		}
		if activity.ZeroPricedReductionExplanation != "" {
			builder.WriteString(fmt.Sprintf("    zero_priced_reduction_explanation: %q\n", activity.ZeroPricedReductionExplanation))
		}
		builder.WriteString("    coverage_tags:\n")
		for _, tag = range activity.CoverageTags {
			builder.WriteString(fmt.Sprintf("      - %s\n", tag))
		}
	}
	builder.WriteString("cases:\n")
	for _, caseRecord = range dataset.Cases {
		builder.WriteString(fmt.Sprintf("  - case_id: %s\n", caseRecord.CaseID))
		builder.WriteString(fmt.Sprintf("    description: %q\n", caseRecord.Description))
		builder.WriteString("    methods:\n")
		for _, method = range caseRecord.Methods {
			builder.WriteString(fmt.Sprintf("      - %s\n", method))
		}
		builder.WriteString(fmt.Sprintf("    year: %d\n", caseRecord.Year))
		builder.WriteString("    asset_identity_keys:\n")
		for _, tag = range caseRecord.AssetIdentityKeys {
			builder.WriteString(fmt.Sprintf("      - %s\n", tag))
		}
		builder.WriteString("    activity_source_ids:\n")
		for _, tag = range caseRecord.ActivitySourceIDs {
			builder.WriteString(fmt.Sprintf("      - %s\n", tag))
		}
		builder.WriteString("    coverage_tags:\n")
		for _, tag = range caseRecord.CoverageTags {
			builder.WriteString(fmt.Sprintf("      - %s\n", tag))
		}
		builder.WriteString(fmt.Sprintf("    oracle_support: %s\n", caseRecord.OracleSupport))
		if caseRecord.UnsupportedReason != "" {
			builder.WriteString(fmt.Sprintf("    unsupported_reason: %q\n", caseRecord.UnsupportedReason))
		}
	}
	for _, tag = range extraLines {
		builder.WriteString(tag)
		builder.WriteByte('\n')
	}

	return builder.String()
}

// inlineDatasetValidationPath returns a stable contract-only path label for one
// inline synthetic dataset case.
// Authored by: OpenCode
func inlineDatasetValidationPath(name string) string {
	return filepath.ToSlash(filepath.Join("testdata", "empirical", "inline", name))
}

// repositoryDatasetFilesystemPath returns the on-disk path used by the
// repository-backed validation subtest.
// Authored by: OpenCode
func repositoryDatasetFilesystemPath() string {
	return filepath.Clean(filepath.Join("..", "..", filepath.FromSlash(empiricalDatasetRepositoryPath)))
}

// removeCostBasisMethod returns one new slice with the target method removed.
// Authored by: OpenCode
func removeCostBasisMethod(methods []reportmodel.CostBasisMethod, target reportmodel.CostBasisMethod) []reportmodel.CostBasisMethod {
	var filtered = make([]reportmodel.CostBasisMethod, 0, len(methods))
	var method reportmodel.CostBasisMethod

	for _, method = range methods {
		if method == target {
			continue
		}

		filtered = append(filtered, method)
	}

	return filtered
}

// removeCasesForMethod returns one new case slice with any case containing the
// target method removed.
// Authored by: OpenCode
func removeCasesForMethod(cases []fixture.EmpiricalCase, target reportmodel.CostBasisMethod) []fixture.EmpiricalCase {
	var filtered = make([]fixture.EmpiricalCase, 0, len(cases))
	var caseRecord fixture.EmpiricalCase

	for _, caseRecord = range cases {
		var containsTarget bool
		var method reportmodel.CostBasisMethod

		for _, method = range caseRecord.Methods {
			if method == target {
				containsTarget = true
				break
			}
		}

		if containsTarget {
			continue
		}

		filtered = append(filtered, caseRecord)
	}

	return filtered
}

// requireDatasetValidationFailure asserts that structural validation rejects one
// inline synthetic dataset and reports the expected non-secret context.
// Authored by: OpenCode
func requireDatasetValidationFailure(t *testing.T, hooks datasetValidationHooks, path string, rawContent string, dataset fixture.EmpiricalDataset, wantSubstrings ...string) {
	t.Helper()

	var err = hooks.validateDataset(path, rawContent, dataset)
	if err == nil {
		t.Fatal("expected dataset validation error, got nil")
	}

	var implementationErr missingDatasetValidationImplementationError
	if errors.As(err, &implementationErr) {
		t.Fatalf("missing dataset validation implementation: %v", err)
	}

	assertErrorContainsAll(t, err, append([]string{path}, wantSubstrings...)...)
}

// requireRepositoryDatasetValidationSuccess asserts the future repository-backed
// dataset validation path can load and validate the canonical dataset file.
// Authored by: OpenCode
func requireRepositoryDatasetValidationSuccess(t *testing.T, hooks datasetValidationHooks) {
	t.Helper()

	var filesystemPath = repositoryDatasetFilesystemPath()
	if _, err := os.Stat(filesystemPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatalf("repository dataset %s is missing", empiricalDatasetRepositoryPath)
		}

		t.Fatalf("stat repository dataset %s: %v", empiricalDatasetRepositoryPath, err)
	}

	var dataset, rawContent, err = hooks.loadDataset(filesystemPath)
	if err != nil {
		var implementationErr missingDatasetValidationImplementationError
		if errors.As(err, &implementationErr) {
			t.Fatalf("missing dataset loading implementation: %v", err)
		}

		t.Fatalf("load repository dataset %s: %v", empiricalDatasetRepositoryPath, err)
	}

	err = hooks.validateDataset(empiricalDatasetRepositoryPath, rawContent, dataset)
	if err != nil {
		var implementationErr missingDatasetValidationImplementationError
		if errors.As(err, &implementationErr) {
			t.Fatalf("missing dataset validation implementation: %v", err)
		}

		t.Fatalf("validate repository dataset %s: %v", empiricalDatasetRepositoryPath, err)
	}

	err = fixture.ValidateDatasetCoverage(dataset)
	if err != nil {
		t.Fatalf("validate repository dataset coverage %s: %v", empiricalDatasetRepositoryPath, err)
	}
}

// assertErrorContainsAll verifies one error message contains every required
// context fragment.
// Authored by: OpenCode
func assertErrorContainsAll(t *testing.T, err error, wantSubstrings ...string) {
	t.Helper()

	var message = err.Error()
	var want string

	for _, want = range wantSubstrings {
		if !strings.Contains(message, want) {
			t.Fatalf("expected error %q to contain %q", message, want)
		}
	}
}
