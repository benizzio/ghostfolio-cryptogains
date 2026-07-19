package testutil

import "strconv"

// appendPresentationFinancialOccurrences adds visible-field and parity keys for
// one matrix case, retaining blank nullable rows for parity while adding only
// present, visible financial values to V.
// Authored by: OpenCode
func appendPresentationFinancialOccurrences(acceptanceCase *ReportPresentationAcceptanceCase, row presentationFinancialRow) {
	var documentRole = ReportPresentationDocumentRoleMain
	if row.DocumentRole != "" {
		documentRole = row.DocumentRole
	}
	for _, field := range row.Fields {
		var assetIdentity, sourceIdentity = presentationFinancialIdentity(row, field)
		if !presentationFieldIsOmitted(*acceptanceCase, field.Name) && (!acceptanceCase.Absent || !field.Nullable) {
			acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
				presentationOccurrence(ReportPresentationPopulationVisibleFinancial, acceptanceCase.ID, ReportPresentationFormatMarkdown, documentRole, row.Section, assetIdentity, sourceIdentity, field.Name, field.AmountKind, field.AmountOrdinal),
				presentationOccurrence(ReportPresentationPopulationVisibleFinancial, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, row.Section, assetIdentity, sourceIdentity, field.Name, field.AmountKind, field.AmountOrdinal),
			)
		}
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, presentationParityOccurrence(acceptanceCase.ID, documentRole, row.Section, assetIdentity, sourceIdentity, field.Name, field.AmountKind, field.AmountOrdinal))
	}
}

// presentationOccurrence creates one complete semantic occurrence identity.
// Authored by: OpenCode
func presentationOccurrence(population ReportPresentationPopulation, caseID string, format ReportPresentationFormat, role ReportPresentationDocumentRole, section string, assetIdentity string, sourceIdentity string, fieldName string, amountKind string, ordinal int) ReportPresentationOccurrenceKey {
	return ReportPresentationOccurrenceKey{Population: population, CaseID: caseID, Format: format, DocumentRole: role, Section: section, AssetIdentity: assetIdentity, SourceOrRowIdentity: sourceIdentity, FieldName: fieldName, AmountKind: amountKind, AmountOrdinal: ordinal}
}

// presentationParityOccurrence creates one cross-format identity for a semantic
// field compared from independent Markdown and PDF observations.
// Authored by: OpenCode
func presentationParityOccurrence(caseID string, role ReportPresentationDocumentRole, section string, assetIdentity string, sourceIdentity string, fieldName string, amountKind string, ordinal int) ReportPresentationOccurrenceKey {
	return presentationOccurrence(ReportPresentationPopulationParity, caseID, reportPresentationFormatCrossFormat, role, section, assetIdentity, sourceIdentity, fieldName, amountKind, ordinal)
}

// presentationFinancialIdentity identifies the concrete repeated report row for
// one matrix field instead of using the acceptance case as a surrogate source.
// Authored by: OpenCode
func presentationFinancialIdentity(row presentationFinancialRow, field ReportPresentationFinancialField) (string, string) {
	switch row.ID {
	case "summary-net-gain-or-loss":
		if field.Name == "overall_yearly_net_total" {
			return "report", "yearly-net-total"
		}
		return "asset-btc", "summary-asset-btc"
	case "position-cost-basis":
		switch field.Name {
		case "historical_cost_basis":
			return "asset-historical", "historical-position"
		case "closing_cost_basis":
			return "asset-btc", "closing-position"
		default:
			return "asset-btc", "opening-position"
		}
	case "in-year-activity", "liquidation-allocated-basis", "liquidation-net-proceeds-gain-or-loss", "audit-activity", "audit-allocated-basis", "audit-net-proceeds-gain-or-loss":
		return "asset-btc", "btc-sell-2024-001"
	case "conversion-amount":
		return "asset-btc", "btc-sell-2024-001"
	default:
		return "acceptance", "acceptance"
	}
}

// presentationFieldIsOmitted reports whether a field is the inherited exact
// zero summary row omitted from visible presentation.
// Authored by: OpenCode
func presentationFieldIsOmitted(acceptanceCase ReportPresentationAcceptanceCase, fieldName string) bool {
	if acceptanceCase.FinancialFieldClass == "summary-net-gain-or-loss" && fieldName == "per_asset_net_gain_or_loss" && (acceptanceCase.ExactValue == "0" || acceptanceCase.ExactValue == "-0") {
		return true
	}
	for _, omittedFieldName := range acceptanceCase.OmittedFieldNames {
		if omittedFieldName == fieldName {
			return true
		}
	}
	return false
}

// appendPresentationCurrencyOccurrences adds currency applicability and the
// monetary source-price control for one Annex case.
// Authored by: OpenCode
func appendPresentationCurrencyOccurrences(acceptanceCase *ReportPresentationAcceptanceCase) {
	var sourceIdentity = renderingPresentationCurrencySourceIdentity(acceptanceCase.VectorCase)
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
		presentationOccurrence(ReportPresentationPopulationVisibleFinancial, acceptanceCase.ID, ReportPresentationFormatMarkdown, ReportPresentationDocumentRoleAnnex, "detailed_per_asset_audit", "asset-btc", sourceIdentity, "unit_price", "unit_price", 0),
		presentationOccurrence(ReportPresentationPopulationVisibleFinancial, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, "detailed_per_asset_audit", "asset-btc", sourceIdentity, "unit_price", "unit_price", 0),
		presentationParityOccurrence(acceptanceCase.ID, ReportPresentationDocumentRoleAnnex, "detailed_per_asset_audit", "asset-btc", sourceIdentity, "unit_price", "unit_price", 0),
	)
	var population = ReportPresentationPopulationUnclassified
	if acceptanceCase.IsZeroPricedHoldingReduction {
		population = ReportPresentationPopulationClassifiedCurrency
	}
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
		presentationOccurrence(population, acceptanceCase.ID, ReportPresentationFormatMarkdown, ReportPresentationDocumentRoleAnnex, "detailed_per_asset_audit", "asset-btc", sourceIdentity, "original_activity_currency", "", 0),
		presentationOccurrence(population, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, "detailed_per_asset_audit", "asset-btc", sourceIdentity, "original_activity_currency", "", 0),
		presentationParityOccurrence(acceptanceCase.ID, ReportPresentationDocumentRoleAnnex, "detailed_per_asset_audit", "asset-btc", sourceIdentity, "original_activity_currency", "", 0),
	)
}

// renderingPresentationCurrencySourceIdentity maps each currency control to its
// concrete Annex source row.
// Authored by: OpenCode
func renderingPresentationCurrencySourceIdentity(vectorCase string) string {
	switch vectorCase {
	case "classified-zero-priced":
		return "xrp-reduction-2024-001"
	case "unclassified-priced":
		return "eth-reference-buy"
	case "unclassified-tiny-positive":
		return "tiny-positive-unclassified"
	default:
		return vectorCase
	}
}

// newPresentationCase builds the common case shape and warning/model evidence.
// Authored by: OpenCode
func newPresentationCase(input ReportPresentationAcceptanceCase) ReportPresentationAcceptanceCase {
	var acceptanceCase = input
	acceptanceCase.FinancialFields = append([]ReportPresentationFinancialField(nil), input.FinancialFields...)
	acceptanceCase.OmittedFieldNames = append([]string(nil), input.OmittedFieldNames...)
	acceptanceCase.ConvertedAmountKinds = append([]string(nil), input.ConvertedAmountKinds...)
	acceptanceCase.OccurrenceKeys = append([]ReportPresentationOccurrenceKey(nil), input.OccurrenceKeys...)
	acceptanceCase.Attempts = []ReportPresentationFormatAttempt{
		{Format: ReportPresentationFormatMarkdown, DocumentRoles: []ReportPresentationDocumentRole{ReportPresentationDocumentRoleMain, ReportPresentationDocumentRoleAnnex}},
		{Format: ReportPresentationFormatPDF, DocumentRoles: []ReportPresentationDocumentRole{ReportPresentationDocumentRoleCombined}},
	}
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
		presentationOccurrence(ReportPresentationPopulationWarning, input.ID, ReportPresentationFormatMarkdown, ReportPresentationDocumentRoleMain, "report_header", "report", "legal_use_warning", "", "", 0),
		presentationOccurrence(ReportPresentationPopulationWarning, input.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, "report_header", "report", "legal_use_warning", "", "", 0),
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationModelIntegrity,
			CaseID:              input.ID,
			Format:              ReportPresentationFormatMarkdown,
			DocumentRole:        reportPresentationDocumentRoleModel,
			Section:             "AUD-001",
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: "exact_model_before_after_render",
		},
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationModelIntegrity,
			CaseID:              input.ID,
			Format:              ReportPresentationFormatPDF,
			DocumentRole:        reportPresentationDocumentRoleModel,
			Section:             "AUD-001",
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: "exact_model_before_after_render",
		})
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, presentationParityOccurrence(input.ID, ReportPresentationDocumentRoleMain, "report_header", "report", "legal_use_warning", "", "", 0))
	return acceptanceCase
}

// appendPresentationConvertedOccurrences adds conversion-row, entry, financial,
// and parity keys while preserving the received amount-kind order.
// Authored by: OpenCode
func appendPresentationConvertedOccurrences(acceptanceCase *ReportPresentationAcceptanceCase) {
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
		presentationOccurrence(ReportPresentationPopulationConversionRow, acceptanceCase.ID, ReportPresentationFormatMarkdown, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), "converted_amount_sequence", "", 0),
		presentationOccurrence(ReportPresentationPopulationConversionRow, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), "converted_amount_sequence", "", 0),
		presentationParityOccurrence(acceptanceCase.ID, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), "converted_amount_sequence", "", 0),
	)
	for amountOrdinal, amountKind := range acceptanceCase.ConvertedAmountKinds {
		var fieldName = "original_" + amountKind
		var convertedFieldName = "converted_" + amountKind
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
			presentationOccurrence(ReportPresentationPopulationVisibleFinancial, acceptanceCase.ID, ReportPresentationFormatMarkdown, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), fieldName, amountKind, amountOrdinal*2),
			presentationOccurrence(ReportPresentationPopulationVisibleFinancial, acceptanceCase.ID, ReportPresentationFormatMarkdown, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), convertedFieldName, amountKind, amountOrdinal*2+1),
			presentationOccurrence(ReportPresentationPopulationVisibleFinancial, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), fieldName, amountKind, amountOrdinal*2),
			presentationOccurrence(ReportPresentationPopulationVisibleFinancial, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), convertedFieldName, amountKind, amountOrdinal*2+1),
			presentationOccurrence(ReportPresentationPopulationConvertedEntry, acceptanceCase.ID, ReportPresentationFormatMarkdown, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), amountKind, amountKind, amountOrdinal),
			presentationOccurrence(ReportPresentationPopulationConvertedEntry, acceptanceCase.ID, ReportPresentationFormatPDF, ReportPresentationDocumentRoleCombined, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), amountKind, amountKind, amountOrdinal),
			presentationParityOccurrence(acceptanceCase.ID, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), fieldName, amountKind, amountOrdinal*2),
			presentationParityOccurrence(acceptanceCase.ID, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), convertedFieldName, amountKind, amountOrdinal*2+1),
			presentationParityOccurrence(acceptanceCase.ID, ReportPresentationDocumentRoleAnnex, "currency_conversion_audit", "conversion-row", presentationConvertedSourceIdentity(acceptanceCase.ID), amountKind, amountKind, amountOrdinal),
		)
	}
}

// presentationConvertedSourceIdentity maps the closed conversion case order to
// the concrete source ID emitted by the acceptance report fixture.
// Authored by: OpenCode
func presentationConvertedSourceIdentity(caseID string) string {
	for index, ID := range []string{"empty", "unit-price", "gross-value", "fee-amount", "unit-price-gross-value", "unit-price-fee-amount", "gross-value-fee-amount", "all"} {
		if caseID == "converted/"+ID {
			return "cv" + strconv.Itoa(index)
		}
	}
	return caseID
}

// countPresentationOccurrences derives every requested counter from the case
// manifest, retaining failed attempts in the denominator represented by keys.
// Authored by: OpenCode
func countPresentationOccurrences(cases []ReportPresentationAcceptanceCase) ReportPresentationAcceptanceCounters {
	var counters = ReportPresentationAcceptanceCounters{
		CaseCount:   len(cases),
		Populations: make(map[ReportPresentationPopulation]int),
	}
	for _, acceptanceCase := range cases {
		for _, occurrence := range acceptanceCase.OccurrenceKeys {
			counters.Populations[occurrence.Population]++
		}
	}
	return counters
}
