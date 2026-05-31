// Package integration verifies black-box workflow behavior for the current
// slice, including diagnostic redaction paths that must never expose secrets in
// user-visible output.
// Authored by: OpenCode
package integration

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

func TestApplicationOwnedDiagnosticsRedactSecrets(t *testing.T) {
	t.Parallel()

	var text = redact.ErrorText(errors.New("request body includes token abc123"), "abc123")
	if strings.Contains(text, "abc123") {
		t.Fatalf("expected secret to be redacted: %q", text)
	}
}

func TestGeneratedDiagnosticsRedactSecretBearingFailureCauseChain(t *testing.T) {
	t.Parallel()

	var service = runtime.NewSyncService(nil, 0, t.TempDir(), false, nil, nil, nil, nil)
	var path, err = service.GenerateDiagnosticReport(context.Background(), runtime.DiagnosticReportRequest{
		FailureReason: runtime.SyncFailureUnsupportedActivityHistory,
		ServerOrigin:  "https://ghostfol.io",
		Attempt: runtime.SyncAttempt{
			AttemptID:   "attempt-integration-redaction",
			Status:      runtime.AttemptStatusFailed,
			StartedAt:   time.Unix(1, 0).UTC(),
			CompletedAt: time.Unix(2, 0).UTC(),
		},
		Context: syncmodel.DiagnosticContext{
			FailureDetail:     "outer failure detail",
			FailureCauseChain: []string{"outer failure detail", "middle token abc123 layer", "Bearer jwt-secret"},
		},
	})
	if err != nil {
		t.Fatalf("generate diagnostic report: %v", err)
	}

	var raw, readErr = os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read diagnostic report: %v", readErr)
	}
	var text = string(raw)
	if strings.Contains(text, "abc123") || strings.Contains(text, "jwt-secret") {
		t.Fatalf("expected persisted cause chain to redact nested secrets, got %q", text)
	}
	for _, expected := range []string{
		`"failure_cause_chain": [`,
		`"outer failure detail"`,
		`"middle token [REDACTED] layer"`,
		`"Bearer [REDACTED]"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected persisted cause chain to contain %q, got %q", expected, text)
		}
	}
}
