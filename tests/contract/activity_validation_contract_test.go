package contract

import (
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

func TestActivityValidationContractSupportsZeroPricedSellWithComment(t *testing.T) {
	t.Parallel()

	records := []syncmodel.ActivityRecord{
		activityValidationContractRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "2", "100", "200", nil),
		activityValidationContractRecord(t, "sell-1", "2024-01-02T10:00:00Z", syncmodel.ActivityTypeSell, "BTC", "1", "0", "0", func(record *syncmodel.ActivityRecord) {
			record.Comment = "manual holding reduction"
		}),
	}

	cache, err := syncnormalize.NewNormalizer().Normalize(records)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := syncvalidate.NewValidator().Validate(cache); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestActivityValidationContractRejectsBelowZeroHoldings(t *testing.T) {
	t.Parallel()

	records := []syncmodel.ActivityRecord{
		activityValidationContractRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
		activityValidationContractRecord(t, "sell-1", "2024-01-02T10:00:00Z", syncmodel.ActivityTypeSell, "BTC", "2", "110", "220", nil),
	}

	cache, err := syncnormalize.NewNormalizer().Normalize(records)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := syncvalidate.NewValidator().Validate(cache); err == nil {
		t.Fatalf("expected below-zero holdings to be rejected")
	}
}

func TestActivityValidationContractDerivesScopeReliabilityOutcomes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		records []syncmodel.ActivityRecord
		want    syncmodel.ScopeReliability
	}{
		{
			name: "reliable",
			records: []syncmodel.ActivityRecord{
				activityValidationContractRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", func(record *syncmodel.ActivityRecord) {
					record.SourceScope = &syncmodel.SourceScope{ID: "account-1", Kind: syncmodel.SourceScopeKindAccount}
				}),
				activityValidationContractRecord(t, "buy-2", "2024-01-02T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", func(record *syncmodel.ActivityRecord) {
					record.SourceScope = &syncmodel.SourceScope{ID: "account-1", Kind: syncmodel.SourceScopeKindAccount}
				}),
			},
			want: syncmodel.ScopeReliabilityReliable,
		},
		{
			name: "partial",
			records: []syncmodel.ActivityRecord{
				activityValidationContractRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", func(record *syncmodel.ActivityRecord) {
					record.SourceScope = &syncmodel.SourceScope{ID: "account-1", Kind: syncmodel.SourceScopeKindAccount}
				}),
				activityValidationContractRecord(t, "buy-2", "2024-01-02T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
			},
			want: syncmodel.ScopeReliabilityPartial,
		},
		{
			name: "unavailable",
			records: []syncmodel.ActivityRecord{
				activityValidationContractRecord(t, "buy-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
			},
			want: syncmodel.ScopeReliabilityUnavailable,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cache, err := syncnormalize.NewNormalizer().Normalize(testCase.records)
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if cache.ScopeReliability != testCase.want {
				t.Fatalf("scope reliability mismatch: got %q want %q", cache.ScopeReliability, testCase.want)
			}
		})
	}
}

func activityValidationContractRecord(
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
		SourceID:     sourceID,
		OccurredAt:   occurredAt,
		ActivityType: activityType,
		AssetSymbol:  assetSymbol,
		Quantity:     parsedQuantity,
		UnitPrice:    parsedUnitPrice,
		GrossValue:   parsedGrossValue,
	}
	if mutate != nil {
		mutate(&record)
	}

	return record
}
