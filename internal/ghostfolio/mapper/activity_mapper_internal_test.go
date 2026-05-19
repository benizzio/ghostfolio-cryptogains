package mapper

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

type failingDecimalService struct{}

func (failingDecimalService) ParseString(string) (apd.Decimal, string, error) {
	return apd.Decimal{}, "", errors.New("parse string boom")
}

func (failingDecimalService) ParseNumber(json.Number) (apd.Decimal, string, error) {
	return apd.Decimal{}, "", errors.New("parse number boom")
}

func (failingDecimalService) CanonicalString(apd.Decimal) (string, error) {
	return "", errors.New("canonical boom")
}

func (failingDecimalService) CanonicalStringPointer(*apd.Decimal) (string, error) {
	return "", errors.New("canonical pointer boom")
}

type diagnosticCarrierError struct {
	context syncmodel.DiagnosticContext
}

func (e diagnosticCarrierError) Error() string {
	return "carrier boom"
}

func (e diagnosticCarrierError) DiagnosticContext() syncmodel.DiagnosticContext {
	return e.context
}

func TestMapActivityHandlesOptionalValuesAndScopeBranches(t *testing.T) {
	t.Parallel()

	record, err := MapActivity(validActivityPageEntry(), "USD", nil)
	if err != nil {
		t.Fatalf("map activity: %v", err)
	}
	orderGrossValue, err := decimalsupport.CanonicalStringPointer(record.OrderGrossValue)
	if err != nil {
		t.Fatalf("canonical order gross value: %v", err)
	}
	if record.OrderGrossValue == nil || orderGrossValue != "120" {
		t.Fatalf("expected explicit order gross value, got %#v", record.OrderGrossValue)
	}
	orderFeeAmount, err := decimalsupport.CanonicalStringPointer(record.OrderFeeAmount)
	if err != nil {
		t.Fatalf("canonical order fee amount: %v", err)
	}
	if record.OrderFeeAmount == nil || orderFeeAmount != "0.2" {
		t.Fatalf("expected explicit order fee amount, got %#v", record.OrderFeeAmount)
	}
	if record.SourceScope == nil || record.SourceScope.ID != "account-1" {
		t.Fatalf("expected mapped source scope, got %#v", record.SourceScope)
	}

	entry := validActivityPageEntry()
	entry.UnitPrice = json.Number("")
	entry.ValueInBaseCurrency = json.Number("")
	entry.UnitPriceInAssetProfileCurrency = json.Number("")
	entry.Fee = json.Number("")
	entry.FeeInAssetProfileCurrency = json.Number("")
	entry.FeeInBaseCurrency = json.Number("")
	entry.Account = nil
	record, err = MapActivity(entry, "USD", decimalsupport.NewService())
	if err != nil {
		t.Fatalf("map activity without optional values: %v", err)
	}
	if record.OrderUnitPrice != nil {
		t.Fatalf("expected no persisted selected unit price when order unit price is absent, got %#v", record.OrderUnitPrice)
	}
	if record.OrderGrossValue == nil {
		t.Fatalf("expected explicit order gross value to remain persisted, got %#v", record.OrderGrossValue)
	}
	if record.OrderFeeAmount != nil {
		t.Fatalf("expected nil explicit order fee amount when absent, got %#v", record.OrderFeeAmount)
	}
	if record.SourceScope != nil {
		t.Fatalf("expected nil source scope when account is absent, got %#v", record.SourceScope)
	}

	entry = validActivityPageEntry()
	entry.Currency = dto.NullableString("")
	entry.Comment = dto.NullableString("")
	record, err = MapActivity(entry, "", decimalsupport.NewService())
	if err != nil {
		t.Fatalf("map activity with uninformed nullable strings: %v", err)
	}
	if record.OrderCurrency != "" || record.Comment != "" {
		t.Fatalf("expected uninformed nullable strings to map cleanly, got %#v", record)
	}

	entry = validActivityPageEntry()
	entry.ID = "  activity-2  "
	entry.Date = "  2024-01-01T10:00:00Z  "
	record, err = MapActivity(entry, "USD", decimalsupport.NewService())
	if err != nil {
		t.Fatalf("map activity with padded identifiers: %v", err)
	}
	if record.SourceID != "activity-2" || record.OccurredAt != "2024-01-01T10:00:00Z" {
		t.Fatalf("expected stored identity fields to be trimmed, got %#v", record)
	}
}

func TestMapActivityAndMapActivitiesSurfaceMappingFailures(t *testing.T) {
	t.Parallel()

	if _, err := MapActivity(validActivityPageEntry(), "USD", failingDecimalService{}); err == nil {
		t.Fatalf("expected decimal parse failure")
	}

	_, err := MapActivities([]dto.ActivityPageEntry{validActivityPageEntry()}, "USD", failingDecimalService{})
	var mappingError *MappingError
	if !errors.As(err, &mappingError) {
		t.Fatalf("expected wrapped mapping error, got %v", err)
	}
	if mappingError.Error() == "" {
		t.Fatalf("expected error text on mapping error")
	}
	if mappingError.DiagnosticContext().FailureStage != syncmodel.DiagnosticFailureStageMapping {
		t.Fatalf("expected mapping failure stage, got %#v", mappingError.DiagnosticContext())
	}
}

// TestMappingErrorAndParseHelpersCoverRemainingBranches verifies nil-helper and
// parsing-error branches on the internal mapping helpers.
// Authored by: OpenCode
func TestMappingErrorAndParseHelpersCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	var nilError *MappingError
	if nilError.Error() != "" {
		t.Fatalf("expected nil mapping error string to be empty")
	}
	if context := nilError.DiagnosticContext(); context.FailureStage != "" || context.FailureDetail != "" || len(context.Records) != 0 {
		t.Fatalf("expected nil mapping error context to be empty, got %#v", context)
	}

	entry := validActivityPageEntry()
	wrapped := wrapMappingError(entry, "USD", diagnosticCarrierError{})
	var mappingError *MappingError
	if !errors.As(wrapped, &mappingError) {
		t.Fatalf("expected wrapped mapping error, got %v", wrapped)
	}
	if mappingError.DiagnosticContext().FailureDetail != "carrier boom" {
		t.Fatalf("expected empty carrier detail to default to error text, got %#v", mappingError.DiagnosticContext())
	}

	unitPriceInvalid := validActivityPageEntry()
	unitPriceInvalid.UnitPriceInAssetProfileCurrency = json.Number("bad")
	if _, err := MapActivity(unitPriceInvalid, "USD", decimalsupport.NewService()); err == nil {
		t.Fatalf("expected unit-price parse failure")
	}

	grossInBaseInvalid := validActivityPageEntry()
	grossInBaseInvalid.ValueInBaseCurrency = json.Number("bad")
	if _, err := MapActivity(grossInBaseInvalid, "USD", decimalsupport.NewService()); err == nil {
		t.Fatalf("expected gross-value parse failure from base currency")
	}

	feeInvalid := validActivityPageEntry()
	feeInvalid.FeeInBaseCurrency = json.Number("bad")
	if _, err := MapActivity(feeInvalid, "USD", decimalsupport.NewService()); err == nil {
		t.Fatalf("expected fee parse failure")
	}

	fallbackGrossInvalid := validActivityPageEntry()
	fallbackGrossInvalid.ValueInBaseCurrency = json.Number("")
	fallbackGrossInvalid.Value = json.Number("bad")
	if _, err := parseMoneyContext(fallbackGrossInvalid, "USD", decimalsupport.NewService()); err == nil {
		t.Fatalf("expected fallback gross-value parse failure")
	}

	if _, err := parseOptionalNumber(json.Number("bad"), decimalsupport.NewService()); err == nil {
		t.Fatalf("expected optional-number parse failure")
	}

	orderUnitPriceInvalid := validActivityPageEntry()
	orderUnitPriceInvalid.UnitPrice = json.Number("bad")
	if _, err := parseMoneyContext(orderUnitPriceInvalid, "USD", decimalsupport.NewService()); err == nil || !strings.Contains(err.Error(), "order unit price") {
		t.Fatalf("expected order unit-price parse failure, got %v", err)
	}

	orderFeeInvalid := validActivityPageEntry()
	orderFeeInvalid.Fee = json.Number("bad")
	if _, err := parseMoneyContext(orderFeeInvalid, "USD", decimalsupport.NewService()); err == nil || !strings.Contains(err.Error(), "order fee") {
		t.Fatalf("expected order fee parse failure, got %v", err)
	}

	assetProfileFeeInvalid := validActivityPageEntry()
	assetProfileFeeInvalid.FeeInAssetProfileCurrency = json.Number("bad")
	if _, err := parseMoneyContext(assetProfileFeeInvalid, "USD", decimalsupport.NewService()); err == nil || !strings.Contains(err.Error(), "asset-profile fee") {
		t.Fatalf("expected asset-profile fee parse failure, got %v", err)
	}
}

func TestWrapMappingErrorUsesCarrierAndFallbackContext(t *testing.T) {
	t.Parallel()

	entry := validActivityPageEntry()
	wrapped := wrapMappingError(entry, "USD", diagnosticCarrierError{context: syncmodel.DiagnosticContext{FailureDetail: "existing detail"}})
	var mappingError *MappingError
	if !errors.As(wrapped, &mappingError) {
		t.Fatalf("expected mapping error wrapper, got %v", wrapped)
	}
	if len(mappingError.DiagnosticContext().Records) != 1 {
		t.Fatalf("expected fallback diagnostic record, got %#v", mappingError.DiagnosticContext())
	}
	if mappingError.DiagnosticContext().FailureStage != syncmodel.DiagnosticFailureStageMapping {
		t.Fatalf("expected default mapping stage, got %#v", mappingError.DiagnosticContext())
	}

	customContext := syncmodel.DiagnosticContext{
		FailureStage:  syncmodel.DiagnosticFailureStageNormalization,
		FailureDetail: "preserved detail",
		Records:       []syncmodel.DiagnosticRecord{{SourceID: "custom"}},
	}
	wrapped = wrapMappingError(entry, "USD", diagnosticCarrierError{context: customContext})
	if !errors.As(wrapped, &mappingError) {
		t.Fatalf("expected mapping error wrapper, got %v", wrapped)
	}
	if mappingError.DiagnosticContext().FailureStage != syncmodel.DiagnosticFailureStageNormalization || mappingError.DiagnosticContext().Records[0].SourceID != "custom" {
		t.Fatalf("expected carrier diagnostic context to be preserved, got %#v", mappingError.DiagnosticContext())
	}

	if errorAsDiagnosticCarrier(nil, new(interface {
		DiagnosticContext() syncmodel.DiagnosticContext
	})) {
		t.Fatalf("expected nil error not to carry diagnostic context")
	}
	if errorAsDiagnosticCarrier(errors.New("plain"), new(interface {
		DiagnosticContext() syncmodel.DiagnosticContext
	})) {
		t.Fatalf("expected plain error not to carry diagnostic context")
	}
}

func TestDiagnosticRecordFromActivityEntryCoversGrossValueBranches(t *testing.T) {
	t.Parallel()

	entry := validActivityPageEntry()
	record := diagnosticRecordFromActivityEntry(entry, "USD")
	if record.UnitPrice != "80" || record.UnitPriceCurrency != "CHF" {
		t.Fatalf("expected transient diagnostic unit price from order currency, got %#v", record)
	}
	if record.GrossValue != "120" {
		t.Fatalf("expected preferred gross value, got %q", record.GrossValue)
	}
	if record.SourceScopeKind != string(syncmodel.SourceScopeKindAccount) || record.SourceScopeReliability != string(syncmodel.ScopeReliabilityReliable) {
		t.Fatalf("unexpected scope diagnostic context: %#v", record)
	}

	entry.ValueInBaseCurrency = json.Number("")
	record = diagnosticRecordFromActivityEntry(entry, "USD")
	if record.GrossValue != "120" {
		t.Fatalf("expected fallback activity value gross value, got %q", record.GrossValue)
	}

	entry = validActivityPageEntry()
	entry.UnitPrice = json.Number("")
	record = diagnosticRecordFromActivityEntry(entry, "USD")
	if record.UnitPrice != "82.3" || record.UnitPriceCurrency != "EUR" {
		t.Fatalf("expected transient diagnostic unit price from asset-profile currency, got %#v", record)
	}

	entry = validActivityPageEntry()
	entry.Account.ID = ""
	record = diagnosticRecordFromActivityEntry(entry, "USD")
	if record.SourceScopeID != "" || record.SourceScopeKind != "" || record.SourceScopeReliability != "" {
		t.Fatalf("expected diagnostic scope to follow shared scope mapping, got %#v", record)
	}
}

// TestDiagnosticHelperDerivationsCoverRemainingBranches verifies diagnostic-only
// currency and decimal derivation helpers across the remaining branches.
// Authored by: OpenCode
func TestDiagnosticHelperDerivationsCoverRemainingBranches(t *testing.T) {
	t.Parallel()

	entry := validActivityPageEntry()
	entry.UnitPrice = json.Number("")
	entry.UnitPriceInAssetProfileCurrency = json.Number("")
	entry.Value = json.Number("")
	record := diagnosticRecordFromActivityEntry(entry, "USD")
	if record.UnitPrice != "82.3" || record.UnitPriceCurrency != "USD" {
		t.Fatalf("expected diagnostic unit price derived from base gross value, got %#v", record)
	}
	if record.GrossValue != "123.45" || record.GrossValueCurrency != "USD" {
		t.Fatalf("expected base-currency gross value preference, got %#v", record)
	}

	entry = validActivityPageEntry()
	entry.Value = json.Number("")
	entry.ValueInBaseCurrency = json.Number("")
	record = diagnosticRecordFromActivityEntry(entry, "USD")
	if record.GrossValue != "123.45" || record.GrossValueCurrency != "EUR" {
		t.Fatalf("expected gross value derived from asset-profile unit price, got %#v", record)
	}

	entry = validActivityPageEntry()
	entry.Value = json.Number("")
	entry.ValueInBaseCurrency = json.Number("")
	entry.UnitPriceInAssetProfileCurrency = json.Number("")
	record = diagnosticRecordFromActivityEntry(entry, "USD")
	if record.GrossValue != "120" || record.GrossValueCurrency != "CHF" {
		t.Fatalf("expected gross value derived from order unit price, got %#v", record)
	}

	entry = validActivityPageEntry()
	entry.Fee = json.Number("")
	entry.FeeInBaseCurrency = json.Number("")
	if amount := diagnosticPreferredFeeAmount(entry); amount != "0.22" {
		t.Fatalf("expected asset-profile fee preference, got %q", amount)
	}
	if currency := diagnosticPreferredFeeAmountCurrency(entry, "USD"); currency != "EUR" {
		t.Fatalf("expected asset-profile fee currency, got %q", currency)
	}

	entry = validActivityPageEntry()
	entry.Fee = json.Number("")
	entry.FeeInAssetProfileCurrency = json.Number("")
	if amount := diagnosticPreferredFeeAmount(entry); amount != "0.25" {
		t.Fatalf("expected base fee preference, got %q", amount)
	}
	if currency := diagnosticPreferredFeeAmountCurrency(entry, "USD"); currency != "USD" {
		t.Fatalf("expected base fee currency, got %q", currency)
	}

	if derived := diagnosticDerivedUnitPrice("", "10"); derived != "" {
		t.Fatalf("expected empty derived unit price for missing quantity, got %q", derived)
	}
	if derived := diagnosticDerivedUnitPrice("bad", "10"); derived != "" {
		t.Fatalf("expected empty derived unit price for unreadable quantity, got %q", derived)
	}
	if derived := diagnosticDerivedUnitPrice("1", "bad"); derived != "" {
		t.Fatalf("expected empty derived unit price for unreadable gross value, got %q", derived)
	}
	if derived := diagnosticDerivedUnitPrice("3", "10"); derived != "" {
		t.Fatalf("expected empty derived unit price for inexact division, got %q", derived)
	}
	if derived := diagnosticDerivedUnitPrice("1.5", "123.45"); derived != "82.3" {
		t.Fatalf("expected exact derived unit price, got %q", derived)
	}

	if derived := diagnosticDerivedGrossValue("", "10"); derived != "" {
		t.Fatalf("expected empty derived gross value for missing quantity, got %q", derived)
	}
	if derived := diagnosticDerivedGrossValue("bad", "10"); derived != "" {
		t.Fatalf("expected empty derived gross value for unreadable quantity, got %q", derived)
	}
	if derived := diagnosticDerivedGrossValue("1", "bad"); derived != "" {
		t.Fatalf("expected empty derived gross value for unreadable unit price, got %q", derived)
	}
	if derived := diagnosticDerivedGrossValue("1.5", "82.3"); derived != "123.45" {
		t.Fatalf("expected exact derived gross value, got %q", derived)
	}
}

func validActivityPageEntry() dto.ActivityPageEntry {
	return dto.ActivityPageEntry{
		ID:                              "activity-1",
		Date:                            "2024-01-01T10:00:00Z",
		Type:                            "buy",
		Quantity:                        json.Number("1.5"),
		Currency:                        "CHF",
		Fee:                             json.Number("0.2"),
		UnitPrice:                       json.Number("80"),
		Value:                           json.Number("120"),
		FeeInAssetProfileCurrency:       json.Number("0.22"),
		ValueInBaseCurrency:             json.Number("123.45"),
		FeeInBaseCurrency:               json.Number("0.25"),
		UnitPriceInAssetProfileCurrency: json.Number("82.3"),
		Comment:                         "comment",
		SymbolProfile:                   dto.ActivitySymbolProfile{Symbol: "BTC", Name: "Bitcoin", Currency: "EUR"},
		Account:                         &dto.ActivityAccountScope{ID: "account-1", Name: "Main Account"},
		DataSource:                      "ghostfolio",
	}
}
