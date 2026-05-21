// Package basis verifies package-local scope-local hybrid guardrails and helper
// paths.
// Authored by: OpenCode
package basis

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/apd/v3"
)

// TestScopeLocalHybridStateGuardrails verifies nil receiver behavior,
// acquisition validation, and one-lot exact disposal paths.
// Authored by: OpenCode
func TestScopeLocalHybridStateGuardrails(t *testing.T) {
	var nilState *ScopeLocalHybridState
	if err := nilState.AddAcquisition(ScopeLocalHybridAcquisition{}); err == nil {
		t.Fatalf("expected nil scope-local hybrid acquisition to fail")
	}
	if _, err := nilState.Dispose("scope-a", decimalFromInt(1)); err == nil {
		t.Fatalf("expected nil scope-local hybrid disposal to fail")
	}

	var nilQuantity, err = nilState.TotalOpenQuantity()
	if err != nil {
		t.Fatalf("nil scope-local hybrid total quantity: %v", err)
	}
	if nilQuantity.Sign() != 0 {
		t.Fatalf("expected nil scope-local hybrid total quantity to be zero")
	}

	var nilBasis apd.Decimal
	nilBasis, err = nilState.TotalOpenBasis()
	if err != nil {
		t.Fatalf("nil scope-local hybrid total basis: %v", err)
	}
	if nilBasis.Sign() != 0 {
		t.Fatalf("expected nil scope-local hybrid total basis to be zero")
	}

	var state = NewScopeLocalHybridState()
	if err = state.AddAcquisition(ScopeLocalHybridAcquisition{SourceID: "lot-001", Quantity: decimalFromInt(1), Basis: decimalFromInt(1)}); err == nil || !strings.Contains(err.Error(), "scope key is required") {
		t.Fatalf("expected blank scope-key acquisition to fail, got %v", err)
	}
	if _, err = state.Dispose("   ", decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "scope key is required") {
		t.Fatalf("expected blank scope-key disposal to fail, got %v", err)
	}
	if _, err = state.Dispose("scope-a", decimalFromInt(0)); err == nil || !strings.Contains(err.Error(), "disposal quantity") {
		t.Fatalf("expected zero-quantity scope-local disposal to fail, got %v", err)
	}
	if _, err = state.Dispose("missing", decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "exceeds open scope quantity") {
		t.Fatalf("expected missing-scope disposal to fail, got %v", err)
	}

	if err = state.AddAcquisition(ScopeLocalHybridAcquisition{
		SourceID:           "lot-001",
		ScopeKey:           " scope-a ",
		AcquiredAt:         time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
		DeterministicOrder: 1,
		Quantity:           decimalFromInt(2),
		Basis:              decimalFromInt(10),
	}); err != nil {
		t.Fatalf("add one-lot scope-local acquisition: %v", err)
	}

	var partial, partialErr = state.Dispose("scope-a", decimalFromInt(1))
	if partialErr != nil {
		t.Fatalf("dispose one-lot scope partially: %v", partialErr)
	}
	if partial.AllocatedBasis.Cmp(apd.New(5, 0)) != 0 || partial.ReachedZero {
		t.Fatalf("expected one-lot partial disposal to allocate basis 5 and remain open, got basis=%v reachedZero=%t", partial.AllocatedBasis, partial.ReachedZero)
	}

	var final, finalErr = state.Dispose("scope-a", decimalFromInt(1))
	if finalErr != nil {
		t.Fatalf("dispose one-lot scope to zero: %v", finalErr)
	}
	if final.AllocatedBasis.Cmp(apd.New(5, 0)) != 0 || !final.ReachedZero {
		t.Fatalf("expected one-lot final disposal to allocate basis 5 and reach zero, got basis=%v reachedZero=%t", final.AllocatedBasis, final.ReachedZero)
	}
	if _, err = state.Dispose("scope-a", decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "exceeds open scope quantity") {
		t.Fatalf("expected exhausted one-lot scope to reject further disposal, got %v", err)
	}
}

// TestScopeLocalHybridHelperPaths verifies direct package-local helper branches
// used by fallback activation and provenance consumption.
// Authored by: OpenCode
func TestScopeLocalHybridHelperPaths(t *testing.T) {
	var root = NewScopeLocalHybridState()
	var created, err = root.ensureScopeState(" scope-a ")
	if err != nil {
		t.Fatalf("ensure scope state: %v", err)
	}
	var reused, reuseErr = root.ensureScopeState("scope-a")
	if reuseErr != nil {
		t.Fatalf("reuse scope state: %v", reuseErr)
	}
	if created != reused {
		t.Fatalf("expected normalized scope key to reuse the same scope state")
	}
	if _, err = root.ensureScopeState("   "); err == nil || !strings.Contains(err.Error(), "scope key is required") {
		t.Fatalf("expected blank ensureScopeState input to fail, got %v", err)
	}

	var nilOpenState *scopeLocalOpenState
	if err = nilOpenState.activateFallback(); err == nil || !strings.Contains(err.Error(), "scope-local open state is required") {
		t.Fatalf("expected nil fallback activation to fail, got %v", err)
	}

	var exactState, exactErr = NewLotMethodState(LotMethodFIFO)
	if exactErr != nil {
		t.Fatalf("new FIFO exact state: %v", exactErr)
	}
	var openState = &scopeLocalOpenState{exactState: exactState}
	if err = openState.activateFallback(); err == nil || !strings.Contains(err.Error(), "fallback requires open quantity") {
		t.Fatalf("expected empty fallback activation to fail, got %v", err)
	}

	var fallbackState = &scopeLocalOpenState{fallbackPool: NewAverageCostState()}
	if err = fallbackState.activateFallback(); err != nil {
		t.Fatalf("activate fallback for already-fallback state: %v", err)
	}
	if !fallbackState.inFallback() {
		t.Fatalf("expected fallback state to report active fallback")
	}

	exactState, err = NewLotMethodState(LotMethodFIFO)
	if err != nil {
		t.Fatalf("new FIFO exact state for activation: %v", err)
	}
	for _, acquisition := range []LotAcquisition{
		{
			SourceID:           "later",
			AcquiredAt:         time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 2,
			RemainingQuantity:  decimalFromInt(1),
			RemainingBasis:     decimalFromInt(20),
		},
		{
			SourceID:           "earlier",
			AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 1,
			RemainingQuantity:  decimalFromInt(2),
			RemainingBasis:     decimalFromInt(10),
		},
	} {
		if err = exactState.AddAcquisition(acquisition); err != nil {
			t.Fatalf("add exact acquisition %q: %v", acquisition.SourceID, err)
		}
	}

	openState = &scopeLocalOpenState{exactState: exactState}
	if err = openState.activateFallback(); err != nil {
		t.Fatalf("activate fallback: %v", err)
	}
	if !openState.inFallback() || openState.exactState != nil {
		t.Fatalf("expected exact state to convert into fallback state")
	}
	if len(openState.provenanceLots) != 2 || openState.provenanceLots[0].RemainingQuantity.Cmp(apd.New(2, 0)) != 0 {
		t.Fatalf("expected fallback provenance to preserve oldest lot first, got %#v", openState.provenanceLots)
	}

	var provenanceOnly = &scopeLocalOpenState{provenanceLots: []scopeLocalProvenanceLot{{RemainingQuantity: decimalFromInt(1)}}}
	if err = provenanceOnly.consumeFallbackProvenance(decimalFromInt(2)); err == nil || !strings.Contains(err.Error(), "exceeds fallback provenance quantity") {
		t.Fatalf("expected oversized fallback provenance consumption to fail, got %v", err)
	}

	provenanceOnly = &scopeLocalOpenState{provenanceLots: []scopeLocalProvenanceLot{{RemainingQuantity: decimalFromInt(0)}, {RemainingQuantity: decimalFromInt(1)}}}
	if err = provenanceOnly.consumeFallbackProvenance(decimalFromInt(1)); err != nil {
		t.Fatalf("consume fallback provenance across zero-quantity entry: %v", err)
	}
	if provenanceOnly.provenanceLots[1].RemainingQuantity.Sign() != 0 {
		t.Fatalf("expected fallback provenance consumption to skip zero-quantity entries")
	}

	var invalid apd.Decimal
	invalid.Form = apd.NaNSignaling
	openState = &scopeLocalOpenState{exactState: &LotMethodState{method: LotMethodFIFO, lots: []LotAcquisition{{
		SourceID:           "bad-lot",
		AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		DeterministicOrder: 1,
		RemainingQuantity:  decimalFromInt(1),
		RemainingBasis:     invalid,
	}}}}
	if err = openState.activateFallback(); err == nil {
		t.Fatalf("expected corrupted exact state to fail fallback activation")
	}
}

// TestScopeLocalHybridOperationalBranches verifies fallback activation, fallback
// disposal, and cross-scope totals on concrete hybrid state.
// Authored by: OpenCode
func TestScopeLocalHybridOperationalBranches(t *testing.T) {
	var state = NewScopeLocalHybridState()
	for _, acquisition := range []ScopeLocalHybridAcquisition{
		{
			SourceID:           "scope-a-lot-1",
			ScopeKey:           "scope-a",
			AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 1,
			Quantity:           decimalFromInt(1),
			Basis:              decimalFromInt(10),
		},
		{
			SourceID:           "scope-a-lot-2",
			ScopeKey:           "scope-a",
			AcquiredAt:         time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 2,
			Quantity:           decimalFromInt(1),
			Basis:              decimalFromInt(20),
		},
		{
			SourceID:           "scope-b-lot-1",
			ScopeKey:           "scope-b",
			AcquiredAt:         time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 3,
			Quantity:           decimalFromInt(2),
			Basis:              decimalFromInt(8),
		},
	} {
		if err := state.AddAcquisition(acquisition); err != nil {
			t.Fatalf("add scope-local acquisition %q: %v", acquisition.SourceID, err)
		}
	}

	var totalQuantity, quantityErr = state.TotalOpenQuantity()
	if quantityErr != nil {
		t.Fatalf("total open quantity before fallback: %v", quantityErr)
	}
	var totalBasis, basisErr = state.TotalOpenBasis()
	if basisErr != nil {
		t.Fatalf("total open basis before fallback: %v", basisErr)
	}
	if totalQuantity.Cmp(apd.New(4, 0)) != 0 || totalBasis.Cmp(apd.New(38, 0)) != 0 {
		t.Fatalf("unexpected scope-local totals before fallback: quantity=%v basis=%v", totalQuantity, totalBasis)
	}

	var fallbackResult, fallbackErr = state.Dispose("scope-a", decimalFromInt(1))
	if fallbackErr != nil {
		t.Fatalf("dispose fallback scope: %v", fallbackErr)
	}
	if fallbackResult.AllocatedBasis.Cmp(apd.New(15, 0)) != 0 || fallbackResult.ReachedZero {
		t.Fatalf("expected first fallback disposal to allocate average basis and remain open, got %#v", fallbackResult)
	}

	totalQuantity, quantityErr = state.TotalOpenQuantity()
	if quantityErr != nil {
		t.Fatalf("total open quantity after fallback: %v", quantityErr)
	}
	totalBasis, basisErr = state.TotalOpenBasis()
	if basisErr != nil {
		t.Fatalf("total open basis after fallback: %v", basisErr)
	}
	if totalQuantity.Cmp(apd.New(3, 0)) != 0 || totalBasis.Cmp(apd.New(23, 0)) != 0 {
		t.Fatalf("unexpected scope-local totals after fallback: quantity=%v basis=%v", totalQuantity, totalBasis)
	}

	var finalResult, finalErr = state.Dispose("scope-a", decimalFromInt(1))
	if finalErr != nil {
		t.Fatalf("dispose fallback scope to zero: %v", finalErr)
	}
	if finalResult.AllocatedBasis.Cmp(apd.New(15, 0)) != 0 || !finalResult.ReachedZero {
		t.Fatalf("expected final fallback disposal to reach zero, got %#v", finalResult)
	}

	totalQuantity, quantityErr = state.TotalOpenQuantity()
	if quantityErr != nil {
		t.Fatalf("total open quantity after removing scope-a: %v", quantityErr)
	}
	totalBasis, basisErr = state.TotalOpenBasis()
	if basisErr != nil {
		t.Fatalf("total open basis after removing scope-a: %v", basisErr)
	}
	if totalQuantity.Cmp(apd.New(2, 0)) != 0 || totalBasis.Cmp(apd.New(8, 0)) != 0 {
		t.Fatalf("unexpected remaining cross-scope totals: quantity=%v basis=%v", totalQuantity, totalBasis)
	}

	if _, finalErr = state.Dispose("scope-b", decimalFromInt(3)); finalErr == nil || !strings.Contains(finalErr.Error(), "exceeds open lot quantity") {
		t.Fatalf("expected oversized fallback disposal to fail, got %v", finalErr)
	}

	var invalid apd.Decimal
	invalid.Form = apd.NaNSignaling
	state.scopes["scope-b"] = &scopeLocalOpenState{fallbackPool: NewAverageCostState(), provenanceLots: []scopeLocalProvenanceLot{{RemainingQuantity: decimalFromInt(0)}}}
	if err := state.scopes["scope-b"].fallbackPool.AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err != nil {
		t.Fatalf("seed fallback pool: %v", err)
	}
	if _, err := state.Dispose("scope-b", decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "fallback provenance quantity") {
		t.Fatalf("expected fallback provenance mismatch to fail, got %v", err)
	}

	state.scopes["scope-b"] = &scopeLocalOpenState{fallbackPool: &AverageCostState{quantity: decimalFromInt(1), basis: invalid}}
	if _, err := state.TotalOpenBasis(); err == nil {
		t.Fatalf("expected corrupted fallback basis total to fail")
	}

	state.scopes["scope-b"] = &scopeLocalOpenState{fallbackPool: &AverageCostState{quantity: invalid, basis: decimalFromInt(1)}}
	if _, err := state.TotalOpenQuantity(); err == nil {
		t.Fatalf("expected corrupted fallback quantity total to fail")
	}
}

// TestScopeLocalHybridWrapsInjectedHelperFailures verifies direct wrapper
// branches through scope-local helper seams.
// Authored by: OpenCode
func TestScopeLocalHybridWrapsInjectedHelperFailures(t *testing.T) {
	var previousNewLotMethodState = scopeLocalNewLotMethodState
	var previousTotalOpenQuantity = scopeLocalLotTotalOpenQuantity
	var previousTotalOpenBasis = scopeLocalLotTotalOpenBasis
	var previousSubtract = scopeLocalSubtractDecimal
	defer func() {
		scopeLocalNewLotMethodState = previousNewLotMethodState
		scopeLocalLotTotalOpenQuantity = previousTotalOpenQuantity
		scopeLocalLotTotalOpenBasis = previousTotalOpenBasis
		scopeLocalSubtractDecimal = previousSubtract
	}()

	scopeLocalNewLotMethodState = func(LotMethod) (*LotMethodState, error) {
		return nil, errors.New("constructor boom")
	}
	if _, err := NewScopeLocalHybridState().ensureScopeState("scope-a"); err == nil || !strings.Contains(err.Error(), "constructor boom") {
		t.Fatalf("expected injected constructor failure, got %v", err)
	}

	scopeLocalNewLotMethodState = previousNewLotMethodState
	var invalid apd.Decimal
	invalid.Form = apd.NaNSignaling
	fallbackAddState := NewScopeLocalHybridState()
	fallbackAddState.scopes["scope-a"] = &scopeLocalOpenState{fallbackPool: &AverageCostState{quantity: invalid}}
	if err := fallbackAddState.AddAcquisition(ScopeLocalHybridAcquisition{
		SourceID:           "scope-a-fallback-add",
		ScopeKey:           "scope-a",
		AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		DeterministicOrder: 1,
		Quantity:           decimalFromInt(1),
		Basis:              decimalFromInt(1),
	}); err == nil {
		t.Fatalf("expected fallback acquisition to surface pool failure")
	}

	scopeLocalNewLotMethodState = func(LotMethod) (*LotMethodState, error) {
		return nil, errors.New("constructor boom through add")
	}
	if err := NewScopeLocalHybridState().AddAcquisition(ScopeLocalHybridAcquisition{
		SourceID:           "scope-a-new",
		ScopeKey:           "scope-a",
		AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		DeterministicOrder: 1,
		Quantity:           decimalFromInt(1),
		Basis:              decimalFromInt(1),
	}); err == nil || !strings.Contains(err.Error(), "constructor boom through add") {
		t.Fatalf("expected constructor failure through AddAcquisition, got %v", err)
	}
	scopeLocalNewLotMethodState = previousNewLotMethodState

	var state = NewScopeLocalHybridState()
	for _, acquisition := range []ScopeLocalHybridAcquisition{
		{
			SourceID:           "scope-a-lot-1",
			ScopeKey:           "scope-a",
			AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 1,
			Quantity:           decimalFromInt(1),
			Basis:              decimalFromInt(10),
		},
		{
			SourceID:           "scope-a-lot-2",
			ScopeKey:           "scope-a",
			AcquiredAt:         time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			DeterministicOrder: 2,
			Quantity:           decimalFromInt(1),
			Basis:              decimalFromInt(20),
		},
	} {
		if err := state.AddAcquisition(acquisition); err != nil {
			t.Fatalf("seed scope-local state %q: %v", acquisition.SourceID, err)
		}
	}

	scopeLocalLotTotalOpenQuantity = func(*LotMethodState) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("quantity boom")
	}
	if _, err := state.TotalOpenQuantity(); err == nil || !strings.Contains(err.Error(), "quantity boom") {
		t.Fatalf("expected injected total-open-quantity failure, got %v", err)
	}

	scopeLocalLotTotalOpenQuantity = previousTotalOpenQuantity
	scopeLocalLotTotalOpenBasis = func(*LotMethodState) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("basis boom")
	}
	if _, err := state.TotalOpenBasis(); err == nil || !strings.Contains(err.Error(), "basis boom") {
		t.Fatalf("expected injected total-open-basis failure, got %v", err)
	}

	scopeLocalLotTotalOpenBasis = previousTotalOpenBasis
	var exactState, err = NewLotMethodState(LotMethodFIFO)
	if err != nil {
		t.Fatalf("new FIFO exact state for exact disposal failure: %v", err)
	}
	if err = exactState.AddAcquisition(LotAcquisition{
		SourceID:           "scope-c-lot",
		AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		DeterministicOrder: 1,
		RemainingQuantity:  decimalFromInt(2),
		RemainingBasis:     decimalFromInt(10),
	}); err != nil {
		t.Fatalf("seed exact scope-local state: %v", err)
	}
	var previousLotSubtract = lotSubtractDecimal
	var lotSubtractCalls int
	lotSubtractDecimal = func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		lotSubtractCalls++
		if lotSubtractCalls == 1 {
			var invalidQuantity apd.Decimal
			invalidQuantity.Form = apd.NaNSignaling
			return invalidQuantity, nil
		}
		return previousLotSubtract(left, right)
	}
	state.scopes = map[string]*scopeLocalOpenState{
		"scope-c": {exactState: exactState},
	}
	if _, err = state.Dispose("scope-c", decimalFromInt(1)); err == nil {
		t.Fatalf("expected exact one-lot disposal to fail while reading remaining quantity")
	}
	lotSubtractDecimal = previousLotSubtract

	exactState, err = NewLotMethodState(LotMethodFIFO)
	if err != nil {
		t.Fatalf("new FIFO exact state for fallback activation failure: %v", err)
	}
	exactState.lots = []LotAcquisition{{
		SourceID:           "scope-d-lot-1",
		AcquiredAt:         time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		DeterministicOrder: 1,
		RemainingQuantity:  decimalFromInt(1),
		RemainingBasis:     invalid,
	}, {
		SourceID:           "scope-d-lot-2",
		AcquiredAt:         time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		DeterministicOrder: 2,
		RemainingQuantity:  decimalFromInt(1),
		RemainingBasis:     decimalFromInt(1),
	}}
	state.scopes = map[string]*scopeLocalOpenState{
		"scope-d": {exactState: exactState},
	}
	if _, err = state.Dispose("scope-d", decimalFromInt(1)); err == nil {
		t.Fatalf("expected fallback activation failure to propagate through Dispose")
	}

	state.scopes = map[string]*scopeLocalOpenState{
		"scope-e": {fallbackPool: &AverageCostState{quantity: decimalFromInt(1), basis: invalid}, provenanceLots: []scopeLocalProvenanceLot{{RemainingQuantity: decimalFromInt(1)}}},
	}
	if _, err = state.Dispose("scope-e", decimalFromInt(1)); err == nil {
		t.Fatalf("expected fallback pool disposal failure to propagate")
	}

	scopeLocalSubtractDecimal = func(apd.Decimal, apd.Decimal) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("subtract boom")
	}
	state.scopes = map[string]*scopeLocalOpenState{
		"scope-b": {
			fallbackPool: NewAverageCostState(),
			provenanceLots: []scopeLocalProvenanceLot{{
				RemainingQuantity: decimalFromInt(1),
			}},
		},
	}
	if err := state.scopes["scope-b"].fallbackPool.AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err != nil {
		t.Fatalf("seed fallback pool: %v", err)
	}
	if _, err := state.Dispose("scope-b", decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "subtract boom") {
		t.Fatalf("expected injected fallback subtraction failure, got %v", err)
	}

	scopeLocalSubtractDecimal = previousSubtract
	var subtractCalls int
	scopeLocalSubtractDecimal = func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		subtractCalls++
		if subtractCalls == 2 {
			return apd.Decimal{}, errors.New("subtract second boom")
		}
		return previousSubtract(left, right)
	}
	state.scopes = map[string]*scopeLocalOpenState{
		"scope-f": {
			fallbackPool: NewAverageCostState(),
			provenanceLots: []scopeLocalProvenanceLot{{
				RemainingQuantity: decimalFromInt(1),
			}},
		},
	}
	if err := state.scopes["scope-f"].fallbackPool.AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err != nil {
		t.Fatalf("seed fallback pool for second subtraction failure: %v", err)
	}
	if _, err := state.Dispose("scope-f", decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "subtract second boom") {
		t.Fatalf("expected injected second fallback subtraction failure, got %v", err)
	}
}
