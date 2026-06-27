// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// parsePositiveRate parses one positive exact provider rate without float math.
// Authored by: OpenCode
func parsePositiveRate(rawRate string) (apd.Decimal, error) {
	var rate, _, err = decimalsupport.ParseString(rawRate)
	if err != nil {
		return apd.Decimal{}, err
	}
	if err = supportmath.RequirePositive(rate); err != nil {
		return apd.Decimal{}, err
	}

	return rate, nil
}
