// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

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
