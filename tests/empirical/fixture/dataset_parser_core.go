// Package fixture contains parser cursor state, tokenized line metadata, and
// parse error formatting for the constrained empirical dataset YAML grammar.
//
// Authored by: OpenCode
package fixture

import (
	"strconv"
	"strings"
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
