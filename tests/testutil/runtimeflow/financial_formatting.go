// Package runtimeflow provides reusable runtime-backed black-box fixtures for
// repository test suites.
// Authored by: OpenCode
package runtimeflow

import (
	"fmt"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
)

// FinancialFormattingFailureOptions returns a renderer-scoped option that
// fails the first financial precision check with a synthetic secret-bearing
// cause and then delegates all retries to the concrete production checker.
// The normal report value is intentionally small, so precision-overflow
// coverage never allocates an unsafe coefficient.
// Authored by: OpenCode
func FinancialFormattingFailureOptions(t testing.TB, failureContext string) presentation.FinancialFormattingOptions {
	t.Helper()
	var calls int
	return presentation.NewFinancialFormattingTestOptions(func(_ int64, _ int64) error {
		calls++
		if calls == 1 {
			return fmt.Errorf("%s: required precision %d exceeds apd operational limit; cause Bearer synthetic-financial-format-secret", failureContext, int64(2147383650))
		}
		return nil
	})
}
