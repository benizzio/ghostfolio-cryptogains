// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
	"github.com/cockroachdb/apd/v3"
)

// MapECBEXRCSVToEvidence maps ECB EXR CSV fixture or provider data into one
// canonical rate evidence value.
// Authored by: OpenCode
func MapECBEXRCSVToEvidence(request RateLookupRequest, payload []byte, datasetReference string) (ExchangeRateEvidence, error) {
	if err := validateSupportedECBSourceCurrency(request.SourceCurrency); err != nil {
		return ExchangeRateEvidence{}, err
	}
	var reader = csv.NewReader(strings.NewReader(string(payload)))
	reader.FieldsPerRecord = -1
	var records, err = reader.ReadAll()
	if err != nil {
		return ExchangeRateEvidence{}, fmt.Errorf("parse ECB EXR CSV: %w", err)
	}
	if len(records) == 0 {
		return ExchangeRateEvidence{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, datesupport.FormatCalendarDate(request.ActivityDate))
	}

	var dateColumn, valueColumn, headerErr = ecbEXRColumnIndexes(records[0])
	if headerErr != nil {
		return ExchangeRateEvidence{}, fmt.Errorf("parse ECB EXR CSV: %w", headerErr)
	}

	var selectedDate time.Time
	var selectedRate apd.Decimal
	var found bool
	for _, record := range records[1:] {
		if len(record) <= valueColumn || len(record) <= dateColumn {
			continue
		}
		var rateDate, dateErr = time.Parse(time.DateOnly, strings.TrimSpace(record[dateColumn]))
		if dateErr != nil || datesupport.CalendarDate(rateDate).After(request.ActivityDate) {
			continue
		}
		var rate, parseErr = parsePositiveRate(strings.TrimSpace(record[valueColumn]))
		if parseErr != nil {
			return ExchangeRateEvidence{}, fmt.Errorf("invalid ECB observation for %s on %s: %w", request.SourceCurrency, strings.TrimSpace(record[dateColumn]), parseErr)
		}
		if !found || selectedDate.Before(datesupport.CalendarDate(rateDate)) {
			selectedDate = datesupport.CalendarDate(rateDate)
			selectedRate = rate
			found = true
		}
	}
	if !found {
		return ExchangeRateEvidence{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, datesupport.FormatCalendarDate(request.ActivityDate))
	}

	return NewExchangeRateEvidence(request, selectedDate, RateAuthorityEuropeanCentralBank, ProviderIDECBEXR, RateKindECBEXRDailyReference, QuoteDirectionSourcePerBase, selectedRate, datasetReference)
}

// ecbEXRColumnIndexes locates ECB EXR observation columns in a provider CSV header.
// Authored by: OpenCode
func ecbEXRColumnIndexes(header []string) (int, int, error) {
	var dateColumn = -1
	var valueColumn = -1
	for index, column := range header {
		switch strings.TrimSpace(column) {
		case "TIME_PERIOD":
			dateColumn = index
		case "OBS_VALUE":
			valueColumn = index
		}
	}
	if dateColumn < 0 || valueColumn < 0 {
		return -1, -1, fmt.Errorf("required columns TIME_PERIOD and OBS_VALUE are missing for ECB EXR")
	}

	return dateColumn, valueColumn, nil
}
