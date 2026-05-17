// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import (
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/cockroachdb/apd/v3"
)

// DiagnosticFailureStage identifies the sync stage that produced diagnostic
// troubleshooting context.
// Authored by: OpenCode
type DiagnosticFailureStage string

const (
	// DiagnosticFailureStageMapping identifies Ghostfolio activity mapping failures.
	DiagnosticFailureStageMapping DiagnosticFailureStage = "mapping"

	// DiagnosticFailureStageNormalization identifies normalized-history ordering or duplicate-handling failures.
	DiagnosticFailureStageNormalization DiagnosticFailureStage = "normalization"

	// DiagnosticFailureStageValidation identifies normalized-history rule failures.
	DiagnosticFailureStageValidation DiagnosticFailureStage = "validation"

	// DiagnosticFailureStageStoredDataCompatibility identifies local stored-data compatibility failures.
	DiagnosticFailureStageStoredDataCompatibility DiagnosticFailureStage = "stored_data_compatibility"

	// DiagnosticFailureStageProtectedPersistence identifies protected-storage write and local artifact failures.
	DiagnosticFailureStageProtectedPersistence DiagnosticFailureStage = "protected_persistence"
)

// DiagnosticRecord stores the offending record context allowed in synced-data
// diagnostic reports.
// Authored by: OpenCode
type DiagnosticRecord struct {
	SourceID               string `json:"source_id,omitempty"`
	OccurredAt             string `json:"occurred_at,omitempty"`
	ActivityType           string `json:"activity_type,omitempty"`
	AssetSymbol            string `json:"asset_symbol,omitempty"`
	AssetName              string `json:"asset_name,omitempty"`
	OrderCurrency          string `json:"order_currency,omitempty"`
	AssetProfileCurrency   string `json:"asset_profile_currency,omitempty"`
	BaseCurrency           string `json:"base_currency,omitempty"`
	Quantity               string `json:"quantity,omitempty"`
	UnitPrice              string `json:"unit_price,omitempty"`
	UnitPriceCurrency      string `json:"unit_price_currency,omitempty"`
	GrossValue             string `json:"gross_value,omitempty"`
	GrossValueCurrency     string `json:"gross_value_currency,omitempty"`
	FeeAmount              string `json:"fee_amount,omitempty"`
	FeeAmountCurrency      string `json:"fee_amount_currency,omitempty"`
	OrderUnitPrice         string `json:"order_unit_price,omitempty"`
	OrderGrossValue        string `json:"order_gross_value,omitempty"`
	OrderFeeAmount         string `json:"order_fee_amount,omitempty"`
	AssetProfileUnitPrice  string `json:"asset_profile_unit_price,omitempty"`
	AssetProfileFeeAmount  string `json:"asset_profile_fee_amount,omitempty"`
	BaseGrossValue         string `json:"base_gross_value,omitempty"`
	BaseFeeAmount          string `json:"base_fee_amount,omitempty"`
	Comment                string `json:"comment,omitempty"`
	DataSource             string `json:"data_source,omitempty"`
	SourceScopeID          string `json:"source_scope_id,omitempty"`
	SourceScopeName        string `json:"source_scope_name,omitempty"`
	SourceScopeKind        string `json:"source_scope_kind,omitempty"`
	SourceScopeReliability string `json:"source_scope_reliability,omitempty"`
}

// DiagnosticContext stores the structured troubleshooting context attached to a
// synced-data failure.
// Authored by: OpenCode
type DiagnosticContext struct {
	FailureStage  DiagnosticFailureStage `json:"failure_stage,omitempty"`
	FailureDetail string                 `json:"failure_detail,omitempty"`
	Records       []DiagnosticRecord     `json:"records,omitempty"`
}

// DiagnosticContextCarrier exposes structured troubleshooting context from
// lower-level sync failures.
// Authored by: OpenCode
type DiagnosticContextCarrier interface {
	DiagnosticContext() DiagnosticContext
}

// DiagnosticRecordFromActivityRecord converts one normalized activity record
// into the structured record context used by diagnostic reports.
//
// Example:
//
//	record := model.DiagnosticRecordFromActivityRecord(activity)
//	_ = record.SourceID
//
// Authored by: OpenCode
func DiagnosticRecordFromActivityRecord(record ActivityRecord) DiagnosticRecord {
	var resolvedAmounts ResolvedActivityAmounts
	if amounts, err := ResolveActivityAmounts(record); err == nil {
		resolvedAmounts = amounts
	}
	var unitPrice = canonicalDiagnosticDecimalPointer(resolvedAmounts.UnitPrice)
	var grossValue = canonicalDiagnosticDecimalPointer(resolvedAmounts.GrossValue)
	var feeAmount = canonicalDiagnosticDecimalPointer(resolvedAmounts.FeeAmount)
	var orderUnitPrice = canonicalDiagnosticDecimalPointer(record.OrderUnitPrice)
	var orderGrossValue = canonicalDiagnosticDecimalPointer(record.OrderGrossValue)
	var orderFeeAmount = canonicalDiagnosticDecimalPointer(record.OrderFeeAmount)
	var assetProfileUnitPrice = canonicalDiagnosticDecimalPointer(record.AssetProfileUnitPrice)
	var assetProfileFeeAmount = canonicalDiagnosticDecimalPointer(record.AssetProfileFeeAmount)
	var baseGrossValue = canonicalDiagnosticDecimalPointer(record.BaseGrossValue)
	var baseFeeAmount = canonicalDiagnosticDecimalPointer(record.BaseFeeAmount)
	var sourceScopeID string
	var sourceScopeName string
	var sourceScopeKind string
	var sourceScopeReliability string
	if record.SourceScope != nil {
		sourceScopeID = record.SourceScope.ID
		sourceScopeName = record.SourceScope.Name
		sourceScopeKind = string(record.SourceScope.Kind)
		sourceScopeReliability = string(record.SourceScope.Reliability)
	}

	return DiagnosticRecord{
		SourceID:               record.SourceID,
		OccurredAt:             record.OccurredAt,
		ActivityType:           string(record.ActivityType),
		AssetSymbol:            record.AssetSymbol,
		AssetName:              record.AssetName,
		OrderCurrency:          record.OrderCurrency,
		AssetProfileCurrency:   record.AssetProfileCurrency,
		BaseCurrency:           record.BaseCurrency,
		Quantity:               canonicalDiagnosticDecimal(record.Quantity),
		UnitPrice:              unitPrice,
		UnitPriceCurrency:      resolvedAmounts.UnitPriceCurrency,
		GrossValue:             grossValue,
		GrossValueCurrency:     resolvedAmounts.GrossValueCurrency,
		FeeAmount:              feeAmount,
		FeeAmountCurrency:      resolvedAmounts.FeeAmountCurrency,
		OrderUnitPrice:         orderUnitPrice,
		OrderGrossValue:        orderGrossValue,
		OrderFeeAmount:         orderFeeAmount,
		AssetProfileUnitPrice:  assetProfileUnitPrice,
		AssetProfileFeeAmount:  assetProfileFeeAmount,
		BaseGrossValue:         baseGrossValue,
		BaseFeeAmount:          baseFeeAmount,
		Comment:                record.Comment,
		DataSource:             record.DataSource,
		SourceScopeID:          sourceScopeID,
		SourceScopeName:        sourceScopeName,
		SourceScopeKind:        sourceScopeKind,
		SourceScopeReliability: sourceScopeReliability,
	}
}

// canonicalDiagnosticDecimal returns a stable decimal string for diagnostic
// context.
// Authored by: OpenCode
func canonicalDiagnosticDecimal(value apd.Decimal) string {
	var canonical, err = decimalsupport.CanonicalString(value)
	if err == nil {
		return canonical
	}

	return value.String()
}

// canonicalDiagnosticDecimalPointer returns a stable optional decimal string
// for diagnostic context.
// Authored by: OpenCode
func canonicalDiagnosticDecimalPointer(value *apd.Decimal) string {
	if value == nil {
		return ""
	}

	var canonical, err = decimalsupport.CanonicalStringPointer(value)
	if err == nil {
		return canonical
	}

	return value.String()
}
