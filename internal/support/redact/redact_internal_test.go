// Package redact tests the secret-redaction helpers used by application-owned
// diagnostics so runtime errors and status text never leak token material.
// Authored by: OpenCode
package redact

import (
	"errors"
	"testing"
)

func TestTextSkipsEmptySecrets(t *testing.T) {
	t.Parallel()

	if got := Text("value", ""); got != "value" {
		t.Fatalf("unexpected redaction result: %q", got)
	}
}

func TestMaskEmptySecretReturnsEmptyString(t *testing.T) {
	t.Parallel()

	if got := Mask(""); got != "" {
		t.Fatalf("unexpected mask result: %q", got)
	}
}

func TestTextAndErrorTextRedactSecrets(t *testing.T) {
	t.Parallel()

	if got := Text("token=abc jwt=xyz", "abc", "xyz"); got != "token=[REDACTED] jwt=[REDACTED]" {
		t.Fatalf("unexpected redacted text: %q", got)
	}
	if got := ErrorText(errors.New("secret abc"), "abc"); got != "secret [REDACTED]" {
		t.Fatalf("unexpected redacted error text: %q", got)
	}
	if got := ErrorText(nil, "abc"); got != "" {
		t.Fatalf("expected empty error text, got %q", got)
	}
	if got := Mask("abc"); got != "[REDACTED]" {
		t.Fatalf("unexpected mask result: %q", got)
	}
}
