// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Test seams wrap secure random reads so runtime tests can exercise identifier
// generation failures safely.
// Authored by: OpenCode
var readRandom = rand.Read

// randomIdentifier creates one opaque hexadecimal identifier.
// Authored by: OpenCode
func randomIdentifier(byteLength int) (string, error) {
	var rawValue = make([]byte, byteLength)
	if _, err := readRandom(rawValue); err != nil {
		return "", fmt.Errorf("read secure random bytes: %w", err)
	}

	return hex.EncodeToString(rawValue), nil
}
