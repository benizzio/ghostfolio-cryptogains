// Package calculate defines the report-calculation currency-rate service seam.
// Authored by: OpenCode
package calculate

import (
	"context"

	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// CurrencyRateService is the narrow report-calculation dependency for canonical
// source-to-base exchange-rate evidence. Implementations may be backed by the
// currency integration service, but provider DTOs, provider URLs, and provider
// selection details must remain outside report calculation.
//
// Example:
//
//	service := currency.NewCurrencyRateService(currency.NewCurrencyRateSessionCache())
//	calculator := calculate.NewCalculator(service)
//	_, _ = calculator.Calculate(context.Background(), request, cache)
//
// Authored by: OpenCode
type CurrencyRateService interface {
	LookupRate(context.Context, currencyintegration.RateLookupRequest) (currencyintegration.ExchangeRateEvidence, error)
}

// Calculator keeps the dependencies required to calculate a capital-gains
// report. The currency-rate dependency is stored here so later conversion logic
// can resolve canonical evidence without changing runtime orchestration again.
// Authored by: OpenCode
type Calculator struct {
	currencyRates CurrencyRateService
}

// NewCalculator creates one report calculator with an optional currency-rate
// service. Passing nil preserves the current same-currency calculation behavior
// until conversion logic begins using the dependency.
//
// Example:
//
//	calculator := calculate.NewCalculator(nil)
//	report, err := calculator.Calculate(context.Background(), request, cache)
//	if err != nil {
//		panic(err)
//	}
//	_ = report.YearlyNetTotal
//
// Authored by: OpenCode
func NewCalculator(currencyRates CurrencyRateService) Calculator {
	return Calculator{currencyRates: currencyRates}
}

// Calculate builds one capital-gains report from the protected activity cache
// using the calculator's stored dependencies. The context is reserved for
// rate lookups through CurrencyRateService when cross-currency rows are present.
//
// Example:
//
//	calculator := calculate.NewCalculator(currencyService)
//	report, err := calculator.Calculate(context.Background(), request, cache)
//	if err != nil {
//		panic(err)
//	}
//	_ = report.ReportCalculationCurrency
//
// Authored by: OpenCode
func (calculator Calculator) Calculate(
	ctx context.Context,
	request reportmodel.ReportRequest,
	cache syncmodel.ProtectedActivityCache,
) (reportmodel.CapitalGainsReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	return calculateReport(ctx, calculator.currencyRates, request, cache)
}
