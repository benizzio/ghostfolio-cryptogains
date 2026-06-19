// Package fixture contains parser cursor state, tokenized line metadata, and
// parse error formatting for the constrained empirical dataset YAML grammar.
//
// Authored by: OpenCode
package fixture

import (
	"strconv"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

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

// parse decodes the full constrained empirical dataset document.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parse() (EmpiricalDataset, error) {
	var dataset EmpiricalDataset

	for parser.index < len(parser.lines) {
		var err = parser.parseTopLevelLine(&dataset)
		if err != nil {
			return EmpiricalDataset{}, err
		}
	}

	return dataset, nil
}

// parseTopLevelLine decodes the field at the parser cursor into the dataset.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseTopLevelLine(dataset *EmpiricalDataset) error {
	var line = parser.lines[parser.index]
	var field, rawValue, err = parser.readTopLevelField(line)
	if err != nil {
		return err
	}

	return parser.applyTopLevelField(line, dataset, field, rawValue)
}

// readTopLevelField verifies and splits one top-level YAML field line.
// Authored by: OpenCode
func (parser *datasetYAMLParser) readTopLevelField(line datasetYAMLLine) (string, string, error) {
	if line.Indent != 0 {
		return "", "", parser.newError(line, "", "", "", "expected top-level field")
	}

	var field, rawValue, ok = splitYAMLField(line.Text)
	if !ok {
		return "", "", parser.newError(line, "", "", "", "expected YAML field")
	}

	return field, rawValue, nil
}

// applyTopLevelField dispatches one top-level dataset field to its parser.
// Authored by: OpenCode
func (parser *datasetYAMLParser) applyTopLevelField(line datasetYAMLLine, dataset *EmpiricalDataset, field string, rawValue string) error {
	var listParser = datasetYAMLListParser{cursor: parser}

	switch field {
	case "dataset_version":
		var value, err = parser.parseDatasetScalarField(line, field, rawValue)
		dataset.DatasetVersion = value
		return err
	case "description":
		var value, err = parser.parseDatasetScalarField(line, field, rawValue)
		dataset.Description = value
		return err
	case "currency":
		var value, err = parser.parseDatasetScalarField(line, field, rawValue)
		dataset.Currency = value
		return err
	case "supported_years":
		var values, err = listParser.parseIntegerList(line, rawValue, field, "", "")
		dataset.SupportedYears = values
		return err
	case "supported_methods":
		var values, err = parser.parseSupportedMethods(line, rawValue, field)
		dataset.SupportedMethods = values
		return err
	case "coverage_tags":
		var values, err = listParser.parseStringList(line, rawValue, field, "", "", false)
		dataset.CoverageTags = values
		return err
	case "activities":
		var activityParser = datasetYAMLActivityParser{cursor: parser}
		var values, err = activityParser.parseActivities(line, rawValue)
		dataset.Activities = values
		return err
	case "cases":
		var caseParser = datasetYAMLCaseParser{cursor: parser}
		var values, err = caseParser.parseCases(line, rawValue)
		dataset.Cases = values
		return err
	default:
		return parser.newError(line, "", "", field, "unknown top-level field")
	}
}

// parseDatasetScalarField decodes one top-level scalar dataset field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseDatasetScalarField(line datasetYAMLLine, field string, rawValue string) (string, error) {
	var value, err = parseYAMLScalarText(rawValue)
	if err != nil {
		return "", parser.newError(line, "", "", field, err.Error())
	}

	parser.index++
	return value, nil
}

// parseSupportedMethods decodes the dataset-level cost basis method list.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseSupportedMethods(line datasetYAMLLine, rawValue string, field string) ([]reportmodel.CostBasisMethod, error) {
	var listParser = datasetYAMLListParser{cursor: parser}
	var values, err = listParser.parseStringList(line, rawValue, field, "", "", false)
	if err != nil {
		return nil, err
	}

	return costBasisMethodsFromStrings(values), nil
}
