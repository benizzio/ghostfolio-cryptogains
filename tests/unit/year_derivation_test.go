// Package unit verifies focused exact-decimal and normalization helpers for the
// sync-and-storage slice.
// Authored by: OpenCode
package unit

import (
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
)

// TestYearDerivationUsesSourceTimestampOffset verifies report-year derivation
// from the source timestamp offset instead of machine-local time.
// Authored by: OpenCode
func TestYearDerivationUsesSourceTimestampOffset(t *testing.T) {
	t.Parallel()

	quantity, _, err := decimalsupport.ParseString("1")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	unitPrice, _, err := decimalsupport.ParseString("10")
	if err != nil {
		t.Fatalf("parse unit price: %v", err)
	}
	grossValue, _, err := decimalsupport.ParseString("10")
	if err != nil {
		t.Fatalf("parse gross value: %v", err)
	}

	cache, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		{SourceID: "activity-1", OccurredAt: "2024-12-31T23:30:00-02:00", ActivityType: syncmodel.ActivityTypeBuy, AssetIdentityKey: "asset-btc-year-001", AssetSymbol: "BTC", OrderCurrency: "USD", BaseCurrency: "USD", Quantity: quantity, OrderUnitPrice: &unitPrice, OrderGrossValue: &grossValue},
		{SourceID: "activity-2", OccurredAt: "2025-01-01T00:15:00+02:00", ActivityType: syncmodel.ActivityTypeBuy, AssetIdentityKey: "asset-btc-year-001", AssetSymbol: "BTC", OrderCurrency: "USD", BaseCurrency: "USD", Quantity: quantity, OrderUnitPrice: &unitPrice, OrderGrossValue: &grossValue},
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(cache.AvailableReportYears) != 2 || cache.AvailableReportYears[0] != 2024 || cache.AvailableReportYears[1] != 2025 {
		t.Fatalf("unexpected derived years: %#v", cache.AvailableReportYears)
	}
}
