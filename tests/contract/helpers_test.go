// Package contract verifies rendered workflow and Ghostfolio-boundary contracts
// for the sync-and-storage slice.
// Authored by: OpenCode
package contract

import (
	"strings"
	"testing"
)

// assertContains verifies that one rendered contract artifact includes the
// required text.
// Authored by: OpenCode
func assertContains(t *testing.T, content any, expected string) {
	t.Helper()
	if !strings.Contains(string(reportDocumentContent(content)), expected) {
		t.Fatalf("expected %q to contain %q", content, expected)
	}
}

// assertNotContains verifies that one rendered contract artifact excludes the
// forbidden text.
// Authored by: OpenCode
func assertNotContains(t *testing.T, content any, unexpected string) {
	t.Helper()
	if strings.Contains(string(reportDocumentContent(content)), unexpected) {
		t.Fatalf("expected %q not to contain %q", content, unexpected)
	}
}

// reportDocumentContent normalizes rendered report payloads for test assertions.
// Authored by: OpenCode
func reportDocumentContent(content any) []byte {
	switch value := content.(type) {
	case string:
		return []byte(value)
	case []byte:
		return value
	default:
		panic("unsupported report document content")
	}
}
