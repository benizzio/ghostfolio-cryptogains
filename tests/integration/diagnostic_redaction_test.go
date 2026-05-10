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
