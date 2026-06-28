// Package markdown verifies package-local rendering helper fallbacks and
// sanitization.
// Authored by: OpenCode
package markdown

import (
	"errors"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
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

	var rowWithPreservedZeroValues = reportmodel.AssetActivityRow{
		UnitPrice:  apdDecimalPointer(0),
		GrossValue: apdDecimalPointer(0),
		FeeAmount:  apdDecimalPointer(0),
	}
	if activityCurrencyColumn(rowWithPreservedZeroValues) != "" {
		t.Fatalf("expected preserved zero-valued holding reduction fields to keep activity currency blank")
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

	var convertedRow = reportmodel.AssetActivityRow{GrossValue: apdDecimalPointer(1), ActivityCurrency: "EUR", CalculationCurrency: "USD"}
	if got := conversionStatusColumn(convertedRow); got != string(reportmodel.ConversionStatusConverted) {
		t.Fatalf("expected converted status, got %q", got)
	}
	var explicitlyConvertedRow = reportmodel.AssetActivityRow{GrossValue: apdDecimalPointer(1), ActivityCurrency: "USD", CalculationCurrency: "USD", ConversionStatus: reportmodel.ConversionStatusConverted}
	if got := conversionStatusColumn(explicitlyConvertedRow); got != string(reportmodel.ConversionStatusConverted) {
		t.Fatalf("expected explicit converted status to override currency inference, got %q", got)
	}
	var blankRow = reportmodel.AssetActivityRow{ActivityCurrency: "EUR", CalculationCurrency: "USD"}
	if got := conversionStatusColumn(blankRow); got != "" {
		t.Fatalf("expected blank status without rendered activity currency, got %q", got)
	}

	if got := rateAuthorityLabel(reportmodel.RateAuthorityEuropeanCentralBank); got != "European Central Bank" {
		t.Fatalf("unexpected ECB authority label %q", got)
	}
	if got := rateAuthorityLabel(reportmodel.RateAuthorityFederalReserve); got != "Federal Reserve" {
		t.Fatalf("unexpected Federal Reserve authority label %q", got)
	}
	if got := rateAuthorityLabel(reportmodel.RateAuthority("custom|authority")); got != "custom\\|authority" {
		t.Fatalf("unexpected custom authority fallback %q", got)
	}
	if got := rateProviderLabel(reportmodel.RateProviderIDECBEXR); !strings.Contains(got, "ECB Data Portal") {
		t.Fatalf("unexpected ECB provider label %q", got)
	}
	if got := rateProviderLabel(reportmodel.RateProviderIDFederalReserveH10); !strings.Contains(got, "Federal Reserve Board") {
		t.Fatalf("unexpected Federal Reserve provider label %q", got)
	}
	if got := rateProviderLabel(reportmodel.RateProviderID("custom|provider")); got != "custom\\|provider" {
		t.Fatalf("unexpected provider fallback %q", got)
	}
	if got := unavailableDateRule(reportmodel.RateProviderIDECBEXR); !strings.Contains(got, "ECB observation") {
		t.Fatalf("unexpected ECB unavailable-date rule %q", got)
	}
	if got := unavailableDateRule(reportmodel.RateProviderIDFederalReserveH10); !strings.Contains(got, "H.10 observation") {
		t.Fatalf("unexpected Federal Reserve unavailable-date rule %q", got)
	}
	if got := unavailableDateRule(reportmodel.RateProviderID("custom")); !strings.Contains(got, "official observation") {
		t.Fatalf("unexpected fallback unavailable-date rule %q", got)
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

	t.Run("closing position invalid decimal", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var section = reportmodel.AssetDetailSection{
			AssetIdentityKey: "asset-2",
			DisplayLabel:     "Asset 2",
			OpeningQuantity:  *apd.New(0, 0),
			OpeningCostBasis: *apd.New(0, 0),
			ClosingQuantity:  *apd.New(0, 0),
			ClosingCostBasis: invalid,
		}

		var err = writeDetailSection(&builder, section, "USD")
		if err == nil || !strings.Contains(err.Error(), `render closing position for "asset-2"`) {
			t.Fatalf("expected wrapped closing-position error, got %v", err)
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
				ActivityType:        reportmodel.ActivityTypeBuy,
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

	t.Run("activity row invalid optional unit price", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var err = writeActivityBlock(&builder, reportmodel.AssetDetailSection{
			ActivityRows: []reportmodel.AssetActivityRow{{
				SourceID:            "row-unit-price",
				OccurredAt:          time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
				ActivityType:        reportmodel.ActivityTypeBuy,
				Quantity:            *apd.New(1, 0),
				UnitPrice:           &invalid,
				BasisAfterRow:       *apd.New(1, 0),
				CalculationCurrency: "USD",
				QuantityAfterRow:    *apd.New(1, 0),
			}},
		})
		if err == nil || !strings.Contains(err.Error(), `render activity row "row-unit-price" unit price`) {
			t.Fatalf("expected wrapped activity-row unit-price error, got %v", err)
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

	t.Run("liquidation block wrapper error", func(t *testing.T) {
		var builder strings.Builder
		var invalid apd.Decimal
		invalid.Form = apd.Infinite

		var section = reportmodel.AssetDetailSection{
			AssetIdentityKey: "asset-3",
			DisplayLabel:     "Asset 3",
			OpeningQuantity:  *apd.New(0, 0),
			OpeningCostBasis: *apd.New(0, 0),
			ClosingQuantity:  *apd.New(0, 0),
			ClosingCostBasis: *apd.New(0, 0),
			LiquidationSummaries: []reportmodel.LiquidationCalculation{{
				SourceID:               "sell-wrap",
				OccurredAt:             time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
				DisposedQuantity:       *apd.New(1, 0),
				AllocatedBasis:         invalid,
				NetLiquidationProceeds: *apd.New(1, 0),
				GainOrLoss:             *apd.New(0, 0),
				ActivityCurrency:       "USD",
			}},
		}

		var err = writeDetailSection(&builder, section, "USD")
		if err == nil || !strings.Contains(err.Error(), `render liquidation calculations for "asset-3"`) {
			t.Fatalf("expected wrapped liquidation-block error, got %v", err)
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

// TestRenderWrapsInjectedHelperFailures verifies exported Render wrapper
// branches through package-local test seams.
// Authored by: OpenCode
func TestRenderWrapsInjectedHelperFailures(t *testing.T) {
	t.Parallel()

	var request, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportmodel.ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	var report, reportErr = reportmodel.NewCapitalGainsReport(request, request.RequestedAt, "USD", nil, *apd.New(0, 0), nil, nil)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}

	t.Run("summary failure propagates from Render", func(t *testing.T) {
		var previous = renderWriteSummarySection
		defer func() {
			renderWriteSummarySection = previous
		}()

		renderWriteSummarySection = func(*strings.Builder, reportmodel.CapitalGainsReport, string) error {
			return errors.New("summary boom")
		}

		if _, renderErr := Render(report); renderErr == nil || !strings.Contains(renderErr.Error(), "summary boom") {
			t.Fatalf("expected summary helper failure to propagate, got %v", renderErr)
		}
	})

	t.Run("reference failure propagates from Render", func(t *testing.T) {
		var previousSummary = renderWriteSummarySection
		var previousReference = renderWriteReferenceSection
		defer func() {
			renderWriteSummarySection = previousSummary
			renderWriteReferenceSection = previousReference
		}()

		renderWriteSummarySection = writeSummarySection
		renderWriteReferenceSection = func(*strings.Builder, reportmodel.CapitalGainsReport) error {
			return errors.New("reference boom")
		}

		if _, renderErr := Render(report); renderErr == nil || !strings.Contains(renderErr.Error(), "reference boom") {
			t.Fatalf("expected reference helper failure to propagate, got %v", renderErr)
		}
	})

	t.Run("rate source failure propagates from Render", func(t *testing.T) {
		var previousSummary = renderWriteSummarySection
		var previousRateSource = renderWriteRateSourceSummary
		defer func() {
			renderWriteSummarySection = previousSummary
			renderWriteRateSourceSummary = previousRateSource
		}()

		renderWriteSummarySection = writeSummarySection
		renderWriteRateSourceSummary = func(*strings.Builder, reportmodel.CapitalGainsReport) error {
			return errors.New("rate source boom")
		}

		if _, renderErr := Render(report); renderErr == nil || !strings.Contains(renderErr.Error(), "rate source boom") {
			t.Fatalf("expected rate source helper failure to propagate, got %v", renderErr)
		}
	})

	t.Run("detail failure propagates from Render", func(t *testing.T) {
		var previousSummary = renderWriteSummarySection
		var previousReference = renderWriteReferenceSection
		var previousDetails = renderWriteDetailSections
		defer func() {
			renderWriteSummarySection = previousSummary
			renderWriteReferenceSection = previousReference
			renderWriteDetailSections = previousDetails
		}()

		renderWriteSummarySection = writeSummarySection
		renderWriteReferenceSection = writeReferenceSection
		renderWriteDetailSections = func(*strings.Builder, reportmodel.CapitalGainsReport, string) error {
			return errors.New("detail boom")
		}

		if _, renderErr := Render(report); renderErr == nil || !strings.Contains(renderErr.Error(), "detail boom") {
			t.Fatalf("expected detail helper failure to propagate, got %v", renderErr)
		}
	})

	t.Run("conversion audit failure propagates from Render", func(t *testing.T) {
		var previousSummary = renderWriteSummarySection
		var previousRateSource = renderWriteRateSourceSummary
		var previousReference = renderWriteReferenceSection
		var previousDetails = renderWriteDetailSections
		var previousAudit = renderWriteConversionAuditSection
		defer func() {
			renderWriteSummarySection = previousSummary
			renderWriteRateSourceSummary = previousRateSource
			renderWriteReferenceSection = previousReference
			renderWriteDetailSections = previousDetails
			renderWriteConversionAuditSection = previousAudit
		}()

		renderWriteSummarySection = writeSummarySection
		renderWriteRateSourceSummary = writeRateSourceSummary
		renderWriteReferenceSection = writeReferenceSection
		renderWriteDetailSections = writeDetailSections
		renderWriteConversionAuditSection = func(*strings.Builder, reportmodel.CapitalGainsReport) error {
			return errors.New("audit boom")
		}

		if _, renderErr := Render(report); renderErr == nil || !strings.Contains(renderErr.Error(), "audit boom") {
			t.Fatalf("expected conversion audit helper failure to propagate, got %v", renderErr)
		}
	})
}

// TestRenderRendersReferenceEmptyState verifies the valid no-reference branch
// in the final Markdown document.
// Authored by: OpenCode
func TestRenderRendersReferenceEmptyState(t *testing.T) {
	var request, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodFIFO, reportmodel.ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	var zero apd.Decimal
	var summaryEntry reportmodel.AssetSummaryEntry
	summaryEntry, err = reportmodel.NewAssetSummaryEntry("asset-1", "Asset 1", zero, "USD")
	if err != nil {
		t.Fatalf("new summary entry: %v", err)
	}
	var section reportmodel.AssetDetailSection
	section, err = reportmodel.NewAssetDetailSection("asset-1", "Asset 1", zero, zero, zero, zero, "USD", nil, nil)
	if err != nil {
		t.Fatalf("new detail section: %v", err)
	}

	var report reportmodel.CapitalGainsReport
	report, err = reportmodel.NewCapitalGainsReport(request, request.RequestedAt, "USD", []reportmodel.AssetSummaryEntry{summaryEntry}, zero, nil, []reportmodel.AssetDetailSection{section})
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
		"| Overall Yearly Net Total | 0 | USD |",
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
		var request, err = reportmodel.NewReportRequest(2024, reportmodel.CostBasisMethodHIFO, reportmodel.ReportBaseCurrencyUSD, time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC))
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
					ActivityType:        reportmodel.ActivityTypeSell,
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
			"| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Quantity After Row | Basis After Row | Original Activity Currency | Calculation Currency | Conversion Status | Note |",
			"| Date | Source ID | Disposed Quantity | Allocated Basis | Net Liquidation Proceeds | Gain Or Loss | Calculation Currency |",
			"| sell-1 | SELL | 1 |  | 12 | 2 | 0 | 0 | USD | USD | same_currency |  |",
			"| sell-1 | 1 | 10 | 10 | 0 | USD |",
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
					ActivityType:        reportmodel.ActivityTypeBuy,
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

// TestRendererRateSourceAndConversionAuditSections verifies provider-level
// disclosure, rate-source aggregation, and grouped audit amount rendering.
// Authored by: OpenCode
func TestRendererRateSourceAndConversionAuditSections(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var report = reportmodel.CapitalGainsReport{
		ReportCalculationCurrency: "USD",
		RateSources: []reportmodel.ExchangeRateEvidence{
			{
				SourceCurrency:   "EUR",
				BaseCurrency:     reportmodel.ReportBaseCurrencyUSD,
				ActivityDate:     activityDate,
				RateDate:         activityDate,
				Authority:        reportmodel.RateAuthorityFederalReserve,
				ProviderID:       reportmodel.RateProviderIDFederalReserveH10,
				RateKind:         "daily noon buying rate",
				QuoteDirection:   reportmodel.QuoteDirectionBasePerSource,
				RateValue:        *apd.New(10946, -4),
				DatasetReference: "H10 fixture",
			},
			{
				SourceCurrency:   "GBP",
				BaseCurrency:     reportmodel.ReportBaseCurrencyUSD,
				ActivityDate:     activityDate,
				RateDate:         activityDate,
				Authority:        reportmodel.RateAuthorityFederalReserve,
				ProviderID:       reportmodel.RateProviderIDFederalReserveH10,
				RateKind:         "daily noon buying rate",
				QuoteDirection:   reportmodel.QuoteDirectionSourcePerBase,
				RateValue:        *apd.New(78, -1),
				DatasetReference: "H10 fixture second rate",
			},
		},
		ConversionAuditEntries: []reportmodel.ConversionAuditEntry{{
			SourceID:           "eur-buy-1",
			AssetLabel:         "BTC",
			ActivityDate:       activityDate,
			SourceCurrency:     "EUR",
			ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
			RateDate:           activityDate,
			RateAuthority:      reportmodel.RateAuthorityFederalReserve,
			RateKind:           "daily noon buying rate",
			RateValue:          *apd.New(10946, -4),
			QuoteDirection:     reportmodel.QuoteDirectionBasePerSource,
			Amounts: []reportmodel.ConvertedActivityAmount{
				{
					AmountKind:      reportmodel.ConvertedAmountKindUnitPrice,
					OriginalAmount:  *apd.New(100, 0),
					ConvertedAmount: *apd.New(10946, -2),
				},
				{
					AmountKind:      reportmodel.ConvertedAmountKindGrossValue,
					OriginalAmount:  *apd.New(200, 0),
					ConvertedAmount: *apd.New(21892, -2),
				},
				{
					AmountKind:      reportmodel.ConvertedAmountKindFeeAmount,
					OriginalAmount:  *apd.New(0, 0),
					ConvertedAmount: *apd.New(0, 0),
				},
			},
		}},
	}

	var builder strings.Builder
	if err := writeRateSourceSummary(&builder, report); err != nil {
		t.Fatalf("write rate source summary: %v", err)
	}
	var summary = builder.String()
	if strings.Count(summary, "- Authority: Federal Reserve") != 1 {
		t.Fatalf("expected provider-level rate source to render once, got %q", summary)
	}
	for _, expected := range []string{"Report Base Currency: USD", "Federal Reserve Board H.10", "most recent previous available H.10 observation"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("expected rate source summary to contain %q, got %q", expected, summary)
		}
	}
	for _, excluded := range []string{"Quote Direction", "Rate Value", "1.0946", "7.8", "base_per_source", "source_per_base"} {
		if strings.Contains(summary, excluded) {
			t.Fatalf("expected rate source summary to exclude %q, got %q", excluded, summary)
		}
	}

	builder.Reset()
	if err := writeConversionAuditSection(&builder, report); err != nil {
		t.Fatalf("write conversion audit section: %v", err)
	}
	var audit = builder.String()
	for _, expected := range []string{"## Currency Conversion Audit", "eur-buy-1", "base_per_source", "Rate Value", "1.0946", "unit_price: 100 -> 109.46; gross_value: 200 -> 218.92"} {
		if !strings.Contains(audit, expected) {
			t.Fatalf("expected conversion audit to contain %q, got %q", expected, audit)
		}
	}
	for _, excluded := range []string{"Rate Authority", "Rate Kind", "Federal Reserve", "daily noon buying rate", "fee_amount", "0 -> 0"} {
		if strings.Contains(audit, excluded) {
			t.Fatalf("expected conversion audit to exclude provider-level field %q, got %q", excluded, audit)
		}
	}
	var expectedHeader = "| Date | Source ID | Asset | Rate Date | Source Currency | Report Base Currency | Converted Amounts | Quote Direction | Rate Value |"
	var expectedRow = "| 2024-01-05 | eur-buy-1 | BTC | 2024-01-05 | EUR | USD | unit_price: 100 -> 109.46; gross_value: 200 -> 218.92 | base_per_source | 1.0946 |"
	if !strings.Contains(audit, expectedHeader) || !strings.Contains(audit, expectedRow) {
		t.Fatalf("expected grouped conversion audit order, got %q", audit)
	}
	if strings.Count(audit, "| 2024-01-05 | eur-buy-1 |") != 1 {
		t.Fatalf("expected one grouped audit row for the source activity, got %q", audit)
	}
}

// TestRendererUsesPreservedConversionStatusForAssetDetails verifies BUG-006
// rendering does not infer same-currency from post-conversion report currency.
// Authored by: OpenCode
func TestRendererUsesPreservedConversionStatusForAssetDetails(t *testing.T) {
	t.Parallel()

	var builder strings.Builder
	var section = reportmodel.AssetDetailSection{
		AssetIdentityKey:    "asset-btc",
		DisplayLabel:        "BTC",
		OpeningQuantity:     *apd.New(1, 0),
		OpeningCostBasis:    *apd.New(10, 0),
		ClosingQuantity:     *apd.New(0, 0),
		ClosingCostBasis:    *apd.New(0, 0),
		CalculationCurrency: "USD",
		ActivityRows: []reportmodel.AssetActivityRow{{
			SourceID:            "audited-converted-row",
			OccurredAt:          time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
			ActivityType:        reportmodel.ActivityTypeSell,
			Quantity:            *apd.New(1, 0),
			GrossValue:          apdDecimalPointer(12),
			ActivityCurrency:    "USD",
			BasisAfterRow:       *apd.New(0, 0),
			CalculationCurrency: "USD",
			QuantityAfterRow:    *apd.New(0, 0),
			ConversionStatus:    reportmodel.ConversionStatusConverted,
		}},
	}

	if err := writeDetailSection(&builder, section, "USD"); err != nil {
		t.Fatalf("write detail section: %v", err)
	}
	var rendered = builder.String()
	if !strings.Contains(rendered, "| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Quantity After Row | Basis After Row | Original Activity Currency | Calculation Currency | Conversion Status | Note |") {
		t.Fatalf("expected BUG-007 activity header, got %q", rendered)
	}
	if strings.Contains(rendered, "| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Activity Currency |") {
		t.Fatalf("expected old activity currency header to be absent, got %q", rendered)
	}
	if !strings.Contains(rendered, "| audited-converted-row | SELL | 1 |  | 12 |  | 0 | 0 | USD | USD | converted |  |") {
		t.Fatalf("expected preserved converted status in detail row, got %q", rendered)
	}
	if strings.Contains(rendered, "| audited-converted-row | SELL | 1 |  | 12 |  | 0 | 0 | USD | USD | same_currency |  |") {
		t.Fatalf("expected audited converted row not to render as same-currency, got %q", rendered)
	}
}

// TestRendererAssetDetailCurrencyColumnContracts verifies BUG-007 activity and
// liquidation table currency-column placement.
// Authored by: OpenCode
func TestRendererAssetDetailCurrencyColumnContracts(t *testing.T) {
	t.Parallel()

	var builder strings.Builder
	var section = reportmodel.AssetDetailSection{
		AssetIdentityKey:    "asset-eth",
		DisplayLabel:        "ETH",
		OpeningQuantity:     *apd.New(1, 0),
		OpeningCostBasis:    *apd.New(10, 0),
		ClosingQuantity:     *apd.New(0, 0),
		ClosingCostBasis:    *apd.New(0, 0),
		CalculationCurrency: "EUR",
		ActivityRows: []reportmodel.AssetActivityRow{{
			SourceID:            "eth-sell",
			OccurredAt:          time.Date(2024, time.March, 2, 3, 4, 5, 0, time.UTC),
			ActivityType:        reportmodel.ActivityTypeSell,
			Quantity:            *apd.New(2, 0),
			UnitPrice:           apdDecimalPointer(100),
			GrossValue:          apdDecimalPointer(200),
			FeeAmount:           apdDecimalPointer(1),
			ActivityCurrency:    "USD",
			BasisAfterRow:       *apd.New(50, 0),
			CalculationCurrency: "EUR",
			QuantityAfterRow:    *apd.New(3, 0),
			ConversionStatus:    reportmodel.ConversionStatusConverted,
		}},
		LiquidationSummaries: []reportmodel.LiquidationCalculation{{
			SourceID:               "eth-sell",
			OccurredAt:             time.Date(2024, time.March, 2, 3, 4, 5, 0, time.UTC),
			DisposedQuantity:       *apd.New(2, 0),
			AllocatedBasis:         *apd.New(50, 0),
			NetLiquidationProceeds: *apd.New(199, 0),
			GainOrLoss:             *apd.New(149, 0),
			ActivityCurrency:       "USD",
			CalculationCurrency:    "EUR",
		}},
	}

	if err := writeDetailSection(&builder, section, "EUR"); err != nil {
		t.Fatalf("write detail section: %v", err)
	}
	var rendered = builder.String()
	for _, expected := range []string{
		"| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Quantity After Row | Basis After Row | Original Activity Currency | Calculation Currency | Conversion Status | Note |",
		"| eth-sell | SELL | 2 | 100 | 200 | 1 | 3 | 50 | USD | EUR | converted |  |",
		"| Date | Source ID | Disposed Quantity | Allocated Basis | Net Liquidation Proceeds | Gain Or Loss | Calculation Currency |",
		"| eth-sell | 2 | 50 | 199 | 149 | EUR |",
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected rendered detail section to contain %q, got %q", expected, rendered)
		}
	}
	for _, excluded := range []string{
		"| Date | Source ID | Type | Quantity | Unit Price | Gross Value | Fee | Activity Currency |",
		"| Date | Source ID | Disposed Quantity | Activity Currency | Allocated Basis |",
	} {
		if strings.Contains(rendered, excluded) {
			t.Fatalf("expected rendered detail section to exclude %q, got %q", excluded, rendered)
		}
	}
}

// TestRendererRateSourceAndConversionAuditErrors verifies invalid decimals in
// rate-source and conversion-audit sections are wrapped with row context.
// Authored by: OpenCode
func TestRendererRateSourceAndConversionAuditErrors(t *testing.T) {
	t.Parallel()

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	var report = reportmodel.CapitalGainsReport{ConversionAuditEntries: []reportmodel.ConversionAuditEntry{{RateValue: invalid}}}
	var builder strings.Builder
	builder.Reset()
	if err := writeConversionAuditSection(&builder, report); err == nil || !strings.Contains(err.Error(), "render conversion audit entry 0 rate value") {
		t.Fatalf("expected audit rate invalid-decimal error, got %v", err)
	}

	report = reportmodel.CapitalGainsReport{ConversionAuditEntries: []reportmodel.ConversionAuditEntry{{
		RateValue: *apd.New(1, 0),
		Amounts:   []reportmodel.ConvertedActivityAmount{{OriginalAmount: invalid}},
	}}}
	builder.Reset()
	if err := writeConversionAuditSection(&builder, report); err == nil || !strings.Contains(err.Error(), "amount 0 original amount") {
		t.Fatalf("expected audit original amount invalid-decimal error, got %v", err)
	}

	report.ConversionAuditEntries[0].Amounts[0].OriginalAmount = *apd.New(1, 0)
	report.ConversionAuditEntries[0].Amounts[0].ConvertedAmount = invalid
	builder.Reset()
	if err := writeConversionAuditSection(&builder, report); err == nil || !strings.Contains(err.Error(), "amount 0 converted amount") {
		t.Fatalf("expected audit converted amount invalid-decimal error, got %v", err)
	}
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
		ActivityType:        reportmodel.ActivityTypeBuy,
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
		ActivityType:        reportmodel.ActivityTypeBuy,
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
		ActivityType:        reportmodel.ActivityTypeBuy,
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
		ActivityType:        reportmodel.ActivityTypeBuy,
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
