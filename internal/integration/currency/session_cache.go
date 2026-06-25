// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"fmt"
	"sync"
	"time"

	datesupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/date"
)

// CurrencyRateSessionCache stores canonical rate evidence only in memory for
// the active TUI process. It must not be serialized into snapshots, setup files,
// report staging files, or any other persistent cache.
//
// Example:
//
//	cache := currency.NewCurrencyRateSessionCache()
//	request, _ := currency.NewRateLookupRequest("USD", currency.BaseCurrencyEUR, time.Now())
//	_, _ = cache.Get(request)
//
// Authored by: OpenCode
type CurrencyRateSessionCache struct {
	mutex   sync.RWMutex
	entries map[rateSessionCacheKey]cachedRateEvidence
}

// rateSessionCacheKey stores the public cache identity without secrets or provider URLs.
// Authored by: OpenCode
type rateSessionCacheKey struct {
	sourceCurrency string
	baseCurrency   string
	activityDate   time.Time
}

// cachedRateEvidence stores one cache entry and the local fetch timestamp.
// Authored by: OpenCode
type cachedRateEvidence struct {
	evidence  ExchangeRateEvidence
	fetchedAt time.Time
}

// NewCurrencyRateSessionCache creates an empty process-local rate cache.
//
// Example:
//
//	cache := currency.NewCurrencyRateSessionCache()
//	_ = cache
//
// Authored by: OpenCode
func NewCurrencyRateSessionCache() *CurrencyRateSessionCache {
	return &CurrencyRateSessionCache{entries: map[rateSessionCacheKey]cachedRateEvidence{}}
}

// Get returns a defensive copy of cached evidence for the lookup key.
// Authored by: OpenCode
func (cache *CurrencyRateSessionCache) Get(request RateLookupRequest) (ExchangeRateEvidence, bool) {
	if cache == nil {
		return ExchangeRateEvidence{}, false
	}

	var key, err = newRateSessionCacheKey(request)
	if err != nil {
		return ExchangeRateEvidence{}, false
	}

	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	var entry, ok = cache.entries[key]
	if !ok {
		return ExchangeRateEvidence{}, false
	}

	return cloneExchangeRateEvidence(entry.evidence), true
}

// Store validates and caches canonical evidence for one lookup key in memory only.
// Authored by: OpenCode
func (cache *CurrencyRateSessionCache) Store(request RateLookupRequest, evidence ExchangeRateEvidence) error {
	return cache.store(request, evidence, time.Now().UTC())
}

// store records one cache entry with an explicit fetch timestamp for tests and providers.
// Authored by: OpenCode
func (cache *CurrencyRateSessionCache) store(request RateLookupRequest, evidence ExchangeRateEvidence, fetchedAt time.Time) error {
	if cache == nil {
		return fmt.Errorf("currency rate session cache is required")
	}

	var key, err = newRateSessionCacheKey(request)
	if err != nil {
		return err
	}
	evidence = cloneExchangeRateEvidence(evidence)
	if err = evidence.Validate(); err != nil {
		return fmt.Errorf("cached exchange rate evidence: %w", err)
	}
	if !evidence.matchesRequest(RateLookupRequest{SourceCurrency: key.sourceCurrency, BaseCurrency: key.baseCurrency, ActivityDate: key.activityDate}) {
		return fmt.Errorf("cached exchange rate evidence does not match lookup key")
	}
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if cache.entries == nil {
		cache.entries = map[rateSessionCacheKey]cachedRateEvidence{}
	}
	cache.entries[key] = cachedRateEvidence{evidence: evidence, fetchedAt: fetchedAt.UTC()}

	return nil
}

// newRateSessionCacheKey validates and normalizes one public lookup key.
// Authored by: OpenCode
func newRateSessionCacheKey(request RateLookupRequest) (rateSessionCacheKey, error) {
	request.ActivityDate = datesupport.CalendarDate(request.ActivityDate)
	if err := request.Validate(); err != nil {
		return rateSessionCacheKey{}, err
	}

	return rateSessionCacheKey{
		sourceCurrency: request.SourceCurrency,
		baseCurrency:   request.BaseCurrency,
		activityDate:   request.ActivityDate,
	}, nil
}
