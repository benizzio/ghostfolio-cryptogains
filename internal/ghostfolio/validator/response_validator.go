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
)

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
	if _, err := time.Parse(time.RFC3339Nano, entry.Date); err != nil {
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
	if _, err := time.Parse(time.RFC3339Nano, entry.Date); err != nil {
		return fmt.Errorf("activity date must be a readable timestamp: %w", err)
	}
	if strings.TrimSpace(entry.SymbolProfile.Symbol) == "" {
		return fmt.Errorf("activity symbol is required")
	}
	if err := requireJSONNumber(entry.Quantity, "activity quantity"); err != nil {
		return err
	}
	if err := requireBasisInput(entry); err != nil {
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
	if err := requireOptionalJSONNumber(
		entry.UnitPriceInAssetProfileCurrency,
		"activity unit price",
	); err != nil {
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

	if strings.TrimSpace(entry.UnitPriceInAssetProfileCurrency.String()) == "" &&
		strings.TrimSpace(entry.Value.String()) == "" &&
		strings.TrimSpace(entry.ValueInBaseCurrency.String()) == "" {
		return fmt.Errorf("activity basis input is required")
	}

	return nil
}

// requireJSONNumber verifies that one required JSON number is present and
// readable.
// Authored by: OpenCode
func requireJSONNumber(raw json.Number, fieldName string) error {
	if strings.TrimSpace(raw.String()) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if _, err := raw.Float64(); err != nil {
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
	if _, err := raw.Float64(); err != nil {
		return fmt.Errorf("%s must be a readable JSON number: %w", fieldName, err)
	}
	return nil
}
