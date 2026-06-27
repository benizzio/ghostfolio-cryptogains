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
	var records, err = readECBEXRCSVRecords(payload)
	if err != nil {
		return ExchangeRateEvidence{}, err
	}
	if len(records) == 0 {
		return ExchangeRateEvidence{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, datesupport.FormatCalendarDate(request.ActivityDate))
	}

	var dateColumn, valueColumn, headerErr = ecbEXRColumnIndexes(records[0])
	if headerErr != nil {
		return ExchangeRateEvidence{}, fmt.Errorf("parse ECB EXR CSV: %w", headerErr)
	}

	var selected, found, selectErr = selectECBEXRObservation(request, records[1:], dateColumn, valueColumn)
	if selectErr != nil {
		return ExchangeRateEvidence{}, selectErr
	}
	if !found {
		return ExchangeRateEvidence{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, datesupport.FormatCalendarDate(request.ActivityDate))
	}

	return NewExchangeRateEvidence(request, selected.date, RateAuthorityEuropeanCentralBank, ProviderIDECBEXR, RateKindECBEXRDailyReference, QuoteDirectionSourcePerBase, selected.rate, datasetReference)
}

// ecbEXRObservation stores one candidate ECB EXR provider observation.
// Authored by: OpenCode
type ecbEXRObservation struct {
	date time.Time
	rate apd.Decimal
}

// readECBEXRCSVRecords parses the CSV payload and rejects empty provider data.
// Authored by: OpenCode
func readECBEXRCSVRecords(payload []byte) ([][]string, error) {
	var reader = csv.NewReader(strings.NewReader(string(payload)))
	reader.FieldsPerRecord = -1
	var records, err = reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse ECB EXR CSV: %w", err)
	}
	return records, nil
}

// selectECBEXRObservation returns the latest usable observation on or before the
// request activity date.
// Authored by: OpenCode
func selectECBEXRObservation(request RateLookupRequest, records [][]string, dateColumn int, valueColumn int) (ecbEXRObservation, bool, error) {
	var selected ecbEXRObservation
	var found bool
	for _, record := range records {
		var observation, ok, err = parseECBEXRObservation(request, record, dateColumn, valueColumn)
		if err != nil {
			return ecbEXRObservation{}, false, err
		}
		if !ok || found && !selected.date.Before(observation.date) {
			continue
		}
		selected = observation
		found = true
	}

	return selected, found, nil
}

// parseECBEXRObservation maps one CSV record into a valid candidate observation.
// Authored by: OpenCode
func parseECBEXRObservation(request RateLookupRequest, record []string, dateColumn int, valueColumn int) (ecbEXRObservation, bool, error) {
	if len(record) <= valueColumn || len(record) <= dateColumn {
		return ecbEXRObservation{}, false, nil
	}
	var rawDate = strings.TrimSpace(record[dateColumn])
	var rateDate, dateErr = time.Parse(time.DateOnly, rawDate)
	if dateErr != nil || datesupport.CalendarDate(rateDate).After(request.ActivityDate) {
		return ecbEXRObservation{}, false, nil
	}
	var rate, parseErr = parsePositiveRate(strings.TrimSpace(record[valueColumn]))
	if parseErr != nil {
		return ecbEXRObservation{}, false, fmt.Errorf("invalid ECB observation for %s on %s: %w", request.SourceCurrency, rawDate, parseErr)
	}

	return ecbEXRObservation{date: datesupport.CalendarDate(rateDate), rate: rate}, true, nil
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
