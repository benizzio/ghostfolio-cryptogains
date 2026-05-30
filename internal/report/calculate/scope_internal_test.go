// Package calculate defines yearly gains-and-losses report calculation
// services built on normalized protected activity history.
// Authored by: OpenCode
package calculate

import (
	"errors"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
)

// TestResolveScopedAssetInputsNarrowsReliableAssetTimelineEvenWhenCacheIsPartial
// verifies that scope-local narrowing is decided from the asset timeline used by
// the report rather than a coarse cache-wide reliability summary.
// Authored by: OpenCode
func TestResolveScopedAssetInputsNarrowsReliableAssetTimelineEvenWhenCacheIsPartial(t *testing.T) {
	t.Parallel()

	var group = assetInputGroup{
		AssetIdentityKey: "asset-avax-001",
		DisplayLabel:     "AVAX",
		Inputs: []reportmodel.ActivityCalculationInput{
			{
				SourceID:         "avax-buy-alpha-2023-001",
				OccurredAt:       time.Date(2023, time.June, 10, 0, 0, 0, 0, time.UTC),
				SourceYear:       2023,
				ActivityType:     reportmodel.ActivityTypeBuy,
				AssetIdentityKey: "asset-avax-001",
				DisplayLabel:     "AVAX",
				Quantity:         *apd.New(1, 0),
				SourceScope: &reportmodel.SourceScope{
					ID:          "wallet-avax-alpha",
					Name:        "AVAX Alpha Wallet",
					Kind:        reportmodel.SourceScopeKindWallet,
					Reliability: reportmodel.ScopeReliabilityReliable,
				},
			},
			{
				SourceID:         "avax-sell-alpha-2024-001",
				OccurredAt:       time.Date(2024, time.August, 15, 0, 0, 0, 0, time.UTC),
				SourceYear:       2024,
				ActivityType:     reportmodel.ActivityTypeSell,
				AssetIdentityKey: "asset-avax-001",
				DisplayLabel:     "AVAX",
				Quantity:         *apd.New(1, 0),
				SourceScope: &reportmodel.SourceScope{
					ID:          "wallet-avax-alpha",
					Name:        "AVAX Alpha Wallet",
					Kind:        reportmodel.SourceScopeKindWallet,
					Reliability: reportmodel.ScopeReliabilityReliable,
				},
			},
		},
	}

	var scopedInputs, err = resolveScopedAssetInputs(reportmodel.CostBasisMethodScopeLocalHybrid, group)
	if err != nil {
		t.Fatalf("resolve scoped inputs: %v", err)
	}
	if len(scopedInputs) != 2 {
		t.Fatalf("unexpected scoped input count: got %d want 2", len(scopedInputs))
	}
	for _, scopedInput := range scopedInputs {
		if scopedInput.ApplicableScope.BroadenedToAsset {
			t.Fatalf("expected reliable asset timeline to stay narrowed, got %#v", scopedInput.ApplicableScope)
		}
		if scopedInput.ApplicableScope.ScopeKind != applicableScopeKindWallet {
			t.Fatalf("expected wallet scope kind, got %#v", scopedInput.ApplicableScope)
		}
		if scopedInput.ApplicableScope.ScopeKey != "wallet-avax-alpha" {
			t.Fatalf("expected narrowed wallet scope key, got %#v", scopedInput.ApplicableScope)
		}
	}
}

// TestResolveScopedAssetInputsBroadensContradictoryTimeline verifies that one
// asset timeline still broadens when its own scope data is contradictory.
// Authored by: OpenCode
func TestResolveScopedAssetInputsBroadensContradictoryTimeline(t *testing.T) {
	t.Parallel()

	var group = assetInputGroup{
		AssetIdentityKey: "asset-btc-001",
		DisplayLabel:     "BTC",
		Inputs: []reportmodel.ActivityCalculationInput{
			{
				SourceID:         "btc-buy-2023-001",
				OccurredAt:       time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				SourceYear:       2023,
				ActivityType:     reportmodel.ActivityTypeBuy,
				AssetIdentityKey: "asset-btc-001",
				DisplayLabel:     "BTC",
				Quantity:         *apd.New(1, 0),
				SourceScope: &reportmodel.SourceScope{
					ID:          "scope-1",
					Kind:        reportmodel.SourceScopeKindWallet,
					Reliability: reportmodel.ScopeReliabilityReliable,
				},
			},
			{
				SourceID:         "btc-buy-2023-002",
				OccurredAt:       time.Date(2023, time.February, 1, 0, 0, 0, 0, time.UTC),
				SourceYear:       2023,
				ActivityType:     reportmodel.ActivityTypeBuy,
				AssetIdentityKey: "asset-btc-001",
				DisplayLabel:     "BTC",
				Quantity:         *apd.New(1, 0),
				SourceScope: &reportmodel.SourceScope{
					ID:          "scope-1",
					Kind:        reportmodel.SourceScopeKindAccount,
					Reliability: reportmodel.ScopeReliabilityReliable,
				},
			},
		},
	}

	var scopedInputs, err = resolveScopedAssetInputs(reportmodel.CostBasisMethodScopeLocalHybrid, group)
	if err != nil {
		t.Fatalf("resolve scoped inputs: %v", err)
	}
	if len(scopedInputs) != 2 {
		t.Fatalf("unexpected scoped input count: got %d want 2", len(scopedInputs))
	}
	for _, scopedInput := range scopedInputs {
		if !scopedInput.ApplicableScope.BroadenedToAsset {
			t.Fatalf("expected contradictory timeline to broaden, got %#v", scopedInput.ApplicableScope)
		}
		if scopedInput.ApplicableScope.ScopeKind != applicableScopeKindAsset {
			t.Fatalf("expected asset-level broadened scope, got %#v", scopedInput.ApplicableScope)
		}
	}
}

// TestScopeHelperBranches verifies direct scope helper guardrails and broadening
// decisions that are easier to exercise without full asset replay.
// Authored by: OpenCode
func TestScopeHelperBranches(t *testing.T) {
	t.Parallel()

	if !shouldBroadenAssetScope(assetInputGroup{Inputs: []reportmodel.ActivityCalculationInput{{SourceScope: &reportmodel.SourceScope{ID: "scope-a", Kind: reportmodel.SourceScopeKindWallet, Reliability: reportmodel.ScopeReliabilityPartial}}}}) {
		t.Fatalf("expected unreliable scope to broaden asset scope")
	}
	if !shouldBroadenAssetScope(assetInputGroup{Inputs: []reportmodel.ActivityCalculationInput{{SourceScope: &reportmodel.SourceScope{ID: " ", Kind: reportmodel.SourceScopeKindWallet, Reliability: reportmodel.ScopeReliabilityReliable}}}}) {
		t.Fatalf("expected blank scope ID to broaden asset scope")
	}
	if !shouldBroadenAssetScope(assetInputGroup{Inputs: []reportmodel.ActivityCalculationInput{{SourceScope: &reportmodel.SourceScope{ID: "scope-a", Kind: reportmodel.SourceScopeKind("portfolio"), Reliability: reportmodel.ScopeReliabilityReliable}}}}) {
		t.Fatalf("expected unsupported scope kind to broaden asset scope")
	}

	if _, err := resolveReliableApplicableScope("asset-btc", reportmodel.ActivityCalculationInput{}); err == nil || !strings.Contains(err.Error(), "source scope is required") {
		t.Fatalf("expected missing source scope to fail, got %v", err)
	}
	if _, err := resolveReliableApplicableScope("asset-btc", reportmodel.ActivityCalculationInput{SourceScope: &reportmodel.SourceScope{ID: "scope-a", Kind: reportmodel.SourceScopeKindWallet, Reliability: reportmodel.ScopeReliabilityPartial}}); err == nil || !strings.Contains(err.Error(), "does not support narrowing") {
		t.Fatalf("expected unreliable source scope to fail, got %v", err)
	}
	if _, err := resolveReliableApplicableScope("asset-btc", reportmodel.ActivityCalculationInput{SourceScope: &reportmodel.SourceScope{ID: "scope-a", Kind: reportmodel.SourceScopeKind("portfolio"), Reliability: reportmodel.ScopeReliabilityReliable}}); err == nil || !strings.Contains(err.Error(), "does not support narrowing") {
		t.Fatalf("expected unsupported source scope kind to fail, got %v", err)
	}
	if _, err := resolveReliableApplicableScope("asset-btc", reportmodel.ActivityCalculationInput{SourceScope: &reportmodel.SourceScope{ID: " ", Kind: reportmodel.SourceScopeKindWallet, Reliability: reportmodel.ScopeReliabilityReliable}}); err == nil || !strings.Contains(err.Error(), "scope ID is required") {
		t.Fatalf("expected blank reliable scope ID to fail, got %v", err)
	}

	if kind, ok := supportedApplicableScopeKind(reportmodel.SourceScopeKind("portfolio")); ok || kind != "" {
		t.Fatalf("expected unsupported scope kind lookup to fail, got kind=%q ok=%t", kind, ok)
	}
}

// TestResolveScopedAssetInputsWrapsReliableScopeFailures verifies the direct
// wrapper branch around reliable-scope resolution.
// Authored by: OpenCode
func TestResolveScopedAssetInputsWrapsReliableScopeFailures(t *testing.T) {
	var previous = resolveReliableApplicableScopeFunc
	defer func() {
		resolveReliableApplicableScopeFunc = previous
	}()

	resolveReliableApplicableScopeFunc = func(string, reportmodel.ActivityCalculationInput) (applicableScope, error) {
		return applicableScope{}, errors.New("scope boom")
	}

	_, err := resolveScopedAssetInputs(reportmodel.CostBasisMethodScopeLocalHybrid, assetInputGroup{
		AssetIdentityKey: "asset-btc-001",
		DisplayLabel:     "BTC",
		Inputs: []reportmodel.ActivityCalculationInput{{
			SourceID:         "btc-buy-1",
			OccurredAt:       time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			SourceYear:       2024,
			ActivityType:     reportmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-btc-001",
			DisplayLabel:     "BTC",
			Quantity:         *apd.New(1, 0),
			SourceScope: &reportmodel.SourceScope{
				ID:          "scope-a",
				Kind:        reportmodel.SourceScopeKindWallet,
				Reliability: reportmodel.ScopeReliabilityReliable,
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "could not resolve the applicable scope for the scope-local method") {
		t.Fatalf("expected wrapped reliable-scope failure, got %v", err)
	}
}
