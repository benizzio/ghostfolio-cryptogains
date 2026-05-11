// Package integration verifies black-box workflow behavior for the current
// slice, including diagnostic redaction paths that must never expose secrets in
// user-visible output.
// Authored by: OpenCode
package integration

import (
	"errors"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
)

func TestApplicationOwnedDiagnosticsRedactSecrets(t *testing.T) {
	t.Parallel()

	var text = redact.ErrorText(errors.New("request body includes token abc123"), "abc123")
	if strings.Contains(text, "abc123") {
		t.Fatalf("expected secret to be redacted: %q", text)
	}
}
