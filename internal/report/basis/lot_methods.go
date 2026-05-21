// Package basis defines cost-basis state and allocation rules used by report
// calculation.
// Authored by: OpenCode
package basis

import (
	"fmt"
	"sort"
	"strings"
	"time"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
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
	if err := validatePositiveDecimal(quantity, "disposal quantity"); err != nil {
		return LotDisposalResult{}, err
	}

	var remainingQuantity = cloneDecimal(quantity)
	var orderedIndexes = state.openLotIndexes()
	var matches []LotMatch
	var allocatedBasis = zeroDecimal()

	for _, index := range orderedIndexes {
		if remainingQuantity.Sign() == 0 {
			break
		}

		var currentLot = &state.lots[index]
		var matchedQuantity = minimumDecimal(currentLot.RemainingQuantity, remainingQuantity)
		if matchedQuantity.Sign() == 0 {
			continue
		}

		var matchedBasis, err = exactProportionalBasis(currentLot.RemainingBasis, currentLot.RemainingQuantity, matchedQuantity)
		if err != nil {
			return LotDisposalResult{}, fmt.Errorf("dispose from lot %q: %w", strings.TrimSpace(currentLot.SourceID), err)
		}

		var remainingLotQuantity, errSubtractQuantity = subtractDecimal(currentLot.RemainingQuantity, matchedQuantity)
		if errSubtractQuantity != nil {
			return LotDisposalResult{}, fmt.Errorf("dispose from lot %q quantity: %w", strings.TrimSpace(currentLot.SourceID), errSubtractQuantity)
		}
		var remainingLotBasis, errSubtractBasis = subtractDecimal(currentLot.RemainingBasis, matchedBasis)
		if errSubtractBasis != nil {
			return LotDisposalResult{}, fmt.Errorf("dispose from lot %q basis: %w", strings.TrimSpace(currentLot.SourceID), errSubtractBasis)
		}
		var nextRemainingQuantity, errSubtractRemaining = subtractDecimal(remainingQuantity, matchedQuantity)
		if errSubtractRemaining != nil {
			return LotDisposalResult{}, fmt.Errorf("dispose remaining quantity: %w", errSubtractRemaining)
		}
		var nextAllocatedBasis, errAddBasis = addDecimal(allocatedBasis, matchedBasis)
		if errAddBasis != nil {
			return LotDisposalResult{}, fmt.Errorf("accumulate allocated basis: %w", errAddBasis)
		}

		currentLot.RemainingQuantity = remainingLotQuantity
		currentLot.RemainingBasis = remainingLotBasis
		remainingQuantity = nextRemainingQuantity
		allocatedBasis = nextAllocatedBasis
		matches = append(matches, LotMatch{
			AcquisitionSourceID: strings.TrimSpace(currentLot.SourceID),
			MatchedQuantity:     matchedQuantity,
			MatchedBasis:        matchedBasis,
		})
	}

	if remainingQuantity.Sign() != 0 {
		return LotDisposalResult{}, fmt.Errorf("disposal quantity exceeds open lot quantity")
	}

	return LotDisposalResult{Matches: matches, AllocatedBasis: allocatedBasis}, nil
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
	var total = zeroDecimal()

	for _, lot := range state.OpenLots() {
		var nextTotal, err = addDecimal(total, lot.RemainingQuantity)
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
	var total = zeroDecimal()

	for _, lot := range state.OpenLots() {
		var nextTotal, err = addDecimal(total, lot.RemainingBasis)
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
	var leftCross, err = multiplyDecimal(left.RemainingBasis, right.RemainingQuantity)
	if err != nil {
		return 0, err
	}
	var rightCross apd.Decimal
	rightCross, err = multiplyDecimal(right.RemainingBasis, left.RemainingQuantity)
	if err != nil {
		return 0, err
	}

	return leftCross.Cmp(&rightCross), nil
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
	if err := validatePositiveDecimal(acquisition.RemainingQuantity, "lot acquisition remaining quantity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(acquisition.RemainingBasis, "lot acquisition remaining basis"); err != nil {
		return err
	}

	return nil
}

// exactProportionalBasis allocates one matched basis fragment without rounding.
// Authored by: OpenCode
func exactProportionalBasis(totalBasis apd.Decimal, totalQuantity apd.Decimal, matchedQuantity apd.Decimal) (apd.Decimal, error) {
	if err := validateNonNegativeDecimal(totalBasis, "total basis"); err != nil {
		return apd.Decimal{}, err
	}
	if err := validatePositiveDecimal(totalQuantity, "total quantity"); err != nil {
		return apd.Decimal{}, err
	}
	if err := validatePositiveDecimal(matchedQuantity, "matched quantity"); err != nil {
		return apd.Decimal{}, err
	}
	if matchedQuantity.Cmp(&totalQuantity) > 0 {
		return apd.Decimal{}, fmt.Errorf("matched quantity exceeds total quantity")
	}
	if matchedQuantity.Cmp(&totalQuantity) == 0 {
		return cloneDecimal(totalBasis), nil
	}

	var numerator, err = multiplyDecimal(totalBasis, matchedQuantity)
	if err != nil {
		return apd.Decimal{}, err
	}

	var quotient, _, divideErr = decimalsupport.DivideExact(numerator, totalQuantity)
	if divideErr != nil {
		return apd.Decimal{}, fmt.Errorf("allocate basis exactly: %w", divideErr)
	}

	return quotient, nil
}

// addDecimal adds two exact decimals.
// Authored by: OpenCode
func addDecimal(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	var sum apd.Decimal
	if _, err := apd.BaseContext.Add(&sum, &left, &right); err != nil {
		return apd.Decimal{}, fmt.Errorf("add decimals: %w", err)
	}

	return sum, nil
}

// subtractDecimal subtracts one exact decimal from another.
// Authored by: OpenCode
func subtractDecimal(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	var difference apd.Decimal
	if _, err := apd.BaseContext.Sub(&difference, &left, &right); err != nil {
		return apd.Decimal{}, fmt.Errorf("subtract decimals: %w", err)
	}

	return difference, nil
}

// multiplyDecimal multiplies two exact decimals.
// Authored by: OpenCode
func multiplyDecimal(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
	var product apd.Decimal
	if _, err := apd.BaseContext.Mul(&product, &left, &right); err != nil {
		return apd.Decimal{}, fmt.Errorf("multiply decimals: %w", err)
	}

	return product, nil
}

// minimumDecimal returns the smaller of two exact decimal values.
// Authored by: OpenCode
func minimumDecimal(left apd.Decimal, right apd.Decimal) apd.Decimal {
	if left.Cmp(&right) <= 0 {
		return cloneDecimal(left)
	}

	return cloneDecimal(right)
}

// zeroDecimal returns one finite zero value.
// Authored by: OpenCode
func zeroDecimal() apd.Decimal {
	return apd.Decimal{}
}

// cloneDecimal returns a copy of one exact decimal value.
// Authored by: OpenCode
func cloneDecimal(value apd.Decimal) apd.Decimal {
	return value
}

// cloneLotAcquisition returns a defensive copy of one lot acquisition.
// Authored by: OpenCode
func cloneLotAcquisition(acquisition LotAcquisition) LotAcquisition {
	return LotAcquisition{
		SourceID:           strings.TrimSpace(acquisition.SourceID),
		AcquiredAt:         acquisition.AcquiredAt,
		DeterministicOrder: acquisition.DeterministicOrder,
		RemainingQuantity:  cloneDecimal(acquisition.RemainingQuantity),
		RemainingBasis:     cloneDecimal(acquisition.RemainingBasis),
	}
}

// validatePositiveDecimal verifies one positive finite decimal value.
// Authored by: OpenCode
func validatePositiveDecimal(value apd.Decimal, label string) error {
	if _, err := decimalsupport.CanonicalString(value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if value.Sign() <= 0 {
		return fmt.Errorf("%s must be greater than zero", label)
	}

	return nil
}

// validateNonNegativeDecimal verifies one non-negative finite decimal value.
// Authored by: OpenCode
func validateNonNegativeDecimal(value apd.Decimal, label string) error {
	if _, err := decimalsupport.CanonicalString(value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if value.Sign() < 0 {
		return fmt.Errorf("%s must not be negative", label)
	}

	return nil
}
