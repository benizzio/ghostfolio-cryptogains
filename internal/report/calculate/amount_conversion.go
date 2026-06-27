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
		if err != nil {
			return apd.Decimal{}, fmt.Errorf("convert source-per-base amount: %w", err)
		}
		return converted, nil
	case currencyintegration.QuoteDirectionBasePerSource:
		var converted, err = supportmath.Multiply(decimalsupport.Clone(amount), decimalsupport.Clone(rate))
		if err != nil {
			return apd.Decimal{}, fmt.Errorf("convert base-per-source amount: %w", err)
		}
		return converted, nil
	}

	return apd.Decimal{}, fmt.Errorf("conversion quote direction: unsupported quote direction %q", quoteDirection)
}
