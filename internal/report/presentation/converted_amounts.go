package presentation

import (
	"fmt"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// ConvertedAmounts formats included converted amount evidence as ordered,
// delimiter-free logical entries. Exact zero-to-zero pairs are omitted before
// formatting, while every other received entry is retained in sequence.
//
// Example:
//
//	entries, err := presentation.ConvertedAmounts(0, amounts)
//	if err != nil {
//		// Handle a component-specific presentation error.
//	}
//	_ = entries
//
// Authored by: OpenCode
func ConvertedAmounts(entryIndex int, amounts []reportmodel.ConvertedActivityAmount) ([]string, error) {
	var entries []string
	for amountIndex, amount := range amounts {
		if amount.OriginalAmount.Sign() == 0 && amount.ConvertedAmount.Sign() == 0 {
			continue
		}

		original, err := formatFinancialValue(amount.OriginalAmount)
		if err != nil {
			return nil, fmt.Errorf("render conversion audit entry %d amount %d original amount: %w", entryIndex, amountIndex, err)
		}
		converted, err := formatFinancialValue(amount.ConvertedAmount)
		if err != nil {
			return nil, fmt.Errorf("render conversion audit entry %d amount %d converted amount: %w", entryIndex, amountIndex, err)
		}

		entries = append(entries, fmt.Sprintf("%s: %s -> %s", sanitize(string(amount.AmountKind)), original, converted))
	}
	return entries, nil
}
