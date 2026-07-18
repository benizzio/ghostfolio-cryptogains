// Package runtimeflow provides reusable runtime-backed black-box fixtures for
// repository test suites.
//
// Authored by: OpenCode
package runtimeflow

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/apd/v3"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// RoundedReportActivityInput describes the compact decimal fields used to build
// normalized report activity fixtures. For example, pass one value to
// RoundedReportActivity to preserve the exact fixture strings while avoiding
// repeated decimal parsing code in an integration scenario.
// Authored by: OpenCode
type RoundedReportActivityInput struct {
	SourceID         string
	OccurredAt       string
	ActivityType     syncmodel.ActivityType
	AssetIdentityKey string
	AssetSymbol      string
	AssetName        string
	Quantity         string
	OrderCurrency    string
	OrderUnitPrice   string
	OrderGrossValue  string
	OrderFeeAmount   string
}

// RoundedReportActivity converts one compact report fixture into a normalized
// activity record using exact decimal parsing. For example, use it when a
// fixture needs the same optional-price and monetary-value semantics as the
// runtime's normalized activity cache.
// Authored by: OpenCode
func RoundedReportActivity(t *testing.T, input RoundedReportActivityInput) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         input.SourceID,
		OccurredAt:       input.OccurredAt,
		ActivityType:     input.ActivityType,
		AssetIdentityKey: input.AssetIdentityKey,
		AssetSymbol:      input.AssetSymbol,
		AssetName:        input.AssetName,
		Quantity:         MustRoundedIntegrationDecimal(t, input.Quantity),
		OrderCurrency:    input.OrderCurrency,
		OrderUnitPrice:   RoundedIntegrationDecimalPointer(t, input.OrderUnitPrice),
		OrderGrossValue:  RoundedIntegrationDecimalPointer(t, input.OrderGrossValue),
		OrderFeeAmount:   RoundedIntegrationDecimalPointer(t, input.OrderFeeAmount),
	}
}

// MixedCurrencyConversionProtectedActivityCache builds the deterministic USD,
// EUR, and GBP priced activity fixture used by report conversion and
// presentation scenarios. For example, pass 6 to reproduce the six-activity
// fixture with its preserved source dates, currencies, and ordering.
// Authored by: OpenCode
func MixedCurrencyConversionProtectedActivityCache(t *testing.T, activityCount int) syncmodel.ProtectedActivityCache {
	t.Helper()

	var activities = make([]syncmodel.ActivityRecord, 0, activityCount)
	var currencies = []string{"USD", "EUR", "GBP"}
	for index := 0; index < activityCount; index++ {
		var year = 2024
		if index%2 == 1 {
			year = 2025
		}
		var currency = currencies[index%len(currencies)]
		var date = time.Date(year, time.January, 2+(index%24), 10, 0, 0, 0, time.FixedZone("source", (index%5-2)*60*60))
		if index == 6 {
			date = time.Date(2024, time.January, 6, 11, 0, 0, 0, time.UTC)
		}

		activities = append(activities, RoundedReportActivity(t, RoundedReportActivityInput{
			SourceID:         MixedCurrencySourceID(currency, year, index),
			OccurredAt:       date.Format(time.RFC3339),
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-mixed-001",
			AssetSymbol:      "MIX",
			AssetName:        "Mixed Currency Asset",
			Quantity:         "1",
			OrderCurrency:    currency,
			OrderUnitPrice:   "10",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "1",
		}))
	}

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             MustReportFixtureTime(t),
		RetrievedCount:       len(activities),
		ActivityCount:        len(activities),
		AvailableReportYears: []int{2024, 2025},
		Activities:           activities,
	}
}

// RoundedUnitPriceProtectedActivityCache builds the exact two-activity
// repeating-decimal unit-price regression fixture. For example, use it to
// verify presentation rounding without changing calculated values.
// Authored by: OpenCode
func RoundedUnitPriceProtectedActivityCache(t *testing.T) syncmodel.ProtectedActivityCache {
	t.Helper()

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             MustReportFixtureTime(t),
		RetrievedCount:       2,
		ActivityCount:        2,
		AvailableReportYears: []int{2024},
		Activities: []syncmodel.ActivityRecord{
			RoundedReportActivity(t, RoundedReportActivityInput{
				SourceID:         "unit-buy-2024-001",
				OccurredAt:       "2024-01-01T10:00:00Z",
				ActivityType:     syncmodel.ActivityTypeBuy,
				AssetIdentityKey: "asset-unit-001",
				AssetSymbol:      "UNIT",
				AssetName:        "Unit Asset",
				Quantity:         "3",
				OrderCurrency:    "USD",
				OrderGrossValue:  "1",
				OrderFeeAmount:   "0",
			}),
			RoundedReportActivity(t, RoundedReportActivityInput{
				SourceID:         "unit-sell-2024-001",
				OccurredAt:       "2024-03-01T10:00:00Z",
				ActivityType:     syncmodel.ActivityTypeSell,
				AssetIdentityKey: "asset-unit-001",
				AssetSymbol:      "UNIT",
				AssetName:        "Unit Asset",
				Quantity:         "1",
				OrderCurrency:    "USD",
				OrderGrossValue:  "1",
				OrderFeeAmount:   "0",
				OrderUnitPrice:   "1",
			}),
		},
	}
}

// SameCurrencyRoundedUnitPriceProtectedActivityCache returns the rounded unit-
// price fixture with every order denominated in the selected report currency.
// For example, pass ReportBaseCurrencyEUR to exercise the EUR path.
// Authored by: OpenCode
func SameCurrencyRoundedUnitPriceProtectedActivityCache(t *testing.T, reportBaseCurrency reportmodel.ReportBaseCurrency) syncmodel.ProtectedActivityCache {
	t.Helper()

	var cache = RoundedUnitPriceProtectedActivityCache(t)
	for index := range cache.Activities {
		cache.Activities[index].OrderCurrency = reportBaseCurrency.Label()
	}

	return cache
}

// OffsetSensitiveCurrencyProtectedActivityCache builds the preserved source-
// offset date fixture used to verify currency-rate calendar selection. For
// example, use it when UTC conversion would otherwise select the wrong date.
// Authored by: OpenCode
func OffsetSensitiveCurrencyProtectedActivityCache(t *testing.T) syncmodel.ProtectedActivityCache {
	t.Helper()

	var activities = []syncmodel.ActivityRecord{
		RoundedReportActivity(t, RoundedReportActivityInput{
			SourceID:         "offset-before-utc-buy",
			OccurredAt:       "2024-01-01T23:30:00-02:00",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-offset-001",
			AssetSymbol:      "OFF",
			AssetName:        "Offset Asset",
			Quantity:         "1",
			OrderCurrency:    "EUR",
			OrderUnitPrice:   "10",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "1",
		}),
		RoundedReportActivity(t, RoundedReportActivityInput{
			SourceID:         "offset-after-utc-buy",
			OccurredAt:       "2024-01-02T00:30:00+02:00",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-offset-001",
			AssetSymbol:      "OFF",
			AssetName:        "Offset Asset",
			Quantity:         "1",
			OrderCurrency:    "GBP",
			OrderUnitPrice:   "10",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "1",
		}),
	}

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             MustReportFixtureTime(t),
		RetrievedCount:       len(activities),
		ActivityCount:        len(activities),
		AvailableReportYears: []int{2024},
		Activities:           activities,
	}
}

// MixedCurrencySourceID returns the stable source identifier used by the
// mixed-currency fixture. For example, currency "USD", year 2024, and index 0
// produce "mixed-usd-buy-2024-000".
// Authored by: OpenCode
func MixedCurrencySourceID(currency string, year int, index int) string {
	return strings.ToLower("mixed-"+currency+"-buy-") + strconv.Itoa(year) + "-" + LeftPadThree(index)
}

// LeftPadThree renders one three-character fixture index. For example, 7 is
// rendered as "007" and 123 remains "123".
// Authored by: OpenCode
func LeftPadThree(value int) string {
	var raw = strconv.Itoa(value)
	for len(raw) < 3 {
		raw = "0" + raw
	}
	return raw
}

// MustIntegrationReportRequest creates a validated FIFO Markdown report
// request for the supplied year and base currency. For example, use it when a
// scenario needs to compare calculated values for multiple base currencies.
// Authored by: OpenCode
func MustIntegrationReportRequest(t *testing.T, year int, reportBaseCurrency reportmodel.ReportBaseCurrency) reportmodel.ReportRequest {
	t.Helper()

	var request, err = reportmodel.NewReportRequest(
		year,
		reportmodel.CostBasisMethodFIFO,
		reportBaseCurrency,
		reportmodel.ReportOutputFormatMarkdown,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new integration report request: %v", err)
	}
	return request
}

// MustIntegrationReportRequestForFormat creates a validated FIFO report
// request for the supplied year and output format using USD. For example, pass
// ReportOutputFormatPDF to exercise the runtime PDF boundary.
// Authored by: OpenCode
func MustIntegrationReportRequestForFormat(t *testing.T, year int, outputFormat reportmodel.ReportOutputFormat) reportmodel.ReportRequest {
	t.Helper()

	var request, err = reportmodel.NewReportRequest(
		year,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		outputFormat,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new integration report request for %s: %v", outputFormat, err)
	}
	return request
}

// MustReportGenerationSyncConfig returns a custom-origin config suitable for a
// sync followed by runtime-backed report generation. For example, pass a test
// server URL to keep both operations in the same temporary app directory.
// Authored by: OpenCode
func MustReportGenerationSyncConfig(t *testing.T, origin string) configmodel.AppSetupConfig {
	t.Helper()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, origin, true, time.Now())
	if err != nil {
		t.Fatalf("new report-generation sync config: %v", err)
	}
	return config
}

// MustReportFixtureTime parses the stable timestamp shared by report cache
// fixtures. For example, use it for SyncedAt when constructing a new cache.
// Authored by: OpenCode
func MustReportFixtureTime(t *testing.T) time.Time {
	t.Helper()

	const raw = "2026-05-20T15:04:05Z"
	var parsed, err = time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("parse report fixture time %q: %v", raw, err)
	}
	return parsed
}

// RoundedIntegrationDecimalPointer parses an optional exact decimal fixture.
// For example, an empty string returns nil while "10" returns a pointer to an
// exact decimal value.
// Authored by: OpenCode
func RoundedIntegrationDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()
	if raw == "" {
		return nil
	}
	var value = MustRoundedIntegrationDecimal(t, raw)
	return &value
}

// MustRoundedIntegrationDecimal parses one exact decimal fixture. For example,
// use it for quantities and monetary values that must not pass through float64.
// Authored by: OpenCode
func MustRoundedIntegrationDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()
	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse rounded integration decimal %q: %v", raw, err)
	}
	return value
}
