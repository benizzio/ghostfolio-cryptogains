// Package calculate verifies package-local report calculation helpers and seams.
// Authored by: OpenCode
package calculate

import (
	"errors"
	"strings"
	"testing"
	"time"

	reportbasis "github.com/benizzio/ghostfolio-cryptogains/internal/report/basis"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// stubAssetBasisState provides one controllable package-local basis state for
// calculator helper tests.
// Authored by: OpenCode
type stubAssetBasisState struct {
	addAcquisitionFunc func(basisAcquisitionInput) error
	disposeFunc        func(basisDisposalInput) (basisDisposalResult, error)
	openQuantityFunc   func() (apd.Decimal, error)
	openBasisFunc      func() (apd.Decimal, error)
}

// AddAcquisition forwards one acquisition into the configured stub callback.
// Authored by: OpenCode
func (state stubAssetBasisState) AddAcquisition(input basisAcquisitionInput) error {
	if state.addAcquisitionFunc == nil {
		return nil
	}

	return state.addAcquisitionFunc(input)
}

// Dispose forwards one disposal into the configured stub callback.
// Authored by: OpenCode
func (state stubAssetBasisState) Dispose(input basisDisposalInput) (basisDisposalResult, error) {
	if state.disposeFunc == nil {
		return basisDisposalResult{}, nil
	}

	return state.disposeFunc(input)
}

// OpenQuantity forwards one quantity lookup into the configured stub callback.
// Authored by: OpenCode
func (state stubAssetBasisState) OpenQuantity() (apd.Decimal, error) {
	if state.openQuantityFunc == nil {
		return apd.Decimal{}, nil
	}

	return state.openQuantityFunc()
}

// OpenBasis forwards one basis lookup into the configured stub callback.
// Authored by: OpenCode
func (state stubAssetBasisState) OpenBasis() (apd.Decimal, error) {
	if state.openBasisFunc == nil {
		return apd.Decimal{}, nil
	}

	return state.openBasisFunc()
}

// TestCalculateRejectsInvalidRequestAndUnavailableYear verifies top-level
// request and availability guardrails.
// Authored by: OpenCode
func TestCalculateRejectsInvalidRequestAndUnavailableYear(t *testing.T) {
	t.Parallel()

	_, err := Calculate(reportmodel.ReportRequest{}, syncmodel.ProtectedActivityCache{})
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindInvalidRequest)
	if !strings.Contains(calcErr.Error(), "report request year must be greater than zero") {
		t.Fatalf("expected invalid request detail, got %q", calcErr.Error())
	}

	var request = validReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	_, err = Calculate(request, syncmodel.ProtectedActivityCache{AvailableReportYears: []int{2023}})
	calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindUnavailableReportYear)
	if !strings.Contains(calcErr.Error(), "report year 2024 is not available") {
		t.Fatalf("expected unavailable year detail, got %q", calcErr.Error())
	}

	if err := requireImplementedCostBasisMethod(reportmodel.CostBasisMethodFIFO); err != nil {
		t.Fatalf("expected supported cost basis method to be accepted, got %v", err)
	}
	calcErr = requireCalculationError(t, requireImplementedCostBasisMethod(reportmodel.CostBasisMethod("unsupported")), reportmodel.CalculationErrorKindUnsupportedCostBasisMethod)
	if !strings.Contains(calcErr.Error(), "unsupported cost basis method") {
		t.Fatalf("expected unsupported cost basis method detail, got %q", calcErr.Error())
	}
}

// TestCalculatePropagatesActivitySelectionFailures verifies that grouped input
// selection preserves offending activity references.
// Authored by: OpenCode
func TestCalculatePropagatesActivitySelectionFailures(t *testing.T) {
	t.Parallel()

	var request = validReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO)
	var cache = syncmodel.ProtectedActivityCache{
		AvailableReportYears: []int{2024},
		Activities: []syncmodel.ActivityRecord{{
			SourceID:         "buy-1",
			OccurredAt:       " ",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-btc",
			AssetSymbol:      "BTC",
			Quantity:         mustReportDecimal(t, "1"),
		}},
	}

	_, err := Calculate(request, cache)
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)
	if calcErr.SourceID() != "buy-1" || calcErr.DisplayLabel() != "BTC" {
		t.Fatalf("expected offending activity references, got %#v", calcErr)
	}

	cache.Activities[0].OccurredAt = "2024-01-02T00:00:00Z"
	cache.Activities[0].AssetIdentityKey = " "
	_, err = Calculate(request, cache)
	calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)
	if !strings.Contains(calcErr.Error(), "missing the stored asset identity key") {
		t.Fatalf("expected missing asset identity key detail, got %q", calcErr.Error())
	}
}

// TestCalculateBuildsReportFromIncludedResults verifies loop behavior for
// included and reference-only asset results.
// Authored by: OpenCode
func TestCalculateBuildsReportFromIncludedResults(t *testing.T) {
	var originalCalculateAssetGroupFunc = calculateAssetGroupFunc
	defer func() {
		calculateAssetGroupFunc = originalCalculateAssetGroupFunc
	}()

	calculateAssetGroupFunc = func(_ reportmodel.CostBasisMethod, _ int, _ syncmodel.ProtectedActivityCache, group assetInputGroup) (assetCalculationResult, error) {
		switch group.AssetIdentityKey {
		case "asset-btc":
			return assetCalculationResult{
				IncludeInMain: true,
				SummaryEntry:  validAssetSummaryEntry(t, "asset-btc", "BTC", "2"),
				DetailSection: validAssetDetailSection(t, "asset-btc", "BTC"),
				YearlyNet:     mustReportDecimal(t, "2"),
			}, nil
		case "asset-eth":
			var referenceEntry, err = reportmodel.NewReferenceLiquidationEntry(
				"asset-eth",
				"ETH",
				1,
				reportmodel.ReferenceSectionStatusReferenceOnly,
			)
			if err != nil {
				t.Fatalf("new reference entry: %v", err)
			}
			return assetCalculationResult{
				ReferenceEntry: &referenceEntry,
				YearlyNet:      mustReportDecimal(t, "999"),
			}, nil
		default:
			return assetCalculationResult{}, errors.New("unexpected asset group")
		}
	}

	var report, err = Calculate(
		validReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO),
		syncmodel.ProtectedActivityCache{
			AvailableReportYears: []int{2024},
			Activities: []syncmodel.ActivityRecord{
				zeroPricedHoldingReductionRecord(t, "btc-sell-1", "2024-01-02T10:00:00Z", "asset-btc", "BTC", "Bitcoin"),
				zeroPricedHoldingReductionRecord(t, "eth-sell-1", "2024-01-03T10:00:00Z", "asset-eth", "ETH", "Ether"),
			},
		},
	)
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}
	if report.YearlyNetTotal.Cmp(apd.New(2, 0)) != 0 {
		t.Fatalf("expected only included asset to contribute yearly net total, got %v", report.YearlyNetTotal)
	}
	if len(report.SummaryEntries) != 1 || len(report.DetailSections) != 1 || len(report.ReferenceEntries) != 1 {
		t.Fatalf("unexpected report section counts: %#v", report)
	}
	if report.SummaryEntries[0].AssetIdentityKey != "asset-btc" || report.ReferenceEntries[0].AssetIdentityKey != "asset-eth" {
		t.Fatalf("unexpected report section entries: %#v", report)
	}
}

// TestCalculateWrapsCalculatedReportValidationFailure verifies final report
// constructor failures are classified consistently.
// Authored by: OpenCode
func TestCalculateWrapsCalculatedReportValidationFailure(t *testing.T) {
	var originalCalculateAssetGroupFunc = calculateAssetGroupFunc
	var originalNewCapitalGainsReport = newCapitalGainsReport
	defer func() {
		calculateAssetGroupFunc = originalCalculateAssetGroupFunc
		newCapitalGainsReport = originalNewCapitalGainsReport
	}()

	calculateAssetGroupFunc = func(_ reportmodel.CostBasisMethod, _ int, _ syncmodel.ProtectedActivityCache, _ assetInputGroup) (assetCalculationResult, error) {
		return assetCalculationResult{
			IncludeInMain: true,
			SummaryEntry:  validAssetSummaryEntry(t, "asset-btc", "BTC", "1"),
			DetailSection: validAssetDetailSection(t, "asset-btc", "BTC"),
			YearlyNet:     mustReportDecimal(t, "1"),
		}, nil
	}
	newCapitalGainsReport = func(
		reportmodel.ReportRequest,
		time.Time,
		string,
		[]reportmodel.AssetSummaryEntry,
		apd.Decimal,
		[]reportmodel.ReferenceLiquidationEntry,
		[]reportmodel.AssetDetailSection,
	) (reportmodel.CapitalGainsReport, error) {
		return reportmodel.CapitalGainsReport{}, errors.New("report invalid")
	}

	_, err := Calculate(
		validReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO),
		syncmodel.ProtectedActivityCache{
			AvailableReportYears: []int{2024},
			Activities: []syncmodel.ActivityRecord{
				zeroPricedHoldingReductionRecord(t, "btc-sell-1", "2024-01-02T10:00:00Z", "asset-btc", "BTC", "Bitcoin"),
			},
		},
	)
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "calculated report validation failed") {
		t.Fatalf("expected wrapped final report failure, got %q", calcErr.Error())
	}
}

// TestSelectAssetInputGroupsThroughYearFiltersAndPreservesOrder verifies replay
// ordering, future-year filtering, and display-label fallback updates.
// Authored by: OpenCode
func TestSelectAssetInputGroupsThroughYearFiltersAndPreservesOrder(t *testing.T) {
	t.Parallel()

	var groups, err = selectAssetInputGroupsThroughYear([]syncmodel.ActivityRecord{
		zeroPricedHoldingReductionRecord(t, "btc-sell-1", "2024-01-02T10:00:00Z", "asset-btc", "BTC", "Bitcoin"),
		zeroPricedHoldingReductionRecord(t, "eth-sell-1", "2024-02-02T10:00:00Z", "asset-eth", "", ""),
		zeroPricedHoldingReductionRecord(t, "btc-sell-2", "2023-12-31T23:00:00Z", "asset-btc", "BTC", "Bitcoin"),
		zeroPricedHoldingReductionRecord(t, "eth-sell-2", "2024-03-02T10:00:00Z", "asset-eth", "ETH", "Ether"),
		zeroPricedHoldingReductionRecord(t, "btc-sell-3", "2025-01-01T00:00:00Z", "asset-btc", "BTC", "Bitcoin"),
	}, 2024)
	if err != nil {
		t.Fatalf("select asset input groups: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected two grouped assets, got %d", len(groups))
	}
	if groups[0].AssetIdentityKey != "asset-btc" || len(groups[0].Inputs) != 2 {
		t.Fatalf("expected BTC group first with two inputs, got %#v", groups[0])
	}
	if groups[1].AssetIdentityKey != "asset-eth" || groups[1].DisplayLabel != "ETH" || len(groups[1].Inputs) != 2 {
		t.Fatalf("expected ETH group label fallback to update from later input, got %#v", groups[1])
	}
}

// TestCaptureOpeningPositionIfNeededSnapshotsOnce verifies the opening snapshot
// boundary and no-op branches.
// Authored by: OpenCode
func TestCaptureOpeningPositionIfNeededSnapshotsOnce(t *testing.T) {
	t.Parallel()

	if err := captureOpeningPositionIfNeeded(nil, stubAssetBasisState{}, 2024, 2024); err != nil {
		t.Fatalf("expected nil replay state to be ignored, got %v", err)
	}

	var state assetReplayState
	var quantity = mustReportDecimal(t, "2")
	var basis = mustReportDecimal(t, "10")
	var basisState = stubAssetBasisState{
		openQuantityFunc: func() (apd.Decimal, error) { return quantity, nil },
		openBasisFunc:    func() (apd.Decimal, error) { return basis, nil },
	}

	if err := captureOpeningPositionIfNeeded(&state, basisState, 2023, 2024); err != nil {
		t.Fatalf("capture opening position for prior year: %v", err)
	}
	if state.openingCaptured {
		t.Fatalf("expected prior-year input not to capture opening position")
	}

	if err := captureOpeningPositionIfNeeded(&state, basisState, 2024, 2024); err != nil {
		t.Fatalf("capture opening position: %v", err)
	}
	if !state.openingCaptured || state.openingQuantity.Cmp(apd.New(2, 0)) != 0 || state.openingBasis.Cmp(apd.New(10, 0)) != 0 {
		t.Fatalf("expected captured opening position, got %#v", state)
	}

	quantity = mustReportDecimal(t, "99")
	basis = mustReportDecimal(t, "99")
	if err := captureOpeningPositionIfNeeded(&state, basisState, 2024, 2024); err != nil {
		t.Fatalf("expected second capture to stay no-op, got %v", err)
	}
	if state.openingQuantity.Cmp(apd.New(2, 0)) != 0 || state.openingBasis.Cmp(apd.New(10, 0)) != 0 {
		t.Fatalf("expected opening snapshot to stay unchanged, got %#v", state)
	}

	var normalized = sourceCalendarDate(time.Date(2024, time.May, 21, 22, 3, 4, 0, time.FixedZone("offset", 2*60*60)))
	if !normalized.Equal(time.Date(2024, time.May, 21, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected source calendar date: %v", normalized)
	}
}

// TestReplayAssetInputCoversYearBoundariesAndWrappedStateFailures verifies the
// per-activity replay wrapper around basis-state lookups.
// Authored by: OpenCode
func TestReplayAssetInputCoversYearBoundariesAndWrappedStateFailures(t *testing.T) {
	t.Parallel()

	var input = reportmodel.ActivityCalculationInput{
		SourceID:         "buy-1",
		OccurredAt:       time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC),
		SourceYear:       2023,
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       decimalPointer(t, "10"),
		FeeAmount:        decimalPointer(t, "0"),
	}

	_, err := replayAssetInput(stubAssetBasisState{
		openQuantityFunc: func() (apd.Decimal, error) { return apd.Decimal{}, errors.New("before open quantity") },
	}, scopedActivityInput{Input: input}, 1, 2024)
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not determine the asset quantity before applying the activity") {
		t.Fatalf("expected wrapped pre-open failure, got %q", calcErr.Error())
	}

	var priorYearCalls int
	var replayResult assetInputReplayResult
	replayResult, err = replayAssetInput(stubAssetBasisState{
		addAcquisitionFunc: func(basisAcquisitionInput) error { return nil },
		openQuantityFunc: func() (apd.Decimal, error) {
			priorYearCalls++
			return mustReportDecimal(t, "1"), nil
		},
		openBasisFunc: func() (apd.Decimal, error) { return mustReportDecimal(t, "10"), nil },
	}, scopedActivityInput{Input: input}, 1, 2024)
	if err != nil {
		t.Fatalf("replay prior-year activity: %v", err)
	}
	if replayResult.activityRow != nil || replayResult.liquidationSummary != nil || replayResult.yearlyNetDelta.Sign() != 0 || priorYearCalls != 2 {
		t.Fatalf("expected prior-year replay to skip in-year artifacts, got %#v calls=%d", replayResult, priorYearCalls)
	}

	var postApplyCalls int
	_, err = replayAssetInput(stubAssetBasisState{
		addAcquisitionFunc: func(basisAcquisitionInput) error { return nil },
		openQuantityFunc: func() (apd.Decimal, error) {
			postApplyCalls++
			return mustReportDecimal(t, "1"), nil
		},
		openBasisFunc: func() (apd.Decimal, error) { return apd.Decimal{}, errors.New("after basis") },
	}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "buy-2",
		OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       decimalPointer(t, "10"),
		FeeAmount:        decimalPointer(t, "0"),
	}}, 1, 2024)
	calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not determine the asset basis after applying the activity") || postApplyCalls != 2 {
		t.Fatalf("expected wrapped post-apply basis failure, got %q calls=%d", calcErr.Error(), postApplyCalls)
	}
}

// TestApplyBasisInputRoutesActivities verifies basis-routing behavior for
// acquisitions, liquidations, and unsupported activity types.
// Authored by: OpenCode
func TestApplyBasisInputRoutesActivities(t *testing.T) {
	t.Parallel()

	var capturedAcquisition basisAcquisitionInput
	_, err := applyBasisInput(stubAssetBasisState{
		addAcquisitionFunc: func(input basisAcquisitionInput) error {
			capturedAcquisition = input
			return nil
		},
	}, scopedActivityInput{
		Input: reportmodel.ActivityCalculationInput{
			SourceID:         "buy-1",
			OccurredAt:       time.Date(2024, time.January, 2, 10, 11, 12, 0, time.UTC),
			SourceYear:       2024,
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
			Quantity:         mustReportDecimal(t, "2"),
			GrossValue:       decimalPointer(t, "20"),
			FeeAmount:        decimalPointer(t, "1"),
		},
		ApplicableScope: applicableScope{ScopeKey: "scope-a"},
	}, 7)
	if err != nil {
		t.Fatalf("apply acquisition: %v", err)
	}
	if capturedAcquisition.SourceID != "buy-1" || capturedAcquisition.DeterministicOrder != 7 || capturedAcquisition.ApplicableScopeKey != "scope-a" || capturedAcquisition.Basis.Cmp(apd.New(21, 0)) != 0 {
		t.Fatalf("unexpected captured acquisition: %#v", capturedAcquisition)
	}
	if !capturedAcquisition.AcquiredAt.Equal(time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected acquisition date normalization, got %v", capturedAcquisition.AcquiredAt)
	}

	_, err = applyBasisInput(stubAssetBasisState{}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "unsupported-1",
		SourceYear:       2024,
		OccurredAt:       time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		ActivityType:     syncmodel.ActivityType("SWAP"),
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
	}}, 1)
	requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)

	var pricedSellResult basisApplicationResult
	pricedSellResult, err = applyBasisInput(stubAssetBasisState{
		disposeFunc: func(input basisDisposalInput) (basisDisposalResult, error) {
			if input.ApplicableScopeKey != "scope-b" || input.Quantity.Cmp(apd.New(2, 0)) != 0 {
				t.Fatalf("unexpected disposal input: %#v", input)
			}
			return basisDisposalResult{AllocatedBasis: mustReportDecimal(t, "7"), ReachedZero: true}, nil
		},
	}, scopedActivityInput{
		Input: reportmodel.ActivityCalculationInput{
			SourceID:                "sell-1",
			OccurredAt:              time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
			SourceYear:              2024,
			ActivityType:            syncmodel.ActivityTypeSell,
			AssetIdentityKey:        "asset-btc",
			DisplayLabel:            "BTC",
			Quantity:                mustReportDecimal(t, "2"),
			GrossValue:              decimalPointer(t, "12"),
			FeeAmount:               decimalPointer(t, "2"),
			SelectedCurrencyCode:    "USD",
			SelectedCurrencyContext: reportmodel.SelectedCurrencyContextBase,
		},
		ApplicableScope: applicableScope{ScopeKey: "scope-b"},
	}, 1)
	if err != nil {
		t.Fatalf("apply priced liquidation: %v", err)
	}
	if pricedSellResult.allocatedBasis == nil || pricedSellResult.netProceeds == nil || pricedSellResult.gainOrLoss == nil {
		t.Fatalf("expected liquidation calculations, got %#v", pricedSellResult)
	}
	if pricedSellResult.allocatedBasis.Cmp(apd.New(7, 0)) != 0 || pricedSellResult.netProceeds.Cmp(apd.New(10, 0)) != 0 || pricedSellResult.gainOrLoss.Cmp(apd.New(3, 0)) != 0 || !pricedSellResult.reachedZero {
		t.Fatalf("unexpected priced liquidation result: %#v", pricedSellResult)
	}

	var zeroPricedResult basisApplicationResult
	zeroPricedResult, err = applyBasisInput(stubAssetBasisState{
		disposeFunc: func(input basisDisposalInput) (basisDisposalResult, error) {
			return basisDisposalResult{AllocatedBasis: mustReportDecimal(t, "5"), ReachedZero: false}, nil
		},
	}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:                     "sell-2",
		OccurredAt:                   time.Date(2024, time.January, 4, 0, 0, 0, 0, time.UTC),
		SourceYear:                   2024,
		ActivityType:                 syncmodel.ActivityTypeSell,
		AssetIdentityKey:             "asset-btc",
		DisplayLabel:                 "BTC",
		Quantity:                     mustReportDecimal(t, "1"),
		IsZeroPricedHoldingReduction: true,
		Comment:                      "manual transfer",
	}}, 1)
	if err != nil {
		t.Fatalf("apply zero-priced holding reduction: %v", err)
	}
	if zeroPricedResult.allocatedBasis == nil || zeroPricedResult.allocatedBasis.Cmp(apd.New(5, 0)) != 0 || zeroPricedResult.netProceeds != nil || zeroPricedResult.gainOrLoss != nil || zeroPricedResult.reachedZero {
		t.Fatalf("unexpected zero-priced disposal result: %#v", zeroPricedResult)
	}

	_, err = applyAcquisition(stubAssetBasisState{}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "buy-missing-values",
		OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
	}}, 1)
	requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	_, err = applyAcquisition(stubAssetBasisState{}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "buy-invalid-basis",
		OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       &invalid,
		FeeAmount:        decimalPointer(t, "1"),
	}}, 1)
	requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)

	_, err = applyAcquisition(stubAssetBasisState{addAcquisitionFunc: func(basisAcquisitionInput) error {
		return errors.New("add boom")
	}}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "buy-add-fail",
		OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       decimalPointer(t, "10"),
		FeeAmount:        decimalPointer(t, "1"),
	}}, 1)
	requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)

	_, err = applyZeroPricedHoldingReduction(stubAssetBasisState{disposeFunc: func(basisDisposalInput) (basisDisposalResult, error) {
		return basisDisposalResult{}, errors.New("dispose boom")
	}}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:                     "sell-zero-fail",
		OccurredAt:                   time.Date(2024, time.January, 4, 0, 0, 0, 0, time.UTC),
		SourceYear:                   2024,
		ActivityType:                 syncmodel.ActivityTypeSell,
		AssetIdentityKey:             "asset-btc",
		DisplayLabel:                 "BTC",
		Quantity:                     mustReportDecimal(t, "1"),
		IsZeroPricedHoldingReduction: true,
	}})
	requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)

	_, err = applyPricedLiquidation(stubAssetBasisState{}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "sell-missing-values",
		OccurredAt:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     syncmodel.ActivityTypeSell,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
	}})
	requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)

	_, err = applyPricedLiquidation(stubAssetBasisState{}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "sell-invalid-proceeds",
		OccurredAt:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     syncmodel.ActivityTypeSell,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       &invalid,
		FeeAmount:        decimalPointer(t, "1"),
	}})
	requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)

	_, err = applyPricedLiquidation(stubAssetBasisState{disposeFunc: func(basisDisposalInput) (basisDisposalResult, error) {
		var allocated apd.Decimal
		allocated.Form = apd.Infinite
		return basisDisposalResult{AllocatedBasis: allocated}, nil
	}}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "sell-invalid-gain",
		OccurredAt:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     syncmodel.ActivityTypeSell,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       decimalPointer(t, "12"),
		FeeAmount:        decimalPointer(t, "1"),
	}})
	requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
}

// TestBuildInYearArtifactsCoversZeroPricedPricedAndValidationFailures verifies
// row and liquidation rendering branches.
// Authored by: OpenCode
func TestBuildInYearArtifactsCoversZeroPricedPricedAndValidationFailures(t *testing.T) {
	t.Parallel()

	var row, liquidation, yearlyNet, err = buildInYearArtifacts(
		reportmodel.ActivityCalculationInput{
			SourceID:                     "sell-1",
			OccurredAt:                   time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC),
			SourceYear:                   2024,
			ActivityType:                 syncmodel.ActivityTypeSell,
			Quantity:                     mustReportDecimal(t, "1"),
			Comment:                      "manual transfer",
			IsZeroPricedHoldingReduction: true,
		},
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		basisApplicationResult{allocatedBasis: decimalPointer(t, "5")},
	)
	if err != nil {
		t.Fatalf("build zero-priced in-year artifacts: %v", err)
	}
	if row == nil || liquidation != nil || yearlyNet.Sign() != 0 || row.ActivityCurrency != "" || row.HoldingReductionExplanation != "manual transfer" {
		t.Fatalf("unexpected zero-priced row artifacts: row=%#v liquidation=%#v yearlyNet=%v", row, liquidation, yearlyNet)
	}

	row, liquidation, yearlyNet, err = buildInYearArtifacts(
		reportmodel.ActivityCalculationInput{
			SourceID:             "sell-2",
			OccurredAt:           time.Date(2024, time.January, 6, 0, 0, 0, 0, time.UTC),
			SourceYear:           2024,
			ActivityType:         syncmodel.ActivityTypeSell,
			Quantity:             mustReportDecimal(t, "1"),
			GrossValue:           decimalPointer(t, "12"),
			FeeAmount:            decimalPointer(t, "2"),
			SelectedCurrencyCode: "USD",
		},
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		basisApplicationResult{
			allocatedBasis: decimalPointer(t, "7"),
			netProceeds:    decimalPointer(t, "10"),
			gainOrLoss:     decimalPointer(t, "3"),
		},
	)
	if err != nil {
		t.Fatalf("build priced in-year artifacts: %v", err)
	}
	if row == nil || liquidation == nil || yearlyNet.Cmp(apd.New(3, 0)) != 0 || row.LiquidationCalculation == nil || row.ActivityCurrency != "USD" {
		t.Fatalf("unexpected priced row artifacts: row=%#v liquidation=%#v yearlyNet=%v", row, liquidation, yearlyNet)
	}

	_, _, _, err = buildInYearArtifacts(
		reportmodel.ActivityCalculationInput{
			SourceID:             "sell-3",
			OccurredAt:           time.Date(2024, time.January, 7, 0, 0, 0, 0, time.UTC),
			SourceYear:           2024,
			ActivityType:         syncmodel.ActivityTypeSell,
			Quantity:             mustReportDecimal(t, "1"),
			GrossValue:           decimalPointer(t, "12"),
			FeeAmount:            decimalPointer(t, "2"),
			SelectedCurrencyCode: " ",
		},
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "0"),
		basisApplicationResult{
			allocatedBasis: decimalPointer(t, "7"),
			netProceeds:    decimalPointer(t, "10"),
			gainOrLoss:     decimalPointer(t, "3"),
		},
	)
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not build the liquidation summary") {
		t.Fatalf("expected liquidation validation failure, got %q", calcErr.Error())
	}

	_, _, _, err = buildInYearArtifacts(
		reportmodel.ActivityCalculationInput{
			SourceID:             "row-invalid",
			OccurredAt:           time.Date(2024, time.January, 7, 0, 0, 0, 0, time.UTC),
			SourceYear:           2024,
			ActivityType:         syncmodel.ActivityTypeSell,
			Quantity:             mustReportDecimal(t, "1"),
			GrossValue:           decimalPointer(t, "12"),
			FeeAmount:            decimalPointer(t, "2"),
			SelectedCurrencyCode: "USD",
		},
		reportInvalidDecimalForCalculator(),
		mustReportDecimal(t, "0"),
		basisApplicationResult{},
	)
	calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not build the in-year activity row") {
		t.Fatalf("expected row validation failure, got %q", calcErr.Error())
	}
}

// TestBuildAssetCalculationResultCoversReferenceOnlyIncludedAndValidationFailure
// verifies result assembly around inclusion and wrapped validation failures.
// Authored by: OpenCode
func TestBuildAssetCalculationResultCoversReferenceOnlyIncludedAndValidationFailure(t *testing.T) {
	t.Parallel()

	var result, err = buildAssetCalculationResult(assetInputGroup{AssetIdentityKey: "asset-btc", DisplayLabel: "BTC"}, assetReplayState{
		fullLiquidationCount: 1,
	})
	if err != nil {
		t.Fatalf("build reference-only asset result: %v", err)
	}
	if result.IncludeInMain || result.ReferenceEntry == nil || result.ReferenceEntry.MainSectionStatus != reportmodel.ReferenceSectionStatusReferenceOnly {
		t.Fatalf("expected reference-only asset result, got %#v", result)
	}

	result, err = buildAssetCalculationResult(assetInputGroup{AssetIdentityKey: "asset-eth", DisplayLabel: "ETH"}, assetReplayState{
		hadInYearFullLiquidation: true,
		fullLiquidationCount:     2,
		yearlyNet:                mustReportDecimal(t, "4"),
	})
	if err != nil {
		t.Fatalf("build included asset result: %v", err)
	}
	if !result.IncludeInMain || result.ReferenceEntry == nil || result.ReferenceEntry.MainSectionStatus != reportmodel.ReferenceSectionStatusIncludedInMainSections || result.SummaryEntry.AssetIdentityKey != "asset-eth" || result.DetailSection.AssetIdentityKey != "asset-eth" {
		t.Fatalf("expected included asset result with reference entry, got %#v", result)
	}

	_, err = buildAssetCalculationResult(assetInputGroup{AssetIdentityKey: " ", DisplayLabel: "BTC"}, assetReplayState{
		closingQuantity:          mustReportDecimal(t, "1"),
		hadInYearFullLiquidation: true,
	})
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not build the summary entry") {
		t.Fatalf("expected wrapped summary-entry failure, got %q", calcErr.Error())
	}

	_, err = buildAssetCalculationResult(assetInputGroup{AssetIdentityKey: "asset-btc", DisplayLabel: "BTC"}, assetReplayState{
		fullLiquidationCount:     1,
		hadInYearFullLiquidation: true,
		closingQuantity:          mustReportDecimal(t, "1"),
		closingBasis:             reportInvalidDecimalForCalculator(),
	})
	calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not build the detail section") {
		t.Fatalf("expected wrapped detail-section failure, got %q", calcErr.Error())
	}
}

// TestNewAssetBasisStateAndCalculationHelpers verifies method selection,
// seam-propagated constructor failures, decimal helpers, and structured error
// wrappers.
// Authored by: OpenCode
func TestNewAssetBasisStateAndCalculationHelpers(t *testing.T) {
	var originalNewLotMethodState = newLotMethodState
	defer func() {
		newLotMethodState = originalNewLotMethodState
	}()

	var fifoState, err = newAssetBasisState(reportmodel.CostBasisMethodFIFO)
	if err != nil {
		t.Fatalf("new FIFO basis state: %v", err)
	}
	if _, ok := fifoState.(lotBasisState); !ok {
		t.Fatalf("expected FIFO to use lot basis state, got %T", fifoState)
	}

	var lifoState assetBasisState
	lifoState, err = newAssetBasisState(reportmodel.CostBasisMethodLIFO)
	if err != nil {
		t.Fatalf("new LIFO basis state: %v", err)
	}
	if _, ok := lifoState.(lotBasisState); !ok {
		t.Fatalf("expected LIFO to use lot basis state, got %T", lifoState)
	}

	var hifoState assetBasisState
	hifoState, err = newAssetBasisState(reportmodel.CostBasisMethodHIFO)
	if err != nil {
		t.Fatalf("new HIFO basis state: %v", err)
	}
	if _, ok := hifoState.(lotBasisState); !ok {
		t.Fatalf("expected HIFO to use lot basis state, got %T", hifoState)
	}

	var averageState assetBasisState
	averageState, err = newAssetBasisState(reportmodel.CostBasisMethodAverageCost)
	if err != nil {
		t.Fatalf("new average-cost basis state: %v", err)
	}
	if _, ok := averageState.(averageCostBasisState); !ok {
		t.Fatalf("expected average cost to use averageCostBasisState, got %T", averageState)
	}

	var hybridState assetBasisState
	hybridState, err = newAssetBasisState(reportmodel.CostBasisMethodScopeLocalHybrid)
	if err != nil {
		t.Fatalf("new scope-local hybrid basis state: %v", err)
	}
	if _, ok := hybridState.(scopeLocalHybridBasisState); !ok {
		t.Fatalf("expected scope-local hybrid to use scopeLocalHybridBasisState, got %T", hybridState)
	}

	newLotMethodState = func(reportbasis.LotMethod) (*reportbasis.LotMethodState, error) {
		return nil, errors.New("lot constructor boom")
	}
	_, err = newAssetBasisState(reportmodel.CostBasisMethodFIFO)
	if err == nil || !strings.Contains(err.Error(), "lot constructor boom") {
		t.Fatalf("expected lot constructor failure to propagate, got %v", err)
	}
	_, err = newAssetBasisState(reportmodel.CostBasisMethodLIFO)
	if err == nil || !strings.Contains(err.Error(), "lot constructor boom") {
		t.Fatalf("expected LIFO constructor failure to propagate, got %v", err)
	}
	_, err = newAssetBasisState(reportmodel.CostBasisMethodHIFO)
	if err == nil || !strings.Contains(err.Error(), "lot constructor boom") {
		t.Fatalf("expected HIFO constructor failure to propagate, got %v", err)
	}

	_, err = newAssetBasisState(reportmodel.CostBasisMethod("unsupported"))
	if err == nil || !strings.Contains(err.Error(), "unsupported cost basis method") {
		t.Fatalf("expected unsupported basis method failure, got %v", err)
	}

	var sum apd.Decimal
	sum, err = addCalculationDecimal(mustReportDecimal(t, "2"), mustReportDecimal(t, "3"))
	if err != nil || sum.Cmp(apd.New(5, 0)) != 0 {
		t.Fatalf("expected exact calculation sum, got %v err=%v", sum, err)
	}
	var difference apd.Decimal
	difference, err = subtractCalculationDecimal(mustReportDecimal(t, "5"), mustReportDecimal(t, "3"))
	if err != nil || difference.Cmp(apd.New(2, 0)) != 0 {
		t.Fatalf("expected exact calculation difference, got %v err=%v", difference, err)
	}

	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	if _, err = addCalculationDecimal(invalid, mustReportDecimal(t, "1")); err == nil || !strings.Contains(err.Error(), "left calculation decimal") {
		t.Fatalf("expected invalid left addend to fail, got %v", err)
	}
	if _, err = addCalculationDecimal(mustReportDecimal(t, "1"), invalid); err == nil || !strings.Contains(err.Error(), "right calculation decimal") {
		t.Fatalf("expected invalid right addend to fail, got %v", err)
	}
	if _, err = subtractCalculationDecimal(invalid, mustReportDecimal(t, "1")); err == nil || !strings.Contains(err.Error(), "left calculation decimal") {
		t.Fatalf("expected invalid left minuend to fail, got %v", err)
	}
	if _, err = subtractCalculationDecimal(mustReportDecimal(t, "1"), invalid); err == nil || !strings.Contains(err.Error(), "right calculation decimal") {
		t.Fatalf("expected invalid right subtrahend to fail, got %v", err)
	}

	var recordErr = newRecordCalculationError(reportmodel.CalculationErrorKindActivityInput, syncmodel.ActivityRecord{
		SourceID:    " rec-1 ",
		AssetSymbol: " BTC ",
	}, "record failure", nil)
	var recordCalcErr = requireCalculationError(t, recordErr, reportmodel.CalculationErrorKindActivityInput)
	if recordCalcErr.SourceID() != "rec-1" || recordCalcErr.DisplayLabel() != "BTC" {
		t.Fatalf("expected record error references, got %#v", recordCalcErr)
	}

	var inputErr = newInputCalculationError(reportmodel.CalculationErrorKindBasisAllocation, reportmodel.ActivityCalculationInput{
		SourceID:     " input-1 ",
		DisplayLabel: " ETH ",
	}, "input failure", nil)
	var inputCalcErr = requireCalculationError(t, inputErr, reportmodel.CalculationErrorKindBasisAllocation)
	if inputCalcErr.SourceID() != "input-1" || inputCalcErr.DisplayLabel() != "ETH" {
		t.Fatalf("expected input error references, got %#v", inputCalcErr)
	}

	var lotWrapper = lotBasisState{}
	if _, err = lotWrapper.Dispose(basisDisposalInput{Quantity: mustReportDecimal(t, "1")}); err == nil {
		t.Fatalf("expected nil lot wrapper state disposal to fail")
	}

	var hybridWrapper = scopeLocalHybridBasisState{}
	if _, err = hybridWrapper.Dispose(basisDisposalInput{Quantity: mustReportDecimal(t, "1")}); err == nil {
		t.Fatalf("expected nil scope-local hybrid wrapper state disposal to fail")
	}
}

// reportInvalidDecimalForCalculator returns one non-finite decimal value for
// direct calculator helper error-path tests.
// Authored by: OpenCode
func reportInvalidDecimalForCalculator() apd.Decimal {
	var invalid apd.Decimal
	invalid.Form = apd.Infinite
	return invalid
}

// requireCalculationError verifies one wrapped calculation error kind.
// Authored by: OpenCode
func requireCalculationError(t *testing.T, err error, kind reportmodel.CalculationErrorKind) *reportmodel.CalculationError {
	t.Helper()

	if err == nil {
		t.Fatalf("expected calculation error kind %q", kind)
	}

	var calcErr *reportmodel.CalculationError
	if !errors.As(err, &calcErr) {
		t.Fatalf("expected calculation error, got %T: %v", err, err)
	}
	if calcErr.Kind() != kind {
		t.Fatalf("unexpected calculation error kind: got %q want %q", calcErr.Kind(), kind)
	}

	return calcErr
}

// validReportRequest returns one validated report request for calculator tests.
// Authored by: OpenCode
func validReportRequest(t *testing.T, year int, method reportmodel.CostBasisMethod) reportmodel.ReportRequest {
	t.Helper()

	var request, err = reportmodel.NewReportRequest(year, method, time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new report request: %v", err)
	}

	return request
}

// validAssetSummaryEntry returns one valid summary entry for calculator tests.
// Authored by: OpenCode
func validAssetSummaryEntry(t *testing.T, assetIdentityKey string, displayLabel string, yearlyNet string) reportmodel.AssetSummaryEntry {
	t.Helper()

	var entry, err = reportmodel.NewAssetSummaryEntry(assetIdentityKey, displayLabel, mustReportDecimal(t, yearlyNet), reportCalculationCurrencyLabel)
	if err != nil {
		t.Fatalf("new asset summary entry: %v", err)
	}

	return entry
}

// validAssetDetailSection returns one valid empty detail section for calculator
// tests.
// Authored by: OpenCode
func validAssetDetailSection(t *testing.T, assetIdentityKey string, displayLabel string) reportmodel.AssetDetailSection {
	t.Helper()

	var zero apd.Decimal
	var section, err = reportmodel.NewAssetDetailSection(
		assetIdentityKey,
		displayLabel,
		zero,
		zero,
		zero,
		zero,
		reportCalculationCurrencyLabel,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("new asset detail section: %v", err)
	}

	return section
}

// zeroPricedHoldingReductionRecord returns one selected-year record that skips
// priced-tier selection and remains valid for grouping tests.
// Authored by: OpenCode
func zeroPricedHoldingReductionRecord(t *testing.T, sourceID string, occurredAt string, assetIdentityKey string, assetSymbol string, assetName string) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         sourceID,
		OccurredAt:       occurredAt,
		ActivityType:     syncmodel.ActivityTypeSell,
		AssetIdentityKey: assetIdentityKey,
		AssetSymbol:      assetSymbol,
		AssetName:        assetName,
		Quantity:         mustReportDecimal(t, "1"),
		Comment:          "manual transfer",
	}
}

// decimalPointer returns one report-decimal pointer for calculator tests.
// Authored by: OpenCode
func decimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = mustReportDecimal(t, raw)
	return &value
}
