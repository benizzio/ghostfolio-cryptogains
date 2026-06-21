// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"fmt"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// ConvertAmountToBase converts one source-currency amount into the report base
// currency using canonical provider evidence quote semantics. Source-per-base
// rates divide the source amount by the rate and bound the quotient with the
// repository's 16-decimal round-half-up policy. Base-per-source rates multiply
// exactly because no recurring quotient is introduced.
//
// Example:
//
//	converted, err := currency.ConvertAmountToBase(amount, rate, currency.QuoteDirectionSourcePerBase)
//	if err != nil {
//		panic(err)
//	}
//	_ = converted
//
// Authored by: OpenCode
func ConvertAmountToBase(amount apd.Decimal, rate apd.Decimal, quoteDirection QuoteDirection) (apd.Decimal, error) {
	if err := supportmath.RequireFinite(amount); err != nil {
		return apd.Decimal{}, fmt.Errorf("conversion amount is invalid: %w", err)
	}
	if err := supportmath.RequirePositive(rate); err != nil {
		return apd.Decimal{}, fmt.Errorf("conversion rate is invalid: %w", err)
	}
	if err := validateQuoteDirection(quoteDirection); err != nil {
		return apd.Decimal{}, fmt.Errorf("conversion quote direction: %w", err)
	}

	switch quoteDirection {
	case QuoteDirectionSourcePerBase:
		var converted, err = supportmath.DivideFiniteRoundHalfUp(ratePreservingDecimal(amount), ratePreservingDecimal(rate))
		if err != nil {
			return apd.Decimal{}, fmt.Errorf("convert source-per-base amount: %w", err)
		}

		return converted, nil
	case QuoteDirectionBasePerSource:
		var converted, err = supportmath.Multiply(ratePreservingDecimal(amount), ratePreservingDecimal(rate))
		if err != nil {
			return apd.Decimal{}, fmt.Errorf("convert base-per-source amount: %w", err)
		}

		return converted, nil
	default:
		return apd.Decimal{}, fmt.Errorf("conversion quote direction: unsupported quote direction %q", quoteDirection)
	}
}

// ratePreservingDecimal returns a defensive copy before arithmetic mutates any
// decimal internals through pointer receivers.
// Authored by: OpenCode
func ratePreservingDecimal(value apd.Decimal) apd.Decimal {
	return cloneDecimal(value)
}
