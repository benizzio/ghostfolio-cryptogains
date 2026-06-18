// Package fixture provides empirical fixture loading and validation helpers.
// Authored by: OpenCode
package fixture

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

var requiredOracleFinancialToleranceFields = []string{
	"realized_gain_or_loss",
	"allocated_basis",
	"closing_basis",
}

// oracleDecimalPolicy stores the parsed decimal-policy scale required for
// tolerance validation.
// Authored by: OpenCode
type oracleDecimalPolicy struct {
	Scale int
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
	validator.validateSHA256Field("metadata.source_checksum", metadata.SourceChecksum)

	validator.validateAdapterArguments(metadata.AdapterArguments)
	if metadata.AdapterConstraints == nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.adapter_constraints", "schema", "adapter_constraints must be present as a JSON array")
	}

	validator.validateSHA256Field("metadata.dataset_input_hash", metadata.DatasetInputHash)
	validator.validateSHA256Field("metadata.external_oracle_input_hash", metadata.ExternalOracleInputHash)
	validator.validateSHA256Field("metadata.oracle_output_hash", metadata.OracleOutputHash)

	var policy, policyValid = validator.validateDecimalPolicy(metadata.DecimalPolicy)
	validator.validateToleranceMetadata(policy, policyValid, metadata.FinancialTolerances, metadata.ToleranceNotes)
}

// validateAdapterArguments enforces non-empty adapter argument metadata.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateAdapterArguments(adapterArguments []string) {
	if adapterArguments == nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.adapter_arguments", "schema", "adapter_arguments must be present as a JSON array")
		return
	}
	if len(adapterArguments) == 0 {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.adapter_arguments", "required_field", "adapter_arguments must contain at least one argument")
		return
	}

	var index int
	for index = range adapterArguments {
		if strings.TrimSpace(adapterArguments[index]) != "" {
			continue
		}

		validator.addIssue("case_id", validator.output.CaseID, "metadata.adapter_arguments", "adapter_arguments", "adapter_arguments must not contain blank values")
		break
	}
}

// validateDecimalPolicy parses decimal policy metadata when it is present.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateDecimalPolicy(raw string) (oracleDecimalPolicy, bool) {
	if strings.TrimSpace(raw) == "" {
		return oracleDecimalPolicy{}, false
	}

	var policy, err = parseOracleDecimalPolicy(raw)
	if err != nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.decimal_policy", "decimal_policy", err.Error())
		return oracleDecimalPolicy{}, false
	}

	return policy, true
}

// validateToleranceMetadata enforces tolerance object and note metadata rules.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateToleranceMetadata(policy oracleDecimalPolicy, policyValid bool, tolerances map[string]string, notes map[string]string) {
	if tolerances == nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances", "schema", "financial_tolerances must be present as a JSON object")
	} else {
		validator.validateRequiredToleranceKeys(tolerances)
		validator.validateFinancialTolerances(policy, policyValid, tolerances, notes)
	}

	if notes == nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.tolerance_notes", "schema", "tolerance_notes must be present as a JSON object")
	} else {
		validator.validateToleranceNotes(tolerances, notes)
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
	var maximumTolerance, maximumValid = validator.maximumFinancialTolerance(policy, policyValid)

	for field, rawValue := range tolerances {
		validator.validateFinancialTolerance(field, rawValue, notes, maximumTolerance, maximumValid)
	}
}

// maximumFinancialTolerance returns the parsed maximum tolerance when policy validation succeeded.
// Authored by: OpenCode
func (validator *oracleOutputValidator) maximumFinancialTolerance(policy oracleDecimalPolicy, policyValid bool) (apd.Decimal, bool) {
	if !policyValid {
		return apd.Decimal{}, false
	}

	var maximumTolerance, _, err = maximumToleranceForScale(policy.Scale)
	if err != nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.decimal_policy", "decimal_policy", err.Error())
		return apd.Decimal{}, false
	}

	return maximumTolerance, true
}

// validateFinancialTolerance enforces one financial tolerance value and its note.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateFinancialTolerance(field string, rawValue string, notes map[string]string, maximumTolerance apd.Decimal, maximumValid bool) {
	if strings.TrimSpace(field) == "" {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances", "financial_tolerances", "financial_tolerances must not contain blank field keys")
		return
	}

	var value, _, err = ParseDecimalString(rawValue)
	if err != nil {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "canonical_decimal", err.Error())
		return
	}
	if value.Sign() < 0 {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "financial_tolerances", "financial tolerances must not be negative")
		return
	}
	if maximumValid && value.Cmp(&maximumTolerance) > 0 {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "financial_tolerances", fmt.Sprintf("financial tolerance exceeds the maximum %s allowed by %s", maximumTolerance.Text('f'), validator.output.Metadata.DecimalPolicy))
	}
	if strings.Contains(strings.ToLower(field), "quantity") && value.Sign() != 0 {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.financial_tolerances."+field, "financial_tolerances", "quantity tolerances must be 0")
	}
	if value.Sign() != 0 && strings.TrimSpace(notes[field]) == "" {
		validator.addIssue("case_id", validator.output.CaseID, "metadata.tolerance_notes."+field, "tolerance_notes", "non-zero financial tolerance requires a tolerance note")
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

// validateSHA256Field enforces the canonical `sha256:`-prefixed hash format.
// Authored by: OpenCode
func (validator *oracleOutputValidator) validateSHA256Field(field string, value string) {
	if err := validateSHA256PrefixedHash(value); err != nil {
		validator.addIssue("case_id", validator.output.CaseID, field, "hash", err.Error())
	}
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
