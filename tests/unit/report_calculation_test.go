// Package unit verifies focused report-calculation seams without the full
// yearly report runtime.
// Authored by: OpenCode
package unit

import (
	"errors"
	"testing"
	"time"

	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	supporttext "github.com/benizzio/ghostfolio-cryptogains/internal/support/text"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// TestCalculateUsesSelectedYearCutoffForOpeningClosingAndLaterActivityExclusion
// verifies opening carry-forward, in-year liquidation inclusion, and later-
// than-selected-year exclusion.
// Authored by: OpenCode
func TestCalculateUsesSelectedYearCutoffForOpeningClosingAndLaterActivityExclusion(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	var report, err = reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "btc-buy-2023-001",
			OccurredAt:       "2023-06-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-btc-001",
			AssetSymbol:      "BTC",
			AssetName:        "Bitcoin",
			Quantity:         "2",
			OrderCurrency:    "USD",
			OrderGrossValue:  "20",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "10",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "btc-sell-2024-001",
			OccurredAt:       "2024-02-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-btc-001",
			AssetSymbol:      "BTC",
			AssetName:        "Bitcoin",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "15",
			OrderFeeAmount:   "1",
			OrderUnitPrice:   "15",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "btc-sell-2025-001",
			OccurredAt:       "2025-01-10T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-btc-001",
			AssetSymbol:      "BTC",
			AssetName:        "Bitcoin",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "40",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "40",
		}),
	))
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}

	if report.ReportCalculationCurrency != "NOT APPLICABLE" {
		t.Fatalf("unexpected report calculation currency: %q", report.ReportCalculationCurrency)
	}
	if len(report.SummaryEntries) != 1 {
		t.Fatalf("unexpected summary entry count: got %d want 1", len(report.SummaryEntries))
	}
	assertCalculationDecimalString(t, report.SummaryEntries[0].NetGainOrLoss, "4", "btc yearly gain or loss")
	assertCalculationDecimalString(t, report.YearlyNetTotal, "4", "yearly net total")
	if len(report.ReferenceEntries) != 0 {
		t.Fatalf("unexpected reference entry count: got %d want 0", len(report.ReferenceEntries))
	}
	if len(report.DetailSections) != 1 {
		t.Fatalf("unexpected detail section count: got %d want 1", len(report.DetailSections))
	}

	var section = report.DetailSections[0]
	assertCalculationDecimalString(t, section.OpeningQuantity, "2", "opening quantity")
	assertCalculationDecimalString(t, section.OpeningCostBasis, "20", "opening basis")
	assertCalculationDecimalString(t, section.ClosingQuantity, "1", "closing quantity")
	assertCalculationDecimalString(t, section.ClosingCostBasis, "10", "closing basis")
	if len(section.ActivityRows) != 1 {
		t.Fatalf("unexpected activity row count: got %d want 1", len(section.ActivityRows))
	}
	if section.ActivityRows[0].SourceID != "btc-sell-2024-001" {
		t.Fatalf("unexpected in-year activity source: %q", section.ActivityRows[0].SourceID)
	}
	if len(section.LiquidationSummaries) != 1 {
		t.Fatalf("unexpected liquidation summary count: got %d want 1", len(section.LiquidationSummaries))
	}
	assertCalculationDecimalString(t, section.LiquidationSummaries[0].AllocatedBasis, "10", "allocated basis")
	assertCalculationDecimalString(t, section.LiquidationSummaries[0].NetLiquidationProceeds, "14", "net liquidation proceeds")
	assertCalculationDecimalString(t, section.LiquidationSummaries[0].GainOrLoss, "4", "gain or loss")
}

// TestCalculateExcludesAssetsWhoseFirstAcquisitionIsAfterSelectedYear verifies
// that later-only assets are ignored completely for the selected report year.
// Authored by: OpenCode
func TestCalculateExcludesAssetsWhoseFirstAcquisitionIsAfterSelectedYear(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	var report, err = reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "main-buy-2023-001",
			OccurredAt:       "2023-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-main-001",
			AssetSymbol:      "MAIN",
			AssetName:        "Main Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "10",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "main-sell-2024-001",
			OccurredAt:       "2024-02-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-main-001",
			AssetSymbol:      "MAIN",
			AssetName:        "Main Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "12",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "12",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "late-buy-2025-001",
			OccurredAt:       "2025-03-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-late-001",
			AssetSymbol:      "LATE",
			AssetName:        "Late Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "5",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "5",
		}),
	))
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}

	if len(report.SummaryEntries) != 1 {
		t.Fatalf("unexpected summary entry count: got %d want 1", len(report.SummaryEntries))
	}
	if report.SummaryEntries[0].AssetIdentityKey != "asset-main-001" {
		t.Fatalf("unexpected included asset identity: %q", report.SummaryEntries[0].AssetIdentityKey)
	}
	if len(report.DetailSections) != 1 {
		t.Fatalf("unexpected detail section count: got %d want 1", len(report.DetailSections))
	}
	if report.DetailSections[0].AssetIdentityKey != "asset-main-001" {
		t.Fatalf("unexpected detail asset identity: %q", report.DetailSections[0].AssetIdentityKey)
	}
	if len(report.ReferenceEntries) != 1 {
		t.Fatalf("unexpected reference entry count: got %d want 1", len(report.ReferenceEntries))
	}
	if report.ReferenceEntries[0].AssetIdentityKey != "asset-main-001" {
		t.Fatalf("unexpected reference asset identity: %q", report.ReferenceEntries[0].AssetIdentityKey)
	}
}

// TestCalculateMarksPreYearLiquidationAsReferenceOnlyAndDoesNotTreatSameDateBuyAsReopen
// verifies the pre-year reference-only exclusion and the same-source-calendar-
// date BUY-before-SELL replay rule.
// Authored by: OpenCode
func TestCalculateMarksPreYearLiquidationAsReferenceOnlyAndDoesNotTreatSameDateBuyAsReopen(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	var report, err = reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "eth-buy-2023-001",
			OccurredAt:       "2023-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-eth-001",
			AssetSymbol:      "ETH",
			AssetName:        "Ethereum",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "10",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "eth-buy-2023-002",
			OccurredAt:       "2023-06-01T18:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-eth-001",
			AssetSymbol:      "ETH",
			AssetName:        "Ethereum",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "12",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "12",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "eth-sell-2023-001",
			OccurredAt:       "2023-06-01T09:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-eth-001",
			AssetSymbol:      "ETH",
			AssetName:        "Ethereum",
			Quantity:         "2",
			OrderCurrency:    "USD",
			OrderGrossValue:  "30",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "15",
		}),
	))
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}

	if len(report.SummaryEntries) != 0 {
		t.Fatalf("unexpected summary entries: got %d want 0", len(report.SummaryEntries))
	}
	if len(report.DetailSections) != 0 {
		t.Fatalf("unexpected detail sections: got %d want 0", len(report.DetailSections))
	}
	assertCalculationDecimalString(t, report.YearlyNetTotal, "0", "yearly net total")
	if len(report.ReferenceEntries) != 1 {
		t.Fatalf("unexpected reference entry count: got %d want 1", len(report.ReferenceEntries))
	}
	if report.ReferenceEntries[0].MainSectionStatus != reportmodel.ReferenceSectionStatusReferenceOnly {
		t.Fatalf("unexpected reference status: %q", report.ReferenceEntries[0].MainSectionStatus)
	}
	if report.ReferenceEntries[0].FullLiquidationCountThroughYearEnd != 1 {
		t.Fatalf("unexpected full liquidation count: got %d want 1", report.ReferenceEntries[0].FullLiquidationCountThroughYearEnd)
	}
}

// TestCalculateIncludesZeroResultLossAndHoldingReductionDetails verifies zero-
// result inclusion, negative losses, full-liquidation counts, and explained
// zero-priced holding-reduction basis removal.
// Authored by: OpenCode
func TestCalculateIncludesZeroResultLossAndHoldingReductionDetails(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	var report, err = reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "zero-buy-2023-001",
			OccurredAt:       "2023-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-zero-001",
			AssetSymbol:      "ZERO",
			AssetName:        "Zero Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "10",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "zero-sell-2024-001",
			OccurredAt:       "2024-02-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-zero-001",
			AssetSymbol:      "ZERO",
			AssetName:        "Zero Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "10",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "loss-buy-2023-001",
			OccurredAt:       "2023-03-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-loss-001",
			AssetSymbol:      "LOSS",
			AssetName:        "Loss Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "10",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "loss-sell-2024-001",
			OccurredAt:       "2024-04-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-loss-001",
			AssetSymbol:      "LOSS",
			AssetName:        "Loss Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "7",
			OrderFeeAmount:   "1",
			OrderUnitPrice:   "7",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "red-buy-2024-001",
			OccurredAt:       "2024-01-10T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-red-001",
			AssetSymbol:      "RED",
			AssetName:        "Reduction Asset",
			Quantity:         "100",
			OrderCurrency:    "USD",
			OrderGrossValue:  "500",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "5",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "red-reduction-2024-001",
			OccurredAt:       "2024-05-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-red-001",
			AssetSymbol:      "RED",
			AssetName:        "Reduction Asset",
			Quantity:         "20",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "0",
			OrderGrossValue:  "0",
			OrderFeeAmount:   "0",
			Comment:          "manual move",
		}),
	))
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}

	if len(report.SummaryEntries) != 3 {
		t.Fatalf("unexpected summary entry count: got %d want 3", len(report.SummaryEntries))
	}
	assertCalculationDecimalString(t, summaryEntryByAsset(t, report, "asset-zero-001").NetGainOrLoss, "0", "zero-result asset net")
	assertCalculationDecimalString(t, summaryEntryByAsset(t, report, "asset-loss-001").NetGainOrLoss, "-4", "loss asset net")
	assertCalculationDecimalString(t, summaryEntryByAsset(t, report, "asset-red-001").NetGainOrLoss, "0", "holding-reduction asset net")
	assertCalculationDecimalString(t, report.YearlyNetTotal, "-4", "yearly net total")

	if len(report.ReferenceEntries) != 2 {
		t.Fatalf("unexpected reference entry count: got %d want 2", len(report.ReferenceEntries))
	}
	if referenceEntryByAsset(t, report, "asset-zero-001").FullLiquidationCountThroughYearEnd != 1 {
		t.Fatalf("unexpected zero asset liquidation count")
	}
	if referenceEntryByAsset(t, report, "asset-loss-001").FullLiquidationCountThroughYearEnd != 1 {
		t.Fatalf("unexpected loss asset liquidation count")
	}

	var reductionSection = detailSectionByAsset(t, report, "asset-red-001")
	assertCalculationDecimalString(t, reductionSection.OpeningQuantity, "0", "reduction opening quantity")
	assertCalculationDecimalString(t, reductionSection.OpeningCostBasis, "0", "reduction opening basis")
	assertCalculationDecimalString(t, reductionSection.ClosingQuantity, "80", "reduction closing quantity")
	assertCalculationDecimalString(t, reductionSection.ClosingCostBasis, "400", "reduction closing basis")
	if len(reductionSection.ActivityRows) != 2 {
		t.Fatalf("unexpected reduction activity row count: got %d want 2", len(reductionSection.ActivityRows))
	}
	var reductionRow = reductionSection.ActivityRows[1]
	assertCalculationDecimalPointerString(t, reductionRow.UnitPrice, "0", "holding reduction unit price")
	assertCalculationDecimalPointerString(t, reductionRow.GrossValue, "0", "holding reduction gross value")
	assertCalculationDecimalPointerString(t, reductionRow.FeeAmount, "0", "holding reduction fee")
	if reductionRow.ActivityCurrency != "" {
		t.Fatalf("expected zero-priced holding reduction row to omit activity currency")
	}
	if reductionRow.LiquidationCalculation != nil {
		t.Fatalf("expected zero-priced holding reduction row to omit liquidation summary")
	}
	if reductionRow.HoldingReductionExplanation != "manual move" {
		t.Fatalf("unexpected holding reduction explanation: %q", reductionRow.HoldingReductionExplanation)
	}
	assertCalculationDecimalString(t, reductionRow.BasisAfterRow, "400", "holding reduction basis after row")
	assertCalculationDecimalString(t, reductionRow.QuantityAfterRow, "80", "holding reduction quantity after row")
}

// TestCalculateReturnsStructuredActivityReferencesForInputFailures verifies the
// non-secret calculation error taxonomy for offending activity rows.
// Authored by: OpenCode
func TestCalculateReturnsStructuredActivityReferencesForInputFailures(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	_, err := reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "bad-buy-2024-001",
			OccurredAt:       "2024-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-bad-001",
			AssetSymbol:      "BAD",
			AssetName:        "Broken Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
		}),
	))
	if err == nil {
		t.Fatalf("expected calculation error")
	}

	var calculationError *reportmodel.CalculationError
	if !errors.As(err, &calculationError) {
		t.Fatalf("expected structured calculation error, got %T", err)
	}
	if calculationError.Kind() != reportmodel.CalculationErrorKindActivityInput {
		t.Fatalf("unexpected calculation error kind: got %q want %q", calculationError.Kind(), reportmodel.CalculationErrorKindActivityInput)
	}
	if calculationError.SourceID() != "bad-buy-2024-001" {
		t.Fatalf("unexpected calculation error source id: %q", calculationError.SourceID())
	}
	if calculationError.DisplayLabel() != "BAD" {
		t.Fatalf("unexpected calculation error display label: %q", calculationError.DisplayLabel())
	}
	if !supporttext.ContainsAll(err.Error(), "BAD", "bad-buy-2024-001", "incomplete") {
		t.Fatalf("expected user-visible error references, got %q", err.Error())
	}
}

// TestCalculateRoundsPartialLotBasisAllocation verifies shared 16-decimal
// internal precision for repeating partial-lot basis allocation.
// Authored by: OpenCode
func TestCalculateRoundsPartialLotBasisAllocation(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	var report, err = reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "lot-buy-2023-001",
			OccurredAt:       "2023-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-lot-001",
			AssetSymbol:      "LOT",
			AssetName:        "Lot Asset",
			Quantity:         "3",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "3.333333333333333333333333333333333",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "lot-sell-2024-001",
			OccurredAt:       "2024-02-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-lot-001",
			AssetSymbol:      "LOT",
			AssetName:        "Lot Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "5",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "5",
		}),
	))
	if err != nil {
		t.Fatalf("calculate rounded partial-lot report: %v", err)
	}

	assertCalculationDecimalString(t, summaryEntryByAsset(t, report, "asset-lot-001").NetGainOrLoss, "1.6666666666666667", "rounded partial-lot asset net")
	assertCalculationDecimalString(t, report.YearlyNetTotal, "1.6666666666666667", "rounded partial-lot yearly net")

	var detail = detailSectionByAsset(t, report, "asset-lot-001")
	assertCalculationDecimalString(t, detail.ClosingCostBasis, "6.6666666666666667", "rounded partial-lot closing basis")
	if len(detail.LiquidationSummaries) != 1 {
		t.Fatalf("unexpected rounded partial-lot liquidation count: got %d want 1", len(detail.LiquidationSummaries))
	}
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].AllocatedBasis, "3.3333333333333333", "rounded partial-lot allocated basis")
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].NetLiquidationProceeds, "5", "rounded partial-lot net proceeds")
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].GainOrLoss, "1.6666666666666667", "rounded partial-lot gain")
}

// TestCalculateRetainsFragmentLevelPricedLiquidationMatches verifies that one
// multi-fragment priced liquidation carries fragment-level matched proceeds and
// matched gain or loss into the calculated report model.
// Authored by: OpenCode
func TestCalculateRetainsFragmentLevelPricedLiquidationMatches(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	var report, err = reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "frag-buy-2023-001",
			OccurredAt:       "2023-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-frag-001",
			AssetSymbol:      "FRAG",
			AssetName:        "Fragment Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "10",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "frag-buy-2023-002",
			OccurredAt:       "2023-02-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-frag-001",
			AssetSymbol:      "FRAG",
			AssetName:        "Fragment Asset",
			Quantity:         "2",
			OrderCurrency:    "USD",
			OrderGrossValue:  "20",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "10",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "frag-sell-2024-001",
			OccurredAt:       "2024-03-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-frag-001",
			AssetSymbol:      "FRAG",
			AssetName:        "Fragment Asset",
			Quantity:         "2",
			OrderCurrency:    "USD",
			OrderGrossValue:  "30",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "15",
		}),
	))
	if err != nil {
		t.Fatalf("calculate fragment-level priced liquidation report: %v", err)
	}

	var detail = detailSectionByAsset(t, report, "asset-frag-001")
	if len(detail.LiquidationSummaries) != 1 {
		t.Fatalf("unexpected liquidation summary count: got %d want 1", len(detail.LiquidationSummaries))
	}
	var liquidation = detail.LiquidationSummaries[0]
	if len(liquidation.Matches) != 2 {
		t.Fatalf("unexpected liquidation basis-match count: got %d want 2", len(liquidation.Matches))
	}

	assertCalculationDecimalString(t, liquidation.Matches[0].MatchedQuantity, "1", "first matched quantity")
	assertCalculationDecimalString(t, liquidation.Matches[0].MatchedBasis, "10", "first matched basis")
	assertCalculationDecimalPointerString(t, liquidation.Matches[0].MatchedProceeds, "15", "first matched proceeds")
	assertCalculationDecimalPointerString(t, liquidation.Matches[0].MatchedGainOrLoss, "5", "first matched gain or loss")
	assertCalculationDecimalString(t, liquidation.Matches[1].MatchedQuantity, "1", "second matched quantity")
	assertCalculationDecimalString(t, liquidation.Matches[1].MatchedBasis, "10", "second matched basis")
	assertCalculationDecimalPointerString(t, liquidation.Matches[1].MatchedProceeds, "15", "second matched proceeds")
	assertCalculationDecimalPointerString(t, liquidation.Matches[1].MatchedGainOrLoss, "5", "second matched gain or loss")
}

// TestCalculateRoundsFragmentLevelProceedsWithRoundedIntermediate verifies the
// rounded proceeds-per-unit intermediate is applied before per-fragment
// multiplication for repeating proportional-proceeds allocation.
// Authored by: OpenCode
func TestCalculateRoundsFragmentLevelProceedsWithRoundedIntermediate(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	var report, err = reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "round-buy-2023-001",
			OccurredAt:       "2023-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-round-001",
			AssetSymbol:      "RND",
			AssetName:        "Rounded Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "0.4",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "0.4",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "round-buy-2023-002",
			OccurredAt:       "2023-02-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-round-001",
			AssetSymbol:      "RND",
			AssetName:        "Rounded Asset",
			Quantity:         "2",
			OrderCurrency:    "USD",
			OrderGrossValue:  "0.6",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "0.3",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "round-sell-2024-001",
			OccurredAt:       "2024-03-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-round-001",
			AssetSymbol:      "RND",
			AssetName:        "Rounded Asset",
			Quantity:         "3",
			OrderCurrency:    "USD",
			OrderGrossValue:  "2",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "0.6666666666666666666666666666666667",
		}),
	))
	if err != nil {
		t.Fatalf("calculate rounded fragment-level proceeds report: %v", err)
	}

	var detail = detailSectionByAsset(t, report, "asset-round-001")
	var liquidation = detail.LiquidationSummaries[0]
	if len(liquidation.Matches) != 2 {
		t.Fatalf("unexpected liquidation basis-match count: got %d want 2", len(liquidation.Matches))
	}

	assertCalculationDecimalPointerString(t, liquidation.Matches[0].MatchedProceeds, "0.6666666666666667", "rounded first matched proceeds")
	assertCalculationDecimalPointerString(t, liquidation.Matches[1].MatchedProceeds, "1.3333333333333334", "rounded second matched proceeds")
	assertCalculationDecimalPointerString(t, liquidation.Matches[0].MatchedGainOrLoss, "0.2666666666666667", "rounded first matched gain or loss")
	assertCalculationDecimalPointerString(t, liquidation.Matches[1].MatchedGainOrLoss, "0.7333333333333334", "rounded second matched gain or loss")
	assertCalculationDecimalString(t, liquidation.AllocatedBasis, "1", "rounded fragment allocated basis")
	assertCalculationDecimalString(t, liquidation.NetLiquidationProceeds, "2", "rounded fragment net liquidation proceeds")
	assertCalculationDecimalString(t, liquidation.GainOrLoss, "1", "rounded fragment overall gain")
}

// calculationActivityInput stores one compact report-calculation activity test
// declaration before conversion into the normalized sync model.
// Authored by: OpenCode
type calculationActivityInput struct {
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
	Comment          string
}

// calculationCache creates one protected-activity cache fixture for calculation
// tests.
// Authored by: OpenCode
func calculationCache(reportYear int, activities ...syncmodel.ActivityRecord) syncmodel.ProtectedActivityCache {
	return syncmodel.ProtectedActivityCache{
		ActivityCount:        len(activities),
		AvailableReportYears: []int{reportYear},
		Activities:           activities,
	}
}

// calculationActivity converts one compact test declaration into the normalized
// activity model used by the calculator.
// Authored by: OpenCode
func calculationActivity(t *testing.T, input calculationActivityInput) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         input.SourceID,
		OccurredAt:       input.OccurredAt,
		ActivityType:     input.ActivityType,
		AssetIdentityKey: input.AssetIdentityKey,
		AssetSymbol:      input.AssetSymbol,
		AssetName:        input.AssetName,
		Quantity:         mustCalculationDecimal(t, input.Quantity),
		OrderCurrency:    input.OrderCurrency,
		OrderGrossValue:  calculationDecimalPointer(t, input.OrderGrossValue),
		OrderFeeAmount:   calculationDecimalPointer(t, input.OrderFeeAmount),
		OrderUnitPrice:   calculationDecimalPointer(t, input.OrderUnitPrice),
		Comment:          input.Comment,
	}
}

// mustReportRequest creates one validated report request for calculation tests.
// Authored by: OpenCode
func mustReportRequest(t *testing.T, year int, method reportmodel.CostBasisMethod) reportmodel.ReportRequest {
	t.Helper()

	var request, err = reportmodel.NewReportRequest(year, method, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	return request
}

// summaryEntryByAsset returns one summary entry by asset identity key.
// Authored by: OpenCode
func summaryEntryByAsset(t *testing.T, report reportmodel.CapitalGainsReport, assetIdentityKey string) reportmodel.AssetSummaryEntry {
	t.Helper()

	for _, entry := range report.SummaryEntries {
		if entry.AssetIdentityKey == assetIdentityKey {
			return entry
		}
	}

	t.Fatalf("summary entry %q not found", assetIdentityKey)
	return reportmodel.AssetSummaryEntry{}
}

// referenceEntryByAsset returns one reference entry by asset identity key.
// Authored by: OpenCode
func referenceEntryByAsset(t *testing.T, report reportmodel.CapitalGainsReport, assetIdentityKey string) reportmodel.ReferenceLiquidationEntry {
	t.Helper()

	for _, entry := range report.ReferenceEntries {
		if entry.AssetIdentityKey == assetIdentityKey {
			return entry
		}
	}

	t.Fatalf("reference entry %q not found", assetIdentityKey)
	return reportmodel.ReferenceLiquidationEntry{}
}

// detailSectionByAsset returns one detail section by asset identity key.
// Authored by: OpenCode
func detailSectionByAsset(t *testing.T, report reportmodel.CapitalGainsReport, assetIdentityKey string) reportmodel.AssetDetailSection {
	t.Helper()

	for _, section := range report.DetailSections {
		if section.AssetIdentityKey == assetIdentityKey {
			return section
		}
	}

	t.Fatalf("detail section %q not found", assetIdentityKey)
	return reportmodel.AssetDetailSection{}
}

// assertCalculationDecimalString verifies one exact decimal value against its
// canonical string form.
// Authored by: OpenCode
func assertCalculationDecimalString(t *testing.T, value apd.Decimal, want string, label string) {
	t.Helper()

	var canonical, err = decimalsupport.CanonicalString(value)
	if err != nil {
		t.Fatalf("canonicalize %s: %v", label, err)
	}
	if canonical != want {
		t.Fatalf("unexpected %s: got %q want %q", label, canonical, want)
	}
}

// assertCalculationDecimalPointerString verifies one optional exact decimal
// pointer against its canonical string form.
// Authored by: OpenCode
func assertCalculationDecimalPointerString(t *testing.T, value *apd.Decimal, want string, label string) {
	t.Helper()

	if value == nil {
		t.Fatalf("expected %s to be present", label)
	}

	var canonical, err = decimalsupport.CanonicalString(*value)
	if err != nil {
		t.Fatalf("canonicalize %s: %v", label, err)
	}
	if canonical != want {
		t.Fatalf("unexpected %s: got %q want %q", label, canonical, want)
	}
}

// calculationDecimalPointer parses one optional calculation-fixture decimal.
// Authored by: OpenCode
func calculationDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	if raw == "" {
		return nil
	}

	var value = mustCalculationDecimal(t, raw)
	return &value
}

// mustCalculationDecimal parses one decimal fixture for calculation tests.
// Authored by: OpenCode
func mustCalculationDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", raw, err)
	}

	return value
}
