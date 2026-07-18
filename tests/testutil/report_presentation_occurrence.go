package testutil

// appendPresentationFinancialOccurrences adds visible-field and parity keys for
// one matrix case, excluding absent and inherited omitted values from V.
// Authored by: OpenCode
func appendPresentationFinancialOccurrences(acceptanceCase *ReportPresentationAcceptanceCase, row presentationFinancialRow) {
	var documentRole = ReportPresentationDocumentRoleMain
	if row.DocumentRole != "" {
		documentRole = row.DocumentRole
	}
	for _, field := range row.Fields {
		if !acceptanceCase.Absent && !presentationFieldIsOmitted(*acceptanceCase, field.Name) {
			acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
				ReportPresentationOccurrenceKey{
					Population:          ReportPresentationPopulationVisibleFinancial,
					CaseID:              acceptanceCase.ID,
					Format:              ReportPresentationFormatMarkdown,
					DocumentRole:        documentRole,
					Section:             row.Section,
					AssetIdentity:       "acceptance",
					SourceOrRowIdentity: acceptanceCase.ID,
					FieldName:           field.Name,
					AmountKind:          field.AmountKind,
					AmountOrdinal:       field.AmountOrdinal,
				},
				ReportPresentationOccurrenceKey{
					Population:          ReportPresentationPopulationVisibleFinancial,
					CaseID:              acceptanceCase.ID,
					Format:              ReportPresentationFormatPDF,
					DocumentRole:        documentRole,
					Section:             row.Section,
					AssetIdentity:       "acceptance",
					SourceOrRowIdentity: acceptanceCase.ID,
					FieldName:           field.Name,
					AmountKind:          field.AmountKind,
					AmountOrdinal:       field.AmountOrdinal,
				})
		}
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationParity,
			CaseID:              acceptanceCase.ID,
			Format:              reportPresentationFormatCrossFormat,
			DocumentRole:        documentRole,
			Section:             row.Section,
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: acceptanceCase.ID,
			FieldName:           field.Name,
			AmountKind:          field.AmountKind,
			AmountOrdinal:       field.AmountOrdinal,
		})
	}
}

// presentationFieldIsOmitted reports whether a field is the inherited exact
// zero summary row omitted from visible presentation.
// Authored by: OpenCode
func presentationFieldIsOmitted(acceptanceCase ReportPresentationAcceptanceCase, fieldName string) bool {
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
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationVisibleFinancial,
			CaseID:              acceptanceCase.ID,
			Format:              ReportPresentationFormatMarkdown,
			DocumentRole:        ReportPresentationDocumentRoleAnnex,
			Section:             "detailed_per_asset_audit",
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: acceptanceCase.ID,
			FieldName:           "unit_price",
			AmountKind:          "unit_price",
		},
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationVisibleFinancial,
			CaseID:              acceptanceCase.ID,
			Format:              ReportPresentationFormatPDF,
			DocumentRole:        ReportPresentationDocumentRoleAnnex,
			Section:             "detailed_per_asset_audit",
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: acceptanceCase.ID,
			FieldName:           "unit_price",
			AmountKind:          "unit_price",
		})
	var population = ReportPresentationPopulationUnclassified
	if acceptanceCase.IsZeroPricedHoldingReduction {
		population = ReportPresentationPopulationClassifiedCurrency
	}
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
		ReportPresentationOccurrenceKey{
			Population:          population,
			CaseID:              acceptanceCase.ID,
			Format:              ReportPresentationFormatMarkdown,
			DocumentRole:        ReportPresentationDocumentRoleAnnex,
			Section:             "detailed_per_asset_audit",
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: acceptanceCase.ID,
			FieldName:           "original_activity_currency",
		},
		ReportPresentationOccurrenceKey{
			Population:          population,
			CaseID:              acceptanceCase.ID,
			Format:              ReportPresentationFormatPDF,
			DocumentRole:        ReportPresentationDocumentRoleAnnex,
			Section:             "detailed_per_asset_audit",
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: acceptanceCase.ID,
			FieldName:           "original_activity_currency",
		})
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
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationWarning,
			CaseID:              input.ID,
			Format:              ReportPresentationFormatMarkdown,
			DocumentRole:        ReportPresentationDocumentRoleMain,
			Section:             "report_header",
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: "legal_use_warning",
		},
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationWarning,
			CaseID:              input.ID,
			Format:              ReportPresentationFormatPDF,
			DocumentRole:        ReportPresentationDocumentRoleMain,
			Section:             "report_header",
			AssetIdentity:       "acceptance",
			SourceOrRowIdentity: "legal_use_warning",
		},
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
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, ReportPresentationOccurrenceKey{
		Population:          ReportPresentationPopulationParity,
		CaseID:              input.ID,
		Format:              reportPresentationFormatCrossFormat,
		DocumentRole:        reportPresentationDocumentRoleModel,
		Section:             "acceptance_control",
		AssetIdentity:       "acceptance",
		SourceOrRowIdentity: "acceptance",
		FieldName:           "warning_and_metadata",
	})
	return acceptanceCase
}

// appendPresentationConvertedOccurrences adds conversion-row, entry, financial,
// and parity keys while preserving the received amount-kind order.
// Authored by: OpenCode
func appendPresentationConvertedOccurrences(acceptanceCase *ReportPresentationAcceptanceCase) {
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationConversionRow,
			CaseID:              acceptanceCase.ID,
			Format:              ReportPresentationFormatMarkdown,
			DocumentRole:        ReportPresentationDocumentRoleAnnex,
			Section:             "currency_conversion_audit",
			AssetIdentity:       "conversion-row",
			SourceOrRowIdentity: "converted_amounts",
		},
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationConversionRow,
			CaseID:              acceptanceCase.ID,
			Format:              ReportPresentationFormatPDF,
			DocumentRole:        ReportPresentationDocumentRoleAnnex,
			Section:             "currency_conversion_audit",
			AssetIdentity:       "conversion-row",
			SourceOrRowIdentity: "converted_amounts",
		},
		ReportPresentationOccurrenceKey{
			Population:          ReportPresentationPopulationParity,
			CaseID:              acceptanceCase.ID,
			Format:              reportPresentationFormatCrossFormat,
			DocumentRole:        ReportPresentationDocumentRoleAnnex,
			Section:             "currency_conversion_audit",
			AssetIdentity:       "conversion-row",
			SourceOrRowIdentity: acceptanceCase.ID,
			FieldName:           "converted_amount_sequence",
		})
	for amountOrdinal, amountKind := range acceptanceCase.ConvertedAmountKinds {
		var fieldName = "original_" + amountKind
		var convertedFieldName = "converted_" + amountKind
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationVisibleFinancial,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatMarkdown,
				DocumentRole:        ReportPresentationDocumentRoleAnnex,
				Section:             "currency_conversion_audit",
				AssetIdentity:       "conversion-row",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           fieldName,
				AmountKind:          amountKind,
				AmountOrdinal:       amountOrdinal * 2,
			},
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationVisibleFinancial,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatMarkdown,
				DocumentRole:        ReportPresentationDocumentRoleAnnex,
				Section:             "currency_conversion_audit",
				AssetIdentity:       "conversion-row",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           convertedFieldName,
				AmountKind:          amountKind,
				AmountOrdinal:       amountOrdinal*2 + 1,
			},
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationVisibleFinancial,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatPDF,
				DocumentRole:        ReportPresentationDocumentRoleAnnex,
				Section:             "currency_conversion_audit",
				AssetIdentity:       "conversion-row",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           convertedFieldName,
				AmountKind:          amountKind,
				AmountOrdinal:       amountOrdinal*2 + 1,
			},
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationVisibleFinancial,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatPDF,
				DocumentRole:        ReportPresentationDocumentRoleAnnex,
				Section:             "currency_conversion_audit",
				AssetIdentity:       "conversion-row",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           fieldName,
				AmountKind:          amountKind,
				AmountOrdinal:       amountOrdinal * 2,
			},
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationConvertedEntry,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatMarkdown,
				DocumentRole:        ReportPresentationDocumentRoleAnnex,
				Section:             "currency_conversion_audit",
				AssetIdentity:       "conversion-row",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           amountKind,
				AmountKind:          amountKind,
				AmountOrdinal:       amountOrdinal,
			},
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationConvertedEntry,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatPDF,
				DocumentRole:        ReportPresentationDocumentRoleAnnex,
				Section:             "currency_conversion_audit",
				AssetIdentity:       "conversion-row",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           amountKind,
				AmountKind:          amountKind,
				AmountOrdinal:       amountOrdinal,
			})
	}
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
