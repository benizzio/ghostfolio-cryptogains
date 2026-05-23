// Package model verifies report-model validation helpers and constructors.
// Authored by: OpenCode
package model

import (
	"errors"
	"strings"
	"testing"
	"time"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// reportInvalidDecimal is a non-finite decimal used to exercise finite-value
// guardrails in report-model validation.
// Authored by: OpenCode
var reportInvalidDecimal = apd.Decimal{Form: apd.Infinite}

// TestCalculationErrorHandlesFallbacks verifies structured calculation-error
// fallback messages, references, and nil-safe accessors.
// Authored by: OpenCode
func TestCalculationErrorHandlesFallbacks(t *testing.T) {
	t.Parallel()

	var cause = errors.New("root cause")
	var err = NewCalculationError(CalculationErrorKindBasisAllocation, "  ", " source-1 ", " BTC ", cause)

	if err.Error() != "root cause (asset \"BTC\", source \"source-1\")" {
		t.Fatalf("unexpected calculation error string: %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected calculation error to unwrap cause")
	}
	if err.Kind() != CalculationErrorKindBasisAllocation {
		t.Fatalf("unexpected calculation error kind: %q", err.Kind())
	}
	if err.SourceID() != "source-1" {
		t.Fatalf("unexpected calculation error source ID: %q", err.SourceID())
	}
	if err.DisplayLabel() != "BTC" {
		t.Fatalf("unexpected calculation error display label: %q", err.DisplayLabel())
	}

	var persistedRecord = &syncmodel.ActivityRecord{SourceID: "source-1", AssetSymbol: "BTC"}
	err = err.WithPersistedActivityRecord(persistedRecord)
	var diagnosticContext = err.DiagnosticReportContext()
	if diagnosticContext.FailureDetail == "" || diagnosticContext.OffendingActivityRecord != persistedRecord {
		t.Fatalf("expected calculation error to expose report diagnostic context, got %#v", diagnosticContext)
	}

	err = NewCalculationError(CalculationErrorKindInvalidRequest, "", "", "", nil)
	if err.Error() != "unsupported report calculation" {
		t.Fatalf("expected default calculation error message, got %q", err.Error())
	}

	var nilError *CalculationError
	if nilError.Error() != "" {
		t.Fatalf("expected nil calculation error string to be empty")
	}
	if got := nilError.DiagnosticReportContext(); got.FailureStage != "" || got.FailureDetail != "" || len(got.Records) != 0 || got.OffendingActivityRecord != nil {
		t.Fatalf("expected nil calculation error diagnostic context to be empty, got %#v", got)
	}
	if nilError.Unwrap() != nil {
		t.Fatalf("expected nil calculation error unwrap to be nil")
	}
	if nilError.Kind() != "" {
		t.Fatalf("expected nil calculation error kind to be empty")
	}
	if nilError.SourceID() != "" {
		t.Fatalf("expected nil calculation error source ID to be empty")
	}
	if nilError.DisplayLabel() != "" {
		t.Fatalf("expected nil calculation error display label to be empty")
	}
}

// TestNewReportRequestValidatesRequiredFields verifies the reusable request
// constructor guardrails.
// Authored by: OpenCode
func TestNewReportRequestValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	_, err := NewReportRequest(0, CostBasisMethodFIFO, time.Now())
	if err == nil || !strings.Contains(err.Error(), "year must be greater than zero") {
		t.Fatalf("expected invalid year error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethod("bad"), time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported cost basis method") {
		t.Fatalf("expected invalid method error, got %v", err)
	}

	_, err = NewReportRequest(2024, CostBasisMethodFIFO, time.Time{})
	if err == nil || !strings.Contains(err.Error(), "requested-at timestamp is required") {
		t.Fatalf("expected missing timestamp error, got %v", err)
	}

	request, err := NewReportRequest(2024, CostBasisMethodFIFO, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}
	if err = request.Validate(); err != nil {
		t.Fatalf("validate request: %v", err)
	}
}

// TestNewCapitalGainsReportValidatesNestedContent verifies that the top-level
// report helper rejects invalid nested rows.
// Authored by: OpenCode
func TestNewCapitalGainsReportValidatesNestedContent(t *testing.T) {
	t.Parallel()

	request, err := NewReportRequest(2024, CostBasisMethodHIFO, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	_, err = NewCapitalGainsReport(
		request,
		time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
		"",
		[]AssetSummaryEntry{{
			AssetIdentityKey: "",
			DisplayLabel:     "BTC",
			NetGainOrLoss:    mustReportDecimal(t, "1"),
		}},
		mustReportDecimal(t, "1"),
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "summary entry 0") {
		t.Fatalf("expected nested summary validation failure, got %v", err)
	}

	report, err := NewCapitalGainsReport(
		request,
		time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
		"NOT APPLICABLE",
		[]AssetSummaryEntry{{
			AssetIdentityKey:          "asset-btc",
			DisplayLabel:              "BTC",
			NetGainOrLoss:             mustReportDecimal(t, "1"),
			ReportCalculationCurrency: "NOT APPLICABLE",
		}},
		mustReportDecimal(t, "1"),
		[]ReferenceLiquidationEntry{{
			AssetIdentityKey:                   "asset-btc",
			DisplayLabel:                       "BTC",
			FullLiquidationCountThroughYearEnd: 1,
			MainSectionStatus:                  ReferenceSectionStatusIncludedInMainSections,
		}},
		[]AssetDetailSection{{
			AssetIdentityKey:    "asset-btc",
			DisplayLabel:        "BTC",
			OpeningQuantity:     mustReportDecimal(t, "1"),
			OpeningCostBasis:    mustReportDecimal(t, "10"),
			ClosingQuantity:     mustReportDecimal(t, "0"),
			ClosingCostBasis:    mustReportDecimal(t, "0"),
			CalculationCurrency: "USD",
			ActivityRows: []AssetActivityRow{{
				SourceID:            "sell-1",
				OccurredAt:          time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				ActivityType:        syncmodel.ActivityTypeSell,
				Quantity:            mustReportDecimal(t, "1"),
				GrossValue:          decimalPointer(t, "12"),
				FeeAmount:           decimalPointer(t, "0"),
				BasisAfterRow:       mustReportDecimal(t, "0"),
				CalculationCurrency: "USD",
				QuantityAfterRow:    mustReportDecimal(t, "0"),
				LiquidationCalculation: &LiquidationCalculation{
					SourceID:               "sell-1",
					OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
					DisposedQuantity:       mustReportDecimal(t, "1"),
					AllocatedBasis:         mustReportDecimal(t, "10"),
					NetLiquidationProceeds: mustReportDecimal(t, "12"),
					GainOrLoss:             mustReportDecimal(t, "2"),
					ActivityCurrency:       "USD",
					CalculationCurrency:    "USD",
				},
			}},
			LiquidationSummaries: []LiquidationCalculation{{
				SourceID:               "sell-1",
				OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				DisposedQuantity:       mustReportDecimal(t, "1"),
				AllocatedBasis:         mustReportDecimal(t, "10"),
				NetLiquidationProceeds: mustReportDecimal(t, "12"),
				GainOrLoss:             mustReportDecimal(t, "2"),
				ActivityCurrency:       "USD",
				CalculationCurrency:    "USD",
			}},
		}},
	)
	if err != nil {
		t.Fatalf("new capital gains report: %v", err)
	}
	if err = report.Validate(); err != nil {
		t.Fatalf("validate capital gains report: %v", err)
	}
}

// TestReferenceAndDetailValidationGuardrails verifies remaining report-model
// validation branches for reference, detail, activity, and liquidation rows.
// Authored by: OpenCode
func TestReferenceAndDetailValidationGuardrails(t *testing.T) {
	t.Parallel()

	_, err := NewReferenceLiquidationEntry("asset-btc", "BTC", -1, ReferenceSectionStatusReferenceOnly)
	if err == nil || !strings.Contains(err.Error(), "full liquidation count must not be negative") {
		t.Fatalf("expected negative liquidation count error, got %v", err)
	}

	_, err = NewReferenceLiquidationEntry("asset-btc", "BTC", 0, ReferenceSectionStatus("bad"))
	if err == nil || !strings.Contains(err.Error(), "unsupported reference section status") {
		t.Fatalf("expected unsupported reference status error, got %v", err)
	}

	_, err = NewAssetDetailSection(
		"asset-btc",
		"BTC",
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		"USD",
		[]AssetActivityRow{{
			SourceID:         "sell-1",
			OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			ActivityType:     syncmodel.ActivityType("swap"),
			Quantity:         mustReportDecimal(t, "1"),
			BasisAfterRow:    mustReportDecimal(t, "0"),
			QuantityAfterRow: mustReportDecimal(t, "0"),
		}},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "activity row 0: asset activity row activity type") {
		t.Fatalf("expected unsupported activity type error, got %v", err)
	}

	_, err = NewAssetDetailSection(
		"asset-btc",
		"BTC",
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		"USD",
		nil,
		[]LiquidationCalculation{{
			SourceID:               "sell-1",
			OccurredAt:             time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			DisposedQuantity:       mustReportDecimal(t, "1"),
			AllocatedBasis:         mustReportDecimal(t, "0"),
			NetLiquidationProceeds: mustReportDecimal(t, "1"),
			GainOrLoss:             reportInvalidDecimal,
			ActivityCurrency:       "USD",
		}},
	)
	if err == nil || !strings.Contains(err.Error(), "liquidation summary 0: liquidation calculation gain or loss") {
		t.Fatalf("expected invalid liquidation gain or loss error, got %v", err)
	}

	var validRow = AssetActivityRow{
		SourceID:         "buy-1",
		OccurredAt:       time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		ActivityType:     syncmodel.ActivityTypeBuy,
		Quantity:         mustReportDecimal(t, "1"),
		BasisAfterRow:    mustReportDecimal(t, "1"),
		QuantityAfterRow: mustReportDecimal(t, "1"),
	}
	validRow.UnitPrice = &reportInvalidDecimal
	if err = validRow.Validate(); err == nil || !strings.Contains(err.Error(), "asset activity row unit price") {
		t.Fatalf("expected invalid unit-price optional decimal error, got %v", err)
	}
	validRow.UnitPrice = nil
	validRow.GrossValue = &reportInvalidDecimal
	if err = validRow.Validate(); err == nil || !strings.Contains(err.Error(), "asset activity row gross value") {
		t.Fatalf("expected invalid optional decimal error, got %v", err)
	}

	var invalidLiquidation = LiquidationCalculation{
		SourceID:               "sell-2",
		OccurredAt:             time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		DisposedQuantity:       mustReportDecimal(t, "1"),
		AllocatedBasis:         mustReportDecimal(t, "0"),
		NetLiquidationProceeds: mustReportDecimal(t, "1"),
		GainOrLoss:             mustReportDecimal(t, "1"),
	}
	if err = invalidLiquidation.Validate(); err == nil || !strings.Contains(err.Error(), "activity currency is required") {
		t.Fatalf("expected missing liquidation currency error, got %v", err)
	}
}

// TestReportDocumentAndOutputValidation verifies the rendered-document and
// output outcome helpers used by renderer and writer packages.
// Authored by: OpenCode
func TestReportDocumentAndOutputValidation(t *testing.T) {
	t.Parallel()

	_, err := NewReportDocument(ReportDocumentType("html"), "# Report\n", 2024, CostBasisMethodFIFO, time.Now())
	if err == nil || !strings.Contains(err.Error(), "unsupported report document type") {
		t.Fatalf("expected invalid document type error, got %v", err)
	}

	document, err := NewReportDocument(ReportDocumentTypeMarkdown, "# Report\n", 2024, CostBasisMethodFIFO, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report document: %v", err)
	}
	if err = document.Validate(); err != nil {
		t.Fatalf("validate report document: %v", err)
	}

	_, err = NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", time.Now(), false, "open failed")
	if err == nil || !strings.Contains(err.Error(), "open error requires an open request") {
		t.Fatalf("expected open error validation failure, got %v", err)
	}

	outputFile, err := NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC), true, "open failed")
	if err != nil {
		t.Fatalf("new report output file: %v", err)
	}
	if err = outputFile.Validate(); err != nil {
		t.Fatalf("validate report output file: %v", err)
	}
}

// TestReportDocumentValidationGuardrails verifies remaining document and report
// constructor validation branches.
// Authored by: OpenCode
func TestReportDocumentValidationGuardrails(t *testing.T) {
	t.Parallel()

	request, err := NewReportRequest(2024, CostBasisMethodFIFO, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	_, err = NewCapitalGainsReport(
		request,
		time.Date(2026, time.May, 21, 12, 0, 0, 0, time.UTC),
		"USD",
		nil,
		reportInvalidDecimal,
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "capital gains report yearly net total") {
		t.Fatalf("expected invalid yearly net total error, got %v", err)
	}

	_, err = NewReportDocument(ReportDocumentTypeMarkdown, "   ", 2024, CostBasisMethodFIFO, time.Now())
	if err == nil || !strings.Contains(err.Error(), "report document content is required") {
		t.Fatalf("expected missing content error, got %v", err)
	}

	_, err = NewReportDocument(ReportDocumentTypeMarkdown, "# Report\n", 2024, CostBasisMethod("bad"), time.Now())
	if err == nil || !strings.Contains(err.Error(), "report document cost basis method") {
		t.Fatalf("expected invalid report document method error, got %v", err)
	}
}

// TestReportConstructorsAndValidationHelpersCoverRemainingBranches verifies
// constructor success paths and direct validation helper branches not exercised
// by the broader report-model scenarios.
// Authored by: OpenCode
func TestReportConstructorsAndValidationHelpersCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	var summaryEntry, err = NewAssetSummaryEntry(" asset-btc ", " BTC ", mustReportDecimal(t, "2"), " USD ")
	if err != nil {
		t.Fatalf("new asset summary entry: %v", err)
	}
	if summaryEntry.AssetIdentityKey != "asset-btc" || summaryEntry.DisplayLabel != "BTC" || summaryEntry.ReportCalculationCurrency != "USD" {
		t.Fatalf("expected trimmed summary entry fields, got %#v", summaryEntry)
	}

	if err = (AssetSummaryEntry{AssetIdentityKey: "asset-btc", NetGainOrLoss: reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "asset summary entry net gain or loss") {
		t.Fatalf("expected invalid net gain or loss to fail, got %v", err)
	}

	var detailSection, detailErr = NewAssetDetailSection(
		" asset-btc ",
		" BTC ",
		mustReportDecimal(t, "1"),
		mustReportDecimal(t, "10"),
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		" USD ",
		nil,
		nil,
	)
	if detailErr != nil {
		t.Fatalf("new asset detail section: %v", detailErr)
	}
	if detailSection.AssetIdentityKey != "asset-btc" || detailSection.DisplayLabel != "BTC" || detailSection.CalculationCurrency != "USD" {
		t.Fatalf("expected trimmed detail section fields, got %#v", detailSection)
	}

	if err = (AssetDetailSection{AssetIdentityKey: "asset-btc", OpeningQuantity: mustReportDecimal(t, "-1")}).Validate(); err == nil || !strings.Contains(err.Error(), "opening quantity") {
		t.Fatalf("expected negative opening quantity to fail, got %v", err)
	}
	if err = (AssetDetailSection{AssetIdentityKey: "asset-btc", OpeningQuantity: mustReportDecimal(t, "0"), OpeningCostBasis: mustReportDecimal(t, "0"), ClosingQuantity: mustReportDecimal(t, "0"), ClosingCostBasis: mustReportDecimal(t, "-1")}).Validate(); err == nil || !strings.Contains(err.Error(), "closing cost basis") {
		t.Fatalf("expected negative closing basis to fail, got %v", err)
	}

	if err = (AssetActivityRow{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: syncmodel.ActivityTypeSell, Quantity: mustReportDecimal(t, "1"), BasisAfterRow: mustReportDecimal(t, "0"), QuantityAfterRow: mustReportDecimal(t, "0"), LiquidationCalculation: &LiquidationCalculation{}}).Validate(); err == nil || !strings.Contains(err.Error(), "asset activity row liquidation calculation") {
		t.Fatalf("expected nested liquidation validation to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: syncmodel.ActivityTypeSell, Quantity: mustReportDecimal(t, "1"), BasisAfterRow: mustReportDecimal(t, "-1"), QuantityAfterRow: mustReportDecimal(t, "0")}).Validate(); err == nil || !strings.Contains(err.Error(), "basis after row") {
		t.Fatalf("expected negative basis-after-row to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: syncmodel.ActivityTypeSell, Quantity: mustReportDecimal(t, "1"), BasisAfterRow: mustReportDecimal(t, "0"), QuantityAfterRow: mustReportDecimal(t, "-1")}).Validate(); err == nil || !strings.Contains(err.Error(), "quantity after row") {
		t.Fatalf("expected negative quantity-after-row to fail, got %v", err)
	}

	if err = (LiquidationCalculation{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), DisposedQuantity: mustReportDecimal(t, "1"), AllocatedBasis: mustReportDecimal(t, "-1"), NetLiquidationProceeds: mustReportDecimal(t, "1"), GainOrLoss: mustReportDecimal(t, "0"), ActivityCurrency: "USD"}).Validate(); err == nil || !strings.Contains(err.Error(), "allocated basis") {
		t.Fatalf("expected negative allocated basis to fail, got %v", err)
	}
	if err = (LiquidationCalculation{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), DisposedQuantity: mustReportDecimal(t, "1"), AllocatedBasis: mustReportDecimal(t, "0"), NetLiquidationProceeds: reportInvalidDecimal, GainOrLoss: mustReportDecimal(t, "0"), ActivityCurrency: "USD"}).Validate(); err == nil || !strings.Contains(err.Error(), "net liquidation proceeds") {
		t.Fatalf("expected invalid proceeds to fail, got %v", err)
	}

	var request, requestErr = NewReportRequest(2024, CostBasisMethodFIFO, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}
	var report, reportErr = NewCapitalGainsReport(request, time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC), "USD", nil, mustReportDecimal(t, "0"), nil, nil)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}
	if report.ReportCalculationCurrency != "USD" {
		t.Fatalf("expected report calculation currency to be preserved, got %#v", report)
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, YearlyNetTotal: mustReportDecimal(t, "0")}).Validate(); err == nil || !strings.Contains(err.Error(), "generated-at timestamp is required") {
		t.Fatalf("expected missing generated-at timestamp to fail, got %v", err)
	}

	if err = (ReferenceLiquidationEntry{}).Validate(); err == nil || !strings.Contains(err.Error(), "asset identity key is required") {
		t.Fatalf("expected blank reference asset identity key to fail, got %v", err)
	}

	if err = (AssetDetailSection{}).Validate(); err == nil || !strings.Contains(err.Error(), "asset identity key is required") {
		t.Fatalf("expected blank detail asset identity key to fail, got %v", err)
	}
	if err = (AssetDetailSection{AssetIdentityKey: "asset-btc", OpeningCostBasis: reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "opening cost basis") {
		t.Fatalf("expected invalid opening cost basis to fail, got %v", err)
	}
	if err = (AssetDetailSection{AssetIdentityKey: "asset-btc", ClosingQuantity: reportInvalidDecimal}).Validate(); err == nil || !strings.Contains(err.Error(), "closing quantity") {
		t.Fatalf("expected invalid closing quantity to fail, got %v", err)
	}

	if err = (AssetActivityRow{}).Validate(); err == nil || !strings.Contains(err.Error(), "source ID is required") {
		t.Fatalf("expected blank activity row source ID to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "row-1"}).Validate(); err == nil || !strings.Contains(err.Error(), "occurred-at timestamp is required") {
		t.Fatalf("expected blank activity row timestamp to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "row-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: syncmodel.ActivityTypeBuy}).Validate(); err == nil || !strings.Contains(err.Error(), "quantity") {
		t.Fatalf("expected missing positive activity quantity to fail, got %v", err)
	}
	if err = (AssetActivityRow{SourceID: "row-1", OccurredAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), ActivityType: syncmodel.ActivityTypeBuy, Quantity: mustReportDecimal(t, "1"), FeeAmount: &reportInvalidDecimal, BasisAfterRow: mustReportDecimal(t, "0"), QuantityAfterRow: mustReportDecimal(t, "0")}).Validate(); err == nil || !strings.Contains(err.Error(), "fee amount") {
		t.Fatalf("expected invalid activity fee amount to fail, got %v", err)
	}

	if err = (LiquidationCalculation{}).Validate(); err == nil || !strings.Contains(err.Error(), "source ID is required") {
		t.Fatalf("expected blank liquidation source ID to fail, got %v", err)
	}
	if err = (LiquidationCalculation{SourceID: "sell-1"}).Validate(); err == nil || !strings.Contains(err.Error(), "occurred-at timestamp is required") {
		t.Fatalf("expected blank liquidation timestamp to fail, got %v", err)
	}
	if err = (LiquidationCalculation{SourceID: "sell-1", OccurredAt: time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC)}).Validate(); err == nil || !strings.Contains(err.Error(), "disposed quantity") {
		t.Fatalf("expected missing positive disposed quantity to fail, got %v", err)
	}

	if _, err = NewCapitalGainsReport(ReportRequest{}, time.Now(), "USD", nil, mustReportDecimal(t, "0"), nil, nil); err == nil || !strings.Contains(err.Error(), "capital gains report request") {
		t.Fatalf("expected invalid report request to fail top-level constructor, got %v", err)
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethod("bad"), GeneratedAt: time.Now(), YearlyNetTotal: mustReportDecimal(t, "0")}).Validate(); err == nil || !strings.Contains(err.Error(), "cost basis method") {
		t.Fatalf("expected invalid report cost basis method to fail, got %v", err)
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now(), YearlyNetTotal: mustReportDecimal(t, "0"), ReferenceEntries: []ReferenceLiquidationEntry{{}}}).Validate(); err == nil || !strings.Contains(err.Error(), "reference entry 0") {
		t.Fatalf("expected invalid nested reference entry to fail, got %v", err)
	}
	if err = (CapitalGainsReport{Year: 2024, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now(), YearlyNetTotal: mustReportDecimal(t, "0"), DetailSections: []AssetDetailSection{{}}}).Validate(); err == nil || !strings.Contains(err.Error(), "detail section 0") {
		t.Fatalf("expected invalid nested detail section to fail, got %v", err)
	}

	if err = (ReportDocument{DocumentType: ReportDocumentTypeMarkdown, Content: "# Report\n", Year: 0, CostBasisMethod: CostBasisMethodFIFO, GeneratedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "report document year must be greater than zero") {
		t.Fatalf("expected missing document year to fail, got %v", err)
	}
	if err = (ReportDocument{DocumentType: ReportDocumentTypeMarkdown, Content: "# Report\n", Year: 2024, CostBasisMethod: CostBasisMethodFIFO}).Validate(); err == nil || !strings.Contains(err.Error(), "generated-at timestamp is required") {
		t.Fatalf("expected missing document timestamp to fail, got %v", err)
	}

	if err = (ReportOutputFile{DocumentsDirectory: "/tmp/docs", Filename: "report.md", Path: "/tmp/docs/report.md", OpenRequested: true}).Validate(); err == nil || !strings.Contains(err.Error(), "saved-at timestamp is required") {
		t.Fatalf("expected missing output timestamp to fail, got %v", err)
	}
	if err = (ReportOutputFile{Filename: "report.md", Path: "/tmp/docs/report.md", SavedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "documents directory") {
		t.Fatalf("expected missing output documents directory to fail, got %v", err)
	}
	if err = (ReportOutputFile{DocumentsDirectory: "/tmp/docs", Filename: "report.md", SavedAt: time.Now()}).Validate(); err == nil || !strings.Contains(err.Error(), "output path") {
		t.Fatalf("expected missing output path to fail, got %v", err)
	}

	if err = validateOptionalDecimal(nil, "optional"); err != nil {
		t.Fatalf("expected nil optional decimal to validate, got %v", err)
	}
	if err = validatePositiveDecimal(reportInvalidDecimal, "positive"); err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("expected non-finite positive decimal to fail, got %v", err)
	}
	if err = validatePositiveDecimal(mustReportDecimal(t, "0"), "positive"); err == nil || !strings.Contains(err.Error(), "must be greater than zero") {
		t.Fatalf("expected zero positive decimal to fail, got %v", err)
	}
	if err = validateNonNegativeDecimal(reportInvalidDecimal, "non-negative"); err == nil || !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("expected non-finite non-negative decimal to fail, got %v", err)
	}
	if err = validateNonNegativeDecimal(mustReportDecimal(t, "-1"), "non-negative"); err == nil || !strings.Contains(err.Error(), "must not be negative") {
		t.Fatalf("expected negative non-negative decimal to fail, got %v", err)
	}
}

// mustReportDecimal parses one decimal literal for report-model tests.
// Authored by: OpenCode
func mustReportDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", raw, err)
	}

	return value
}

// decimalPointer returns one decimal pointer for report-model tests.
// Authored by: OpenCode
func decimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = mustReportDecimal(t, raw)
	return &value
}
