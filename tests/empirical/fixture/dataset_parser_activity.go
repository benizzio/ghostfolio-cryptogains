// Package fixture contains activity, activity-field, and source-scope parsing
// for the constrained empirical dataset YAML parser.
//
// Authored by: OpenCode
package fixture

import (
	"strings"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// datasetYAMLActivityParser owns activity and source-scope parsing while sharing
// the parent parser cursor.
// Authored by: OpenCode
type datasetYAMLActivityParser struct {
	cursor *datasetYAMLParser
}

// parseActivities parses the top-level empirical activity list.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivities(parentLine datasetYAMLLine, rawValue string) ([]EmpiricalActivity, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "[]" {
		activityParser.cursor.index++
		return []EmpiricalActivity{}, nil
	}
	if trimmedValue != "" {
		return nil, activityParser.cursor.newError(parentLine, "", "", "activities", "expected block list or []")
	}

	activityParser.cursor.index++
	return activityParser.parseActivityItems(parentLine)
}

// parseActivityItems parses all block activity items under a parent list field.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivityItems(parentLine datasetYAMLLine) ([]EmpiricalActivity, error) {
	var activities = make([]EmpiricalActivity, 0)

	for activityParser.cursor.index < len(activityParser.cursor.lines) {
		var line = activityParser.cursor.lines[activityParser.cursor.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}
		if line.Indent != parentLine.Indent+2 || !strings.HasPrefix(line.Text, "- ") {
			return nil, activityParser.cursor.newError(line, "", "", "activities", "expected activity list item")
		}

		var activity EmpiricalActivity
		var err = activityParser.parseActivity(line, &activity)
		if err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}

	return activities, nil
}

// parseActivity parses one empirical activity item.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivity(startLine datasetYAMLLine, activity *EmpiricalActivity) error {
	var err = activityParser.parseActivityFirstField(startLine, activity)
	if err != nil {
		return err
	}

	activityParser.cursor.index++
	return activityParser.parseActivityFields(startLine, activity)
}

// parseActivityFirstField decodes the activity field embedded in the list marker.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivityFirstField(startLine datasetYAMLLine, activity *EmpiricalActivity) error {
	var firstField = strings.TrimSpace(strings.TrimPrefix(startLine.Text, "- "))
	var field, rawValue, ok = splitYAMLField(firstField)
	if !ok {
		return activityParser.cursor.newError(startLine, "", "", "activities", "expected activity field")
	}

	return activityParser.applyActivityField(startLine, activity, field, rawValue)
}

// parseActivityFields parses all indented fields that belong to one activity.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivityFields(startLine datasetYAMLLine, activity *EmpiricalActivity) error {
	for activityParser.cursor.index < len(activityParser.cursor.lines) {
		var line = activityParser.cursor.lines[activityParser.cursor.index]
		if line.Indent < startLine.Indent+2 {
			break
		}

		var err = activityParser.parseActivityFieldLine(startLine, line, activity)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseActivityFieldLine decodes one nested activity field line.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivityFieldLine(startLine datasetYAMLLine, line datasetYAMLLine, activity *EmpiricalActivity) error {
	if line.Indent != startLine.Indent+2 {
		return activityParser.cursor.newError(line, "source_id", activity.SourceID, "activities", "unexpected nested indentation")
	}

	var field, rawValue, ok = splitYAMLField(line.Text)
	if !ok {
		return activityParser.cursor.newError(line, "source_id", activity.SourceID, "activities", "expected activity field")
	}

	switch field {
	case "source_scope":
		return activityParser.parseActivityScopeField(line, activity, rawValue)
	case "coverage_tags":
		return activityParser.parseActivityCoverageTagsField(line, activity, field, rawValue)
	default:
		return activityParser.parseActivityScalarField(line, activity, field, rawValue)
	}
}

// parseActivityScopeField decodes one nested source_scope activity field.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivityScopeField(line datasetYAMLLine, activity *EmpiricalActivity, rawValue string) error {
	var scope, err = activityParser.parseScope(line, rawValue, activity.SourceID)
	if err != nil {
		return err
	}

	activity.SourceScope = scope
	return nil
}

// parseActivityCoverageTagsField decodes one activity coverage_tags list field.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivityCoverageTagsField(line datasetYAMLLine, activity *EmpiricalActivity, field string, rawValue string) error {
	var listParser = datasetYAMLListParser{cursor: activityParser.cursor}
	var values, err = listParser.parseStringList(line, rawValue, field, "source_id", activity.SourceID, false)
	if err != nil {
		return err
	}

	activity.CoverageTags = values
	return nil
}

// parseActivityScalarField decodes one scalar activity field and advances the cursor.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseActivityScalarField(line datasetYAMLLine, activity *EmpiricalActivity, field string, rawValue string) error {
	var err = activityParser.applyActivityField(line, activity, field, rawValue)
	if err != nil {
		return err
	}

	activityParser.cursor.index++
	return nil
}

// applyActivityField decodes one scalar empirical activity field.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) applyActivityField(line datasetYAMLLine, activity *EmpiricalActivity, field string, rawValue string) error {
	var err error

	switch field {
	case "source_id":
		activity.SourceID, err = parseYAMLScalarText(rawValue)
	case "occurred_at":
		activity.OccurredAt, err = parseYAMLScalarText(rawValue)
	case "deterministic_order":
		activity.DeterministicOrder, err = parseYAMLInteger(rawValue)
		if err != nil {
			return activityParser.cursor.newError(line, "source_id", activity.SourceID, field, err.Error())
		}
		return nil
	case "activity_type":
		var value string
		value, err = parseYAMLScalarText(rawValue)
		activity.ActivityType = syncmodel.ActivityType(value)
	case "asset_identity_key":
		activity.AssetIdentityKey, err = parseYAMLScalarText(rawValue)
	case "asset_symbol":
		activity.AssetSymbol, err = parseYAMLScalarText(rawValue)
	case "quantity":
		activity.Quantity, err = parseQuotedYAMLString(rawValue)
	case "gross_value":
		activity.GrossValue, err = parseQuotedYAMLString(rawValue)
	case "unit_price":
		activity.UnitPrice, err = parseQuotedYAMLString(rawValue)
	case "fee_amount":
		activity.FeeAmount, err = parseQuotedYAMLString(rawValue)
	case "currency":
		activity.Currency, err = parseYAMLScalarText(rawValue)
	case "zero_priced_reduction_explanation":
		activity.ZeroPricedReductionExplanation, err = parseYAMLScalarText(rawValue)
	default:
		return activityParser.cursor.newError(line, "source_id", activity.SourceID, field, "unknown activity field")
	}

	if err != nil {
		return activityParser.cursor.newError(line, "source_id", activity.SourceID, field, err.Error())
	}

	return nil
}

// parseScope parses one nested empirical source scope map.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseScope(parentLine datasetYAMLLine, rawValue string, sourceID string) (*EmpiricalScope, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "{}" {
		activityParser.cursor.index++
		return &EmpiricalScope{}, nil
	}
	if trimmedValue != "" {
		return nil, activityParser.cursor.newError(parentLine, "source_id", sourceID, "source_scope", "expected nested mapping or {}")
	}

	activityParser.cursor.index++
	return activityParser.parseScopeFields(parentLine, sourceID)
}

// parseScopeFields parses all fields under one source_scope mapping.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseScopeFields(parentLine datasetYAMLLine, sourceID string) (*EmpiricalScope, error) {
	var scope EmpiricalScope

	for activityParser.cursor.index < len(activityParser.cursor.lines) {
		var line = activityParser.cursor.lines[activityParser.cursor.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}

		var err = activityParser.parseScopeFieldLine(line, parentLine, &scope, sourceID)
		if err != nil {
			return nil, err
		}
		activityParser.cursor.index++
	}

	return &scope, nil
}

// parseScopeFieldLine decodes one source_scope field line.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) parseScopeFieldLine(line datasetYAMLLine, parentLine datasetYAMLLine, scope *EmpiricalScope, sourceID string) error {
	if line.Indent != parentLine.Indent+2 {
		return activityParser.cursor.newError(line, "source_id", sourceID, "source_scope", "unexpected nested indentation")
	}

	var field, nestedRawValue, ok = splitYAMLField(line.Text)
	if !ok {
		return activityParser.cursor.newError(line, "source_id", sourceID, "source_scope", "expected scope field")
	}

	return activityParser.applyScopeField(line, scope, field, nestedRawValue, sourceID)
}

// applyScopeField decodes one scalar empirical source scope field.
// Authored by: OpenCode
func (activityParser *datasetYAMLActivityParser) applyScopeField(line datasetYAMLLine, scope *EmpiricalScope, field string, rawValue string, sourceID string) error {
	var err error

	switch field {
	case "scope_id":
		scope.ScopeID, err = parseYAMLScalarText(rawValue)
	case "scope_kind":
		var value string
		value, err = parseYAMLScalarText(rawValue)
		scope.ScopeKind = syncmodel.SourceScopeKind(value)
	case "reliability":
		var value string
		value, err = parseYAMLScalarText(rawValue)
		scope.Reliability = syncmodel.ScopeReliability(value)
	case "display_name":
		scope.DisplayName, err = parseYAMLScalarText(rawValue)
	default:
		return activityParser.cursor.newError(line, "source_id", sourceID, field, "unknown source_scope field")
	}

	if err != nil {
		return activityParser.cursor.newError(line, "source_id", sourceID, field, err.Error())
	}

	return nil
}
