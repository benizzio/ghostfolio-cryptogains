package model

import (
	"encoding/json"
	"strings"
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

	record := DiagnosticRecordFromActivityRecord(
		ActivityRecord{
			SourceID:              "activity-1",
			OccurredAt:            "2024-01-01T10:00:00Z",
			ActivityType:          ActivityTypeBuy,
			AssetSymbol:           "BTC",
			AssetName:             "Bitcoin",
			OrderCurrency:         "CHF",
			AssetProfileCurrency:  "EUR",
			BaseCurrency:          "USD",
			Quantity:              quantity,
			OrderUnitPrice:        &unitPrice,
			OrderGrossValue:       &grossValue,
			OrderFeeAmount:        &feeAmount,
			AssetProfileUnitPrice: &unitPrice,
			AssetProfileFeeAmount: &feeAmount,
			BaseGrossValue:        &grossValue,
			BaseFeeAmount:         &feeAmount,
			Comment:               "comment",
			DataSource:            "ghostfolio",
			SourceScope: &SourceScope{
				ID:          "account-1",
				Name:        "Main Account",
				Kind:        SourceScopeKindAccount,
				Reliability: ScopeReliabilityReliable,
			},
		},
	)

	if record.Quantity != "1.23" || record.OrderUnitPrice != "100.5" || record.OrderGrossValue != "123.615" || record.OrderFeeAmount != "0.1" {
		t.Fatalf("unexpected canonical diagnostic values: %#v", record)
	}
	if record.OrderCurrency != "CHF" || record.AssetProfileCurrency != "EUR" || record.BaseCurrency != "USD" {
		t.Fatalf("unexpected currency context: %#v", record)
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

func TestDiagnosticRecordFromActivityRecordPreservesSourceAmounts(t *testing.T) {
	t.Parallel()

	quantity, _, err := decimalsupport.ParseString("1")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	orderUnitPrice, _, err := decimalsupport.ParseString("90")
	if err != nil {
		t.Fatalf("parse order unit price: %v", err)
	}
	assetProfileUnitPrice, _, err := decimalsupport.ParseString("95")
	if err != nil {
		t.Fatalf("parse asset-profile unit price: %v", err)
	}
	baseGrossValue, _, err := decimalsupport.ParseString("100")
	if err != nil {
		t.Fatalf("parse base gross value: %v", err)
	}

	record := DiagnosticRecordFromActivityRecord(
		ActivityRecord{
			SourceID:              "activity-1",
			OccurredAt:            "2024-01-01T10:00:00Z",
			ActivityType:          ActivityTypeBuy,
			AssetSymbol:           "BTC",
			Quantity:              quantity,
			OrderCurrency:         "",
			AssetProfileCurrency:  "EUR",
			BaseCurrency:          "USD",
			OrderUnitPrice:        &orderUnitPrice,
			AssetProfileUnitPrice: &assetProfileUnitPrice,
			BaseGrossValue:        &baseGrossValue,
		},
	)

	if record.OrderUnitPrice != "90" || record.AssetProfileUnitPrice != "95" || record.BaseGrossValue != "100" {
		t.Fatalf("expected diagnostics to preserve source amount tiers, got %#v", record)
	}
}

func TestDiagnosticRecordMarshalJSONRendersExplicitNullFields(t *testing.T) {
	t.Parallel()

	var raw, err = json.Marshal(DiagnosticRecord{SourceID: "activity-1"})
	if err != nil {
		t.Fatalf("marshal diagnostic record: %v", err)
	}
	var text = string(raw)
	if !strings.Contains(text, `"source_id":"activity-1"`) {
		t.Fatalf("expected source_id to be preserved, got %q", text)
	}
	for _, expected := range []string{
		`"order_currency":null`,
		`"asset_profile_currency":null`,
		`"base_currency":null`,
		`"quantity":null`,
		`"source_scope_id":null`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected explicit null field %q, got %q", expected, text)
		}
	}
}

func TestDiagnosticRecordUnmarshalJSONHandlesErrorAndNullFields(t *testing.T) {
	t.Parallel()

	var record DiagnosticRecord
	if err := json.Unmarshal([]byte(`{"source_id":`), &record); err == nil {
		t.Fatalf("expected invalid diagnostic record JSON to fail")
	}
	if err := record.UnmarshalJSON([]byte(`true`)); err == nil {
		t.Fatalf("expected non-object diagnostic record JSON to fail typed unmarshal")
	}

	if err := json.Unmarshal([]byte(`{"source_id":"activity-1","order_currency":null,"source_scope_id":null}`), &record); err != nil {
		t.Fatalf("unmarshal diagnostic record with explicit nulls: %v", err)
	}
	if record.SourceID != "activity-1" {
		t.Fatalf("expected source id to be restored, got %#v", record)
	}
	if record.OrderCurrency != "" || record.SourceScopeID != "" {
		t.Fatalf("expected explicit null fields to restore as empty strings, got %#v", record)
	}
}

func TestDiagnosticActivityRecordFromActivityRecordPreservesOriginalShape(t *testing.T) {
	t.Parallel()

	var quantity, _, err = decimalsupport.ParseString("1")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}

	var record = DiagnosticActivityRecordFromActivityRecord(ActivityRecord{
		SourceID:         "activity-1",
		OccurredAt:       "2025-01-01T10:00:00Z",
		ActivityType:     ActivityTypeBuy,
		AssetIdentityKey: "asset-btc-001",
		AssetSymbol:      "BTC",
		Quantity:         quantity,
		RawHash:          "hash-1",
	})
	if record.SourceID == nil || *record.SourceID != "activity-1" {
		t.Fatalf("expected original source_id, got %#v", record)
	}
	if record.AssetIdentityKey == nil || *record.AssetIdentityKey != "asset-btc-001" {
		t.Fatalf("expected original asset_identity_key, got %#v", record)
	}
	if record.OrderCurrency != nil || record.AssetProfileCurrency != nil || record.BaseCurrency != nil || record.SourceScope != nil {
		t.Fatalf("expected absent source fields to stay nil for explicit-null JSON rendering, got %#v", record)
	}
	redacted := record.RedactFinancialValues()
	if redacted.Quantity != nil || redacted.OrderUnitPrice != nil || redacted.OrderGrossValue != nil || redacted.OrderFeeAmount != nil {
		t.Fatalf("expected redaction to clear financial values, got %#v", redacted)
	}
}
