// Package validator validates the minimal Ghostfolio response contract used by
// the sync-validation slice.
// Authored by: OpenCode
package validator

import (
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

// ValidateActivitiesProbeResponse verifies that the one-page activities probe
// satisfies the supported contract for this slice.
//
// Example:
//
//	err := validator.ValidateActivitiesProbeResponse(dto.ActivitiesProbeResponse{Count: 0, Activities: []dto.ActivityProbeEntry{}})
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ValidateActivitiesProbeResponse(response dto.ActivitiesProbeResponse) error {
	if response.Count < 0 {
		return fmt.Errorf("count must be non-negative")
	}
	if len(response.Activities) > 1 {
		return fmt.Errorf("activities probe must return at most one activity")
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
