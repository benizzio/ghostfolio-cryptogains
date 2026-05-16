package unit

import (
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
)

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
		{SourceID: "activity-1", OccurredAt: "2024-12-31T23:30:00-02:00", ActivityType: syncmodel.ActivityTypeBuy, AssetSymbol: "BTC", Quantity: quantity, UnitPrice: unitPrice, GrossValue: grossValue},
		{SourceID: "activity-2", OccurredAt: "2025-01-01T00:15:00+02:00", ActivityType: syncmodel.ActivityTypeBuy, AssetSymbol: "BTC", Quantity: quantity, UnitPrice: unitPrice, GrossValue: grossValue},
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(cache.AvailableReportYears) != 2 || cache.AvailableReportYears[0] != 2024 || cache.AvailableReportYears[1] != 2025 {
		t.Fatalf("unexpected derived years: %#v", cache.AvailableReportYears)
	}
}
