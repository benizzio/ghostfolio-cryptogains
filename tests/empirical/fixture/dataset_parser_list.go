// Package fixture contains constrained integer and string list parsing for the
// empirical dataset YAML parser.
//
// Authored by: OpenCode
package fixture

import (
	"strconv"
	"strings"
)

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
	return parser.parseStringListItems(parentLine, field, referenceKind, referenceValue, requireQuoted)
}

// parseStringListItems parses block list items until indentation leaves the list.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseStringListItems(parentLine datasetYAMLLine, field string, referenceKind string, referenceValue string, requireQuoted bool) ([]string, error) {
	var values = make([]string, 0)

	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}

		var value, err = parser.parseStringListItem(line, parentLine, field, referenceKind, referenceValue, requireQuoted)
		if err != nil {
			return nil, err
		}

		values = append(values, value)
		parser.index++
	}

	return values, nil
}

// parseStringListItem parses one block list item line.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseStringListItem(line datasetYAMLLine, parentLine datasetYAMLLine, field string, referenceKind string, referenceValue string, requireQuoted bool) (string, error) {
	if line.Indent != parentLine.Indent+2 || !strings.HasPrefix(line.Text, "- ") {
		return "", parser.newError(line, referenceKind, referenceValue, field, "expected list item")
	}

	var rawItem = strings.TrimSpace(strings.TrimPrefix(line.Text, "- "))
	return parser.parseStringListItemValue(line, rawItem, field, referenceKind, referenceValue, requireQuoted)
}

// parseStringListItemValue decodes one raw string list item value.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseStringListItemValue(line datasetYAMLLine, rawItem string, field string, referenceKind string, referenceValue string, requireQuoted bool) (string, error) {
	if requireQuoted {
		var value, err = parseQuotedYAMLString(rawItem)
		if err != nil {
			return "", parser.newError(line, referenceKind, referenceValue, field, err.Error())
		}

		return value, nil
	}

	var value, err = parseYAMLScalarText(rawItem)
	if err != nil {
		return "", parser.newError(line, referenceKind, referenceValue, field, err.Error())
	}

	return value, nil
}
