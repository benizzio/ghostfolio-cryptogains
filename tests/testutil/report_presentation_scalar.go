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
	acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys, ReportPresentationOccurrenceKey{
		Population:          ReportPresentationPopulationParity,
		CaseID:              acceptanceCase.ID,
		Format:              reportPresentationFormatCrossFormat,
		DocumentRole:        reportPresentationDocumentRoleModel,
		Section:             "acceptance_control",
		AssetIdentity:       "acceptance",
		SourceOrRowIdentity: fieldName,
	})
	if kind == ReportPresentationCaseKindQuantity {
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationQuantity,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatMarkdown,
				DocumentRole:        ReportPresentationDocumentRoleMain,
				Section:             "quantity_controls",
				AssetIdentity:       "acceptance",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           fieldName,
			},
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationQuantity,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatPDF,
				DocumentRole:        ReportPresentationDocumentRoleMain,
				Section:             "quantity_controls",
				AssetIdentity:       "acceptance",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           fieldName,
			})
	}
	if kind == ReportPresentationCaseKindBoolean {
		acceptanceCase.OccurrenceKeys = append(acceptanceCase.OccurrenceKeys,
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationBoolean,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatMarkdown,
				DocumentRole:        ReportPresentationDocumentRoleAnnex,
				Section:             "detailed_per_asset_audit",
				AssetIdentity:       "acceptance",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           fieldName,
			},
			ReportPresentationOccurrenceKey{
				Population:          ReportPresentationPopulationBoolean,
				CaseID:              acceptanceCase.ID,
				Format:              ReportPresentationFormatPDF,
				DocumentRole:        ReportPresentationDocumentRoleAnnex,
				Section:             "detailed_per_asset_audit",
				AssetIdentity:       "acceptance",
				SourceOrRowIdentity: acceptanceCase.ID,
				FieldName:           fieldName,
			})
	}
	return acceptanceCase
}

// presentationScalarCase stores one exact-value control used outside the
// financial matrix.
// Authored by: OpenCode
type presentationScalarCase struct {
	ID            string
	ExactValue    string
	ExpectedValue string
}
