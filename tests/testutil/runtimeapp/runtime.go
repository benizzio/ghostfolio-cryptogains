// Package runtimeapp provides runtime assembly helpers for tests that exercise
// application services through the production runtime package.
// Authored by: OpenCode
package runtimeapp

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
)

// NewWithReportCurrencyRateService creates a runtime app for tests that need
// deterministic report currency-rate behavior.
// Authored by: OpenCode
func NewWithReportCurrencyRateService(
	t testing.TB,
	options bootstrap.Options,
	currencyRates reportcalculate.CurrencyRateService,
) *runtime.App {
	return NewWithReportCurrencyRateServiceAndPDFByteFinalizer(t, options, currencyRates, nil)
}

// NewWithReportCurrencyRateServiceAndPDFByteFinalizer creates a deterministic
// test runtime with one renderer-scoped PDF byte-finalizer option.
// Authored by: OpenCode
func NewWithReportCurrencyRateServiceAndPDFByteFinalizer(
	t testing.TB,
	options bootstrap.Options,
	currencyRates reportcalculate.CurrencyRateService,
	finalizer reportpdf.ByteFinalizer,
) *runtime.App {
	t.Helper()

	var app, err = runtime.NewWithReportCurrencyRateServiceAndPDFByteFinalizer(options, currencyRates, finalizer)
	if err != nil {
		t.Fatalf("runtime new: %v", err)
	}

	return app
}
