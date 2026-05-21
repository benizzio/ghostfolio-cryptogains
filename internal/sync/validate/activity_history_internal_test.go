package validate

import (
	"errors"
	"strings"
	"testing"

	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
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
		{name: "missing asset identity key", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.AssetIdentityKey = ""; return record }()}}},
		{name: "non-positive quantity", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.Quantity = zeroPrice; return record }()}}},
		{name: "negative gross value", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.OrderGrossValue = &negativeValue; return record }()}}},
		{name: "negative fee", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.OrderFeeAmount = &negativeValue; return record }()}}},
		{name: "buy zero unit price", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := buy; record.OrderUnitPrice = &zeroPrice; return record }()}}},
		{name: "sell negative unit price", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord { record := sell; record.OrderUnitPrice = &negativeValue; return record }()}}},
		{name: "sell zero price without comment", cache: syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{func() syncmodel.ActivityRecord {
			record := sell
			record.OrderUnitPrice = &zeroPrice
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

	if syncmodel.CompareActivityOrdering(syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01"}, syncmodel.ActivityOrderingKey{SourceDate: "2024-01-02"}) >= 0 {
		t.Fatalf("expected source date comparison branch")
	}
	if syncmodel.CompareActivityOrdering(syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC"}, syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "ETH"}) >= 0 {
		t.Fatalf("expected asset-symbol comparison branch")
	}
	if syncmodel.CompareActivityOrdering(syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy}, syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeSell}) >= 0 {
		t.Fatalf("expected activity-order comparison branch")
	}
	if syncmodel.CompareActivityOrdering(syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy, SourceID: "a"}, syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy, SourceID: "b"}) >= 0 {
		t.Fatalf("expected source-id comparison branch")
	}
	if syncmodel.CompareActivityOrdering(syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy, SourceID: "a", OccurredAt: "2024-01-01T10:00:00Z"}, syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy, SourceID: "a", OccurredAt: "2024-01-01T11:00:00Z"}) >= 0 {
		t.Fatalf("expected occurred-at comparison branch")
	}
	if syncmodel.CompareActivityOrdering(syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy, SourceID: "a", OccurredAt: "2024-01-01T10:00:00Z", RawHash: "a"}, syncmodel.ActivityOrderingKey{SourceDate: "2024-01-01", AssetSymbol: "BTC", ActivityType: syncmodel.ActivityTypeBuy, SourceID: "a", OccurredAt: "2024-01-01T10:00:00Z", RawHash: "b"}) >= 0 {
		t.Fatalf("expected raw-hash comparison branch")
	}
	if syncmodel.CompareActivityOrdering(syncmodel.ActivityOrderingKey{ActivityType: syncmodel.ActivityType("OTHER")}, syncmodel.ActivityOrderingKey{ActivityType: syncmodel.ActivityTypeSell}) <= 0 {
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

// TestApplyRunningQuantityRejectsInvalidArithmetic verifies invalid decimal arithmetic fails fast.
// Authored by: OpenCode
func TestApplyRunningQuantityRejectsInvalidArithmetic(t *testing.T) {
	t.Parallel()

	var zero apd.Decimal
	var invalidQuantity apd.Decimal
	invalidQuantity.Form = apd.NaNSignaling

	record := validationTestRecord(t, "buy-invalid", syncmodel.ActivityTypeBuy)
	record.Quantity = invalidQuantity

	err := applyRunningQuantity(record, map[string]apd.Decimal{}, &zero)
	if err == nil {
		t.Fatalf("expected invalid arithmetic to fail")
	}
	if !strings.Contains(err.Error(), "invalid quantity arithmetic") {
		t.Fatalf("expected invalid arithmetic error, got %v", err)
	}
	if !strings.Contains(err.Error(), apd.InvalidOperation.String()) {
		t.Fatalf("expected invalid operation condition in error, got %v", err)
	}
}

// TestValidateCurrencyContextFailuresExposeDiagnosticContext verifies that
// incomplete explicit currency identity stays visible in validation diagnostics.
// Authored by: OpenCode
func TestValidateCurrencyContextFailuresExposeDiagnosticContext(t *testing.T) {
	t.Parallel()

	validator := NewValidator()
	record := validationTestRecord(t, "buy-1", syncmodel.ActivityTypeBuy)
	record.OrderCurrency = ""
	record.AssetProfileCurrency = ""
	record.BaseCurrency = ""
	record.BaseGrossValue = nil

	err := validator.Validate(syncmodel.ProtectedActivityCache{
		ActivityCount: 1,
		Activities:    []syncmodel.ActivityRecord{record},
	})
	if err == nil {
		t.Fatalf("expected contradictory currency context rejection")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected typed validation error, got %v", err)
	}
	context := validationErr.DiagnosticContext()
	if context.FailureStage != syncmodel.DiagnosticFailureStageValidation {
		t.Fatalf("expected validation failure stage, got %#v", context)
	}
	if !strings.Contains(context.FailureDetail, "uninformed across order, asset-profile, and base tiers") {
		t.Fatalf("expected incomplete currency detail, got %#v", context)
	}
	if len(context.Records) != 1 {
		t.Fatalf("expected one diagnostic record, got %#v", context)
	}
	if context.Records[0].OrderCurrency != "" || context.Records[0].OrderUnitPrice != "100" {
		t.Fatalf("expected preserved currency diagnostic context, got %#v", context.Records[0])
	}
}

func TestValidateCurrencyContextAllowsSingleUninformedTierWhenOthersRemainInformed(t *testing.T) {
	t.Parallel()

	validator := NewValidator()
	record := validationTestRecord(t, "buy-1", syncmodel.ActivityTypeBuy)
	record.OrderCurrency = ""
	record.AssetProfileCurrency = "EUR"
	record.BaseCurrency = "USD"
	assetProfileUnitPrice, _, err := decimalsupport.ParseString("95")
	if err != nil {
		t.Fatalf("parse asset-profile unit price: %v", err)
	}
	baseGrossValue, _, err := decimalsupport.ParseString("100")
	if err != nil {
		t.Fatalf("parse base gross value: %v", err)
	}
	record.AssetProfileUnitPrice = &assetProfileUnitPrice
	record.BaseGrossValue = &baseGrossValue

	err = validator.Validate(syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{record}})
	if err != nil {
		t.Fatalf("expected one uninformed tier to stay valid, got %v", err)
	}
}

// TestValidateAllowsProdLikeOrderTierPrecisionDifferences verifies that
// preserved Ghostfolio source values remain valid even when `value` does not
// equal `quantity * unitPrice` exactly.
// Authored by: OpenCode
func TestValidateAllowsProdLikeOrderTierPrecisionDifferences(t *testing.T) {
	t.Parallel()

	validator := NewValidator()
	record := validationTestRecord(t, "prod-like-buy", syncmodel.ActivityTypeBuy)
	quantity, _, err := decimalsupport.ParseString("238.70829827")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	orderUnitPrice, _, err := decimalsupport.ParseString("1.254775813")
	if err != nil {
		t.Fatalf("parse order unit price: %v", err)
	}
	orderGrossValue, _, err := decimalsupport.ParseString("299.5253990315857")
	if err != nil {
		t.Fatalf("parse order gross value: %v", err)
	}
	baseGrossValue, _, err := decimalsupport.ParseString("260.52719207767325")
	if err != nil {
		t.Fatalf("parse base gross value: %v", err)
	}
	record.Quantity = quantity
	record.OrderCurrency = "USD"
	record.AssetProfileCurrency = "USD"
	record.BaseCurrency = "EUR"
	record.OrderUnitPrice = &orderUnitPrice
	record.OrderGrossValue = &orderGrossValue
	record.BaseGrossValue = &baseGrossValue

	err = validator.Validate(syncmodel.ProtectedActivityCache{ActivityCount: 1, Activities: []syncmodel.ActivityRecord{record}})
	if err != nil {
		t.Fatalf("expected prod-like precision mismatch to remain valid, got %v", err)
	}
}

// TestDirectValidationHelpersCoverRemainingBranches verifies the direct helper
// branches not covered through end-to-end cache validation.
// Authored by: OpenCode
func TestDirectValidationHelpersCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	var zero apd.Decimal
	record := validationTestRecord(t, "missing-unit-price", syncmodel.ActivityTypeBuy)
	if err := validateActivityType(record, syncmodel.ResolvedActivityAmounts{}, &zero); err == nil {
		t.Fatalf("expected missing resolved unit price to fail")
	}

	var invalidQuantity apd.Decimal
	invalidQuantity.Form = apd.NaNSignaling
	record = validationTestRecord(t, "sell-invalid", syncmodel.ActivityTypeSell)
	record.Quantity = invalidQuantity
	record.Comment = "sell"

	err := applyRunningQuantity(record, map[string]apd.Decimal{}, &zero)
	if err == nil {
		t.Fatalf("expected invalid sell arithmetic to fail")
	}
	if !strings.Contains(err.Error(), "invalid quantity arithmetic") {
		t.Fatalf("expected invalid arithmetic error, got %v", err)
	}
	if !strings.Contains(err.Error(), apd.InvalidOperation.String()) {
		t.Fatalf("expected invalid operation condition in error, got %v", err)
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
		SourceID:         sourceID,
		OccurredAt:       "2024-01-01T10:00:00Z",
		ActivityType:     activityType,
		AssetIdentityKey: "asset-btc-validation-001",
		AssetSymbol:      "BTC",
		OrderCurrency:    "USD",
		BaseCurrency:     "USD",
		Quantity:         quantity,
		OrderUnitPrice:   &unitPrice,
		OrderGrossValue:  &grossValue,
		RawHash:          sourceID,
	}
}
