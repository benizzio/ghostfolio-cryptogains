// Package runtimeapp provides runtime assembly helpers for tests that exercise
// application services through the production runtime package.
// Authored by: OpenCode
package runtimeapp

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
)

// NewWithReportCurrencyRateService creates a runtime app for tests that need
// deterministic report currency-rate behavior.
// Authored by: OpenCode
func NewWithReportCurrencyRateService(
	t testing.TB,
	options bootstrap.Options,
	currencyRates reportcalculate.CurrencyRateService,
) *runtime.App {
	t.Helper()

	var app, err = runtime.NewWithReportCurrencyRateService(options, currencyRates)
	if err != nil {
		t.Fatalf("runtime new: %v", err)
	}

	return app
}
