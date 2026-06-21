// Package currency verifies in-memory rate cache behavior.
// Authored by: OpenCode
package currency

import (
	"context"
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
