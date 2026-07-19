// Package runtimeflow provides reusable runtime-backed black-box fixtures for
// repository test suites.
// Authored by: OpenCode
package runtimeflow

import (
	"fmt"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/report/presentation"
)

// RequiredPrecisionFailureOptions returns a renderer-scoped option that fails
// the first financial precision check with a synthetic secret-bearing cause and
// then delegates retries to the concrete production checker. It models only the
// unsafe coefficient size needed to exceed apd's precision limit.
// Authored by: OpenCode
func RequiredPrecisionFailureOptions(t testing.TB) presentation.FinancialFormattingOptions {
	t.Helper()
	var calls int
	return presentation.NewFinancialFormattingTestOptions(func(_ int64, _ int64) error {
		calls++
		if calls == 1 {
			return fmt.Errorf("required precision %d exceeds apd operational limit; cause Bearer synthetic-financial-format-secret", int64(2147383650))
		}
		return nil
	})
}
