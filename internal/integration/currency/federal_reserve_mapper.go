// Package currency owns Federal Reserve H.10 canonicalization for USD report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/apd/v3"
)

// MapFederalReserveH10CSVToEvidence maps Federal Reserve H.10 CSV fixture or
// provider data into one canonical rate evidence value.
// Authored by: OpenCode
func MapFederalReserveH10CSVToEvidence(request RateLookupRequest, payload []byte, datasetReference string) (ExchangeRateEvidence, error) {
	if err := validateSupportedFederalReserveSourceCurrency(request.SourceCurrency); err != nil {
		return ExchangeRateEvidence{}, err
	}
	var records, err = csv.NewReader(strings.NewReader(string(payload))).ReadAll()
	if err != nil {
		return ExchangeRateEvidence{}, fmt.Errorf("parse Federal Reserve H.10 CSV: %w", err)
	}
	if len(records) == 0 {
		return ExchangeRateEvidence{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, formatDate(request.ActivityDate))
	}

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

	var selectedDate time.Time
	var selectedRate apd.Decimal
	var found bool
	for column := 3; column < len(records[0]) && column < len(sourceRecord); column++ {
		var rateDate, dateErr = time.Parse(time.DateOnly, strings.TrimSpace(records[0][column]))
		if dateErr != nil || canonicalDate(rateDate).After(request.ActivityDate) {
			continue
		}
		var rawRate = strings.TrimSpace(sourceRecord[column])
		if strings.EqualFold(rawRate, "ND") || rawRate == "" {
			continue
		}
		var rate, parseErr = parsePositiveRate(rawRate)
		if parseErr != nil {
			return ExchangeRateEvidence{}, fmt.Errorf("invalid Federal Reserve observation for %s on %s: %w", request.SourceCurrency, strings.TrimSpace(records[0][column]), parseErr)
		}
		if !found || selectedDate.Before(canonicalDate(rateDate)) {
			selectedDate = canonicalDate(rateDate)
			selectedRate = rate
			found = true
		}
	}
	if !found {
		return ExchangeRateEvidence{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, formatDate(request.ActivityDate))
	}

	return NewExchangeRateEvidence(request, selectedDate, RateAuthorityFederalReserve, ProviderIDFederalReserveH10, RateKindFederalReserveH10NoonBuying, direction, selectedRate, datasetReference)
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
