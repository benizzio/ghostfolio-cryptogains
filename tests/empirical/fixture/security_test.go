package fixture

import (
	"errors"
	"strings"
	"testing"
)

// TestScanSyntheticOnlyContentRejectsSensitiveAndCopiedPatterns verifies the required synthetic-only scanner detections.
//
// Authored by: OpenCode
func TestScanSyntheticOnlyContentRejectsSensitiveAndCopiedPatterns(t *testing.T) {
	t.Parallel()

	var content = strings.Join([]string{
		`access_token: "tok_live_1234567890abcdefghijklmnop"`,
		`authorization: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTAwMSIsIm5hbWUiOiJKb2huIERvZSJ9.signaturevalue123"`,
		`owner_name: "John Doe"`,
		`2024-01-01 Opening Balances`,
		`Assets:Brokerage:BTC  1 BTC @ 100 USD`,
		`plugin "beancount.plugins.auto_accounts"`,
	}, "\n")

	var issues = ScanSyntheticOnlyContent("testdata/empirical/financial-dataset.yaml", content)

	if len(issues) != 7 {
		t.Fatalf("expected 7 issues, got %d", len(issues))
	}

	assertIssueKind(t, issues[0], violationTokenLikeValue, "access_token", 1)
	assertIssueKind(t, issues[1], violationBearerToken, "authorization", 2)
	assertIssueKind(t, issues[2], violationJWTLikeValue, "authorization", 2)
	assertIssueKind(t, issues[3], violationRealNameLike, "owner_name", 3)
	assertIssueKind(t, issues[4], violationCopiedFixture, "", 4)
	assertIssueKind(t, issues[5], violationCopiedFixture, "", 5)
	assertIssueKind(t, issues[6], violationCopiedFixture, "", 6)
}

// TestScanSyntheticOnlyContentAllowsReviewedSyntheticPlaceholders ensures obviously synthetic placeholders do not trip the heuristics.
//
// Authored by: OpenCode
func TestScanSyntheticOnlyContentAllowsReviewedSyntheticPlaceholders(t *testing.T) {
	t.Parallel()

	var content = strings.Join([]string{
		`access_token: "synthetic-demo-token"`,
		`display_name: "Synthetic Wallet Alpha"`,
		`description: "Synthetic empirical fixture for review"`,
		`note: "replace bearer-style values with synthetic placeholder text"`,
	}, "\n")

	var issues = ScanSyntheticOnlyContent("testdata/empirical/golden/fifo.json", content)

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d: %#v", len(issues), issues)
	}
}

// TestScanSyntheticOnlyContentRejectsGenericFreeTextSensitiveContent verifies generic prose fields still reject real names and token-like content.
//
// Authored by: OpenCode
func TestScanSyntheticOnlyContentRejectsGenericFreeTextSensitiveContent(t *testing.T) {
	t.Parallel()

	var content = strings.Join([]string{
		`description: "Reviewed against John Doe sample history"`,
		`note: "Temporary secret tok_live_1234567890abcdefghijklmnop must not persist"`,
	}, "\n")

	var issues = ScanSyntheticOnlyContent("testdata/empirical/financial-dataset.yaml", content)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d: %#v", len(issues), issues)
	}

	assertIssueKind(t, issues[0], violationRealNameLike, "description", 1)
	assertIssueKind(t, issues[1], violationTokenLikeValue, "note", 2)
}

// TestScanSyntheticOnlyContentAllowsGenericFreeTextSyntheticPlaceholders verifies generic prose fields do not reject clearly synthetic placeholders.
//
// Authored by: OpenCode
func TestScanSyntheticOnlyContentAllowsGenericFreeTextSyntheticPlaceholders(t *testing.T) {
	t.Parallel()

	var content = strings.Join([]string{
		`description: "Synthetic Placeholder User reviewed this fixture"`,
		`note: "Use synthetic_demo_token_0001 during oracle review"`,
	}, "\n")

	var issues = ScanSyntheticOnlyContent("testdata/empirical/golden/fifo.json", content)

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d: %#v", len(issues), issues)
	}
}

// TestValidateSyntheticOnlyContentReturnsGroupedNonSecretError verifies grouped validation output remains actionable and does not echo secret-like text.
//
// Authored by: OpenCode
func TestValidateSyntheticOnlyContentReturnsGroupedNonSecretError(t *testing.T) {
	t.Parallel()

	const jwtValue = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTAwMSIsIm5hbWUiOiJKb2huIERvZSJ9.signaturevalue123"
	var content = strings.Join([]string{
		`authorization: "Bearer ` + jwtValue + `"`,
		`owner_name: "John Doe"`,
	}, "\n")

	var err = ValidateSyntheticOnlyContent("testdata/empirical/golden/fifo/case-alpha.json", content)

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var syntheticContentError SyntheticContentError
	if !errors.As(err, &syntheticContentError) {
		t.Fatalf("expected SyntheticContentError, got %T", err)
	}

	var message = err.Error()

	if !strings.Contains(message, "testdata/empirical/golden/fifo/case-alpha.json failed synthetic-only content validation with 3 issue(s):") {
		t.Fatalf("expected grouped validation summary, got %q", message)
	}

	if !strings.Contains(message, "field authorization: bearer_token") {
		t.Fatalf("expected bearer-token issue in message, got %q", message)
	}

	if !strings.Contains(message, "field owner_name: real_name_like_value") {
		t.Fatalf("expected real-name issue in message, got %q", message)
	}

	if strings.Contains(message, jwtValue) {
		t.Fatalf("expected non-secret message, got %q", message)
	}

	if strings.Contains(message, "John Doe") {
		t.Fatalf("expected non-secret message, got %q", message)
	}

	if strings.Contains(message, "Bearer ") {
		t.Fatalf("expected non-secret message without raw bearer value, got %q", message)
	}
}

// TestSyntheticContentIssueErrorUsesStableFallbackLocation verifies empty locations use a deterministic label.
//
// Authored by: OpenCode
func TestSyntheticContentIssueErrorUsesStableFallbackLocation(t *testing.T) {
	t.Parallel()

	var issue = newSyntheticContentIssue(normalizeLocation(""), 7, "token", violationTokenLikeValue, "replace with synthetic placeholder text")

	if got, want := issue.Error(), "fixture content:7 field token: token_like_value: replace with synthetic placeholder text"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// assertIssueKind verifies the issue metadata expected by the scanner contract.
//
// Authored by: OpenCode
func assertIssueKind(t *testing.T, issue SyntheticContentIssue, kind string, field string, line int) {
	t.Helper()

	if issue.Kind != kind {
		t.Fatalf("expected issue kind %q, got %q", kind, issue.Kind)
	}

	if issue.Field != field {
		t.Fatalf("expected issue field %q, got %q", field, issue.Field)
	}

	if issue.Line != line {
		t.Fatalf("expected issue line %d, got %d", line, issue.Line)
	}

	if issue.Location == "" {
		t.Fatal("expected issue location to be set")
	}

	if issue.Message == "" {
		t.Fatal("expected issue message to be set")
	}
}
