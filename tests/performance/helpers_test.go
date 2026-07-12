//go:build performance

// Package performance isolates resource-sensitive black-box performance checks
// from deterministic test and coverage suites.
//
// Authored by: OpenCode
package performance

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/cockroachdb/apd/v3"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"
)

func selectReportBaseCurrency(t *testing.T, model *flow.Model, currency reportmodel.ReportBaseCurrency) *flow.Model {
	t.Helper()
	var updated tea.Model
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = mustFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = mustFlowModel(t, updated)
	var marker = "> " + currency.Label()
	for attempt := 0; attempt < len(reportmodel.SupportedReportBaseCurrencies())+1; attempt++ {
		if strings.Contains(runtimeflow.NormalizeRenderedText(model.View().Content), marker) {
			return model
		}
		updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		model = mustFlowModel(t, updated)
	}
	t.Fatalf("expected report base currency %q to be selected", currency.Label())
	return model
}

func mustFlowModel(t *testing.T, updated tea.Model) *flow.Model {
	t.Helper()
	var model, ok = updated.(*flow.Model)
	if !ok {
		t.Fatalf("expected updated model to be *flow.Model, got %T", updated)
	}
	return model
}

func largeCrossCurrencyCache(t *testing.T, activityCount int) syncmodel.ProtectedActivityCache {
	t.Helper()
	var activities = make([]syncmodel.ActivityRecord, 0, activityCount)
	var currencies = []string{"EUR", "GBP", "USD"}
	for index := 0; index < activityCount; index++ {
		var currency = currencies[index%len(currencies)]
		activities = append(activities, performanceActivity(t, fmt.Sprintf("responsiveness-%s-buy-%05d", strings.ToLower(currency), index), time.Date(2025, time.Month(1+index%3), 1+index%28, 9, 0, 0, 0, time.FixedZone("source", 2*60*60)).Format(time.RFC3339), currency))
	}
	return syncmodel.ProtectedActivityCache{SyncedAt: time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC), RetrievedCount: len(activities), ActivityCount: len(activities), AvailableReportYears: []int{2025}, Activities: activities}
}

type largeReportPerformanceFixture struct {
	ProtectedActivityCache syncmodel.ProtectedActivityCache
	ReportYear             int
	ActivityCount          int
	CalendarYearSpan       int
}

func largeReportFixture(t *testing.T) largeReportPerformanceFixture {
	t.Helper()
	const activityCount = 10000
	const assetActivityCount = activityCount / 2
	const buysPerAsset = assetActivityCount / 2
	const sellsPerAsset = assetActivityCount / 2
	var activities = make([]syncmodel.ActivityRecord, 0, activityCount)
	for _, asset := range []struct {
		key, symbol, name   string
		buyValue, sellValue int
	}{{"asset-btc-performance-001", "BTC", "Bitcoin", 100, 1200}, {"asset-eth-performance-001", "ETH", "Ethereum", 300, 900}} {
		for index := 0; index < buysPerAsset; index++ {
			activities = append(activities, largeReportActivity(t, asset.key, asset.symbol, asset.name, "buy", index, 2020+index%5, syncmodel.ActivityTypeBuy, asset.buyValue+index%900))
		}
		for index := 0; index < sellsPerAsset; index++ {
			activities = append(activities, largeReportActivity(t, asset.key, asset.symbol, asset.name, "sell", index, 2025, syncmodel.ActivityTypeSell, asset.sellValue+index%700))
		}
	}
	var cache = syncmodel.ProtectedActivityCache{SyncedAt: time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC), RetrievedCount: len(activities), ActivityCount: len(activities), AvailableReportYears: []int{2020, 2021, 2022, 2023, 2024, 2025}, ScopeReliability: syncmodel.ScopeReliabilityPartial, Activities: activities}
	return largeReportPerformanceFixture{ProtectedActivityCache: cache, ReportYear: 2025, ActivityCount: len(activities), CalendarYearSpan: 6}
}

func largeReportActivity(t *testing.T, key string, symbol string, name string, action string, index int, year int, activityType syncmodel.ActivityType, grossValue int) syncmodel.ActivityRecord {
	t.Helper()
	var sourceID = fmt.Sprintf("%s-%s-performance-%05d", strings.ToLower(symbol), action, index+1)
	var occurredAt = time.Date(year, time.Month(index%12+1), index%28+1, index%24, index%60, 0, 0, time.UTC).Format(time.RFC3339)
	var record = performanceActivity(t, sourceID, occurredAt, "USD")
	record.ActivityType = activityType
	record.AssetIdentityKey = key
	record.AssetSymbol = symbol
	record.AssetName = name
	record.OrderGrossValue = mustDecimalPointer(t, fmt.Sprintf("%d", grossValue))
	record.OrderFeeAmount = mustDecimalPointer(t, fmt.Sprintf("%d", index%5+1))
	if activityType == syncmodel.ActivityTypeBuy {
		record.OrderUnitPrice = nil
	}
	return record
}

func performanceActivity(t *testing.T, sourceID string, occurredAt string, currency string) syncmodel.ActivityRecord {
	t.Helper()
	var quantity, _, err = decimalsupport.ParseString("1")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	var unitPrice = mustDecimalPointer(t, "10")
	var grossValue = mustDecimalPointer(t, "10")
	var feeAmount = mustDecimalPointer(t, "1")
	return syncmodel.ActivityRecord{SourceID: sourceID, OccurredAt: occurredAt, ActivityType: syncmodel.ActivityTypeBuy, AssetIdentityKey: "asset-responsive-001", AssetSymbol: "RSP", AssetName: "Responsive Asset", Quantity: quantity, OrderCurrency: currency, OrderUnitPrice: unitPrice, OrderGrossValue: grossValue, OrderFeeAmount: feeAmount}
}

func mustDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()
	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", raw, err)
	}
	return &value
}
