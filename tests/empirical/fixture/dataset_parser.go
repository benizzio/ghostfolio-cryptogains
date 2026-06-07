package fixture

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// LoadEmpiricalDataset reads one persisted empirical dataset file and parses it
// through the constrained project-owned YAML parser.
//
// Example:
//
//	dataset, rawContent, err := fixture.LoadEmpiricalDataset("testdata/empirical/financial-dataset.yaml")
//	if err != nil {
//		panic(err)
//	}
//	_, _ = dataset, rawContent
//
// Authored by: OpenCode
func LoadEmpiricalDataset(path string) (EmpiricalDataset, string, error) {
	var rawContent, err = os.ReadFile(path)
	if err != nil {
		return EmpiricalDataset{}, "", fmt.Errorf("read empirical dataset %s: %w", path, err)
	}

	var dataset EmpiricalDataset
	dataset, err = ParseEmpiricalDataset(path, rawContent)
	if err != nil {
		return EmpiricalDataset{}, "", err
	}

	return dataset, string(rawContent), nil
}

// ParseEmpiricalDataset parses one constrained project-owned YAML dataset fixture
// into the shared empirical dataset models.
//
// Example:
//
//	dataset, err := fixture.ParseEmpiricalDataset(path, rawContent)
//	if err != nil {
//		panic(err)
//	}
//	_ = dataset
//
// ParseEmpiricalDataset accepts only the repository-owned schema used by the
// empirical test suite. It rejects decimal fields that are not quoted strings.
// Authored by: OpenCode
func ParseEmpiricalDataset(path string, content []byte) (EmpiricalDataset, error) {
	var parser = newDatasetYAMLParser(path, string(content))
	return parser.parse()
}

// datasetYAMLLine stores one non-empty non-comment dataset line with its stable
// indentation and line number metadata.
// Authored by: OpenCode
type datasetYAMLLine struct {
	Number int
	Indent int
	Text   string
}

// datasetYAMLParser keeps the current parse cursor for the constrained empirical
// dataset YAML grammar.
// Authored by: OpenCode
type datasetYAMLParser struct {
	path  string
	index int
	lines []datasetYAMLLine
}

// datasetParserError reports one actionable parse failure without exposing more
// content than necessary.
// Authored by: OpenCode
type datasetParserError struct {
	Field          string
	Message        string
	Path           string
	ReferenceKind  string
	ReferenceValue string
	Line           int
}

// Error formats one constrained parser failure for test and loader callers.
// Authored by: OpenCode
func (parseErr datasetParserError) Error() string {
	var builder strings.Builder

	builder.WriteString(parseErr.Path)
	if parseErr.Line > 0 {
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(parseErr.Line))
	}
	if parseErr.ReferenceKind != "" && parseErr.ReferenceValue != "" {
		builder.WriteString(" ")
		builder.WriteString(parseErr.ReferenceKind)
		builder.WriteString(" ")
		builder.WriteString(parseErr.ReferenceValue)
	}
	if parseErr.Field != "" {
		builder.WriteString(" field ")
		builder.WriteString(parseErr.Field)
	}
	builder.WriteString(": ")
	builder.WriteString(parseErr.Message)

	return builder.String()
}

// newDatasetYAMLParser tokenizes one raw dataset payload into the stable line
// representation used by the constrained parser.
// Authored by: OpenCode
func newDatasetYAMLParser(path string, content string) datasetYAMLParser {
	var normalized = strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	var rawLines = strings.Split(normalized, "\n")
	var lines = make([]datasetYAMLLine, 0, len(rawLines))
	var index int

	for index = range rawLines {
		var rawLine = rawLines[index]
		var trimmed = strings.TrimSpace(rawLine)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		lines = append(lines, datasetYAMLLine{
			Number: index + 1,
			Indent: len(rawLine) - len(strings.TrimLeft(rawLine, " ")),
			Text:   trimmed,
		})
	}

	return datasetYAMLParser{path: path, lines: lines}
}

// parse decodes the full constrained empirical dataset document.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parse() (EmpiricalDataset, error) {
	var dataset EmpiricalDataset

	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent != 0 {
			return EmpiricalDataset{}, parser.newError(line, "", "", "", "expected top-level field")
		}

		var field, rawValue, ok = splitYAMLField(line.Text)
		if !ok {
			return EmpiricalDataset{}, parser.newError(line, "", "", "", "expected YAML field")
		}

		switch field {
		case "dataset_version":
			var value string
			var err error
			value, err = parseYAMLScalarText(rawValue)
			if err != nil {
				return EmpiricalDataset{}, parser.newError(line, "", "", field, err.Error())
			}
			dataset.DatasetVersion = value
			parser.index++
		case "description":
			var value string
			var err error
			value, err = parseYAMLScalarText(rawValue)
			if err != nil {
				return EmpiricalDataset{}, parser.newError(line, "", "", field, err.Error())
			}
			dataset.Description = value
			parser.index++
		case "currency":
			var value string
			var err error
			value, err = parseYAMLScalarText(rawValue)
			if err != nil {
				return EmpiricalDataset{}, parser.newError(line, "", "", field, err.Error())
			}
			dataset.Currency = value
			parser.index++
		case "supported_years":
			var values []int
			var err error
			values, err = parser.parseIntegerList(line, rawValue, field, "", "")
			if err != nil {
				return EmpiricalDataset{}, err
			}
			dataset.SupportedYears = values
		case "supported_methods":
			var values []string
			var err error
			values, err = parser.parseStringList(line, rawValue, field, "", "", false)
			if err != nil {
				return EmpiricalDataset{}, err
			}
			dataset.SupportedMethods = make([]reportmodel.CostBasisMethod, len(values))
			var index int
			for index = range values {
				dataset.SupportedMethods[index] = reportmodel.CostBasisMethod(values[index])
			}
		case "coverage_tags":
			var values []string
			var err error
			values, err = parser.parseStringList(line, rawValue, field, "", "", false)
			if err != nil {
				return EmpiricalDataset{}, err
			}
			dataset.CoverageTags = values
		case "activities":
			var values []EmpiricalActivity
			var err error
			values, err = parser.parseActivities(line, rawValue)
			if err != nil {
				return EmpiricalDataset{}, err
			}
			dataset.Activities = values
		case "cases":
			var values []EmpiricalCase
			var err error
			values, err = parser.parseCases(line, rawValue)
			if err != nil {
				return EmpiricalDataset{}, err
			}
			dataset.Cases = values
		default:
			return EmpiricalDataset{}, parser.newError(line, "", "", field, "unknown top-level field")
		}
	}

	return dataset, nil
}

// parseIntegerList parses one block or inline-empty YAML list of integer values.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseIntegerList(parentLine datasetYAMLLine, rawValue string, field string, referenceKind string, referenceValue string) ([]int, error) {
	var rawItems []string
	var err error

	rawItems, err = parser.parseStringList(parentLine, rawValue, field, referenceKind, referenceValue, false)
	if err != nil {
		return nil, err
	}

	var values = make([]int, 0, len(rawItems))
	var index int

	for index = range rawItems {
		var value int
		value, err = strconv.Atoi(rawItems[index])
		if err != nil {
			return nil, parser.newError(parentLine, referenceKind, referenceValue, field, "expected integer list value")
		}
		values = append(values, value)
	}

	return values, nil
}

// parseStringList parses one block or inline-empty YAML list of string scalars.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseStringList(parentLine datasetYAMLLine, rawValue string, field string, referenceKind string, referenceValue string, requireQuoted bool) ([]string, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "[]" {
		parser.index++
		return []string{}, nil
	}
	if trimmedValue != "" {
		return nil, parser.newError(parentLine, referenceKind, referenceValue, field, "expected block list or []")
	}

	parser.index++
	var values = make([]string, 0)

	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}
		if line.Indent != parentLine.Indent+2 || !strings.HasPrefix(line.Text, "- ") {
			return nil, parser.newError(line, referenceKind, referenceValue, field, "expected list item")
		}

		var rawItem = strings.TrimSpace(strings.TrimPrefix(line.Text, "- "))
		var value string
		var err error

		if requireQuoted {
			value, err = parseQuotedYAMLString(rawItem)
			if err != nil {
				return nil, parser.newError(line, referenceKind, referenceValue, field, err.Error())
			}
		} else {
			value, err = parseYAMLScalarText(rawItem)
			if err != nil {
				return nil, parser.newError(line, referenceKind, referenceValue, field, err.Error())
			}
		}

		values = append(values, value)
		parser.index++
	}

	return values, nil
}

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
		if err := parser.parseActivity(line, &activity); err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}

	return activities, nil
}

// parseActivity parses one empirical activity item.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseActivity(startLine datasetYAMLLine, activity *EmpiricalActivity) error {
	var firstField = strings.TrimSpace(strings.TrimPrefix(startLine.Text, "- "))
	var field, rawValue, ok = splitYAMLField(firstField)
	if !ok {
		return parser.newError(startLine, "", "", "activities", "expected activity field")
	}
	if err := parser.applyActivityField(startLine, activity, field, rawValue); err != nil {
		return err
	}

	parser.index++
	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < startLine.Indent+2 {
			break
		}
		if line.Indent != startLine.Indent+2 {
			return parser.newError(line, "source_id", activity.SourceID, "activities", "unexpected nested indentation")
		}

		field, rawValue, ok = splitYAMLField(line.Text)
		if !ok {
			return parser.newError(line, "source_id", activity.SourceID, "activities", "expected activity field")
		}

		switch field {
		case "source_scope":
			var scope *EmpiricalScope
			var err error
			scope, err = parser.parseScope(line, rawValue, activity.SourceID)
			if err != nil {
				return err
			}
			activity.SourceScope = scope
		case "coverage_tags":
			var values []string
			var err error
			values, err = parser.parseStringList(line, rawValue, field, "source_id", activity.SourceID, false)
			if err != nil {
				return err
			}
			activity.CoverageTags = values
		default:
			if err := parser.applyActivityField(line, activity, field, rawValue); err != nil {
				return err
			}
			parser.index++
		}
	}

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
	var scope EmpiricalScope

	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}
		if line.Indent != parentLine.Indent+2 {
			return nil, parser.newError(line, "source_id", sourceID, "source_scope", "unexpected nested indentation")
		}

		var field, nestedRawValue, ok = splitYAMLField(line.Text)
		if !ok {
			return nil, parser.newError(line, "source_id", sourceID, "source_scope", "expected scope field")
		}
		if err := parser.applyScopeField(line, &scope, field, nestedRawValue, sourceID); err != nil {
			return nil, err
		}
		parser.index++
	}

	return &scope, nil
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

// parseCases parses the top-level empirical case list.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCases(parentLine datasetYAMLLine, rawValue string) ([]EmpiricalCase, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "[]" {
		parser.index++
		return []EmpiricalCase{}, nil
	}
	if trimmedValue != "" {
		return nil, parser.newError(parentLine, "", "", "cases", "expected block list or []")
	}

	parser.index++
	var cases = make([]EmpiricalCase, 0)

	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}
		if line.Indent != parentLine.Indent+2 || !strings.HasPrefix(line.Text, "- ") {
			return nil, parser.newError(line, "", "", "cases", "expected case list item")
		}

		var caseRecord EmpiricalCase
		if err := parser.parseCase(line, &caseRecord); err != nil {
			return nil, err
		}
		cases = append(cases, caseRecord)
	}

	return cases, nil
}

// parseCase parses one empirical case item.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCase(startLine datasetYAMLLine, caseRecord *EmpiricalCase) error {
	var firstField = strings.TrimSpace(strings.TrimPrefix(startLine.Text, "- "))
	var field, rawValue, ok = splitYAMLField(firstField)
	if !ok {
		return parser.newError(startLine, "", "", "cases", "expected case field")
	}
	if err := parser.applyCaseField(startLine, caseRecord, field, rawValue); err != nil {
		return err
	}

	parser.index++
	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < startLine.Indent+2 {
			break
		}
		if line.Indent != startLine.Indent+2 {
			return parser.newError(line, "case_id", caseRecord.CaseID, "cases", "unexpected nested indentation")
		}

		field, rawValue, ok = splitYAMLField(line.Text)
		if !ok {
			return parser.newError(line, "case_id", caseRecord.CaseID, "cases", "expected case field")
		}

		switch field {
		case "methods":
			var values []string
			var err error
			values, err = parser.parseStringList(line, rawValue, field, "case_id", caseRecord.CaseID, false)
			if err != nil {
				return err
			}
			caseRecord.Methods = make([]reportmodel.CostBasisMethod, len(values))
			var index int
			for index = range values {
				caseRecord.Methods[index] = reportmodel.CostBasisMethod(values[index])
			}
		case "asset_identity_keys":
			var values []string
			var err error
			values, err = parser.parseStringList(line, rawValue, field, "case_id", caseRecord.CaseID, false)
			if err != nil {
				return err
			}
			caseRecord.AssetIdentityKeys = values
		case "activity_source_ids":
			var values []string
			var err error
			values, err = parser.parseStringList(line, rawValue, field, "case_id", caseRecord.CaseID, false)
			if err != nil {
				return err
			}
			caseRecord.ActivitySourceIDs = values
		case "coverage_tags":
			var values []string
			var err error
			values, err = parser.parseStringList(line, rawValue, field, "case_id", caseRecord.CaseID, false)
			if err != nil {
				return err
			}
			caseRecord.CoverageTags = values
		default:
			if err := parser.applyCaseField(line, caseRecord, field, rawValue); err != nil {
				return err
			}
			parser.index++
		}
	}

	return nil
}

// applyCaseField decodes one scalar empirical case field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) applyCaseField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string) error {
	var err error

	switch field {
	case "case_id":
		caseRecord.CaseID, err = parseYAMLScalarText(rawValue)
	case "description":
		caseRecord.Description, err = parseYAMLScalarText(rawValue)
	case "year":
		caseRecord.Year, err = parseYAMLInteger(rawValue)
		if err != nil {
			return parser.newError(line, "case_id", caseRecord.CaseID, field, err.Error())
		}
		return nil
	case "oracle_support":
		var value string
		value, err = parseYAMLScalarText(rawValue)
		caseRecord.OracleSupport = OracleSupport(value)
	case "unsupported_reason":
		caseRecord.UnsupportedReason, err = parseYAMLScalarText(rawValue)
	default:
		return parser.newError(line, "case_id", caseRecord.CaseID, field, "unknown case field")
	}

	if err != nil {
		return parser.newError(line, "case_id", caseRecord.CaseID, field, err.Error())
	}

	return nil
}

// newError builds one parser error with stable path and record context.
// Authored by: OpenCode
func (parser *datasetYAMLParser) newError(line datasetYAMLLine, referenceKind string, referenceValue string, field string, message string) error {
	return datasetParserError{
		Field:          field,
		Message:        message,
		Path:           parser.path,
		ReferenceKind:  referenceKind,
		ReferenceValue: referenceValue,
		Line:           line.Number,
	}
}

// splitYAMLField splits one constrained YAML field line into key and raw value.
// Authored by: OpenCode
func splitYAMLField(text string) (string, string, bool) {
	var field, rawValue, ok = strings.Cut(text, ":")
	if !ok {
		return "", "", false
	}

	return strings.TrimSpace(field), strings.TrimSpace(rawValue), true
}

// parseYAMLInteger parses one scalar YAML integer field.
// Authored by: OpenCode
func parseYAMLInteger(rawValue string) (int, error) {
	var value, err = parseYAMLScalarText(rawValue)
	if err != nil {
		return 0, err
	}

	var parsed, parseErr = strconv.Atoi(value)
	if parseErr != nil {
		return 0, fmt.Errorf("expected integer value")
	}

	return parsed, nil
}

// parseYAMLScalarText parses one quoted or bare scalar field into plain text.
// Authored by: OpenCode
func parseYAMLScalarText(rawValue string) (string, error) {
	var trimmed = strings.TrimSpace(rawValue)
	if trimmed == "" {
		return "", nil
	}

	if isQuotedYAMLScalar(trimmed) {
		return parseQuotedYAMLString(trimmed)
	}
	if looksLikeUnterminatedQuotedScalar(trimmed) {
		return "", fmt.Errorf("unterminated quoted string")
	}

	return trimmed, nil
}

// parseQuotedYAMLString parses one quoted YAML scalar and rejects bare values.
// Authored by: OpenCode
func parseQuotedYAMLString(rawValue string) (string, error) {
	var trimmed = strings.TrimSpace(rawValue)
	if !isQuotedYAMLScalar(trimmed) {
		return "", fmt.Errorf("expected quoted string value")
	}

	if strings.HasPrefix(trimmed, "\"") {
		var value, err = strconv.Unquote(trimmed)
		if err != nil {
			return "", fmt.Errorf("invalid quoted string value")
		}
		return value, nil
	}

	return strings.Trim(trimmed, "'"), nil
}

// isQuotedYAMLScalar reports whether one scalar uses matching single or double quotes.
// Authored by: OpenCode
func isQuotedYAMLScalar(value string) bool {
	return len(value) >= 2 && ((strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")))
}

// looksLikeUnterminatedQuotedScalar detects a scalar that starts with a quote but does not end with it.
// Authored by: OpenCode
func looksLikeUnterminatedQuotedScalar(value string) bool {
	return (strings.HasPrefix(value, "\"") && !strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && !strings.HasSuffix(value, "'"))
}
