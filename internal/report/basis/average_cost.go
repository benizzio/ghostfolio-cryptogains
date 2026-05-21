// Package basis defines cost-basis state and allocation rules used by report
// calculation.
// Authored by: OpenCode
package basis

import (
	"fmt"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// AverageCostDisposalResult stores one exact basis allocation from the moving
// average-cost pool.
// Authored by: OpenCode
type AverageCostDisposalResult struct {
	DisposedQuantity  apd.Decimal
	AllocatedBasis    apd.Decimal
	RemainingQuantity apd.Decimal
	RemainingBasis    apd.Decimal
}

// AverageCostState stores one moving weighted-average basis pool.
// Authored by: OpenCode
type AverageCostState struct {
	quantity apd.Decimal
	basis    apd.Decimal
}

// NewAverageCostState creates one empty moving average-cost basis pool.
//
// Example:
//
//	state := basis.NewAverageCostState()
//	_ = state.IsEmpty()
//
// Authored by: OpenCode
func NewAverageCostState() *AverageCostState {
	return &AverageCostState{}
}

// AddAcquisition adds one exact acquisition quantity and basis to the moving
// pool.
// Authored by: OpenCode
func (state *AverageCostState) AddAcquisition(quantity apd.Decimal, basis apd.Decimal) error {
	if state == nil {
		return fmt.Errorf("average cost state is required")
	}
	if err := validatePositiveDecimal(quantity, "average cost acquisition quantity"); err != nil {
		return err
	}
	if err := validateNonNegativeDecimal(basis, "average cost acquisition basis"); err != nil {
		return err
	}

	var nextQuantity, err = addDecimal(state.quantity, quantity)
	if err != nil {
		return err
	}
	var nextBasis apd.Decimal
	nextBasis, err = addDecimal(state.basis, basis)
	if err != nil {
		return err
	}

	state.quantity = nextQuantity
	state.basis = nextBasis
	return nil
}

// Dispose removes one quantity from the moving pool using exact average-cost
// allocation with no rounding.
// Authored by: OpenCode
func (state *AverageCostState) Dispose(quantity apd.Decimal) (AverageCostDisposalResult, error) {
	if state == nil {
		return AverageCostDisposalResult{}, fmt.Errorf("average cost state is required")
	}
	if err := validatePositiveDecimal(quantity, "average cost disposal quantity"); err != nil {
		return AverageCostDisposalResult{}, err
	}
	if state.quantity.Sign() == 0 {
		return AverageCostDisposalResult{}, fmt.Errorf("average cost disposal quantity exceeds open pool quantity")
	}
	if quantity.Cmp(&state.quantity) > 0 {
		return AverageCostDisposalResult{}, fmt.Errorf("average cost disposal quantity exceeds open pool quantity")
	}

	var allocatedBasis, err = exactProportionalBasis(state.basis, state.quantity, quantity)
	if err != nil {
		return AverageCostDisposalResult{}, err
	}
	var remainingQuantity, errSubtractQuantity = subtractDecimal(state.quantity, quantity)
	if errSubtractQuantity != nil {
		return AverageCostDisposalResult{}, errSubtractQuantity
	}
	var remainingBasis, errSubtractBasis = subtractDecimal(state.basis, allocatedBasis)
	if errSubtractBasis != nil {
		return AverageCostDisposalResult{}, errSubtractBasis
	}

	if remainingQuantity.Sign() == 0 {
		state.quantity = zeroDecimal()
		state.basis = zeroDecimal()
		return AverageCostDisposalResult{
			DisposedQuantity:  cloneDecimal(quantity),
			AllocatedBasis:    allocatedBasis,
			RemainingQuantity: zeroDecimal(),
			RemainingBasis:    zeroDecimal(),
		}, nil
	}

	state.quantity = remainingQuantity
	state.basis = remainingBasis
	return AverageCostDisposalResult{
		DisposedQuantity:  cloneDecimal(quantity),
		AllocatedBasis:    allocatedBasis,
		RemainingQuantity: remainingQuantity,
		RemainingBasis:    remainingBasis,
	}, nil
}

// Quantity returns the current open pool quantity.
// Authored by: OpenCode
func (state *AverageCostState) Quantity() apd.Decimal {
	if state == nil {
		return zeroDecimal()
	}

	return cloneDecimal(state.quantity)
}

// Basis returns the current open pool basis.
// Authored by: OpenCode
func (state *AverageCostState) Basis() apd.Decimal {
	if state == nil {
		return zeroDecimal()
	}

	return cloneDecimal(state.basis)
}

// AverageUnitCost returns the exact current moving average unit cost.
// Authored by: OpenCode
func (state *AverageCostState) AverageUnitCost() (apd.Decimal, error) {
	if state == nil {
		return apd.Decimal{}, fmt.Errorf("average cost state is required")
	}
	if state.quantity.Sign() == 0 {
		return zeroDecimal(), nil
	}

	var average, _, err = decimalsupport.DivideExact(state.basis, state.quantity)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("calculate average unit cost exactly: %w", err)
	}

	return average, nil
}

// IsEmpty reports whether the moving pool has reset to zero quantity.
// Authored by: OpenCode
func (state *AverageCostState) IsEmpty() bool {
	if state == nil {
		return true
	}

	return state.quantity.Sign() == 0
}
