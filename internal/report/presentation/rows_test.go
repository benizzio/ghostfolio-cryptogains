// Package presentation tests format-neutral report table row builders.
// Authored by: OpenCode
package presentation

import (
	"reflect"
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
	if activityRow.Date != "2024-01-02 01:04:05" || activityRow.SourceID != "activity|id" || activityRow.Quantity != "1.5" || activityRow.GrossValue != "1234.00" || activityRow.ConversionStatus != "Same currency" || activityRow.Note != "note text" {
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
	if annexRow.FullLiquidationEvent != "Yes" || annexRow.ActivityType != "SELL" {
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
			_, err := BuildAnnexActivityRow(reportmodel.AuditActivityEntry{SourceID: "annex", Quantity: invalid})
			return err
		}, want: "annex activity row \"annex\" quantity"},
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

// TestBuildAnnexActivityRowReturnsContextualErrors verifies every
// canonicalization and label failure identifies the affected audit activity.
// Authored by: OpenCode
func TestBuildAnnexActivityRowReturnsContextualErrors(t *testing.T) {
	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	var entry = reportmodel.AuditActivityEntry{
		SourceID:              "annex-activity",
		ActivityType:          reportmodel.ActivityTypeBuy,
		Quantity:              *apd.New(1, 0),
		QuantityAfterActivity: *apd.New(1, 0),
		BasisAfterActivity:    *apd.New(1, 0),
	}
	for _, testCase := range []struct {
		name      string
		configure func(*reportmodel.AuditActivityEntry)
		operation string
	}{
		{name: "quantity", configure: func(entry *reportmodel.AuditActivityEntry) { entry.Quantity = invalid }, operation: "quantity"},
		{name: "unit price", configure: func(entry *reportmodel.AuditActivityEntry) { entry.UnitPrice = &invalid }, operation: "unit price"},
		{name: "gross value", configure: func(entry *reportmodel.AuditActivityEntry) { entry.GrossValue = &invalid }, operation: "gross value"},
		{name: "fee", configure: func(entry *reportmodel.AuditActivityEntry) { entry.FeeAmount = &invalid }, operation: "fee"},
		{name: "quantity after activity", configure: func(entry *reportmodel.AuditActivityEntry) { entry.QuantityAfterActivity = invalid }, operation: "quantity after activity"},
		{name: "basis after activity", configure: func(entry *reportmodel.AuditActivityEntry) { entry.BasisAfterActivity = invalid }, operation: "basis after activity"},
		{name: "allocated basis", configure: func(entry *reportmodel.AuditActivityEntry) { entry.AllocatedBasis = &invalid }, operation: "allocated basis"},
		{name: "net liquidation proceeds", configure: func(entry *reportmodel.AuditActivityEntry) { entry.NetLiquidationProceeds = &invalid }, operation: "net liquidation proceeds"},
		{name: "gain or loss", configure: func(entry *reportmodel.AuditActivityEntry) { entry.GainOrLoss = &invalid }, operation: "gain or loss"},
		{name: "activity type label", configure: func(entry *reportmodel.AuditActivityEntry) {
			entry.ActivityType = reportmodel.ActivityType("unsupported")
		}, operation: "activity type label"},
		{name: "conversion status label", configure: func(entry *reportmodel.AuditActivityEntry) {
			entry.ConversionStatus = reportmodel.ConversionStatus("unsupported")
		}, operation: "conversion status label"},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var configured = entry
			testCase.configure(&configured)

			_, err := BuildAnnexActivityRow(configured)
			var want = "render annex activity row \"annex-activity\" " + testCase.operation
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want context %q", err, want)
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

// TestBuildActivityRowFormatsEveryFinancialField verifies two-place financial
// presentation while preserving canonical activity quantities.
// Authored by: OpenCode
func TestBuildActivityRowFormatsEveryFinancialField(t *testing.T) {
	var activity = reportmodel.AssetActivityRow{
		SourceID:                    "activity-financial",
		OccurredAt:                  time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
		ActivityType:                reportmodel.ActivityTypeBuy,
		Quantity:                    mustFinancialDecimal(t, "2.000"),
		UnitPrice:                   testDecimalPointer(t, "1.005"),
		GrossValue:                  testDecimalPointer(t, "12.004"),
		FeeAmount:                   testDecimalPointer(t, "0"),
		ActivityCurrency:            "USD",
		BasisAfterRow:               mustFinancialDecimal(t, "8.005"),
		CalculationCurrency:         "USD",
		QuantityAfterRow:            mustFinancialDecimal(t, "0.1000"),
		ConversionStatus:            reportmodel.ConversionStatusSameCurrency,
		HoldingReductionExplanation: "synthetic activity",
	}

	var rendered, err = BuildActivityRow(activity)
	if err != nil {
		t.Fatalf("build activity row: %v", err)
	}

	var want = ActivityRow{
		Date:                "2024-01-02 03:04:05",
		SourceID:            "activity-financial",
		ActivityType:        "BUY",
		Quantity:            "2",
		UnitPrice:           "1.01",
		GrossValue:          "12.00",
		Fee:                 "0.00",
		QuantityAfterRow:    "0.1",
		BasisAfterRow:       "8.01",
		ActivityCurrency:    "USD",
		CalculationCurrency: "USD",
		ConversionStatus:    "Same currency",
		Note:                "synthetic activity",
	}
	if !reflect.DeepEqual(rendered, want) {
		t.Fatalf("activity row = %#v, want %#v", rendered, want)
	}
}

// TestBuildActivityRowPreservesOptionalAndPresentZeroValues verifies that nil
// monetary pointers remain blank while present exact zeros remain visible.
// Authored by: OpenCode
func TestBuildActivityRowPreservesOptionalAndPresentZeroValues(t *testing.T) {
	var absent = reportmodel.AssetActivityRow{
		SourceID:            "activity-absent",
		OccurredAt:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		ActivityType:        reportmodel.ActivityTypeBuy,
		Quantity:            mustFinancialDecimal(t, "1"),
		BasisAfterRow:       mustFinancialDecimal(t, "1"),
		CalculationCurrency: "USD",
		QuantityAfterRow:    mustFinancialDecimal(t, "1"),
	}
	var rendered, err = BuildActivityRow(absent)
	if err != nil {
		t.Fatalf("build activity row with absent amounts: %v", err)
	}
	if rendered.UnitPrice != "" || rendered.GrossValue != "" || rendered.Fee != "" {
		t.Fatalf("absent activity amounts = %#v, want blank optional fields", rendered)
	}

	var zero = reportmodel.AssetActivityRow{
		SourceID:            "activity-zero",
		OccurredAt:          absent.OccurredAt,
		ActivityType:        reportmodel.ActivityTypeBuy,
		Quantity:            mustFinancialDecimal(t, "1"),
		UnitPrice:           testDecimalPointer(t, "0"),
		GrossValue:          testDecimalPointer(t, "0"),
		FeeAmount:           testDecimalPointer(t, "0"),
		ActivityCurrency:    "USD",
		BasisAfterRow:       mustFinancialDecimal(t, "0"),
		CalculationCurrency: "USD",
		QuantityAfterRow:    mustFinancialDecimal(t, "0"),
	}
	rendered, err = BuildActivityRow(zero)
	if err != nil {
		t.Fatalf("build activity row with present zeros: %v", err)
	}
	if rendered.UnitPrice != "0.00" || rendered.GrossValue != "0.00" || rendered.Fee != "0.00" || rendered.BasisAfterRow != "0.00" {
		t.Fatalf("present activity zeros = %#v, want 0.00 monetary values", rendered)
	}
}

// TestBuildLiquidationRowFormatsEveryFinancialField verifies liquidation
// monetary fields use financial presentation while disposed quantity remains
// canonical.
// Authored by: OpenCode
func TestBuildLiquidationRowFormatsEveryFinancialField(t *testing.T) {
	var liquidation = reportmodel.LiquidationCalculation{
		SourceID:               "liquidation-financial",
		OccurredAt:             time.Date(2024, time.February, 3, 4, 5, 6, 0, time.UTC),
		DisposedQuantity:       mustFinancialDecimal(t, "0.00000001"),
		AllocatedBasis:         mustFinancialDecimal(t, "10.005"),
		NetLiquidationProceeds: mustFinancialDecimal(t, "12.004"),
		GainOrLoss:             mustFinancialDecimal(t, "-2.005"),
		CalculationCurrency:    "USD",
	}

	var rendered, err = BuildLiquidationRow(liquidation, "USD")
	if err != nil {
		t.Fatalf("build liquidation row: %v", err)
	}
	var want = LiquidationRow{
		Date:                "2024-02-03 04:05:06",
		SourceID:            "liquidation-financial",
		DisposedQuantity:    "0.00000001",
		AllocatedBasis:      "10.01",
		NetProceeds:         "12.00",
		GainOrLoss:          "-2.01",
		CalculationCurrency: "USD",
	}
	if !reflect.DeepEqual(rendered, want) {
		t.Fatalf("liquidation row = %#v, want %#v", rendered, want)
	}
}

// TestBuildAnnexActivityRowFormatsEveryFinancialField verifies every Annex
// monetary field and the canonical quantity fields.
// Authored by: OpenCode
func TestBuildAnnexActivityRowFormatsEveryFinancialField(t *testing.T) {
	var entry = reportmodel.AuditActivityEntry{
		SourceID:               "annex-financial",
		OccurredAt:             time.Date(2024, time.March, 4, 5, 6, 7, 0, time.UTC),
		ActivityType:           reportmodel.ActivityTypeBuy,
		Quantity:               mustFinancialDecimal(t, "0.00000001"),
		UnitPrice:              testDecimalPointer(t, "1.005"),
		GrossValue:             testDecimalPointer(t, "2.004"),
		FeeAmount:              testDecimalPointer(t, "0"),
		ActivityCurrency:       "USD",
		CalculationCurrency:    "USD",
		QuantityAfterActivity:  mustFinancialDecimal(t, "2.000"),
		BasisAfterActivity:     mustFinancialDecimal(t, "3.005"),
		FullLiquidationEvent:   false,
		AllocatedBasis:         testDecimalPointer(t, "4.005"),
		NetLiquidationProceeds: testDecimalPointer(t, "5.005"),
		GainOrLoss:             testDecimalPointer(t, "-6.005"),
	}

	var rendered, err = BuildAnnexActivityRow(entry)
	if err != nil {
		t.Fatalf("build annex activity row: %v", err)
	}
	var want = AnnexActivityRow{
		Date:                 "2024-03-04 05:06:07",
		SourceID:             "annex-financial",
		ActivityType:         "BUY",
		Quantity:             "0.00000001",
		UnitPrice:            "1.01",
		GrossValue:           "2.00",
		Fee:                  "0.00",
		ActivityCurrency:     "USD",
		CalculationCurrency:  "USD",
		QuantityAfter:        "2",
		BasisAfter:           "3.01",
		FullLiquidationEvent: "No",
		AllocatedBasis:       "4.01",
		NetProceeds:          "5.01",
		GainOrLoss:           "-6.01",
	}
	if !reflect.DeepEqual(rendered, want) {
		t.Fatalf("annex activity row = %#v, want %#v", rendered, want)
	}
}

// TestBuildAnnexActivityRowPreservesOptionalAndPresentZeroValues verifies
// Annex optional monetary fields distinguish absence from exact zero.
// Authored by: OpenCode
func TestBuildAnnexActivityRowPreservesOptionalAndPresentZeroValues(t *testing.T) {
	var entry = reportmodel.AuditActivityEntry{
		SourceID:              "annex-optional",
		OccurredAt:            time.Date(2024, time.March, 4, 0, 0, 0, 0, time.UTC),
		ActivityType:          reportmodel.ActivityTypeBuy,
		Quantity:              mustFinancialDecimal(t, "1"),
		ActivityCurrency:      "USD",
		CalculationCurrency:   "USD",
		QuantityAfterActivity: mustFinancialDecimal(t, "1"),
		BasisAfterActivity:    mustFinancialDecimal(t, "1"),
	}
	var rendered, err = BuildAnnexActivityRow(entry)
	if err != nil {
		t.Fatalf("build annex row with absent amounts: %v", err)
	}
	if rendered.UnitPrice != "" || rendered.GrossValue != "" || rendered.Fee != "" || rendered.AllocatedBasis != "" || rendered.NetProceeds != "" || rendered.GainOrLoss != "" {
		t.Fatalf("absent annex amounts = %#v, want blank optional fields", rendered)
	}

	entry.UnitPrice = testDecimalPointer(t, "0")
	entry.GrossValue = testDecimalPointer(t, "0")
	entry.FeeAmount = testDecimalPointer(t, "0")
	entry.AllocatedBasis = testDecimalPointer(t, "0")
	entry.NetLiquidationProceeds = testDecimalPointer(t, "0")
	entry.GainOrLoss = testDecimalPointer(t, "0")
	rendered, err = BuildAnnexActivityRow(entry)
	if err != nil {
		t.Fatalf("build annex row with present zeros: %v", err)
	}
	if rendered.UnitPrice != "0.00" || rendered.GrossValue != "0.00" || rendered.Fee != "0.00" || rendered.BasisAfter != "1.00" || rendered.AllocatedBasis != "0.00" || rendered.NetProceeds != "0.00" || rendered.GainOrLoss != "0.00" {
		t.Fatalf("present annex zeros = %#v, want 0.00 monetary values", rendered)
	}
}

// TestBuildConversionAuditRowFormatsAmountsAndCanonicalRate verifies all
// conversion monetary fields use two-place presentation while the disclosed
// normalized rate remains canonical and unrounded.
// Authored by: OpenCode
func TestBuildConversionAuditRowFormatsAmountsAndCanonicalRate(t *testing.T) {
	var entry = reportmodel.ConversionAuditEntry{
		SourceID:           "conversion-financial",
		AssetLabel:         "Synthetic Asset",
		ActivityDate:       time.Date(2024, time.April, 5, 0, 0, 0, 0, time.UTC),
		SourceCurrency:     "EUR",
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		RateDate:           time.Date(2024, time.April, 4, 0, 0, 0, 0, time.UTC),
		RateValue:          mustFinancialDecimal(t, "0.86010"),
		QuoteDirection:     reportmodel.QuoteDirectionSourcePerBase,
		Amounts: []reportmodel.ConvertedActivityAmount{
			{AmountKind: reportmodel.ConvertedAmountKindUnitPrice, OriginalAmount: mustFinancialDecimal(t, "1.005"), ConvertedAmount: mustFinancialDecimal(t, "2.005")},
			{AmountKind: reportmodel.ConvertedAmountKindGrossValue, OriginalAmount: mustFinancialDecimal(t, "3.004"), ConvertedAmount: mustFinancialDecimal(t, "4.004")},
			{AmountKind: reportmodel.ConvertedAmountKindFeeAmount, OriginalAmount: mustFinancialDecimal(t, "5.005"), ConvertedAmount: mustFinancialDecimal(t, "6.005")},
		},
	}

	var rendered, err = BuildConversionAuditRow(7, entry)
	if err != nil {
		t.Fatalf("build conversion audit row: %v", err)
	}
	if rendered.RateValue != "0.8601" {
		t.Fatalf("conversion rate = %q, want canonical %q", rendered.RateValue, "0.8601")
	}
	var wantAmounts = []string{"unit_price: 1.01 -> 2.01", "gross_value: 3.00 -> 4.00", "fee_amount: 5.01 -> 6.01"}
	if !reflect.DeepEqual(rendered.ConvertedAmountEntries, wantAmounts) {
		t.Fatalf("converted amount entries = %#v, want %#v", rendered.ConvertedAmountEntries, wantAmounts)
	}
}

// TestBuildConversionAuditRowUsesExactZeroDecisions verifies that only an
// exact zero-to-zero pair is omitted and a non-zero pair remains visible even
// when both financial values round to 0.00.
// Authored by: OpenCode
func TestBuildConversionAuditRowUsesExactZeroDecisions(t *testing.T) {
	var entry = reportmodel.ConversionAuditEntry{
		SourceID:           "conversion-zero",
		ActivityDate:       time.Date(2024, time.May, 6, 0, 0, 0, 0, time.UTC),
		SourceCurrency:     "EUR",
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		RateValue:          mustFinancialDecimal(t, "1.094600"),
		QuoteDirection:     reportmodel.QuoteDirectionSourcePerBase,
		Amounts: []reportmodel.ConvertedActivityAmount{
			{AmountKind: reportmodel.ConvertedAmountKindUnitPrice, OriginalAmount: mustFinancialDecimal(t, "0"), ConvertedAmount: mustFinancialDecimal(t, "0")},
			{AmountKind: reportmodel.ConvertedAmountKindGrossValue, OriginalAmount: mustFinancialDecimal(t, "0.004"), ConvertedAmount: mustFinancialDecimal(t, "0.004")},
			{AmountKind: reportmodel.ConvertedAmountKindFeeAmount, OriginalAmount: mustFinancialDecimal(t, "0"), ConvertedAmount: mustFinancialDecimal(t, "0.004")},
		},
	}

	var rendered, err = BuildConversionAuditRow(8, entry)
	if err != nil {
		t.Fatalf("build conversion audit row: %v", err)
	}
	var want = []string{"gross_value: 0.00 -> 0.00", "fee_amount: 0.00 -> 0.00"}
	if !reflect.DeepEqual(rendered.ConvertedAmountEntries, want) {
		t.Fatalf("converted amount entries = %#v, want %#v", rendered.ConvertedAmountEntries, want)
	}
}

// TestBuildRowsDoNotMutateSourceDecimals verifies every row builder leaves its
// input decimal values and nested conversion amounts unchanged.
// Authored by: OpenCode
func TestBuildRowsDoNotMutateSourceDecimals(t *testing.T) {
	var activity = reportmodel.AssetActivityRow{
		SourceID:            "activity-immutable",
		OccurredAt:          time.Date(2024, time.June, 7, 0, 0, 0, 0, time.UTC),
		ActivityType:        reportmodel.ActivityTypeBuy,
		Quantity:            mustFinancialDecimal(t, "2.000"),
		UnitPrice:           testDecimalPointer(t, "1.005"),
		GrossValue:          testDecimalPointer(t, "2.005"),
		FeeAmount:           testDecimalPointer(t, "0.005"),
		ActivityCurrency:    "USD",
		BasisAfterRow:       mustFinancialDecimal(t, "3.005"),
		CalculationCurrency: "USD",
		QuantityAfterRow:    mustFinancialDecimal(t, "0.1000"),
	}
	var activityBefore = activity
	activityBefore.Quantity = cloneRowDecimal(activity.Quantity)
	activityBefore.UnitPrice = cloneRowDecimalPointer(activity.UnitPrice)
	activityBefore.GrossValue = cloneRowDecimalPointer(activity.GrossValue)
	activityBefore.FeeAmount = cloneRowDecimalPointer(activity.FeeAmount)
	activityBefore.BasisAfterRow = cloneRowDecimal(activity.BasisAfterRow)
	activityBefore.QuantityAfterRow = cloneRowDecimal(activity.QuantityAfterRow)
	if _, err := BuildActivityRow(activity); err != nil {
		t.Fatalf("build immutable activity row: %v", err)
	}
	if !reflect.DeepEqual(activity, activityBefore) {
		t.Fatalf("activity source decimals changed: before=%#v after=%#v", activityBefore, activity)
	}

	var liquidation = reportmodel.LiquidationCalculation{
		SourceID:               "liquidation-immutable",
		OccurredAt:             activity.OccurredAt,
		DisposedQuantity:       mustFinancialDecimal(t, "0.1000"),
		AllocatedBasis:         mustFinancialDecimal(t, "4.005"),
		NetLiquidationProceeds: mustFinancialDecimal(t, "5.005"),
		GainOrLoss:             mustFinancialDecimal(t, "-6.005"),
		CalculationCurrency:    "USD",
	}
	var liquidationBefore = liquidation
	liquidationBefore.DisposedQuantity = cloneRowDecimal(liquidation.DisposedQuantity)
	liquidationBefore.AllocatedBasis = cloneRowDecimal(liquidation.AllocatedBasis)
	liquidationBefore.NetLiquidationProceeds = cloneRowDecimal(liquidation.NetLiquidationProceeds)
	liquidationBefore.GainOrLoss = cloneRowDecimal(liquidation.GainOrLoss)
	if _, err := BuildLiquidationRow(liquidation, "USD"); err != nil {
		t.Fatalf("build immutable liquidation row: %v", err)
	}
	if !reflect.DeepEqual(liquidation, liquidationBefore) {
		t.Fatalf("liquidation source decimals changed: before=%#v after=%#v", liquidationBefore, liquidation)
	}

	var annex = reportmodel.AuditActivityEntry{
		SourceID:               "annex-immutable",
		OccurredAt:             activity.OccurredAt,
		ActivityType:           reportmodel.ActivityTypeBuy,
		Quantity:               mustFinancialDecimal(t, "0.00000001"),
		UnitPrice:              testDecimalPointer(t, "1.005"),
		GrossValue:             testDecimalPointer(t, "2.005"),
		FeeAmount:              testDecimalPointer(t, "0.005"),
		ActivityCurrency:       "USD",
		CalculationCurrency:    "USD",
		QuantityAfterActivity:  mustFinancialDecimal(t, "2.000"),
		BasisAfterActivity:     mustFinancialDecimal(t, "3.005"),
		AllocatedBasis:         testDecimalPointer(t, "4.005"),
		NetLiquidationProceeds: testDecimalPointer(t, "5.005"),
		GainOrLoss:             testDecimalPointer(t, "-6.005"),
	}
	var annexBefore = annex
	annexBefore.Quantity = cloneRowDecimal(annex.Quantity)
	annexBefore.UnitPrice = cloneRowDecimalPointer(annex.UnitPrice)
	annexBefore.GrossValue = cloneRowDecimalPointer(annex.GrossValue)
	annexBefore.FeeAmount = cloneRowDecimalPointer(annex.FeeAmount)
	annexBefore.QuantityAfterActivity = cloneRowDecimal(annex.QuantityAfterActivity)
	annexBefore.BasisAfterActivity = cloneRowDecimal(annex.BasisAfterActivity)
	annexBefore.AllocatedBasis = cloneRowDecimalPointer(annex.AllocatedBasis)
	annexBefore.NetLiquidationProceeds = cloneRowDecimalPointer(annex.NetLiquidationProceeds)
	annexBefore.GainOrLoss = cloneRowDecimalPointer(annex.GainOrLoss)
	if _, err := BuildAnnexActivityRow(annex); err != nil {
		t.Fatalf("build immutable annex row: %v", err)
	}
	if !reflect.DeepEqual(annex, annexBefore) {
		t.Fatalf("annex source decimals changed: before=%#v after=%#v", annexBefore, annex)
	}

	var conversion = reportmodel.ConversionAuditEntry{
		SourceID:           "conversion-immutable",
		ActivityDate:       activity.OccurredAt,
		SourceCurrency:     "EUR",
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		RateValue:          mustFinancialDecimal(t, "16.9140"),
		QuoteDirection:     reportmodel.QuoteDirectionSourcePerBase,
		Amounts: []reportmodel.ConvertedActivityAmount{
			{AmountKind: reportmodel.ConvertedAmountKindUnitPrice, OriginalAmount: mustFinancialDecimal(t, "1.005"), ConvertedAmount: mustFinancialDecimal(t, "2.005")},
			{AmountKind: reportmodel.ConvertedAmountKindGrossValue, OriginalAmount: mustFinancialDecimal(t, "3.005"), ConvertedAmount: mustFinancialDecimal(t, "4.005")},
			{AmountKind: reportmodel.ConvertedAmountKindFeeAmount, OriginalAmount: mustFinancialDecimal(t, "5.005"), ConvertedAmount: mustFinancialDecimal(t, "6.005")},
		},
	}
	var conversionBefore = conversion
	conversionBefore.RateValue = cloneRowDecimal(conversion.RateValue)
	conversionBefore.Amounts = append([]reportmodel.ConvertedActivityAmount(nil), conversion.Amounts...)
	for index := range conversionBefore.Amounts {
		conversionBefore.Amounts[index].OriginalAmount = cloneRowDecimal(conversion.Amounts[index].OriginalAmount)
		conversionBefore.Amounts[index].ConvertedAmount = cloneRowDecimal(conversion.Amounts[index].ConvertedAmount)
	}
	if _, err := BuildConversionAuditRow(9, conversion); err != nil {
		t.Fatalf("build immutable conversion row: %v", err)
	}
	if !reflect.DeepEqual(conversion, conversionBefore) {
		t.Fatalf("conversion source decimals changed: before=%#v after=%#v", conversionBefore, conversion)
	}
}

// TestBuildRowsReturnContextualErrorsForEveryFinancialField verifies each
// monetary formatting failure identifies its row and semantic field.
// Authored by: OpenCode
func TestBuildRowsReturnContextualErrorsForEveryFinancialField(t *testing.T) {
	var invalid apd.Decimal
	invalid.Form = apd.Infinite

	var activity = reportmodel.AssetActivityRow{
		SourceID:            "activity-error",
		OccurredAt:          time.Date(2024, time.July, 8, 0, 0, 0, 0, time.UTC),
		ActivityType:        reportmodel.ActivityTypeBuy,
		Quantity:            mustFinancialDecimal(t, "1"),
		UnitPrice:           testDecimalPointer(t, "1"),
		GrossValue:          testDecimalPointer(t, "1"),
		FeeAmount:           testDecimalPointer(t, "1"),
		ActivityCurrency:    "USD",
		BasisAfterRow:       mustFinancialDecimal(t, "1"),
		CalculationCurrency: "USD",
		QuantityAfterRow:    mustFinancialDecimal(t, "1"),
	}
	for _, testCase := range []struct {
		name      string
		configure func(*reportmodel.AssetActivityRow)
		want      string
	}{
		{name: "quantity", configure: func(row *reportmodel.AssetActivityRow) { row.Quantity = invalid }, want: "activity row \"activity-error\" quantity"},
		{name: "unit price", configure: func(row *reportmodel.AssetActivityRow) { row.UnitPrice = &invalid }, want: "activity row \"activity-error\" unit price"},
		{name: "gross value", configure: func(row *reportmodel.AssetActivityRow) { row.GrossValue = &invalid }, want: "activity row \"activity-error\" gross value"},
		{name: "fee", configure: func(row *reportmodel.AssetActivityRow) { row.FeeAmount = &invalid }, want: "activity row \"activity-error\" fee"},
		{name: "basis after row", configure: func(row *reportmodel.AssetActivityRow) { row.BasisAfterRow = invalid }, want: "activity row \"activity-error\" basis after row"},
		{name: "quantity after row", configure: func(row *reportmodel.AssetActivityRow) { row.QuantityAfterRow = invalid }, want: "activity row \"activity-error\" quantity after row"},
	} {
		var testCase = testCase
		t.Run("activity/"+testCase.name, func(t *testing.T) {
			var configured = activity
			testCase.configure(&configured)
			_, err := BuildActivityRow(configured)
			if err == nil || !strings.Contains(err.Error(), "render "+testCase.want) {
				t.Fatalf("error = %v, want context %q", err, testCase.want)
			}
		})
	}

	var liquidation = reportmodel.LiquidationCalculation{
		SourceID:               "liquidation-error",
		OccurredAt:             activity.OccurredAt,
		DisposedQuantity:       mustFinancialDecimal(t, "1"),
		AllocatedBasis:         mustFinancialDecimal(t, "1"),
		NetLiquidationProceeds: mustFinancialDecimal(t, "1"),
		GainOrLoss:             mustFinancialDecimal(t, "1"),
	}
	for _, testCase := range []struct {
		name      string
		configure func(*reportmodel.LiquidationCalculation)
		want      string
	}{
		{name: "disposed quantity", configure: func(row *reportmodel.LiquidationCalculation) { row.DisposedQuantity = invalid }, want: "liquidation \"liquidation-error\" disposed quantity"},
		{name: "allocated basis", configure: func(row *reportmodel.LiquidationCalculation) { row.AllocatedBasis = invalid }, want: "liquidation \"liquidation-error\" allocated basis"},
		{name: "net proceeds", configure: func(row *reportmodel.LiquidationCalculation) { row.NetLiquidationProceeds = invalid }, want: "liquidation \"liquidation-error\" net proceeds"},
		{name: "gain or loss", configure: func(row *reportmodel.LiquidationCalculation) { row.GainOrLoss = invalid }, want: "liquidation \"liquidation-error\" gain or loss"},
	} {
		var testCase = testCase
		t.Run("liquidation/"+testCase.name, func(t *testing.T) {
			var configured = liquidation
			testCase.configure(&configured)
			_, err := BuildLiquidationRow(configured, "USD")
			if err == nil || !strings.Contains(err.Error(), "render "+testCase.want) {
				t.Fatalf("error = %v, want context %q", err, testCase.want)
			}
		})
	}

	var annex = reportmodel.AuditActivityEntry{
		SourceID:               "annex-error",
		OccurredAt:             activity.OccurredAt,
		ActivityType:           reportmodel.ActivityTypeBuy,
		Quantity:               mustFinancialDecimal(t, "1"),
		UnitPrice:              testDecimalPointer(t, "1"),
		GrossValue:             testDecimalPointer(t, "1"),
		FeeAmount:              testDecimalPointer(t, "1"),
		CalculationCurrency:    "USD",
		QuantityAfterActivity:  mustFinancialDecimal(t, "1"),
		BasisAfterActivity:     mustFinancialDecimal(t, "1"),
		AllocatedBasis:         testDecimalPointer(t, "1"),
		NetLiquidationProceeds: testDecimalPointer(t, "1"),
		GainOrLoss:             testDecimalPointer(t, "1"),
	}
	for _, testCase := range []struct {
		name      string
		configure func(*reportmodel.AuditActivityEntry)
		want      string
	}{
		{name: "quantity", configure: func(entry *reportmodel.AuditActivityEntry) { entry.Quantity = invalid }, want: "annex activity row \"annex-error\" quantity"},
		{name: "unit price", configure: func(entry *reportmodel.AuditActivityEntry) { entry.UnitPrice = &invalid }, want: "annex activity row \"annex-error\" unit price"},
		{name: "gross value", configure: func(entry *reportmodel.AuditActivityEntry) { entry.GrossValue = &invalid }, want: "annex activity row \"annex-error\" gross value"},
		{name: "fee", configure: func(entry *reportmodel.AuditActivityEntry) { entry.FeeAmount = &invalid }, want: "annex activity row \"annex-error\" fee"},
		{name: "quantity after activity", configure: func(entry *reportmodel.AuditActivityEntry) { entry.QuantityAfterActivity = invalid }, want: "annex activity row \"annex-error\" quantity after activity"},
		{name: "basis after activity", configure: func(entry *reportmodel.AuditActivityEntry) { entry.BasisAfterActivity = invalid }, want: "annex activity row \"annex-error\" basis after activity"},
		{name: "allocated basis", configure: func(entry *reportmodel.AuditActivityEntry) { entry.AllocatedBasis = &invalid }, want: "annex activity row \"annex-error\" allocated basis"},
		{name: "net liquidation proceeds", configure: func(entry *reportmodel.AuditActivityEntry) { entry.NetLiquidationProceeds = &invalid }, want: "annex activity row \"annex-error\" net liquidation proceeds"},
		{name: "gain or loss", configure: func(entry *reportmodel.AuditActivityEntry) { entry.GainOrLoss = &invalid }, want: "annex activity row \"annex-error\" gain or loss"},
	} {
		var testCase = testCase
		t.Run("annex/"+testCase.name, func(t *testing.T) {
			var configured = annex
			testCase.configure(&configured)
			_, err := BuildAnnexActivityRow(configured)
			if err == nil || !strings.Contains(err.Error(), "render "+testCase.want) {
				t.Fatalf("error = %v, want context %q", err, testCase.want)
			}
		})
	}

	var conversion = reportmodel.ConversionAuditEntry{
		SourceID:           "conversion-error",
		ActivityDate:       activity.OccurredAt,
		SourceCurrency:     "EUR",
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		RateValue:          mustFinancialDecimal(t, "1"),
		Amounts: []reportmodel.ConvertedActivityAmount{
			{AmountKind: reportmodel.ConvertedAmountKindUnitPrice, OriginalAmount: mustFinancialDecimal(t, "1"), ConvertedAmount: mustFinancialDecimal(t, "1")},
		},
	}
	for _, testCase := range []struct {
		name      string
		configure func(*reportmodel.ConversionAuditEntry)
		want      string
	}{
		{name: "rate value", configure: func(entry *reportmodel.ConversionAuditEntry) { entry.RateValue = invalid }, want: "conversion audit entry 10 rate value"},
		{name: "original amount", configure: func(entry *reportmodel.ConversionAuditEntry) { entry.Amounts[0].OriginalAmount = invalid }, want: "conversion audit entry 10 amount 0 original amount"},
		{name: "converted amount", configure: func(entry *reportmodel.ConversionAuditEntry) { entry.Amounts[0].ConvertedAmount = invalid }, want: "conversion audit entry 10 amount 0 converted amount"},
	} {
		var testCase = testCase
		t.Run("conversion/"+testCase.name, func(t *testing.T) {
			var configured = conversion
			configured.Amounts = append([]reportmodel.ConvertedActivityAmount(nil), conversion.Amounts...)
			testCase.configure(&configured)
			_, err := BuildConversionAuditRow(10, configured)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want context %q", err, testCase.want)
			}
		})
	}
}

// testDecimalPointer parses one synthetic decimal and returns an optional
// pointer suitable for a report-model fixture.
// Authored by: OpenCode
func testDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = mustFinancialDecimal(t, raw)
	return &value
}

// cloneRowDecimal makes a deep decimal copy for source-immutability snapshots.
// Authored by: OpenCode
func cloneRowDecimal(value apd.Decimal) apd.Decimal {
	var clone apd.Decimal
	clone.Set(&value)
	return clone
}

// cloneRowDecimalPointer makes a deep optional decimal copy for source snapshots.
// Authored by: OpenCode
func cloneRowDecimalPointer(value *apd.Decimal) *apd.Decimal {
	if value == nil {
		return nil
	}
	var clone = cloneRowDecimal(*value)
	return &clone
}

// TestBuildAnnexActivityRowUsesInheritedClassificationForVisibleCurrency
// verifies that presentation blanks only classified original currencies and
// retains an unclassified tiny-positive currency that displays as 0.00.
// Authored by: OpenCode
func TestBuildAnnexActivityRowUsesInheritedClassificationForVisibleCurrency(t *testing.T) {
	for _, testCase := range []struct {
		name             string
		classified       bool
		unitPrice        string
		wantActivityCode string
	}{
		{name: "classified exact zero", classified: true, unitPrice: "0", wantActivityCode: ""},
		{name: "unclassified tiny positive", classified: false, unitPrice: "0.00000001", wantActivityCode: "EUR"},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var entry = annexClassificationPresentationEntry(t, testCase.classified, testCase.unitPrice)
			var before = entry
			before.Quantity = cloneRowDecimal(entry.Quantity)
			before.UnitPrice = cloneRowDecimalPointer(entry.UnitPrice)
			before.GrossValue = cloneRowDecimalPointer(entry.GrossValue)
			before.FeeAmount = cloneRowDecimalPointer(entry.FeeAmount)
			before.QuantityAfterActivity = cloneRowDecimal(entry.QuantityAfterActivity)
			before.BasisAfterActivity = cloneRowDecimal(entry.BasisAfterActivity)
			before.AllocatedBasis = cloneRowDecimalPointer(entry.AllocatedBasis)
			before.NetLiquidationProceeds = cloneRowDecimalPointer(entry.NetLiquidationProceeds)
			before.GainOrLoss = cloneRowDecimalPointer(entry.GainOrLoss)

			var rendered, err = BuildAnnexActivityRow(entry)
			if err != nil {
				t.Fatalf("build classified presentation row: %v", err)
			}
			if rendered.ActivityCurrency != testCase.wantActivityCode {
				t.Fatalf("visible activity currency = %q, want %q", rendered.ActivityCurrency, testCase.wantActivityCode)
			}
			if rendered.CalculationCurrency != "USD" {
				t.Fatalf("calculation currency = %q, want %q", rendered.CalculationCurrency, "USD")
			}
			if testCase.unitPrice == "0.00000001" && rendered.UnitPrice != "0.00" {
				t.Fatalf("tiny-positive unit price = %q, want %q", rendered.UnitPrice, "0.00")
			}
			if !reflect.DeepEqual(entry, before) {
				t.Fatalf("annex source changed: before=%#v after=%#v", before, entry)
			}
		})
	}
}

// annexClassificationPresentationEntry creates a synthetic Annex row with an
// inherited classification and distinct source and calculation currencies.
// Authored by: OpenCode
func annexClassificationPresentationEntry(t *testing.T, classified bool, unitPrice string) reportmodel.AuditActivityEntry {
	return reportmodel.AuditActivityEntry{
		SourceID:                     "annex-classification-" + unitPrice,
		OccurredAt:                   time.Date(2024, time.August, 9, 0, 0, 0, 0, time.UTC),
		ActivityType:                 reportmodel.ActivityTypeSell,
		Quantity:                     mustFinancialDecimal(t, "1"),
		UnitPrice:                    testDecimalPointer(t, unitPrice),
		GrossValue:                   testDecimalPointer(t, unitPrice),
		FeeAmount:                    testDecimalPointer(t, "0"),
		ActivityCurrency:             "EUR",
		CalculationCurrency:          "USD",
		QuantityAfterActivity:        mustFinancialDecimal(t, "0"),
		BasisAfterActivity:           mustFinancialDecimal(t, "1"),
		FullLiquidationEvent:         false,
		AllocatedBasis:               testDecimalPointer(t, "1"),
		NetLiquidationProceeds:       testDecimalPointer(t, unitPrice),
		GainOrLoss:                   testDecimalPointer(t, unitPrice),
		IsZeroPricedHoldingReduction: classified,
	}
}

// TestBuildAnnexActivityRowMapsBooleanAndCurrencyPresentation verifies the
// shared Annex row derives exact boolean labels and blanks only classified
// original currencies while retaining calculation currency and visible
// unclassified controls.
// Authored by: OpenCode
func TestBuildAnnexActivityRowMapsBooleanAndCurrencyPresentation(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		classified      bool
		fullLiquidation bool
		unitPrice       string
		wantBoolean     string
		wantActivity    string
	}{
		{name: "classified", classified: true, fullLiquidation: true, unitPrice: "0", wantBoolean: "Yes", wantActivity: ""},
		{name: "unclassified tiny positive", classified: false, fullLiquidation: false, unitPrice: "0.00000001", wantBoolean: "No", wantActivity: "EUR"},
	} {
		var testCase = testCase
		t.Run(testCase.name, func(t *testing.T) {
			var entry = annexClassificationPresentationEntry(t, testCase.classified, testCase.unitPrice)
			entry.FullLiquidationEvent = testCase.fullLiquidation

			var rendered, err = BuildAnnexActivityRow(entry)
			if err != nil {
				t.Fatalf("build Annex presentation row: %v", err)
			}
			if rendered.FullLiquidationEvent != testCase.wantBoolean {
				t.Fatalf("full liquidation label = %q, want %q", rendered.FullLiquidationEvent, testCase.wantBoolean)
			}
			if rendered.ActivityCurrency != testCase.wantActivity {
				t.Fatalf("original activity currency = %q, want %q", rendered.ActivityCurrency, testCase.wantActivity)
			}
			if rendered.CalculationCurrency != "USD" {
				t.Fatalf("calculation currency = %q, want %q", rendered.CalculationCurrency, "USD")
			}
			if rendered.UnitPrice != "0.00" {
				t.Fatalf("unit price = %q, want %q", rendered.UnitPrice, "0.00")
			}
		})
	}
}
