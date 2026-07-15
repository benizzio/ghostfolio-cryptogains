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
	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
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

	// #nosec G304 -- path is returned by the test-owned diagnostic writer.
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

// TestProductionConversionFailureDiagnosticsRedactFinancialValues verifies that
// US3 conversion failures preserve non-secret activity lookup context while
// removing secrets and financial values from persisted production diagnostics.
// Authored by: OpenCode
func TestProductionConversionFailureDiagnosticsRedactFinancialValues(t *testing.T) {
	var rateService = &failingIntegrationCurrencyRates{failures: map[string]error{
		"EUR|USD|2024-07-15": errors.New("provider_unavailable: Federal Reserve H.10 failed with Bearer jwt-secret and token token-123 for amount 1000.25"),
	}}
	var harness = runtimeflow.NewRuntimeBackedFlowHarnessWithCurrencyRateService(t, t.TempDir(), runtimeflow.MustCloudSetupConfig(t), false, rateService)
	var token = "token-123"
	var cache = conversionFailureProtectedActivityCache(t, "redaction-eur-buy", currencyintegration.BaseCurrencyEUR, "2024-07-15T10:00:00Z")
	cache.Activities[0].Comment = "Bearer jwt-secret reusable verifier token-123"
	cache.Activities[0].Quantity = mustReportFlowDecimal(t, "42.123")
	cache.Activities[0].OrderGrossValue = reportFlowDecimalPointer(t, "1000.25")
	cache.Activities[0].OrderUnitPrice = reportFlowDecimalPointer(t, "1000.25")

	seedProtectedSnapshot(t, harness, token, cache)
	var unlock = harness.App.SyncService.UnlockSelectedServerSnapshot(context.Background(), harness.Config, token)
	if !unlock.ProtectedData.HasReadableSnapshot {
		t.Fatalf("expected readable snapshot after unlock, got %#v", unlock)
	}

	var request, requestErr = reportmodel.NewReportRequest(
		2024,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatMarkdown,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}
	var outcome = harness.App.ReportService.Generate(context.Background(), runtime.ReportGenerationRequest{
		Request:                 request,
		ServerOrigin:            harness.Config.ServerOrigin,
		AttemptID:               "attempt-conversion-redaction",
		ExplicitDevelopmentMode: false,
	})
	if outcome.Success || outcome.FailureReason != runtime.ReportFailureUnsupportedReportCalculation {
		t.Fatalf("expected conversion failure outcome, got %#v", outcome)
	}
	if !outcome.Diagnostic.Eligible || !outcome.Diagnostic.Request.RedactFinancialValues {
		t.Fatalf("expected production diagnostic eligibility with financial redaction, got %#v", outcome.Diagnostic)
	}

	var path, err = harness.App.SyncService.GenerateDiagnosticReport(context.Background(), outcome.Diagnostic.Request)
	if err != nil {
		t.Fatalf("generate conversion failure diagnostic report: %v", err)
	}
	testutil.AssertRegularFile(t, path)
	// #nosec G304 -- path is returned by the test-owned diagnostic writer.
	var raw, readErr = os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read conversion failure diagnostic report: %v", readErr)
	}
	var text = string(raw)
	for _, forbidden := range []string{"token-123", "jwt-secret", "1000.25", "42.123", "Bearer jwt-secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("expected production conversion diagnostics to exclude %q, got %q", forbidden, text)
		}
	}
	for _, expected := range []string{
		`"financial_values_redacted": true`,
		`"failure_category": "unsupported report calculation"`,
		`"source_id": "redaction-eur-buy"`,
		`"order_currency": "EUR"`,
		`provider_unavailable`,
		`EUR`,
		`USD`,
		`2024-07-15`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected production conversion diagnostics to contain %q, got %q", expected, text)
		}
	}
}
