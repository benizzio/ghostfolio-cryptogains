package unit

import (
	"testing"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
)

func TestScopeReliabilityDerivationByAssetTimeline(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		records []syncmodel.ActivityRecord
		want    syncmodel.ScopeReliability
	}{
		{
			name: "reliable when stable non-empty scope matches",
			records: []syncmodel.ActivityRecord{
				unitActivityRecord(t, "a-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", func(record *syncmodel.ActivityRecord) {
					record.SourceScope = &syncmodel.SourceScope{ID: "account-1", Kind: syncmodel.SourceScopeKindAccount}
				}),
				unitActivityRecord(t, "a-2", "2024-01-02T10:00:00Z", syncmodel.ActivityTypeSell, "BTC", "1", "100", "100", func(record *syncmodel.ActivityRecord) {
					record.SourceScope = &syncmodel.SourceScope{ID: "account-1", Kind: syncmodel.SourceScopeKindAccount}
				}),
			},
			want: syncmodel.ScopeReliabilityReliable,
		},
		{
			name: "partial when scope is missing for part of the timeline",
			records: []syncmodel.ActivityRecord{
				unitActivityRecord(t, "a-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", func(record *syncmodel.ActivityRecord) {
					record.SourceScope = &syncmodel.SourceScope{ID: "account-1", Kind: syncmodel.SourceScopeKindAccount}
				}),
				unitActivityRecord(t, "a-2", "2024-01-02T10:00:00Z", syncmodel.ActivityTypeSell, "BTC", "1", "100", "100", nil),
			},
			want: syncmodel.ScopeReliabilityPartial,
		},
		{
			name: "partial when scope conflicts across the timeline",
			records: []syncmodel.ActivityRecord{
				unitActivityRecord(t, "a-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", func(record *syncmodel.ActivityRecord) {
					record.SourceScope = &syncmodel.SourceScope{ID: "account-1", Kind: syncmodel.SourceScopeKindAccount}
				}),
				unitActivityRecord(t, "a-2", "2024-01-02T10:00:00Z", syncmodel.ActivityTypeSell, "BTC", "1", "100", "100", func(record *syncmodel.ActivityRecord) {
					record.SourceScope = &syncmodel.SourceScope{ID: "wallet-1", Kind: syncmodel.SourceScopeKindWallet}
				}),
			},
			want: syncmodel.ScopeReliabilityPartial,
		},
		{
			name: "unavailable when no usable scope is present",
			records: []syncmodel.ActivityRecord{
				unitActivityRecord(t, "a-1", "2024-01-01T10:00:00Z", syncmodel.ActivityTypeBuy, "BTC", "1", "100", "100", nil),
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
