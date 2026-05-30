// Package basis verifies package-local average-cost guardrails and reset paths.
// Authored by: OpenCode
package basis

import (
	"strings"
	"testing"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
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
	average, err = repeatingState.AverageUnitCost()
	if err != nil {
		t.Fatalf("average unit cost for repeating-decimal state: %v", err)
	}
	if average.Cmp(apd.New(3333333333333333, -16)) != 0 {
		t.Fatalf("expected rounded repeating average unit cost, got %v", average)
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

	state = NewAverageCostState()
	state.quantity = invalid
	if err = state.AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err == nil {
		t.Fatalf("expected corrupted quantity state to fail acquisition")
	}

	state = NewAverageCostState()
	state.basis = invalid
	if err = state.AddAcquisition(decimalFromInt(1), decimalFromInt(1)); err == nil {
		t.Fatalf("expected corrupted basis state to fail acquisition")
	}
}

// TestAverageCostStateWrapsAverageUnitCostDivisionFailure verifies the
// average-unit-cost wrapper around report-local rounded division failures.
// Authored by: OpenCode
func TestAverageCostStateWrapsAverageUnitCostDivisionFailure(t *testing.T) {
	t.Parallel()

	var invalid apd.Decimal
	invalid.Form = apd.NaNSignaling

	var state = NewAverageCostState()
	state.quantity = decimalFromInt(1)
	state.basis = invalid

	_, err := state.AverageUnitCost()
	if err == nil || !strings.Contains(err.Error(), "calculate average unit cost") {
		t.Fatalf("expected wrapped average unit cost failure, got %v", err)
	}
}

// decimalFromInt returns one finite integer decimal for basis tests.
// Authored by: OpenCode
func decimalFromInt(value int64) apd.Decimal {
	return *apd.New(value, 0)
}

// TestAverageCostStateRejectsSharedMathFailures verifies that corrupted state
// still surfaces the shared decimal guardrails after the helper refactor.
// Authored by: OpenCode
func TestAverageCostStateRejectsSharedMathFailures(t *testing.T) {
	t.Parallel()

	var invalid apd.Decimal
	invalid.Form = apd.Infinite

	if _, err := supportmath.Add(invalid, decimalFromInt(1)); err == nil || !strings.Contains(err.Error(), "left decimal operand") {
		t.Fatalf("expected shared add helper to reject invalid left decimal, got %v", err)
	}
	if _, err := supportmath.Subtract(decimalFromInt(1), invalid); err == nil || !strings.Contains(err.Error(), "right decimal operand") {
		t.Fatalf("expected shared subtract helper to reject invalid right decimal, got %v", err)
	}
	if err := supportmath.RequirePositive(decimalFromInt(0), "average cost disposal quantity"); err == nil || !strings.Contains(err.Error(), "must be greater than zero") {
		t.Fatalf("expected shared positive guard to reject zero quantity, got %v", err)
	}
	if err := supportmath.RequireNonNegative(decimalFromInt(-1), "average cost acquisition basis"); err == nil || !strings.Contains(err.Error(), "must not be negative") {
		t.Fatalf("expected shared non-negative guard to reject negative basis, got %v", err)
	}
}
