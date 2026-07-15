// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// validateCostBasisMethod rejects unsupported report cost-basis method values.
// Authored by: OpenCode
func validateCostBasisMethod(method CostBasisMethod) error {
	for _, supportedMethod := range SupportedCostBasisMethods() {
		if method == supportedMethod {
			return nil
		}
	}

	return fmt.Errorf("unsupported cost basis method %q", method)
}

// validateReferenceSectionStatus rejects unsupported reference-entry statuses.
// Authored by: OpenCode
func validateReferenceSectionStatus(status ReferenceSectionStatus) error {
	switch status {
	case ReferenceSectionStatusIncludedInMainSections, ReferenceSectionStatusReferenceOnly:
		return nil
	default:
		return fmt.Errorf("unsupported reference section status %q", status)
	}
}

// validateReportDocumentType rejects unsupported rendered-document types.
// Authored by: OpenCode
func validateReportDocumentType(documentType ReportDocumentType) error {
	switch documentType {
	case ReportDocumentTypeMarkdown, ReportDocumentTypePDF:
		return nil
	default:
		return fmt.Errorf("unsupported report document type %q", documentType)
	}
}

// validateReportDocumentRole rejects unsupported rendered-document roles.
// Authored by: OpenCode
func validateReportDocumentRole(role ReportDocumentRole) error {
	switch role {
	case ReportDocumentRoleMain, ReportDocumentRoleAnnex, ReportDocumentRoleCombined:
		return nil
	default:
		return fmt.Errorf("unsupported report document role %q", role)
	}
}

// validateActivityType rejects unsupported activity-row activity types.
// Authored by: OpenCode
func validateActivityType(activityType ActivityType) error {
	switch activityType {
	case ActivityTypeBuy, ActivityTypeSell:
		return nil
	default:
		return fmt.Errorf("unsupported activity type %q", activityType)
	}
}

// validateOptionalDecimal verifies one optional exact decimal value.
// Authored by: OpenCode
func validateOptionalDecimal(value *apd.Decimal, label string) error {
	if value == nil {
		return nil
	}

	return validateFiniteDecimal(*value, label)
}

// validatePositiveDecimal verifies one positive exact decimal value.
// Authored by: OpenCode
func validatePositiveDecimal(value apd.Decimal, label string) error {
	if err := supportmath.RequirePositive(value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	return nil
}

// validateNonNegativeDecimal verifies one non-negative exact decimal value.
// Authored by: OpenCode
func validateNonNegativeDecimal(value apd.Decimal, label string) error {
	if err := supportmath.RequireNonNegative(value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	return nil
}

// validateFiniteDecimal verifies one finite exact decimal value.
// Authored by: OpenCode
func validateFiniteDecimal(value apd.Decimal, label string) error {
	if err := supportmath.RequireFinite(value); err != nil {
		return fmt.Errorf("%s is invalid: %w", label, err)
	}

	return nil
}
