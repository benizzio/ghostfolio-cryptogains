package unit

import (
	"errors"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
)

func TestTextRedactsKnownSecrets(t *testing.T) {
	t.Parallel()

	var redacted = redact.Text("token=abc123 jwt=secret", "abc123", "secret")
	if strings.Contains(redacted, "abc123") || strings.Contains(redacted, "secret") {
		t.Fatalf("secret was not redacted: %q", redacted)
	}
}

func TestErrorTextHandlesNilError(t *testing.T) {
	t.Parallel()

	if got := redact.ErrorText(nil, "secret"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestMaskReturnsMarkerForNonEmptySecret(t *testing.T) {
	t.Parallel()

	if got := redact.Mask("secret"); got != "[REDACTED]" {
		t.Fatalf("mask mismatch: %q", got)
	}
}

func TestErrorTextRedactsErrorText(t *testing.T) {
	t.Parallel()

	var message = redact.ErrorText(errors.New("token secret leaked"), "secret")
	if strings.Contains(message, "secret") {
		t.Fatalf("error text still contains secret: %q", message)
	}
}
