package mapper

import (
	"encoding/json"
	"errors"
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

	record, err := MapActivity(validActivityPageEntry(), nil)
	if err != nil {
		t.Fatalf("map activity: %v", err)
	}
	if record.GrossValue.String() != "123.45" {
		t.Fatalf("expected base-currency gross value preference, got %s", record.GrossValue.String())
	}
	if record.FeeAmount == nil || record.FeeAmount.String() != "0.25" {
		t.Fatalf("expected fee amount to be parsed, got %#v", record.FeeAmount)
	}
	if record.SourceScope == nil || record.SourceScope.ID != "account-1" {
		t.Fatalf("expected mapped source scope, got %#v", record.SourceScope)
	}

	entry := validActivityPageEntry()
	entry.ValueInBaseCurrency = json.Number("")
	entry.FeeInBaseCurrency = json.Number("")
	entry.Account = nil
	record, err = MapActivity(entry, decimalsupport.NewService())
	if err != nil {
		t.Fatalf("map activity without optional values: %v", err)
	}
	grossValue, err := decimalsupport.CanonicalString(record.GrossValue)
	if err != nil {
		t.Fatalf("canonical fallback gross value: %v", err)
	}
	if grossValue != "120" {
		t.Fatalf("expected fallback gross value from activity value, got %s", grossValue)
	}
	if record.FeeAmount != nil {
		t.Fatalf("expected nil fee amount when absent, got %#v", record.FeeAmount)
	}
	if record.SourceScope != nil {
		t.Fatalf("expected nil source scope when account is absent, got %#v", record.SourceScope)
	}
}

func TestMapActivityAndMapActivitiesSurfaceMappingFailures(t *testing.T) {
	t.Parallel()

	if _, err := MapActivity(validActivityPageEntry(), failingDecimalService{}); err == nil {
		t.Fatalf("expected decimal parse failure")
	}

	_, err := MapActivities([]dto.ActivityPageEntry{validActivityPageEntry()}, failingDecimalService{})
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
	wrapped := wrapMappingError(entry, diagnosticCarrierError{})
	var mappingError *MappingError
	if !errors.As(wrapped, &mappingError) {
		t.Fatalf("expected wrapped mapping error, got %v", wrapped)
	}
	if mappingError.DiagnosticContext().FailureDetail != "carrier boom" {
		t.Fatalf("expected empty carrier detail to default to error text, got %#v", mappingError.DiagnosticContext())
	}

	unitPriceInvalid := validActivityPageEntry()
	unitPriceInvalid.UnitPriceInAssetProfileCurrency = json.Number("bad")
	if _, err := MapActivity(unitPriceInvalid, decimalsupport.NewService()); err == nil {
		t.Fatalf("expected unit-price parse failure")
	}

	grossInBaseInvalid := validActivityPageEntry()
	grossInBaseInvalid.ValueInBaseCurrency = json.Number("bad")
	if _, err := MapActivity(grossInBaseInvalid, decimalsupport.NewService()); err == nil {
		t.Fatalf("expected gross-value parse failure from base currency")
	}

	feeInvalid := validActivityPageEntry()
	feeInvalid.FeeInBaseCurrency = json.Number("bad")
	if _, err := MapActivity(feeInvalid, decimalsupport.NewService()); err == nil {
		t.Fatalf("expected fee parse failure")
	}

	fallbackGrossInvalid := validActivityPageEntry()
	fallbackGrossInvalid.ValueInBaseCurrency = json.Number("")
	fallbackGrossInvalid.Value = json.Number("bad")
	if _, err := parseGrossValue(fallbackGrossInvalid, decimalsupport.NewService()); err == nil {
		t.Fatalf("expected fallback gross-value parse failure")
	}

	if _, err := parseOptionalNumber(json.Number("bad"), decimalsupport.NewService()); err == nil {
		t.Fatalf("expected optional-number parse failure")
	}
}

func TestWrapMappingErrorUsesCarrierAndFallbackContext(t *testing.T) {
	t.Parallel()

	entry := validActivityPageEntry()
	wrapped := wrapMappingError(entry, diagnosticCarrierError{context: syncmodel.DiagnosticContext{FailureDetail: "existing detail"}})
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
	wrapped = wrapMappingError(entry, diagnosticCarrierError{context: customContext})
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
	record := diagnosticRecordFromActivityEntry(entry)
	if record.GrossValue != "123.45" {
		t.Fatalf("expected preferred gross value, got %q", record.GrossValue)
	}
	if record.SourceScopeKind != string(syncmodel.SourceScopeKindAccount) || record.SourceScopeReliability != string(syncmodel.ScopeReliabilityReliable) {
		t.Fatalf("unexpected scope diagnostic context: %#v", record)
	}

	entry.ValueInBaseCurrency = json.Number("")
	record = diagnosticRecordFromActivityEntry(entry)
	if record.GrossValue != "120" {
		t.Fatalf("expected fallback activity value gross value, got %q", record.GrossValue)
	}

	entry = validActivityPageEntry()
	entry.Account.ID = ""
	record = diagnosticRecordFromActivityEntry(entry)
	if record.SourceScopeID != "" || record.SourceScopeKind != "" || record.SourceScopeReliability != "" {
		t.Fatalf("expected diagnostic scope to follow shared scope mapping, got %#v", record)
	}
}

func validActivityPageEntry() dto.ActivityPageEntry {
	return dto.ActivityPageEntry{
		ID:                              "activity-1",
		Date:                            "2024-01-01T10:00:00Z",
		Type:                            "buy",
		Quantity:                        json.Number("1.5"),
		Value:                           json.Number("120"),
		ValueInBaseCurrency:             json.Number("123.45"),
		FeeInBaseCurrency:               json.Number("0.25"),
		UnitPriceInAssetProfileCurrency: json.Number("82.3"),
		Comment:                         "comment",
		SymbolProfile:                   dto.ActivitySymbolProfile{Symbol: "BTC", Name: "Bitcoin"},
		Account:                         &dto.ActivityAccountScope{ID: "account-1", Name: "Main Account"},
		DataSource:                      "ghostfolio",
		BaseCurrency:                    "USD",
	}
}
