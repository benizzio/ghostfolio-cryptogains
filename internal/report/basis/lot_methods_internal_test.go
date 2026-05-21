// Package basis verifies package-local lot-method fallback and guardrail paths.
// Authored by: OpenCode
package basis

import (
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/apd/v3"
)

// TestLotMethodFallbackAndNilPaths verifies unsupported-method handling, nil
// receiver behavior, and helper fallbacks.
// Authored by: OpenCode
func TestLotMethodFallbackAndNilPaths(t *testing.T) {
	if _, err := NewLotMethodState(LotMethod("bad")); err == nil {
		t.Fatalf("expected unsupported lot method to fail")
	}

	var nilState *LotMethodState
	if err := nilState.AddAcquisition(validLotAcquisition()); err == nil {
		t.Fatalf("expected nil lot-method acquisition to fail")
	}
	if _, err := nilState.Dispose(decimalFromInt(1)); err == nil {
		t.Fatalf("expected nil lot-method disposal to fail")
	}
	if nilState.Method() != "" {
		t.Fatalf("expected nil lot-method state to return empty method")
	}
	if nilState.OpenLots() != nil {
		t.Fatalf("expected nil lot-method state to return nil open lots")
	}
	if nilState.OpenLotCount() != 0 {
		t.Fatalf("expected nil lot-method state to return zero open-lot count")
	}

	var openQuantity, err = nilState.TotalOpenQuantity()
	if err != nil {
		t.Fatalf("nil lot-method total quantity: %v", err)
	}
	if openQuantity.Sign() != 0 {
		t.Fatalf("expected nil lot-method total quantity to be zero")
	}

	var openBasis apd.Decimal
	openBasis, err = nilState.TotalOpenBasis()
	if err != nil {
		t.Fatalf("nil lot-method total basis: %v", err)
	}
	if openBasis.Sign() != 0 {
		t.Fatalf("expected nil lot-method total basis to be zero")
	}

	var invalidState = &LotMethodState{method: LotMethod("bad")}
	if err = invalidState.AddAcquisition(validLotAcquisition()); err == nil {
		t.Fatalf("expected add acquisition with unsupported method to fail")
	}

	if lotSortsBefore(LotMethod("bad"), validLotAcquisition(), validLotAcquisition()) {
		t.Fatalf("expected unsupported lot sort order to fall back to false")
	}

	var earlierTieBreak = validLotAcquisition()
	var laterTieBreak = validLotAcquisition()
	laterTieBreak.SourceID = "lot-002"
	laterTieBreak.DeterministicOrder = 2
	if compareLotChronology(laterTieBreak, earlierTieBreak) <= 0 {
		t.Fatalf("expected greater deterministic order to sort after the earlier tie-break lot")
	}

	var exactBasis, exactErr = exactProportionalBasis(decimalFromInt(10), decimalFromInt(2), decimalFromInt(2))
	if exactErr != nil {
		t.Fatalf("exact proportional basis for full match: %v", exactErr)
	}
	if exactBasis.Cmp(apd.New(10, 0)) != 0 {
		t.Fatalf("expected full-match proportional basis to return the original basis, got %v", exactBasis)
	}

	_, err = exactProportionalBasis(decimalFromInt(10), decimalFromInt(2), decimalFromInt(3))
	if err == nil || !strings.Contains(err.Error(), "matched quantity exceeds total quantity") {
		t.Fatalf("expected oversized proportional match to fail, got %v", err)
	}
}

// TestLotMethodValidationAndHelperPaths verifies concise helper and validation
// branches for package-local lot-method logic.
// Authored by: OpenCode
func TestLotMethodValidationAndHelperPaths(t *testing.T) {
	var acquisition = validLotAcquisition()

	acquisition.SourceID = "   "
	if err := validateLotAcquisition(acquisition); err == nil || !strings.Contains(err.Error(), "source ID is required") {
		t.Fatalf("expected blank source ID to fail, got %v", err)
	}

	acquisition = validLotAcquisition()
	acquisition.AcquiredAt = time.Time{}
	if err := validateLotAcquisition(acquisition); err == nil || !strings.Contains(err.Error(), "time is required") {
		t.Fatalf("expected zero acquisition time to fail, got %v", err)
	}

	acquisition = validLotAcquisition()
	acquisition.RemainingQuantity = decimalFromInt(0)
	if err := validateLotAcquisition(acquisition); err == nil || !strings.Contains(err.Error(), "remaining quantity must be greater than zero") {
		t.Fatalf("expected zero remaining quantity to fail, got %v", err)
	}

	acquisition = validLotAcquisition()
	acquisition.RemainingBasis = decimalFromInt(-1)
	if err := validateLotAcquisition(acquisition); err == nil || !strings.Contains(err.Error(), "remaining basis must not be negative") {
		t.Fatalf("expected negative remaining basis to fail, got %v", err)
	}

	if _, err := exactProportionalBasis(decimalFromInt(1), decimalFromInt(3), decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "allocate basis exactly") {
		t.Fatalf("expected non-terminating proportional basis allocation to fail, got %v", err)
	}

	var left = validLotAcquisition()
	left.SourceID = " z-lot "
	var right = validLotAcquisition()
	right.SourceID = "a-lot"
	if compareLotChronology(left, right) <= 0 {
		t.Fatalf("expected chronology tie-break to compare trimmed source IDs")
	}

	left = validLotAcquisition()
	left.RemainingBasis = decimalFromInt(12)
	var comparison, err = compareUnitCostsCrossMultiply(left, validLotAcquisition())
	if err != nil {
		t.Fatalf("compare unit costs: %v", err)
	}
	if comparison <= 0 {
		t.Fatalf("expected greater unit cost to compare higher, got %d", comparison)
	}

	var minimum = minimumDecimal(decimalFromInt(2), decimalFromInt(1))
	if minimum.Cmp(apd.New(1, 0)) != 0 {
		t.Fatalf("expected minimum decimal helper to return the smaller value, got %v", minimum)
	}

	var invalid apd.Decimal
	invalid.Form = apd.NaNSignaling
	if _, err = exactProportionalBasis(decimalFromInt(1), invalid, decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "total quantity") {
		t.Fatalf("expected invalid total quantity to fail, got %v", err)
	}
	if _, err = exactProportionalBasis(decimalFromInt(1), decimalFromInt(1), invalid); err == nil || !strings.Contains(err.Error(), "matched quantity") {
		t.Fatalf("expected invalid matched quantity to fail, got %v", err)
	}
	if _, err = addDecimal(decimalFromInt(1), invalid); err == nil || !strings.Contains(err.Error(), "add decimals") {
		t.Fatalf("expected invalid add decimal to fail, got %v", err)
	}
	if _, err = subtractDecimal(decimalFromInt(1), invalid); err == nil || !strings.Contains(err.Error(), "subtract decimals") {
		t.Fatalf("expected invalid subtract decimal to fail, got %v", err)
	}
	if _, err = multiplyDecimal(decimalFromInt(1), invalid); err == nil || !strings.Contains(err.Error(), "multiply decimals") {
		t.Fatalf("expected invalid multiply decimal to fail, got %v", err)
	}
	if err = validateNonNegativeDecimal(invalid, "non-negative"); err == nil || !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("expected non-finite non-negative decimal to fail, got %v", err)
	}
}

// TestLotMethodOperationalBranches verifies disposal ordering, totals, and HIFO
// tie-break behavior on concrete lot states.
// Authored by: OpenCode
func TestLotMethodOperationalBranches(t *testing.T) {
	t.Run("reports configured method and aggregate totals", func(t *testing.T) {
		var state, err = NewLotMethodState(LotMethodFIFO)
		if err != nil {
			t.Fatalf("new FIFO lot method state: %v", err)
		}
		if state.Method() != LotMethodFIFO {
			t.Fatalf("unexpected configured lot method: %q", state.Method())
		}
		for _, acquisition := range []LotAcquisition{
			{
				SourceID:           "lot-001",
				AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				DeterministicOrder: 1,
				RemainingQuantity:  decimalFromInt(1),
				RemainingBasis:     decimalFromInt(10),
			},
			{
				SourceID:           "lot-002",
				AcquiredAt:         time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				DeterministicOrder: 2,
				RemainingQuantity:  decimalFromInt(2),
				RemainingBasis:     decimalFromInt(20),
			},
		} {
			if err = state.AddAcquisition(acquisition); err != nil {
				t.Fatalf("add acquisition %q: %v", acquisition.SourceID, err)
			}
		}

		var totalQuantity, quantityErr = state.TotalOpenQuantity()
		if quantityErr != nil {
			t.Fatalf("total open quantity: %v", quantityErr)
		}
		var totalBasis, basisErr = state.TotalOpenBasis()
		if basisErr != nil {
			t.Fatalf("total open basis: %v", basisErr)
		}
		if totalQuantity.Cmp(apd.New(3, 0)) != 0 || totalBasis.Cmp(apd.New(30, 0)) != 0 {
			t.Fatalf("unexpected lot totals: quantity=%v basis=%v", totalQuantity, totalBasis)
		}

		var disposal, disposeErr = state.Dispose(decimalFromInt(2))
		if disposeErr != nil {
			t.Fatalf("dispose FIFO lots: %v", disposeErr)
		}
		if len(disposal.Matches) != 2 || disposal.Matches[0].AcquisitionSourceID != "lot-001" || disposal.Matches[1].AcquisitionSourceID != "lot-002" || disposal.AllocatedBasis.Cmp(apd.New(20, 0)) != 0 {
			t.Fatalf("unexpected FIFO disposal result: %#v", disposal)
		}

		if _, disposeErr = state.Dispose(decimalFromInt(0)); disposeErr == nil || !strings.Contains(disposeErr.Error(), "disposal quantity") {
			t.Fatalf("expected zero disposal quantity to fail, got %v", disposeErr)
		}

		if _, disposeErr = state.Dispose(decimalFromInt(2)); disposeErr == nil || !strings.Contains(disposeErr.Error(), "exceeds open lot quantity") {
			t.Fatalf("expected oversized disposal after depletion to fail, got %v", disposeErr)
		}
	})

	t.Run("orders HIFO ties by older chronology", func(t *testing.T) {
		var earlier = LotAcquisition{
			SourceID:           "earlier",
			AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 1,
			RemainingQuantity:  decimalFromInt(1),
			RemainingBasis:     decimalFromInt(10),
		}
		var later = LotAcquisition{
			SourceID:           "later",
			AcquiredAt:         time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 2,
			RemainingQuantity:  decimalFromInt(1),
			RemainingBasis:     decimalFromInt(10),
		}

		if comparison := compareHIFOPriority(earlier, later); comparison >= 0 {
			t.Fatalf("expected older equal-cost lot to sort before later lot, got %d", comparison)
		}
		if !lotSortsBefore(LotMethodFIFO, earlier, later) {
			t.Fatalf("expected FIFO to sort older lot first")
		}
		if !lotSortsBefore(LotMethodLIFO, later, earlier) {
			t.Fatalf("expected LIFO to sort newer lot first")
		}
		if !lotSortsBefore(LotMethodHIFO, earlier, later) {
			t.Fatalf("expected HIFO tie to fall back to older chronology")
		}
	})

	t.Run("corrupted lot state surfaces defensive helper failures", func(t *testing.T) {
		var invalid apd.Decimal
		invalid.Form = apd.NaNSignaling

		var corruptedTotals = &LotMethodState{method: LotMethodFIFO, lots: []LotAcquisition{{
			SourceID:           "lot-invalid",
			AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 1,
			RemainingQuantity:  decimalFromInt(1),
			RemainingBasis:     invalid,
		}}}
		if _, err := corruptedTotals.TotalOpenBasis(); err == nil {
			t.Fatalf("expected corrupted total open basis to fail")
		}

		corruptedTotals = &LotMethodState{method: LotMethodFIFO, lots: []LotAcquisition{{
			SourceID:           "lot-invalid",
			AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 1,
			RemainingQuantity:  invalid,
			RemainingBasis:     decimalFromInt(1),
		}}}
		if _, err := corruptedTotals.TotalOpenQuantity(); err == nil {
			t.Fatalf("expected corrupted total open quantity to fail")
		}

		var corruptedDispose = &LotMethodState{method: LotMethodFIFO, lots: []LotAcquisition{{
			SourceID:           "lot-invalid",
			AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 1,
			RemainingQuantity:  decimalFromInt(1),
			RemainingBasis:     invalid,
		}}}
		if _, err := corruptedDispose.Dispose(decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "dispose from lot") {
			t.Fatalf("expected corrupted disposal basis allocation to fail, got %v", err)
		}

		var corruptedIndexes = &LotMethodState{method: LotMethodFIFO, lots: []LotAcquisition{{
			SourceID:           "lot-zero",
			AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 1,
			RemainingQuantity:  decimalFromInt(0),
			RemainingBasis:     decimalFromInt(1),
		}}}
		if indexes := corruptedIndexes.openLotIndexes(); len(indexes) != 0 {
			t.Fatalf("expected zero-quantity lots to be excluded from open indexes, got %#v", indexes)
		}

		var comparison, err = compareUnitCostsCrossMultiply(LotAcquisition{RemainingQuantity: decimalFromInt(1), RemainingBasis: invalid}, validLotAcquisition())
		if err == nil || comparison != 0 {
			t.Fatalf("expected corrupted unit-cost comparison to fail, got comparison=%d err=%v", comparison, err)
		}
	})
}

// validLotAcquisition returns one reusable finite lot fixture for helper tests.
// Authored by: OpenCode
func validLotAcquisition() LotAcquisition {
	return LotAcquisition{
		SourceID:           "lot-001",
		AcquiredAt:         time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
		DeterministicOrder: 1,
		RemainingQuantity:  decimalFromInt(1),
		RemainingBasis:     decimalFromInt(10),
	}
}
