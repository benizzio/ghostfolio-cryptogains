// Package unit verifies focused basis-method seams for the report slice.
// Authored by: OpenCode
package unit

import (
	"errors"
	"strings"
	"testing"
	"time"

	reportbasis "github.com/benizzio/ghostfolio-cryptogains/internal/report/basis"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// TestHIFOTieBreakingUsesOlderLotThenDeterministicOrder verifies the HIFO tie
// breakers required by the spec when unit costs are equal.
// Authored by: OpenCode
func TestHIFOTieBreakingUsesOlderLotThenDeterministicOrder(t *testing.T) {
	t.Parallel()

	t.Run("older lot wins equal unit cost", func(t *testing.T) {
		var state, err = reportbasis.NewLotMethodState(reportbasis.LotMethodHIFO)
		if err != nil {
			t.Fatalf("new HIFO lot state: %v", err)
		}

		for _, acquisition := range []reportbasis.LotAcquisition{
			{
				SourceID:           "older-cost-match",
				AcquiredAt:         time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC),
				DeterministicOrder: 2,
				RemainingQuantity:  mustCalculationDecimal(t, "1"),
				RemainingBasis:     mustCalculationDecimal(t, "10"),
			},
			{
				SourceID:           "newer-cost-match",
				AcquiredAt:         time.Date(2024, time.February, 10, 0, 0, 0, 0, time.UTC),
				DeterministicOrder: 1,
				RemainingQuantity:  mustCalculationDecimal(t, "1"),
				RemainingBasis:     mustCalculationDecimal(t, "10"),
			},
		} {
			if err = state.AddAcquisition(acquisition); err != nil {
				t.Fatalf("add acquisition %q: %v", acquisition.SourceID, err)
			}
		}

		var result, disposeErr = state.Dispose(mustCalculationDecimal(t, "1"))
		if disposeErr != nil {
			t.Fatalf("dispose tied older/newer lots: %v", disposeErr)
		}
		if len(result.Matches) != 1 || result.Matches[0].AcquisitionSourceID != "older-cost-match" {
			t.Fatalf("expected HIFO tie to prefer older lot, got %#v", result.Matches)
		}
	})

	t.Run("lower deterministic order wins same day tie", func(t *testing.T) {
		var state, err = reportbasis.NewLotMethodState(reportbasis.LotMethodHIFO)
		if err != nil {
			t.Fatalf("new HIFO lot state: %v", err)
		}

		for _, acquisition := range []reportbasis.LotAcquisition{
			{
				SourceID:           "same-day-late-order",
				AcquiredAt:         time.Date(2024, time.March, 10, 0, 0, 0, 0, time.UTC),
				DeterministicOrder: 9,
				RemainingQuantity:  mustCalculationDecimal(t, "1"),
				RemainingBasis:     mustCalculationDecimal(t, "10"),
			},
			{
				SourceID:           "same-day-early-order",
				AcquiredAt:         time.Date(2024, time.March, 10, 0, 0, 0, 0, time.UTC),
				DeterministicOrder: 3,
				RemainingQuantity:  mustCalculationDecimal(t, "1"),
				RemainingBasis:     mustCalculationDecimal(t, "10"),
			},
		} {
			if err = state.AddAcquisition(acquisition); err != nil {
				t.Fatalf("add acquisition %q: %v", acquisition.SourceID, err)
			}
		}

		var result, disposeErr = state.Dispose(mustCalculationDecimal(t, "1"))
		if disposeErr != nil {
			t.Fatalf("dispose same-day tied lots: %v", disposeErr)
		}
		if len(result.Matches) != 1 || result.Matches[0].AcquisitionSourceID != "same-day-early-order" {
			t.Fatalf("expected same-day HIFO tie to prefer lower deterministic order, got %#v", result.Matches)
		}
	})
}

// TestCalculateFailsWhenAverageCostRequiresNonTerminatingDivision verifies the
// no-rounding exact-division rule for average cost basis.
// Authored by: OpenCode
func TestCalculateFailsWhenAverageCostRequiresNonTerminatingDivision(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodAverageCost)
	_, err := reportcalculate.Calculate(request, calculationCache(
		2024,
		calculationActivity(t, calculationActivityInput{
			SourceID:         "avg-buy-2023-001",
			OccurredAt:       "2023-01-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-avg-001",
			AssetSymbol:      "AVG",
			AssetName:        "Average Asset",
			Quantity:         "3",
			OrderCurrency:    "USD",
			OrderGrossValue:  "10",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "3.333333333333333333333333333333333",
		}),
		calculationActivity(t, calculationActivityInput{
			SourceID:         "avg-sell-2024-001",
			OccurredAt:       "2024-02-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-avg-001",
			AssetSymbol:      "AVG",
			AssetName:        "Average Asset",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "5",
			OrderFeeAmount:   "0",
			OrderUnitPrice:   "5",
		}),
	))
	if err == nil {
		t.Fatalf("expected exact-division failure")
	}

	var calculationError *reportmodel.CalculationError
	if !errors.As(err, &calculationError) {
		t.Fatalf("expected structured calculation error, got %T", err)
	}
	if calculationError.Kind() != reportmodel.CalculationErrorKindBasisAllocation {
		t.Fatalf("unexpected calculation error kind: got %q want %q", calculationError.Kind(), reportmodel.CalculationErrorKindBasisAllocation)
	}
	if calculationError.SourceID() != "avg-sell-2024-001" || calculationError.DisplayLabel() != "AVG" {
		t.Fatalf("expected offending activity references, got source=%q label=%q", calculationError.SourceID(), calculationError.DisplayLabel())
	}
	if !strings.Contains(calculationError.Error(), "could not allocate basis for the priced liquidation") {
		t.Fatalf("expected wrapped exact-division failure detail, got %v", calculationError)
	}
	if calculationError.Unwrap() == nil || !strings.Contains(calculationError.Unwrap().Error(), "allocate basis exactly") {
		t.Fatalf("expected underlying exact-division failure, got %v", calculationError.Unwrap())
	}
}

// TestCalculateScopeLocalHybridNarrowsReliableScopeTimeline verifies that one
// asset with reliable scope data still narrows correctly even when other asset
// timelines make the cache-wide scope summary partial.
// Authored by: OpenCode
func TestCalculateScopeLocalHybridNarrowsReliableScopeTimeline(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, 2024, reportmodel.CostBasisMethodScopeLocalHybrid)
	var report, err = reportcalculate.Calculate(request, syncmodel.ProtectedActivityCache{
		ActivityCount:        3,
		AvailableReportYears: []int{2024},
		ScopeReliability:     syncmodel.ScopeReliabilityPartial,
		Activities: []syncmodel.ActivityRecord{
			scopeLocalActivityRecord(t, "avax-buy-beta-2023-001", "2023-01-10T10:00:00Z", syncmodel.ActivityTypeBuy, "asset-avax-001", "AVAX", "Avalanche", "1", "500", "0", scopeLocalReliableWallet("wallet-avax-beta")),
			scopeLocalActivityRecord(t, "avax-buy-alpha-2023-001", "2023-06-10T10:00:00Z", syncmodel.ActivityTypeBuy, "asset-avax-001", "AVAX", "Avalanche", "1", "100", "0", scopeLocalReliableWallet("wallet-avax-alpha")),
			scopeLocalActivityRecord(t, "avax-sell-alpha-2024-001", "2024-08-15T10:00:00Z", syncmodel.ActivityTypeSell, "asset-avax-001", "AVAX", "Avalanche", "1", "250", "0", scopeLocalReliableWallet("wallet-avax-alpha")),
		},
	})
	if err != nil {
		t.Fatalf("calculate hybrid report: %v", err)
	}

	assertCalculationDecimalString(t, summaryEntryByAsset(t, report, "asset-avax-001").NetGainOrLoss, "150", "hybrid AVAX net gain")
	assertCalculationDecimalString(t, report.YearlyNetTotal, "150", "hybrid yearly net total")

	var reference = referenceEntryByAsset(t, report, "asset-avax-001")
	if reference.FullLiquidationCountThroughYearEnd != 1 {
		t.Fatalf("unexpected hybrid liquidation count: got %d want 1", reference.FullLiquidationCountThroughYearEnd)
	}
	if reference.MainSectionStatus != reportmodel.ReferenceSectionStatusIncludedInMainSections {
		t.Fatalf("unexpected hybrid reference status: got %q want %q", reference.MainSectionStatus, reportmodel.ReferenceSectionStatusIncludedInMainSections)
	}

	var detail = detailSectionByAsset(t, report, "asset-avax-001")
	assertCalculationDecimalString(t, detail.ClosingCostBasis, "500", "hybrid AVAX closing basis")
	if len(detail.LiquidationSummaries) != 1 {
		t.Fatalf("unexpected hybrid liquidation summary count: got %d want 1", len(detail.LiquidationSummaries))
	}
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].AllocatedBasis, "100", "hybrid AVAX allocated basis")
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].GainOrLoss, "150", "hybrid AVAX gain or loss")
}

// TestScopeLocalHybridFallbackCarriesForwardUntilZero verifies fallback
// activation on the first non-defensible disposal and continued average-cost
// valuation until that scope reaches zero.
// Authored by: OpenCode
func TestScopeLocalHybridFallbackCarriesForwardUntilZero(t *testing.T) {
	t.Parallel()

	var state = reportbasis.NewScopeLocalHybridState()
	for _, acquisition := range []reportbasis.ScopeLocalHybridAcquisition{
		scopeLocalAcquisition(t, "scope-a-buy-001", "scope-a", "2023-01-10T00:00:00Z", 1, "1", "100"),
		scopeLocalAcquisition(t, "scope-a-buy-002", "scope-a", "2023-02-10T00:00:00Z", 2, "1", "300"),
	} {
		if err := state.AddAcquisition(acquisition); err != nil {
			t.Fatalf("add scope-local acquisition %q: %v", acquisition.SourceID, err)
		}
	}

	var firstDisposal, err = state.Dispose("scope-a", mustCalculationDecimal(t, "1"))
	if err != nil {
		t.Fatalf("dispose after fallback activation: %v", err)
	}
	assertCalculationDecimalString(t, firstDisposal.AllocatedBasis, "200", "first fallback allocated basis")
	if firstDisposal.ReachedZero {
		t.Fatalf("expected scope to remain open after first fallback disposal")
	}

	if err = state.AddAcquisition(scopeLocalAcquisition(t, "scope-a-buy-003", "scope-a", "2023-03-10T00:00:00Z", 3, "1", "500")); err != nil {
		t.Fatalf("add acquisition while fallback is active: %v", err)
	}

	var secondDisposal, secondErr = state.Dispose("scope-a", mustCalculationDecimal(t, "1"))
	if secondErr != nil {
		t.Fatalf("dispose while fallback remains active: %v", secondErr)
	}
	assertCalculationDecimalString(t, secondDisposal.AllocatedBasis, "350", "second fallback allocated basis")
	if secondDisposal.ReachedZero {
		t.Fatalf("expected scope to remain open until the last unit is disposed")
	}
}

// TestScopeLocalHybridResetsAfterZeroAndKeepsScopesIndependent verifies that one
// scope's fallback lifecycle does not affect another scope and that same-scope
// reacquisition after zero starts a fresh exact-matching state.
// Authored by: OpenCode
func TestScopeLocalHybridResetsAfterZeroAndKeepsScopesIndependent(t *testing.T) {
	t.Parallel()

	var state = reportbasis.NewScopeLocalHybridState()
	for _, acquisition := range []reportbasis.ScopeLocalHybridAcquisition{
		scopeLocalAcquisition(t, "scope-a-buy-001", "scope-a", "2023-01-10T00:00:00Z", 1, "1", "100"),
		scopeLocalAcquisition(t, "scope-a-buy-002", "scope-a", "2023-02-10T00:00:00Z", 2, "1", "300"),
		scopeLocalAcquisition(t, "scope-b-buy-001", "scope-b", "2023-01-15T00:00:00Z", 1, "1", "900"),
	} {
		if err := state.AddAcquisition(acquisition); err != nil {
			t.Fatalf("add scope-local acquisition %q: %v", acquisition.SourceID, err)
		}
	}

	var scopeADisposal, err = state.Dispose("scope-a", mustCalculationDecimal(t, "1"))
	if err != nil {
		t.Fatalf("dispose scope-a first time: %v", err)
	}
	assertCalculationDecimalString(t, scopeADisposal.AllocatedBasis, "200", "scope-a first allocated basis")
	if scopeADisposal.ReachedZero {
		t.Fatalf("expected scope-a to remain open after first disposal")
	}

	var scopeBDisposal, scopeBErr = state.Dispose("scope-b", mustCalculationDecimal(t, "1"))
	if scopeBErr != nil {
		t.Fatalf("dispose independent scope-b: %v", scopeBErr)
	}
	assertCalculationDecimalString(t, scopeBDisposal.AllocatedBasis, "900", "scope-b allocated basis")
	if !scopeBDisposal.ReachedZero {
		t.Fatalf("expected scope-b to reach zero independently")
	}

	var scopeAFinalDisposal, finalErr = state.Dispose("scope-a", mustCalculationDecimal(t, "1"))
	if finalErr != nil {
		t.Fatalf("dispose scope-a to zero: %v", finalErr)
	}
	assertCalculationDecimalString(t, scopeAFinalDisposal.AllocatedBasis, "200", "scope-a final fallback basis")
	if !scopeAFinalDisposal.ReachedZero {
		t.Fatalf("expected scope-a to reach zero on second disposal")
	}

	if err = state.AddAcquisition(scopeLocalAcquisition(t, "scope-a-buy-003", "scope-a", "2024-01-10T00:00:00Z", 1, "1", "50")); err != nil {
		t.Fatalf("reacquire in scope-a after zero: %v", err)
	}

	var resetDisposal, resetErr = state.Dispose("scope-a", mustCalculationDecimal(t, "1"))
	if resetErr != nil {
		t.Fatalf("dispose reacquired scope-a position: %v", resetErr)
	}
	assertCalculationDecimalString(t, resetDisposal.AllocatedBasis, "50", "scope-a reset exact-match basis")
	if !resetDisposal.ReachedZero {
		t.Fatalf("expected reacquired scope-a position to reach zero")
	}
}

// scopeLocalActivityRecord builds one scoped activity fixture for hybrid-method
// calculator tests.
// Authored by: OpenCode
func scopeLocalActivityRecord(t *testing.T, sourceID string, occurredAt string, activityType syncmodel.ActivityType, assetIdentityKey string, assetSymbol string, assetName string, quantity string, grossValue string, feeAmount string, scope *syncmodel.SourceScope) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         sourceID,
		OccurredAt:       occurredAt,
		ActivityType:     activityType,
		AssetIdentityKey: assetIdentityKey,
		AssetSymbol:      assetSymbol,
		AssetName:        assetName,
		Quantity:         mustCalculationDecimal(t, quantity),
		OrderCurrency:    "USD",
		OrderGrossValue:  calculationDecimalPointer(t, grossValue),
		OrderFeeAmount:   calculationDecimalPointer(t, feeAmount),
		SourceScope:      scope,
	}
}

// scopeLocalReliableWallet returns one reliable wallet scope fixture for the
// hybrid-method tests.
// Authored by: OpenCode
func scopeLocalReliableWallet(id string) *syncmodel.SourceScope {
	return &syncmodel.SourceScope{
		ID:          id,
		Kind:        syncmodel.SourceScopeKindWallet,
		Reliability: syncmodel.ScopeReliabilityReliable,
	}
}

// scopeLocalAcquisition builds one direct scope-local hybrid acquisition test
// input.
// Authored by: OpenCode
func scopeLocalAcquisition(t *testing.T, sourceID string, scopeKey string, acquiredAt string, deterministicOrder int, quantity string, basis string) reportbasis.ScopeLocalHybridAcquisition {
	t.Helper()

	var parsedTime, err = time.Parse(time.RFC3339, acquiredAt)
	if err != nil {
		t.Fatalf("parse scope-local acquisition time %q: %v", acquiredAt, err)
	}

	return reportbasis.ScopeLocalHybridAcquisition{
		SourceID:           sourceID,
		ScopeKey:           scopeKey,
		AcquiredAt:         parsedTime,
		DeterministicOrder: deterministicOrder,
		Quantity:           mustCalculationDecimal(t, quantity),
		Basis:              mustCalculationDecimal(t, basis),
	}
}
