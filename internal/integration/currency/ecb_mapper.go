// Package currency owns official exchange-rate provider integration for report
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

// MapECBEXRCSVToEvidence maps ECB EXR CSV fixture or provider data into one
// canonical rate evidence value.
// Authored by: OpenCode
func MapECBEXRCSVToEvidence(request RateLookupRequest, payload []byte, datasetReference string) (ExchangeRateEvidence, error) {
	if err := validateSupportedECBSourceCurrency(request.SourceCurrency); err != nil {
		return ExchangeRateEvidence{}, err
	}
	var records, err = csv.NewReader(strings.NewReader(string(payload))).ReadAll()
	if err != nil {
		return ExchangeRateEvidence{}, fmt.Errorf("parse ECB EXR CSV: %w", err)
	}
	if len(records) == 0 {
		return ExchangeRateEvidence{}, fmt.Errorf("no current or prior available observation for %s/%s on %s", request.SourceCurrency, request.BaseCurrency, formatDate(request.ActivityDate))
	}

	var dateColumn, valueColumn, headerErr = findCSVColumns(records[0], "TIME_PERIOD", "OBS_VALUE")
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
		if dateErr != nil || canonicalDate(rateDate).After(request.ActivityDate) {
			continue
		}
		var rate, parseErr = parsePositiveRate(strings.TrimSpace(record[valueColumn]))
		if parseErr != nil {
			return ExchangeRateEvidence{}, fmt.Errorf("invalid ECB observation for %s on %s: %w", request.SourceCurrency, strings.TrimSpace(record[dateColumn]), parseErr)
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

	return NewExchangeRateEvidence(request, selectedDate, RateAuthorityEuropeanCentralBank, ProviderIDECBEXR, RateKindECBEXRDailyReference, QuoteDirectionSourcePerBase, selectedRate, datasetReference)
}
