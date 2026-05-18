package contract

import (
	"encoding/json"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/validator"
)

func TestGhostfolioSyncStorageContract(t *testing.T) {
	t.Parallel()

	if err := validator.ValidateAuthResponse(dto.AuthResponse{AuthToken: "jwt"}); err != nil {
		t.Fatalf("expected auth response to satisfy contract: %v", err)
	}

	if err := validator.ValidateUserResponse(dto.UserResponse{Settings: &dto.UserSettings{BaseCurrency: "USD"}}); err != nil {
		t.Fatalf("expected user response to satisfy contract: %v", err)
	}
	if err := validator.ValidateUserResponse(dto.UserResponse{Settings: &dto.UserSettings{BaseCurrency: ""}}); err != nil {
		t.Fatalf("expected missing user base currency to satisfy contract: %v", err)
	}

	if err := validator.ValidateActivityPageResponse(dto.ActivityPageResponse{
		Count: 2,
		Activities: []dto.ActivityPageEntry{
			{
				ID:                              "activity-1",
				Date:                            "2026-01-31T10:00:00+01:00",
				Type:                            "BUY",
				Quantity:                        json.Number("1.25"),
				Currency:                        dto.NullableString("CHF"),
				Fee:                             json.Number("20"),
				UnitPrice:                       json.Number("49000"),
				Value:                           json.Number("61250"),
				FeeInAssetProfileCurrency:       json.Number("18"),
				ValueInBaseCurrency:             json.Number("62500"),
				FeeInBaseCurrency:               json.Number("25"),
				UnitPriceInAssetProfileCurrency: json.Number("50000"),
				SymbolProfile:                   dto.ActivitySymbolProfile{Symbol: "BTC", Name: "Bitcoin", Currency: "EUR"},
				Account:                         &dto.ActivityAccountScope{ID: "account-1", Name: "Main"},
			},
			{
				ID:                              "activity-2",
				Date:                            "2026-02-01T09:00:00Z",
				Type:                            "SELL",
				Quantity:                        json.Number("0.25"),
				Currency:                        dto.NullableString(""),
				Value:                           json.Number("14800"),
				ValueInBaseCurrency:             json.Number("15000"),
				UnitPriceInAssetProfileCurrency: json.Number("60000"),
				Comment:                         dto.NullableString(""),
				SymbolProfile:                   dto.ActivitySymbolProfile{Symbol: "BTC", Name: "Bitcoin", Currency: ""},
			},
		},
	}); err != nil {
		t.Fatalf("expected paginated activity response to satisfy contract: %v", err)
	}

	if err := validator.ValidateActivityPageResponse(dto.ActivityPageResponse{Count: 0, Activities: []dto.ActivityPageEntry{}}); err != nil {
		t.Fatalf("expected empty history response to satisfy contract: %v", err)
	}
}
