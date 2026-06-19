// Package text provides small reusable predicates for working with plain text.
// Authored by: OpenCode
package text

import "strings"

// ContainsAll reports whether value contains every fragment in parts. Passing
// no parts returns true because no required fragment is missing.
//
// Example:
//
//	matched := text.ContainsAll("startup failed: missing token", "startup", "token")
//	_ = matched
//
// Authored by: OpenCode
func ContainsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}

	return true
}

// ContainsASCIILetter reports whether value contains at least one ASCII letter.
// Non-ASCII letters are ignored.
//
// Example:
//
//	matched := text.ContainsASCIILetter("token-abc123")
//	_ = matched
//
// Authored by: OpenCode
func ContainsASCIILetter(value string) bool {
	return strings.IndexFunc(value, func(character rune) bool {
		return character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
	}) >= 0
}

// ContainsASCIIDigit reports whether value contains at least one ASCII digit.
// Non-ASCII digits are ignored.
//
// Example:
//
//	matched := text.ContainsASCIIDigit("token-abc123")
//	_ = matched
//
// Authored by: OpenCode
func ContainsASCIIDigit(value string) bool {
	return strings.IndexFunc(value, func(character rune) bool {
		return character >= '0' && character <= '9'
	}) >= 0
}
