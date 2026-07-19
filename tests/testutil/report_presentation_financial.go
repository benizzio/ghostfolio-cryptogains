package testutil

// presentationFinancialRow describes one row in the closed financial matrix.
// Authored by: OpenCode
type presentationFinancialRow struct {
	ID              string
	Section         string
	DocumentRole    ReportPresentationDocumentRole
	Signed          bool
	SummaryOmission bool
	Fields          []ReportPresentationFinancialField
}

// presentationFinancialRows returns all nine financial field classes from the
// specification's matrix.
// Authored by: OpenCode
func presentationFinancialRows() []presentationFinancialRow {
	return []presentationFinancialRow{
		{
			ID:              "summary-net-gain-or-loss",
			Section:         "gains_and_losses_summary",
			DocumentRole:    ReportPresentationDocumentRoleMain,
			Signed:          true,
			SummaryOmission: true,
			Fields: []ReportPresentationFinancialField{
				{Name: "per_asset_net_gain_or_loss", AmountKind: "gain_or_loss", AmountOrdinal: 0},
				{Name: "overall_yearly_net_total", AmountKind: "gain_or_loss", AmountOrdinal: 1},
			},
		},
		{
			ID:           "position-cost-basis",
			Section:      "position",
			DocumentRole: ReportPresentationDocumentRoleMain,
			Fields: []ReportPresentationFinancialField{
				{Name: "opening_cost_basis", AmountKind: "cost_basis", AmountOrdinal: 0},
				{Name: "closing_cost_basis", AmountKind: "cost_basis", AmountOrdinal: 1},
				{Name: "historical_cost_basis", AmountKind: "cost_basis", AmountOrdinal: 2},
			},
		},
		{
			ID:           "in-year-activity",
			Section:      "in_year_activity",
			DocumentRole: ReportPresentationDocumentRoleMain,
			Fields: []ReportPresentationFinancialField{
				{Name: "unit_price", AmountKind: "unit_price", AmountOrdinal: 0, Nullable: true},
				{Name: "gross_value", AmountKind: "gross_value", AmountOrdinal: 1, Nullable: true},
				{Name: "fee_amount", AmountKind: "fee_amount", AmountOrdinal: 2, Nullable: true},
				{Name: "basis_after_row", AmountKind: "cost_basis", AmountOrdinal: 3},
			},
		},
		{
			ID:           "liquidation-allocated-basis",
			Section:      "liquidation_calculations",
			DocumentRole: ReportPresentationDocumentRoleMain,
			Fields:       []ReportPresentationFinancialField{{Name: "allocated_basis", AmountKind: "cost_basis", AmountOrdinal: 0}},
		},
		{
			ID:           "liquidation-net-proceeds-gain-or-loss",
			Section:      "liquidation_calculations",
			DocumentRole: ReportPresentationDocumentRoleMain,
			Signed:       true,
			Fields: []ReportPresentationFinancialField{
				{Name: "net_proceeds", AmountKind: "proceeds", AmountOrdinal: 0},
				{Name: "gain_or_loss", AmountKind: "gain_or_loss", AmountOrdinal: 1},
			},
		},
		{
			ID:           "audit-activity",
			Section:      "detailed_per_asset_audit",
			DocumentRole: ReportPresentationDocumentRoleAnnex,
			Fields: []ReportPresentationFinancialField{
				{Name: "unit_price", AmountKind: "unit_price", AmountOrdinal: 0, Nullable: true},
				{Name: "gross_value", AmountKind: "gross_value", AmountOrdinal: 1, Nullable: true},
				{Name: "fee_amount", AmountKind: "fee_amount", AmountOrdinal: 2, Nullable: true},
				{Name: "basis_after_activity", AmountKind: "cost_basis", AmountOrdinal: 3},
			},
		},
		{
			ID:           "audit-allocated-basis",
			Section:      "detailed_per_asset_audit",
			DocumentRole: ReportPresentationDocumentRoleAnnex,
			Fields:       []ReportPresentationFinancialField{{Name: "allocated_basis", AmountKind: "cost_basis", AmountOrdinal: 0, Nullable: true}},
		},
		{
			ID:           "audit-net-proceeds-gain-or-loss",
			Section:      "detailed_per_asset_audit",
			DocumentRole: ReportPresentationDocumentRoleAnnex,
			Signed:       true,
			Fields: []ReportPresentationFinancialField{
				{Name: "net_proceeds", AmountKind: "proceeds", AmountOrdinal: 0, Nullable: true},
				{Name: "gain_or_loss", AmountKind: "gain_or_loss", AmountOrdinal: 1, Nullable: true},
			},
		},
		{
			ID:           "conversion-amount",
			Section:      "currency_conversion_audit",
			DocumentRole: ReportPresentationDocumentRoleAnnex,
			Fields: []ReportPresentationFinancialField{
				{Name: "original_unit_price", AmountKind: "unit_price", AmountOrdinal: 0},
				{Name: "converted_unit_price", AmountKind: "unit_price", AmountOrdinal: 1},
				{Name: "original_gross_value", AmountKind: "gross_value", AmountOrdinal: 2},
				{Name: "converted_gross_value", AmountKind: "gross_value", AmountOrdinal: 3},
				{Name: "original_fee_amount", AmountKind: "fee_amount", AmountOrdinal: 4},
				{Name: "converted_fee_amount", AmountKind: "fee_amount", AmountOrdinal: 5},
			},
		},
	}
}

// presentationNonNegativeVectors returns the complete non-negative financial
// rounding vector in specification order.
// Authored by: OpenCode
func presentationNonNegativeVectors() []presentationScalarCase {
	return []presentationScalarCase{
		{ID: "zero", ExactValue: "0", ExpectedValue: "0.00"},
		{ID: "tiny-positive", ExactValue: "0.00000001", ExpectedValue: "0.00"},
		{ID: "whole", ExactValue: "1", ExpectedValue: "1.00"},
		{ID: "one-place", ExactValue: "1.2", ExpectedValue: "1.20"},
		{ID: "two-place", ExactValue: "1.23", ExpectedValue: "1.23"},
		{ID: "below-positive-tie", ExactValue: "1.004", ExpectedValue: "1.00"},
		{ID: "positive-tie", ExactValue: "1.005", ExpectedValue: "1.01"},
		{ID: "above-positive-tie", ExactValue: "1.006", ExpectedValue: "1.01"},
		{ID: "positive-carry", ExactValue: "9.995", ExpectedValue: "10.00"},
		{ID: "large-positive", ExactValue: "12345678901234567890.123456789", ExpectedValue: "12345678901234567890.12"},
	}
}

// presentationSignedVectors extends the non-negative vector with every signed
// boundary required by the financial display contract.
// Authored by: OpenCode
func presentationSignedVectors() []presentationScalarCase {
	var vectors = append([]presentationScalarCase(nil), presentationNonNegativeVectors()...)
	vectors = append(vectors,
		presentationScalarCase{ID: "negative-whole", ExactValue: "-1", ExpectedValue: "-1.00"},
		presentationScalarCase{ID: "negative-below-tie", ExactValue: "-1.004", ExpectedValue: "-1.00"},
		presentationScalarCase{ID: "negative-tie", ExactValue: "-1.005", ExpectedValue: "-1.01"},
		presentationScalarCase{ID: "negative-above-tie", ExactValue: "-1.006", ExpectedValue: "-1.01"},
		presentationScalarCase{ID: "negative-carry", ExactValue: "-9.995", ExpectedValue: "-10.00"},
		presentationScalarCase{ID: "signed-zero", ExactValue: "-0", ExpectedValue: "0.00"},
		presentationScalarCase{ID: "negative-tiny", ExactValue: "-0.004", ExpectedValue: "0.00"},
		presentationScalarCase{ID: "negative-zero-adjacent-tie", ExactValue: "-0.005", ExpectedValue: "-0.01"},
		presentationScalarCase{ID: "large-negative", ExactValue: "-12345678901234567890.123456789", ExpectedValue: "-12345678901234567890.12"},
	)
	return vectors
}
