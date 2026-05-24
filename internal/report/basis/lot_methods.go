// Package basis defines cost-basis state and allocation rules used by report
// calculation.
// Authored by: OpenCode
package basis

import (
	"fmt"
	"sort"
	"strings"
	"time"

	reportdecimal "github.com/benizzio/ghostfolio-cryptogains/internal/report/decimal"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// LotMethod identifies one lot-selection strategy for exact-lot basis methods.
// Authored by: OpenCode
type LotMethod string

const (
	// LotMethodFIFO consumes the oldest open acquisition first.
	// Authored by: OpenCode
	LotMethodFIFO LotMethod = "fifo"

	// LotMethodLIFO consumes the newest open acquisition first.
	// Authored by: OpenCode
	LotMethodLIFO LotMethod = "lifo"

	// LotMethodHIFO consumes the highest-unit-cost open acquisition first, with
	// older acquisitions and deterministic order as tie-breakers.
	// Authored by: OpenCode
	LotMethodHIFO LotMethod = "hifo"
)

// LotAcquisition stores one open lot tracked by the lot-based basis methods.
// Authored by: OpenCode
type LotAcquisition struct {
	SourceID           string
	AcquiredAt         time.Time
	DeterministicOrder int
	RemainingQuantity  apd.Decimal
	RemainingBasis     apd.Decimal
}

// LotMatch stores one exact lot fragment consumed by a disposal or holding
// reduction.
// Authored by: OpenCode
type LotMatch struct {
	AcquisitionSourceID string
	MatchedQuantity     apd.Decimal
	MatchedBasis        apd.Decimal
}

// LotDisposalResult stores the exact basis fragments removed by one disposal.
// Authored by: OpenCode
type LotDisposalResult struct {
	Matches        []LotMatch
	AllocatedBasis apd.Decimal
}

// LotMethodState stores the open-lot state for FIFO, LIFO, and HIFO basis
// methods.
// Authored by: OpenCode
type LotMethodState struct {
	method LotMethod
	lots   []LotAcquisition
}

// Test seams keep defensive lot wrapper branches directly coverable.
// Authored by: OpenCode
var (
	lotAddDecimal = func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return supportmath.Add(left, right, "left decimal", "right decimal", "add decimals")
	}
	lotSubtractDecimal = func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return supportmath.Subtract(left, right, "left decimal", "right decimal", "subtract decimals")
	}
	lotMultiplyDecimal = func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		return supportmath.Multiply(left, right, "left decimal", "right decimal", "multiply decimals")
	}
)

// NewLotMethodState creates one empty exact-lot basis state for the requested
// lot-selection method.
//
// Example:
//
//	state, err := basis.NewLotMethodState(basis.LotMethodFIFO)
//	if err != nil {
//		panic(err)
//	}
//	_ = state.OpenLotCount()
//
// Authored by: OpenCode
func NewLotMethodState(method LotMethod) (*LotMethodState, error) {
	if err := validateLotMethod(method); err != nil {
		return nil, err
	}

	return &LotMethodState{method: method}, nil
}

// AddAcquisition appends one open acquisition lot after validating its exact
// quantity and basis values.
// Authored by: OpenCode
func (state *LotMethodState) AddAcquisition(acquisition LotAcquisition) error {
	if state == nil {
		return fmt.Errorf("lot method state is required")
	}
	if err := validateLotMethod(state.method); err != nil {
		return err
	}
	if err := validateLotAcquisition(acquisition); err != nil {
		return err
	}

	state.lots = append(state.lots, cloneLotAcquisition(acquisition))
	return nil
}

// Dispose removes one exact quantity from the currently open lots using the
// configured FIFO, LIFO, or HIFO order.
// Authored by: OpenCode
func (state *LotMethodState) Dispose(quantity apd.Decimal) (LotDisposalResult, error) {
	if state == nil {
		return LotDisposalResult{}, fmt.Errorf("lot method state is required")
	}
	if err := supportmath.RequirePositive(quantity, "disposal quantity"); err != nil {
		return LotDisposalResult{}, err
	}

	var remainingQuantity = supportmath.Clone(quantity)
	var orderedIndexes = state.openLotIndexes()
	var matches []LotMatch
	var allocatedBasis = supportmath.Zero()

	for _, index := range orderedIndexes {
		if disposalComplete(remainingQuantity) {
			break
		}

		var disposalMatch LotMatch
		var err error
		remainingQuantity, allocatedBasis, disposalMatch, err = state.disposeFromLot(index, remainingQuantity, allocatedBasis)
		if err != nil {
			return LotDisposalResult{}, err
		}
		matches = append(matches, disposalMatch)
	}

	if remainingQuantity.Sign() != 0 {
		return LotDisposalResult{}, fmt.Errorf("disposal quantity exceeds open lot quantity")
	}

	return LotDisposalResult{Matches: matches, AllocatedBasis: allocatedBasis}, nil
}

// disposalComplete reports whether one disposal has consumed the full requested quantity.
// Authored by: OpenCode
func disposalComplete(remainingQuantity apd.Decimal) bool {
	return remainingQuantity.Sign() == 0
}

// disposeFromLot allocates one lot fragment and mutates the tracked lot state.
// Authored by: OpenCode
func (state *LotMethodState) disposeFromLot(index int, remainingQuantity apd.Decimal, allocatedBasis apd.Decimal) (apd.Decimal, apd.Decimal, LotMatch, error) {
	var currentLot = &state.lots[index]
	var matchedQuantity = supportmath.Minimum(currentLot.RemainingQuantity, remainingQuantity)

	var matchedBasis, err = exactProportionalBasis(currentLot.RemainingBasis, currentLot.RemainingQuantity, matchedQuantity)
	if err != nil {
		return apd.Decimal{}, apd.Decimal{}, LotMatch{}, fmt.Errorf("dispose from lot %q: %w", strings.TrimSpace(currentLot.SourceID), err)
	}

	var nextState disposalLotState
	nextState, err = nextLotDisposalState(*currentLot, matchedQuantity, matchedBasis, remainingQuantity, allocatedBasis)
	if err != nil {
		return apd.Decimal{}, apd.Decimal{}, LotMatch{}, err
	}

	currentLot.RemainingQuantity = nextState.remainingLotQuantity
	currentLot.RemainingBasis = nextState.remainingLotBasis
	return nextState.remainingQuantity, nextState.allocatedBasis, LotMatch{
		AcquisitionSourceID: strings.TrimSpace(currentLot.SourceID),
		MatchedQuantity:     matchedQuantity,
		MatchedBasis:        matchedBasis,
	}, nil
}

// disposalLotState stores one intermediate lot-disposal mutation result.
// Authored by: OpenCode
type disposalLotState struct {
	remainingLotQuantity apd.Decimal
	remainingLotBasis    apd.Decimal
	remainingQuantity    apd.Decimal
	allocatedBasis       apd.Decimal
}

// nextLotDisposalState calculates the next lot and disposal accumulator state.
// Authored by: OpenCode
func nextLotDisposalState(lot LotAcquisition, matchedQuantity apd.Decimal, matchedBasis apd.Decimal, remainingQuantity apd.Decimal, allocatedBasis apd.Decimal) (disposalLotState, error) {
	var remainingLotQuantity, errSubtractQuantity = lotSubtractDecimal(lot.RemainingQuantity, matchedQuantity)
	if errSubtractQuantity != nil {
		return disposalLotState{}, fmt.Errorf("dispose from lot %q quantity: %w", strings.TrimSpace(lot.SourceID), errSubtractQuantity)
	}
	var remainingLotBasis, errSubtractBasis = lotSubtractDecimal(lot.RemainingBasis, matchedBasis)
	if errSubtractBasis != nil {
		return disposalLotState{}, fmt.Errorf("dispose from lot %q basis: %w", strings.TrimSpace(lot.SourceID), errSubtractBasis)
	}
	var nextRemainingQuantity, errSubtractRemaining = lotSubtractDecimal(remainingQuantity, matchedQuantity)
	if errSubtractRemaining != nil {
		return disposalLotState{}, fmt.Errorf("dispose remaining quantity: %w", errSubtractRemaining)
	}
	var nextAllocatedBasis, errAddBasis = lotAddDecimal(allocatedBasis, matchedBasis)
	if errAddBasis != nil {
		return disposalLotState{}, fmt.Errorf("accumulate allocated basis: %w", errAddBasis)
	}

	return disposalLotState{
		remainingLotQuantity: remainingLotQuantity,
		remainingLotBasis:    remainingLotBasis,
		remainingQuantity:    nextRemainingQuantity,
		allocatedBasis:       nextAllocatedBasis,
	}, nil
}

// Method returns the configured lot-selection method.
// Authored by: OpenCode
func (state *LotMethodState) Method() LotMethod {
	if state == nil {
		return ""
	}

	return state.method
}

// OpenLots returns a defensive copy of the currently open acquisition lots.
// Authored by: OpenCode
func (state *LotMethodState) OpenLots() []LotAcquisition {
	if state == nil {
		return nil
	}

	var openLots = make([]LotAcquisition, 0, len(state.lots))
	for _, lot := range state.lots {
		if lot.RemainingQuantity.Sign() == 0 {
			continue
		}
		openLots = append(openLots, cloneLotAcquisition(lot))
	}

	return openLots
}

// OpenLotCount returns the number of lots that still carry open quantity.
// Authored by: OpenCode
func (state *LotMethodState) OpenLotCount() int {
	return len(state.OpenLots())
}

// TotalOpenQuantity returns the exact open quantity tracked across all lots.
// Authored by: OpenCode
func (state *LotMethodState) TotalOpenQuantity() (apd.Decimal, error) {
	var total = supportmath.Zero()

	for _, lot := range state.OpenLots() {
		var nextTotal, err = supportmath.Add(total, lot.RemainingQuantity, "left decimal", "right decimal", "add decimals")
		if err != nil {
			return apd.Decimal{}, err
		}
		total = nextTotal
	}

	return total, nil
}

// TotalOpenBasis returns the exact remaining basis tracked across all open lots.
// Authored by: OpenCode
func (state *LotMethodState) TotalOpenBasis() (apd.Decimal, error) {
	var total = supportmath.Zero()

	for _, lot := range state.OpenLots() {
		var nextTotal, err = supportmath.Add(total, lot.RemainingBasis, "left decimal", "right decimal", "add decimals")
		if err != nil {
			return apd.Decimal{}, err
		}
		total = nextTotal
	}

	return total, nil
}

// openLotIndexes returns the open-lot indexes in the configured disposal order.
// Authored by: OpenCode
func (state *LotMethodState) openLotIndexes() []int {
	var indexes = make([]int, 0, len(state.lots))
	for index, lot := range state.lots {
		if lot.RemainingQuantity.Sign() == 0 {
			continue
		}
		indexes = append(indexes, index)
	}

	sort.SliceStable(indexes, func(left int, right int) bool {
		return lotSortsBefore(state.method, state.lots[indexes[left]], state.lots[indexes[right]])
	})

	return indexes
}

// lotSortsBefore compares two open lots for the configured disposal order.
// Authored by: OpenCode
func lotSortsBefore(method LotMethod, left LotAcquisition, right LotAcquisition) bool {
	switch method {
	case LotMethodFIFO:
		return compareLotChronology(left, right) < 0
	case LotMethodLIFO:
		return compareLotChronology(left, right) > 0
	case LotMethodHIFO:
		return compareHIFOPriority(left, right) < 0
	default:
		return false
	}
}

// compareLotChronology orders two lots by acquisition time, then deterministic
// order, then source ID.
// Authored by: OpenCode
func compareLotChronology(left LotAcquisition, right LotAcquisition) int {
	if left.AcquiredAt.Before(right.AcquiredAt) {
		return -1
	}
	if left.AcquiredAt.After(right.AcquiredAt) {
		return 1
	}
	if left.DeterministicOrder < right.DeterministicOrder {
		return -1
	}
	if left.DeterministicOrder > right.DeterministicOrder {
		return 1
	}
	return strings.Compare(strings.TrimSpace(left.SourceID), strings.TrimSpace(right.SourceID))
}

// compareHIFOPriority orders two open lots by highest unit cost, then older
// acquisition, then deterministic order.
// Authored by: OpenCode
func compareHIFOPriority(left LotAcquisition, right LotAcquisition) int {
	var comparison, err = compareUnitCostsCrossMultiply(left, right)
	if err == nil {
		if comparison > 0 {
			return -1
		}
		if comparison < 0 {
			return 1
		}
	}

	return compareLotChronology(left, right)
}

// compareUnitCostsCrossMultiply compares two lot unit costs using exact cross
// multiplication instead of division.
// Authored by: OpenCode
func compareUnitCostsCrossMultiply(left LotAcquisition, right LotAcquisition) (int, error) {
	var leftCross, err = lotMultiplyDecimal(left.RemainingBasis, right.RemainingQuantity)
	if err != nil {
		return 0, err
	}
	var rightCross apd.Decimal
	rightCross, err = lotMultiplyDecimal(right.RemainingBasis, left.RemainingQuantity)
	if err != nil {
		return 0, err
	}

	return supportmath.Compare(leftCross, rightCross, "left lot unit-cost cross product", "right lot unit-cost cross product")
}

// validateLotMethod rejects unsupported lot-selection methods.
// Authored by: OpenCode
func validateLotMethod(method LotMethod) error {
	switch method {
	case LotMethodFIFO, LotMethodLIFO, LotMethodHIFO:
		return nil
	default:
		return fmt.Errorf("unsupported lot method %q", method)
	}
}

// validateLotAcquisition verifies one open acquisition lot.
// Authored by: OpenCode
func validateLotAcquisition(acquisition LotAcquisition) error {
	if strings.TrimSpace(acquisition.SourceID) == "" {
		return fmt.Errorf("lot acquisition source ID is required")
	}
	if acquisition.AcquiredAt.IsZero() {
		return fmt.Errorf("lot acquisition time is required")
	}
	if err := supportmath.RequirePositive(acquisition.RemainingQuantity, "lot acquisition remaining quantity"); err != nil {
		return err
	}
	if err := supportmath.RequireNonNegative(acquisition.RemainingBasis, "lot acquisition remaining basis"); err != nil {
		return err
	}

	return nil
}

// exactProportionalBasis allocates one matched basis fragment using the shared
// internal report-calculation precision when the proportional division repeats.
// Authored by: OpenCode
func exactProportionalBasis(totalBasis apd.Decimal, totalQuantity apd.Decimal, matchedQuantity apd.Decimal) (apd.Decimal, error) {
	return supportmath.AllocateProportional(
		totalBasis,
		totalQuantity,
		matchedQuantity,
		"total basis",
		"total quantity",
		"matched quantity",
		"allocate basis proportionally",
		lotMultiplyDecimal,
		reportdecimal.DivideRoundHalfUp,
	)
}

// cloneLotAcquisition returns a defensive copy of one lot acquisition.
// Authored by: OpenCode
func cloneLotAcquisition(acquisition LotAcquisition) LotAcquisition {
	return LotAcquisition{
		SourceID:           strings.TrimSpace(acquisition.SourceID),
		AcquiredAt:         acquisition.AcquiredAt,
		DeterministicOrder: acquisition.DeterministicOrder,
		RemainingQuantity:  supportmath.Clone(acquisition.RemainingQuantity),
		RemainingBasis:     supportmath.Clone(acquisition.RemainingBasis),
	}
}
