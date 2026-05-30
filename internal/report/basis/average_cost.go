// Package basis defines cost-basis state and allocation rules used by report
// calculation.
// Authored by: OpenCode
package basis

import (
	"fmt"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
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
//
// Example:
//
//	state := basis.NewAverageCostState()
//	if err := state.AddAcquisition(*apd.New(2, 0), *apd.New(10, 0)); err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func (state *AverageCostState) AddAcquisition(quantity apd.Decimal, basis apd.Decimal) error {
	if state == nil {
		return fmt.Errorf("average cost state is required")
	}
	if err := supportmath.RequirePositive(quantity, "average cost acquisition quantity"); err != nil {
		return err
	}
	if err := supportmath.RequireNonNegative(basis, "average cost acquisition basis"); err != nil {
		return err
	}

	var nextQuantity, err = supportmath.Add(state.quantity, quantity)
	if err != nil {
		return err
	}
	var nextBasis apd.Decimal
	nextBasis, err = supportmath.Add(state.basis, basis)
	if err != nil {
		return err
	}

	state.quantity = nextQuantity
	state.basis = nextBasis
	return nil
}

// Dispose removes one quantity from the moving pool using the shared internal
// report-calculation precision for proportional allocation when needed.
//
// Example:
//
//	state := basis.NewAverageCostState()
//	_ = state.AddAcquisition(*apd.New(4, 0), *apd.New(20, 0))
//	result, err := state.Dispose(*apd.New(1, 0))
//	if err != nil {
//		panic(err)
//	}
//	_ = result.AllocatedBasis
//
// Authored by: OpenCode
func (state *AverageCostState) Dispose(quantity apd.Decimal) (AverageCostDisposalResult, error) {
	if state == nil {
		return AverageCostDisposalResult{}, fmt.Errorf("average cost state is required")
	}
	if err := supportmath.RequirePositive(quantity, "average cost disposal quantity"); err != nil {
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
	// exactProportionalBasis already proved that the pool state and disposal input
	// are finite and compatible, so these exact subtractions cannot fail.
	var remainingQuantity apd.Decimal
	_, _ = apd.BaseContext.Sub(&remainingQuantity, &state.quantity, &quantity)
	var remainingBasis apd.Decimal
	_, _ = apd.BaseContext.Sub(&remainingBasis, &state.basis, &allocatedBasis)

	if remainingQuantity.Sign() == 0 {
		state.quantity = supportmath.Zero()
		state.basis = supportmath.Zero()
		return AverageCostDisposalResult{
			DisposedQuantity:  supportmath.Clone(quantity),
			AllocatedBasis:    allocatedBasis,
			RemainingQuantity: supportmath.Zero(),
			RemainingBasis:    supportmath.Zero(),
		}, nil
	}

	state.quantity = remainingQuantity
	state.basis = remainingBasis
	return AverageCostDisposalResult{
		DisposedQuantity:  supportmath.Clone(quantity),
		AllocatedBasis:    allocatedBasis,
		RemainingQuantity: remainingQuantity,
		RemainingBasis:    remainingBasis,
	}, nil
}

// Quantity returns the current open pool quantity.
//
// Example:
//
//	state := basis.NewAverageCostState()
//	_ = state.AddAcquisition(*apd.New(2, 0), *apd.New(10, 0))
//	openQuantity := state.Quantity()
//	_ = openQuantity
//
// Authored by: OpenCode
func (state *AverageCostState) Quantity() apd.Decimal {
	if state == nil {
		return supportmath.Zero()
	}

	return supportmath.Clone(state.quantity)
}

// Basis returns the current open pool basis.
//
// Example:
//
//	state := basis.NewAverageCostState()
//	_ = state.AddAcquisition(*apd.New(2, 0), *apd.New(10, 0))
//	openBasis := state.Basis()
//	_ = openBasis
//
// Authored by: OpenCode
func (state *AverageCostState) Basis() apd.Decimal {
	if state == nil {
		return supportmath.Zero()
	}

	return supportmath.Clone(state.basis)
}

// AverageUnitCost returns the current moving average unit cost using the shared
// internal report-calculation precision when division does not terminate.
//
// Example:
//
//	state := basis.NewAverageCostState()
//	_ = state.AddAcquisition(*apd.New(4, 0), *apd.New(20, 0))
//	averageUnitCost, err := state.AverageUnitCost()
//	if err != nil {
//		panic(err)
//	}
//	_ = averageUnitCost
//
// Authored by: OpenCode
func (state *AverageCostState) AverageUnitCost() (apd.Decimal, error) {
	if state == nil {
		return apd.Decimal{}, fmt.Errorf("average cost state is required")
	}
	if state.quantity.Sign() == 0 {
		return supportmath.Zero(), nil
	}

	var average, err = supportmath.DivideFiniteRoundHalfUp(state.basis, state.quantity)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("calculate average unit cost: %w", err)
	}

	return average, nil
}

// IsEmpty reports whether the moving pool has reset to zero quantity.
//
// Example:
//
//	state := basis.NewAverageCostState()
//	if !state.IsEmpty() {
//		panic("expected a new average-cost state to be empty")
//	}
//
// Authored by: OpenCode
func (state *AverageCostState) IsEmpty() bool {
	if state == nil {
		return true
	}

	return state.quantity.Sign() == 0
}
