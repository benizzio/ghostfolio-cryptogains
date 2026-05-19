// Package testutil provides shared helpers for the repository's black-box test
// suites so unit and integration tests can reuse the same Bubble Tea command
// execution behavior.
// Authored by: OpenCode
package testutil

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// RunCmd executes one Bubble Tea command and returns its resulting message.
//
// It keeps tests focused on workflow state changes without repeating the nil
// command guard at each call site.
//
// Example usage:
//
//	updated, cmd := model.Update(msg)
//	result := testutil.RunCmd(cmd)
//	updated, _ = model.Update(result)
//
// Authored by: OpenCode
func RunCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// GhostfolioUserBody returns one authenticated-user response body with an
// optional base-currency field.
//
// Authored by: OpenCode
func GhostfolioUserBody(baseCurrency string) string {
	if strings.TrimSpace(baseCurrency) == "" {
		return `{"settings":{}}`
	}

	return fmt.Sprintf(`{"settings":{"baseCurrency":%q}}`, strings.TrimSpace(baseCurrency))
}

// GhostfolioNullableOrderCurrencyActivityJSON returns one mixed-currency
// activity whose order-currency tier is uninformed.
//
// Authored by: OpenCode
func GhostfolioNullableOrderCurrencyActivityJSON() string {
	return `{"id":"buy-null-order-currency","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"currency":null,"unitPrice":90,"value":90,"fee":2,"feeInAssetProfileCurrency":1.8,"valueInBaseCurrency":100,"feeInBaseCurrency":2.2,"unitPriceInAssetProfileCurrency":95,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"EUR"}}`
}

// GhostfolioMissingSymbolProfileCurrencyActivityJSON returns one mixed-currency
// activity whose asset-profile currency tier is uninformed.
//
// Authored by: OpenCode
func GhostfolioMissingSymbolProfileCurrencyActivityJSON() string {
	return `{"id":"buy-missing-symbol-profile-currency","date":"2024-01-02T10:00:00Z","type":"BUY","quantity":1,"currency":"CHF","unitPrice":90,"value":90,"fee":2,"feeInAssetProfileCurrency":1.8,"valueInBaseCurrency":100,"feeInBaseCurrency":2.2,"unitPriceInAssetProfileCurrency":95,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}`
}

// GhostfolioAllTierUninformedCurrencyActivityJSON returns one activity whose
// preserved monetary data remains uninformed across all tracked currency tiers.
//
// Authored by: OpenCode
func GhostfolioAllTierUninformedCurrencyActivityJSON() string {
	return `{"id":"buy-all-tiers-uninformed","date":"2024-01-03T10:00:00Z","type":"BUY","quantity":1,"currency":null,"unitPrice":90,"value":90,"fee":2,"feeInAssetProfileCurrency":1.8,"valueInBaseCurrency":100,"feeInBaseCurrency":2.2,"unitPriceInAssetProfileCurrency":95,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}`
}
