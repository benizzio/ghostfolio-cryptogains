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

// TestActivityValidationContractOrdersSameAssetSameDayUsingActivityTypeBeforeSourceID
// verifies the reopened same-day Ghostfolio ordering contract.
// Authored by: OpenCode
func TestActivityValidationContractOrdersSameAssetSameDayUsingActivityTypeBeforeSourceID(t *testing.T) {
	t.Parallel()

	records := []syncmodel.ActivityRecord{
		activityValidationContractRecord(t, "sell-z", "2024-01-10T00:00:00Z", syncmodel.ActivityTypeSell, "BTC", "1", "120", "120", nil),
		activityValidationContractRecord(t, "buy-a", "2024-01-10T23:59:59Z", syncmodel.ActivityTypeBuy, "BTC", "2", "100", "200", nil),
		activityValidationContractRecord(t, "buy-b", "2024-01-10T12:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "110", "110", nil),
	}

	cache, err := syncnormalize.NewNormalizer().Normalize(records)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := syncvalidate.NewValidator().Validate(cache); err != nil {
		t.Fatalf("validate: %v", err)
	}

	var got = []string{
		cache.Activities[0].SourceID,
		cache.Activities[1].SourceID,
		cache.Activities[2].SourceID,
	}
	var want = []string{"buy-a", "buy-b", "sell-z"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected same-day order: got %#v want %#v", got, want)
		}
	}
	if cache.Activities[0].OccurredAt != "2024-01-10T23:59:59Z" {
		t.Fatalf("expected original occurred_at to stay preserved, got %q", cache.Activities[0].OccurredAt)
	}
	if cache.Activities[2].OccurredAt != "2024-01-10T00:00:00Z" {
		t.Fatalf("expected original occurred_at to stay preserved, got %q", cache.Activities[2].OccurredAt)
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
