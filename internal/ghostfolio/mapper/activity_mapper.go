// Package mapper converts Ghostfolio transport DTOs into normalized activity
// records used by the sync pipeline.
// Authored by: OpenCode
package mapper

import (
	"fmt"
	"strings"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// MapActivities converts a page of Ghostfolio activity entries into normalized
// activity records for the sync pipeline.
//
// Example:
//
//	records, err := mapper.MapActivities(entries, decimal.NewService())
//	if err != nil {
//		panic(err)
//	}
//	_ = records
//
// Authored by: OpenCode
func MapActivities(entries []dto.ActivityPageEntry, decimalService decimalsupport.Service) ([]syncmodel.ActivityRecord, error) {
	var records = make([]syncmodel.ActivityRecord, 0, len(entries))
	for _, entry := range entries {
		record, err := MapActivity(entry, decimalService)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// MapActivity converts one Ghostfolio activity entry into a normalized activity
// record for the sync pipeline.
//
// Example:
//
//	record, err := mapper.MapActivity(entry, decimal.NewService())
//	if err != nil {
//		panic(err)
//	}
//	_ = record.SourceID
//
// Authored by: OpenCode
func MapActivity(entry dto.ActivityPageEntry, decimalService decimalsupport.Service) (syncmodel.ActivityRecord, error) {
	if decimalService == nil {
		decimalService = decimalsupport.NewService()
	}

	quantity, _, err := decimalService.ParseNumber(entry.Quantity)
	if err != nil {
		return syncmodel.ActivityRecord{}, fmt.Errorf("map activity quantity: %w", err)
	}

	unitPrice, _, err := decimalService.ParseNumber(entry.UnitPriceInAssetProfileCurrency)
	if err != nil {
		return syncmodel.ActivityRecord{}, fmt.Errorf("map activity unit price: %w", err)
	}

	grossValue, err := parseGrossValue(entry, decimalService)
	if err != nil {
		return syncmodel.ActivityRecord{}, err
	}

	feeAmount, err := parseOptionalNumber(entry.FeeInBaseCurrency, decimalService)
	if err != nil {
		return syncmodel.ActivityRecord{}, fmt.Errorf("map activity fee: %w", err)
	}

	return syncmodel.ActivityRecord{
		SourceID:     entry.ID,
		OccurredAt:   entry.Date,
		ActivityType: syncmodel.ActivityType(strings.ToUpper(strings.TrimSpace(entry.Type))),
		AssetSymbol:  strings.TrimSpace(entry.SymbolProfile.Symbol),
		AssetName:    strings.TrimSpace(entry.SymbolProfile.Name),
		BaseCurrency: strings.TrimSpace(entry.BaseCurrency),
		Quantity:     quantity,
		UnitPrice:    unitPrice,
		GrossValue:   grossValue,
		FeeAmount:    feeAmount,
		Comment:      strings.TrimSpace(entry.Comment),
		DataSource:   strings.TrimSpace(entry.DataSource),
		SourceScope:  mapSourceScope(entry),
	}, nil
}

// parseGrossValue selects the supported gross-value field for one activity.
// Authored by: OpenCode
func parseGrossValue(entry dto.ActivityPageEntry, decimalService decimalsupport.Service) (apd.Decimal, error) {
	if strings.TrimSpace(entry.ValueInBaseCurrency.String()) != "" {
		value, _, err := decimalService.ParseNumber(entry.ValueInBaseCurrency)
		if err != nil {
			return apd.Decimal{}, fmt.Errorf("map activity gross value: %w", err)
		}
		return value, nil
	}

	value, _, err := decimalService.ParseNumber(entry.Value)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("map activity gross value: %w", err)
	}

	return value, nil
}

// parseOptionalNumber parses one optional decimal number when it is present.
// Authored by: OpenCode
func parseOptionalNumber(rawValue interface{ String() string }, decimalService decimalsupport.Service) (*apd.Decimal, error) {
	if strings.TrimSpace(rawValue.String()) == "" {
		return nil, nil
	}

	value, _, err := decimalService.ParseString(rawValue.String())
	if err != nil {
		return nil, err
	}

	return &value, nil
}

// mapSourceScope preserves optional Ghostfolio account metadata on normalized activities.
// Authored by: OpenCode
func mapSourceScope(entry dto.ActivityPageEntry) *syncmodel.SourceScope {
	if entry.Account == nil || strings.TrimSpace(entry.Account.ID) == "" {
		return nil
	}

	return &syncmodel.SourceScope{
		ID:          strings.TrimSpace(entry.Account.ID),
		Name:        strings.TrimSpace(entry.Account.Name),
		Kind:        syncmodel.SourceScopeKindAccount,
		Reliability: syncmodel.ScopeReliabilityReliable,
	}
}
