// Package fixture contains activity, activity-field, and source-scope parsing
// for the constrained empirical dataset YAML parser.
//
// Authored by: OpenCode
package fixture

import (
	"strings"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// parseActivities parses the top-level empirical activity list.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivities(parentLine datasetYAMLLine, rawValue string) ([]EmpiricalActivity, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "[]" {
		parser.index++
		return []EmpiricalActivity{}, nil
	}
	if trimmedValue != "" {
		return nil, parser.newError(parentLine, "", "", "activities", "expected block list or []")
	}

	parser.index++
	return parser.parseActivityItems(parentLine)
}

// parseActivityItems parses all block activity items under a parent list field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivityItems(parentLine datasetYAMLLine) ([]EmpiricalActivity, error) {
	var activities = make([]EmpiricalActivity, 0)

	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}
		if line.Indent != parentLine.Indent+2 || !strings.HasPrefix(line.Text, "- ") {
			return nil, parser.newError(line, "", "", "activities", "expected activity list item")
		}

		var activity EmpiricalActivity
		var err = parser.parseActivity(line, &activity)
		if err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}

	return activities, nil
}

// parseActivity parses one empirical activity item.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivity(startLine datasetYAMLLine, activity *EmpiricalActivity) error {
	var err = parser.parseActivityFirstField(startLine, activity)
	if err != nil {
		return err
	}

	parser.index++
	return parser.parseActivityFields(startLine, activity)
}

// parseActivityFirstField decodes the activity field embedded in the list marker.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivityFirstField(startLine datasetYAMLLine, activity *EmpiricalActivity) error {
	var firstField = strings.TrimSpace(strings.TrimPrefix(startLine.Text, "- "))
	var field, rawValue, ok = splitYAMLField(firstField)
	if !ok {
		return parser.newError(startLine, "", "", "activities", "expected activity field")
	}

	return parser.applyActivityField(startLine, activity, field, rawValue)
}

// parseActivityFields parses all indented fields that belong to one activity.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivityFields(startLine datasetYAMLLine, activity *EmpiricalActivity) error {
	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < startLine.Indent+2 {
			break
		}

		var err = parser.parseActivityFieldLine(startLine, line, activity)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseActivityFieldLine decodes one nested activity field line.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivityFieldLine(startLine datasetYAMLLine, line datasetYAMLLine, activity *EmpiricalActivity) error {
	if line.Indent != startLine.Indent+2 {
		return parser.newError(line, "source_id", activity.SourceID, "activities", "unexpected nested indentation")
	}

	var field, rawValue, ok = splitYAMLField(line.Text)
	if !ok {
		return parser.newError(line, "source_id", activity.SourceID, "activities", "expected activity field")
	}

	switch field {
	case "source_scope":
		return parser.parseActivityScopeField(line, activity, rawValue)
	case "coverage_tags":
		return parser.parseActivityCoverageTagsField(line, activity, field, rawValue)
	default:
		return parser.parseActivityScalarField(line, activity, field, rawValue)
	}
}

// parseActivityScopeField decodes one nested source_scope activity field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivityScopeField(line datasetYAMLLine, activity *EmpiricalActivity, rawValue string) error {
	var scope, err = parser.parseScope(line, rawValue, activity.SourceID)
	if err != nil {
		return err
	}

	activity.SourceScope = scope
	return nil
}

// parseActivityCoverageTagsField decodes one activity coverage_tags list field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivityCoverageTagsField(line datasetYAMLLine, activity *EmpiricalActivity, field string, rawValue string) error {
	var values, err = parser.parseStringList(line, rawValue, field, "source_id", activity.SourceID, false)
	if err != nil {
		return err
	}

	activity.CoverageTags = values
	return nil
}

// parseActivityScalarField decodes one scalar activity field and advances the cursor.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivityScalarField(line datasetYAMLLine, activity *EmpiricalActivity, field string, rawValue string) error {
	var err = parser.applyActivityField(line, activity, field, rawValue)
	if err != nil {
		return err
	}

	parser.index++
	return nil
}

// applyActivityField decodes one scalar empirical activity field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) applyActivityField(line datasetYAMLLine, activity *EmpiricalActivity, field string, rawValue string) error {
	var err error

	switch field {
	case "source_id":
		activity.SourceID, err = parseYAMLScalarText(rawValue)
	case "occurred_at":
		activity.OccurredAt, err = parseYAMLScalarText(rawValue)
	case "deterministic_order":
		activity.DeterministicOrder, err = parseYAMLInteger(rawValue)
		if err != nil {
			return parser.newError(line, "source_id", activity.SourceID, field, err.Error())
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
		return parser.newError(line, "source_id", activity.SourceID, field, "unknown activity field")
	}

	if err != nil {
		return parser.newError(line, "source_id", activity.SourceID, field, err.Error())
	}

	return nil
}

// parseScope parses one nested empirical source scope map.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseScope(parentLine datasetYAMLLine, rawValue string, sourceID string) (*EmpiricalScope, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "{}" {
		parser.index++
		return &EmpiricalScope{}, nil
	}
	if trimmedValue != "" {
		return nil, parser.newError(parentLine, "source_id", sourceID, "source_scope", "expected nested mapping or {}")
	}

	parser.index++
	return parser.parseScopeFields(parentLine, sourceID)
}

// parseScopeFields parses all fields under one source_scope mapping.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseScopeFields(parentLine datasetYAMLLine, sourceID string) (*EmpiricalScope, error) {
	var scope EmpiricalScope

	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}

		var err = parser.parseScopeFieldLine(line, parentLine, &scope, sourceID)
		if err != nil {
			return nil, err
		}
		parser.index++
	}

	return &scope, nil
}

// parseScopeFieldLine decodes one source_scope field line.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseScopeFieldLine(line datasetYAMLLine, parentLine datasetYAMLLine, scope *EmpiricalScope, sourceID string) error {
	if line.Indent != parentLine.Indent+2 {
		return parser.newError(line, "source_id", sourceID, "source_scope", "unexpected nested indentation")
	}

	var field, nestedRawValue, ok = splitYAMLField(line.Text)
	if !ok {
		return parser.newError(line, "source_id", sourceID, "source_scope", "expected scope field")
	}

	return parser.applyScopeField(line, scope, field, nestedRawValue, sourceID)
}

// applyScopeField decodes one scalar empirical source scope field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) applyScopeField(line datasetYAMLLine, scope *EmpiricalScope, field string, rawValue string, sourceID string) error {
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
		return parser.newError(line, "source_id", sourceID, field, "unknown source_scope field")
	}

	if err != nil {
		return parser.newError(line, "source_id", sourceID, field, err.Error())
	}

	return nil
}
