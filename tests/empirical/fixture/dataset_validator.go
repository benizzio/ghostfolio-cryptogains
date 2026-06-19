package fixture

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

const (
	minimumEmpiricalDatasetActivityCount = 150
	minimumEmpiricalDatasetYearSpan      = 3
)

// ValidateEmpiricalDataset applies the structural empirical dataset contract to
// one already-parsed dataset and its persisted raw content.
//
// Example:
//
//	err := fixture.ValidateEmpiricalDataset(path, rawContent, dataset)
//	if err != nil {
//		panic(err)
//	}
//
// ValidateEmpiricalDataset enforces dataset-count, year-span, supported-method,
// deterministic-ordering, single-currency, zero-priced-reduction, scope, and
// synthetic-only rules. Method and edge-case coverage validation stays in
// `ValidateDatasetCoverage`.
// Authored by: OpenCode
func ValidateEmpiricalDataset(path string, rawContent string, dataset EmpiricalDataset) error {
	if err := ValidateSyntheticOnlyContent(path, rawContent); err != nil {
		return err
	}

	var validator = newEmpiricalDatasetValidator(path, dataset)
	validator.validate()

	if len(validator.issues) == 0 {
		return nil
	}

	return empiricalDatasetValidationError{Issues: validator.issues, Path: path}
}

// empiricalDatasetValidationIssue stores one actionable structural dataset
// validation failure.
// Authored by: OpenCode
type empiricalDatasetValidationIssue struct {
	Field          string
	Kind           string
	Message        string
	Path           string
	ReferenceKind  string
	ReferenceValue string
}

// Error formats one structural dataset validation issue.
// Authored by: OpenCode
func (issue empiricalDatasetValidationIssue) Error() string {
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

// empiricalDatasetValidationError groups structural dataset validation failures.
// Authored by: OpenCode
type empiricalDatasetValidationError struct {
	Issues []empiricalDatasetValidationIssue
	Path   string
}

// Error formats grouped structural dataset validation failures.
// Authored by: OpenCode
func (issueError empiricalDatasetValidationError) Error() string {
	var builder strings.Builder

	builder.WriteString(issueError.Path)
	builder.WriteString(" failed empirical dataset validation with ")
	builder.WriteString(strconv.Itoa(len(issueError.Issues)))
	builder.WriteString(" issue(s):")

	for _, issue := range issueError.Issues {
		builder.WriteString("\n- ")
		builder.WriteString(issue.Error())
	}

	return builder.String()
}

// empiricalDatasetValidator keeps the current structural validation state.
// Authored by: OpenCode
type empiricalDatasetValidator struct {
	activitiesBySourceID                 map[string]EmpiricalActivity
	dataset                              EmpiricalDataset
	hasReliableScope                     bool
	hasUnreliableOrUnavailableScope      bool
	issues                               []empiricalDatasetValidationIssue
	path                                 string
	seenActivityOrdering                 map[string]string
	seenActivitySourceIDs                map[string]struct{}
	seenCaseIDs                          map[string]struct{}
	seenSupportedMethods                 map[reportmodel.CostBasisMethod]struct{}
	supportedYearsFromActivities         map[int]struct{}
	supportedYearsFromDatasetDeclaration map[int]struct{}
}

// newEmpiricalDatasetValidator builds one structural validator for an already-parsed dataset.
// Authored by: OpenCode
func newEmpiricalDatasetValidator(path string, dataset EmpiricalDataset) empiricalDatasetValidator {
	return empiricalDatasetValidator{
		activitiesBySourceID:                 make(map[string]EmpiricalActivity, len(dataset.Activities)),
		dataset:                              dataset,
		issues:                               make([]empiricalDatasetValidationIssue, 0),
		path:                                 path,
		seenActivityOrdering:                 make(map[string]string, len(dataset.Activities)),
		seenActivitySourceIDs:                make(map[string]struct{}, len(dataset.Activities)),
		seenCaseIDs:                          make(map[string]struct{}, len(dataset.Cases)),
		seenSupportedMethods:                 make(map[reportmodel.CostBasisMethod]struct{}, len(dataset.SupportedMethods)),
		supportedYearsFromActivities:         make(map[int]struct{}, len(dataset.Activities)),
		supportedYearsFromDatasetDeclaration: make(map[int]struct{}, len(dataset.SupportedYears)),
	}
}

// validate runs the full structural dataset validation pass.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validate() {
	validator.validateDatasetVersion()
	validator.validateDatasetCount()
	validator.validateSupportedMethods()
	validator.validateDeclaredYears()
	validator.validateActivities()
	validator.validateYearSpan()
	validator.validateCases()
	validator.validateScopePresence()
}

// validateCases delegates case-specific integrity checks to the case validator.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateCases() {
	newEmpiricalDatasetCaseValidator(validator).validateCases()
}

// validateDatasetVersion enforces the required dataset schema version marker.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateDatasetVersion() {
	if strings.TrimSpace(validator.dataset.DatasetVersion) != "" {
		return
	}

	validator.addIssue("", "", "dataset_version", "required_field", "dataset_version must be non-empty")
}

// validateDatasetCount enforces the minimum empirical activity count.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateDatasetCount() {
	if len(validator.dataset.Activities) >= minimumEmpiricalDatasetActivityCount {
		return
	}

	validator.addIssue("", "", "activities", "activity_count", fmt.Sprintf("dataset must contain at least %d activities", minimumEmpiricalDatasetActivityCount))
}

// validateSupportedMethods enforces the required supported cost-basis methods.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateSupportedMethods() {
	var requiredMethods = reportmodel.SupportedCostBasisMethods()
	var method reportmodel.CostBasisMethod

	for _, method = range validator.dataset.SupportedMethods {
		validator.seenSupportedMethods[method] = struct{}{}
	}

	for _, method = range requiredMethods {
		if _, ok := validator.seenSupportedMethods[method]; ok {
			continue
		}

		validator.addIssue("", "", "supported_methods", "supported_methods", fmt.Sprintf("missing required supported method %s", method))
	}

	for _, method = range validator.dataset.SupportedMethods {
		if !isSupportedCostBasisMethod(method) {
			validator.addIssue("", "", "supported_methods", "supported_methods", fmt.Sprintf("unsupported cost basis method %s", method))
		}
	}
}

// validateDeclaredYears records the dataset-declared report years.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateDeclaredYears() {
	var year int

	for _, year = range validator.dataset.SupportedYears {
		validator.supportedYearsFromDatasetDeclaration[year] = struct{}{}
	}
	if len(validator.supportedYearsFromDatasetDeclaration) >= minimumEmpiricalDatasetYearSpan {
		return
	}

	validator.addIssue("", "", "supported_years", "year_span", fmt.Sprintf("dataset must declare at least %d source-calendar years", minimumEmpiricalDatasetYearSpan))
}

// validateActivities enforces activity-level structural rules.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateActivities() {
	var index int

	for index = range validator.dataset.Activities {
		validator.validateActivity(&validator.dataset.Activities[index])
	}
}

// validateActivity enforces one activity row.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateActivity(activity *EmpiricalActivity) {
	var referenceValue = strings.TrimSpace(activity.SourceID)

	validator.validateActivitySourceID(referenceValue)
	if referenceValue != "" {
		if _, exists := validator.activitiesBySourceID[referenceValue]; !exists {
			validator.activitiesBySourceID[referenceValue] = *activity
		}
	}
	validator.validateActivityOrdering(activity, referenceValue)
	validator.validateActivityBasics(activity, referenceValue)
	validator.validateActivityFinancialFields(activity, referenceValue)
	validator.validateActivityScope(activity, referenceValue)
}

// validateActivitySourceID enforces non-empty unique activity source identifiers.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateActivitySourceID(sourceID string) {
	if sourceID == "" {
		validator.addIssue("", "", "source_id", "deterministic_source_id", "activity source_id must be non-empty")
		return
	}
	if _, exists := validator.seenActivitySourceIDs[sourceID]; exists {
		validator.addIssue("source_id", sourceID, "source_id", "deterministic_source_id", "activity source_id must be unique and deterministic")
		return
	}

	validator.seenActivitySourceIDs[sourceID] = struct{}{}
}

// validateActivityOrdering enforces timestamp and deterministic ordering metadata.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateActivityOrdering(activity *EmpiricalActivity, sourceID string) {
	if activity.DeterministicOrder <= 0 {
		validator.addIssue("source_id", sourceID, "deterministic_order", "ordering_metadata", "deterministic_order must be greater than zero")
	}

	var occurredAt, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(activity.OccurredAt))
	if err != nil {
		validator.addIssue("source_id", sourceID, "occurred_at", "ordering_metadata", "occurred_at must use RFC3339 timestamp text with source offset")
		return
	}

	validator.supportedYearsFromActivities[occurredAt.Year()] = struct{}{}

	var orderingKey = activity.OccurredAt + "|" + activity.AssetIdentityKey + "|" + strconv.Itoa(activity.DeterministicOrder)
	if priorSourceID, exists := validator.seenActivityOrdering[orderingKey]; exists {
		validator.addIssue("source_id", sourceID, "deterministic_order", "ordering_metadata", fmt.Sprintf("ordering tuple collides with source_id %s", priorSourceID))
		return
	}

	validator.seenActivityOrdering[orderingKey] = sourceID
}

// validateActivityBasics enforces core activity enum and identifier fields.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateActivityBasics(activity *EmpiricalActivity, sourceID string) {
	if activity.ActivityType != syncmodel.ActivityTypeBuy && activity.ActivityType != syncmodel.ActivityTypeSell {
		validator.addIssue("source_id", sourceID, "activity_type", "activity_type", "activity_type must be BUY or SELL")
	}
	if strings.TrimSpace(activity.AssetIdentityKey) == "" {
		validator.addIssue("source_id", sourceID, "asset_identity_key", "asset_identity_key", "asset_identity_key must be non-empty")
	}
	if strings.TrimSpace(activity.AssetSymbol) == "" {
		validator.addIssue("source_id", sourceID, "asset_symbol", "asset_symbol", "asset_symbol must be non-empty")
	}

	validator.requirePositiveDecimal(sourceID, "quantity", activity.Quantity)
}

// validateActivityFinancialFields enforces priced-row and zero-priced-reduction rules.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateActivityFinancialFields(activity *EmpiricalActivity, sourceID string) {
	validator.requireOptionalDecimal(sourceID, "gross_value", activity.GrossValue)
	validator.requireOptionalDecimal(sourceID, "unit_price", activity.UnitPrice)
	validator.requireOptionalDecimal(sourceID, "fee_amount", activity.FeeAmount)

	if strings.TrimSpace(activity.ZeroPricedReductionExplanation) != "" {
		validator.validateZeroPricedReduction(activity, sourceID)
		return
	}
	if activity.ActivityType == syncmodel.ActivityTypeSell && strings.TrimSpace(activity.GrossValue) == "" && strings.TrimSpace(activity.UnitPrice) == "" {
		validator.addIssue("source_id", sourceID, "zero_priced_reduction_explanation", "pricing", "SELL activity rows must include gross_value and unit_price or declare zero_priced_reduction_explanation")
		return
	}

	validator.validatePricedCurrency(activity, sourceID)
}

// validateZeroPricedReduction enforces no-proceeds holding-reduction activity rules.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateZeroPricedReduction(activity *EmpiricalActivity, sourceID string) {
	if activity.ActivityType != syncmodel.ActivityTypeSell {
		validator.addIssue("source_id", sourceID, "zero_priced_reduction_explanation", "zero_priced_reduction", "zero-priced holding reductions must use SELL activity_type")
	}
	if strings.TrimSpace(activity.GrossValue) != "" || strings.TrimSpace(activity.UnitPrice) != "" || strings.TrimSpace(activity.FeeAmount) != "" || strings.TrimSpace(activity.Currency) != "" {
		validator.addIssue("source_id", sourceID, "zero_priced_reduction_explanation", "zero_priced_reduction", "zero-priced holding reductions must leave priced monetary fields and currency empty")
	}
}

// validatePricedCurrency enforces one-currency priced activity rules.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validatePricedCurrency(activity *EmpiricalActivity, sourceID string) {
	var expectsCurrency = activity.ActivityType == syncmodel.ActivityTypeBuy || strings.TrimSpace(activity.GrossValue) != "" || strings.TrimSpace(activity.UnitPrice) != "" || strings.TrimSpace(activity.FeeAmount) != "" || strings.TrimSpace(activity.Currency) != ""
	if !expectsCurrency {
		return
	}

	if strings.TrimSpace(activity.Currency) == "" {
		validator.addIssue("source_id", sourceID, "currency", "single_currency", "priced activity rows must declare the dataset currency")
		return
	}
	if strings.TrimSpace(validator.dataset.Currency) == "" {
		validator.addIssue("source_id", sourceID, "currency", "single_currency", "dataset currency must be declared before priced activity rows can be validated")
		return
	}
	if activity.Currency != validator.dataset.Currency {
		validator.addIssue("source_id", sourceID, "currency", "single_currency", fmt.Sprintf("priced activity currency %s does not match dataset currency %s", activity.Currency, validator.dataset.Currency))
	}
	if strings.TrimSpace(activity.GrossValue) == "" || strings.TrimSpace(activity.UnitPrice) == "" {
		validator.addIssue("source_id", sourceID, "currency", "single_currency", "priced activity rows must include gross_value and unit_price without cross-currency conversion")
	}
}

// validateActivityScope enforces scope metadata rules and records required scope diversity.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateActivityScope(activity *EmpiricalActivity, sourceID string) {
	if activity.SourceScope == nil {
		return
	}

	var scope = activity.SourceScope
	switch scope.Reliability {
	case syncmodel.ScopeReliabilityReliable:
		validator.hasReliableScope = true
		if strings.TrimSpace(scope.ScopeID) == "" {
			validator.addIssue("source_id", sourceID, "source_scope", "scopes", "reliable scope rows require non-empty scope_id")
		}
		if !isSupportedScopeKind(scope.ScopeKind) {
			validator.addIssue("source_id", sourceID, "source_scope", "scopes", "reliable scope rows require scope_kind account or wallet")
		}
	case syncmodel.ScopeReliabilityPartial, syncmodel.ScopeReliabilityUnavailable:
		validator.hasUnreliableOrUnavailableScope = true
		if scope.ScopeKind != "" && !isSupportedScopeKind(scope.ScopeKind) {
			validator.addIssue("source_id", sourceID, "source_scope", "scopes", "scope_kind must be account or wallet when present")
		}
	default:
		validator.addIssue("source_id", sourceID, "source_scope", "scopes", "scope reliability must be reliable, partial, or unavailable")
	}
}

// validateYearSpan enforces minimum multi-year activity coverage and declaration alignment.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateYearSpan() {
	if len(validator.supportedYearsFromActivities) < minimumEmpiricalDatasetYearSpan {
		validator.addIssue("", "", "supported_years", "year_span", fmt.Sprintf("activities must span at least %d source-calendar years", minimumEmpiricalDatasetYearSpan))
	}

	var declaredYears = sortedYearsFromSet(validator.supportedYearsFromDatasetDeclaration)
	var activityYears = sortedYearsFromSet(validator.supportedYearsFromActivities)
	if len(activityYears) == 0 {
		return
	}
	if !equalIntSlices(declaredYears, activityYears) {
		validator.addIssue("", "", "supported_years", "year_span", fmt.Sprintf("supported_years must match activity years %v", activityYears))
	}
}

// validateScopePresence enforces the required reliable and broadened-scope activity presence.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) validateScopePresence() {
	if !validator.hasReliableScope {
		validator.addIssue("", "", "source_scope", "scopes", "dataset must include at least one reliable scoped activity")
	}
	if !validator.hasUnreliableOrUnavailableScope {
		validator.addIssue("", "", "source_scope", "scopes", "dataset must include at least one partial or unavailable scoped activity")
	}
}

// requirePositiveDecimal enforces one required positive decimal string field.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) requirePositiveDecimal(sourceID string, field string, raw string) {
	var value, _, err = ParseDecimalString(raw)
	if err != nil {
		validator.addIssue("source_id", sourceID, field, field, err.Error())
		return
	}
	if value.Sign() <= 0 {
		validator.addIssue("source_id", sourceID, field, field, field+" must be greater than zero")
	}
}

// requireOptionalDecimal enforces one optional canonical decimal string field when present.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) requireOptionalDecimal(sourceID string, field string, raw string) {
	if strings.TrimSpace(raw) == "" {
		return
	}
	if _, _, err := ParseDecimalString(raw); err != nil {
		validator.addIssue("source_id", sourceID, field, field, err.Error())
	}
}

// addIssue records one structural dataset validation issue.
// Authored by: OpenCode
func (validator *empiricalDatasetValidator) addIssue(referenceKind string, referenceValue string, field string, kind string, message string) {
	validator.issues = append(validator.issues, empiricalDatasetValidationIssue{
		Field:          field,
		Kind:           kind,
		Message:        message,
		Path:           validator.path,
		ReferenceKind:  referenceKind,
		ReferenceValue: referenceValue,
	})
}

// equalIntSlices reports whether two sorted integer slices are identical.
// Authored by: OpenCode
func equalIntSlices(left []int, right []int) bool {
	if len(left) != len(right) {
		return false
	}

	var index int
	for index = range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

// isSupportedCostBasisMethod reports whether one method is part of the supported application set.
// Authored by: OpenCode
func isSupportedCostBasisMethod(method reportmodel.CostBasisMethod) bool {
	var supportedMethod reportmodel.CostBasisMethod

	for _, supportedMethod = range reportmodel.SupportedCostBasisMethods() {
		if supportedMethod == method {
			return true
		}
	}

	return false
}

// isSupportedScopeKind reports whether one scope kind matches the empirical dataset contract.
// Authored by: OpenCode
func isSupportedScopeKind(kind syncmodel.SourceScopeKind) bool {
	return kind == syncmodel.SourceScopeKindAccount || kind == syncmodel.SourceScopeKindWallet
}

// sortedYearsFromSet returns the stable ascending year slice for one set.
// Authored by: OpenCode
func sortedYearsFromSet(years map[int]struct{}) []int {
	var values = make([]int, 0, len(years))
	var year int

	for year = range years {
		values = append(values, year)
	}

	sort.Ints(values)
	return values
}

// empiricalActivityYear returns the source-calendar year for one activity when the timestamp is valid.
// Authored by: OpenCode
func empiricalActivityYear(activity EmpiricalActivity) (int, bool) {
	var occurredAt, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(activity.OccurredAt))
	if err != nil {
		return 0, false
	}

	return occurredAt.Year(), true
}

// caseHasCoverageTag reports whether one case declares the required coverage tag.
// Authored by: OpenCode
func caseHasCoverageTag(caseRecord EmpiricalCase, tag string) bool {
	var coverageTag string
	for _, coverageTag = range caseRecord.CoverageTags {
		if strings.TrimSpace(coverageTag) == strings.TrimSpace(tag) {
			return true
		}
	}

	return false
}
