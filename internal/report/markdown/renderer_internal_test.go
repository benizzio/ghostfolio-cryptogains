// Package markdown verifies package-local rendering helper fallbacks and
// sanitization.
// Authored by: OpenCode
package markdown

import (
	"fmt"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// TestRendererHelperFallbacks verifies default labels, display-label fallback,
// activity-currency blanking, and inline sanitization.
// Authored by: OpenCode
func TestRendererHelperFallbacks(t *testing.T) {
	if calculationCurrencyLabel("") != notApplicableCalculationCurrency {
		t.Fatalf("expected empty calculation currency to fall back to %q", notApplicableCalculationCurrency)
	}
	if calculationCurrencyLabelWithFallback("", " USD\n") != "USD" {
		t.Fatalf("expected empty row calculation currency to fall back to the report currency")
	}
	if renderDisplayLabel("", " asset-key\n") != "asset-key" {
		t.Fatalf("expected missing display label to fall back to asset identity key")
	}
	if renderDisplayLabel("\n\t", "\r") != "Unknown Asset" {
		t.Fatalf("expected missing display label and asset key to fall back to Unknown Asset")
	}

	var rowWithoutMonetaryContext = reportmodel.AssetActivityRow{ActivityCurrency: "USD"}
	if activityCurrencyColumn(rowWithoutMonetaryContext) != "" {
		t.Fatalf("expected row without monetary context to leave activity currency blank")
	}

	var pricedValue = *apd.New(1, 0)
	var rowWithMonetaryContext = reportmodel.AssetActivityRow{
		GrossValue:       &pricedValue,
		ActivityCurrency: " US|D\n",
	}
	if activityCurrencyColumn(rowWithMonetaryContext) != "US\\|D" {
		t.Fatalf("expected activity currency to be sanitized when monetary context exists")
	}

	var sanitized = sanitizeInlineText("Bearer secret-token\nlabel\t| token=abc")
	if strings.Contains(sanitized, "secret-token") || strings.Contains(sanitized, "abc") {
		t.Fatalf("expected secret-shaped substrings to be redacted, got %q", sanitized)
	}
	if !strings.Contains(sanitized, "Bearer [REDACTED]") || !strings.Contains(sanitized, "\\|") {
		t.Fatalf("expected sanitization to preserve redaction and pipe escaping, got %q", sanitized)
	}
}

// TestRendererInternalErrorBranches verifies internal helper failures for
// summary, activity, liquidation, and position rendering.
// Authored by: OpenCode
func TestRendererInternalErrorBranches(t *testing.T) {
	t.Run("summary entry invalid decimal", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var report = reportmodel.CapitalGainsReport{
			SummaryEntries: []reportmodel.AssetSummaryEntry{{
				AssetIdentityKey:          "asset-1",
				DisplayLabel:              "Asset 1",
				NetGainOrLoss:             invalid,
				ReportCalculationCurrency: "USD",
			}},
		}

		var err = writeSummarySection(&builder, report, "USD")
		if err == nil || !strings.Contains(err.Error(), `render summary entry "asset-1" net gain or loss`) {
			t.Fatalf("expected wrapped summary-entry error, got %v", err)
		}
	})

	t.Run("yearly total invalid decimal", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var report = reportmodel.CapitalGainsReport{YearlyNetTotal: invalid}
		var err = writeSummarySection(&builder, report, "USD")
		if err == nil || !strings.Contains(err.Error(), "render yearly net total") {
			t.Fatalf("expected wrapped yearly-total error, got %v", err)
		}
	})

	t.Run("opening position invalid decimal", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var section = reportmodel.AssetDetailSection{
			AssetIdentityKey: "asset-1",
			DisplayLabel:     "Asset 1",
			OpeningQuantity:  invalid,
		}

		var err = writeDetailSection(&builder, section, "USD")
		if err == nil || !strings.Contains(err.Error(), `render opening position for "asset-1"`) {
			t.Fatalf("expected wrapped opening-position error, got %v", err)
		}
	})

	t.Run("activity row invalid optional decimal", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var err = writeActivityBlock(&builder, reportmodel.AssetDetailSection{
			ActivityRows: []reportmodel.AssetActivityRow{{
				SourceID:            "row-1",
				OccurredAt:          time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
				ActivityType:        syncmodel.ActivityTypeBuy,
				Quantity:            *apd.New(1, 0),
				GrossValue:          &invalid,
				BasisAfterRow:       *apd.New(1, 0),
				CalculationCurrency: "USD",
				QuantityAfterRow:    *apd.New(1, 0),
			}},
		})
		if err == nil || !strings.Contains(err.Error(), `render activity row "row-1" gross value`) {
			t.Fatalf("expected wrapped activity-row error, got %v", err)
		}
	})

	t.Run("liquidation invalid decimal", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var err = writeLiquidationBlock(&builder, reportmodel.AssetDetailSection{
			LiquidationSummaries: []reportmodel.LiquidationCalculation{{
				SourceID:               "sell-1",
				OccurredAt:             time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
				DisposedQuantity:       *apd.New(1, 0),
				AllocatedBasis:         invalid,
				NetLiquidationProceeds: *apd.New(1, 0),
				GainOrLoss:             *apd.New(0, 0),
				ActivityCurrency:       "USD",
			}},
		}, "USD")
		if err == nil || !strings.Contains(err.Error(), `render liquidation "sell-1" allocated basis`) {
			t.Fatalf("expected wrapped liquidation error, got %v", err)
		}
	})
}

// TestRenderRejectsInvalidReport verifies exported rendering stops at report
// validation before helper rendering starts.
// Authored by: OpenCode
func TestRenderRejectsInvalidReport(t *testing.T) {
	_, err := Render(reportmodel.CapitalGainsReport{})
	if err == nil || !strings.Contains(err.Error(), "capital gains report year must be greater than zero") {
		t.Fatalf("expected report validation error, got %v", err)
	}
}

// TestRenderRendersReferenceEmptyState verifies the valid no-reference branch
// in the final Markdown document.
// Authored by: OpenCode
func TestRenderRendersReferenceEmptyState(t *testing.T) {
	var request, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	var zero apd.Decimal
	var summaryEntry reportmodel.AssetSummaryEntry
	summaryEntry, err = reportmodel.NewAssetSummaryEntry("asset-1", "Asset 1", zero, "")
	if err != nil {
		t.Fatalf("new summary entry: %v", err)
	}
	var section reportmodel.AssetDetailSection
	section, err = reportmodel.NewAssetDetailSection("asset-1", "Asset 1", zero, zero, zero, zero, "", nil, nil)
	if err != nil {
		t.Fatalf("new detail section: %v", err)
	}

	var report reportmodel.CapitalGainsReport
	report, err = reportmodel.NewCapitalGainsReport(request, request.RequestedAt, "", []reportmodel.AssetSummaryEntry{summaryEntry}, zero, nil, []reportmodel.AssetDetailSection{section})
	if err != nil {
		t.Fatalf("new capital gains report: %v", err)
	}

	var document reportmodel.ReportDocument
	document, err = Render(report)
	if err != nil {
		t.Fatalf("render report: %v", err)
	}

	for _, expected := range []string{
		"## Reference Section",
		"No assets reached full liquidation by year end.",
		fmt.Sprintf("| Overall Yearly Net Total | 0 | %s |", notApplicableCalculationCurrency),
	} {
		if !strings.Contains(document.Content, expected) {
			t.Fatalf("expected rendered document to contain %q", expected)
		}
	}
}

// TestRenderCoversDetailAndLiquidationBranches verifies successful non-empty
// detail rendering plus remaining helper failure branches.
// Authored by: OpenCode
func TestRenderCoversDetailAndLiquidationBranches(t *testing.T) {
	t.Run("renders full detail and liquidation sections", func(t *testing.T) {
		var request, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodHIFO, time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("new report request: %v", err)
		}

		var report, reportErr = reportmodel.NewCapitalGainsReport(
			request,
			request.RequestedAt,
			"USD",
			[]reportmodel.AssetSummaryEntry{{
				AssetIdentityKey:          "asset-btc",
				DisplayLabel:              "BTC",
				NetGainOrLoss:             *apd.New(2, 0),
				ReportCalculationCurrency: "USD",
			}},
			*apd.New(2, 0),
			[]reportmodel.ReferenceLiquidationEntry{{
				AssetIdentityKey:                   "asset-btc",
				DisplayLabel:                       "BTC",
				FullLiquidationCountThroughYearEnd: 1,
				MainSectionStatus:                  reportmodel.ReferenceSectionStatusIncludedInMainSections,
			}},
			[]reportmodel.AssetDetailSection{{
				AssetIdentityKey:    "asset-btc",
				DisplayLabel:        "BTC",
				OpeningQuantity:     *apd.New(1, 0),
				OpeningCostBasis:    *apd.New(10, 0),
				ClosingQuantity:     *apd.New(0, 0),
				ClosingCostBasis:    *apd.New(0, 0),
				CalculationCurrency: "USD",
				ActivityRows: []reportmodel.AssetActivityRow{{
					SourceID:            "sell-1",
					OccurredAt:          time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
					ActivityType:        syncmodel.ActivityTypeSell,
					Quantity:            *apd.New(1, 0),
					GrossValue:          apdDecimalPointer(12),
					FeeAmount:           apdDecimalPointer(2),
					ActivityCurrency:    "USD",
					BasisAfterRow:       *apd.New(0, 0),
					CalculationCurrency: "USD",
					QuantityAfterRow:    *apd.New(0, 0),
				}},
				LiquidationSummaries: []reportmodel.LiquidationCalculation{{
					SourceID:               "sell-1",
					OccurredAt:             time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
					DisposedQuantity:       *apd.New(1, 0),
					AllocatedBasis:         *apd.New(10, 0),
					NetLiquidationProceeds: *apd.New(10, 0),
					GainOrLoss:             *apd.New(0, 0),
					ActivityCurrency:       "USD",
					CalculationCurrency:    "USD",
				}},
			}},
		)
		if reportErr != nil {
			t.Fatalf("new capital gains report: %v", reportErr)
		}

		var document, renderErr = Render(report)
		if renderErr != nil {
			t.Fatalf("render report: %v", renderErr)
		}
		for _, expected := range []string{
			"## Asset Detail: BTC",
			"### Opening Position",
			"### In-Year Activity",
			"### Liquidation Calculations",
			"### Closing Position",
			"| sell-1 | SELL | 1 | 12 | 2 | USD | 0 | USD | 0 |  |",
		} {
			if !strings.Contains(document.Content, expected) {
				t.Fatalf("expected rendered report to contain %q", expected)
			}
		}
	})

	t.Run("render wraps detail-section failure", func(t *testing.T) {
		var builder strings.Builder
		var err = writeDetailSections(&builder, reportmodel.CapitalGainsReport{
			DetailSections: []reportmodel.AssetDetailSection{{
				AssetIdentityKey:    "asset-btc",
				DisplayLabel:        "BTC",
				OpeningQuantity:     *apd.New(0, 0),
				OpeningCostBasis:    *apd.New(0, 0),
				ClosingQuantity:     *apd.New(0, 0),
				ClosingCostBasis:    *apd.New(0, 0),
				CalculationCurrency: "USD",
				ActivityRows: []reportmodel.AssetActivityRow{{
					SourceID:            "row-1",
					OccurredAt:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
					ActivityType:        syncmodel.ActivityTypeBuy,
					Quantity:            *apd.New(1, 0),
					GrossValue:          infiniteDecimalPointer(),
					BasisAfterRow:       *apd.New(1, 0),
					CalculationCurrency: "USD",
					QuantityAfterRow:    *apd.New(1, 0),
				}},
			}},
		}, "USD")
		if err == nil || !strings.Contains(err.Error(), `render in-year activity for "asset-btc"`) {
			t.Fatalf("expected wrapped detail-section render failure, got %v", err)
		}
	})

	t.Run("liquidation block wraps gain-or-loss failure", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var err = writeLiquidationBlock(&builder, reportmodel.AssetDetailSection{
			LiquidationSummaries: []reportmodel.LiquidationCalculation{{
				SourceID:               "sell-2",
				OccurredAt:             time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
				DisposedQuantity:       *apd.New(1, 0),
				AllocatedBasis:         *apd.New(1, 0),
				NetLiquidationProceeds: *apd.New(2, 0),
				GainOrLoss:             invalid,
				ActivityCurrency:       "USD",
			}},
		}, "USD")
		if err == nil || !strings.Contains(err.Error(), `render liquidation "sell-2" gain or loss`) {
			t.Fatalf("expected wrapped liquidation gain-or-loss error, got %v", err)
		}
	})
}

// TestRendererAdditionalHelperFailures verifies the remaining direct helper
// error branches that exported rendering rejects earlier via report validation.
// Authored by: OpenCode
func TestRendererAdditionalHelperFailures(t *testing.T) {
	t.Parallel()

	var builder strings.Builder
	if err := writePositionBlock(&builder, "Opening Position", *apd.New(1, 0), reportInvalidDecimalForRenderer(), "USD", "USD"); err == nil || !strings.Contains(err.Error(), "render cost basis") {
		t.Fatalf("expected invalid position cost basis to fail, got %v", err)
	}

	builder.Reset()
	if err := writeActivityBlock(&builder, reportmodel.AssetDetailSection{ActivityRows: []reportmodel.AssetActivityRow{{
		SourceID:            "row-quantity",
		OccurredAt:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		ActivityType:        syncmodel.ActivityTypeBuy,
		Quantity:            reportInvalidDecimalForRenderer(),
		BasisAfterRow:       *apd.New(1, 0),
		CalculationCurrency: "USD",
		QuantityAfterRow:    *apd.New(1, 0),
	}}}); err == nil || !strings.Contains(err.Error(), `render activity row "row-quantity" quantity`) {
		t.Fatalf("expected invalid activity quantity to fail, got %v", err)
	}

	builder.Reset()
	if err := writeActivityBlock(&builder, reportmodel.AssetDetailSection{ActivityRows: []reportmodel.AssetActivityRow{{
		SourceID:            "row-fee",
		OccurredAt:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		ActivityType:        syncmodel.ActivityTypeBuy,
		Quantity:            *apd.New(1, 0),
		FeeAmount:           infiniteDecimalPointer(),
		BasisAfterRow:       *apd.New(1, 0),
		CalculationCurrency: "USD",
		QuantityAfterRow:    *apd.New(1, 0),
	}}}); err == nil || !strings.Contains(err.Error(), `render activity row "row-fee" fee`) {
		t.Fatalf("expected invalid activity fee to fail, got %v", err)
	}

	builder.Reset()
	if err := writeActivityBlock(&builder, reportmodel.AssetDetailSection{ActivityRows: []reportmodel.AssetActivityRow{{
		SourceID:            "row-basis",
		OccurredAt:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		ActivityType:        syncmodel.ActivityTypeBuy,
		Quantity:            *apd.New(1, 0),
		BasisAfterRow:       reportInvalidDecimalForRenderer(),
		CalculationCurrency: "USD",
		QuantityAfterRow:    *apd.New(1, 0),
	}}}); err == nil || !strings.Contains(err.Error(), `render activity row "row-basis" basis after row`) {
		t.Fatalf("expected invalid activity basis-after-row to fail, got %v", err)
	}

	builder.Reset()
	if err := writeActivityBlock(&builder, reportmodel.AssetDetailSection{ActivityRows: []reportmodel.AssetActivityRow{{
		SourceID:            "row-after",
		OccurredAt:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		ActivityType:        syncmodel.ActivityTypeBuy,
		Quantity:            *apd.New(1, 0),
		BasisAfterRow:       *apd.New(1, 0),
		CalculationCurrency: "USD",
		QuantityAfterRow:    reportInvalidDecimalForRenderer(),
	}}}); err == nil || !strings.Contains(err.Error(), `render activity row "row-after" quantity after row`) {
		t.Fatalf("expected invalid activity quantity-after-row to fail, got %v", err)
	}

	builder.Reset()
	if err := writeLiquidationBlock(&builder, reportmodel.AssetDetailSection{LiquidationSummaries: []reportmodel.LiquidationCalculation{{
		SourceID:               "sell-qty",
		OccurredAt:             time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
		DisposedQuantity:       reportInvalidDecimalForRenderer(),
		AllocatedBasis:         *apd.New(1, 0),
		NetLiquidationProceeds: *apd.New(2, 0),
		GainOrLoss:             *apd.New(1, 0),
		ActivityCurrency:       "USD",
	}}}, "USD"); err == nil || !strings.Contains(err.Error(), `render liquidation "sell-qty" disposed quantity`) {
		t.Fatalf("expected invalid liquidation quantity to fail, got %v", err)
	}

	builder.Reset()
	if err := writeLiquidationBlock(&builder, reportmodel.AssetDetailSection{LiquidationSummaries: []reportmodel.LiquidationCalculation{{
		SourceID:               "sell-proceeds",
		OccurredAt:             time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
		DisposedQuantity:       *apd.New(1, 0),
		AllocatedBasis:         *apd.New(1, 0),
		NetLiquidationProceeds: reportInvalidDecimalForRenderer(),
		GainOrLoss:             *apd.New(1, 0),
		ActivityCurrency:       "USD",
	}}}, "USD"); err == nil || !strings.Contains(err.Error(), `render liquidation "sell-proceeds" net proceeds`) {
		t.Fatalf("expected invalid liquidation proceeds to fail, got %v", err)
	}
}

// apdDecimalPointer returns one finite decimal pointer for renderer tests.
// Authored by: OpenCode
func apdDecimalPointer(value int64) *apd.Decimal {
	var decimal = *apd.New(value, 0)
	return &decimal
}

// infiniteDecimalPointer returns one non-finite decimal pointer for renderer
// error-path tests.
// Authored by: OpenCode
func infiniteDecimalPointer() *apd.Decimal {
	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	return &invalid
}

// reportInvalidDecimalForRenderer returns one non-finite decimal value for
// direct renderer helper error-path tests.
// Authored by: OpenCode
func reportInvalidDecimalForRenderer() apd.Decimal {
	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	return invalid
}
