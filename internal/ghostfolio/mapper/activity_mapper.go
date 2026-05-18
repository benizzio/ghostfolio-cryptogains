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
		UnitPrice:             diagnosticPreferredUnitPrice(entry, quantity),
		UnitPriceCurrency:     diagnosticPreferredUnitPriceCurrency(entry, baseCurrency),
		GrossValue:            diagnosticPreferredGrossValue(entry, quantity),
		GrossValueCurrency:    diagnosticPreferredGrossValueCurrency(entry, baseCurrency),
		FeeAmount:             diagnosticPreferredFeeAmount(entry),
		FeeAmountCurrency:     diagnosticPreferredFeeAmountCurrency(entry, baseCurrency),
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

// diagnosticPreferredUnitPrice returns the current-slice transient unit-price
// view used only in mapper diagnostic context.
// Authored by: OpenCode
func diagnosticPreferredUnitPrice(entry dto.ActivityPageEntry, quantity string) string {
	if raw := strings.TrimSpace(entry.UnitPrice.String()); raw != "" {
		return raw
	}
	if raw := strings.TrimSpace(entry.UnitPriceInAssetProfileCurrency.String()); raw != "" {
		return raw
	}

	return diagnosticDerivedUnitPrice(quantity, diagnosticPreferredGrossValue(entry, quantity))
}

// diagnosticPreferredUnitPriceCurrency returns the transient unit-price currency
// used only in mapper diagnostic context.
// Authored by: OpenCode
func diagnosticPreferredUnitPriceCurrency(entry dto.ActivityPageEntry, baseCurrency string) string {
	if strings.TrimSpace(entry.UnitPrice.String()) != "" {
		return strings.TrimSpace(entry.Currency.String())
	}
	if strings.TrimSpace(entry.UnitPriceInAssetProfileCurrency.String()) != "" {
		return strings.TrimSpace(entry.SymbolProfile.Currency)
	}

	return diagnosticPreferredGrossValueCurrency(entry, baseCurrency)
}

// diagnosticPreferredGrossValue returns the current-slice transient gross-value
// view used only in mapper diagnostic context.
// Authored by: OpenCode
func diagnosticPreferredGrossValue(entry dto.ActivityPageEntry, quantity string) string {
	if raw := strings.TrimSpace(entry.Value.String()); raw != "" {
		return raw
	}
	if raw := strings.TrimSpace(entry.ValueInBaseCurrency.String()); raw != "" {
		return raw
	}
	if raw := strings.TrimSpace(entry.UnitPriceInAssetProfileCurrency.String()); raw != "" {
		return diagnosticDerivedGrossValue(quantity, raw)
	}

	return diagnosticDerivedGrossValue(quantity, strings.TrimSpace(entry.UnitPrice.String()))
}

// diagnosticPreferredGrossValueCurrency returns the transient gross-value
// currency used only in mapper diagnostic context.
// Authored by: OpenCode
func diagnosticPreferredGrossValueCurrency(entry dto.ActivityPageEntry, baseCurrency string) string {
	if strings.TrimSpace(entry.Value.String()) != "" {
		return strings.TrimSpace(entry.Currency.String())
	}
	if strings.TrimSpace(entry.ValueInBaseCurrency.String()) != "" {
		return strings.TrimSpace(baseCurrency)
	}
	if strings.TrimSpace(entry.UnitPriceInAssetProfileCurrency.String()) != "" {
		return strings.TrimSpace(entry.SymbolProfile.Currency)
	}

	return strings.TrimSpace(entry.Currency.String())
}

// diagnosticPreferredFeeAmount returns the transient fee view used only in
// mapper diagnostic context.
// Authored by: OpenCode
func diagnosticPreferredFeeAmount(entry dto.ActivityPageEntry) string {
	if raw := strings.TrimSpace(entry.Fee.String()); raw != "" {
		return raw
	}
	if raw := strings.TrimSpace(entry.FeeInAssetProfileCurrency.String()); raw != "" {
		return raw
	}

	return strings.TrimSpace(entry.FeeInBaseCurrency.String())
}

// diagnosticPreferredFeeAmountCurrency returns the transient fee currency used
// only in mapper diagnostic context.
// Authored by: OpenCode
func diagnosticPreferredFeeAmountCurrency(entry dto.ActivityPageEntry, baseCurrency string) string {
	if strings.TrimSpace(entry.Fee.String()) != "" {
		return strings.TrimSpace(entry.Currency.String())
	}
	if strings.TrimSpace(entry.FeeInAssetProfileCurrency.String()) != "" {
		return strings.TrimSpace(entry.SymbolProfile.Currency)
	}

	return strings.TrimSpace(baseCurrency)
}

// diagnosticDerivedUnitPrice preserves exact-decimal unit-price derivation for
// mapper diagnostics without storing the derived value in the persisted model.
// Authored by: OpenCode
func diagnosticDerivedUnitPrice(quantity string, grossValue string) string {
	if strings.TrimSpace(quantity) == "" || strings.TrimSpace(grossValue) == "" {
		return ""
	}

	var parsedQuantity, _, quantityErr = decimalsupport.ParseString(quantity)
	if quantityErr != nil {
		return ""
	}
	var parsedGrossValue, _, grossValueErr = decimalsupport.ParseString(grossValue)
	if grossValueErr != nil {
		return ""
	}
	_, canonical, err := decimalsupport.DivideExact(parsedGrossValue, parsedQuantity)
	if err != nil {
		return ""
	}

	return canonical
}

// diagnosticDerivedGrossValue preserves exact-decimal gross-value derivation for
// mapper diagnostics without storing the derived value in the persisted model.
// Authored by: OpenCode
func diagnosticDerivedGrossValue(quantity string, unitPrice string) string {
	if strings.TrimSpace(quantity) == "" || strings.TrimSpace(unitPrice) == "" {
		return ""
	}

	var parsedQuantity, _, quantityErr = decimalsupport.ParseString(quantity)
	if quantityErr != nil {
		return ""
	}
	var parsedUnitPrice, _, unitPriceErr = decimalsupport.ParseString(unitPrice)
	if unitPriceErr != nil {
		return ""
	}
	var grossValue apd.Decimal
	_, _ = apd.BaseContext.Mul(&grossValue, &parsedQuantity, &parsedUnitPrice)
	grossValue.Reduce(&grossValue)

	return grossValue.Text('f')
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
