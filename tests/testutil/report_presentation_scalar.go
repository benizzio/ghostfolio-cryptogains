package testutil

// newPresentationScalarCase builds a non-matrix acceptance case and its base
// warning, model, and parity occurrences.
// Authored by: OpenCode
func newPresentationScalarCase(id string, kind ReportPresentationCaseKind, fieldName string, value presentationScalarCase) ReportPresentationAcceptanceCase {
	var acceptanceCase = newPresentationCase(ReportPresentationAcceptanceCase{
		ID:                   id,
		Kind:                 kind,
		Section:              fieldName,
		VectorCase:           value.ID,
		ExactValue:           value.ExactValue,
		ExpectedVisibleValue: value.ExpectedValue,
	})
	if kind == ReportPresentationCaseKindQuantity {
		for _, definition := range presentationQuantityOccurrences() {
			acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
				presentationOccurrence(ReportPresentationPopulationQuantity, acceptanceCase.ID, ReportPresentationFormatMarkdown, definition.role, definition.section, definition.asset, definition.source, definition.field, "", 0),
				presentationOccurrence(ReportPresentationPopulationQuantity, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, definition.section, definition.asset, definition.source, definition.field, "", 0),
				presentationParityOccurrence(acceptanceCase.ID, definition.role, definition.section, definition.asset, definition.source, definition.field, "", 0),
			)
		}
	}
	if kind == ReportPresentationCaseKindBoolean {
		for _, definition := range presentationBooleanOccurrences() {
			acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
				presentationOccurrence(ReportPresentationPopulationBoolean, acceptanceCase.ID, ReportPresentationFormatMarkdown, ReportPresentationDocumentRoleAnnex, "detailed_per_asset_audit", definition.asset, definition.source, fieldName, "", 0),
				presentationOccurrence(ReportPresentationPopulationBoolean, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, "detailed_per_asset_audit", definition.asset, definition.source, fieldName, "", 0),
				presentationParityOccurrence(acceptanceCase.ID, ReportPresentationDocumentRoleAnnex, "detailed_per_asset_audit", definition.asset, definition.source, fieldName, "", 0),
			)
		}
	}
	if kind == ReportPresentationCaseKindRate {
		for _, field := range presentationRateMetadataFields() {
			acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, presentationParityOccurrence(acceptanceCase.ID, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "asset-btc", "btc-sell-2024-001", field, "", 0))
		}
	}
	return acceptanceCase
}

// presentationQuantityOccurrenceDefinition identifies every report quantity
// emitted by the deterministic acceptance report.
// Authored by: OpenCode
type presentationQuantityOccurrenceDefinition struct {
	role                          ReportPresentationDocumentRole
	section, asset, source, field string
}

// presentationQuantityOccurrences returns all required quantity fields.
// Authored by: OpenCode
func presentationQuantityOccurrences() []presentationQuantityOccurrenceDefinition {
	return []presentationQuantityOccurrenceDefinition{
		{ReportPresentationDocumentRoleMain, "position", "asset-btc", "opening-position", "opening_quantity"},
		{ReportPresentationDocumentRoleMain, "position", "asset-btc", "closing-position", "closing_quantity"},
		{ReportPresentationDocumentRoleMain, "position", "asset-historical", "historical-position", "historical_position_quantity"},
		{ReportPresentationDocumentRoleMain, "in_year_activity", "asset-btc", "btc-sell-2024-001", "activity_quantity"},
		{ReportPresentationDocumentRoleMain, "in_year_activity", "asset-btc", "btc-sell-2024-001", "quantity_after_row"},
		{ReportPresentationDocumentRoleMain, "liquidation_calculations", "asset-btc", "btc-sell-2024-001", "disposed_quantity"},
		{ReportPresentationDocumentRoleAnnex, "detailed_per_asset_audit", "asset-btc", "btc-sell-2024-001", "audit_quantity"},
		{ReportPresentationDocumentRoleAnnex, "detailed_per_asset_audit", "asset-btc", "btc-sell-2024-001", "quantity_after_activity"},
	}
}

// presentationBooleanOccurrenceDefinition identifies one Annex boolean row.
// Authored by: OpenCode
type presentationBooleanOccurrenceDefinition struct {
	asset, source string
}

// presentationBooleanOccurrences returns every deterministic Annex boolean row.
// Authored by: OpenCode
func presentationBooleanOccurrences() []presentationBooleanOccurrenceDefinition {
	return []presentationBooleanOccurrenceDefinition{
		{"asset-btc", "btc-sell-2024-001"},
		{"asset-xrp", "xrp-reduction-2024-001"},
		{"asset-eth", "eth-reference-buy"},
		{"asset-eth", "tiny-positive-unclassified"},
	}
}

// presentationRateMetadataFields returns all visible normalized-rate row fields
// whose Markdown and PDF values must be compared for parity.
// Authored by: OpenCode
func presentationRateMetadataFields() []string {
	return []string{"source_id", "asset", "rate_date", "source_currency", "report_base_currency", "quote_direction", "rate_value"}
}

// presentationScalarCase stores one exact-value control used outside the
// financial matrix.
// Authored by: OpenCode
type presentationScalarCase struct {
	ID            string
	ExactValue    string
	ExpectedValue string
}
