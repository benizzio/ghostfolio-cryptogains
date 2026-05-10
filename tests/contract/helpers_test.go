package contract

import (
	"strings"
	"testing"
)

func assertContains(t *testing.T, content string, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Fatalf("expected %q to contain %q", content, expected)
	}
}

func assertNotContains(t *testing.T, content string, unexpected string) {
	t.Helper()
	if strings.Contains(content, unexpected) {
		t.Fatalf("expected %q not to contain %q", content, unexpected)
	}
}
