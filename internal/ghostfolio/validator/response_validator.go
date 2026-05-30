// Package validator validates the minimal Ghostfolio response contract used by
// the sync-and-storage slice.
// Authored by: OpenCode
package validator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/cockroachdb/apd/v3"
)

// Test seams wrap decimal multiplication so validator tests can inject gross-
// value derivation failures safely.
// Authored by: OpenCode
var multiplyDecimals = apd.BaseContext.Mul

// ValidateAuthResponse verifies that the anonymous-auth response satisfies the
// supported contract for this slice.
//
// Example:
//
//	err := validator.ValidateAuthResponse(dto.AuthResponse{AuthToken: "jwt"})
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateAuthResponse(response dto.AuthResponse) error {
	if strings.TrimSpace(response.AuthToken) == "" {
		return fmt.Errorf("authToken is required")
	}
	return nil
}

// ValidateUserResponse verifies that the authenticated user response satisfies
// the supported contract for this slice.
//
// Example:
//
//	err := validator.ValidateUserResponse(dto.UserResponse{Settings: &dto.UserSettings{BaseCurrency: "USD"}})
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateUserResponse(response dto.UserResponse) error {
	if response.Settings == nil {
		return fmt.Errorf("user settings are required")
	}

	return nil
}

// ValidateActivityPageResponse verifies that one paginated activities response
// satisfies the supported full-history contract for this slice.
//
// Example:
//
//	err := validator.ValidateActivityPageResponse(dto.ActivityPageResponse{Count: 0, Activities: []dto.ActivityPageEntry{}})
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateActivityPageResponse(response dto.ActivityPageResponse) error {
	if response.Count < 0 {
		return fmt.Errorf("count must be non-negative")
	}
	if response.Activities == nil {
		return fmt.Errorf("activities must be present")
	}
	if response.Count == 0 && len(response.Activities) > 0 {
		return fmt.Errorf("activities must be empty when count is zero")
	}
	if response.Count > 0 && len(response.Activities) == 0 {
		return fmt.Errorf("activities must include at least one item when count is positive")
	}
	if len(response.Activities) > response.Count && response.Count >= 0 {
		return fmt.Errorf("activities cannot exceed count")
	}

	for _, entry := range response.Activities {
		if err := ValidateActivityPageEntry(entry); err != nil {
			return err
		}
	}

	return nil
}

// ValidateSingleActivityPageResponse verifies that one single-page activities
// response satisfies the supported contract for focused contract checks.
//
// Example:
//
//	err := validator.ValidateSingleActivityPageResponse(dto.ActivityPageResponse{Count: 0, Activities: []dto.ActivityPageEntry{}})
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateSingleActivityPageResponse(response dto.ActivityPageResponse) error {
	if response.Count < 0 {
		return fmt.Errorf("count must be non-negative")
	}
	if response.Activities == nil {
		return fmt.Errorf("activities must be present")
	}
	if len(response.Activities) > 1 {
		return fmt.Errorf("single-page activities check must return at most one activity")
	}
	if response.Count == 0 && len(response.Activities) > 0 {
		return fmt.Errorf("activities must be empty when count is zero")
	}
	if response.Count > 0 && len(response.Activities) == 0 {
		return fmt.Errorf("activities must include the first item when count is positive")
	}
	if len(response.Activities) == 0 {
		return nil
	}

	var entry = response.Activities[0]
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("activity id is required")
	}
	if strings.TrimSpace(entry.Type) == "" {
		return fmt.Errorf("activity type is required")
	}
	if strings.TrimSpace(entry.Date) == "" {
		return fmt.Errorf("activity date is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(entry.Date)); err != nil {
		return fmt.Errorf("activity date must be a readable timestamp: %w", err)
	}

	return nil
}

// ValidateActivityPageEntry verifies that one paginated Ghostfolio activity
// contains the minimum supported normalized inputs for this slice.
//
// Authored by: OpenCode
func ValidateActivityPageEntry(entry dto.ActivityPageEntry) error {
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("activity id is required")
	}
	if strings.TrimSpace(entry.Type) == "" {
		return fmt.Errorf("activity type is required")
	}
	if strings.TrimSpace(entry.Date) == "" {
		return fmt.Errorf("activity date is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(entry.Date)); err != nil {
		return fmt.Errorf("activity date must be a readable timestamp: %w", err)
	}
	if strings.TrimSpace(entry.SymbolProfile.Symbol) == "" {
		return fmt.Errorf("activity symbol is required")
	}
	if err := requireJSONNumber(entry.Quantity, "activity quantity"); err != nil {
		return err
	}
	if err := requireOptionalJSONNumber(entry.UnitPrice, "activity unit price"); err != nil {
		return err
	}
	if err := requireBasisInput(entry); err != nil {
		return err
	}
	if err := requireOptionalJSONNumber(entry.Fee, "activity order fee"); err != nil {
		return err
	}
	if err := requireOptionalJSONNumber(entry.FeeInAssetProfileCurrency, "activity asset-profile fee"); err != nil {
		return err
	}
	if err := requireOptionalJSONNumber(entry.FeeInBaseCurrency, "activity fee"); err != nil {
		return err
	}

	return nil
}

// requireBasisInput verifies that at least one supported basis input is present
// and readable for later normalization.
// Authored by: OpenCode
func requireBasisInput(entry dto.ActivityPageEntry) error {
	if err := requireOptionalJSONNumber(entry.UnitPrice, "activity order unit price"); err != nil {
		return err
	}
	if err := requireOptionalJSONNumber(entry.UnitPriceInAssetProfileCurrency, "activity asset-profile unit price"); err != nil {
		return err
	}
	if err := requireOptionalJSONNumber(entry.Value, "activity value"); err != nil {
		return err
	}
	if err := requireOptionalJSONNumber(
		entry.ValueInBaseCurrency,
		"activity value in base currency",
	); err != nil {
		return err
	}

	if !hasUnitPriceInput(entry) &&
		strings.TrimSpace(entry.Value.String()) == "" &&
		strings.TrimSpace(entry.ValueInBaseCurrency.String()) == "" {
		return fmt.Errorf("activity basis input is required")
	}
	if !hasUnitPriceInput(entry) {
		if err := requireDerivableUnitPrice(entry); err != nil {
			return err
		}
	}
	if !hasGrossValueInput(entry) {
		if err := requireDerivableGrossValue(entry); err != nil {
			return err
		}
	}

	return nil
}

// hasUnitPriceInput reports whether the activity exposes either supported unit-price field.
// Authored by: OpenCode
func hasUnitPriceInput(entry dto.ActivityPageEntry) bool {
	return strings.TrimSpace(entry.UnitPrice.String()) != "" ||
		strings.TrimSpace(entry.UnitPriceInAssetProfileCurrency.String()) != ""
}

// hasGrossValueInput reports whether the activity exposes either supported gross-value field.
// Authored by: OpenCode
func hasGrossValueInput(entry dto.ActivityPageEntry) bool {
	return strings.TrimSpace(entry.Value.String()) != "" ||
		strings.TrimSpace(entry.ValueInBaseCurrency.String()) != ""
}

// requireDerivableUnitPrice ensures that gross-value-only inputs can still be
// normalized into an exact unit price.
// Authored by: OpenCode
func requireDerivableUnitPrice(entry dto.ActivityPageEntry) error {
	quantity, _, err := decimalsupport.ParseNumber(entry.Quantity)
	if err != nil {
		return fmt.Errorf("activity quantity must remain readable for basis derivation: %w", err)
	}
	grossValue, _, err := decimalsupport.ParseNumber(selectGrossValue(entry))
	if err != nil {
		return fmt.Errorf("activity basis input must support unit-price derivation: %w", err)
	}
	if _, err := deriveRoundedUnitPrice(grossValue, quantity); err != nil {
		return fmt.Errorf("activity basis input must support unit-price derivation: %w", err)
	}

	return nil
}

// requireDerivableGrossValue ensures that unit-price-only inputs can still be
// normalized into an exact gross value.
// Authored by: OpenCode
func requireDerivableGrossValue(entry dto.ActivityPageEntry) error {
	quantity, _, err := decimalsupport.ParseNumber(entry.Quantity)
	if err != nil {
		return fmt.Errorf("activity quantity must remain readable for basis derivation: %w", err)
	}
	unitPrice, _, err := decimalsupport.ParseNumber(selectUnitPrice(entry))
	if err != nil {
		return fmt.Errorf("activity unit price must support exact gross-value derivation: %w", err)
	}
	var product apd.Decimal
	if _, err := multiplyDecimals(&product, &quantity, &unitPrice); err != nil {
		return fmt.Errorf("activity unit price must support exact gross-value derivation: %w", err)
	}

	return nil
}

// deriveRoundedUnitPrice accepts finite same-tier division inputs that may need
// 16-decimal round-half-up handling later in reporting.
// Authored by: OpenCode
func deriveRoundedUnitPrice(grossValue apd.Decimal, quantity apd.Decimal) (apd.Decimal, error) {
	if err := supportmath.RequireFinite(grossValue, "gross value for unit-price derivation"); err != nil {
		return apd.Decimal{}, fmt.Errorf("prepare gross value for unit-price derivation: %w", err)
	}
	if err := supportmath.RequireFinite(quantity, "quantity for unit-price derivation"); err != nil {
		return apd.Decimal{}, fmt.Errorf("prepare quantity for unit-price derivation: %w", err)
	}
	if quantity.Sign() == 0 {
		return apd.Decimal{}, fmt.Errorf("unit-price derivation requires a non-zero quantity")
	}

	return supportmath.DivideFiniteRoundHalfUp(grossValue, quantity, supportmath.InternalCalculationScale)
}

// selectGrossValue applies the shared DTO gross-value fallback rule for
// contract validation.
// Authored by: OpenCode
func selectGrossValue(entry dto.ActivityPageEntry) json.Number {
	if strings.TrimSpace(entry.Value.String()) != "" {
		return entry.Value
	}

	return entry.ValueInBaseCurrency
}

// selectUnitPrice applies the shared DTO unit-price fallback rule for contract validation.
// Authored by: OpenCode
func selectUnitPrice(entry dto.ActivityPageEntry) json.Number {
	if strings.TrimSpace(entry.UnitPrice.String()) != "" {
		return entry.UnitPrice
	}

	return entry.UnitPriceInAssetProfileCurrency
}

// requireJSONNumber verifies that one required JSON number is present and
// readable.
// Authored by: OpenCode
func requireJSONNumber(raw json.Number, fieldName string) error {
	if strings.TrimSpace(raw.String()) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if _, _, err := decimalsupport.ParseNumber(raw); err != nil {
		return fmt.Errorf("%s must be a readable JSON number: %w", fieldName, err)
	}
	return nil
}

// requireOptionalJSONNumber verifies that one optional JSON number remains
// readable when present.
// Authored by: OpenCode
func requireOptionalJSONNumber(raw json.Number, fieldName string) error {
	if strings.TrimSpace(raw.String()) == "" {
		return nil
	}
	if _, _, err := decimalsupport.ParseNumber(raw); err != nil {
		return fmt.Errorf("%s must be a readable JSON number: %w", fieldName, err)
	}
	return nil
}
