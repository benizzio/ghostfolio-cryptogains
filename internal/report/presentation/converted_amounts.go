package presentation

import (
	"fmt"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// ConvertedAmountEntry contains the format-neutral fields of one visible
// original-to-converted amount pair. Renderers own the fixed syntax and
// format-specific escaping for these fields.
// Authored by: OpenCode
type ConvertedAmountEntry struct {
	Label           string
	OriginalAmount  string
	ConvertedAmount string
}

// ConvertedAmounts formats included converted amount evidence as ordered
// logical entries. Exact zero-to-zero pairs are omitted before formatting,
// while every other received entry is retained in sequence.
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
func ConvertedAmounts(entryIndex int, amounts []reportmodel.ConvertedActivityAmount) ([]ConvertedAmountEntry, error) {
	return ConvertedAmountsWithFinancialFormatting(entryIndex, amounts, DefaultFinancialFormattingOptions())
}

// ConvertedAmountsWithFinancialFormatting formats conversion components with a
// renderer-scoped policy after applying exact zero-to-zero omission.
// Authored by: OpenCode
func ConvertedAmountsWithFinancialFormatting(entryIndex int, amounts []reportmodel.ConvertedActivityAmount, options FinancialFormattingOptions) ([]ConvertedAmountEntry, error) {
	var entries []ConvertedAmountEntry
	for amountIndex, amount := range amounts {
		if amount.OriginalAmount.Sign() == 0 && amount.ConvertedAmount.Sign() == 0 {
			continue
		}

		original, err := options.Format(amount.OriginalAmount)
		if err != nil {
			return nil, fmt.Errorf("render conversion audit entry %d amount %d original amount: %w", entryIndex, amountIndex, err)
		}
		converted, err := options.Format(amount.ConvertedAmount)
		if err != nil {
			return nil, fmt.Errorf("render conversion audit entry %d amount %d converted amount: %w", entryIndex, amountIndex, err)
		}

		entries = append(entries, ConvertedAmountEntry{
			Label:           sanitize(string(amount.AmountKind)),
			OriginalAmount:  original,
			ConvertedAmount: converted,
		})
	}
	return entries, nil
}
