package validate

import (
	"errors"
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

func TestValidationErrorHelpersAndValidationBranches(t *testing.T) {
	t.Parallel()

	var nilError *ValidationError
	if nilError.Error() != "" {
		t.Fatalf("expected nil validation error string to be empty")
	}
	var context = nilError.DiagnosticContext()
	if context.FailureStage != "" || context.FailureDetail != "" || len(context.Records) != 0 {
		t.Fatalf("expected nil validation error context to be empty, got %#v", context)
	}

	validator := NewValidator()
	cache := syncmodel.ProtectedActivityCache{ActivityCount: 2, Activities: []syncmodel.ActivityRecord{validationTestRecord(t, "activity-1", syncmodel.ActivityTypeBuy)}}
	if err := validator.Validate(cache); err == nil {
		t.Fatalf("expected activity-count mismatch to fail")
	}

	cache = syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{{SourceID: "activity-1", OccurredAt: "bad-time"}}}
	if err := validator.Validate(cache); err == nil {
		t.Fatalf("expected unreadable timestamp to fail")
	}

	cache = syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{{SourceID: "activity-1", OccurredAt: "   "}}}
	if err := validator.Validate(cache); err == nil {
		t.Fatalf("expected blank timestamp to fail")
	} else if !strings.Contains(err.Error(), "timestamp is incomplete") {
		t.Fatalf("expected blank timestamp error, got %v", err)
	}

	buy := validationTestRecord(t, "buy-1", syncmodel.ActivityTypeBuy)
	sell := validationTestRecord(t, "sell-1", syncmodel.ActivityTypeSell)
	sell.Comment = "zero-price explanation"
	zeroPrice, _, err := decimalsupport.ParseString("0")
	if err != nil {
		t.Fatalf("parse zero price: %v", err)
	}
	negativeValue, _, err := decimalsupport.ParseString("-1")
	if err != nil {
		t.Fatalf("parse negative value: %v", err)
	}

	invalidCases := []struct {
		name  string
		cache syncmodel.ProtectedActivityCache
	}{
		{name: "missing identity", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{{OccurredAt: "2024-01-01T10:00:00Z"}}}},
		{name: "non-positive quantity", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.Quantity = zeroPrice; return record }()}}},
		{name: "negative gross value", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.GrossValue = negativeValue; return record }()}}},
		{name: "negative fee", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.FeeAmount = &negativeValue; return record }()}}},
		{name: "buy zero unit price", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.UnitPrice = zeroPrice; return record }()}}},
		{name: "sell negative unit price", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := sell; record.UnitPrice = negativeValue; return record }()}}},
		{name: "sell zero price without comment", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord {
			record := sell
			record.UnitPrice = zeroPrice
			record.Comment = ""
			return record
		}()}}},
		{name: "unsupported type", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord {
			record := buy
			record.ActivityType = syncmodel.ActivityType("TRANSFER")
			return record
		}()}}},
		{name: "below zero holdings", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{sell}}},
	}

	for _, testCase := range invalidCases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validator.Validate(testCase.cache); err == nil {
				t.Fatalf("expected validation failure")
			}
		})
	}

	orderedCache := syncmodel.ProtectedActivityCache{ActivityCount: 2, Activities: []syncmodel.ActivityRecord{sell, buy}}
	if err := validator.Validate(orderedCache); err == nil {
		t.Fatalf("expected out-of-order records to fail")
	}

	chronologyLeft := validationTestRecord(t, "source-a", syncmodel.ActivityTypeBuy)
	chronologyRight := validationTestRecord(t, "source-b", syncmodel.ActivityTypeBuy)
	chronologyLeft.RawHash = "a"
	chronologyRight.RawHash = "b"
	chronologyLeft.OccurredAt = "2024-01-01T11:00:00Z"
	chronologyRight.OccurredAt = "2024-01-01T10:00:00Z"
	chronologyLeft.SourceID = "same-source"
	chronologyRight.SourceID = "same-source"
	if err := validator.Validate(syncmodel.ProtectedActivityCache{ActivityCount: 2, Activities: []syncmodel.ActivityRecord{chronologyLeft, chronologyRight}}); err == nil {
		t.Fatalf("expected chronological-order violation to fail")
	}

	ambiguousCache := syncmodel.ProtectedActivityCache{ActivityCount: 2, Activities: []syncmodel.ActivityRecord{buy, func() syncmodel.ActivityRecord { record := buy; record.RawHash = "different"; return record }()}}
	if err := validator.Validate(ambiguousCache); err == nil {
		t.Fatalf("expected ambiguous same-source ordering to fail")
	}

	validCache := syncmodel.ProtectedActivityCache{ActivityCount: 2, Activities: []syncmodel.ActivityRecord{buy, sell}}
	if err := validator.Validate(validCache); err != nil {
		t.Fatalf("expected valid cache to pass, got %v", err)
	}

	if compareValidationOrderKeys(validationOrderKey{SourceDate: "2024-01-01"}, validationOrderKey{SourceDate: "2024-01-02"}) >= 0 {
		t.Fatalf("expected source date comparison branch")
	}
	if compareValidationOrderKeys(validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC"}, validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "ETH"}) >= 0 {
		t.Fatalf("expected asset-symbol comparison branch")
	}
	if compareValidationOrderKeys(validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityOrder: 0}, validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityOrder: 1}) >= 0 {
		t.Fatalf("expected activity-order comparison branch")
	}
	if compareValidationOrderKeys(validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityOrder: 0, SourceID: "a"}, validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityOrder: 0, SourceID: "b"}) >= 0 {
		t.Fatalf("expected source-id comparison branch")
	}
	if compareValidationOrderKeys(validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityOrder: 0, SourceID: "a", OccurredAt: "2024-01-01T10:00:00Z"}, validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityOrder: 0, SourceID: "a", OccurredAt: "2024-01-01T11:00:00Z"}) >= 0 {
		t.Fatalf("expected occurred-at comparison branch")
	}
	if compareValidationOrderKeys(validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityOrder: 0, SourceID: "a", OccurredAt: "2024-01-01T10:00:00Z", RawHash: "a"}, validationOrderKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityOrder: 0, SourceID: "a", OccurredAt: "2024-01-01T10:00:00Z", RawHash: "b"}) >= 0 {
		t.Fatalf("expected raw-hash comparison branch")
	}
	if validationActivityTypeOrder(syncmodel.ActivityType("OTHER")) != 2 {
		t.Fatalf("expected default activity type order branch")
	}

	validationErr := newValidationError("boom", buy)
	var typed *ValidationError
	if !errors.As(validationErr, &typed) {
		t.Fatalf("expected typed validation error, got %v", validationErr)
	}
	if typed.Error() == "" || len(typed.DiagnosticContext().Records) != 1 {
		t.Fatalf("expected populated validation error details, got %#v", typed)
	}
}

// TestValidateCoversAdditionalRunningQuantityBranches verifies the remaining
// successful and failing validation branches for asset timelines.
// Authored by: OpenCode
func TestValidateCoversAdditionalRunningQuantityBranches(t *testing.T) {
	t.Parallel()

	validator := NewValidator()
	buy := validationTestRecord(t, "buy-1", syncmodel.ActivityTypeBuy)
	sell := validationTestRecord(t, "sell-1", syncmodel.ActivityTypeSell)
	sell.Comment = "sell"

	otherAssetSell := validationTestRecord(t, "sell-eth", syncmodel.ActivityTypeSell)
	otherAssetSell.AssetSymbol = "ETH"
	otherAssetSell.Comment = "sell"
	if err := validator.Validate(syncmodel.ProtectedActivityCache{ActivityCount: 2, Activities: []syncmodel.ActivityRecord{buy, otherAssetSell}}); err == nil {
		t.Fatalf("expected holdings check to fail independently per asset")
	}

	otherAssetBuy := validationTestRecord(t, "buy-eth", syncmodel.ActivityTypeBuy)
	otherAssetBuy.AssetSymbol = "ETH"
	if err := validator.Validate(syncmodel.ProtectedActivityCache{ActivityCount: 3, Activities: []syncmodel.ActivityRecord{buy, otherAssetBuy, otherAssetSell}}); err != nil {
		t.Fatalf("expected independent asset timelines to validate, got %v", err)
	}
}

func validationTestRecord(t *testing.T, sourceID string, activityType syncmodel.ActivityType) syncmodel.ActivityRecord {
	t.Helper()

	quantity, _, err := decimalsupport.ParseString("1")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	unitPrice, _, err := decimalsupport.ParseString("100")
	if err != nil {
		t.Fatalf("parse unit price: %v", err)
	}
	grossValue, _, err := decimalsupport.ParseString("100")
	if err != nil {
		t.Fatalf("parse gross value: %v", err)
	}

	return syncmodel.ActivityRecord{
		SourceID:     sourceID,
		OccurredAt:   "2024-01-01T10:00:00Z",
		ActivityType: activityType,
		AssetSymbol:  "BTC",
		Quantity:     quantity,
		UnitPrice:    unitPrice,
		GrossValue:   grossValue,
		RawHash:      sourceID,
	}
}
