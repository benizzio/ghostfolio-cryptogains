package unit

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/validator"
)

func TestValidateAuthResponseRequiresToken(t *testing.T) {
	t.Parallel()

	if err := validator.ValidateAuthResponse(dto.AuthResponse{}); err == nil {
		t.Fatalf("expected auth validation error")
	}
}

func TestValidateActivitiesProbeAllowsEmptyHistory(t *testing.T) {
	t.Parallel()

	var err = validator.ValidateActivitiesProbeResponse(dto.ActivitiesProbeResponse{Count: 0, Activities: []dto.ActivityProbeEntry{}})
	if err != nil {
		t.Fatalf("expected empty history to be valid: %v", err)
	}
}

func TestValidateActivitiesProbeRejectsContradictoryCount(t *testing.T) {
	t.Parallel()

	var err = validator.ValidateActivitiesProbeResponse(dto.ActivitiesProbeResponse{
		Count:      0,
		Activities: []dto.ActivityProbeEntry{{ID: "id", Date: "2026-01-31T10:00:00Z", Type: "BUY"}},
	})
	if err == nil {
		t.Fatalf("expected contradictory count validation error")
	}
}

func TestValidateActivitiesProbeRejectsUnreadableTimestamp(t *testing.T) {
	t.Parallel()

	var err = validator.ValidateActivitiesProbeResponse(dto.ActivitiesProbeResponse{
		Count:      1,
		Activities: []dto.ActivityProbeEntry{{ID: "id", Date: "not-a-date", Type: "BUY"}},
	})
	if err == nil {
		t.Fatalf("expected timestamp validation error")
	}
}
