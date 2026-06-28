// Package calculate defines the report calculation currency-conversion boundary.
// Authored by: OpenCode
package calculate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	currencyintegration "github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// reportCurrencyBoundaryResult stores converted calculation inputs and the
// report-visible artifacts created while applying the currency boundary.
// Authored by: OpenCode
type reportCurrencyBoundaryResult struct {
	Groups                 []assetInputGroup
	ConversionAuditEntries []reportmodel.ConversionAuditEntry
	RateSources            []reportmodel.ExchangeRateEvidence
}

// reportCurrencyBoundaryContext keeps per-report rate evidence reuse bounded by
// unique source/base/activity-date keys.
// Authored by: OpenCode
type reportCurrencyBoundaryContext struct {
	ctx                     context.Context
	currencyRates           CurrencyRateService
	reportBaseCurrency      reportmodel.ReportBaseCurrency
	recordBySourceID        map[string]syncmodel.ActivityRecord
	resolvedEvidenceByKey   map[string]currencyintegration.ExchangeRateEvidence
	reportEvidenceByKey     map[string]reportmodel.ExchangeRateEvidence
	orderedRateEvidenceKeys []string
}

// applyReportCurrencyBoundary prepares selected activity monetary values for the
// requested report base currency before basis replay consumes them.
// Authored by: OpenCode
func applyReportCurrencyBoundary(
	ctx context.Context,
	currencyRates CurrencyRateService,
	reportBaseCurrency reportmodel.ReportBaseCurrency,
	groups []assetInputGroup,
) (reportCurrencyBoundaryResult, error) {
	return applyReportCurrencyBoundaryWithRecords(ctx, currencyRates, reportBaseCurrency, groups, nil)
}

// applyReportCurrencyBoundaryWithRecords prepares selected activity monetary
// values and keeps source records available for failure diagnostics.
// Authored by: OpenCode
func applyReportCurrencyBoundaryWithRecords(
	ctx context.Context,
	currencyRates CurrencyRateService,
	reportBaseCurrency reportmodel.ReportBaseCurrency,
	groups []assetInputGroup,
	records []syncmodel.ActivityRecord,
) (reportCurrencyBoundaryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var boundary = reportCurrencyBoundaryContext{
		ctx:                   ctx,
		currencyRates:         currencyRates,
		reportBaseCurrency:    reportBaseCurrency,
		recordBySourceID:      recordBySourceID(records),
		resolvedEvidenceByKey: make(map[string]currencyintegration.ExchangeRateEvidence),
		reportEvidenceByKey:   make(map[string]reportmodel.ExchangeRateEvidence),
	}

	var result reportCurrencyBoundaryResult
	var convertedGroups = make([]assetInputGroup, 0, len(groups))
	for _, group := range groups {
		var convertedGroup = group
		convertedGroup.Inputs = make([]reportmodel.ActivityCalculationInput, 0, len(group.Inputs))

		for _, input := range group.Inputs {
			var convertedInput, auditEntry, err = boundary.applyInputReportCurrencyBoundary(group, input)
			if err != nil {
				return reportCurrencyBoundaryResult{}, boundary.withInputDiagnosticRecord(input, err)
			}
			convertedGroup.Inputs = append(convertedGroup.Inputs, convertedInput)
			if auditEntry != nil {
				result.ConversionAuditEntries = append(result.ConversionAuditEntries, *auditEntry)
			}
		}

		convertedGroups = append(convertedGroups, convertedGroup)
	}

	for _, key := range boundary.orderedRateEvidenceKeys {
		result.RateSources = append(result.RateSources, boundary.reportEvidenceByKey[key])
	}
	result.Groups = convertedGroups
	return result, nil
}

// withInputDiagnosticRecord attaches the persisted source activity to one
// conversion-boundary calculation error when the caller supplied source records.
// Authored by: OpenCode
func (boundary *reportCurrencyBoundaryContext) withInputDiagnosticRecord(input reportmodel.ActivityCalculationInput, err error) error {
	if err == nil || boundary == nil || len(boundary.recordBySourceID) == 0 {
		return err
	}

	var calculationError *reportmodel.CalculationError
	if !errors.As(err, &calculationError) || calculationError == nil {
		return err
	}

	var record, ok = boundary.recordBySourceID[strings.TrimSpace(input.SourceID)]
	if !ok {
		return err
	}
	record.Comment = ""
	record.DataSource = redact.Text(record.DataSource)
	record.RawHash = redact.Text(record.RawHash)

	return withPersistedActivityRecord(calculationError, &record)
}

// recordBySourceID indexes persisted activity records by non-secret source ID.
// Authored by: OpenCode
func recordBySourceID(records []syncmodel.ActivityRecord) map[string]syncmodel.ActivityRecord {
	if len(records) == 0 {
		return nil
	}

	var indexed = make(map[string]syncmodel.ActivityRecord, len(records))
	for _, record := range records {
		var sourceID = strings.TrimSpace(record.SourceID)
		if sourceID == "" {
			continue
		}
		indexed[sourceID] = record
	}

	return indexed
}

// applyInputReportCurrencyBoundary bypasses same-currency and zero-priced rows,
// and converts cross-currency priced rows when rate evidence can be resolved.
// Authored by: OpenCode
func (boundary *reportCurrencyBoundaryContext) applyInputReportCurrencyBoundary(
	group assetInputGroup,
	input reportmodel.ActivityCalculationInput,
) (reportmodel.ActivityCalculationInput, *reportmodel.ConversionAuditEntry, error) {
	var baseCurrency = boundary.reportBaseCurrency.Label()
	if input.IsZeroPricedHoldingReduction {
		input.SelectedCurrencyCode = baseCurrency
		return input, nil, nil
	}

	var sourceCurrency = strings.TrimSpace(input.SelectedCurrencyCode)
	if sourceCurrency == baseCurrency {
		input.ConversionStatus = reportmodel.ConversionStatusSameCurrency
		input.ActivityCurrencyCode = sourceCurrency
		return input, nil, nil
	}

	var evidence, reportEvidence, err = boundary.resolveCrossCurrencyBoundary(input, sourceCurrency, baseCurrency)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, nil, err
	}

	var convertedInput reportmodel.ActivityCalculationInput
	var amounts []reportmodel.ConvertedActivityAmount
	convertedInput, amounts, err = convertInputMonetaryAmounts(input, boundary.reportBaseCurrency, evidence, reportEvidence)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, nil, err
	}

	var auditEntry reportmodel.ConversionAuditEntry
	auditEntry, err = buildConversionAuditEntry(group, convertedInput, reportEvidence, amounts)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, nil, err
	}

	return convertedInput, &auditEntry, nil
}

// resolveCrossCurrencyBoundary validates the report-calculation seam for one
// priced row that cannot be calculated as same-currency.
// Authored by: OpenCode
func (boundary *reportCurrencyBoundaryContext) resolveCrossCurrencyBoundary(
	input reportmodel.ActivityCalculationInput,
	sourceCurrency string,
	baseCurrency string,
) (currencyintegration.ExchangeRateEvidence, reportmodel.ExchangeRateEvidence, error) {
	var lookupRequest, err = currencyintegration.NewRateLookupRequest(sourceCurrency, baseCurrency, datesupport.CalendarDate(input.OccurredAt))
	if err != nil {
		return currencyintegration.ExchangeRateEvidence{}, reportmodel.ExchangeRateEvidence{}, newInputCalculationError(
			reportmodel.CalculationErrorKindActivityInput,
			input,
			fmt.Sprintf(
				"could not prepare currency conversion from %s to %s on %s",
				sourceCurrency,
				baseCurrency,
				datesupport.FormatCalendarDate(input.OccurredAt),
			),
			err,
		)
	}

	var key = rateLookupBoundaryKey(lookupRequest)
	if evidence, ok := boundary.resolvedEvidenceByKey[key]; ok {
		return evidence, boundary.reportEvidenceByKey[key], nil
	}

	if boundary.currencyRates == nil {
		return currencyintegration.ExchangeRateEvidence{}, reportmodel.ExchangeRateEvidence{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			fmt.Sprintf(
				"currency conversion from %s to %s on %s requires a configured currency rate service",
				sourceCurrency,
				baseCurrency,
				datesupport.FormatCalendarDate(input.OccurredAt),
			),
			nil,
		)
	}

	var evidence currencyintegration.ExchangeRateEvidence
	evidence, err = boundary.currencyRates.LookupRate(boundary.ctx, lookupRequest)
	if err != nil {
		return currencyintegration.ExchangeRateEvidence{}, reportmodel.ExchangeRateEvidence{}, newConversionLookupCalculationError(
			input,
			fmt.Sprintf(
				"could not resolve currency conversion rate from %s to %s on %s",
				sourceCurrency,
				baseCurrency,
				datesupport.FormatCalendarDate(input.OccurredAt),
			),
			err,
		)
	}

	var reportEvidence reportmodel.ExchangeRateEvidence
	reportEvidence, err = mapIntegrationEvidenceToReportEvidence(evidence)
	if err != nil {
		return currencyintegration.ExchangeRateEvidence{}, reportmodel.ExchangeRateEvidence{}, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			fmt.Sprintf(
				"could not validate currency conversion evidence from %s to %s on %s",
				sourceCurrency,
				baseCurrency,
				datesupport.FormatCalendarDate(input.OccurredAt),
			),
			err,
		)
	}

	boundary.resolvedEvidenceByKey[key] = evidence
	boundary.reportEvidenceByKey[key] = reportEvidence
	boundary.orderedRateEvidenceKeys = append(boundary.orderedRateEvidenceKeys, key)
	return evidence, reportEvidence, nil
}

// convertInputMonetaryAmounts converts every selected monetary field that can
// affect report basis, proceeds, fees, gains, losses, or totals.
// Authored by: OpenCode
func convertInputMonetaryAmounts(
	input reportmodel.ActivityCalculationInput,
	reportBaseCurrency reportmodel.ReportBaseCurrency,
	evidence currencyintegration.ExchangeRateEvidence,
	reportEvidence reportmodel.ExchangeRateEvidence,
) (reportmodel.ActivityCalculationInput, []reportmodel.ConvertedActivityAmount, error) {
	var convertedInput = input
	var amounts []reportmodel.ConvertedActivityAmount
	var err error

	convertedInput.UnitPrice, amounts, err = convertOptionalInputAmount(input, reportBaseCurrency, evidence, reportEvidence, reportmodel.ConvertedAmountKindUnitPrice, input.UnitPrice, amounts)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, nil, err
	}
	convertedInput.GrossValue, amounts, err = convertOptionalInputAmount(input, reportBaseCurrency, evidence, reportEvidence, reportmodel.ConvertedAmountKindGrossValue, input.GrossValue, amounts)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, nil, err
	}
	convertedInput.FeeAmount, amounts, err = convertOptionalInputAmount(input, reportBaseCurrency, evidence, reportEvidence, reportmodel.ConvertedAmountKindFeeAmount, input.FeeAmount, amounts)
	if err != nil {
		return reportmodel.ActivityCalculationInput{}, nil, err
	}

	convertedInput.SelectedCurrencyCode = reportBaseCurrency.Label()
	convertedInput.ActivityCurrencyCode = evidence.SourceCurrency
	convertedInput.ConversionStatus = reportmodel.ConversionStatusConverted
	return convertedInput, amounts, nil
}

// convertOptionalInputAmount converts one optional monetary field and appends an
// audit amount when the field is present.
// Authored by: OpenCode
func convertOptionalInputAmount(
	input reportmodel.ActivityCalculationInput,
	reportBaseCurrency reportmodel.ReportBaseCurrency,
	evidence currencyintegration.ExchangeRateEvidence,
	reportEvidence reportmodel.ExchangeRateEvidence,
	kind reportmodel.ConvertedAmountKind,
	amount *apd.Decimal,
	amounts []reportmodel.ConvertedActivityAmount,
) (*apd.Decimal, []reportmodel.ConvertedActivityAmount, error) {
	if amount == nil {
		return nil, amounts, nil
	}

	var original = decimalsupport.Clone(*amount)
	var converted, err = convertAmountToBase(original, evidence.RateValue, evidence.QuoteDirection)
	if err != nil {
		return nil, nil, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			fmt.Sprintf("could not convert %s from %s to %s", kind, evidence.SourceCurrency, evidence.BaseCurrency),
			err,
		)
	}

	var amountEvidence = reportEvidence
	var convertedAmount = reportmodel.ConvertedActivityAmount{
		SourceID:             input.SourceID,
		AmountKind:           kind,
		OriginalCurrency:     evidence.SourceCurrency,
		OriginalAmount:       original,
		ReportBaseCurrency:   reportBaseCurrency,
		ConvertedAmount:      decimalsupport.Clone(converted),
		ExchangeRateEvidence: &amountEvidence,
		ConversionStatus:     reportmodel.ConversionStatusConverted,
	}
	if err = convertedAmount.Validate(); err != nil {
		return nil, nil, newInputCalculationError(
			reportmodel.CalculationErrorKindBasisAllocation,
			input,
			fmt.Sprintf("could not validate converted %s from %s to %s", kind, evidence.SourceCurrency, evidence.BaseCurrency),
			err,
		)
	}

	amounts = append(amounts, convertedAmount)
	return &converted, amounts, nil
}

// mapIntegrationEvidenceToReportEvidence converts canonical integration-layer
// evidence into the report-owned audit model without exposing provider DTOs.
// Authored by: OpenCode
func mapIntegrationEvidenceToReportEvidence(evidence currencyintegration.ExchangeRateEvidence) (reportmodel.ExchangeRateEvidence, error) {
	var baseCurrency reportmodel.ReportBaseCurrency
	switch strings.TrimSpace(evidence.BaseCurrency) {
	case reportmodel.ReportBaseCurrencyUSD.Label():
		baseCurrency = reportmodel.ReportBaseCurrencyUSD
	case reportmodel.ReportBaseCurrencyEUR.Label():
		baseCurrency = reportmodel.ReportBaseCurrencyEUR
	default:
		return reportmodel.ExchangeRateEvidence{}, fmt.Errorf("unsupported report base currency %q", evidence.BaseCurrency)
	}

	var reportEvidence = reportmodel.ExchangeRateEvidence{
		SourceCurrency:   evidence.SourceCurrency,
		BaseCurrency:     baseCurrency,
		ActivityDate:     evidence.ActivityDate,
		RateDate:         evidence.RateDate,
		Authority:        reportmodel.RateAuthority(evidence.Authority),
		ProviderID:       reportmodel.RateProviderID(evidence.ProviderID),
		RateKind:         evidence.RateKind,
		QuoteDirection:   reportmodel.QuoteDirection(evidence.QuoteDirection),
		RateValue:        decimalsupport.Clone(evidence.RateValue),
		DatasetReference: evidence.DatasetReference,
	}
	if err := reportEvidence.Validate(); err != nil {
		return reportmodel.ExchangeRateEvidence{}, err
	}

	return reportEvidence, nil
}

// rateLookupBoundaryKey returns the per-report unique evidence lookup key.
// Authored by: OpenCode
func rateLookupBoundaryKey(request currencyintegration.RateLookupRequest) string {
	return request.SourceCurrency + "|" + request.BaseCurrency + "|" + request.ActivityDate.Format(time.DateOnly)
}
