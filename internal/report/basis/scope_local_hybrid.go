// Package basis defines cost-basis state and allocation rules used by report
// calculation.
// Authored by: OpenCode
package basis

import (
	"fmt"
	"sort"
	"strings"
	"time"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// ScopeLocalHybridAcquisition stores one acquisition routed into one applicable
// scope.
// Authored by: OpenCode
type ScopeLocalHybridAcquisition struct {
	SourceID           string
	ScopeKey           string
	AcquiredAt         time.Time
	DeterministicOrder int
	Quantity           apd.Decimal
	Basis              apd.Decimal
}

// ScopeLocalHybridDisposalResult stores one scope-local disposal result.
// Authored by: OpenCode
type ScopeLocalHybridDisposalResult struct {
	AllocatedBasis apd.Decimal
	Matches        []LotMatch
	ReachedZero    bool
}

// ScopeLocalHybridState stores independent exact-or-fallback basis state per
// applicable scope.
// Authored by: OpenCode
type ScopeLocalHybridState struct {
	scopes map[string]*scopeLocalOpenState
}

// scopeLocalOpenState stores one open applicable-scope state.
// Authored by: OpenCode
type scopeLocalOpenState struct {
	exactState     *LotMethodState
	fallbackPool   *AverageCostState
	provenanceLots []scopeLocalProvenanceLot
}

// scopeLocalProvenanceLot stores oldest-acquired queue state used while one
// scope remains in average-cost fallback mode.
// Authored by: OpenCode
type scopeLocalProvenanceLot struct {
	AcquiredAt         time.Time
	DeterministicOrder int
	RemainingQuantity  apd.Decimal
}

// Test seams keep defensive scope-local wrapper branches directly coverable.
// Authored by: OpenCode
var (
	scopeLocalNewLotMethodState    = NewLotMethodState
	scopeLocalLotTotalOpenBasis    = func(state *LotMethodState) (apd.Decimal, error) { return state.TotalOpenBasis() }
	scopeLocalLotTotalOpenQuantity = func(state *LotMethodState) (apd.Decimal, error) { return state.TotalOpenQuantity() }
	scopeLocalSubtractDecimal      = subtractDecimal
)

// NewScopeLocalHybridState creates one empty scope-local hybrid basis state.
//
// Example:
//
//	state := basis.NewScopeLocalHybridState()
//	_, _ = state.TotalOpenQuantity()
//
// Authored by: OpenCode
func NewScopeLocalHybridState() *ScopeLocalHybridState {
	return &ScopeLocalHybridState{scopes: make(map[string]*scopeLocalOpenState)}
}

// AddAcquisition adds one acquisition into one applicable scope.
// Authored by: OpenCode
func (state *ScopeLocalHybridState) AddAcquisition(acquisition ScopeLocalHybridAcquisition) error {
	if state == nil {
		return fmt.Errorf("scope-local hybrid state is required")
	}
	if err := validateScopeLocalHybridAcquisition(acquisition); err != nil {
		return err
	}

	var scopeState, err = state.ensureScopeState(acquisition.ScopeKey)
	if err != nil {
		return err
	}

	if scopeState.inFallback() {
		err = scopeState.fallbackPool.AddAcquisition(acquisition.Quantity, acquisition.Basis)
		if err != nil {
			return err
		}
		scopeState.provenanceLots = append(scopeState.provenanceLots, scopeLocalProvenanceLot{
			AcquiredAt:         acquisition.AcquiredAt,
			DeterministicOrder: acquisition.DeterministicOrder,
			RemainingQuantity:  supportmath.Clone(acquisition.Quantity),
		})
		return nil
	}

	return scopeState.exactState.AddAcquisition(LotAcquisition{
		SourceID:           acquisition.SourceID,
		AcquiredAt:         acquisition.AcquiredAt,
		DeterministicOrder: acquisition.DeterministicOrder,
		RemainingQuantity:  acquisition.Quantity,
		RemainingBasis:     acquisition.Basis,
	})
}

// Dispose removes one quantity from one applicable scope.
// Authored by: OpenCode
func (state *ScopeLocalHybridState) Dispose(scopeKey string, quantity apd.Decimal) (ScopeLocalHybridDisposalResult, error) {
	if state == nil {
		return ScopeLocalHybridDisposalResult{}, fmt.Errorf("scope-local hybrid state is required")
	}
	if strings.TrimSpace(scopeKey) == "" {
		return ScopeLocalHybridDisposalResult{}, fmt.Errorf("scope-local hybrid disposal scope key is required")
	}
	if err := validatePositiveDecimal(quantity, "scope-local hybrid disposal quantity"); err != nil {
		return ScopeLocalHybridDisposalResult{}, err
	}

	var normalizedScopeKey = strings.TrimSpace(scopeKey)
	var scopeState = state.scopes[normalizedScopeKey]
	if scopeState == nil {
		return ScopeLocalHybridDisposalResult{}, fmt.Errorf("scope-local hybrid disposal quantity exceeds open scope quantity")
	}

	if scopeState.inFallback() {
		return state.disposeWithFallback(normalizedScopeKey, scopeState, quantity)
	}

	if scopeState.exactState.OpenLotCount() == 1 {
		var exactResult, err = scopeState.exactState.Dispose(quantity)
		if err != nil {
			return ScopeLocalHybridDisposalResult{}, err
		}
		var remainingQuantity apd.Decimal
		remainingQuantity, err = scopeState.exactState.TotalOpenQuantity()
		if err != nil {
			return ScopeLocalHybridDisposalResult{}, err
		}
		if remainingQuantity.Sign() == 0 {
			delete(state.scopes, normalizedScopeKey)
			return ScopeLocalHybridDisposalResult{AllocatedBasis: exactResult.AllocatedBasis, Matches: append([]LotMatch(nil), exactResult.Matches...), ReachedZero: true}, nil
		}
		return ScopeLocalHybridDisposalResult{AllocatedBasis: exactResult.AllocatedBasis, Matches: append([]LotMatch(nil), exactResult.Matches...), ReachedZero: false}, nil
	}

	err := scopeState.activateFallback()
	if err != nil {
		return ScopeLocalHybridDisposalResult{}, err
	}
	return state.disposeWithFallback(normalizedScopeKey, scopeState, quantity)
}

// TotalOpenQuantity returns the exact open quantity across all applicable
// scopes.
// Authored by: OpenCode
func (state *ScopeLocalHybridState) TotalOpenQuantity() (apd.Decimal, error) {
	if state == nil {
		return supportmath.Zero(), nil
	}

	var total = supportmath.Zero()
	for _, scopeState := range state.scopes {
		var scopeQuantity apd.Decimal
		var err error
		if scopeState.inFallback() {
			scopeQuantity = scopeState.fallbackPool.Quantity()
		} else {
			scopeQuantity, err = scopeLocalLotTotalOpenQuantity(scopeState.exactState)
			if err != nil {
				return apd.Decimal{}, err
			}
		}
		total, err = addDecimal(total, scopeQuantity)
		if err != nil {
			return apd.Decimal{}, err
		}
	}

	return total, nil
}

// TotalOpenBasis returns the exact open basis across all applicable scopes.
// Authored by: OpenCode
func (state *ScopeLocalHybridState) TotalOpenBasis() (apd.Decimal, error) {
	if state == nil {
		return supportmath.Zero(), nil
	}

	var total = supportmath.Zero()
	for _, scopeState := range state.scopes {
		var scopeBasis apd.Decimal
		var err error
		if scopeState.inFallback() {
			scopeBasis = scopeState.fallbackPool.Basis()
		} else {
			scopeBasis, err = scopeLocalLotTotalOpenBasis(scopeState.exactState)
			if err != nil {
				return apd.Decimal{}, err
			}
		}
		total, err = addDecimal(total, scopeBasis)
		if err != nil {
			return apd.Decimal{}, err
		}
	}

	return total, nil
}

// ensureScopeState returns one existing or newly created scope-local state.
// Authored by: OpenCode
func (state *ScopeLocalHybridState) ensureScopeState(scopeKey string) (*scopeLocalOpenState, error) {
	var normalizedScopeKey = strings.TrimSpace(scopeKey)
	if normalizedScopeKey == "" {
		return nil, fmt.Errorf("scope-local hybrid acquisition scope key is required")
	}

	var existing = state.scopes[normalizedScopeKey]
	if existing != nil {
		return existing, nil
	}

	var exactState, err = scopeLocalNewLotMethodState(LotMethodFIFO)
	if err != nil {
		return nil, err
	}

	var scopeState = &scopeLocalOpenState{exactState: exactState}
	state.scopes[normalizedScopeKey] = scopeState
	return scopeState, nil
}

// disposeWithFallback allocates basis from the scope-local average-cost pool and
// consumes provenance quantity in oldest-acquired order.
// Authored by: OpenCode
func (state *ScopeLocalHybridState) disposeWithFallback(scopeKey string, scopeState *scopeLocalOpenState, quantity apd.Decimal) (ScopeLocalHybridDisposalResult, error) {
	var poolResult, err = scopeState.fallbackPool.Dispose(quantity)
	if err != nil {
		return ScopeLocalHybridDisposalResult{}, err
	}
	if err = scopeState.consumeFallbackProvenance(quantity); err != nil {
		return ScopeLocalHybridDisposalResult{}, err
	}

	if poolResult.RemainingQuantity.Sign() == 0 {
		delete(state.scopes, scopeKey)
		return ScopeLocalHybridDisposalResult{AllocatedBasis: poolResult.AllocatedBasis, ReachedZero: true}, nil
	}

	return ScopeLocalHybridDisposalResult{AllocatedBasis: poolResult.AllocatedBasis, ReachedZero: false}, nil
}

// inFallback reports whether one scope is already using average-cost fallback.
// Authored by: OpenCode
func (state *scopeLocalOpenState) inFallback() bool {
	return state != nil && state.fallbackPool != nil
}

// activateFallback snapshots the current exact state into one scope-local
// average-cost pool and oldest-acquired provenance queue.
// Authored by: OpenCode
func (state *scopeLocalOpenState) activateFallback() error {
	if state == nil {
		return fmt.Errorf("scope-local open state is required")
	}
	if state.inFallback() {
		return nil
	}

	var openLots = state.exactState.OpenLots()
	if len(openLots) == 0 {
		return fmt.Errorf("scope-local hybrid fallback requires open quantity")
	}

	sort.SliceStable(openLots, func(left int, right int) bool {
		return compareLotChronology(openLots[left], openLots[right]) < 0
	})

	state.fallbackPool = NewAverageCostState()
	state.provenanceLots = make([]scopeLocalProvenanceLot, 0, len(openLots))
	for _, lot := range openLots {
		if err := state.fallbackPool.AddAcquisition(lot.RemainingQuantity, lot.RemainingBasis); err != nil {
			return err
		}
		state.provenanceLots = append(state.provenanceLots, scopeLocalProvenanceLot{
			AcquiredAt:         lot.AcquiredAt,
			DeterministicOrder: lot.DeterministicOrder,
			RemainingQuantity:  supportmath.Clone(lot.RemainingQuantity),
		})
	}

	state.exactState = nil
	return nil
}

// consumeFallbackProvenance removes quantity from the oldest-acquired fallback
// provenance queue.
// Authored by: OpenCode
func (state *scopeLocalOpenState) consumeFallbackProvenance(quantity apd.Decimal) error {
	var remainingQuantity = supportmath.Clone(quantity)

	for index := range state.provenanceLots {
		if remainingQuantity.Sign() == 0 {
			break
		}

		var matchedQuantity = supportmath.Minimum(state.provenanceLots[index].RemainingQuantity, remainingQuantity)
		if matchedQuantity.Sign() == 0 {
			continue
		}

		var nextLotQuantity, err = scopeLocalSubtractDecimal(state.provenanceLots[index].RemainingQuantity, matchedQuantity)
		if err != nil {
			return err
		}
		var nextRemainingQuantity apd.Decimal
		nextRemainingQuantity, err = scopeLocalSubtractDecimal(remainingQuantity, matchedQuantity)
		if err != nil {
			return err
		}

		state.provenanceLots[index].RemainingQuantity = nextLotQuantity
		remainingQuantity = nextRemainingQuantity
	}

	if remainingQuantity.Sign() != 0 {
		return fmt.Errorf("scope-local hybrid disposal quantity exceeds fallback provenance quantity")
	}

	return nil
}

// validateScopeLocalHybridAcquisition verifies one scope-local acquisition.
// Authored by: OpenCode
func validateScopeLocalHybridAcquisition(acquisition ScopeLocalHybridAcquisition) error {
	if strings.TrimSpace(acquisition.ScopeKey) == "" {
		return fmt.Errorf("scope-local hybrid acquisition scope key is required")
	}
	return validateLotAcquisition(LotAcquisition{
		SourceID:           acquisition.SourceID,
		AcquiredAt:         acquisition.AcquiredAt,
		DeterministicOrder: acquisition.DeterministicOrder,
		RemainingQuantity:  acquisition.Quantity,
		RemainingBasis:     acquisition.Basis,
	})
}
