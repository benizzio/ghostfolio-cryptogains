// Package fixture provides empirical fixture loading and validation helpers.
// Authored by: OpenCode
package fixture

import (
	"fmt"
	"strconv"
	"strings"
)

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

// validateMatches delegates match-evidence validation to the focused evidence validator.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateMatches() {
	var evidenceValidator = newOracleOutputEvidenceValidator(validator)
	evidenceValidator.validateMatches()
}

// validateUnsupportedSegments delegates unsupported-segment validation to the
// focused evidence validator.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateUnsupportedSegments() {
	var evidenceValidator = newOracleOutputEvidenceValidator(validator)
	evidenceValidator.validateUnsupportedSegments()
}

// validateMetadata delegates metadata validation to the focused metadata validator.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateMetadata() {
	var metadataValidator = newOracleOutputMetadataValidator(validator)
	metadataValidator.validateMetadata()
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
