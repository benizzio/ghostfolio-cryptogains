package fixture

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// empiricalDatasetCaseValidator owns case-specific empirical dataset validation.
// Authored by: OpenCode
type empiricalDatasetCaseValidator struct {
	*empiricalDatasetValidator
}

// newEmpiricalDatasetCaseValidator builds a case validator backed by the parent dataset validator state.
// Authored by: OpenCode
func newEmpiricalDatasetCaseValidator(validator *empiricalDatasetValidator) empiricalDatasetCaseValidator {
	return empiricalDatasetCaseValidator{empiricalDatasetValidator: validator}
}

// validateCases enforces basic case identifier and reference integrity.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCases() {
	var index int

	for index = range validator.dataset.Cases {
		validator.validateCase(&validator.dataset.Cases[index])
	}
}

// validateCase enforces one empirical case row.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCase(caseRecord *EmpiricalCase) {
	var caseID = validator.validateAndRegisterCaseID(caseRecord)

	validator.validateCaseMethods(caseRecord, caseID)
	var declaredAssets = validator.validateCaseAssets(caseRecord, caseID)
	var referencedAssets, hasSelectedYearActivity = validator.validateCaseActivityReferences(caseRecord, caseID, declaredAssets)
	validator.validateCaseYearCoverage(caseRecord, caseID, declaredAssets, referencedAssets, hasSelectedYearActivity)
	validator.validateCaseOracleSupport(caseRecord, caseID)
}

// validateAndRegisterCaseID validates one case identifier and records it for uniqueness checks.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateAndRegisterCaseID(caseRecord *EmpiricalCase) string {
	var caseID = strings.TrimSpace(caseRecord.CaseID)
	if caseID == "" {
		validator.addIssue("", "", "case_id", "case_id", "case_id must be non-empty")
		return caseID
	}
	if _, exists := validator.seenCaseIDs[caseID]; exists {
		validator.addIssue("case_id", caseID, "case_id", "case_id", "case_id must be unique")
		return caseID
	}

	validator.seenCaseIDs[caseID] = struct{}{}
	return caseID
}

// validateCaseMethods validates one case's method declarations against supported methods.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCaseMethods(caseRecord *EmpiricalCase, caseID string) {
	var seenCaseMethods = make(map[reportmodel.CostBasisMethod]struct{}, len(caseRecord.Methods))
	var method reportmodel.CostBasisMethod

	if len(caseRecord.Methods) == 0 {
		validator.addIssue("case_id", caseID, "methods", "supported_methods", "case must declare at least one supported method")
	}
	for _, method = range caseRecord.Methods {
		if _, exists := seenCaseMethods[method]; exists {
			validator.addIssue("case_id", caseID, "methods", "supported_methods", fmt.Sprintf("case must not repeat method %s", method))
			continue
		}
		seenCaseMethods[method] = struct{}{}
		if !isSupportedCostBasisMethod(method) {
			validator.addIssue("case_id", caseID, "methods", "supported_methods", fmt.Sprintf("case uses unsupported method %s", method))
			continue
		}
		if _, exists := validator.seenSupportedMethods[method]; !exists {
			validator.addIssue("case_id", caseID, "methods", "supported_methods", fmt.Sprintf("case method %s must also be declared in supported_methods", method))
		}
	}
}

// validateCaseAssets validates one case's declared asset identity keys.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCaseAssets(caseRecord *EmpiricalCase, caseID string) map[string]struct{} {
	var declaredAssets = make(map[string]struct{}, len(caseRecord.AssetIdentityKeys))

	if len(caseRecord.AssetIdentityKeys) == 0 {
		validator.addIssue("case_id", caseID, "asset_identity_keys", "asset_identity_keys", "case must declare at least one asset_identity_key")
	}
	for _, assetIdentityKey := range caseRecord.AssetIdentityKeys {
		var trimmedAssetIdentityKey = strings.TrimSpace(assetIdentityKey)
		if trimmedAssetIdentityKey == "" {
			validator.addIssue("case_id", caseID, "asset_identity_keys", "asset_identity_keys", "case asset_identity_keys must not contain blank values")
			continue
		}
		if _, exists := declaredAssets[trimmedAssetIdentityKey]; exists {
			validator.addIssue("case_id", caseID, "asset_identity_keys", "asset_identity_keys", fmt.Sprintf("case must not repeat asset_identity_key %s", trimmedAssetIdentityKey))
			continue
		}

		declaredAssets[trimmedAssetIdentityKey] = struct{}{}
	}

	return declaredAssets
}

// validateCaseActivityReferences validates case source references and activity asset linkage.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCaseActivityReferences(caseRecord *EmpiricalCase, caseID string, declaredAssets map[string]struct{}) (map[string]struct{}, bool) {
	var referencedAssets = make(map[string]struct{}, len(caseRecord.AssetIdentityKeys))
	var seenCaseSourceIDs = make(map[string]struct{}, len(caseRecord.ActivitySourceIDs))
	var hasSelectedYearActivity bool
	var sourceID string

	if len(caseRecord.ActivitySourceIDs) == 0 {
		validator.addIssue("case_id", caseID, "activity_source_ids", "activity_source_ids", "case must reference at least one activity source_id")
	}
	for _, sourceID = range caseRecord.ActivitySourceIDs {
		var trimmedSourceID = strings.TrimSpace(sourceID)
		var activity, validReference = validator.validateCaseActivityReference(caseID, trimmedSourceID, seenCaseSourceIDs)
		if !validReference {
			continue
		}

		validator.validateCaseActivityAssetReference(caseID, trimmedSourceID, activity, declaredAssets, referencedAssets)
		if caseActivityOccursInYear(activity, caseRecord.Year) {
			hasSelectedYearActivity = true
		}
	}

	return referencedAssets, hasSelectedYearActivity
}

// validateCaseActivityReference validates one source ID reference and returns its activity when valid.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCaseActivityReference(caseID string, sourceID string, seenCaseSourceIDs map[string]struct{}) (EmpiricalActivity, bool) {
	if sourceID == "" {
		validator.addIssue("case_id", caseID, "activity_source_ids", "activity_source_ids", "case activity_source_ids must not contain blank values")
		return EmpiricalActivity{}, false
	}
	if _, exists := seenCaseSourceIDs[sourceID]; exists {
		validator.addIssue("case_id", caseID, "activity_source_ids", "activity_source_ids", fmt.Sprintf("case must not repeat source_id %s", sourceID))
		return EmpiricalActivity{}, false
	}
	seenCaseSourceIDs[sourceID] = struct{}{}

	var activity, exists = validator.activitiesBySourceID[sourceID]
	if !exists {
		validator.addIssue("case_id", caseID, "activity_source_ids", "activity_source_ids", fmt.Sprintf("case references unknown source_id %s", sourceID))
		return EmpiricalActivity{}, false
	}

	return activity, true
}

// validateCaseActivityAssetReference validates one referenced activity against the declared case assets.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCaseActivityAssetReference(caseID string, sourceID string, activity EmpiricalActivity, declaredAssets map[string]struct{}, referencedAssets map[string]struct{}) {
	if len(declaredAssets) == 0 {
		return
	}

	var assetIdentityKey = strings.TrimSpace(activity.AssetIdentityKey)
	if _, assetExists := declaredAssets[assetIdentityKey]; !assetExists {
		validator.addIssue("case_id", caseID, "asset_identity_keys", "asset_identity_keys", fmt.Sprintf("case source_id %s uses asset_identity_key %s outside asset_identity_keys", sourceID, assetIdentityKey))
		return
	}

	referencedAssets[assetIdentityKey] = struct{}{}
}

// caseActivityOccursInYear reports whether one activity belongs to a selected report year.
// Authored by: OpenCode
func caseActivityOccursInYear(activity EmpiricalActivity, year int) bool {
	var activityYear, yearValid = empiricalActivityYear(activity)
	if !yearValid {
		return false
	}

	return activityYear == year
}

// validateCaseYearCoverage validates declared case year and selected-year activity coverage.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCaseYearCoverage(caseRecord *EmpiricalCase, caseID string, declaredAssets map[string]struct{}, referencedAssets map[string]struct{}, hasSelectedYearActivity bool) {
	if _, exists := validator.supportedYearsFromDatasetDeclaration[caseRecord.Year]; !exists {
		validator.addIssue("case_id", caseID, "year", "supported_years", fmt.Sprintf("case year %d is not declared in supported_years", caseRecord.Year))
	}
	for assetIdentityKey := range declaredAssets {
		if _, exists := referencedAssets[assetIdentityKey]; exists {
			continue
		}

		validator.addIssue("case_id", caseID, "asset_identity_keys", "asset_identity_keys", fmt.Sprintf("case asset_identity_key %s must be referenced by at least one activity_source_id", assetIdentityKey))
	}
	if len(caseRecord.ActivitySourceIDs) != 0 && !hasSelectedYearActivity {
		validator.addIssue("case_id", caseID, "year", "supported_years", "case must reference at least one activity_source_id that occurs in the selected year")
	}
}

// validateCaseOracleSupport enforces basic oracle support metadata consistency.
// Authored by: OpenCode
func (validator empiricalDatasetCaseValidator) validateCaseOracleSupport(caseRecord *EmpiricalCase, caseID string) {
	switch caseRecord.OracleSupport {
	case OracleSupportSupported:
		if strings.TrimSpace(caseRecord.UnsupportedReason) != "" {
			validator.addIssue("case_id", caseID, "unsupported_reason", "oracle_support", "supported cases must not declare unsupported_reason")
		}
	case OracleSupportPartiallySupported, OracleSupportUnsupported:
		if strings.TrimSpace(caseRecord.UnsupportedReason) == "" {
			validator.addIssue("case_id", caseID, "unsupported_reason", "oracle_support", "partially supported or unsupported cases must declare unsupported_reason")
		}
	default:
		validator.addIssue("case_id", caseID, "oracle_support", "oracle_support", "oracle_support must be supported, partially_supported, or unsupported")
	}
}
