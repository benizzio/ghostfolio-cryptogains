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
