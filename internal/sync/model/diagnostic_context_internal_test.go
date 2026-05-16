package model

import (
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

func TestDiagnosticRecordFromActivityRecordCanonicalizesAndPreservesScope(t *testing.T) {
	t.Parallel()

	quantity, _, err := decimalsupport.ParseString("001.2300")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	unitPrice, _, err := decimalsupport.ParseString("100.5000")
	if err != nil {
		t.Fatalf("parse unit price: %v", err)
	}
	grossValue, _, err := decimalsupport.ParseString("123.6150")
	if err != nil {
		t.Fatalf("parse gross value: %v", err)
	}
	feeAmount, _, err := decimalsupport.ParseString("0.1000")
	if err != nil {
		t.Fatalf("parse fee amount: %v", err)
	}

	record := DiagnosticRecordFromActivityRecord(ActivityRecord{
		SourceID:     "activity-1",
		OccurredAt:   "2024-01-01T10:00:00Z",
		ActivityType: ActivityTypeBuy,
		AssetSymbol:  "BTC",
		AssetName:    "Bitcoin",
		BaseCurrency: "USD",
		Quantity:     quantity,
		UnitPrice:    unitPrice,
		GrossValue:   grossValue,
		FeeAmount:    &feeAmount,
		Comment:      "comment",
		DataSource:   "ghostfolio",
		SourceScope: &SourceScope{
			ID:          "account-1",
			Name:        "Main Account",
			Kind:        SourceScopeKindAccount,
			Reliability: ScopeReliabilityReliable,
		},
	})

	if record.Quantity != "1.23" || record.UnitPrice != "100.5" || record.GrossValue != "123.615" || record.FeeAmount != "0.1" {
		t.Fatalf("unexpected canonical diagnostic values: %#v", record)
	}
	if record.SourceScopeID != "account-1" || record.SourceScopeName != "Main Account" || record.SourceScopeKind != string(SourceScopeKindAccount) || record.SourceScopeReliability != string(ScopeReliabilityReliable) {
		t.Fatalf("unexpected scope diagnostic context: %#v", record)
	}
}

func TestCanonicalDiagnosticDecimalFallbackBranches(t *testing.T) {
	t.Parallel()

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	if got := canonicalDiagnosticDecimal(invalid); got != invalid.String() {
		t.Fatalf("expected decimal fallback string, got %q want %q", got, invalid.String())
	}
	if got := canonicalDiagnosticDecimalPointer(nil); got != "" {
		t.Fatalf("expected nil pointer to return empty string, got %q", got)
	}
	if got := canonicalDiagnosticDecimalPointer(&invalid); got != invalid.String() {
		t.Fatalf("expected pointer fallback string, got %q want %q", got, invalid.String())
	}
}
