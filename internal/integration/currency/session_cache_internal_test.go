// Package currency verifies in-memory rate cache behavior.
// Authored by: OpenCode
package currency

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestCurrencyRateSessionCacheReturnsDefensiveCopies verifies stored and returned evidence isolation.
// Authored by: OpenCode
func TestCurrencyRateSessionCacheReturnsDefensiveCopies(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var evidence = mustExchangeRateEvidence(t, request, "1.09")
	var cache = NewCurrencyRateSessionCache()
	if err := cache.store(request, evidence, time.Date(2024, time.January, 2, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("store evidence: %v", err)
	}

	evidence.RateValue.Coeff.SetInt64(999)
	var cached, ok = cache.Get(request)
	if !ok {
		t.Fatalf("expected cached evidence")
	}
	assertCurrencyDecimalString(t, cached.RateValue, "1.09")

	cached.RateValue.Coeff.SetInt64(777)
	var cachedAgain, okAgain = cache.Get(request)
	if !okAgain {
		t.Fatalf("expected cached evidence after returned copy mutation")
	}
	assertCurrencyDecimalString(t, cachedAgain.RateValue, "1.09")
}

// TestCurrencyRateSessionCacheRejectsMismatchedEvidence verifies key/evidence consistency.
// Authored by: OpenCode
func TestCurrencyRateSessionCacheRejectsMismatchedEvidence(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var mismatchedRequest = mustRateLookupRequest(t, "GBP", BaseCurrencyEUR)
	var evidence = mustExchangeRateEvidence(t, mismatchedRequest, "1.09")
	var cache = NewCurrencyRateSessionCache()

	var err = cache.Store(request, evidence)
	if err == nil {
		t.Fatalf("expected mismatched evidence rejection")
	}
}

// TestCurrencyRateSessionCacheDefensiveBranches verifies nil, invalid-key, miss,
// and zero fetched-at behavior without persisting any evidence.
// Authored by: OpenCode
func TestCurrencyRateSessionCacheDefensiveBranches(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var evidence = mustExchangeRateEvidence(t, request, "1.09")
	var nilCache *CurrencyRateSessionCache
	if _, ok := nilCache.Get(request); ok {
		t.Fatalf("expected nil cache miss")
	}
	if err := nilCache.Store(request, evidence); err == nil || !strings.Contains(err.Error(), "cache is required") {
		t.Fatalf("expected nil cache store rejection, got %v", err)
	}

	var cache = &CurrencyRateSessionCache{}
	if _, ok := cache.Get(RateLookupRequest{}); ok {
		t.Fatalf("expected invalid key lookup to miss")
	}
	if _, ok := cache.Get(request); ok {
		t.Fatalf("expected empty cache lookup to miss")
	}
	if err := cache.store(request, evidence, time.Time{}); err != nil {
		t.Fatalf("expected zero fetched-at to be defaulted: %v", err)
	}
	if _, ok := cache.Get(request); !ok {
		t.Fatalf("expected evidence to be cached after zero fetched-at store")
	}
	if err := cache.Store(RateLookupRequest{}, evidence); err == nil || !strings.Contains(err.Error(), "source currency is required") {
		t.Fatalf("expected invalid cache key rejection, got %v", err)
	}
	evidence.RateValue = mustCurrencyDecimal(t, "0")
	if err := cache.Store(request, evidence); err == nil || !strings.Contains(err.Error(), "cached exchange rate evidence") {
		t.Fatalf("expected invalid evidence rejection, got %v", err)
	}
}

// TestCurrencyRateSessionCacheReusesSameKeyEvidenceWithinProcess verifies that
// a service instance does not refetch currently cached evidence for the same key.
// Authored by: OpenCode
func TestCurrencyRateSessionCacheReusesSameKeyEvidenceWithinProcess(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var provider = &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, evidence: mustExchangeRateEvidence(t, request, "1.09")}
	var cache = NewCurrencyRateSessionCache()
	var service, err = newCurrencyRateService(cache, provider)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	var first, firstErr = service.LookupRate(context.Background(), request)
	if firstErr != nil {
		t.Fatalf("first lookup: %v", firstErr)
	}
	provider.evidence = mustExchangeRateEvidence(t, request, "1.11")
	var second, secondErr = service.LookupRate(context.Background(), request)
	if secondErr != nil {
		t.Fatalf("second lookup: %v", secondErr)
	}

	if provider.calls != 1 {
		t.Fatalf("expected same-key cache reuse, got %d provider calls", provider.calls)
	}
	assertCurrencyDecimalString(t, first.RateValue, "1.09")
	assertCurrencyDecimalString(t, second.RateValue, "1.09")
}

// TestCurrencyRateSessionCacheRevisionBehaviorFetchesPublishedValuesForNewService
// verifies that cache reuse is process-local and not persisted across service/cache instances.
// Authored by: OpenCode
func TestCurrencyRateSessionCacheRevisionBehaviorFetchesPublishedValuesForNewService(t *testing.T) {
	t.Parallel()

	var request = mustRateLookupRequest(t, "USD", BaseCurrencyEUR)
	var firstProvider = &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, evidence: mustExchangeRateEvidence(t, request, "1.09")}
	var firstService, firstErr = newCurrencyRateService(NewCurrencyRateSessionCache(), firstProvider)
	if firstErr != nil {
		t.Fatalf("create first service: %v", firstErr)
	}
	var first, lookupErr = firstService.LookupRate(context.Background(), request)
	if lookupErr != nil {
		t.Fatalf("first lookup: %v", lookupErr)
	}

	var revisedProvider = &testRateProvider{baseCurrencyValue: BaseCurrencyEUR, evidence: mustExchangeRateEvidence(t, request, "1.11")}
	var revisedService, revisedErr = newCurrencyRateService(NewCurrencyRateSessionCache(), revisedProvider)
	if revisedErr != nil {
		t.Fatalf("create revised service: %v", revisedErr)
	}
	var revised, revisedLookupErr = revisedService.LookupRate(context.Background(), request)
	if revisedLookupErr != nil {
		t.Fatalf("revised lookup: %v", revisedLookupErr)
	}

	if firstProvider.calls != 1 || revisedProvider.calls != 1 {
		t.Fatalf("expected each service to fetch once, got first=%d revised=%d", firstProvider.calls, revisedProvider.calls)
	}
	assertCurrencyDecimalString(t, first.RateValue, "1.09")
	assertCurrencyDecimalString(t, revised.RateValue, "1.11")
}
