// Package presentation tests format-neutral report table row builders.
// Authored by: OpenCode
package presentation

import (
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
)

// TestBuildRowsCanonicalizesSharedRendererValues verifies that the shared
// builders preserve renderer-visible semantics before format-specific escaping.
// Authored by: OpenCode
func TestBuildRowsCanonicalizesSharedRendererValues(t *testing.T) {
	var activity = reportmodel.AssetActivityRow{
		SourceID:                    "activity|id",
		OccurredAt:                  time.Date(2024, time.January, 2, 3, 4, 5, 0, time.FixedZone("UTC+2", 2*60*60)),
		ActivityType:                reportmodel.ActivityTypeSell,
		Quantity:                    *apd.New(150, -2),
		GrossValue:                  apd.New(123400, -2),
		BasisAfterRow:               *apd.New(125, -1),
		QuantityAfterRow:            *apd.New(25, -1),
		ActivityCurrency:            "USD",
		CalculationCurrency:         "USD",
		HoldingReductionExplanation: "note\ntext",
	}
	var liquidation = reportmodel.LiquidationCalculation{SourceID: "liquidation", OccurredAt: activity.OccurredAt, DisposedQuantity: *apd.New(15, -1), AllocatedBasis: *apd.New(10, 0), NetLiquidationProceeds: *apd.New(12, 0), GainOrLoss: *apd.New(2, 0)}
	var annex = reportmodel.AuditActivityEntry{SourceID: "annex", OccurredAt: activity.OccurredAt, ActivityType: reportmodel.ActivityTypeSell, Quantity: *apd.New(15, -1), QuantityAfterActivity: *apd.New(0, 0), BasisAfterActivity: *apd.New(0, 0), FullLiquidationEvent: true}

	var activityRow, err = BuildActivityRow(activity)
	if err != nil {
		t.Fatalf("build activity row: %v", err)
	}
	if activityRow.Date != "2024-01-02 01:04:05" || activityRow.SourceID != "activity|id" || activityRow.Quantity != "1.5" || activityRow.GrossValue != "1234" || activityRow.ConversionStatus != "Same currency" || activityRow.Note != "note text" {
		t.Fatalf("unexpected activity row: %#v", activityRow)
	}
	var liquidationRow LiquidationRow
	liquidationRow, err = BuildLiquidationRow(liquidation, "USD")
	if err != nil {
		t.Fatalf("build liquidation row: %v", err)
	}
	if liquidationRow.DisposedQuantity != "1.5" || liquidationRow.CalculationCurrency != "USD" {
		t.Fatalf("unexpected liquidation row: %#v", liquidationRow)
	}
	var annexRow AnnexActivityRow
	annexRow, err = BuildAnnexActivityRow(annex)
	if err != nil {
		t.Fatalf("build annex row: %v", err)
	}
	if annexRow.FullLiquidationEvent != "true" || annexRow.ActivityType != "SELL" {
		t.Fatalf("unexpected annex row: %#v", annexRow)
	}
}

// TestBuildRowsReturnContextualErrors verifies each builder retains its
// renderer-facing decimal and label failure context.
// Authored by: OpenCode
func TestBuildRowsReturnContextualErrors(t *testing.T) {
	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	for _, testCase := range []struct {
		name  string
		build func() error
		want  string
	}{
		{name: "activity", build: func() error {
			_, err := BuildActivityRow(reportmodel.AssetActivityRow{SourceID: "activity", Quantity: invalid})
			return err
		}, want: "activity row \"activity\" quantity"},
		{name: "liquidation", build: func() error {
			_, err := BuildLiquidationRow(reportmodel.LiquidationCalculation{SourceID: "liquidation", DisposedQuantity: invalid}, "USD")
			return err
		}, want: "liquidation \"liquidation\" disposed quantity"},
		{name: "annex", build: func() error {
			_, err := BuildAnnexActivityRow(reportmodel.AuditActivityEntry{Quantity: invalid})
			return err
		}, want: "quantity"},
		{name: "conversion", build: func() error {
			_, err := BuildConversionAuditRow(3, reportmodel.ConversionAuditEntry{RateValue: invalid})
			return err
		}, want: "entry 3 rate value"},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.build(); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
}

// TestCalculationCurrencyLabelUsesFallback verifies blank calculation currency
// values use the report-visible not-applicable label.
// Authored by: OpenCode
func TestCalculationCurrencyLabelUsesFallback(t *testing.T) {
	t.Parallel()

	if got := CalculationCurrencyLabel(" \n "); got != "NOT APPLICABLE" {
		t.Fatalf("calculation currency fallback = %q", got)
	}
}
