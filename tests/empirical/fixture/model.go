package fixture

import (
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// OracleSupport identifies whether one empirical case is fully comparable to the
// selected external-oracle boundary.
// Authored by: OpenCode
type OracleSupport string

const (
	// OracleSupportSupported identifies a case that the external oracle can model
	// completely.
	// Authored by: OpenCode
	OracleSupportSupported OracleSupport = "supported"

	// OracleSupportPartiallySupported identifies a case that the external oracle
	// can model only in part.
	// Authored by: OpenCode
	OracleSupportPartiallySupported OracleSupport = "partially_supported"

	// OracleSupportUnsupported identifies a case that the external oracle cannot
	// model faithfully.
	// Authored by: OpenCode
	OracleSupportUnsupported OracleSupport = "unsupported"
)

// EvidenceSupportLabel identifies whether one comparable evidence segment comes
// from rotki-backed arithmetic evidence or from a documented project-owned
// composition rule.
// Authored by: OpenCode
type EvidenceSupportLabel string

const (
	// EvidenceSupportLabelRotkiBacked identifies evidence that comes directly from
	// the rotki-backed arithmetic boundary.
	// Authored by: OpenCode
	EvidenceSupportLabelRotkiBacked EvidenceSupportLabel = "rotki_backed"

	// EvidenceSupportLabelProjectCompositionRule identifies evidence that is
	// produced by a documented project-owned composition rule.
	// Authored by: OpenCode
	EvidenceSupportLabelProjectCompositionRule EvidenceSupportLabel = "project_composition_rule"
)

// ComparisonPolicy identifies how empirical comparison code treats a dataset
// segment that the selected external oracle cannot represent faithfully.
// Authored by: OpenCode
type ComparisonPolicy string

const (
	// ComparisonPolicySkipExternalOracle skips unsupported external-oracle
	// comparison for the affected segment.
	// Authored by: OpenCode
	ComparisonPolicySkipExternalOracle ComparisonPolicy = "skip_external_oracle"

	// ComparisonPolicyProjectCompositionOnly restricts comparison to documented
	// project-owned composition assertions.
	// Authored by: OpenCode
	ComparisonPolicyProjectCompositionOnly ComparisonPolicy = "project_composition_only"

	// ComparisonPolicyFailIfSelected reports an error when the affected segment is
	// selected for external-oracle comparison.
	// Authored by: OpenCode
	ComparisonPolicyFailIfSelected ComparisonPolicy = "fail_if_selected"
)

// EmpiricalDataset stores the complete synthetic empirical dataset shared by
// dataset validation, oracle generation, and fixture-backed calculation tests.
// Authored by: OpenCode
type EmpiricalDataset struct {
	DatasetVersion   string                        `json:"dataset_version" yaml:"dataset_version"`
	Description      string                        `json:"description" yaml:"description"`
	Currency         string                        `json:"currency" yaml:"currency"`
	SupportedYears   []int                         `json:"supported_years" yaml:"supported_years"`
	SupportedMethods []reportmodel.CostBasisMethod `json:"supported_methods" yaml:"supported_methods"`
	CoverageTags     []string                      `json:"coverage_tags" yaml:"coverage_tags"`
	Activities       []EmpiricalActivity           `json:"activities" yaml:"activities"`
	Cases            []EmpiricalCase               `json:"cases" yaml:"cases"`
}

// EmpiricalActivity stores one synthetic input activity in the project-owned
// empirical dataset schema.
// Authored by: OpenCode
type EmpiricalActivity struct {
	SourceID                       string                 `json:"source_id" yaml:"source_id"`
	OccurredAt                     string                 `json:"occurred_at" yaml:"occurred_at"`
	DeterministicOrder             int                    `json:"deterministic_order" yaml:"deterministic_order"`
	ActivityType                   syncmodel.ActivityType `json:"activity_type" yaml:"activity_type"`
	AssetIdentityKey               string                 `json:"asset_identity_key" yaml:"asset_identity_key"`
	AssetSymbol                    string                 `json:"asset_symbol" yaml:"asset_symbol"`
	Quantity                       string                 `json:"quantity" yaml:"quantity"`
	GrossValue                     string                 `json:"gross_value,omitempty" yaml:"gross_value,omitempty"`
	UnitPrice                      string                 `json:"unit_price,omitempty" yaml:"unit_price,omitempty"`
	FeeAmount                      string                 `json:"fee_amount,omitempty" yaml:"fee_amount,omitempty"`
	Currency                       string                 `json:"currency,omitempty" yaml:"currency,omitempty"`
	SourceScope                    *EmpiricalScope        `json:"source_scope,omitempty" yaml:"source_scope,omitempty"`
	ZeroPricedReductionExplanation string                 `json:"zero_priced_reduction_explanation,omitempty" yaml:"zero_priced_reduction_explanation,omitempty"`
	CoverageTags                   []string               `json:"coverage_tags" yaml:"coverage_tags"`
}

// EmpiricalScope stores one synthetic scope grouping used by scope-local
// empirical cases.
// Authored by: OpenCode
type EmpiricalScope struct {
	ScopeID     string                     `json:"scope_id" yaml:"scope_id"`
	ScopeKind   syncmodel.SourceScopeKind  `json:"scope_kind" yaml:"scope_kind"`
	Reliability syncmodel.ScopeReliability `json:"reliability" yaml:"reliability"`
	DisplayName string                     `json:"display_name,omitempty" yaml:"display_name,omitempty"`
}

// EmpiricalCase stores one named validation slice covering selected dataset
// activities, methods, year, and oracle support expectations.
// Authored by: OpenCode
type EmpiricalCase struct {
	CaseID            string                        `json:"case_id" yaml:"case_id"`
	Description       string                        `json:"description" yaml:"description"`
	Methods           []reportmodel.CostBasisMethod `json:"methods" yaml:"methods"`
	Year              int                           `json:"year" yaml:"year"`
	AssetIdentityKeys []string                      `json:"asset_identity_keys" yaml:"asset_identity_keys"`
	ActivitySourceIDs []string                      `json:"activity_source_ids" yaml:"activity_source_ids"`
	CoverageTags      []string                      `json:"coverage_tags" yaml:"coverage_tags"`
	OracleSupport     OracleSupport                 `json:"oracle_support" yaml:"oracle_support"`
	UnsupportedReason string                        `json:"unsupported_reason,omitempty" yaml:"unsupported_reason,omitempty"`
}

// OracleInputLedger stores one generated external-oracle input artifact and its
// reproducibility metadata.
// Authored by: OpenCode
type OracleInputLedger struct {
	LedgerID                string   `json:"ledger_id" yaml:"ledger_id"`
	Method                  string   `json:"method" yaml:"method"`
	CaseIDs                 []string `json:"case_ids" yaml:"case_ids"`
	ExternalOracleInputPath string   `json:"external_oracle_input_path" yaml:"external_oracle_input_path"`
	DatasetInputHash        string   `json:"dataset_input_hash" yaml:"dataset_input_hash"`
	ExternalOracleInputHash string   `json:"external_oracle_input_hash" yaml:"external_oracle_input_hash"`
	GenerationNotes         []string `json:"generation_notes" yaml:"generation_notes"`
}

// ComparableOutputValues stores the canonical decimal-string values that are
// compared between oracle fixtures and normalized project calculation output.
// Authored by: OpenCode
type ComparableOutputValues struct {
	RealizedGainOrLoss string `json:"realized_gain_or_loss" yaml:"realized_gain_or_loss"`
	AllocatedBasis     string `json:"allocated_basis" yaml:"allocated_basis"`
	ClosingQuantity    string `json:"closing_quantity" yaml:"closing_quantity"`
	ClosingBasis       string `json:"closing_basis" yaml:"closing_basis"`
}

// OracleGenerationRun stores the persisted metadata recorded for one
// deterministic external-oracle or composite-oracle generation run.
// Authored by: OpenCode
type OracleGenerationRun struct {
	RunID                   string            `json:"run_id,omitempty" yaml:"run_id,omitempty"`
	OracleName              string            `json:"oracle_name" yaml:"oracle_name"`
	SourceURL               string            `json:"source_url" yaml:"source_url"`
	SourceChecksum          string            `json:"source_checksum" yaml:"source_checksum"`
	VersionOrCommit         string            `json:"version_or_commit" yaml:"version_or_commit"`
	AdapterArguments        []string          `json:"adapter_arguments" yaml:"adapter_arguments"`
	AdapterConstraints      []string          `json:"adapter_constraints" yaml:"adapter_constraints"`
	DatasetInputHash        string            `json:"dataset_input_hash" yaml:"dataset_input_hash"`
	ExternalOracleInputHash string            `json:"external_oracle_input_hash" yaml:"external_oracle_input_hash"`
	DecimalPolicy           string            `json:"decimal_policy" yaml:"decimal_policy"`
	NormalizationVersion    string            `json:"normalization_version" yaml:"normalization_version"`
	CompositeRuleVersion    string            `json:"composite_rule_version,omitempty" yaml:"composite_rule_version,omitempty"`
	FinancialTolerances     map[string]string `json:"financial_tolerances" yaml:"financial_tolerances"`
	ToleranceNotes          map[string]string `json:"tolerance_notes" yaml:"tolerance_notes"`
	OracleOutputHash        string            `json:"oracle_output_hash" yaml:"oracle_output_hash"`
	GeneratedAt             string            `json:"generated_at,omitempty" yaml:"generated_at,omitempty"`
}

// OracleOutput stores one normalized golden fixture derived from external-oracle
// or composite-oracle output and project-owned normalization.
// Authored by: OpenCode
type OracleOutput struct {
	FixtureVersion      string                      `json:"fixture_version" yaml:"fixture_version"`
	DatasetVersion      string                      `json:"dataset_version" yaml:"dataset_version"`
	CaseID              string                      `json:"case_id" yaml:"case_id"`
	Method              reportmodel.CostBasisMethod `json:"method" yaml:"method"`
	Year                int                         `json:"year" yaml:"year"`
	AssetIdentityKey    string                      `json:"asset_identity_key" yaml:"asset_identity_key"`
	Values              ComparableOutputValues      `json:"values" yaml:"values"`
	Matches             []OracleMatchEvidence       `json:"matches" yaml:"matches"`
	UnsupportedSegments []UnsupportedOracleSegment  `json:"unsupported_segments" yaml:"unsupported_segments"`
	Metadata            OracleGenerationRun         `json:"metadata" yaml:"metadata"`
}

// OracleMatchEvidence stores one comparable lot, pool, or scope evidence row
// recorded in a normalized oracle fixture.
// Authored by: OpenCode
type OracleMatchEvidence struct {
	DisposedSourceID    string               `json:"disposed_source_id" yaml:"disposed_source_id"`
	AcquisitionSourceID string               `json:"acquisition_source_id,omitempty" yaml:"acquisition_source_id,omitempty"`
	ScopeID             string               `json:"scope_id,omitempty" yaml:"scope_id,omitempty"`
	MatchedQuantity     string               `json:"matched_quantity" yaml:"matched_quantity"`
	MatchedBasis        string               `json:"matched_basis" yaml:"matched_basis"`
	MatchedProceeds     string               `json:"matched_proceeds,omitempty" yaml:"matched_proceeds,omitempty"`
	MatchedGainOrLoss   string               `json:"matched_gain_or_loss,omitempty" yaml:"matched_gain_or_loss,omitempty"`
	SupportLabel        EvidenceSupportLabel `json:"support_label,omitempty" yaml:"support_label,omitempty"`
	CompositionRuleID   string               `json:"composition_rule_id,omitempty" yaml:"composition_rule_id,omitempty"`
}

// UnsupportedOracleSegment stores one explicit unsupported external-oracle
// segment and the policy that comparison code must apply to it.
// Authored by: OpenCode
type UnsupportedOracleSegment struct {
	CaseID            string                      `json:"case_id" yaml:"case_id"`
	Method            reportmodel.CostBasisMethod `json:"method" yaml:"method"`
	ActivitySourceIDs []string                    `json:"activity_source_ids" yaml:"activity_source_ids"`
	Reason            string                      `json:"reason" yaml:"reason"`
	ComparisonPolicy  ComparisonPolicy            `json:"comparison_policy" yaml:"comparison_policy"`
}

// ProjectCalculationOutput stores one normalized project calculation segment in
// the same comparable shape as one oracle fixture segment.
// Authored by: OpenCode
type ProjectCalculationOutput struct {
	CaseID           string                      `json:"case_id" yaml:"case_id"`
	Method           reportmodel.CostBasisMethod `json:"method" yaml:"method"`
	Year             int                         `json:"year" yaml:"year"`
	AssetIdentityKey string                      `json:"asset_identity_key" yaml:"asset_identity_key"`
	Values           ComparableOutputValues      `json:"values" yaml:"values"`
	Matches          []ProjectMatchEvidence      `json:"matches" yaml:"matches"`
}

// ProjectMatchEvidence stores one normalized project-side comparable match or
// composition evidence row.
// Authored by: OpenCode
type ProjectMatchEvidence struct {
	DisposedSourceID    string               `json:"disposed_source_id" yaml:"disposed_source_id"`
	AcquisitionSourceID string               `json:"acquisition_source_id,omitempty" yaml:"acquisition_source_id,omitempty"`
	ScopeID             string               `json:"scope_id,omitempty" yaml:"scope_id,omitempty"`
	MatchedQuantity     string               `json:"matched_quantity" yaml:"matched_quantity"`
	MatchedBasis        string               `json:"matched_basis" yaml:"matched_basis"`
	MatchedProceeds     string               `json:"matched_proceeds,omitempty" yaml:"matched_proceeds,omitempty"`
	MatchedGainOrLoss   string               `json:"matched_gain_or_loss,omitempty" yaml:"matched_gain_or_loss,omitempty"`
	SupportLabel        EvidenceSupportLabel `json:"support_label,omitempty" yaml:"support_label,omitempty"`
	CompositionRuleID   string               `json:"composition_rule_id,omitempty" yaml:"composition_rule_id,omitempty"`
}

// EmpiricalComparisonResult stores one normalized per-field comparison between
// oracle output and project calculation output.
// Authored by: OpenCode
type EmpiricalComparisonResult struct {
	CaseID            string                      `json:"case_id" yaml:"case_id"`
	Method            reportmodel.CostBasisMethod `json:"method" yaml:"method"`
	Year              int                         `json:"year" yaml:"year"`
	AssetIdentityKey  string                      `json:"asset_identity_key" yaml:"asset_identity_key"`
	Field             string                      `json:"field" yaml:"field"`
	ExpectedValue     string                      `json:"expected_value" yaml:"expected_value"`
	ActualValue       string                      `json:"actual_value" yaml:"actual_value"`
	Difference        string                      `json:"difference" yaml:"difference"`
	DecimalPolicy     string                      `json:"decimal_policy" yaml:"decimal_policy"`
	Tolerance         string                      `json:"tolerance" yaml:"tolerance"`
	Passed            bool                        `json:"passed" yaml:"passed"`
	DiagnosticContext string                      `json:"diagnostic_context,omitempty" yaml:"diagnostic_context,omitempty"`
	RelevantSourceIDs []string                    `json:"relevant_source_ids,omitempty" yaml:"relevant_source_ids,omitempty"`
}
