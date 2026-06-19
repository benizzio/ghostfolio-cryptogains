// Package fixture contains constrained YAML field, integer, and scalar helpers
// used by the empirical dataset parser.
//
// Authored by: OpenCode
package fixture

import (
	"fmt"
	"strconv"
	"strings"
)

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
