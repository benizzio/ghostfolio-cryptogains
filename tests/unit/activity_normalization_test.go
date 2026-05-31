package unit

import (
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/cockroachdb/apd/v3"
)

func TestActivityNormalizationRemovesDuplicatesByNormalizedHash(t *testing.T) {
	t.Parallel()

	cache, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		unitActivityRecord(t, "activity-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1.0", "100.00", "100.0", nil),
		unitActivityRecord(t, "activity-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100.00", nil),
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cache.ActivityCount != 1 {
		t.Fatalf("expected one deduplicated activity, got %d", cache.ActivityCount)
	}
	if cache.Activities[0].RawHash == "" {
		t.Fatalf("expected persisted duplicate hash")
	}
}

// TestActivityNormalizationOrdersSameAssetSameDayByActivityTypeThenSourceID
// verifies that same-asset same-day ordering ignores arbitrary Ghostfolio clock values.
// Authored by: OpenCode
func TestActivityNormalizationOrdersSameAssetSameDayByActivityTypeThenSourceID(t *testing.T) {
	t.Parallel()

	cache, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		unitActivityRecord(t, "sell-z", "2024-01-01T00:00:00Z", syncmodel.ActivityTypeSell, "BTC", "1", "100", "100", nil),
		unitActivityRecord(t, "buy-b", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
		unitActivityRecord(t, "buy-a", "2024-01-01T23:59:59Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cache.Activities[0].SourceID != "buy-a" || cache.Activities[1].SourceID != "buy-b" || cache.Activities[2].SourceID != "sell-z" {
		t.Fatalf("unexpected deterministic order: %#v", cache.Activities)
	}
	if cache.Activities[0].OccurredAt != "2024-01-01T23:59:59Z" || cache.Activities[2].OccurredAt != "2024-01-01T00:00:00Z" {
		t.Fatalf("expected normalization to preserve original timestamps, got %#v", cache.Activities)
	}
}

// TestActivityNormalizationRejectsAmbiguousSameAssetSameDaySameTypeSameSourceOrdering
// verifies that unresolved same-day tie-breaking still fails normalization.
// Authored by: OpenCode
func TestActivityNormalizationRejectsAmbiguousSameAssetSameDaySameTypeSameSourceOrdering(t *testing.T) {
	t.Parallel()

	_, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		unitActivityRecord(t, "activity-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
		unitActivityRecord(t, "activity-1", "2024-01-01T23:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "2", "100", "200", nil),
	})
	if err == nil {
		t.Fatalf("expected ambiguous same-source ordering rejection")
	}
}

// TestActivityNormalizationValidationUsesSameDayReplayOrderForBelowZeroHoldings
// verifies that replay defensibility follows the reopened same-day ordering rule.
// Authored by: OpenCode
func TestActivityNormalizationValidationUsesSameDayReplayOrderForBelowZeroHoldings(t *testing.T) {
	t.Parallel()

	cache, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		unitActivityRecord(t, "sell-early-clock", "2024-01-01T00:00:00Z", syncmodel.ActivityTypeSell, "BTC", "1", "100", "100", nil),
		unitActivityRecord(t, "buy-late-clock", "2024-01-01T23:59:59Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := syncvalidate.NewValidator().Validate(cache); err != nil {
		t.Fatalf("expected same-day replay ordering to keep holdings defensible, got %v", err)
	}
	if cache.Activities[0].SourceID != "buy-late-clock" || cache.Activities[1].SourceID != "sell-early-clock" {
		t.Fatalf("expected validator input to honor same-day replay order, got %#v", cache.Activities)
	}
}

func TestActivityNormalizationValidationRejectsBelowZeroReplay(t *testing.T) {
	t.Parallel()

	cache, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		unitActivityRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
		unitActivityRecord(t, "sell-1", "2024-01-02T10:00:00Z", syncmodel.ActivityTypeSell, "BTC", "2", "100", "200", nil),
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := syncvalidate.NewValidator().Validate(cache); err == nil {
		t.Fatalf("expected below-zero replay rejection")
	}
}

func TestActivityNormalizationValidationRejectsIncompleteCurrencyContext(t *testing.T) {
	t.Parallel()

	cache, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		func() syncmodel.ActivityRecord {
			record := unitActivityRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil)
			record.OrderCurrency = ""
			record.AssetProfileCurrency = ""
			record.BaseCurrency = ""
			record.BaseGrossValue = nil
			return record
		}(),
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := syncvalidate.NewValidator().Validate(cache); err == nil {
		t.Fatalf("expected incomplete currency context rejection")
	}
}

func TestActivityNormalizationValidationAllowsOneUninformedCurrencyTierWhenOthersRemainInformed(t *testing.T) {
	t.Parallel()

	cache, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		func() syncmodel.ActivityRecord {
			record := unitActivityRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "90", "90", nil)
			assetProfileUnitPrice, _, err := decimalsupport.ParseString("95")
			if err != nil {
				t.Fatalf("parse asset-profile unit price: %v", err)
			}
			baseGrossValue, _, err := decimalsupport.ParseString("100")
			if err != nil {
				t.Fatalf("parse base gross value: %v", err)
			}
			record.OrderCurrency = ""
			record.AssetProfileCurrency = "EUR"
			record.BaseCurrency = "USD"
			record.AssetProfileUnitPrice = &assetProfileUnitPrice
			record.BaseGrossValue = &baseGrossValue
			return record
		}(),
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := syncvalidate.NewValidator().Validate(cache); err != nil {
		t.Fatalf("expected one uninformed tier to remain valid, got %v", err)
	}
}

func TestActivityNormalizationValidationPreservesMixedCurrencyContext(t *testing.T) {
	t.Parallel()

	cache, err := syncnormalize.NewNormalizer().Normalize([]syncmodel.ActivityRecord{
		func() syncmodel.ActivityRecord {
			record := unitActivityRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "90", "90", nil)
			assetProfileUnitPrice, _, err := decimalsupport.ParseString("95")
			if err != nil {
				t.Fatalf("parse asset-profile unit price: %v", err)
			}
			baseGrossValue, _, err := decimalsupport.ParseString("100")
			if err != nil {
				t.Fatalf("parse base gross value: %v", err)
			}
			record.OrderCurrency = "CHF"
			record.AssetProfileCurrency = "EUR"
			record.BaseCurrency = "USD"
			record.OrderUnitPrice = mustUnitActivityDecimalPointer(t, "90")
			record.OrderGrossValue = mustUnitActivityDecimalPointer(t, "90")
			record.AssetProfileUnitPrice = &assetProfileUnitPrice
			record.BaseGrossValue = &baseGrossValue
			return record
		}(),
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := syncvalidate.NewValidator().Validate(cache); err != nil {
		t.Fatalf("expected mixed-currency record to validate, got %v", err)
	}
	if cache.Activities[0].OrderCurrency != "CHF" || cache.Activities[0].AssetProfileCurrency != "EUR" || cache.Activities[0].BaseCurrency != "USD" {
		t.Fatalf("expected mixed-currency context to stay preserved, got %#v", cache.Activities[0])
	}
}

func unitActivityRecord(
	t *testing.T,
	sourceID string,
	occurredAt string,
	activityType syncmodel.ActivityType,
	assetSymbol string,
	quantity string,
	unitPrice string,
	grossValue string,
	mutate func(*syncmodel.ActivityRecord),
) syncmodel.ActivityRecord {
	t.Helper()

	parsedQuantity, _, err := decimalsupport.ParseString(quantity)
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	parsedUnitPrice, _, err := decimalsupport.ParseString(unitPrice)
	if err != nil {
		t.Fatalf("parse unit price: %v", err)
	}
	parsedGrossValue, _, err := decimalsupport.ParseString(grossValue)
	if err != nil {
		t.Fatalf("parse gross value: %v", err)
	}

	record := syncmodel.ActivityRecord{
		SourceID:         sourceID,
		OccurredAt:       occurredAt,
		ActivityType:     activityType,
		AssetIdentityKey: "asset-btc-unit-001",
		AssetSymbol:      assetSymbol,
		OrderCurrency:    "USD",
		BaseCurrency:     "USD",
		Quantity:         parsedQuantity,
		OrderUnitPrice:   &parsedUnitPrice,
		OrderGrossValue:  &parsedGrossValue,
	}
	if mutate != nil {
		mutate(&record)
	}

	return record
}

func mustUnitActivityDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	value, _, err := decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse decimal pointer: %v", err)
	}

	return &value
}
