// Package fixture contains constrained integer and string list parsing for the
// empirical dataset YAML parser.
//
// Authored by: OpenCode
package fixture

import (
	"strconv"
	"strings"
)

// datasetYAMLListParser owns constrained scalar list parsing while sharing the
// parent parser cursor.
// Authored by: OpenCode
type datasetYAMLListParser struct {
	cursor *datasetYAMLParser
}

// parseIntegerList parses one block or inline-empty YAML list of integer values.
// Authored by: OpenCode
func (listParser *datasetYAMLListParser) parseIntegerList(parentLine datasetYAMLLine, rawValue string, field string, referenceKind string, referenceValue string) ([]int, error) {
	var rawItems []string
	var err error

	rawItems, err = listParser.parseStringList(parentLine, rawValue, field, referenceKind, referenceValue, false)
	if err != nil {
		return nil, err
	}

	var values = make([]int, 0, len(rawItems))
	var index int

	for index = range rawItems {
		var value int
		value, err = strconv.Atoi(rawItems[index])
		if err != nil {
			return nil, listParser.cursor.newError(parentLine, referenceKind, referenceValue, field, "expected integer list value")
		}
		values = append(values, value)
	}

	return values, nil
}

// parseStringList parses one block or inline-empty YAML list of string scalars.
// Authored by: OpenCode
func (listParser *datasetYAMLListParser) parseStringList(parentLine datasetYAMLLine, rawValue string, field string, referenceKind string, referenceValue string, requireQuoted bool) ([]string, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "[]" {
		listParser.cursor.index++
		return []string{}, nil
	}
	if trimmedValue != "" {
		return nil, listParser.cursor.newError(parentLine, referenceKind, referenceValue, field, "expected block list or []")
	}

	listParser.cursor.index++
	return listParser.parseStringListItems(parentLine, field, referenceKind, referenceValue, requireQuoted)
}

// parseStringListItems parses block list items until indentation leaves the list.
// Authored by: OpenCode
func (listParser *datasetYAMLListParser) parseStringListItems(parentLine datasetYAMLLine, field string, referenceKind string, referenceValue string, requireQuoted bool) ([]string, error) {
	var values = make([]string, 0)

	for listParser.cursor.index < len(listParser.cursor.lines) {
		var line = listParser.cursor.lines[listParser.cursor.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}

		var value, err = listParser.parseStringListItem(line, parentLine, field, referenceKind, referenceValue, requireQuoted)
		if err != nil {
			return nil, err
		}

		values = append(values, value)
		listParser.cursor.index++
	}

	return values, nil
}

// parseStringListItem parses one block list item line.
// Authored by: OpenCode
func (listParser *datasetYAMLListParser) parseStringListItem(line datasetYAMLLine, parentLine datasetYAMLLine, field string, referenceKind string, referenceValue string, requireQuoted bool) (string, error) {
	if line.Indent != parentLine.Indent+2 || !strings.HasPrefix(line.Text, "- ") {
		return "", listParser.cursor.newError(line, referenceKind, referenceValue, field, "expected list item")
	}

	var rawItem = strings.TrimSpace(strings.TrimPrefix(line.Text, "- "))
	return listParser.parseStringListItemValue(line, rawItem, field, referenceKind, referenceValue, requireQuoted)
}

// parseStringListItemValue decodes one raw string list item value.
// Authored by: OpenCode
func (listParser *datasetYAMLListParser) parseStringListItemValue(line datasetYAMLLine, rawItem string, field string, referenceKind string, referenceValue string, requireQuoted bool) (string, error) {
	if requireQuoted {
		var value, err = parseQuotedYAMLString(rawItem)
		if err != nil {
			return "", listParser.cursor.newError(line, referenceKind, referenceValue, field, err.Error())
		}

		return value, nil
	}

	var value, err = parseYAMLScalarText(rawItem)
	if err != nil {
		return "", listParser.cursor.newError(line, referenceKind, referenceValue, field, err.Error())
	}

	return value, nil
}
