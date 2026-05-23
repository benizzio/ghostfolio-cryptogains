// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import (
	"encoding/json"

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

// DiagnosticRecord stores source-faithful offending-record context allowed in
// synced-data diagnostic reports.
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

// DiagnosticSourceScopeRecord stores one serialized source-scope object for an
// offending persisted activity record included in a diagnostic report.
// Authored by: OpenCode
type DiagnosticSourceScopeRecord struct {
	ID          *string `json:"id"`
	Name        *string `json:"name"`
	Kind        *string `json:"kind"`
	Reliability *string `json:"reliability"`
}

// DiagnosticActivityRecord stores the original persisted activity-record shape
// used by report-failure diagnostics.
// Authored by: OpenCode
type DiagnosticActivityRecord struct {
	SourceID              *string                      `json:"source_id"`
	OccurredAt            *string                      `json:"occurred_at"`
	ActivityType          *string                      `json:"activity_type"`
	AssetIdentityKey      *string                      `json:"asset_identity_key"`
	AssetSymbol           *string                      `json:"asset_symbol"`
	AssetName             *string                      `json:"asset_name"`
	Quantity              *string                      `json:"quantity"`
	OrderCurrency         *string                      `json:"order_currency"`
	OrderUnitPrice        *string                      `json:"order_unit_price"`
	OrderGrossValue       *string                      `json:"order_gross_value"`
	OrderFeeAmount        *string                      `json:"order_fee_amount"`
	AssetProfileCurrency  *string                      `json:"asset_profile_currency"`
	AssetProfileUnitPrice *string                      `json:"asset_profile_unit_price"`
	AssetProfileFeeAmount *string                      `json:"asset_profile_fee_amount"`
	BaseCurrency          *string                      `json:"base_currency"`
	BaseGrossValue        *string                      `json:"base_gross_value"`
	BaseFeeAmount         *string                      `json:"base_fee_amount"`
	Comment               *string                      `json:"comment"`
	DataSource            *string                      `json:"data_source"`
	SourceScope           *DiagnosticSourceScopeRecord `json:"source_scope"`
	RawHash               *string                      `json:"raw_hash"`
}

// DiagnosticContext stores the structured troubleshooting context attached to a
// synced-data failure.
// Authored by: OpenCode
type DiagnosticContext struct {
	FailureStage            DiagnosticFailureStage `json:"failure_stage,omitempty"`
	FailureDetail           string                 `json:"failure_detail,omitempty"`
	Records                 []DiagnosticRecord     `json:"records,omitempty"`
	OffendingActivityRecord *ActivityRecord        `json:"-"`
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

// MarshalJSON renders explicit `null` values for absent source fields so
// diagnostic artifacts remain source-faithful.
// Authored by: OpenCode
func (record DiagnosticRecord) MarshalJSON() ([]byte, error) {
	type diagnosticRecordDocument struct {
		SourceID               *string `json:"source_id"`
		OccurredAt             *string `json:"occurred_at"`
		ActivityType           *string `json:"activity_type"`
		AssetSymbol            *string `json:"asset_symbol"`
		AssetName              *string `json:"asset_name"`
		OrderCurrency          *string `json:"order_currency"`
		AssetProfileCurrency   *string `json:"asset_profile_currency"`
		BaseCurrency           *string `json:"base_currency"`
		Quantity               *string `json:"quantity"`
		OrderUnitPrice         *string `json:"order_unit_price"`
		OrderGrossValue        *string `json:"order_gross_value"`
		OrderFeeAmount         *string `json:"order_fee_amount"`
		AssetProfileUnitPrice  *string `json:"asset_profile_unit_price"`
		AssetProfileFeeAmount  *string `json:"asset_profile_fee_amount"`
		BaseGrossValue         *string `json:"base_gross_value"`
		BaseFeeAmount          *string `json:"base_fee_amount"`
		Comment                *string `json:"comment"`
		DataSource             *string `json:"data_source"`
		SourceScopeID          *string `json:"source_scope_id"`
		SourceScopeName        *string `json:"source_scope_name"`
		SourceScopeKind        *string `json:"source_scope_kind"`
		SourceScopeReliability *string `json:"source_scope_reliability"`
	}

	return json.Marshal(diagnosticRecordDocument{
		SourceID:               diagnosticStringPointer(record.SourceID),
		OccurredAt:             diagnosticStringPointer(record.OccurredAt),
		ActivityType:           diagnosticStringPointer(record.ActivityType),
		AssetSymbol:            diagnosticStringPointer(record.AssetSymbol),
		AssetName:              diagnosticStringPointer(record.AssetName),
		OrderCurrency:          diagnosticStringPointer(record.OrderCurrency),
		AssetProfileCurrency:   diagnosticStringPointer(record.AssetProfileCurrency),
		BaseCurrency:           diagnosticStringPointer(record.BaseCurrency),
		Quantity:               diagnosticStringPointer(record.Quantity),
		OrderUnitPrice:         diagnosticStringPointer(record.OrderUnitPrice),
		OrderGrossValue:        diagnosticStringPointer(record.OrderGrossValue),
		OrderFeeAmount:         diagnosticStringPointer(record.OrderFeeAmount),
		AssetProfileUnitPrice:  diagnosticStringPointer(record.AssetProfileUnitPrice),
		AssetProfileFeeAmount:  diagnosticStringPointer(record.AssetProfileFeeAmount),
		BaseGrossValue:         diagnosticStringPointer(record.BaseGrossValue),
		BaseFeeAmount:          diagnosticStringPointer(record.BaseFeeAmount),
		Comment:                diagnosticStringPointer(record.Comment),
		DataSource:             diagnosticStringPointer(record.DataSource),
		SourceScopeID:          diagnosticStringPointer(record.SourceScopeID),
		SourceScopeName:        diagnosticStringPointer(record.SourceScopeName),
		SourceScopeKind:        diagnosticStringPointer(record.SourceScopeKind),
		SourceScopeReliability: diagnosticStringPointer(record.SourceScopeReliability),
	})
}

// UnmarshalJSON restores one diagnostic record while accepting explicit `null`
// values for absent source fields.
// Authored by: OpenCode
func (record *DiagnosticRecord) UnmarshalJSON(raw []byte) error {
	type diagnosticRecordDocument struct {
		SourceID               *string `json:"source_id"`
		OccurredAt             *string `json:"occurred_at"`
		ActivityType           *string `json:"activity_type"`
		AssetSymbol            *string `json:"asset_symbol"`
		AssetName              *string `json:"asset_name"`
		OrderCurrency          *string `json:"order_currency"`
		AssetProfileCurrency   *string `json:"asset_profile_currency"`
		BaseCurrency           *string `json:"base_currency"`
		Quantity               *string `json:"quantity"`
		OrderUnitPrice         *string `json:"order_unit_price"`
		OrderGrossValue        *string `json:"order_gross_value"`
		OrderFeeAmount         *string `json:"order_fee_amount"`
		AssetProfileUnitPrice  *string `json:"asset_profile_unit_price"`
		AssetProfileFeeAmount  *string `json:"asset_profile_fee_amount"`
		BaseGrossValue         *string `json:"base_gross_value"`
		BaseFeeAmount          *string `json:"base_fee_amount"`
		Comment                *string `json:"comment"`
		DataSource             *string `json:"data_source"`
		SourceScopeID          *string `json:"source_scope_id"`
		SourceScopeName        *string `json:"source_scope_name"`
		SourceScopeKind        *string `json:"source_scope_kind"`
		SourceScopeReliability *string `json:"source_scope_reliability"`
	}

	var document diagnosticRecordDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return err
	}

	record.SourceID = diagnosticStringValue(document.SourceID)
	record.OccurredAt = diagnosticStringValue(document.OccurredAt)
	record.ActivityType = diagnosticStringValue(document.ActivityType)
	record.AssetSymbol = diagnosticStringValue(document.AssetSymbol)
	record.AssetName = diagnosticStringValue(document.AssetName)
	record.OrderCurrency = diagnosticStringValue(document.OrderCurrency)
	record.AssetProfileCurrency = diagnosticStringValue(document.AssetProfileCurrency)
	record.BaseCurrency = diagnosticStringValue(document.BaseCurrency)
	record.Quantity = diagnosticStringValue(document.Quantity)
	record.OrderUnitPrice = diagnosticStringValue(document.OrderUnitPrice)
	record.OrderGrossValue = diagnosticStringValue(document.OrderGrossValue)
	record.OrderFeeAmount = diagnosticStringValue(document.OrderFeeAmount)
	record.AssetProfileUnitPrice = diagnosticStringValue(document.AssetProfileUnitPrice)
	record.AssetProfileFeeAmount = diagnosticStringValue(document.AssetProfileFeeAmount)
	record.BaseGrossValue = diagnosticStringValue(document.BaseGrossValue)
	record.BaseFeeAmount = diagnosticStringValue(document.BaseFeeAmount)
	record.Comment = diagnosticStringValue(document.Comment)
	record.DataSource = diagnosticStringValue(document.DataSource)
	record.SourceScopeID = diagnosticStringValue(document.SourceScopeID)
	record.SourceScopeName = diagnosticStringValue(document.SourceScopeName)
	record.SourceScopeKind = diagnosticStringValue(document.SourceScopeKind)
	record.SourceScopeReliability = diagnosticStringValue(document.SourceScopeReliability)
	return nil
}

// DiagnosticActivityRecordFromActivityRecord converts one persisted activity
// record into the source-faithful report-diagnostics activity shape.
//
// Example:
//
//	record := model.DiagnosticActivityRecordFromActivityRecord(activity)
//	_ = record.AssetIdentityKey
//
// Authored by: OpenCode
func DiagnosticActivityRecordFromActivityRecord(record ActivityRecord) DiagnosticActivityRecord {
	var sourceScope *DiagnosticSourceScopeRecord
	if record.SourceScope != nil {
		sourceScope = &DiagnosticSourceScopeRecord{
			ID:          diagnosticStringPointer(record.SourceScope.ID),
			Name:        diagnosticStringPointer(record.SourceScope.Name),
			Kind:        diagnosticStringPointer(string(record.SourceScope.Kind)),
			Reliability: diagnosticStringPointer(string(record.SourceScope.Reliability)),
		}
	}

	return DiagnosticActivityRecord{
		SourceID:              diagnosticStringPointer(record.SourceID),
		OccurredAt:            diagnosticStringPointer(record.OccurredAt),
		ActivityType:          diagnosticStringPointer(string(record.ActivityType)),
		AssetIdentityKey:      diagnosticStringPointer(record.AssetIdentityKey),
		AssetSymbol:           diagnosticStringPointer(record.AssetSymbol),
		AssetName:             diagnosticStringPointer(record.AssetName),
		Quantity:              diagnosticStringPointer(canonicalDiagnosticDecimal(record.Quantity)),
		OrderCurrency:         diagnosticStringPointer(record.OrderCurrency),
		OrderUnitPrice:        diagnosticStringPointer(canonicalDiagnosticDecimalPointer(record.OrderUnitPrice)),
		OrderGrossValue:       diagnosticStringPointer(canonicalDiagnosticDecimalPointer(record.OrderGrossValue)),
		OrderFeeAmount:        diagnosticStringPointer(canonicalDiagnosticDecimalPointer(record.OrderFeeAmount)),
		AssetProfileCurrency:  diagnosticStringPointer(record.AssetProfileCurrency),
		AssetProfileUnitPrice: diagnosticStringPointer(canonicalDiagnosticDecimalPointer(record.AssetProfileUnitPrice)),
		AssetProfileFeeAmount: diagnosticStringPointer(canonicalDiagnosticDecimalPointer(record.AssetProfileFeeAmount)),
		BaseCurrency:          diagnosticStringPointer(record.BaseCurrency),
		BaseGrossValue:        diagnosticStringPointer(canonicalDiagnosticDecimalPointer(record.BaseGrossValue)),
		BaseFeeAmount:         diagnosticStringPointer(canonicalDiagnosticDecimalPointer(record.BaseFeeAmount)),
		Comment:               diagnosticStringPointer(record.Comment),
		DataSource:            diagnosticStringPointer(record.DataSource),
		SourceScope:           sourceScope,
		RawHash:               diagnosticStringPointer(record.RawHash),
	}
}

// RedactFinancialValues removes quantity and monetary values from one
// serialized persisted activity record while preserving non-financial context.
// Authored by: OpenCode
func (record DiagnosticActivityRecord) RedactFinancialValues() DiagnosticActivityRecord {
	record.Quantity = nil
	record.OrderUnitPrice = nil
	record.OrderGrossValue = nil
	record.OrderFeeAmount = nil
	record.AssetProfileUnitPrice = nil
	record.AssetProfileFeeAmount = nil
	record.BaseGrossValue = nil
	record.BaseFeeAmount = nil
	return record
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

// diagnosticStringPointer converts one optional string field into a nullable
// JSON-ready pointer.
// Authored by: OpenCode
func diagnosticStringPointer(value string) *string {
	if value == "" {
		return nil
	}

	var copied = value
	return &copied
}

// diagnosticStringValue converts one nullable decoded string back into the
// runtime diagnostic record representation.
// Authored by: OpenCode
func diagnosticStringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
