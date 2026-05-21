package validator

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	"github.com/cockroachdb/apd/v3"
)

func TestValidateActivityPageResponseAndEntryCoverBranches(t *testing.T) {
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
		{name: "valid date with whitespace", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Date = " 2024-01-01T10:00:00Z "
			return entry
		}(), wantErr: false},
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
		{name: "large exact quantity stays readable", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Quantity = json.Number("1e309")
			return entry
		}(), wantErr: false},
		{name: "unreadable unit price", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.UnitPrice = json.Number("bad")
			return entry
		}(), wantErr: true},
		{name: "unreadable asset-profile unit price", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.UnitPriceInAssetProfileCurrency = json.Number("bad")
			return entry
		}(), wantErr: true},
		{name: "unreadable order fee", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Fee = json.Number("bad")
			return entry
		}(), wantErr: true},
		{name: "large exact optional fee stays readable", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Fee = json.Number("1e309")
			return entry
		}(), wantErr: false},
		{name: "unreadable asset-profile fee", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.FeeInAssetProfileCurrency = json.Number("bad")
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
			entry.UnitPrice = json.Number("")
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
		{name: "only asset-profile unit price basis input", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.UnitPrice = json.Number("")
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
		{name: "gross value basis input requires derivable unit price", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.UnitPrice = json.Number("")
			entry.UnitPriceInAssetProfileCurrency = json.Number("")
			entry.ValueInBaseCurrency = json.Number("")
			entry.Value = json.Number("1")
			entry.Quantity = json.Number("3")
			return entry
		}(), wantErr: true},
		{name: "valid page entry", entry: validActivityPageEntry(), wantErr: false},
		{name: "nullable order currency and comment stay allowed", entry: func() dto.ActivityPageEntry {
			entry := validActivityPageEntry()
			entry.Currency = dto.NullableString("")
			entry.Comment = dto.NullableString("")
			return entry
		}(), wantErr: false},
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
		Currency:                        "CHF",
		Fee:                             json.Number("0.2"),
		UnitPrice:                       json.Number("80"),
		Value:                           json.Number("120"),
		FeeInAssetProfileCurrency:       json.Number("0.22"),
		ValueInBaseCurrency:             json.Number("123.45"),
		FeeInBaseCurrency:               json.Number("0.25"),
		UnitPriceInAssetProfileCurrency: json.Number("82.3"),
		SymbolProfile:                   dto.ActivitySymbolProfile{Symbol: "BTC", Name: "Bitcoin", Currency: "EUR", SymbolProfileID: "asset-btc-validator-001"},
	}
}

// TestBasisDerivationHelpersCoverRemainingBranches verifies the internal basis-
// derivation helpers across their direct error and fallback branches.
// Authored by: OpenCode
func TestBasisDerivationHelpersCoverRemainingBranches(t *testing.T) {
	entry := validActivityPageEntry()
	entry.UnitPrice = json.Number("bad")
	if err := requireBasisInput(entry); err == nil {
		t.Fatalf("expected invalid direct unit-price basis to fail")
	}

	entry = validActivityPageEntry()
	entry.UnitPrice = json.Number("")
	entry.UnitPriceInAssetProfileCurrency = json.Number("")
	entry.Value = json.Number("")
	entry.ValueInBaseCurrency = json.Number("10")
	entry.Quantity = json.Number("2")
	if err := requireDerivableUnitPrice(entry); err != nil {
		t.Fatalf("expected base-currency gross value to derive exact unit price, got %v", err)
	}
	if got := selectGrossValue(entry); got.String() != "10" {
		t.Fatalf("expected base-currency gross value fallback, got %q", got.String())
	}

	entry = validActivityPageEntry()
	entry.UnitPrice = json.Number("")
	entry.UnitPriceInAssetProfileCurrency = json.Number("")
	entry.Quantity = json.Number("bad")
	if err := requireDerivableUnitPrice(entry); err == nil {
		t.Fatalf("expected unreadable quantity to fail unit-price derivation")
	}

	entry = validActivityPageEntry()
	entry.UnitPrice = json.Number("")
	entry.UnitPriceInAssetProfileCurrency = json.Number("")
	entry.Value = json.Number("")
	entry.ValueInBaseCurrency = json.Number("bad")
	if err := requireDerivableUnitPrice(entry); err == nil {
		t.Fatalf("expected unreadable gross value fallback to fail unit-price derivation")
	}

	entry = validActivityPageEntry()
	entry.Value = json.Number("")
	entry.ValueInBaseCurrency = json.Number("")
	entry.Quantity = json.Number("bad")
	if err := requireDerivableGrossValue(entry); err == nil {
		t.Fatalf("expected unreadable quantity to fail gross-value derivation")
	}

	entry = validActivityPageEntry()
	entry.Value = json.Number("")
	entry.ValueInBaseCurrency = json.Number("")
	entry.UnitPrice = json.Number("")
	entry.UnitPriceInAssetProfileCurrency = json.Number("bad")
	if err := requireDerivableGrossValue(entry); err == nil {
		t.Fatalf("expected unreadable unit-price fallback to fail gross-value derivation")
	}

	originalMultiplyDecimals := multiplyDecimals
	multiplyDecimals = func(*apd.Decimal, *apd.Decimal, *apd.Decimal) (apd.Condition, error) {
		return apd.InvalidOperation, errors.New("mul boom")
	}
	defer func() {
		multiplyDecimals = originalMultiplyDecimals
	}()

	entry = validActivityPageEntry()
	entry.Value = json.Number("")
	entry.ValueInBaseCurrency = json.Number("")
	if err := requireBasisInput(entry); err == nil {
		t.Fatalf("expected gross-value derivation failure to propagate from basis validation")
	}

	if err := requireDerivableGrossValue(entry); err == nil {
		t.Fatalf("expected injected multiplication failure to fail gross-value derivation")
	}
}
