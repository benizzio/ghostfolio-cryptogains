// Package calculate owns exact report-domain amount conversion formulas.
// Authored by: OpenCode
package calculate

import (
	"fmt"

	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// convertAmountToBase converts one source-currency amount into the report base
// currency using canonical provider evidence quote semantics.
// Authored by: OpenCode
func convertAmountToBase(amount apd.Decimal, rate apd.Decimal, quoteDirection currencyintegration.QuoteDirection) (apd.Decimal, error) {
	if err := supportmath.RequireFinite(amount); err != nil {
		return apd.Decimal{}, fmt.Errorf("conversion amount is invalid: %w", err)
	}
	if err := supportmath.RequirePositive(rate); err != nil {
		return apd.Decimal{}, fmt.Errorf("conversion rate is invalid: %w", err)
	}
	switch quoteDirection {
	case currencyintegration.QuoteDirectionSourcePerBase:
		var converted, err = supportmath.DivideFiniteRoundHalfUp(decimalsupport.Clone(amount), decimalsupport.Clone(rate))
		return convertedAmountResult(converted, err, "convert source-per-base amount")
	case currencyintegration.QuoteDirectionBasePerSource:
		var converted, err = supportmath.Multiply(decimalsupport.Clone(amount), decimalsupport.Clone(rate))
		return convertedAmountResult(converted, err, "convert base-per-source amount")
	}

	return apd.Decimal{}, fmt.Errorf("conversion quote direction: unsupported quote direction %q", quoteDirection)
}

// convertedAmountResult adds report-domain context to lower-level decimal
// operation failures.
// Authored by: OpenCode
func convertedAmountResult(converted apd.Decimal, err error, operation string) (apd.Decimal, error) {
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("%s: %w", operation, err)
	}

	return converted, nil
}
