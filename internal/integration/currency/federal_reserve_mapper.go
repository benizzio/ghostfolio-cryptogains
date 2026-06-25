// Package currency owns Federal Reserve H.10 canonicalization for USD report
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

// MapFederalReserveH10CSVToEvidence maps Federal Reserve H.10 CSV fixture or
// provider data into one canonical rate evidence value.
// Authored by: OpenCode
func MapFederalReserveH10CSVToEvidence(request RateLookupRequest, payload []byte, datasetReference string) (ExchangeRateEvidence, error) {
	if err := validateSupportedFederalReserveSourceCurrency(request.SourceCurrency); err != nil {
		return ExchangeRateEvidence{}, err
	}
	var reader = csv.NewReader(strings.NewReader(string(payload)))
	reader.FieldsPerRecord = -1
	var records, err = reader.ReadAll()
	if err != nil {
		return ExchangeRateEvidence{}, fmt.Errorf("parse Federal Reserve H.10 CSV: %w", err)
	}
	if len(records) == 0 {
		return ExchangeRateEvidence{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, datesupport.FormatCalendarDate(request.ActivityDate))
	}

	return mapFederalReserveH10RecordsToEvidence(request, records, datasetReference)
}

// mapFederalReserveH10RecordsToEvidence maps parsed Federal Reserve CSV records
// into canonical evidence after detecting the DDP package shape.
// Authored by: OpenCode
func mapFederalReserveH10RecordsToEvidence(request RateLookupRequest, records [][]string, datasetReference string) (ExchangeRateEvidence, error) {
	var header = records[0]
	if federalReserveH10HeaderIndex(header, "Currency:") >= 0 && federalReserveH10HeaderIndex(header, "Series Name:") >= 0 {
		return mapFederalReserveDDPSeriesRowRecordsToEvidence(request, records, datasetReference)
	}

	return mapFederalReserveLegacyFixtureRecordsToEvidence(request, records, datasetReference)
}

// mapFederalReserveLegacyFixtureRecordsToEvidence maps the earlier compact test
// fixture shape retained for deterministic tests outside the live DDP contract.
// Authored by: OpenCode
func mapFederalReserveLegacyFixtureRecordsToEvidence(request RateLookupRequest, records [][]string, datasetReference string) (ExchangeRateEvidence, error) {
	var sourceRecord []string
	for _, record := range records[1:] {
		if len(record) > 1 && strings.TrimSpace(record[1]) == request.SourceCurrency {
			sourceRecord = record
			break
		}
	}
	if sourceRecord == nil {
		return ExchangeRateEvidence{}, fmt.Errorf("unsupported source currency %s for Federal Reserve H.10", request.SourceCurrency)
	}
	if len(sourceRecord) < 4 || len(records[0]) < 4 {
		return ExchangeRateEvidence{}, fmt.Errorf("parse Federal Reserve H.10 CSV: date observations are required")
	}

	var direction, directionErr = parseFederalReserveQuoteDirection(sourceRecord[2])
	if directionErr != nil {
		return ExchangeRateEvidence{}, directionErr
	}

	var selectedDate, selectedRate, observationErr = selectFederalReserveH10Observation(request, records[0], sourceRecord, 3)
	if observationErr != nil {
		return ExchangeRateEvidence{}, observationErr
	}

	return NewExchangeRateEvidence(request, selectedDate, RateAuthorityFederalReserve, ProviderIDFederalReserveH10, RateKindFederalReserveH10NoonBuying, direction, selectedRate, datasetReference)
}

// mapFederalReserveDDPSeriesRowRecordsToEvidence maps the Federal Reserve Data
// Download Program Output.aspx package CSV layout to canonical evidence.
// Authored by: OpenCode
func mapFederalReserveDDPSeriesRowRecordsToEvidence(request RateLookupRequest, records [][]string, datasetReference string) (ExchangeRateEvidence, error) {
	var header = records[0]
	var currencyColumn = federalReserveH10HeaderIndex(header, "Currency:")
	var seriesNameColumn = federalReserveH10HeaderIndex(header, "Series Name:")
	var firstDateColumn = seriesNameColumn + 1
	if currencyColumn < 0 || seriesNameColumn < 0 || firstDateColumn >= len(header) {
		return ExchangeRateEvidence{}, fmt.Errorf("parse Federal Reserve H.10 CSV: DDP seriesrow date observations are required")
	}

	var sourceRecord []string
	for _, record := range records[1:] {
		if len(record) > currencyColumn && strings.TrimSpace(record[currencyColumn]) == request.SourceCurrency {
			sourceRecord = record
			break
		}
	}
	if sourceRecord == nil {
		return ExchangeRateEvidence{}, fmt.Errorf("unsupported source currency %s for Federal Reserve H.10", request.SourceCurrency)
	}
	if len(sourceRecord) <= seriesNameColumn {
		return ExchangeRateEvidence{}, fmt.Errorf("parse Federal Reserve H.10 CSV: DDP seriesrow metadata is required")
	}

	var direction, directionErr = parseFederalReserveDDPQuoteDirection(sourceRecord[seriesNameColumn])
	if directionErr != nil {
		return ExchangeRateEvidence{}, directionErr
	}

	var selectedDate, selectedRate, observationErr = selectFederalReserveH10Observation(request, header, sourceRecord, firstDateColumn)
	if observationErr != nil {
		return ExchangeRateEvidence{}, observationErr
	}

	return NewExchangeRateEvidence(request, selectedDate, RateAuthorityFederalReserve, ProviderIDFederalReserveH10, RateKindFederalReserveH10NoonBuying, direction, selectedRate, datasetReference)
}

// selectFederalReserveH10Observation chooses the latest valid observation on or
// before the requested activity date from a Federal Reserve H.10 series row.
// Authored by: OpenCode
func selectFederalReserveH10Observation(request RateLookupRequest, header []string, sourceRecord []string, firstDateColumn int) (time.Time, apd.Decimal, error) {
	var selectedDate time.Time
	var selectedRate apd.Decimal
	var found bool
	for column := firstDateColumn; column < len(header) && column < len(sourceRecord); column++ {
		var rateDate, dateErr = time.Parse(time.DateOnly, strings.TrimSpace(header[column]))
		if dateErr != nil || datesupport.CalendarDate(rateDate).After(request.ActivityDate) {
			continue
		}
		var rawRate = strings.TrimSpace(sourceRecord[column])
		if strings.EqualFold(rawRate, "ND") || rawRate == "" {
			continue
		}
		var rate, parseErr = parsePositiveRate(rawRate)
		if parseErr != nil {
			return time.Time{}, apd.Decimal{}, fmt.Errorf("invalid Federal Reserve observation for %s on %s: %w", request.SourceCurrency, strings.TrimSpace(header[column]), parseErr)
		}
		if !found || selectedDate.Before(datesupport.CalendarDate(rateDate)) {
			selectedDate = datesupport.CalendarDate(rateDate)
			selectedRate = rate
			found = true
		}
	}
	if !found {
		return time.Time{}, apd.Decimal{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, datesupport.FormatCalendarDate(request.ActivityDate))
	}

	return selectedDate, selectedRate, nil
}

// federalReserveH10HeaderIndex finds one metadata column in a DDP seriesrow CSV header.
// Authored by: OpenCode
func federalReserveH10HeaderIndex(header []string, expected string) int {
	for index, value := range header {
		if strings.EqualFold(strings.TrimSpace(value), expected) {
			return index
		}
	}

	return -1
}

// parseFederalReserveDDPQuoteDirection maps DDP series identifiers to canonical
// quote direction. RXI$US rows are H.10 starred rows quoted as USD per source.
// Authored by: OpenCode
func parseFederalReserveDDPQuoteDirection(seriesName string) (QuoteDirection, error) {
	var normalized = strings.ToUpper(strings.TrimSpace(seriesName))
	if strings.HasPrefix(normalized, "RXI$US_") {
		return QuoteDirectionBasePerSource, nil
	}
	if strings.HasPrefix(normalized, "RXI_") {
		return QuoteDirectionSourcePerBase, nil
	}

	return "", fmt.Errorf("ambiguous quote direction %q", seriesName)
}

// parseFederalReserveQuoteDirection maps H.10 row text to canonical quote direction.
// Authored by: OpenCode
func parseFederalReserveQuoteDirection(rawDirection string) (QuoteDirection, error) {
	var direction = strings.ToLower(strings.TrimSpace(rawDirection))
	switch direction {
	case "currency units per usd", "currency units per u.s. dollar":
		return QuoteDirectionSourcePerBase, nil
	case "usd per currency unit", "u.s. dollars per currency unit":
		return QuoteDirectionBasePerSource, nil
	default:
		return "", fmt.Errorf("ambiguous quote direction %q", rawDirection)
	}
}

// validateSupportedFederalReserveSourceCurrency rejects unmapped H.10 source currencies.
// Authored by: OpenCode
func validateSupportedFederalReserveSourceCurrency(sourceCurrency string) error {
	if !supportedFederalReserveSourceCurrencies[sourceCurrency] {
		return fmt.Errorf("unsupported source currency %s for Federal Reserve H.10", sourceCurrency)
	}

	return nil
}

var supportedFederalReserveSourceCurrencies = map[string]bool{
	"AUD": true, "BRL": true, "CAD": true, "CHF": true, "CNY": true, "DKK": true, "EUR": true, "GBP": true, "HKD": true, "INR": true,
	"JPY": true, "KRW": true, "LKR": true, "MXN": true, "MYR": true, "NOK": true, "NZD": true, "SEK": true, "SGD": true, "THB": true,
	"TWD": true, "ZAR": true,
}
