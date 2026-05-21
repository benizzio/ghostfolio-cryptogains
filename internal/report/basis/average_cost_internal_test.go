// Package basis verifies package-local average-cost guardrails and reset paths.
// Authored by: OpenCode
package basis

import (
	"errors"
	"strings"
	"testing"

	"github.com/cockroachdb/apd/v3"
)

// TestAverageCostStateNilAndResetPaths verifies nil receiver guardrails, zero
// average handling, and full-disposal reset behavior.
// Authored by: OpenCode
func TestAverageCostStateNilAndResetPaths(t *testing.T) {
	var nilState *AverageCostState

	if err := nilState.AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err == nil {
		t.Fatalf("expected nil average-cost state acquisition to fail")
	}
	if _, err := nilState.Dispose(decimalFromInt(1)); err == nil {
		t.Fatalf("expected nil average-cost state disposal to fail")
	}
	if _, err := nilState.AverageUnitCost(); err == nil {
		t.Fatalf("expected nil average-cost unit-cost lookup to fail")
	}
	var nilQuantity = nilState.Quantity()
	if nilQuantity.Cmp(apd.New(0, 0)) != 0 {
		t.Fatalf("expected nil average-cost quantity to be zero")
	}
	var nilBasis = nilState.Basis()
	if nilBasis.Cmp(apd.New(0, 0)) != 0 {
		t.Fatalf("expected nil average-cost basis to be zero")
	}
	if !nilState.IsEmpty() {
		t.Fatalf("expected nil average-cost state to report empty")
	}

	var state = NewAverageCostState()
	var average, err = state.AverageUnitCost()
	if err != nil {
		t.Fatalf("average unit cost for empty state: %v", err)
	}
	if average.Sign() != 0 {
		t.Fatalf("expected empty average-cost state to report zero unit cost")
	}

	var repeatingState = NewAverageCostState()
	if err = repeatingState.AddAcquisition(decimalFromInt(3), decimalFromInt(1)); err != nil {
		t.Fatalf("add repeating-decimal acquisition: %v", err)
	}
	_, err = repeatingState.AverageUnitCost()
	if err == nil || !strings.Contains(err.Error(), "calculate average unit cost exactly") {
		t.Fatalf("expected repeating-decimal average-cost failure, got %v", err)
	}

	if err = state.AddAcquisition(decimalFromInt(2), decimalFromInt(10)); err != nil {
		t.Fatalf("add acquisition: %v", err)
	}
	_, err = state.Dispose(decimalFromInt(3))
	if err == nil || !strings.Contains(err.Error(), "exceeds open pool quantity") {
		t.Fatalf("expected oversized average-cost disposal to fail, got %v", err)
	}

	var disposal, disposeErr = state.Dispose(decimalFromInt(2))
	if disposeErr != nil {
		t.Fatalf("dispose full average-cost position: %v", disposeErr)
	}
	if disposal.RemainingQuantity.Sign() != 0 || disposal.RemainingBasis.Sign() != 0 {
		t.Fatalf("expected full disposal to reset remaining pool, got quantity=%v basis=%v", disposal.RemainingQuantity, disposal.RemainingBasis)
	}
	var stateQuantity = state.Quantity()
	var stateBasis = state.Basis()
	if stateQuantity.Cmp(apd.New(0, 0)) != 0 || stateBasis.Cmp(apd.New(0, 0)) != 0 || !state.IsEmpty() {
		t.Fatalf("expected average-cost state to reset after full disposal")
	}

	_, err = state.Dispose(decimalFromInt(1))
	if err == nil || !strings.Contains(err.Error(), "exceeds open pool quantity") {
		t.Fatalf("expected empty-pool disposal failure, got %v", err)
	}
}

// TestAverageCostStateValidationPaths verifies input validation guardrails for
// average-cost state operations.
// Authored by: OpenCode
func TestAverageCostStateValidationPaths(t *testing.T) {
	var state = NewAverageCostState()

	if err := state.AddAcquisition(decimalFromInt(0), decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "average cost acquisition quantity must be greater than zero") {
		t.Fatalf("expected zero-quantity acquisition to fail, got %v", err)
	}
	if err := state.AddAcquisition(decimalFromInt(1), decimalFromInt(-1)); err == nil || !strings.Contains(err.Error(), "average cost acquisition basis must not be negative") {
		t.Fatalf("expected negative-basis acquisition to fail, got %v", err)
	}
	if _, err := state.Dispose(decimalFromInt(0)); err == nil || !strings.Contains(err.Error(), "average cost disposal quantity must be greater than zero") {
		t.Fatalf("expected zero-quantity disposal to fail, got %v", err)
	}
}

// TestAverageCostStatePartialDisposalAndFiniteValidation verifies the
// non-reset disposal branch and finite-value guardrails.
// Authored by: OpenCode
func TestAverageCostStatePartialDisposalAndFiniteValidation(t *testing.T) {
	var state = NewAverageCostState()
	if err := state.AddAcquisition(decimalFromInt(4), decimalFromInt(20)); err != nil {
		t.Fatalf("add acquisition: %v", err)
	}

	var average, averageErr = state.AverageUnitCost()
	if averageErr != nil || average.Cmp(apd.New(5, 0)) != 0 {
		t.Fatalf("expected finite average unit cost, got %v err=%v", average, averageErr)
	}

	var disposal, err = state.Dispose(decimalFromInt(1))
	if err != nil {
		t.Fatalf("dispose one average-cost unit: %v", err)
	}
	if disposal.DisposedQuantity.Cmp(apd.New(1, 0)) != 0 || disposal.AllocatedBasis.Cmp(apd.New(5, 0)) != 0 || disposal.RemainingQuantity.Cmp(apd.New(3, 0)) != 0 || disposal.RemainingBasis.Cmp(apd.New(15, 0)) != 0 {
		t.Fatalf("unexpected partial average-cost disposal: %#v", disposal)
	}

	var invalid apd.Decimal
	invalid.Form = apd.NaNSignaling
	if err = state.AddAcquisition(invalid, decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "average cost acquisition quantity") {
		t.Fatalf("expected non-finite acquisition quantity to fail, got %v", err)
	}
	if _, err = state.Dispose(invalid); err == nil || !strings.Contains(err.Error(), "average cost disposal quantity") {
		t.Fatalf("expected non-finite disposal quantity to fail, got %v", err)
	}

	state.quantity = invalid
	if _, err = state.Dispose(decimalFromInt(1)); err == nil {
		t.Fatalf("expected corrupted quantity state to fail disposal")
	}

	state = NewAverageCostState()
	if err = state.AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err != nil {
		t.Fatalf("seed average-cost state: %v", err)
	}
	state.basis = invalid
	if _, err = state.Dispose(decimalFromInt(1)); err == nil {
		t.Fatalf("expected corrupted basis state to fail disposal")
	}
}

// TestAverageCostStateWrapsInjectedDecimalFailures verifies direct wrapper
// branches through average-cost decimal-operation seams.
// Authored by: OpenCode
func TestAverageCostStateWrapsInjectedDecimalFailures(t *testing.T) {
	var previousAdd = averageCostAddDecimal
	var previousSubtract = averageCostSubtractDecimal
	defer func() {
		averageCostAddDecimal = previousAdd
		averageCostSubtractDecimal = previousSubtract
	}()

	averageCostAddDecimal = func(apd.Decimal, apd.Decimal) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("add boom")
	}
	if err := NewAverageCostState().AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "add boom") {
		t.Fatalf("expected injected add failure, got %v", err)
	}

	averageCostAddDecimal = previousAdd
	averageCostSubtractDecimal = func(apd.Decimal, apd.Decimal) (apd.Decimal, error) {
		return apd.Decimal{}, errors.New("subtract boom")
	}

	var state = NewAverageCostState()
	if err := state.AddAcquisition(decimalFromInt(2), decimalFromInt(10)); err != nil {
		t.Fatalf("seed average-cost state: %v", err)
	}
	if _, err := state.Dispose(decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "subtract boom") {
		t.Fatalf("expected injected subtract failure, got %v", err)
	}

	averageCostSubtractDecimal = previousSubtract
	var addCalls int
	averageCostAddDecimal = func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		addCalls++
		if addCalls == 2 {
			return apd.Decimal{}, errors.New("add second boom")
		}
		return previousAdd(left, right)
	}
	if err := NewAverageCostState().AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "add second boom") {
		t.Fatalf("expected injected second add failure, got %v", err)
	}

	averageCostAddDecimal = previousAdd
	var subtractCalls int
	averageCostSubtractDecimal = func(left apd.Decimal, right apd.Decimal) (apd.Decimal, error) {
		subtractCalls++
		if subtractCalls == 2 {
			return apd.Decimal{}, errors.New("subtract second boom")
		}
		return previousSubtract(left, right)
	}

	state = NewAverageCostState()
	if err := state.AddAcquisition(decimalFromInt(2), decimalFromInt(10)); err != nil {
		t.Fatalf("seed average-cost state for second subtract failure: %v", err)
	}
	if _, err := state.Dispose(decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "subtract second boom") {
		t.Fatalf("expected injected second subtract failure, got %v", err)
	}
}

// decimalFromInt returns one finite integer decimal for basis tests.
// Authored by: OpenCode
func decimalFromInt(value int64) apd.Decimal {
	return *apd.New(value, 0)
}
