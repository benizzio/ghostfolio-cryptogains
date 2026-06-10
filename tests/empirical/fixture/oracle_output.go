package fixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
)

var requiredOracleFinancialToleranceFields = []string{
	"realized_gain_or_loss",
	"allocated_basis",
	"closing_basis",
}

// LoadOracleOutputs reads and validates every oracle-output JSON fixture below
// one repository-controlled root directory.
//
// Example:
//
//	outputs, err := fixture.LoadOracleOutputs("testdata/empirical/golden")
//	if err != nil {
//		panic(err)
//	}
//	_ = outputs
//
// LoadOracleOutputs walks the root recursively, loads every `.json` file in
// stable path order, validates synthetic-only content, validates canonical
// decimal strings, and verifies the stored stable hash.
// Authored by: OpenCode
func LoadOracleOutputs(rootPath string) ([]OracleOutput, error) {
	var paths, err = collectOracleOutputPaths(rootPath)
	if err != nil {
		return nil, err
	}

	var outputs = make([]OracleOutput, 0, len(paths))
	var index int

	for index = range paths {
		var output OracleOutput
		output, _, err = LoadOracleOutput(paths[index])
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

// LoadOracleOutput reads, parses, and validates one persisted oracle-output
// fixture file.
//
// Example:
//
//	output, rawContent, err := fixture.LoadOracleOutput("testdata/empirical/golden/fifo.json")
//	if err != nil {
//		panic(err)
//	}
//	_, _ = output, rawContent
//
// Authored by: OpenCode
func LoadOracleOutput(path string) (OracleOutput, string, error) {
	var rawContent, err = os.ReadFile(path)
	if err != nil {
		return OracleOutput{}, "", fmt.Errorf("read oracle output %s: %w", path, err)
	}

	var output OracleOutput
	output, err = ParseOracleOutput(path, rawContent)
	if err != nil {
		return OracleOutput{}, "", err
	}

	return output, string(rawContent), nil
}

// ParseOracleOutput parses one raw oracle-output JSON payload into the shared
// oracle fixture model and applies the persisted-fixture validation contract.
//
// Example:
//
//	output, err := fixture.ParseOracleOutput(path, rawContent)
//	if err != nil {
//		panic(err)
//	}
//	_ = output
//
// ParseOracleOutput rejects unknown JSON fields, trailing JSON content,
// float-style JSON numbers in string-decimal fields, and any persisted decimal
// text that is not already canonical.
// Authored by: OpenCode
func ParseOracleOutput(path string, content []byte) (OracleOutput, error) {
	if err := ValidateSyntheticOnlyContent(path, string(content)); err != nil {
		return OracleOutput{}, err
	}

	var decoder = json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()

	var output OracleOutput
	if err := decoder.Decode(&output); err != nil {
		return OracleOutput{}, fmt.Errorf("decode oracle output JSON %s: %w", path, err)
	}

	var trailing struct{}
	var err = decoder.Decode(&trailing)
	if err == nil {
		return OracleOutput{}, fmt.Errorf("decode oracle output JSON %s: unexpected trailing JSON content", path)
	}
	if err != io.EOF {
		return OracleOutput{}, fmt.Errorf("decode oracle output JSON %s: %w", path, err)
	}

	err = validateOracleOutputStructure(path, output)
	if err != nil {
		return OracleOutput{}, err
	}

	return output, nil
}

// ValidateOracleOutput applies the oracle-output fixture contract to one
// already-parsed fixture value and its persisted raw JSON content.
//
// Example:
//
//	err := fixture.ValidateOracleOutput(path, rawContent, output)
//	if err != nil {
//		panic(err)
//	}
//
// ValidateOracleOutput enforces required metadata, canonical decimal strings,
// tolerance metadata, supported methods, match evidence rules, unsupported
// segment rules, and stable-hash verification.
// Authored by: OpenCode
func ValidateOracleOutput(path string, rawContent string, output OracleOutput) error {
	if err := ValidateSyntheticOnlyContent(path, rawContent); err != nil {
		return err
	}

	return validateOracleOutputStructure(path, output)
}

// StableOracleOutputHash returns the deterministic stable hash for one oracle
// output fixture.
//
// Example:
//
//	hash, err := fixture.StableOracleOutputHash(output)
//	if err != nil {
//		panic(err)
//	}
//	_ = hash
//
// StableOracleOutputHash hashes a canonical JSON representation and excludes
// `metadata.oracle_output_hash`, `metadata.run_id`, and `metadata.generated_at`
// so equivalent normalized fixtures keep the same stored hash across
// regenerations.
// Authored by: OpenCode
func StableOracleOutputHash(output OracleOutput) (string, error) {
	var canonicalOutput, err = canonicalOracleOutputForHash(output)
	if err != nil {
		return "", fmt.Errorf("canonicalize oracle output hash input: %w", err)
	}

	var payload []byte
	payload, err = json.Marshal(canonicalOutput)
	if err != nil {
		return "", fmt.Errorf("marshal oracle output hash input: %w", err)
	}

	var digest = sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

// oracleOutputValidationIssue stores one actionable oracle-output fixture
// validation failure.
// Authored by: OpenCode
type oracleOutputValidationIssue struct {
	Field          string
	Kind           string
	Message        string
	Path           string
	ReferenceKind  string
	ReferenceValue string
}

// Error formats one oracle-output fixture validation issue.
// Authored by: OpenCode
func (issue oracleOutputValidationIssue) Error() string {
	var builder strings.Builder

	builder.WriteString(issue.Path)
	if issue.ReferenceKind != "" && issue.ReferenceValue != "" {
		builder.WriteString(" ")
		builder.WriteString(issue.ReferenceKind)
		builder.WriteString(" ")
		builder.WriteString(issue.ReferenceValue)
	}
	if issue.Field != "" {
		builder.WriteString(" field ")
		builder.WriteString(issue.Field)
	}
	builder.WriteString(": ")
	builder.WriteString(issue.Kind)
	builder.WriteString(": ")
	builder.WriteString(issue.Message)

	return builder.String()
}

// oracleOutputValidationError groups oracle-output fixture validation issues.
// Authored by: OpenCode
type oracleOutputValidationError struct {
	Issues []oracleOutputValidationIssue
	Path   string
}

// Error formats grouped oracle-output fixture validation issues.
// Authored by: OpenCode
func (issueError oracleOutputValidationError) Error() string {
	var builder strings.Builder

	builder.WriteString(issueError.Path)
	builder.WriteString(" failed oracle output validation with ")
	builder.WriteString(strconv.Itoa(len(issueError.Issues)))
	builder.WriteString(" issue(s):")

	for _, issue := range issueError.Issues {
		builder.WriteString("\n- ")
		builder.WriteString(issue.Error())
	}

	return builder.String()
}

// oracleOutputValidator stores the current oracle-output validation state.
// Authored by: OpenCode
type oracleOutputValidator struct {
	issues []oracleOutputValidationIssue
	output OracleOutput
	path   string
}

// oracleDecimalPolicy stores the parsed decimal-policy scale required for
// tolerance validation.
// Authored by: OpenCode
type oracleDecimalPolicy struct {
	Scale int
}

// validateOracleOutputStructure validates one already-parsed oracle-output
// fixture value without re-reading the persisted file.
// Authored by: OpenCode
func validateOracleOutputStructure(path string, output OracleOutput) error {
	var validator = newOracleOutputValidator(path, output)
	validator.validate()

	if len(validator.issues) == 0 {
		return nil
	}

	return oracleOutputValidationError{Issues: validator.issues, Path: path}
}

// newOracleOutputValidator builds one validator instance for one parsed oracle fixture.
// Authored by: OpenCode
func newOracleOutputValidator(path string, output OracleOutput) oracleOutputValidator {
	return oracleOutputValidator{
		issues: make([]oracleOutputValidationIssue, 0),
		output: output,
		path:   path,
	}
}

// validate runs the full oracle-output fixture validation pass.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validate() {
	validator.validateTopLevelFields()
	validator.validateComparableValues()
	validator.validateMatches()
	validator.validateUnsupportedSegments()
	validator.validateMetadata()

	if len(validator.issues) == 0 {
		validator.validateStoredHash()
	}
}

// validateTopLevelFields enforces required top-level schema fields.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateTopLevelFields() {
	validator.validateRequiredTextField("fixture_version", validator.output.FixtureVersion)
	validator.validateRequiredTextField("dataset_version", validator.output.DatasetVersion)
	validator.validateRequiredTextField("case_id", validator.output.CaseID)
	validator.validateRequiredTextField("asset_identity_key", validator.output.AssetIdentityKey)

	if !isSupportedCostBasisMethod(validator.output.Method) {
		validator.addIssue("case_id", validator.output.CaseID, "method", "supported_method", fmt.Sprintf("unsupported cost basis method %s", validator.output.Method))
	}
	if validator.output.Year <= 0 {
		validator.addIssue("case_id", validator.output.CaseID, "year", "year", "year must be greater than zero")
	}
	if validator.output.Matches == nil {
		validator.addIssue("case_id", validator.output.CaseID, "matches", "schema", "matches must be present as a JSON array")
	}
	if validator.output.UnsupportedSegments == nil {
		validator.addIssue("case_id", validator.output.CaseID, "unsupported_segments", "schema", "unsupported_segments must be present as a JSON array")
	}
}

// validateComparableValues enforces the canonical decimal-string contract for
// the top-level comparable values.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateComparableValues() {
	validator.validateRequiredCanonicalDecimal("case_id", validator.output.CaseID, "values.realized_gain_or_loss", validator.output.Values.RealizedGainOrLoss)
	validator.validateRequiredCanonicalDecimal("case_id", validator.output.CaseID, "values.allocated_basis", validator.output.Values.AllocatedBasis)
	validator.validateRequiredCanonicalDecimal("case_id", validator.output.CaseID, "values.closing_quantity", validator.output.Values.ClosingQuantity)
	validator.validateRequiredCanonicalDecimal("case_id", validator.output.CaseID, "values.closing_basis", validator.output.Values.ClosingBasis)
}

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

// validateMetadata enforces required metadata presence, hash format, and
// tolerance metadata rules.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateMetadata() {
	var metadata = validator.output.Metadata
	validator.validateRequiredTextField("metadata.oracle_name", metadata.OracleName)
	validator.validateRequiredTextField("metadata.source_url", metadata.SourceURL)
	validator.validateRequiredTextField("metadata.version_or_commit", metadata.VersionOrCommit)
	validator.validateRequiredTextField("metadata.decimal_policy", metadata.DecimalPolicy)
	validator.validateRequiredTextField("metadata.normalization_version", metadata.NormalizationVersion)

	if metadata.AdapterArguments == nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.adapter_arguments", "schema", "adapter_arguments must be present as a JSON array")
	} else if len(metadata.AdapterArguments) == 0 {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.adapter_arguments", "required_field", "adapter_arguments must contain at least one argument")
	} else {
		var index int
		for index = range metadata.AdapterArguments {
			if strings.TrimSpace(metadata.AdapterArguments[index]) != "" {
				continue
			}

			validator.addIssue("case_id", validator.output.CaseID, "metadata.adapter_arguments", "adapter_arguments", "adapter_arguments must not contain blank values")
			break
		}
	}
	if metadata.AdapterConstraints == nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.adapter_constraints", "schema", "adapter_constraints must be present as a JSON array")
	}

	validator.validateSHA256Field("metadata.dataset_input_hash", metadata.DatasetInputHash)
	validator.validateSHA256Field("metadata.external_oracle_input_hash", metadata.ExternalOracleInputHash)
	validator.validateSHA256Field("metadata.oracle_output_hash", metadata.OracleOutputHash)

	var policy oracleDecimalPolicy
	var policyValid bool
	if strings.TrimSpace(metadata.DecimalPolicy) != "" {
		var err error
		policy, err = parseOracleDecimalPolicy(metadata.DecimalPolicy)
		if err != nil {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.decimal_policy", "decimal_policy", err.Error())
		} else {
			policyValid = true
		}
	}

	if metadata.FinancialTolerances == nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances", "schema", "financial_tolerances must be present as a JSON object")
	} else {
		validator.validateRequiredToleranceKeys(metadata.FinancialTolerances)
		validator.validateFinancialTolerances(policy, policyValid, metadata.FinancialTolerances, metadata.ToleranceNotes)
	}

	if metadata.ToleranceNotes == nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.tolerance_notes", "schema", "tolerance_notes must be present as a JSON object")
	} else {
		validator.validateToleranceNotes(metadata.FinancialTolerances, metadata.ToleranceNotes)
	}
}

// validateRequiredToleranceKeys enforces the core comparable financial tolerance keys.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateRequiredToleranceKeys(tolerances map[string]string) {
	var field string

	for _, field = range requiredOracleFinancialToleranceFields {
		if _, ok := tolerances[field]; ok {
			continue
		}

		validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "required_field", fmt.Sprintf("financial_tolerances must include %s", field))
	}
}

// validateFinancialTolerances enforces canonical tolerance decimals, the
// decimal-policy maximum, and the required note for every non-zero tolerance.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateFinancialTolerances(policy oracleDecimalPolicy, policyValid bool, tolerances map[string]string, notes map[string]string) {
	var maximumTolerance apd.Decimal
	if policyValid {
		var err error
		maximumTolerance, _, err = maximumToleranceForScale(policy.Scale)
		if err != nil {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.decimal_policy", "decimal_policy", err.Error())
			policyValid = false
		}
	}

	for field, rawValue := range tolerances {
		if strings.TrimSpace(field) == "" {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances", "financial_tolerances", "financial_tolerances must not contain blank field keys")
			continue
		}

		var value, _, err = ParseDecimalString(rawValue)
		if err != nil {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "canonical_decimal", err.Error())
			continue
		}
		if value.Sign() < 0 {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "financial_tolerances", "financial tolerances must not be negative")
			continue
		}
		if policyValid && value.Cmp(&maximumTolerance) > 0 {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "financial_tolerances", fmt.Sprintf("financial tolerance exceeds the maximum %s allowed by %s", maximumTolerance.Text('f'), validator.output.Metadata.DecimalPolicy))
		}
		if strings.Contains(strings.ToLower(field), "quantity") && value.Sign() != 0 {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "financial_tolerances", "quantity tolerances must be 0")
		}
		if value.Sign() != 0 && strings.TrimSpace(notes[field]) == "" {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.tolerance_notes."+field, "tolerance_notes", "non-zero financial tolerance requires a tolerance note")
		}
	}
}

// validateToleranceNotes enforces non-empty tolerance-note entries and rejects
// orphan note keys.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateToleranceNotes(tolerances map[string]string, notes map[string]string) {
	for field, note := range notes {
		if strings.TrimSpace(field) == "" {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.tolerance_notes", "tolerance_notes", "tolerance_notes must not contain blank field keys")
			continue
		}
		if strings.TrimSpace(note) == "" {
			validator.addIssue("case_id", validator.output.CaseID, "metadata.tolerance_notes."+field, "tolerance_notes", "tolerance note must be non-empty")
		}
		if tolerances != nil {
			if _, ok := tolerances[field]; ok {
				continue
			}
		}

		validator.addIssue("case_id", validator.output.CaseID, "metadata.tolerance_notes."+field, "tolerance_notes", "tolerance note key must match a financial_tolerances key")
	}
}

// validateStoredHash recomputes the stable hash and verifies that the persisted
// `oracle_output_hash` matches it exactly.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateStoredHash() {
	var expectedHash, err = StableOracleOutputHash(validator.output)
	if err != nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.oracle_output_hash", "hash", err.Error())
		return
	}
	if validator.output.Metadata.OracleOutputHash == expectedHash {
		return
	}

	validator.addIssue("case_id", validator.output.CaseID, "metadata.oracle_output_hash", "hash", fmt.Sprintf("stored hash %s does not match recomputed hash %s", validator.output.Metadata.OracleOutputHash, expectedHash))
}

// validateRequiredTextField enforces one non-empty required string field.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateRequiredTextField(field string, value string) {
	if strings.TrimSpace(value) != "" {
		return
	}

	validator.addIssue("case_id", validator.output.CaseID, field, "required_field", fmt.Sprintf("%s must be non-empty", field))
}

// validateRequiredCanonicalDecimal enforces one required canonical decimal string field.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateRequiredCanonicalDecimal(referenceKind string, referenceValue string, field string, raw string) {
	if strings.TrimSpace(raw) == "" {
		validator.addIssue(referenceKind, referenceValue, field, "required_field", fmt.Sprintf("%s must be non-empty", field))
		return
	}

	if _, _, err := ParseDecimalString(raw); err != nil {
		validator.addIssue(referenceKind, referenceValue, field, "canonical_decimal", err.Error())
	}
}

// validateOptionalCanonicalDecimal enforces one optional canonical decimal string field.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateOptionalCanonicalDecimal(referenceKind string, referenceValue string, field string, raw string) {
	if strings.TrimSpace(raw) == "" {
		return
	}

	if _, _, err := ParseDecimalString(raw); err != nil {
		validator.addIssue(referenceKind, referenceValue, field, "canonical_decimal", err.Error())
	}
}

// validateSHA256Field enforces the canonical `sha256:`-prefixed hash format.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateSHA256Field(field string, value string) {
	if err := validateSHA256PrefixedHash(value); err != nil {
		validator.addIssue("case_id", validator.output.CaseID, field, "hash", err.Error())
	}
}

// addIssue records one oracle-output validation issue.
// Authored by: OpenCode
func (validator *oracleOutputValidator) addIssue(referenceKind string, referenceValue string, field string, kind string, message string) {
	validator.issues = append(validator.issues, oracleOutputValidationIssue{
		Field:          field,
		Kind:           kind,
		Message:        message,
		Path:           validator.path,
		ReferenceKind:  referenceKind,
		ReferenceValue: referenceValue,
	})
}

// collectOracleOutputPaths returns every JSON fixture path below one root in
// stable lexical order.
// Authored by: OpenCode
func collectOracleOutputPaths(rootPath string) ([]string, error) {
	var paths = make([]string, 0)

	var err = filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			return nil
		}

		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk oracle output fixtures %s: %w", rootPath, err)
	}

	sort.Strings(paths)
	if len(paths) != 0 {
		return paths, nil
	}

	return nil, fmt.Errorf("walk oracle output fixtures %s: no JSON fixtures found", rootPath)
}

// parseOracleDecimalPolicy parses the repository-supported oracle decimal-policy
// text format.
// Authored by: OpenCode
func parseOracleDecimalPolicy(raw string) (oracleDecimalPolicy, error) {
	var trimmed = strings.TrimSpace(raw)
	var parts = strings.Split(trimmed, ",")
	if len(parts) != 2 {
		return oracleDecimalPolicy{}, fmt.Errorf("decimal policy must use the form scale=<digits>,rounding=half_up")
	}
	if !strings.HasPrefix(parts[0], "scale=") {
		return oracleDecimalPolicy{}, fmt.Errorf("decimal policy must declare scale first")
	}
	if parts[1] != "rounding=half_up" {
		return oracleDecimalPolicy{}, fmt.Errorf("decimal policy must use rounding=half_up")
	}

	var scale, err = strconv.Atoi(strings.TrimPrefix(parts[0], "scale="))
	if err != nil {
		return oracleDecimalPolicy{}, fmt.Errorf("decimal policy scale must be an integer")
	}
	if scale < 0 {
		return oracleDecimalPolicy{}, fmt.Errorf("decimal policy scale must not be negative")
	}

	return oracleDecimalPolicy{Scale: scale}, nil
}

// maximumToleranceForScale returns the maximum permitted one-unit tolerance for
// the selected decimal-policy scale.
// Authored by: OpenCode
func maximumToleranceForScale(scale int) (apd.Decimal, string, error) {
	if scale < 0 {
		return apd.Decimal{}, "", fmt.Errorf("decimal policy scale must not be negative")
	}

	if scale == 0 {
		return ParseDecimalString("1")
	}

	var builder strings.Builder
	builder.Grow(scale + 2)
	builder.WriteString("0.")
	builder.WriteString(strings.Repeat("0", scale-1))
	builder.WriteByte('1')

	return ParseDecimalString(builder.String())
}

// validateSHA256PrefixedHash validates the canonical `sha256:`-prefixed
// lowercase hexadecimal hash format.
// Authored by: OpenCode
func validateSHA256PrefixedHash(raw string) error {
	var trimmed = strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("hash must be non-empty")
	}
	if !strings.HasPrefix(trimmed, "sha256:") {
		return fmt.Errorf("hash must use the sha256: prefix")
	}

	var digest = strings.TrimPrefix(trimmed, "sha256:")
	if len(digest) != sha256.Size*2 {
		return fmt.Errorf("hash must contain a 64-character hexadecimal sha256 digest")
	}
	if digest != strings.ToLower(digest) {
		return fmt.Errorf("hash digest must use lowercase hexadecimal text")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return fmt.Errorf("hash digest must be valid hexadecimal text")
	}

	return nil
}

// canonicalOracleOutputForHash prepares one oracle-output fixture for stable
// hashing by canonicalizing decimal strings, sorting evidence slices, and
// clearing self-referential or ephemeral metadata fields.
// Authored by: OpenCode
func canonicalOracleOutputForHash(output OracleOutput) (OracleOutput, error) {
	var canonical = output
	var err error

	canonical.Values, err = canonicalComparableOutputValues(output.Values)
	if err != nil {
		return OracleOutput{}, err
	}

	canonical.Matches, err = canonicalOracleMatches(output.Matches)
	if err != nil {
		return OracleOutput{}, err
	}

	canonical.UnsupportedSegments, err = canonicalUnsupportedOracleSegments(output.UnsupportedSegments)
	if err != nil {
		return OracleOutput{}, err
	}

	canonical.Metadata, err = canonicalOracleGenerationRun(output.Metadata)
	if err != nil {
		return OracleOutput{}, err
	}

	canonical.Metadata.RunID = ""
	canonical.Metadata.GeneratedAt = ""
	canonical.Metadata.OracleOutputHash = ""

	return canonical, nil
}

// canonicalComparableOutputValues canonicalizes the comparable decimal-string values.
// Authored by: OpenCode
func canonicalComparableOutputValues(values ComparableOutputValues) (ComparableOutputValues, error) {
	var err error

	values.RealizedGainOrLoss, err = canonicalRequiredPersistedDecimal(values.RealizedGainOrLoss)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("canonicalize realized_gain_or_loss: %w", err)
	}
	values.AllocatedBasis, err = canonicalRequiredPersistedDecimal(values.AllocatedBasis)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("canonicalize allocated_basis: %w", err)
	}
	values.ClosingQuantity, err = canonicalRequiredPersistedDecimal(values.ClosingQuantity)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("canonicalize closing_quantity: %w", err)
	}
	values.ClosingBasis, err = canonicalRequiredPersistedDecimal(values.ClosingBasis)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("canonicalize closing_basis: %w", err)
	}

	return values, nil
}

// canonicalOracleMatches canonicalizes and sorts comparable match evidence.
// Authored by: OpenCode
func canonicalOracleMatches(matches []OracleMatchEvidence) ([]OracleMatchEvidence, error) {
	var canonical = make([]OracleMatchEvidence, len(matches))
	copy(canonical, matches)

	var index int
	for index = range canonical {
		var err error
		canonical[index].MatchedQuantity, err = canonicalRequiredPersistedDecimal(canonical[index].MatchedQuantity)
		if err != nil {
			return nil, fmt.Errorf("canonicalize match %d matched_quantity: %w", index, err)
		}
		canonical[index].MatchedBasis, err = canonicalRequiredPersistedDecimal(canonical[index].MatchedBasis)
		if err != nil {
			return nil, fmt.Errorf("canonicalize match %d matched_basis: %w", index, err)
		}
		canonical[index].MatchedProceeds, err = canonicalOptionalPersistedDecimal(canonical[index].MatchedProceeds)
		if err != nil {
			return nil, fmt.Errorf("canonicalize match %d matched_proceeds: %w", index, err)
		}
		canonical[index].MatchedGainOrLoss, err = canonicalOptionalPersistedDecimal(canonical[index].MatchedGainOrLoss)
		if err != nil {
			return nil, fmt.Errorf("canonicalize match %d matched_gain_or_loss: %w", index, err)
		}
	}

	sort.Slice(canonical, func(left int, right int) bool {
		return oracleMatchSortKey(canonical[left]) < oracleMatchSortKey(canonical[right])
	})

	return canonical, nil
}

// canonicalUnsupportedOracleSegments canonicalizes and sorts unsupported segments.
// Authored by: OpenCode
func canonicalUnsupportedOracleSegments(segments []UnsupportedOracleSegment) ([]UnsupportedOracleSegment, error) {
	var canonical = make([]UnsupportedOracleSegment, len(segments))
	copy(canonical, segments)

	var index int
	for index = range canonical {
		canonical[index].ActivitySourceIDs = copyStringSlice(canonical[index].ActivitySourceIDs)
		sort.Strings(canonical[index].ActivitySourceIDs)
	}

	sort.Slice(canonical, func(left int, right int) bool {
		return unsupportedOracleSegmentSortKey(canonical[left]) < unsupportedOracleSegmentSortKey(canonical[right])
	})

	return canonical, nil
}

// canonicalOracleGenerationRun canonicalizes the hash-relevant generation metadata.
// Authored by: OpenCode
func canonicalOracleGenerationRun(metadata OracleGenerationRun) (OracleGenerationRun, error) {
	metadata.AdapterArguments = copyStringSlice(metadata.AdapterArguments)
	metadata.AdapterConstraints = copyStringSlice(metadata.AdapterConstraints)
	metadata.FinancialTolerances = copyStringMap(metadata.FinancialTolerances)
	metadata.ToleranceNotes = copyStringMap(metadata.ToleranceNotes)

	for field, rawValue := range metadata.FinancialTolerances {
		var canonicalValue, err = canonicalRequiredPersistedDecimal(rawValue)
		if err != nil {
			return OracleGenerationRun{}, fmt.Errorf("canonicalize financial_tolerances.%s: %w", field, err)
		}

		metadata.FinancialTolerances[field] = canonicalValue
	}

	return metadata, nil
}

// canonicalRequiredPersistedDecimal canonicalizes one required persisted decimal string.
// Authored by: OpenCode
func canonicalRequiredPersistedDecimal(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("decimal value is required")
	}

	_, canonical, err := ParseDecimalString(raw)
	if err != nil {
		return "", err
	}

	return canonical, nil
}

// canonicalOptionalPersistedDecimal canonicalizes one optional persisted decimal string.
// Authored by: OpenCode
func canonicalOptionalPersistedDecimal(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}

	return canonicalRequiredPersistedDecimal(raw)
}

// oracleMatchSortKey returns the stable lexical sort key for one match-evidence row.
// Authored by: OpenCode
func oracleMatchSortKey(match OracleMatchEvidence) string {
	return strings.Join([]string{
		match.DisposedSourceID,
		match.AcquisitionSourceID,
		match.ScopeID,
		match.MatchedQuantity,
		match.MatchedBasis,
		match.MatchedProceeds,
		match.MatchedGainOrLoss,
		string(match.SupportLabel),
		match.CompositionRuleID,
	}, "\x00")
}

// unsupportedOracleSegmentSortKey returns the stable lexical sort key for one
// unsupported segment.
// Authored by: OpenCode
func unsupportedOracleSegmentSortKey(segment UnsupportedOracleSegment) string {
	return strings.Join([]string{
		segment.CaseID,
		string(segment.Method),
		strings.Join(segment.ActivitySourceIDs, "\x01"),
		segment.Reason,
		string(segment.ComparisonPolicy),
	}, "\x00")
}

// copyStringSlice returns one stable non-nil copy of a string slice.
// Authored by: OpenCode
func copyStringSlice(values []string) []string {
	var copied = make([]string, len(values))
	copy(copied, values)
	return copied
}

// copyStringMap returns one stable non-nil copy of a string map.
// Authored by: OpenCode
func copyStringMap(values map[string]string) map[string]string {
	var copied = make(map[string]string, len(values))

	for key, value := range values {
		copied[key] = value
	}

	return copied
}
