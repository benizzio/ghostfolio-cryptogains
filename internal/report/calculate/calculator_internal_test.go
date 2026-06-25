// Package calculate verifies package-local report calculation helpers and seams.
// Authored by: OpenCode
package calculate

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportbasis "github.com/benizzio/ghostfolio-cryptogains/internal/report/basis"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
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

// stubCurrencyRateService records calculator rate lookups for boundary tests.
// Authored by: OpenCode
type stubCurrencyRateService struct {
	lookupErr error
	evidences map[string]currencyintegration.ExchangeRateEvidence
	requests  []currencyintegration.RateLookupRequest
}

// LookupRate records one lookup and returns the configured result.
// Authored by: OpenCode
func (service *stubCurrencyRateService) LookupRate(_ context.Context, request currencyintegration.RateLookupRequest) (currencyintegration.ExchangeRateEvidence, error) {
	service.requests = append(service.requests, request)
	if service.lookupErr != nil {
		return currencyintegration.ExchangeRateEvidence{}, service.lookupErr
	}
	if service.evidences != nil {
		var evidence, ok = service.evidences[rateLookupRequestKey(request)]
		if ok {
			return evidence, nil
		}
	}

	return currencyintegration.ExchangeRateEvidence{}, nil
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

// TestCalculateAndAssetReplayWrapWrapperFailures verifies the remaining direct
// calculator wrapper branches through package-local seams.
// Authored by: OpenCode
func TestCalculateAndAssetReplayWrapWrapperFailures(t *testing.T) {
	t.Run("propagates asset-group calculation failure", func(t *testing.T) {
		var originalCalculateAssetGroupFunc = calculateAssetGroupFunc
		defer func() {
			calculateAssetGroupFunc = originalCalculateAssetGroupFunc
		}()

		calculateAssetGroupFunc = func(_ reportmodel.CostBasisMethod, _ int, _ syncmodel.ProtectedActivityCache, _ assetInputGroup) (assetCalculationResult, error) {
			return assetCalculationResult{}, errors.New("asset group boom")
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
		if err == nil || !strings.Contains(err.Error(), "asset group boom") {
			t.Fatalf("expected asset-group failure to propagate, got %v", err)
		}
	})

	t.Run("wraps yearly total accumulation failure", func(t *testing.T) {
		var originalCalculateAssetGroupFunc = calculateAssetGroupFunc
		defer func() {
			calculateAssetGroupFunc = originalCalculateAssetGroupFunc
		}()

		calculateAssetGroupFunc = func(_ reportmodel.CostBasisMethod, _ int, _ syncmodel.ProtectedActivityCache, _ assetInputGroup) (assetCalculationResult, error) {
			return assetCalculationResult{
				IncludeInMain: true,
				SummaryEntry:  validAssetSummaryEntry(t, "asset-btc", "BTC", "1"),
				DetailSection: validAssetDetailSection(t, "asset-btc", "BTC"),
				YearlyNet:     reportInvalidDecimalForCalculator(),
			}, nil
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
		if !strings.Contains(calcErr.Error(), "could not accumulate the yearly report total") {
			t.Fatalf("expected wrapped yearly-total accumulation failure, got %q", calcErr.Error())
		}
	})

	t.Run("wraps basis-state constructor failure", func(t *testing.T) {
		var originalNewAssetBasisState = newAssetBasisStateFunc
		defer func() {
			newAssetBasisStateFunc = originalNewAssetBasisState
		}()

		newAssetBasisStateFunc = func(reportmodel.CostBasisMethod) (assetBasisState, error) {
			return nil, errors.New("basis constructor boom")
		}

		_, err := calculateAssetGroup(reportmodel.CostBasisMethodFIFO, 2024, syncmodel.ProtectedActivityCache{}, assetInputGroup{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
		})
		var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindUnsupportedCostBasisMethod)
		if !strings.Contains(calcErr.Error(), "basis constructor boom") {
			t.Fatalf("expected wrapped basis constructor failure, got %q", calcErr.Error())
		}
	})

	t.Run("wraps scoped-input resolution failure", func(t *testing.T) {
		var originalResolveScopedInputs = resolveScopedInputsFunc
		defer func() {
			resolveScopedInputsFunc = originalResolveScopedInputs
		}()

		resolveScopedInputsFunc = func(reportmodel.CostBasisMethod, assetInputGroup) ([]scopedActivityInput, error) {
			return nil, errors.New("scope resolution boom")
		}

		_, err := calculateAssetGroup(reportmodel.CostBasisMethodFIFO, 2024, syncmodel.ProtectedActivityCache{}, assetInputGroup{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
		})
		if err == nil || !strings.Contains(err.Error(), "scope resolution boom") {
			t.Fatalf("expected scoped-input failure to propagate, got %v", err)
		}
	})

	t.Run("wraps opening position lookup failure", func(t *testing.T) {
		var originalNewAssetBasisState = newAssetBasisStateFunc
		var originalResolveScopedInputs = resolveScopedInputsFunc
		defer func() {
			newAssetBasisStateFunc = originalNewAssetBasisState
			resolveScopedInputsFunc = originalResolveScopedInputs
		}()

		newAssetBasisStateFunc = func(reportmodel.CostBasisMethod) (assetBasisState, error) {
			return stubAssetBasisState{
				openQuantityFunc: func() (apd.Decimal, error) { return apd.Decimal{}, errors.New("opening quantity boom") },
				openBasisFunc:    func() (apd.Decimal, error) { return mustReportDecimal(t, "10"), nil },
			}, nil
		}

		resolveScopedInputsFunc = func(reportmodel.CostBasisMethod, assetInputGroup) ([]scopedActivityInput, error) {
			return []scopedActivityInput{{Input: reportmodel.ActivityCalculationInput{
				SourceID:         "buy-1",
				OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				SourceYear:       2024,
				ActivityType:     reportmodel.ActivityTypeBuy,
				AssetIdentityKey: "asset-btc",
				DisplayLabel:     "BTC",
				Quantity:         mustReportDecimal(t, "1"),
			}}}, nil
		}

		_, err := calculateAssetGroup(reportmodel.CostBasisMethodAverageCost, 2024, syncmodel.ProtectedActivityCache{}, assetInputGroup{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
		})
		var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
		if !strings.Contains(calcErr.Error(), "could not determine the opening position carried into the selected year") {
			t.Fatalf("expected wrapped opening-position failure, got %q", calcErr.Error())
		}
	})

	t.Run("wraps opening basis lookup failure", func(t *testing.T) {
		var originalNewAssetBasisState = newAssetBasisStateFunc
		var originalResolveScopedInputs = resolveScopedInputsFunc
		defer func() {
			newAssetBasisStateFunc = originalNewAssetBasisState
			resolveScopedInputsFunc = originalResolveScopedInputs
		}()

		newAssetBasisStateFunc = func(reportmodel.CostBasisMethod) (assetBasisState, error) {
			return stubAssetBasisState{
				openQuantityFunc: func() (apd.Decimal, error) { return mustReportDecimal(t, "1"), nil },
				openBasisFunc:    func() (apd.Decimal, error) { return apd.Decimal{}, errors.New("opening basis boom") },
			}, nil
		}

		resolveScopedInputsFunc = func(reportmodel.CostBasisMethod, assetInputGroup) ([]scopedActivityInput, error) {
			return []scopedActivityInput{{Input: reportmodel.ActivityCalculationInput{
				SourceID:         "buy-1",
				OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				SourceYear:       2024,
				ActivityType:     reportmodel.ActivityTypeBuy,
				AssetIdentityKey: "asset-btc",
				DisplayLabel:     "BTC",
				Quantity:         mustReportDecimal(t, "1"),
			}}}, nil
		}

		_, err := calculateAssetGroup(reportmodel.CostBasisMethodAverageCost, 2024, syncmodel.ProtectedActivityCache{}, assetInputGroup{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
		})
		var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
		if !strings.Contains(calcErr.Error(), "could not determine the opening position carried into the selected year") {
			t.Fatalf("expected wrapped opening-basis failure, got %q", calcErr.Error())
		}
	})

	t.Run("wraps replay and closing-state failures", func(t *testing.T) {
		var originalResolveScopedInputs = resolveScopedInputsFunc
		var originalReplayAssetInput = replayAssetInputFunc
		var originalNewAssetBasisState = newAssetBasisStateFunc
		defer func() {
			resolveScopedInputsFunc = originalResolveScopedInputs
			replayAssetInputFunc = originalReplayAssetInput
			newAssetBasisStateFunc = originalNewAssetBasisState
		}()

		var scopedInputs = []scopedActivityInput{{Input: reportmodel.ActivityCalculationInput{
			SourceID:         "buy-2",
			OccurredAt:       time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC),
			SourceYear:       2023,
			ActivityType:     reportmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
			Quantity:         mustReportDecimal(t, "1"),
		}}}
		resolveScopedInputsFunc = func(reportmodel.CostBasisMethod, assetInputGroup) ([]scopedActivityInput, error) {
			return scopedInputs, nil
		}

		replayAssetInputFunc = func(assetBasisState, scopedActivityInput, int, int) (assetInputReplayResult, error) {
			return assetInputReplayResult{}, errors.New("replay boom")
		}
		newAssetBasisStateFunc = func(reportmodel.CostBasisMethod) (assetBasisState, error) {
			return stubAssetBasisState{
				openQuantityFunc: func() (apd.Decimal, error) { return mustReportDecimal(t, "1"), nil },
				openBasisFunc:    func() (apd.Decimal, error) { return mustReportDecimal(t, "10"), nil },
			}, nil
		}
		_, err := calculateAssetGroup(reportmodel.CostBasisMethodAverageCost, 2024, syncmodel.ProtectedActivityCache{}, assetInputGroup{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
		})
		if err == nil || !strings.Contains(err.Error(), "replay boom") {
			t.Fatalf("expected replay failure to propagate, got %v", err)
		}

		replayAssetInputFunc = func(assetBasisState, scopedActivityInput, int, int) (assetInputReplayResult, error) {
			return assetInputReplayResult{}, nil
		}
		newAssetBasisStateFunc = func(reportmodel.CostBasisMethod) (assetBasisState, error) {
			return stubAssetBasisState{
				openQuantityFunc: func() (apd.Decimal, error) { return apd.Decimal{}, errors.New("closing quantity boom") },
				openBasisFunc:    func() (apd.Decimal, error) { return mustReportDecimal(t, "10"), nil },
			}, nil
		}
		_, err = calculateAssetGroup(reportmodel.CostBasisMethodAverageCost, 2024, syncmodel.ProtectedActivityCache{}, assetInputGroup{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
		})
		var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
		if !strings.Contains(calcErr.Error(), "could not determine the asset closing quantity") {
			t.Fatalf("expected wrapped closing-quantity failure, got %q", calcErr.Error())
		}
		if calcErr.DisplayLabel() != "BTC" {
			t.Fatalf("expected closing-quantity failure to preserve display label, got %#v", calcErr)
		}
		newAssetBasisStateFunc = func(reportmodel.CostBasisMethod) (assetBasisState, error) {
			return stubAssetBasisState{
				openQuantityFunc: func() (apd.Decimal, error) { return mustReportDecimal(t, "1"), nil },
				openBasisFunc:    func() (apd.Decimal, error) { return apd.Decimal{}, errors.New("closing basis boom") },
			}, nil
		}
		_, err = calculateAssetGroup(reportmodel.CostBasisMethodFIFO, 2024, syncmodel.ProtectedActivityCache{}, assetInputGroup{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
		})
		calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
		if !strings.Contains(calcErr.Error(), "could not determine the asset closing basis") {
			t.Fatalf("expected wrapped closing-basis failure, got %q", calcErr.Error())
		}

		replayAssetInputFunc = func(assetBasisState, scopedActivityInput, int, int) (assetInputReplayResult, error) {
			return assetInputReplayResult{yearlyNetDelta: reportInvalidDecimalForCalculator()}, nil
		}
		newAssetBasisStateFunc = func(reportmodel.CostBasisMethod) (assetBasisState, error) {
			return stubAssetBasisState{
				openQuantityFunc: func() (apd.Decimal, error) { return mustReportDecimal(t, "1"), nil },
				openBasisFunc:    func() (apd.Decimal, error) { return mustReportDecimal(t, "10"), nil },
			}, nil
		}
		_, err = calculateAssetGroup(reportmodel.CostBasisMethodFIFO, 2024, syncmodel.ProtectedActivityCache{}, assetInputGroup{
			AssetIdentityKey: "asset-btc",
			DisplayLabel:     "BTC",
		})
		calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
		if !strings.Contains(calcErr.Error(), "could not accumulate the asset yearly gain or loss") {
			t.Fatalf("expected wrapped asset-yearly-net failure, got %q", calcErr.Error())
		}
	})

	t.Run("wraps in-year artifact failure", func(t *testing.T) {
		_, err := replayAssetInput(stubAssetBasisState{
			disposeFunc: func(basisDisposalInput) (basisDisposalResult, error) {
				return basisDisposalResult{AllocatedBasis: mustReportDecimal(t, "5")}, nil
			},
			openQuantityFunc: func() (apd.Decimal, error) { return mustReportDecimal(t, "1"), nil },
			openBasisFunc:    func() (apd.Decimal, error) { return mustReportDecimal(t, "10"), nil },
		}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
			SourceID:             "buy-3",
			OccurredAt:           time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			SourceYear:           2024,
			ActivityType:         reportmodel.ActivityTypeSell,
			AssetIdentityKey:     "asset-btc",
			DisplayLabel:         "BTC",
			Quantity:             reportInvalidDecimalForCalculator(),
			GrossValue:           decimalPointer(t, "10"),
			FeeAmount:            decimalPointer(t, "0"),
			SelectedCurrencyCode: "USD",
		}}, 1, 2024)
		var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
		if !strings.Contains(calcErr.Error(), "could not build the in-year activity row") {
			t.Fatalf("expected wrapped in-year artifact failure, got %q", calcErr.Error())
		}
	})

	t.Run("wraps reference entry validation failure", func(t *testing.T) {
		_, err := buildAssetCalculationResult(assetInputGroup{AssetIdentityKey: " ", DisplayLabel: "BTC"}, assetReplayState{fullLiquidationCount: 1})
		var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
		if !strings.Contains(calcErr.Error(), "could not build the reference-section entry") {
			t.Fatalf("expected wrapped reference-entry failure, got %q", calcErr.Error())
		}
	})
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

// TestApplyReportCurrencyBoundaryBypassesRowsWithoutLookup verifies same-
// currency rows and zero-priced holding reductions do not call the rate seam.
// Authored by: OpenCode
func TestApplyReportCurrencyBoundaryBypassesRowsWithoutLookup(t *testing.T) {
	t.Parallel()

	var service = &stubCurrencyRateService{lookupErr: errors.New("unexpected lookup")}
	var result, err = applyReportCurrencyBoundary(context.Background(), service, reportmodel.ReportBaseCurrencyUSD, []assetInputGroup{{
		AssetIdentityKey: "asset-unit",
		DisplayLabel:     "UNIT",
		Inputs: []reportmodel.ActivityCalculationInput{
			{
				SourceID:             "same-currency-buy",
				OccurredAt:           time.Date(2024, time.January, 2, 10, 0, 0, 0, time.UTC),
				SourceYear:           2024,
				ActivityType:         reportmodel.ActivityTypeBuy,
				Quantity:             mustReportDecimal(t, "1"),
				SelectedCurrencyCode: "USD",
			},
			{
				SourceID:                     "zero-priced-sell",
				OccurredAt:                   time.Date(2024, time.January, 3, 10, 0, 0, 0, time.UTC),
				SourceYear:                   2024,
				ActivityType:                 reportmodel.ActivityTypeSell,
				Quantity:                     mustReportDecimal(t, "1"),
				UnitPrice:                    activityInputDecimalPointer(t, "0"),
				GrossValue:                   activityInputDecimalPointer(t, "0"),
				FeeAmount:                    activityInputDecimalPointer(t, "0"),
				SelectedCurrencyCode:         "EUR",
				IsZeroPricedHoldingReduction: true,
			},
		},
	}})
	if err != nil {
		t.Fatalf("apply report currency boundary: %v", err)
	}
	if len(service.requests) != 0 {
		t.Fatalf("expected no rate lookups, got %#v", service.requests)
	}
	var groups = result.Groups
	if groups[0].Inputs[0].SelectedCurrencyCode != "USD" || groups[0].Inputs[1].SelectedCurrencyCode != "USD" {
		t.Fatalf("expected selected report currency to be prepared on bypassed rows, got %#v", groups[0].Inputs)
	}
	if len(result.ConversionAuditEntries) != 0 || len(result.RateSources) != 0 {
		t.Fatalf("expected no conversion artifacts for bypassed rows, got %#v", result)
	}
}

// TestCalculateZeroPricedHoldingReductionDoesNotLookupCrossCurrencyRate verifies
// a zero-priced no-cost reduction does not require rate evidence even when its
// preserved source currency differs from the report base currency.
// Authored by: OpenCode
func TestCalculateZeroPricedHoldingReductionDoesNotLookupCrossCurrencyRate(t *testing.T) {
	t.Parallel()

	var service = &stubCurrencyRateService{lookupErr: errors.New("unexpected rate lookup for zero-priced reduction")}
	var calculator = NewCalculator(service)
	var buy = zeroPricedNoLookupAcquisitionRecord(t)
	var reduction = zeroPricedHoldingReductionRecord(t, "zero-reduction-eur-sell", "2024-02-02T10:00:00Z", "asset-zero-no-lookup", "ZNL", "Zero No Lookup")
	reduction.OrderCurrency = "EUR"

	var report, err = calculator.Calculate(context.Background(), validReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO), syncmodel.ProtectedActivityCache{
		AvailableReportYears: []int{2024},
		Activities:           []syncmodel.ActivityRecord{buy, reduction},
	})
	if err != nil {
		t.Fatalf("expected zero-priced cross-currency holding reduction to calculate without lookup, got %v", err)
	}
	if len(service.requests) != 0 {
		t.Fatalf("expected no lookup for zero-priced holding reduction, got %#v", service.requests)
	}
	if len(report.RateSources) != 0 || len(report.ConversionAuditEntries) != 0 {
		t.Fatalf("expected no conversion artifacts for zero-priced holding reduction, got %#v", report)
	}
}

// TestCalculateFutureZeroPricedHoldingReductionDoesNotLookupRate verifies rows
// after the selected report year are ignored before any rate lookup can occur.
// Authored by: OpenCode
func TestCalculateFutureZeroPricedHoldingReductionDoesNotLookupRate(t *testing.T) {
	t.Parallel()

	var service = &stubCurrencyRateService{lookupErr: errors.New("unexpected future-year lookup")}
	var calculator = NewCalculator(service)
	var reduction = zeroPricedHoldingReductionRecord(t, "future-zero-reduction-gbp-sell", "2025-02-02T10:00:00Z", "asset-zero-no-lookup", "ZNL", "Zero No Lookup")
	reduction.OrderCurrency = "GBP"

	_, err := calculator.Calculate(context.Background(), validReportRequest(t, 2024, reportmodel.CostBasisMethodFIFO), syncmodel.ProtectedActivityCache{
		AvailableReportYears: []int{2024},
		Activities:           []syncmodel.ActivityRecord{reduction},
	})
	if err != nil {
		t.Fatalf("expected future zero-priced holding reduction to be ignored without lookup, got %v", err)
	}
	if len(service.requests) != 0 {
		t.Fatalf("expected no lookup for future zero-priced holding reduction, got %#v", service.requests)
	}
}

// TestApplyReportCurrencyBoundaryUsesRateServiceForCrossCurrency verifies cross-
// currency priced rows use the seam and surface lookup failures safely.
// Authored by: OpenCode
func TestApplyReportCurrencyBoundaryUsesRateServiceForCrossCurrency(t *testing.T) {
	t.Parallel()

	var occurredAt = time.Date(2024, time.February, 3, 22, 0, 0, 0, time.FixedZone("source", 2*60*60))
	var activityDate = time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC)
	var request = mustCalculatorRateLookupRequest(t, "EUR", "USD", activityDate)
	var service = &stubCurrencyRateService{lookupErr: currencyintegration.NewConversionFailure(
		request,
		currencyintegration.ProviderIDFederalReserveH10,
		currencyintegration.ConversionFailureReasonMissingRate,
		"raw provider detail with Bearer jwt-secret and amount 1000.25",
	)}
	_, err := applyReportCurrencyBoundary(context.Background(), service, reportmodel.ReportBaseCurrencyUSD, []assetInputGroup{{
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Inputs: []reportmodel.ActivityCalculationInput{{
			SourceID:             "eur-buy",
			OccurredAt:           occurredAt,
			SourceYear:           2024,
			ActivityType:         reportmodel.ActivityTypeBuy,
			Quantity:             mustReportDecimal(t, "1"),
			SelectedCurrencyCode: "EUR",
		}},
	}})
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)
	for _, expected := range []string{"reason=missing_rate", "source_currency=EUR", "report_base_currency=USD", "activity_date=2024-02-03", "provider=federal_reserve_h10", `source "eur-buy"`} {
		if !strings.Contains(calcErr.Error(), expected) {
			t.Fatalf("expected classified conversion failure to contain %q, got %q", expected, calcErr.Error())
		}
	}
	var reason, ok = currencyintegration.ConversionFailureReasonOf(err)
	if !ok || reason != currencyintegration.ConversionFailureReasonMissingRate {
		t.Fatalf("expected conversion failure reason extraction through calculation error, got reason=%q ok=%v", reason, ok)
	}
	var diagnosticChain = strings.Join(calcErr.DiagnosticFailureCauseChain(), "\n")
	for _, forbidden := range []string{"jwt-secret", "1000.25", "raw provider detail"} {
		if strings.Contains(diagnosticChain, forbidden) {
			t.Fatalf("expected diagnostic cause chain to exclude %q, got %#v", forbidden, calcErr.DiagnosticFailureCauseChain())
		}
	}
	if len(service.requests) != 1 {
		t.Fatalf("expected one rate lookup, got %#v", service.requests)
	}
	var actualRequest = service.requests[0]
	if actualRequest.SourceCurrency != "EUR" || actualRequest.BaseCurrency != "USD" || !actualRequest.ActivityDate.Equal(activityDate) {
		t.Fatalf("unexpected rate lookup request: %#v", actualRequest)
	}
}

// TestApplyReportCurrencyBoundaryConvertsSingleActivityWithoutTierMixing
// verifies that one selected monetary context is converted as a unit and that
// lower-priority monetary tiers cannot mix into the converted report input.
// Authored by: OpenCode
func TestApplyReportCurrencyBoundaryConvertsSingleActivityWithoutTierMixing(t *testing.T) {
	t.Parallel()

	var occurredAt = time.Date(2024, time.January, 5, 14, 30, 0, 0, time.FixedZone("source", -5*60*60))
	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var request = mustCalculatorRateLookupRequest(t, "EUR", "USD", activityDate)
	var service = &stubCurrencyRateService{evidences: map[string]currencyintegration.ExchangeRateEvidence{
		rateLookupRequestKey(request): mustCalculatorRateEvidence(
			t,
			request,
			activityDate,
			currencyintegration.RateAuthorityFederalReserve,
			currencyintegration.ProviderIDFederalReserveH10,
			currencyintegration.RateKindFederalReserveH10NoonBuying,
			currencyintegration.QuoteDirectionSourcePerBase,
			"2",
			"H10/EUR/unstarred/2024-01-05",
		),
	}}

	var result, err = applyReportCurrencyBoundary(context.Background(), service, reportmodel.ReportBaseCurrencyUSD, []assetInputGroup{{
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Inputs: []reportmodel.ActivityCalculationInput{{
			SourceID:                "eur-order-tier-buy",
			OccurredAt:              occurredAt,
			SourceYear:              2024,
			ActivityType:            reportmodel.ActivityTypeBuy,
			AssetIdentityKey:        "asset-btc",
			DisplayLabel:            "BTC",
			Quantity:                mustReportDecimal(t, "2"),
			UnitPrice:               decimalPointer(t, "50"),
			GrossValue:              decimalPointer(t, "100"),
			FeeAmount:               decimalPointer(t, "4"),
			SelectedCurrencyContext: reportmodel.SelectedCurrencyContextOrder,
			SelectedCurrencyCode:    "EUR",
		}},
	}})
	if err != nil {
		t.Fatalf("expected single-activity conversion success, got %v", err)
	}
	if len(service.requests) != 1 {
		t.Fatalf("expected one lookup for one selected context, got %#v", service.requests)
	}

	var input = result.Groups[0].Inputs[0]
	if input.SelectedCurrencyContext != reportmodel.SelectedCurrencyContextOrder || input.SelectedCurrencyCode != "USD" {
		t.Fatalf("expected converted order context in USD without tier mixing, got %#v", input)
	}
	assertReportDecimalPointer(t, input.UnitPrice, "25")
	assertReportDecimalPointer(t, input.GrossValue, "50")
	assertReportDecimalPointer(t, input.FeeAmount, "2")
	if len(result.ConversionAuditEntries) != 1 || len(result.RateSources) != 1 {
		t.Fatalf("expected one conversion audit and one rate source, got %#v", result)
	}
	var audit = result.ConversionAuditEntries[0]
	if audit.SourceID != "eur-order-tier-buy" || audit.AssetLabel != "BTC" || audit.SourceCurrency != "EUR" || audit.ReportBaseCurrency != reportmodel.ReportBaseCurrencyUSD {
		t.Fatalf("unexpected conversion audit identity: %#v", audit)
	}
	if len(audit.Amounts) != 3 {
		t.Fatalf("expected three converted audit amounts, got %#v", audit.Amounts)
	}
	assertReportConvertedAmount(t, audit.Amounts[0], reportmodel.ConvertedAmountKindUnitPrice, "50", "25")
	assertReportConvertedAmount(t, audit.Amounts[1], reportmodel.ConvertedAmountKindGrossValue, "100", "50")
	assertReportConvertedAmount(t, audit.Amounts[2], reportmodel.ConvertedAmountKindFeeAmount, "4", "2")
}

// TestApplyReportCurrencyBoundaryResolvesEvidencePerUniqueRateKey verifies that
// repeated source/base/date keys reuse one resolved evidence record before replay.
// Authored by: OpenCode
func TestApplyReportCurrencyBoundaryResolvesEvidencePerUniqueRateKey(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var request = mustCalculatorRateLookupRequest(t, "EUR", "USD", activityDate)
	var service = &stubCurrencyRateService{evidences: map[string]currencyintegration.ExchangeRateEvidence{
		rateLookupRequestKey(request): mustCalculatorRateEvidence(
			t,
			request,
			activityDate,
			currencyintegration.RateAuthorityFederalReserve,
			currencyintegration.ProviderIDFederalReserveH10,
			currencyintegration.RateKindFederalReserveH10NoonBuying,
			currencyintegration.QuoteDirectionSourcePerBase,
			"2",
			"H10/EUR/unstarred/2024-01-05",
		),
	}}

	var result, err = applyReportCurrencyBoundary(context.Background(), service, reportmodel.ReportBaseCurrencyUSD, []assetInputGroup{{
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Inputs: []reportmodel.ActivityCalculationInput{
			{
				SourceID:             "eur-buy-1",
				OccurredAt:           activityDate,
				SourceYear:           2024,
				ActivityType:         reportmodel.ActivityTypeBuy,
				AssetIdentityKey:     "asset-btc",
				DisplayLabel:         "BTC",
				Quantity:             mustReportDecimal(t, "1"),
				GrossValue:           decimalPointer(t, "100"),
				SelectedCurrencyCode: "EUR",
			},
			{
				SourceID:             "eur-buy-2",
				OccurredAt:           activityDate.Add(2 * time.Hour),
				SourceYear:           2024,
				ActivityType:         reportmodel.ActivityTypeBuy,
				AssetIdentityKey:     "asset-btc",
				DisplayLabel:         "BTC",
				Quantity:             mustReportDecimal(t, "1"),
				GrossValue:           decimalPointer(t, "80"),
				SelectedCurrencyCode: "EUR",
			},
		},
	}})
	if err != nil {
		t.Fatalf("expected unique-key conversion success, got %v", err)
	}
	if len(service.requests) != 1 {
		t.Fatalf("expected one lookup for repeated key, got %#v", service.requests)
	}
	if len(result.ConversionAuditEntries) != 2 || len(result.RateSources) != 1 {
		t.Fatalf("expected two audit entries and one rate source, got %#v", result)
	}
	assertReportDecimalPointer(t, result.Groups[0].Inputs[0].GrossValue, "50")
	assertReportDecimalPointer(t, result.Groups[0].Inputs[1].GrossValue, "40")
}

// TestApplyReportCurrencyBoundaryConvertsZeroValuedMonetaryFields verifies that
// explicit zero unit prices, gross amounts, and fees remain valid retained audit
// values after a cross-currency conversion boundary.
// Authored by: OpenCode
func TestApplyReportCurrencyBoundaryConvertsZeroValuedMonetaryFields(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.February, 9, 0, 0, 0, 0, time.UTC)
	var request = mustCalculatorRateLookupRequest(t, "GBP", "USD", activityDate)
	var service = &stubCurrencyRateService{evidences: map[string]currencyintegration.ExchangeRateEvidence{
		rateLookupRequestKey(request): mustCalculatorRateEvidence(
			t,
			request,
			activityDate,
			currencyintegration.RateAuthorityFederalReserve,
			currencyintegration.ProviderIDFederalReserveH10,
			currencyintegration.RateKindFederalReserveH10NoonBuying,
			currencyintegration.QuoteDirectionBasePerSource,
			"1.25",
			"H10/GBP/starred/2024-02-09",
		),
	}}

	var result, err = applyReportCurrencyBoundary(context.Background(), service, reportmodel.ReportBaseCurrencyUSD, []assetInputGroup{{
		AssetIdentityKey: "asset-zero",
		DisplayLabel:     "ZERO",
		Inputs: []reportmodel.ActivityCalculationInput{{
			SourceID:                "gbp-zero-buy",
			OccurredAt:              activityDate,
			SourceYear:              2024,
			ActivityType:            reportmodel.ActivityTypeBuy,
			AssetIdentityKey:        "asset-zero",
			DisplayLabel:            "ZERO",
			Quantity:                mustReportDecimal(t, "1"),
			UnitPrice:               decimalPointer(t, "0"),
			GrossValue:              decimalPointer(t, "0"),
			FeeAmount:               decimalPointer(t, "0"),
			SelectedCurrencyContext: reportmodel.SelectedCurrencyContextOrder,
			SelectedCurrencyCode:    "GBP",
		}},
	}})
	if err != nil {
		t.Fatalf("expected zero-valued conversion success, got %v", err)
	}

	var input = result.Groups[0].Inputs[0]
	if input.SelectedCurrencyCode != "USD" {
		t.Fatalf("expected converted input currency USD, got %#v", input)
	}
	assertReportDecimalPointer(t, input.UnitPrice, "0")
	assertReportDecimalPointer(t, input.GrossValue, "0")
	assertReportDecimalPointer(t, input.FeeAmount, "0")
	if len(result.ConversionAuditEntries) != 1 || len(result.ConversionAuditEntries[0].Amounts) != 3 {
		t.Fatalf("expected grouped retained zero-valued fields in conversion audit, got %#v", result.ConversionAuditEntries)
	}
	for _, amount := range result.ConversionAuditEntries[0].Amounts {
		if amount.OriginalAmount.Sign() != 0 || amount.ConvertedAmount.Sign() != 0 {
			t.Fatalf("expected retained zero-to-zero amount slots for calculation integrity, got %#v", result.ConversionAuditEntries[0].Amounts)
		}
	}
}

// TestReportCurrencyBoundaryAttachesRedactedDiagnosticRecord verifies persisted
// source activity context is attached only to structured calculation failures.
// Authored by: OpenCode
func TestReportCurrencyBoundaryAttachesRedactedDiagnosticRecord(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var service = &stubCurrencyRateService{lookupErr: errors.New("unexpected lookup")}
	var _, err = applyReportCurrencyBoundaryWithRecords(context.Background(), service, reportmodel.ReportBaseCurrencyUSD, []assetInputGroup{{
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Inputs: []reportmodel.ActivityCalculationInput{{
			SourceID:             "bad-currency-buy",
			OccurredAt:           activityDate,
			SourceYear:           2024,
			ActivityType:         reportmodel.ActivityTypeBuy,
			Quantity:             mustReportDecimal(t, "1"),
			SelectedCurrencyCode: "usd",
		}},
	}}, []syncmodel.ActivityRecord{{
		SourceID:   "bad-currency-buy",
		OccurredAt: activityDate.Format(time.RFC3339),
		DataSource: "Bearer source-secret",
		RawHash:    "token=hash-secret",
		Comment:    "private user comment",
	}})
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)
	var diagnosticErr, ok = err.(diagnosticCalculationError)
	if !ok {
		t.Fatalf("expected diagnostic calculation error, got %T", err)
	}
	var context = diagnosticErr.DiagnosticReportContext()
	if context.OffendingActivityRecord == nil || context.OffendingActivityRecord.SourceID != "bad-currency-buy" {
		t.Fatalf("expected offending persisted activity record, got %#v", context.OffendingActivityRecord)
	}
	if context.OffendingActivityRecord.Comment != "" || strings.Contains(context.OffendingActivityRecord.DataSource, "source-secret") || strings.Contains(context.OffendingActivityRecord.RawHash, "hash-secret") {
		t.Fatalf("expected persisted activity record to be redacted, got %#v", context.OffendingActivityRecord)
	}
	if !strings.Contains(calcErr.Error(), "could not prepare currency conversion") {
		t.Fatalf("expected invalid currency calculation error, got %q", calcErr.Error())
	}
}

// TestReportCurrencyBoundaryDefensiveConversionBranches verifies nil context,
// nil rate service, malformed evidence, and invalid arithmetic branches.
// Authored by: OpenCode
func TestReportCurrencyBoundaryDefensiveConversionBranches(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var request = mustCalculatorRateLookupRequest(t, "EUR", "USD", activityDate)
	var input = reportmodel.ActivityCalculationInput{
		SourceID:             "eur-buy",
		OccurredAt:           activityDate,
		SourceYear:           2024,
		ActivityType:         reportmodel.ActivityTypeBuy,
		AssetIdentityKey:     "asset-btc",
		DisplayLabel:         "BTC",
		Quantity:             mustReportDecimal(t, "1"),
		GrossValue:           decimalPointer(t, "100"),
		SelectedCurrencyCode: "EUR",
	}

	var _, nilServiceErr = applyReportCurrencyBoundary(nil, nil, reportmodel.ReportBaseCurrencyUSD, []assetInputGroup{{
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Inputs:           []reportmodel.ActivityCalculationInput{input},
	}})
	if nilServiceErr == nil || !strings.Contains(nilServiceErr.Error(), "requires a configured currency rate service") {
		t.Fatalf("expected nil rate service failure, got %v", nilServiceErr)
	}

	var malformedEvidence = mustCalculatorRateEvidence(
		t,
		request,
		activityDate,
		currencyintegration.RateAuthorityFederalReserve,
		currencyintegration.ProviderIDFederalReserveH10,
		currencyintegration.RateKindFederalReserveH10NoonBuying,
		currencyintegration.QuoteDirectionSourcePerBase,
		"1.09",
		"H10/EUR",
	)
	malformedEvidence.BaseCurrency = "GBP"
	var service = &stubCurrencyRateService{evidences: map[string]currencyintegration.ExchangeRateEvidence{rateLookupRequestKey(request): malformedEvidence}}
	var _, evidenceErr = applyReportCurrencyBoundary(context.Background(), service, reportmodel.ReportBaseCurrencyUSD, []assetInputGroup{{
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Inputs:           []reportmodel.ActivityCalculationInput{input},
	}})
	if evidenceErr == nil || !strings.Contains(evidenceErr.Error(), "could not validate currency conversion evidence") {
		t.Fatalf("expected malformed evidence mapping failure, got %v", evidenceErr)
	}

	var invalidQuoteEvidence = mustCalculatorRateEvidence(
		t,
		request,
		activityDate,
		currencyintegration.RateAuthorityFederalReserve,
		currencyintegration.ProviderIDFederalReserveH10,
		currencyintegration.RateKindFederalReserveH10NoonBuying,
		currencyintegration.QuoteDirectionSourcePerBase,
		"1.09",
		"H10/EUR",
	)
	invalidQuoteEvidence.QuoteDirection = currencyintegration.QuoteDirection("ambiguous")
	var reportEvidence = reportmodel.ExchangeRateEvidence{
		SourceCurrency:   "EUR",
		BaseCurrency:     reportmodel.ReportBaseCurrencyUSD,
		ActivityDate:     activityDate,
		RateDate:         activityDate,
		Authority:        currencyintegration.RateAuthorityFederalReserve,
		ProviderID:       currencyintegration.ProviderIDFederalReserveH10,
		RateKind:         string(currencyintegration.RateKindFederalReserveH10NoonBuying),
		QuoteDirection:   currencyintegration.QuoteDirectionSourcePerBase,
		RateValue:        mustReportDecimal(t, "1.09"),
		DatasetReference: "H10/EUR",
	}
	var _, _, conversionErr = convertInputMonetaryAmounts(input, reportmodel.ReportBaseCurrencyUSD, invalidQuoteEvidence, reportEvidence)
	if conversionErr == nil || !strings.Contains(conversionErr.Error(), "could not convert gross_value") {
		t.Fatalf("expected invalid quote conversion failure, got %v", conversionErr)
	}

	input.GrossValue = nil
	input.UnitPrice = decimalPointer(t, "100")
	_, _, conversionErr = convertInputMonetaryAmounts(input, reportmodel.ReportBaseCurrencyUSD, invalidQuoteEvidence, reportEvidence)
	if conversionErr == nil || !strings.Contains(conversionErr.Error(), "could not convert unit_price") {
		t.Fatalf("expected invalid unit-price conversion failure, got %v", conversionErr)
	}

	input.UnitPrice = nil
	input.FeeAmount = decimalPointer(t, "1")
	_, _, conversionErr = convertInputMonetaryAmounts(input, reportmodel.ReportBaseCurrencyUSD, invalidQuoteEvidence, reportEvidence)
	if conversionErr == nil || !strings.Contains(conversionErr.Error(), "could not convert fee_amount") {
		t.Fatalf("expected invalid fee conversion failure, got %v", conversionErr)
	}

	var eurRequest = mustCalculatorRateLookupRequest(t, "USD", "EUR", activityDate)
	var eurEvidence = mustCalculatorRateEvidence(
		t,
		eurRequest,
		activityDate,
		currencyintegration.RateAuthorityEuropeanCentralBank,
		currencyintegration.ProviderIDECBEXR,
		currencyintegration.RateKindECBEXRDailyReference,
		currencyintegration.QuoteDirectionSourcePerBase,
		"1.09",
		"EXR/D.USD.EUR.SP00.A",
	)
	if _, err := mapIntegrationEvidenceToReportEvidence(eurEvidence); err != nil {
		t.Fatalf("expected EUR report evidence mapping: %v", err)
	}
}

// TestReportCurrencyBoundaryHelperFallbackBranches verifies helper branches not
// naturally reached by successful report calculation flows.
// Authored by: OpenCode
func TestReportCurrencyBoundaryHelperFallbackBranches(t *testing.T) {
	t.Parallel()

	var boundary = &reportCurrencyBoundaryContext{recordBySourceID: map[string]syncmodel.ActivityRecord{"other": {SourceID: "other"}}}
	var passthrough = errors.New("plain")
	if got := boundary.withInputDiagnosticRecord(reportmodel.ActivityCalculationInput{SourceID: "missing"}, passthrough); got != passthrough {
		t.Fatalf("expected non-calculation error passthrough")
	}
	var calcErr = reportmodel.NewCalculationError(reportmodel.CalculationErrorKindActivityInput, "failure", "missing", "", nil)
	if got := boundary.withInputDiagnosticRecord(reportmodel.ActivityCalculationInput{SourceID: "missing"}, calcErr); got != calcErr {
		t.Fatalf("expected missing diagnostic record passthrough")
	}
	var indexed = recordBySourceID([]syncmodel.ActivityRecord{{SourceID: " "}, {SourceID: " kept "}})
	if len(indexed) != 1 || indexed["kept"].SourceID != " kept " {
		t.Fatalf("expected blank source IDs to be skipped, got %#v", indexed)
	}

	var calculator = NewCalculator(nil)
	_, err := calculator.Calculate(nil, reportmodel.ReportRequest{}, syncmodel.ProtectedActivityCache{})
	if err == nil || !strings.Contains(err.Error(), "report request year") {
		t.Fatalf("expected nil context calculator path to validate request, got %v", err)
	}

	if got := safeConversionLookupFallbackMessage("", errors.New("missing_rate: raw detail")); got != "currency conversion lookup failed: reason=missing_rate" {
		t.Fatalf("unexpected empty fallback conversion message %q", got)
	}
	if got := safeConversionLookupFallbackMessage("fallback", nil); got != "fallback" {
		t.Fatalf("unexpected nil-cause fallback conversion message %q", got)
	}
	if got := safeConversionReasonPrefix(errors.New("unknown: raw detail")); got != "" {
		t.Fatalf("expected unknown conversion reason to be ignored, got %q", got)
	}
	var safeCause safeConversionFailureCause
	if got := safeCause.Error(); got != "conversion failed" {
		t.Fatalf("unexpected nil safe conversion cause message %q", got)
	}
	var target string
	if safeCause.As(&target) {
		t.Fatalf("expected safe conversion cause not to match unrelated target")
	}
}

// TestReportCurrencyBoundaryDirectValidationFailureBranches covers validation
// failures reachable only through package-local helper seams after rate evidence
// has already been resolved.
// Authored by: OpenCode
func TestReportCurrencyBoundaryDirectValidationFailureBranches(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var request = mustCalculatorRateLookupRequest(t, "EUR", "USD", activityDate)
	var evidence = mustCalculatorRateEvidence(
		t,
		request,
		activityDate,
		currencyintegration.RateAuthorityFederalReserve,
		currencyintegration.ProviderIDFederalReserveH10,
		currencyintegration.RateKindFederalReserveH10NoonBuying,
		currencyintegration.QuoteDirectionSourcePerBase,
		"2",
		"H10/EUR",
	)
	var reportEvidence, err = mapIntegrationEvidenceToReportEvidence(evidence)
	if err != nil {
		t.Fatalf("map report evidence: %v", err)
	}
	var input = reportmodel.ActivityCalculationInput{
		SourceID:             "eur-buy",
		OccurredAt:           activityDate,
		SourceYear:           2024,
		ActivityType:         reportmodel.ActivityTypeBuy,
		AssetIdentityKey:     "asset-btc",
		DisplayLabel:         "BTC",
		Quantity:             mustReportDecimal(t, "1"),
		GrossValue:           decimalPointer(t, "100"),
		SelectedCurrencyCode: "EUR",
	}

	var invalidReportEvidence = reportEvidence
	invalidReportEvidence.BaseCurrency = reportmodel.ReportBaseCurrencyEUR
	var _, _, amountErr = convertOptionalInputAmount(input, reportmodel.ReportBaseCurrencyUSD, evidence, invalidReportEvidence, reportmodel.ConvertedAmountKindGrossValue, input.GrossValue, nil)
	if amountErr == nil || !strings.Contains(amountErr.Error(), "could not validate converted gross_value") {
		t.Fatalf("expected converted amount validation failure, got %v", amountErr)
	}

	var invalidEvidence = evidence
	invalidEvidence.RateKind = " "
	if _, err = mapIntegrationEvidenceToReportEvidence(invalidEvidence); err == nil || !strings.Contains(err.Error(), "rate kind") {
		t.Fatalf("expected report evidence validation failure, got %v", err)
	}

	var key = rateLookupBoundaryKey(request)
	var boundary = &reportCurrencyBoundaryContext{
		ctx:                   context.Background(),
		currencyRates:         &stubCurrencyRateService{lookupErr: errors.New("unexpected lookup")},
		reportBaseCurrency:    reportmodel.ReportBaseCurrencyUSD,
		resolvedEvidenceByKey: map[string]currencyintegration.ExchangeRateEvidence{key: evidence},
		reportEvidenceByKey:   map[string]reportmodel.ExchangeRateEvidence{key: invalidReportEvidence},
	}
	if _, _, err = boundary.applyInputReportCurrencyBoundary(assetInputGroup{AssetIdentityKey: "asset-btc", DisplayLabel: "BTC"}, input); err == nil || !strings.Contains(err.Error(), "could not validate converted gross_value") {
		t.Fatalf("expected apply input conversion validation failure, got %v", err)
	}

	boundary.reportEvidenceByKey[key] = reportEvidence
	input.SourceID = "eur-no-amounts"
	input.GrossValue = nil
	input.UnitPrice = nil
	input.FeeAmount = nil
	if _, _, err = boundary.applyInputReportCurrencyBoundary(assetInputGroup{AssetIdentityKey: "asset-btc", DisplayLabel: "BTC"}, input); err == nil || !strings.Contains(err.Error(), "conversion audit entry") {
		t.Fatalf("expected conversion audit validation failure, got %v", err)
	}
}

// TestConversionAuditEntryConstructionGuardrails verifies audit-entry label
// fallback and invalid-entry wrapping branches.
// Authored by: OpenCode
func TestConversionAuditEntryConstructionGuardrails(t *testing.T) {
	t.Parallel()

	var activityDate = time.Date(2024, time.January, 5, 0, 0, 0, 0, time.UTC)
	var request = mustCalculatorRateLookupRequest(t, "EUR", "USD", activityDate)
	var evidence = reportmodel.ExchangeRateEvidence{
		SourceCurrency:   "EUR",
		BaseCurrency:     reportmodel.ReportBaseCurrencyUSD,
		ActivityDate:     activityDate,
		RateDate:         activityDate,
		Authority:        currencyintegration.RateAuthorityFederalReserve,
		ProviderID:       currencyintegration.ProviderIDFederalReserveH10,
		RateKind:         string(currencyintegration.RateKindFederalReserveH10NoonBuying),
		QuoteDirection:   currencyintegration.QuoteDirectionSourcePerBase,
		RateValue:        mustReportDecimal(t, "2"),
		DatasetReference: "H10/EUR/2024-01-05",
	}
	var amount = reportmodel.ConvertedActivityAmount{
		SourceID:             "eur-buy-1",
		AmountKind:           reportmodel.ConvertedAmountKindGrossValue,
		OriginalCurrency:     "EUR",
		OriginalAmount:       mustReportDecimal(t, "100"),
		ReportBaseCurrency:   reportmodel.ReportBaseCurrencyUSD,
		ConvertedAmount:      mustReportDecimal(t, "50"),
		ExchangeRateEvidence: &evidence,
		ConversionStatus:     reportmodel.ConversionStatusConverted,
	}

	var entry, err = buildConversionAuditEntry(assetInputGroup{AssetIdentityKey: "asset-btc"}, reportmodel.ActivityCalculationInput{
		SourceID:             "eur-buy-1",
		OccurredAt:           request.ActivityDate,
		DisplayLabel:         "BTC",
		SelectedCurrencyCode: "EUR",
	}, evidence, []reportmodel.ConvertedActivityAmount{amount})
	if err != nil {
		t.Fatalf("expected audit entry using input label fallback: %v", err)
	}
	if entry.AssetLabel != "BTC" {
		t.Fatalf("expected input display label fallback, got %#v", entry)
	}

	entry, err = buildConversionAuditEntry(assetInputGroup{AssetIdentityKey: " asset-btc "}, reportmodel.ActivityCalculationInput{
		SourceID:             "eur-buy-1",
		OccurredAt:           request.ActivityDate,
		SelectedCurrencyCode: "EUR",
	}, evidence, []reportmodel.ConvertedActivityAmount{amount})
	if err != nil {
		t.Fatalf("expected audit entry using asset-key fallback: %v", err)
	}
	if entry.AssetLabel != "asset asset-btc" {
		t.Fatalf("expected asset identity fallback, got %#v", entry)
	}

	_, err = buildConversionAuditEntry(assetInputGroup{DisplayLabel: "BTC"}, reportmodel.ActivityCalculationInput{
		SourceID:             "eur-buy-1",
		OccurredAt:           request.ActivityDate,
		SelectedCurrencyCode: "EUR",
	}, evidence, nil)
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not build the conversion audit entry") {
		t.Fatalf("expected wrapped audit-entry validation failure, got %q", calcErr.Error())
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

	var normalized = datesupport.CalendarDate(time.Date(2024, time.May, 21, 22, 3, 4, 0, time.FixedZone("offset", 2*60*60)))
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
		ActivityType:     reportmodel.ActivityTypeBuy,
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

	_, err = replayAssetInput(stubAssetBasisState{
		openQuantityFunc: func() (apd.Decimal, error) { return mustReportDecimal(t, "1"), nil },
	}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "unsupported-activity",
		OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     reportmodel.ActivityType("SWAP"),
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
	}}, 1, 2024)
	calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)
	if !strings.Contains(calcErr.Error(), "unsupported activity type") {
		t.Fatalf("expected apply-basis failure to propagate, got %q", calcErr.Error())
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
		ActivityType:     reportmodel.ActivityTypeBuy,
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

	postApplyCalls = 0
	_, err = replayAssetInput(stubAssetBasisState{
		addAcquisitionFunc: func(basisAcquisitionInput) error { return nil },
		openQuantityFunc: func() (apd.Decimal, error) {
			postApplyCalls++
			if postApplyCalls == 1 {
				return mustReportDecimal(t, "1"), nil
			}
			return apd.Decimal{}, errors.New("after quantity")
		},
		openBasisFunc: func() (apd.Decimal, error) { return mustReportDecimal(t, "10"), nil },
	}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "buy-4",
		OccurredAt:       time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     reportmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       decimalPointer(t, "10"),
		FeeAmount:        decimalPointer(t, "0"),
	}}, 1, 2024)
	calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not determine the asset quantity after applying the activity") {
		t.Fatalf("expected wrapped post-apply quantity failure, got %q", calcErr.Error())
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
			ActivityType:     reportmodel.ActivityTypeBuy,
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
		ActivityType:     reportmodel.ActivityType("SWAP"),
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
			ActivityType:            reportmodel.ActivityTypeSell,
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
		ActivityType:                 reportmodel.ActivityTypeSell,
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
		ActivityType:     reportmodel.ActivityTypeBuy,
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
		ActivityType:     reportmodel.ActivityTypeBuy,
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
		ActivityType:     reportmodel.ActivityTypeBuy,
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
		ActivityType:                 reportmodel.ActivityTypeSell,
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
		ActivityType:     reportmodel.ActivityTypeSell,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
	}})
	requireCalculationError(t, err, reportmodel.CalculationErrorKindActivityInput)

	_, err = applyPricedLiquidation(stubAssetBasisState{}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "sell-invalid-proceeds",
		OccurredAt:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     reportmodel.ActivityTypeSell,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       &invalid,
		FeeAmount:        decimalPointer(t, "1"),
	}})
	requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)

	_, err = applyPricedLiquidation(stubAssetBasisState{disposeFunc: func(basisDisposalInput) (basisDisposalResult, error) {
		return basisDisposalResult{}, errors.New("dispose boom")
	}}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "sell-dispose-fail",
		OccurredAt:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     reportmodel.ActivityTypeSell,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       decimalPointer(t, "12"),
		FeeAmount:        decimalPointer(t, "1"),
	}})
	var calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not allocate basis for the priced liquidation") {
		t.Fatalf("expected priced-liquidation dispose failure, got %q", calcErr.Error())
	}

	_, err = applyPricedLiquidation(stubAssetBasisState{disposeFunc: func(basisDisposalInput) (basisDisposalResult, error) {
		var allocated apd.Decimal
		allocated.Form = apd.Infinite
		return basisDisposalResult{AllocatedBasis: allocated}, nil
	}}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "sell-invalid-gain",
		OccurredAt:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     reportmodel.ActivityTypeSell,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       decimalPointer(t, "12"),
		FeeAmount:        decimalPointer(t, "1"),
	}})
	requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)

	result, err := applyPricedLiquidation(stubAssetBasisState{disposeFunc: func(basisDisposalInput) (basisDisposalResult, error) {
		return basisDisposalResult{
			AllocatedBasis: mustReportDecimal(t, "1"),
			Matches: []reportmodel.BasisMatch{{
				AcquisitionSourceID: "buy-1",
				MatchedQuantity:     mustReportDecimal(t, "2"),
				MatchedBasis:        mustReportDecimal(t, "1"),
			}},
		}, nil
	}}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:             "sell-fragment-success",
		OccurredAt:           time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		SourceYear:           2024,
		ActivityType:         reportmodel.ActivityTypeSell,
		AssetIdentityKey:     "asset-btc",
		DisplayLabel:         "BTC",
		Quantity:             mustReportDecimal(t, "3"),
		GrossValue:           decimalPointer(t, "2"),
		FeeAmount:            decimalPointer(t, "0"),
		SelectedCurrencyCode: "USD",
	}})
	if err != nil {
		t.Fatalf("apply priced liquidation with fragment matches: %v", err)
	}
	if len(result.basisMatches) != 1 || result.basisMatches[0].MatchedProceeds == nil || result.basisMatches[0].MatchedGainOrLoss == nil {
		t.Fatalf("expected fragment-level priced liquidation matches, got %#v", result.basisMatches)
	}
	if result.basisMatches[0].MatchedProceeds.Cmp(decimalPointer(t, "1.3333333333333334")) != 0 {
		t.Fatalf("unexpected matched proceeds: %#v", result.basisMatches[0])
	}
	if result.basisMatches[0].MatchedGainOrLoss.Cmp(decimalPointer(t, "0.3333333333333334")) != 0 {
		t.Fatalf("unexpected matched gain or loss: %#v", result.basisMatches[0])
	}

	_, err = applyPricedLiquidation(stubAssetBasisState{disposeFunc: func(basisDisposalInput) (basisDisposalResult, error) {
		return basisDisposalResult{
			AllocatedBasis: mustReportDecimal(t, "1"),
			Matches: []reportmodel.BasisMatch{{
				AcquisitionSourceID: "buy-invalid-fragment",
				MatchedQuantity:     reportInvalidDecimalForCalculator(),
				MatchedBasis:        mustReportDecimal(t, "1"),
			}},
		}, nil
	}}, scopedActivityInput{Input: reportmodel.ActivityCalculationInput{
		SourceID:         "sell-fragment-fail",
		OccurredAt:       time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
		SourceYear:       2024,
		ActivityType:     reportmodel.ActivityTypeSell,
		AssetIdentityKey: "asset-btc",
		DisplayLabel:     "BTC",
		Quantity:         mustReportDecimal(t, "1"),
		GrossValue:       decimalPointer(t, "2"),
		FeeAmount:        decimalPointer(t, "0"),
	}})
	calcErr = requireCalculationError(t, err, reportmodel.CalculationErrorKindBasisAllocation)
	if !strings.Contains(calcErr.Error(), "could not calculate fragment-level priced liquidation matches") {
		t.Fatalf("expected priced-liquidation fragment-match failure, got %q", calcErr.Error())
	}
}

// TestBuildPricedLiquidationMatchesCoversRemainingBranches verifies the helper
// guardrails and wrapped per-fragment failure branches.
// Authored by: OpenCode
func TestBuildPricedLiquidationMatchesCoversRemainingBranches(t *testing.T) {
	var originalReportDivideRoundHalfUp = reportDivideRoundHalfUp
	defer func() {
		reportDivideRoundHalfUp = originalReportDivideRoundHalfUp
	}()

	var matches, err = buildPricedLiquidationMatches(nil, mustReportDecimal(t, "1"), mustReportDecimal(t, "1"))
	if err != nil || matches != nil {
		t.Fatalf("expected nil fragment matches to short-circuit, got matches=%#v err=%v", matches, err)
	}

	reportDivideRoundHalfUp = func(apd.Decimal, apd.Decimal) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("divide boom")
	}
	_, err = buildPricedLiquidationMatches(
		[]reportmodel.BasisMatch{{AcquisitionSourceID: "buy-divide", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: mustReportDecimal(t, "1")}},
		mustReportDecimal(t, "1"),
		mustReportDecimal(t, "1"),
	)
	if err == nil || !strings.Contains(err.Error(), "calculate proceeds per unit") {
		t.Fatalf("expected wrapped proceeds-per-unit failure, got %v", err)
	}
	reportDivideRoundHalfUp = originalReportDivideRoundHalfUp

	_, err = buildPricedLiquidationMatches(
		[]reportmodel.BasisMatch{{AcquisitionSourceID: "buy-1", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: mustReportDecimal(t, "1")}},
		reportInvalidDecimalForCalculator(),
		mustReportDecimal(t, "1"),
	)
	if err == nil || !strings.Contains(err.Error(), "disposed quantity") {
		t.Fatalf("expected non-finite disposed quantity to fail, got %v", err)
	}

	_, err = buildPricedLiquidationMatches(
		[]reportmodel.BasisMatch{{AcquisitionSourceID: "buy-1", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: mustReportDecimal(t, "1")}},
		mustReportDecimal(t, "0"),
		mustReportDecimal(t, "1"),
	)
	if err == nil || !strings.Contains(err.Error(), "disposed quantity must be greater than zero") {
		t.Fatalf("expected zero disposed quantity to fail, got %v", err)
	}

	_, err = buildPricedLiquidationMatches(
		[]reportmodel.BasisMatch{{AcquisitionSourceID: "buy-1", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: mustReportDecimal(t, "1")}},
		mustReportDecimal(t, "1"),
		reportInvalidDecimalForCalculator(),
	)
	if err == nil || !strings.Contains(err.Error(), "net proceeds") {
		t.Fatalf("expected non-finite net proceeds to fail, got %v", err)
	}

	_, err = buildPricedLiquidationMatches(
		[]reportmodel.BasisMatch{{AcquisitionSourceID: "buy-proceeds", MatchedQuantity: reportInvalidDecimalForCalculator(), MatchedBasis: mustReportDecimal(t, "1")}},
		mustReportDecimal(t, "1"),
		mustReportDecimal(t, "1"),
	)
	if err == nil || !strings.Contains(err.Error(), "calculate matched proceeds") {
		t.Fatalf("expected invalid matched quantity to fail proceeds allocation, got %v", err)
	}

	_, err = buildPricedLiquidationMatches(
		[]reportmodel.BasisMatch{{AcquisitionSourceID: "buy-gain", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: reportInvalidDecimalForCalculator()}},
		mustReportDecimal(t, "1"),
		mustReportDecimal(t, "1"),
	)
	if err == nil || !strings.Contains(err.Error(), "calculate matched gain or loss") {
		t.Fatalf("expected invalid matched basis to fail gain-or-loss allocation, got %v", err)
	}

	matches, err = buildPricedLiquidationMatches(
		[]reportmodel.BasisMatch{{AcquisitionSourceID: "buy-round", MatchedQuantity: mustReportDecimal(t, "1"), MatchedBasis: mustReportDecimal(t, "0")}},
		mustReportDecimal(t, "6"),
		mustReportDecimal(t, "1"),
	)
	if err != nil {
		t.Fatalf("expected rounded proceeds-per-unit success, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one rounded fragment match, got %#v", matches)
	}
	if matches[0].MatchedProceeds == nil {
		t.Fatalf("expected rounded helper matched proceeds, got %#v", matches[0])
	}
	if got := matches[0].MatchedProceeds.Text('f'); got != "0.1666666666666667" {
		t.Fatalf("unexpected rounded helper matched proceeds: got %q want %q", got, "0.1666666666666667")
	}
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
			ActivityType:                 reportmodel.ActivityTypeSell,
			Quantity:                     mustReportDecimal(t, "1"),
			UnitPrice:                    decimalPointer(t, "0"),
			GrossValue:                   decimalPointer(t, "0"),
			FeeAmount:                    decimalPointer(t, "0"),
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
	var zero = mustReportDecimal(t, "0")
	if row.UnitPrice == nil || row.UnitPrice.Cmp(&zero) != 0 || row.GrossValue == nil || row.GrossValue.Cmp(&zero) != 0 || row.FeeAmount == nil || row.FeeAmount.Cmp(&zero) != 0 {
		t.Fatalf("expected zero-priced row to preserve explicit zero-valued fields, got %#v", row)
	}

	row, liquidation, yearlyNet, err = buildInYearArtifacts(
		reportmodel.ActivityCalculationInput{
			SourceID:             "sell-2",
			OccurredAt:           time.Date(2024, time.January, 6, 0, 0, 0, 0, time.UTC),
			SourceYear:           2024,
			ActivityType:         reportmodel.ActivityTypeSell,
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
			basisMatches: []reportmodel.BasisMatch{{
				AcquisitionSourceID: "buy-1",
				MatchedQuantity:     mustReportDecimal(t, "1"),
				MatchedBasis:        mustReportDecimal(t, "7"),
				MatchedProceeds:     decimalPointer(t, "10"),
				MatchedGainOrLoss:   decimalPointer(t, "3"),
			}},
		},
	)
	if err != nil {
		t.Fatalf("build priced in-year artifacts: %v", err)
	}
	if row == nil || liquidation == nil || yearlyNet.Cmp(apd.New(3, 0)) != 0 || row.LiquidationCalculation == nil || row.ActivityCurrency != "USD" {
		t.Fatalf("unexpected priced row artifacts: row=%#v liquidation=%#v yearlyNet=%v", row, liquidation, yearlyNet)
	}
	if len(liquidation.Matches) != 1 || liquidation.Matches[0].MatchedProceeds == nil || liquidation.Matches[0].MatchedGainOrLoss == nil {
		t.Fatalf("expected one liquidation basis match, got %#v", liquidation.Matches)
	}

	_, _, _, err = buildInYearArtifacts(
		reportmodel.ActivityCalculationInput{
			SourceID:             "sell-3",
			OccurredAt:           time.Date(2024, time.January, 7, 0, 0, 0, 0, time.UTC),
			SourceYear:           2024,
			ActivityType:         reportmodel.ActivityTypeSell,
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
			basisMatches: []reportmodel.BasisMatch{{
				AcquisitionSourceID: "buy-1",
				MatchedQuantity:     mustReportDecimal(t, "1"),
				MatchedBasis:        mustReportDecimal(t, "7"),
			}},
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
			ActivityType:         reportmodel.ActivityTypeSell,
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
// seam-propagated constructor failures, and structured error wrappers.
// Authored by: OpenCode
func TestNewAssetBasisStateAndCalculationHelpers(t *testing.T) {
	var originalNewLotMethodState = newLotMethodState
	var originalLotStateTotalOpenQuantity = lotStateTotalOpenQuantity
	defer func() {
		newLotMethodState = originalNewLotMethodState
		lotStateTotalOpenQuantity = originalLotStateTotalOpenQuantity
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

	var recordErr = newRecordCalculationError(reportmodel.CalculationErrorKindActivityInput, syncmodel.ActivityRecord{
		SourceID:    " rec-1 ",
		AssetSymbol: " BTC ",
	}, "record failure", nil)
	var recordCalcErr = requireCalculationError(t, recordErr, reportmodel.CalculationErrorKindActivityInput)
	if recordCalcErr.SourceID() != "rec-1" || recordCalcErr.DisplayLabel() != "BTC" {
		t.Fatalf("expected record error references, got %#v", recordCalcErr)
	}
	var diagnosticRecordErr, ok = recordErr.(diagnosticCalculationError)
	if !ok {
		t.Fatalf("expected diagnostic calculation error, got %T", recordErr)
	}
	var stringTarget string
	if diagnosticRecordErr.As(&stringTarget) {
		t.Fatalf("expected diagnostic calculation error not to match unrelated target")
	}
	var diagnosticErr = diagnosticCalculationError{}
	if context := diagnosticErr.DiagnosticReportContext(); context.FailureStage != "" || context.FailureDetail != "" || len(context.FailureCauseChain) != 0 || len(context.Records) != 0 || context.OffendingActivityRecord != nil {
		t.Fatalf("expected empty diagnostic context for nil calculation error, got %#v", context)
	}
	if passthrough := withPersistedActivityRecord(nil, &syncmodel.ActivityRecord{SourceID: "rec-2"}); passthrough != nil {
		t.Fatalf("expected nil calculation error passthrough, got %#v", passthrough)
	}
	var orphanCalcErr = reportmodel.NewCalculationError(reportmodel.CalculationErrorKindActivityInput, "orphan", "", "", nil)
	if passthrough := withPersistedActivityRecord(orphanCalcErr, nil); passthrough != orphanCalcErr {
		t.Fatalf("expected missing record to preserve calculation error pointer")
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
	var averageWrapper = averageCostBasisState{}
	if _, err = averageWrapper.Dispose(basisDisposalInput{Quantity: mustReportDecimal(t, "1")}); err == nil || !strings.Contains(err.Error(), "average cost state is required") {
		t.Fatalf("expected nil average-cost wrapper state disposal to fail, got %v", err)
	}
	lotStateTotalOpenQuantity = func(*reportbasis.LotMethodState) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("total quantity boom")
	}
	var concreteLotState, concreteErr = reportbasis.NewLotMethodState(reportbasis.LotMethodFIFO)
	if concreteErr != nil {
		t.Fatalf("new concrete FIFO basis state: %v", concreteErr)
	}
	if addErr := concreteLotState.AddAcquisition(reportbasis.LotAcquisition{
		SourceID:           "lot-1",
		AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		DeterministicOrder: 1,
		RemainingQuantity:  mustReportDecimal(t, "1"),
		RemainingBasis:     mustReportDecimal(t, "1"),
	}); addErr != nil {
		t.Fatalf("seed concrete lot basis state: %v", addErr)
	}
	if _, err = (lotBasisState{state: concreteLotState}).Dispose(basisDisposalInput{Quantity: mustReportDecimal(t, "1")}); err == nil || !strings.Contains(err.Error(), "total quantity boom") {
		t.Fatalf("expected injected lot total-quantity failure, got %v", err)
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

// mustReportDecimal parses one exact decimal for report calculation tests.
// Authored by: OpenCode
func mustReportDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse report decimal %q: %v", raw, err)
	}

	return value
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

	var request, err = reportmodel.NewReportRequest(
		year,
		method,
		reportmodel.ReportBaseCurrencyUSD,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
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
		OrderCurrency:    "USD",
		OrderUnitPrice:   decimalPointer(t, "0"),
		OrderGrossValue:  decimalPointer(t, "0"),
		OrderFeeAmount:   decimalPointer(t, "0"),
		Comment:          "manual transfer",
	}
}

// zeroPricedNoLookupAcquisitionRecord returns one same-currency acquisition that
// gives the zero-priced reduction tests open basis without using rate lookup.
// Authored by: OpenCode
func zeroPricedNoLookupAcquisitionRecord(t *testing.T) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         "zero-no-lookup-usd-buy",
		OccurredAt:       "2024-01-02T10:00:00Z",
		ActivityType:     syncmodel.ActivityTypeBuy,
		AssetIdentityKey: "asset-zero-no-lookup",
		AssetSymbol:      "ZNL",
		AssetName:        "Zero No Lookup",
		Quantity:         mustReportDecimal(t, "1"),
		OrderCurrency:    "USD",
		OrderUnitPrice:   decimalPointer(t, "10"),
		OrderGrossValue:  decimalPointer(t, "10"),
		OrderFeeAmount:   decimalPointer(t, "0"),
	}
}

// decimalPointer returns one report-decimal pointer for calculator tests.
// Authored by: OpenCode
func decimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	var value = mustReportDecimal(t, raw)
	return &value
}

// rateLookupRequestKey returns the stable key used by calculator rate-service
// stubs.
// Authored by: OpenCode
func rateLookupRequestKey(request currencyintegration.RateLookupRequest) string {
	return request.SourceCurrency + "|" + request.BaseCurrency + "|" + request.ActivityDate.Format(time.DateOnly)
}

// mustCalculatorRateLookupRequest creates one validated calculator lookup fixture.
// Authored by: OpenCode
func mustCalculatorRateLookupRequest(t *testing.T, sourceCurrency string, baseCurrency string, activityDate time.Time) currencyintegration.RateLookupRequest {
	t.Helper()

	var request, err = currencyintegration.NewRateLookupRequest(sourceCurrency, baseCurrency, activityDate)
	if err != nil {
		t.Fatalf("new calculator rate lookup request: %v", err)
	}

	return request
}

// mustCalculatorRateEvidence creates one validated canonical rate evidence
// fixture for calculator conversion-boundary tests.
// Authored by: OpenCode
func mustCalculatorRateEvidence(
	t *testing.T,
	request currencyintegration.RateLookupRequest,
	rateDate time.Time,
	authority currencyintegration.RateAuthority,
	providerID currencyintegration.ProviderID,
	rateKind string,
	quoteDirection currencyintegration.QuoteDirection,
	rateValue string,
	datasetReference string,
) currencyintegration.ExchangeRateEvidence {
	t.Helper()

	var evidence, err = currencyintegration.NewExchangeRateEvidence(
		request,
		rateDate,
		authority,
		providerID,
		rateKind,
		quoteDirection,
		mustReportDecimal(t, rateValue),
		datasetReference,
	)
	if err != nil {
		t.Fatalf("new calculator rate evidence: %v", err)
	}

	return evidence
}

// assertReportDecimalPointer verifies one optional decimal has the expected
// canonical text value.
// Authored by: OpenCode
func assertReportDecimalPointer(t *testing.T, actual *apd.Decimal, expected string) {
	t.Helper()

	if actual == nil {
		t.Fatalf("expected decimal %s, got nil", expected)
	}
	var expectedDecimal = mustReportDecimal(t, expected)
	if actual.Cmp(&expectedDecimal) != 0 {
		t.Fatalf("unexpected decimal: got %s want %s", actual.Text('f'), expected)
	}
}

// assertReportConvertedAmount verifies one conversion audit amount fixture.
// Authored by: OpenCode
func assertReportConvertedAmount(t *testing.T, actual reportmodel.ConvertedActivityAmount, kind reportmodel.ConvertedAmountKind, original string, converted string) {
	t.Helper()

	if actual.AmountKind != kind {
		t.Fatalf("unexpected converted amount kind: got %s want %s", actual.AmountKind, kind)
	}
	var originalDecimal = mustReportDecimal(t, original)
	if actual.OriginalAmount.Cmp(&originalDecimal) != 0 {
		t.Fatalf("unexpected original amount: got %s want %s", actual.OriginalAmount.Text('f'), original)
	}
	var convertedDecimal = mustReportDecimal(t, converted)
	if actual.ConvertedAmount.Cmp(&convertedDecimal) != 0 {
		t.Fatalf("unexpected converted amount: got %s want %s", actual.ConvertedAmount.Text('f'), converted)
	}
	if actual.ConversionStatus != reportmodel.ConversionStatusConverted || actual.ExchangeRateEvidence == nil {
		t.Fatalf("expected converted status with evidence, got %#v", actual)
	}
}
