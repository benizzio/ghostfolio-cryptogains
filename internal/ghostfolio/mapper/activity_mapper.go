// Package mapper converts Ghostfolio transport DTOs into normalized activity
// records used by the sync pipeline.
// Authored by: OpenCode
package mapper

import (
	"errors"
	"fmt"
	"strings"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

type activityMoneyContext struct {
	orderCurrency         string
	assetProfileCurrency  string
	baseCurrency          string
	orderUnitPrice        *apd.Decimal
	orderGrossValue       *apd.Decimal
	orderFeeAmount        *apd.Decimal
	assetProfileUnitPrice *apd.Decimal
	assetProfileFeeAmount *apd.Decimal
	baseGrossValue        *apd.Decimal
	baseFeeAmount         *apd.Decimal
}

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
//	records, err := mapper.MapActivities(entries, "USD", decimal.NewService())
//	if err != nil {
//		panic(err)
//	}
//	_ = records
//
// Authored by: OpenCode
func MapActivities(entries []dto.ActivityPageEntry, baseCurrency string, decimalService decimalsupport.Service) ([]syncmodel.ActivityRecord, error) {
	var records = make([]syncmodel.ActivityRecord, 0, len(entries))
	for _, entry := range entries {
		record, err := MapActivity(entry, baseCurrency, decimalService)
		if err != nil {
			return nil, wrapMappingError(entry, baseCurrency, err)
		}
		records = append(records, record)
	}

	return records, nil
}

// wrapMappingError converts a raw mapping failure into diagnostic-capable context.
// Authored by: OpenCode
func wrapMappingError(entry dto.ActivityPageEntry, baseCurrency string, err error) error {
	var carrier interface {
		DiagnosticContext() syncmodel.DiagnosticContext
	}
	if ok := errorAsDiagnosticCarrier(err, &carrier); ok {
		var context = carrier.DiagnosticContext()
		if len(context.Records) == 0 {
			context.Records = []syncmodel.DiagnosticRecord{diagnosticRecordFromActivityEntry(entry, baseCurrency)}
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
			Records:       []syncmodel.DiagnosticRecord{diagnosticRecordFromActivityEntry(entry, baseCurrency)},
		},
	}
}

// diagnosticRecordFromActivityEntry captures the non-secret transport context for one offending source activity.
// Authored by: OpenCode
func diagnosticRecordFromActivityEntry(entry dto.ActivityPageEntry, baseCurrency string) syncmodel.DiagnosticRecord {
	var sourceScope = mapSourceScope(entry)
	var quantity = strings.TrimSpace(entry.Quantity.String())
	var record = syncmodel.DiagnosticRecord{
		SourceID:              strings.TrimSpace(entry.ID),
		OccurredAt:            strings.TrimSpace(entry.Date),
		ActivityType:          strings.ToUpper(strings.TrimSpace(entry.Type)),
		AssetSymbol:           strings.TrimSpace(entry.SymbolProfile.Symbol),
		AssetName:             strings.TrimSpace(entry.SymbolProfile.Name),
		OrderCurrency:         strings.TrimSpace(entry.Currency.String()),
		AssetProfileCurrency:  strings.TrimSpace(entry.SymbolProfile.Currency),
		BaseCurrency:          strings.TrimSpace(baseCurrency),
		Quantity:              quantity,
		OrderUnitPrice:        strings.TrimSpace(entry.UnitPrice.String()),
		OrderGrossValue:       strings.TrimSpace(entry.Value.String()),
		OrderFeeAmount:        strings.TrimSpace(entry.Fee.String()),
		AssetProfileUnitPrice: strings.TrimSpace(entry.UnitPriceInAssetProfileCurrency.String()),
		AssetProfileFeeAmount: strings.TrimSpace(entry.FeeInAssetProfileCurrency.String()),
		BaseGrossValue:        strings.TrimSpace(entry.ValueInBaseCurrency.String()),
		BaseFeeAmount:         strings.TrimSpace(entry.FeeInBaseCurrency.String()),
		Comment:               strings.TrimSpace(entry.Comment.String()),
		DataSource:            strings.TrimSpace(entry.DataSource),
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
	if !errors.As(err, &carrier) {
		return false
	}
	*target = carrier
	return true
}

// MapActivity converts one Ghostfolio activity entry into a normalized activity
// record for the sync pipeline.
//
// Example:
//
//	record, err := mapper.MapActivity(entry, "USD", decimal.NewService())
//	if err != nil {
//		panic(err)
//	}
//	_ = record.SourceID
//
// Authored by: OpenCode
func MapActivity(entry dto.ActivityPageEntry, baseCurrency string, decimalService decimalsupport.Service) (syncmodel.ActivityRecord, error) {
	if decimalService == nil {
		decimalService = decimalsupport.NewService()
	}
	var sourceID = strings.TrimSpace(entry.ID)
	var occurredAt = strings.TrimSpace(entry.Date)

	quantity, _, err := decimalService.ParseNumber(entry.Quantity)
	if err != nil {
		return syncmodel.ActivityRecord{}, fmt.Errorf("map activity quantity: %w", err)
	}

	moneyContext, err := parseMoneyContext(entry, baseCurrency, decimalService)
	if err != nil {
		return syncmodel.ActivityRecord{}, err
	}

	return syncmodel.ActivityRecord{
		SourceID:              sourceID,
		OccurredAt:            occurredAt,
		ActivityType:          syncmodel.ActivityType(strings.ToUpper(strings.TrimSpace(entry.Type))),
		AssetIdentityKey:      strings.TrimSpace(entry.SymbolProfile.ID),
		AssetSymbol:           strings.TrimSpace(entry.SymbolProfile.Symbol),
		AssetName:             strings.TrimSpace(entry.SymbolProfile.Name),
		Quantity:              quantity,
		OrderCurrency:         moneyContext.orderCurrency,
		OrderUnitPrice:        moneyContext.orderUnitPrice,
		OrderGrossValue:       moneyContext.orderGrossValue,
		OrderFeeAmount:        moneyContext.orderFeeAmount,
		AssetProfileCurrency:  moneyContext.assetProfileCurrency,
		AssetProfileUnitPrice: moneyContext.assetProfileUnitPrice,
		AssetProfileFeeAmount: moneyContext.assetProfileFeeAmount,
		BaseCurrency:          moneyContext.baseCurrency,
		BaseGrossValue:        moneyContext.baseGrossValue,
		BaseFeeAmount:         moneyContext.baseFeeAmount,
		Comment:               strings.TrimSpace(entry.Comment.String()),
		DataSource:            strings.TrimSpace(entry.DataSource),
		SourceScope:           mapSourceScope(entry),
	}, nil
}

// parseMoneyContext preserves every supported monetary amount together with the
// currency context required for storage and validation.
// Authored by: OpenCode
func parseMoneyContext(entry dto.ActivityPageEntry, baseCurrency string, decimalService decimalsupport.Service) (activityMoneyContext, error) {
	var context activityMoneyContext
	context.orderCurrency = strings.TrimSpace(entry.Currency.String())
	context.assetProfileCurrency = strings.TrimSpace(entry.SymbolProfile.Currency)
	context.baseCurrency = strings.TrimSpace(baseCurrency)

	var err error
	context.orderUnitPrice, err = parseOptionalNumber(entry.UnitPrice, decimalService)
	if err != nil {
		return activityMoneyContext{}, fmt.Errorf("map activity order unit price: %w", err)
	}
	context.orderGrossValue, err = parseOptionalNumber(entry.Value, decimalService)
	if err != nil {
		return activityMoneyContext{}, fmt.Errorf("map activity order gross value: %w", err)
	}
	context.orderFeeAmount, err = parseOptionalNumber(entry.Fee, decimalService)
	if err != nil {
		return activityMoneyContext{}, fmt.Errorf("map activity order fee: %w", err)
	}
	context.assetProfileUnitPrice, err = parseOptionalNumber(entry.UnitPriceInAssetProfileCurrency, decimalService)
	if err != nil {
		return activityMoneyContext{}, fmt.Errorf("map activity asset-profile unit price: %w", err)
	}
	context.assetProfileFeeAmount, err = parseOptionalNumber(entry.FeeInAssetProfileCurrency, decimalService)
	if err != nil {
		return activityMoneyContext{}, fmt.Errorf("map activity asset-profile fee: %w", err)
	}
	context.baseGrossValue, err = parseOptionalNumber(entry.ValueInBaseCurrency, decimalService)
	if err != nil {
		return activityMoneyContext{}, fmt.Errorf("map activity base-currency gross value: %w", err)
	}
	context.baseFeeAmount, err = parseOptionalNumber(entry.FeeInBaseCurrency, decimalService)
	if err != nil {
		return activityMoneyContext{}, fmt.Errorf("map activity base-currency fee: %w", err)
	}

	return context, nil
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
