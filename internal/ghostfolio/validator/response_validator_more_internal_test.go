package validator

import (
	"encoding/json"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
)

func TestValidateActivityPageResponseAndEntryCoverBranches(t *testing.T) {
	t.Parallel()

	responseCases := []struct {
		name     string
		response dto.ActivityPageResponse
		wantErr  bool
	}{
		{name: "negative count", response: dto.ActivityPageResponse{Count: -1}, wantErr: true},
		{name: "missing activities", response: dto.ActivityPageResponse{Count: 0}, wantErr: true},
		{name: "zero count with activities", response: dto.ActivityPageResponse{Count: 0, Activities: []dto.ActivityPageEntry{validActivityPageEntry()}}, wantErr: true},
		{name: "positive count without activities", response: dto.ActivityPageResponse{Count: 1, Activities: []dto.ActivityPageEntry{}}, wantErr: true},
		{name: "activities exceed count", response: dto.ActivityPageResponse{Count: 1, Activities: []dto.ActivityPageEntry{validActivityPageEntry(), validActivityPageEntry()}}, wantErr: true},
		{name: "valid empty history", response: dto.ActivityPageResponse{Count: 0, Activities: []dto.ActivityPageEntry{}}, wantErr: false},
		{name: "valid page", response: dto.ActivityPageResponse{Count: 1, Activities: []dto.ActivityPageEntry{validActivityPageEntry()}}, wantErr: false},
	}

	for _, testCase := range responseCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateActivityPageResponse(testCase.response)
			if testCase.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("expected validation success, got %v", err)
			}
		})
	}

	entryCases := []struct {
		name    string
		entry   dto.ActivityPageEntry
		wantErr bool
	}{
		{name: "missing id", entry: dto.ActivityPageEntry{}, wantErr: true},
		{name: "missing type", entry: dto.ActivityPageEntry{ID: "1"}, wantErr: true},
		{name: "missing date", entry: dto.ActivityPageEntry{ID: "1", Type: "BUY"}, wantErr: true},
		{name: "invalid date", entry: dto.ActivityPageEntry{ID: "1", Type: "BUY", Date: "bad"}, wantErr: true},
		{name: "missing symbol", entry: dto.ActivityPageEntry{ID: "1", Type: "BUY", Date: "2024-01-01T10:00:00Z"}, wantErr: true},
		{name: "missing quantity", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Quantity = json.Number("")
			return entry
		}(), wantErr: true},
		{name: "unreadable quantity", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Quantity = json.Number("bad")
			return entry
		}(), wantErr: true},
		{name: "unreadable unit price", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.UnitPriceInAssetProfileCurrency = json.Number("bad")
			return entry
		}(), wantErr: true},
		{name: "unreadable fee", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.FeeInBaseCurrency = json.Number("bad")
			return entry
		}(), wantErr: true},
		{name: "unreadable value in base currency", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.ValueInBaseCurrency = json.Number("bad")
			return entry
		}(), wantErr: true},
		{name: "missing basis inputs", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.UnitPriceInAssetProfileCurrency = json.Number("")
			entry.Value = json.Number("")
			entry.ValueInBaseCurrency = json.Number("")
			return entry
		}(), wantErr: true},
		{name: "unreadable basis input", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Value = json.Number("bad")
			return entry
		}(), wantErr: true},
		{name: "only unit price basis input", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Value = json.Number("")
			entry.ValueInBaseCurrency = json.Number("")
			return entry
		}(), wantErr: false},
		{name: "only value basis input", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.UnitPriceInAssetProfileCurrency = json.Number("")
			entry.ValueInBaseCurrency = json.Number("")
			return entry
		}(), wantErr: false},
		{name: "only base-currency value basis input", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.UnitPriceInAssetProfileCurrency = json.Number("")
			entry.Value = json.Number("")
			return entry
		}(), wantErr: false},
		{name: "valid page entry", entry: validActivityPageEntry(), wantErr: false},
	}

	for _, testCase := range entryCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateActivityPageEntry(testCase.entry)
			if testCase.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("expected validation success, got %v", err)
			}
		})
	}
}

func validActivityPageEntry() dto.ActivityPageEntry {
	return dto.ActivityPageEntry{
		ID:                              "activity-1",
		Date:                            "2024-01-01T10:00:00Z",
		Type:                            "BUY",
		Quantity:                        json.Number("1.5"),
		Value:                           json.Number("120"),
		ValueInBaseCurrency:             json.Number("123.45"),
		FeeInBaseCurrency:               json.Number("0.25"),
		UnitPriceInAssetProfileCurrency: json.Number("82.3"),
		SymbolProfile:                   dto.ActivitySymbolProfile{Symbol: "BTC", Name: "Bitcoin"},
	}
}
