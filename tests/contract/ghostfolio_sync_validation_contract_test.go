package contract

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/validator"
)

func TestGhostfolioSyncValidationContract(t *testing.T) {
	t.Parallel()

	if err := validator.ValidateAuthResponse(dto.AuthResponse{AuthToken: "jwt"}); err != nil {
		t.Fatalf("expected auth response to satisfy contract: %v", err)
	}

	if err := validator.ValidateSingleActivityPageResponse(dto.ActivityPageResponse{
		Count:      1,
		Activities: []dto.ActivityPageEntry{{ID: "activity-id", Date: "2026-01-31T10:00:00Z", Type: "BUY"}},
	}); err != nil {
		t.Fatalf("expected single-page activities response to satisfy contract: %v", err)
	}
}
