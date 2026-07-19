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

	"github.com/cockroachdb/apd/v3"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

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
