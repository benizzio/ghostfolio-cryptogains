// Package calculate defines calculator-local adapters for report basis states.
// Authored by: OpenCode
package calculate

import (
	"fmt"
	"strings"
	"time"

	reportbasis "github.com/benizzio/ghostfolio-cryptogains/internal/report/basis"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
)

// basisAcquisitionInput stores the exact acquisition values forwarded into one
// method-specific basis state.
// Authored by: OpenCode
type basisAcquisitionInput struct {
	SourceID           string
	AcquiredAt         time.Time
	DeterministicOrder int
	Quantity           apd.Decimal
	Basis              apd.Decimal
	ApplicableScopeKey string
}

// assetBasisState adapts one method-specific open-position state behind a
// minimal calculator-local interface.
// Authored by: OpenCode
type assetBasisState interface {
	AddAcquisition(basisAcquisitionInput) error
	Dispose(basisDisposalInput) (basisDisposalResult, error)
	OpenQuantity() (apd.Decimal, error)
	OpenBasis() (apd.Decimal, error)
}

// lotBasisState adapts FIFO, LIFO, and HIFO lot tracking.
// Authored by: OpenCode
type lotBasisState struct {
	state *reportbasis.LotMethodState
}

// averageCostBasisState adapts the moving average-cost pool.
// Authored by: OpenCode
type averageCostBasisState struct {
	state *reportbasis.AverageCostState
}

// scopeLocalHybridBasisState adapts the scope-local hybrid method state.
// Authored by: OpenCode
type scopeLocalHybridBasisState struct {
	state *reportbasis.ScopeLocalHybridState
}

// basisDisposalInput stores one disposal routed through the active basis state.
// Authored by: OpenCode
type basisDisposalInput struct {
	Quantity           apd.Decimal
	ApplicableScopeKey string
}

// basisDisposalResult stores one basis allocation and whether the relevant
// asset or scope transitioned to zero.
// Authored by: OpenCode
type basisDisposalResult struct {
	AllocatedBasis apd.Decimal
	Matches        []reportmodel.BasisMatch
	ReachedZero    bool
}

// newAssetBasisState creates one method-specific open-position state for the
// requested cost-basis method.
// Authored by: OpenCode
func newAssetBasisState(method reportmodel.CostBasisMethod) (assetBasisState, error) {
	switch method {
	case reportmodel.CostBasisMethodFIFO:
		var state, err = newLotMethodState(reportbasis.LotMethodFIFO)
		if err != nil {
			return nil, err
		}
		return lotBasisState{state: state}, nil
	case reportmodel.CostBasisMethodLIFO:
		var state, err = newLotMethodState(reportbasis.LotMethodLIFO)
		if err != nil {
			return nil, err
		}
		return lotBasisState{state: state}, nil
	case reportmodel.CostBasisMethodHIFO:
		var state, err = newLotMethodState(reportbasis.LotMethodHIFO)
		if err != nil {
			return nil, err
		}
		return lotBasisState{state: state}, nil
	case reportmodel.CostBasisMethodAverageCost:
		return averageCostBasisState{state: reportbasis.NewAverageCostState()}, nil
	case reportmodel.CostBasisMethodScopeLocalHybrid:
		return scopeLocalHybridBasisState{state: reportbasis.NewScopeLocalHybridState()}, nil
	default:
		return nil, fmt.Errorf("unsupported cost basis method %q", method)
	}
}

// AddAcquisition adds one acquisition lot to a lot-based method state.
// Authored by: OpenCode
func (state lotBasisState) AddAcquisition(input basisAcquisitionInput) error {
	return state.state.AddAcquisition(reportbasis.LotAcquisition{
		SourceID:           input.SourceID,
		AcquiredAt:         input.AcquiredAt,
		DeterministicOrder: input.DeterministicOrder,
		RemainingQuantity:  input.Quantity,
		RemainingBasis:     input.Basis,
	})
}

// Dispose removes one quantity from a lot-based method state and returns the
// exact allocated basis.
// Authored by: OpenCode
func (state lotBasisState) Dispose(input basisDisposalInput) (basisDisposalResult, error) {
	var result, err = state.state.Dispose(input.Quantity)
	if err != nil {
		return basisDisposalResult{}, err
	}
	var remainingQuantity apd.Decimal
	remainingQuantity, err = lotStateTotalOpenQuantity(state.state)
	if err != nil {
		return basisDisposalResult{}, err
	}

	var matches = make([]reportmodel.BasisMatch, 0, len(result.Matches))
	for _, match := range result.Matches {
		matches = append(matches, reportmodel.BasisMatch{
			AcquisitionSourceID: strings.TrimSpace(match.AcquisitionSourceID),
			MatchedQuantity:     match.MatchedQuantity,
			MatchedBasis:        match.MatchedBasis,
		})
	}

	return basisDisposalResult{AllocatedBasis: result.AllocatedBasis, Matches: matches, ReachedZero: remainingQuantity.Sign() == 0}, nil
}

// OpenQuantity returns the exact remaining lot quantity.
// Authored by: OpenCode
func (state lotBasisState) OpenQuantity() (apd.Decimal, error) {
	return state.state.TotalOpenQuantity()
}

// OpenBasis returns the exact remaining lot basis.
// Authored by: OpenCode
func (state lotBasisState) OpenBasis() (apd.Decimal, error) {
	return state.state.TotalOpenBasis()
}

// AddAcquisition adds one acquisition into the moving average-cost pool.
// Authored by: OpenCode
func (state averageCostBasisState) AddAcquisition(input basisAcquisitionInput) error {
	return state.state.AddAcquisition(input.Quantity, input.Basis)
}

// Dispose removes one quantity from the moving average-cost pool and returns the
// exact allocated basis.
// Authored by: OpenCode
func (state averageCostBasisState) Dispose(input basisDisposalInput) (basisDisposalResult, error) {
	var result, err = state.state.Dispose(input.Quantity)
	if err != nil {
		return basisDisposalResult{}, err
	}

	return basisDisposalResult{AllocatedBasis: result.AllocatedBasis, Matches: []reportmodel.BasisMatch{{AcquisitionSourceID: "AVERAGE_COST_POOL", MatchedQuantity: result.DisposedQuantity, MatchedBasis: result.AllocatedBasis}}, ReachedZero: result.RemainingQuantity.Sign() == 0}, nil
}

// OpenQuantity returns the exact remaining moving-pool quantity.
// Authored by: OpenCode
func (state averageCostBasisState) OpenQuantity() (apd.Decimal, error) {
	return state.state.Quantity(), nil
}

// OpenBasis returns the exact remaining moving-pool basis.
// Authored by: OpenCode
func (state averageCostBasisState) OpenBasis() (apd.Decimal, error) {
	return state.state.Basis(), nil
}

// AddAcquisition adds one acquisition into one scope-local scope partition.
// Authored by: OpenCode
func (state scopeLocalHybridBasisState) AddAcquisition(input basisAcquisitionInput) error {
	return state.state.AddAcquisition(reportbasis.ScopeLocalHybridAcquisition{
		SourceID:           input.SourceID,
		ScopeKey:           input.ApplicableScopeKey,
		AcquiredAt:         input.AcquiredAt,
		DeterministicOrder: input.DeterministicOrder,
		Quantity:           input.Quantity,
		Basis:              input.Basis,
	})
}

// Dispose removes one quantity from one scope-local scope partition.
// Authored by: OpenCode
func (state scopeLocalHybridBasisState) Dispose(input basisDisposalInput) (basisDisposalResult, error) {
	var result, err = state.state.Dispose(input.ApplicableScopeKey, input.Quantity)
	if err != nil {
		return basisDisposalResult{}, err
	}

	var matches = make([]reportmodel.BasisMatch, 0, len(result.Matches))
	for _, match := range result.Matches {
		matches = append(matches, reportmodel.BasisMatch{
			AcquisitionSourceID: strings.TrimSpace(match.AcquisitionSourceID),
			MatchedQuantity:     match.MatchedQuantity,
			MatchedBasis:        match.MatchedBasis,
		})
	}
	if len(matches) == 0 {
		matches = []reportmodel.BasisMatch{{
			AcquisitionSourceID: strings.TrimSpace(input.ApplicableScopeKey),
			MatchedQuantity:     input.Quantity,
			MatchedBasis:        result.AllocatedBasis,
		}}
	}

	return basisDisposalResult{AllocatedBasis: result.AllocatedBasis, Matches: matches, ReachedZero: result.ReachedZero}, nil
}

// OpenQuantity returns the exact remaining quantity across all open scopes.
// Authored by: OpenCode
func (state scopeLocalHybridBasisState) OpenQuantity() (apd.Decimal, error) {
	return state.state.TotalOpenQuantity()
}

// OpenBasis returns the exact remaining basis across all open scopes.
// Authored by: OpenCode
func (state scopeLocalHybridBasisState) OpenBasis() (apd.Decimal, error) {
	return state.state.TotalOpenBasis()
}
