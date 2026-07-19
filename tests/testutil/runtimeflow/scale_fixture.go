// Package runtimeflow provides reusable runtime-backed black-box fixtures for
// repository test suites.
//
// Authored by: OpenCode
package runtimeflow

import (
	"fmt"
	"strings"
	"testing"
	"time"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// LargeReportScaleFixture contains the authoritative 10,000-activity report
// workload and the request metadata shared by deterministic scale-content and
// isolated performance tests. Its cache has two assets, 2,500 BUY activities
// per asset distributed across 2020 through 2024, 2,500 SELL activities per
// asset in 2025, and exactly 3,334 USD, 3,333 EUR, and 3,333 GBP activities.
// Authored by: OpenCode
type LargeReportScaleFixture struct {
	ProtectedActivityCache syncmodel.ProtectedActivityCache
	ReportYear             int
	ActivityCount          int
	CalendarYearSpan       int
}

// LargeReportFixture builds the deterministic 10,000-activity workload used
// by report scale-content and performance scenarios. For example, pass the
// returned cache to SeedProtectedSnapshot before generating a 2025 HIFO/USD
// report; all financial values are exact fixture decimals and no provider
// network is contacted.
// Authored by: OpenCode
func LargeReportFixture(t *testing.T) LargeReportScaleFixture {
	t.Helper()
	const activityCount = 10000
	const assetActivityCount = activityCount / 2
	const buysPerAsset = assetActivityCount / 2
	const sellsPerAsset = assetActivityCount / 2
	var activities = make([]syncmodel.ActivityRecord, 0, activityCount)
	var activityIndex int
	for _, asset := range []struct {
		key, symbol, name          string
		buyValue, sellValue        int
		reliableScope              *syncmodel.SourceScope
		forceUnavailableScopeEntry bool
	}{
		{"asset-btc-performance-001", "BTC", "Bitcoin", 100, 1200, largeReportWalletScope("wallet-performance-main", "Performance Main Wallet"), false},
		{"asset-eth-performance-001", "ETH", "Ethereum", 300, 900, largeReportWalletScope("wallet-performance-fallback", "Performance Fallback Wallet"), true},
	} {
		for index := 0; index < buysPerAsset; index++ {
			var sourceScope = largeReportAssetScope(asset.reliableScope, asset.forceUnavailableScopeEntry, index)
			activities = append(activities, largeReportActivity(t, asset.key, asset.symbol, asset.name, "buy", index, 2020+index%5, syncmodel.ActivityTypeBuy, asset.buyValue+index%900, largeReportCurrencyForIndex(activityIndex), sourceScope))
			activityIndex++
		}
		for index := 0; index < sellsPerAsset; index++ {
			var fixtureIndex = index + buysPerAsset
			var sourceScope = largeReportAssetScope(asset.reliableScope, asset.forceUnavailableScopeEntry, fixtureIndex)
			activities = append(activities, largeReportActivity(t, asset.key, asset.symbol, asset.name, "sell", index, 2025, syncmodel.ActivityTypeSell, asset.sellValue+index%700, largeReportCurrencyForIndex(activityIndex), sourceScope))
			activityIndex++
		}
	}
	var cache = syncmodel.ProtectedActivityCache{SyncedAt: time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC), RetrievedCount: len(activities), ActivityCount: len(activities), AvailableReportYears: []int{2020, 2021, 2022, 2023, 2024, 2025}, ScopeReliability: syncmodel.ScopeReliabilityPartial, Activities: activities}
	return LargeReportScaleFixture{ProtectedActivityCache: cache, ReportYear: 2025, ActivityCount: len(activities), CalendarYearSpan: 6}
}

// largeReportCurrencyForIndex assigns the exact three-currency distribution
// required by the named 10,000-activity workload.
// Authored by: OpenCode
func largeReportCurrencyForIndex(index int) string {
	switch {
	case index < 3334:
		return "USD"
	case index < 6667:
		return "EUR"
	default:
		return "GBP"
	}
}

// largeReportActivity creates one quantity-one, non-zero priced activity with
// gross value and fee in one order tier so calculation derives its unit price.
// Authored by: OpenCode
func largeReportActivity(t *testing.T, key string, symbol string, name string, action string, index int, year int, activityType syncmodel.ActivityType, grossValue int, currency string, sourceScope *syncmodel.SourceScope) syncmodel.ActivityRecord {
	t.Helper()
	var sourceID = fmt.Sprintf("%s-%s-performance-%05d", strings.ToLower(symbol), action, index+1)
	var occurredAt = time.Date(year, time.Month(index%12+1), index%28+1, index%24, index%60, 0, 0, time.UTC).Format(time.RFC3339)
	var record = RoundedReportActivity(t, RoundedReportActivityInput{SourceID: sourceID, OccurredAt: occurredAt, ActivityType: syncmodel.ActivityTypeBuy, AssetIdentityKey: "asset-responsive-001", AssetSymbol: "RSP", AssetName: "Responsive Asset", Quantity: "1", OrderCurrency: currency, OrderUnitPrice: "10", OrderGrossValue: "10", OrderFeeAmount: "1"})
	record.ActivityType = activityType
	record.AssetIdentityKey = key
	record.AssetSymbol = symbol
	record.AssetName = name
	record.OrderUnitPrice = nil
	record.OrderGrossValue = RoundedIntegrationDecimalPointer(t, fmt.Sprintf("%d", grossValue))
	record.OrderFeeAmount = RoundedIntegrationDecimalPointer(t, fmt.Sprintf("%d", index%5+1))
	record.SourceScope = sourceScope
	return record
}

// largeReportWalletScope returns one reliable wallet scope for the named
// large-history report fixture.
// Authored by: OpenCode
func largeReportWalletScope(id string, name string) *syncmodel.SourceScope {
	return &syncmodel.SourceScope{ID: id, Name: name, Kind: syncmodel.SourceScopeKindWallet, Reliability: syncmodel.ScopeReliabilityReliable}
}

// largeReportAssetScope keeps scope-local fallback active for one asset by
// mixing reliable and unavailable source-scope entries.
// Authored by: OpenCode
func largeReportAssetScope(reliableScope *syncmodel.SourceScope, forceUnavailableScopeEntry bool, index int) *syncmodel.SourceScope {
	if forceUnavailableScopeEntry && index%4 == 0 {
		return nil
	}
	return reliableScope
}
