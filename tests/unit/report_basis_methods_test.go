// Package unit verifies focused basis-method seams for the report slice.
// Authored by: OpenCode
package unit

import (
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
		assertHIFOTieWinner(t, []reportbasis.LotAcquisition{
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
		}, "older-cost-match")
	})

	t.Run("lower deterministic order wins same day tie", func(t *testing.T) {
		assertHIFOTieWinner(t, []reportbasis.LotAcquisition{
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
		}, "same-day-early-order")
	})
}

// assertHIFOTieWinner adds tied HIFO acquisitions and verifies the selected
// acquisition source ID.
// Authored by: OpenCode
func assertHIFOTieWinner(t *testing.T, acquisitions []reportbasis.LotAcquisition, expectedSourceID string) {
	t.Helper()

	var state, err = reportbasis.NewLotMethodState(reportbasis.LotMethodHIFO)
	if err != nil {
		t.Fatalf("new HIFO lot state: %v", err)
	}
	for _, acquisition := range acquisitions {
		if err = state.AddAcquisition(acquisition); err != nil {
			t.Fatalf("add acquisition %q: %v", acquisition.SourceID, err)
		}
	}

	var result, disposeErr = state.Dispose(mustCalculationDecimal(t, "1"))
	if disposeErr != nil {
		t.Fatalf("dispose tied lots: %v", disposeErr)
	}
	if len(result.Matches) != 1 || result.Matches[0].AcquisitionSourceID != expectedSourceID {
		t.Fatalf("expected HIFO tie to prefer %q, got %#v", expectedSourceID, result.Matches)
	}
}

// TestCalculateRoundsAverageCostWhenDivisionRepeats verifies the shared
// 16-decimal internal precision for average-cost liquidation math.
// Authored by: OpenCode
func TestCalculateRoundsAverageCostWhenDivisionRepeats(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, reportmodel.CostBasisMethodAverageCost)
	var report, err = reportcalculate.Calculate(request, calculationCache(
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
	if err != nil {
		t.Fatalf("calculate rounded average-cost report: %v", err)
	}

	assertCalculationDecimalString(t, summaryEntryByAsset(t, report, "asset-avg-001").NetGainOrLoss, "1.6666666666666667", "rounded average-cost asset net")
	assertCalculationDecimalString(t, report.YearlyNetTotal, "1.6666666666666667", "rounded average-cost yearly net")

	var detail = detailSectionByAsset(t, report, "asset-avg-001")
	assertCalculationDecimalString(t, detail.OpeningQuantity, "3", "rounded average-cost opening quantity")
	assertCalculationDecimalString(t, detail.OpeningCostBasis, "10", "rounded average-cost opening basis")
	assertCalculationDecimalString(t, detail.ClosingQuantity, "2", "rounded average-cost closing quantity")
	assertCalculationDecimalString(t, detail.ClosingCostBasis, "6.6666666666666667", "rounded average-cost closing basis")
	if len(detail.LiquidationSummaries) != 1 {
		t.Fatalf("unexpected rounded average-cost liquidation count: got %d want 1", len(detail.LiquidationSummaries))
	}
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].AllocatedBasis, "3.3333333333333333", "rounded average-cost allocated basis")
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].NetLiquidationProceeds, "5", "rounded average-cost net proceeds")
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].GainOrLoss, "1.6666666666666667", "rounded average-cost gain")
}

// TestAverageCostStateUsesDirectRoundedProportionalAllocation verifies the
// current shared 16-decimal proportional-allocation path for the differentiating
// 2-of-3 disposal shape.
// Authored by: OpenCode
func TestAverageCostStateUsesDirectRoundedProportionalAllocation(t *testing.T) {
	t.Parallel()

	var state = reportbasis.NewAverageCostState()
	if err := state.AddAcquisition(mustCalculationDecimal(t, "3"), mustCalculationDecimal(t, "1")); err != nil {
		t.Fatalf("add average-cost acquisition: %v", err)
	}

	var averageUnitCost, averageErr = state.AverageUnitCost()
	if averageErr != nil {
		t.Fatalf("average unit cost: %v", averageErr)
	}
	assertCalculationDecimalString(t, averageUnitCost, "0.3333333333333333", "average unit cost")

	var disposal, err = state.Dispose(mustCalculationDecimal(t, "2"))
	if err != nil {
		t.Fatalf("dispose average-cost quantity: %v", err)
	}

	assertCalculationDecimalString(t, disposal.AllocatedBasis, "0.6666666666666667", "average-cost allocated basis")
	assertCalculationDecimalString(t, disposal.RemainingBasis, "0.3333333333333333", "average-cost remaining basis")

	var proportionalShortcut = mustCalculationDecimal(t, "0.6666666666666666")
	if disposal.AllocatedBasis.Cmp(&proportionalShortcut) == 0 {
		t.Fatalf("expected direct proportional allocation to stay distinguishable from rounded-unit-cost-then-multiply shortcut")
	}
}

// TestLotMethodStateUsesRoundedProportionalFragmentAllocation verifies the
// current direct proportional allocation result for a partial lot whose unit
// cost repeats.
// Authored by: OpenCode
func TestLotMethodStateUsesRoundedProportionalFragmentAllocation(t *testing.T) {
	t.Parallel()

	var state, err = reportbasis.NewLotMethodState(reportbasis.LotMethodFIFO)
	if err != nil {
		t.Fatalf("new lot method state: %v", err)
	}
	if err = state.AddAcquisition(reportbasis.LotAcquisition{
		SourceID:           "lot-rounding-001",
		AcquiredAt:         time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC),
		DeterministicOrder: 1,
		RemainingQuantity:  mustCalculationDecimal(t, "3"),
		RemainingBasis:     mustCalculationDecimal(t, "1"),
	}); err != nil {
		t.Fatalf("add FIFO acquisition: %v", err)
	}

	var disposal, disposeErr = state.Dispose(mustCalculationDecimal(t, "2"))
	if disposeErr != nil {
		t.Fatalf("dispose FIFO lot quantity: %v", disposeErr)
	}

	assertCalculationDecimalString(t, disposal.AllocatedBasis, "0.6666666666666667", "lot allocated basis")
	if len(disposal.Matches) != 1 {
		t.Fatalf("unexpected lot match count: got %d want 1", len(disposal.Matches))
	}
	assertCalculationDecimalString(t, disposal.Matches[0].MatchedBasis, "0.6666666666666667", "lot matched basis")

	var openLots = state.OpenLots()
	if len(openLots) != 1 {
		t.Fatalf("unexpected open lot count after disposal: got %d want 1", len(openLots))
	}
	assertCalculationDecimalString(t, openLots[0].RemainingBasis, "0.3333333333333333", "lot remaining basis")

	var proportionalShortcut = mustCalculationDecimal(t, "0.6666666666666666")
	if disposal.AllocatedBasis.Cmp(&proportionalShortcut) == 0 {
		t.Fatalf("expected lot allocation to remain distinguishable from rounded-unit-cost-then-multiply shortcut")
	}
}

// TestCalculateScopeLocalHybridNarrowsReliableScopeTimeline verifies that one
// asset with reliable scope data still narrows correctly even when other asset
// timelines make the cache-wide scope summary partial.
// Authored by: OpenCode
func TestCalculateScopeLocalHybridNarrowsReliableScopeTimeline(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, reportmodel.CostBasisMethodScopeLocalHybrid)
	var report, err = reportcalculate.Calculate(request, syncmodel.ProtectedActivityCache{
		ActivityCount:        3,
		AvailableReportYears: []int{2024},
		ScopeReliability:     syncmodel.ScopeReliabilityPartial,
		Activities: []syncmodel.ActivityRecord{
			scopeLocalActivityRecord(t, "avax-buy-beta-2023-001", "2023-01-10T10:00:00Z", syncmodel.ActivityTypeBuy, "asset-avax-001", "AVAX", "Avalanche", "500", scopeLocalReliableWallet("wallet-avax-beta")),
			scopeLocalActivityRecord(t, "avax-buy-alpha-2023-001", "2023-06-10T10:00:00Z", syncmodel.ActivityTypeBuy, "asset-avax-001", "AVAX", "Avalanche", "100", scopeLocalReliableWallet("wallet-avax-alpha")),
			scopeLocalActivityRecord(t, "avax-sell-alpha-2024-001", "2024-08-15T10:00:00Z", syncmodel.ActivityTypeSell, "asset-avax-001", "AVAX", "Avalanche", "250", scopeLocalReliableWallet("wallet-avax-alpha")),
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
		scopeLocalAcquisition(t, "scope-a-buy-001", "scope-a", "2023-01-10T00:00:00Z", 1, "100"),
		scopeLocalAcquisition(t, "scope-a-buy-002", "scope-a", "2023-02-10T00:00:00Z", 2, "300"),
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

	if err = state.AddAcquisition(scopeLocalAcquisition(t, "scope-a-buy-003", "scope-a", "2023-03-10T00:00:00Z", 3, "500")); err != nil {
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

	var finalDisposal, finalErr = state.Dispose("scope-a", mustCalculationDecimal(t, "1"))
	if finalErr != nil {
		t.Fatalf("dispose final fallback quantity: %v", finalErr)
	}
	assertCalculationDecimalString(t, finalDisposal.AllocatedBasis, "350", "final fallback allocated basis")
	if !finalDisposal.ReachedZero {
		t.Fatalf("expected fallback scope to reset once quantity reaches zero")
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
		scopeLocalAcquisition(t, "scope-a-buy-001", "scope-a", "2023-01-10T00:00:00Z", 1, "100"),
		scopeLocalAcquisition(t, "scope-a-buy-002", "scope-a", "2023-02-10T00:00:00Z", 2, "300"),
		scopeLocalAcquisition(t, "scope-b-buy-001", "scope-b", "2023-01-15T00:00:00Z", 1, "900"),
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

	if err = state.AddAcquisition(scopeLocalAcquisition(t, "scope-a-buy-003", "scope-a", "2024-01-10T00:00:00Z", 1, "50")); err != nil {
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

// TestCalculateScopeLocalHybridUsesScopeLocalReliableResolution verifies that a
// reliable scoped liquidation uses its own applicable scope even when another
// open scope for the same asset holds a lower basis.
// Authored by: OpenCode
func TestCalculateScopeLocalHybridUsesScopeLocalReliableResolution(t *testing.T) {
	t.Parallel()

	var request = mustReportRequest(t, reportmodel.CostBasisMethodScopeLocalHybrid)
	var report, err = reportcalculate.Calculate(request, syncmodel.ProtectedActivityCache{
		ActivityCount:        3,
		AvailableReportYears: []int{2024},
		ScopeReliability:     syncmodel.ScopeReliabilityReliable,
		Activities: []syncmodel.ActivityRecord{
			scopeLocalActivityRecord(t, "atom-buy-alpha-2023-001", "2023-01-10T10:00:00Z", syncmodel.ActivityTypeBuy, "asset-atom-001", "ATOM", "Cosmos", "100", scopeLocalReliableWallet("wallet-atom-alpha")),
			scopeLocalActivityRecord(t, "atom-buy-beta-2023-001", "2023-06-10T10:00:00Z", syncmodel.ActivityTypeBuy, "asset-atom-001", "ATOM", "Cosmos", "20", scopeLocalReliableWallet("wallet-atom-beta")),
			scopeLocalActivityRecord(t, "atom-sell-alpha-2024-001", "2024-08-15T10:00:00Z", syncmodel.ActivityTypeSell, "asset-atom-001", "ATOM", "Cosmos", "250", scopeLocalReliableWallet("wallet-atom-alpha")),
		},
	})
	if err != nil {
		t.Fatalf("calculate reliable scope-local report: %v", err)
	}

	assertCalculationDecimalString(t, summaryEntryByAsset(t, report, "asset-atom-001").NetGainOrLoss, "150", "reliable scope-local net gain")
	var detail = detailSectionByAsset(t, report, "asset-atom-001")
	assertCalculationDecimalString(t, detail.ClosingCostBasis, "20", "reliable scope-local closing basis")
	if len(detail.LiquidationSummaries) != 1 {
		t.Fatalf("unexpected reliable scope-local liquidation count: got %d want 1", len(detail.LiquidationSummaries))
	}
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].AllocatedBasis, "100", "reliable scope-local allocated basis")
	assertCalculationDecimalString(t, detail.LiquidationSummaries[0].GainOrLoss, "150", "reliable scope-local gain or loss")
}

// TestScopeLocalHybridFallbackAllocationUsesCurrentProportionalRule verifies the
// differentiating 2-of-3 disposal shape inside one fallback scope.
// Authored by: OpenCode
func TestScopeLocalHybridFallbackAllocationUsesCurrentProportionalRule(t *testing.T) {
	t.Parallel()

	var state = reportbasis.NewScopeLocalHybridState()
	for _, acquisition := range []reportbasis.ScopeLocalHybridAcquisition{
		scopeLocalAcquisition(t, "scope-rounding-001", "scope-a", "2023-01-10T00:00:00Z", 1, "0.2"),
		scopeLocalAcquisition(t, "scope-rounding-002", "scope-a", "2023-02-10T00:00:00Z", 2, "0.3"),
		scopeLocalAcquisition(t, "scope-rounding-003", "scope-a", "2023-03-10T00:00:00Z", 3, "0.5"),
	} {
		if err := state.AddAcquisition(acquisition); err != nil {
			t.Fatalf("add rounding acquisition %q: %v", acquisition.SourceID, err)
		}
	}

	var disposal, err = state.Dispose("scope-a", mustCalculationDecimal(t, "2"))
	if err != nil {
		t.Fatalf("dispose fallback rounding quantity: %v", err)
	}
	assertCalculationDecimalString(t, disposal.AllocatedBasis, "0.6666666666666667", "scope-local fallback allocated basis")
	if disposal.ReachedZero {
		t.Fatalf("expected fallback scope to remain open after disposing 2 of 3 units")
	}

	var totalBasis, totalBasisErr = state.TotalOpenBasis()
	if totalBasisErr != nil {
		t.Fatalf("remaining fallback basis: %v", totalBasisErr)
	}
	assertCalculationDecimalString(t, totalBasis, "0.3333333333333333", "scope-local fallback remaining basis")

	var proportionalShortcut = mustCalculationDecimal(t, "0.6666666666666666")
	if disposal.AllocatedBasis.Cmp(&proportionalShortcut) == 0 {
		t.Fatalf("expected fallback allocation to remain distinguishable from rounded-unit-cost-then-multiply shortcut")
	}
}

// scopeLocalActivityRecord builds one scoped activity fixture for hybrid-method
// calculator tests.
// Authored by: OpenCode
func scopeLocalActivityRecord(t *testing.T, sourceID string, occurredAt string, activityType syncmodel.ActivityType, assetIdentityKey string, assetSymbol string, assetName string, grossValue string, scope *syncmodel.SourceScope) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         sourceID,
		OccurredAt:       occurredAt,
		ActivityType:     activityType,
		AssetIdentityKey: assetIdentityKey,
		AssetSymbol:      assetSymbol,
		AssetName:        assetName,
		Quantity:         mustCalculationDecimal(t, "1"),
		OrderCurrency:    "USD",
		OrderGrossValue:  calculationDecimalPointer(t, grossValue),
		OrderFeeAmount:   calculationDecimalPointer(t, "0"),
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
func scopeLocalAcquisition(t *testing.T, sourceID string, scopeKey string, acquiredAt string, deterministicOrder int, basis string) reportbasis.ScopeLocalHybridAcquisition {
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
		Quantity:           mustCalculationDecimal(t, "1"),
		Basis:              mustCalculationDecimal(t, basis),
	}
}
