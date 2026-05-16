// Package mapper converts Ghostfolio transport DTOs into normalized activity
// records used by the sync pipeline.
// Authored by: OpenCode
package mapper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// MappingError stores offending-record context for synced-data mapping failures.
// Authored by: OpenCode
type MappingError struct {
	message string
	context syncmodel.DiagnosticContext
}

// Error returns the non-secret mapping failure detail.
// Authored by: OpenCode
func (e *MappingError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

// DiagnosticContext returns the offending-record context for one mapping failure.
// Authored by: OpenCode
func (e *MappingError) DiagnosticContext() syncmodel.DiagnosticContext {
	if e == nil {
		return syncmodel.DiagnosticContext{}
	}
	return e.context
}

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
			return nil, wrapMappingError(entry, err)
		}
		records = append(records, record)
	}

	return records, nil
}

// wrapMappingError converts a raw mapping failure into diagnostic-capable context.
// Authored by: OpenCode
func wrapMappingError(entry dto.ActivityPageEntry, err error) error {
	var carrier interface {
		DiagnosticContext() syncmodel.DiagnosticContext
	}
	if ok := errorAsDiagnosticCarrier(err, &carrier); ok {
		var context = carrier.DiagnosticContext()
		if len(context.Records) == 0 {
			context.Records = []syncmodel.DiagnosticRecord{diagnosticRecordFromActivityEntry(entry)}
		}
		if context.FailureStage == "" {
			context.FailureStage = syncmodel.DiagnosticFailureStageMapping
		}
		if context.FailureDetail == "" {
			context.FailureDetail = err.Error()
		}
		return &MappingError{message: err.Error(), context: context}
	}

	return &MappingError{
		message: err.Error(),
		context: syncmodel.DiagnosticContext{
			FailureStage:  syncmodel.DiagnosticFailureStageMapping,
			FailureDetail: err.Error(),
			Records:       []syncmodel.DiagnosticRecord{diagnosticRecordFromActivityEntry(entry)},
		},
	}
}

// diagnosticRecordFromActivityEntry captures the non-secret transport context for one offending source activity.
// Authored by: OpenCode
func diagnosticRecordFromActivityEntry(entry dto.ActivityPageEntry) syncmodel.DiagnosticRecord {
	var sourceScope = mapSourceScope(entry)
	var record = syncmodel.DiagnosticRecord{
		SourceID:     strings.TrimSpace(entry.ID),
		OccurredAt:   strings.TrimSpace(entry.Date),
		ActivityType: strings.ToUpper(strings.TrimSpace(entry.Type)),
		AssetSymbol:  strings.TrimSpace(entry.SymbolProfile.Symbol),
		AssetName:    strings.TrimSpace(entry.SymbolProfile.Name),
		BaseCurrency: strings.TrimSpace(entry.BaseCurrency),
		Quantity:     strings.TrimSpace(entry.Quantity.String()),
		UnitPrice:    strings.TrimSpace(entry.UnitPriceInAssetProfileCurrency.String()),
		GrossValue:   strings.TrimSpace(selectGrossValue(entry).String()),
		FeeAmount:    strings.TrimSpace(entry.FeeInBaseCurrency.String()),
		Comment:      strings.TrimSpace(entry.Comment),
		DataSource:   strings.TrimSpace(entry.DataSource),
	}
	if sourceScope != nil {
		record.SourceScopeID = sourceScope.ID
		record.SourceScopeName = sourceScope.Name
		record.SourceScopeKind = string(sourceScope.Kind)
		record.SourceScopeReliability = string(sourceScope.Reliability)
	}

	return record
}

// errorAsDiagnosticCarrier keeps compile-time compatibility local to this package.
// Authored by: OpenCode
func errorAsDiagnosticCarrier(err error, target *interface {
	DiagnosticContext() syncmodel.DiagnosticContext
}) bool {
	if err == nil {
		return false
	}
	var carrier interface {
		DiagnosticContext() syncmodel.DiagnosticContext
	}
	typed, ok := err.(interface {
		DiagnosticContext() syncmodel.DiagnosticContext
	})
	if !ok {
		return false
	}
	carrier = typed
	*target = carrier
	return true
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
	value, _, err := decimalService.ParseNumber(selectGrossValue(entry))
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("map activity gross value: %w", err)
	}

	return value, nil
}

// selectGrossValue applies the shared DTO gross-value fallback rule used by
// both activity and diagnostic mapping.
// Authored by: OpenCode
func selectGrossValue(entry dto.ActivityPageEntry) json.Number {
	if strings.TrimSpace(entry.ValueInBaseCurrency.String()) != "" {
		return entry.ValueInBaseCurrency
	}

	return entry.Value
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
