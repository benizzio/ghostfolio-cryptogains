// Package dto defines the Ghostfolio transport models required by the sync and
// protected-storage slices.
// Authored by: OpenCode
package dto

import "encoding/json"

// ActivityPageResponse is the successful paginated activities response required
// by the full-history sync slice.
//
// Authored by: OpenCode
type ActivityPageResponse struct {
	Activities []ActivityPageEntry `json:"activities"`
	Count      int                 `json:"count"`
}

// ActivityPageEntry is the paginated Ghostfolio activity shape required for
// normalized full-history sync.
//
// Authored by: OpenCode
type ActivityPageEntry struct {
	ID                              string                `json:"id"`
	Date                            string                `json:"date"`
	Type                            string                `json:"type"`
	Quantity                        json.Number           `json:"quantity"`
	Value                           json.Number           `json:"value"`
	ValueInBaseCurrency             json.Number           `json:"valueInBaseCurrency"`
	FeeInBaseCurrency               json.Number           `json:"feeInBaseCurrency"`
	UnitPriceInAssetProfileCurrency json.Number           `json:"unitPriceInAssetProfileCurrency"`
	Comment                         string                `json:"comment"`
	SymbolProfile                   ActivitySymbolProfile `json:"SymbolProfile"`
	Account                         *ActivityAccountScope `json:"account"`
	DataSource                      string                `json:"dataSource"`
	BaseCurrency                    string                `json:"baseCurrency"`
}

// ActivitySymbolProfile preserves the minimum asset identity metadata required
// for normalization.
// Authored by: OpenCode
type ActivitySymbolProfile struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// ActivityAccountScope preserves optional source account metadata that may
// later inform scope reliability.
// Authored by: OpenCode
type ActivityAccountScope struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
