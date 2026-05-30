// Package calculate defines yearly gains-and-losses report calculation
// services built on normalized protected activity history.
// Authored by: OpenCode
package calculate

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// applicableScopeKind identifies the calculator-local scope partition used by
// the scope-local hybrid method.
// Authored by: OpenCode
type applicableScopeKind string

const (
	// applicableScopeKindAccount identifies an account-local scope.
	// Authored by: OpenCode
	applicableScopeKindAccount applicableScopeKind = "account"

	// applicableScopeKindWallet identifies a wallet-local scope.
	// Authored by: OpenCode
	applicableScopeKindWallet applicableScopeKind = "wallet"

	// applicableScopeKindAsset identifies asset-level broadened scope.
	// Authored by: OpenCode
	applicableScopeKindAsset applicableScopeKind = "asset"
)

// applicableScope stores one resolved scope-local partition.
// Authored by: OpenCode
type applicableScope struct {
	AssetIdentityKey string
	ScopeKey         string
	ScopeKind        applicableScopeKind
	BroadenedToAsset bool
}

// scopedActivityInput stores one selected activity input with its resolved
// applicable scope.
// Authored by: OpenCode
type scopedActivityInput struct {
	Input           reportmodel.ActivityCalculationInput
	ApplicableScope applicableScope
}

// Test seams keep the scope-local wrapper branch directly coverable.
// Authored by: OpenCode
var resolveReliableApplicableScopeFunc = resolveReliableApplicableScope

// resolveScopedAssetInputs resolves the applicable scope for each asset input.
// Authored by: OpenCode
func resolveScopedAssetInputs(
	method reportmodel.CostBasisMethod,
	group assetInputGroup,
) ([]scopedActivityInput, error) {
	var scopedInputs = make([]scopedActivityInput, 0, len(group.Inputs))
	if method != reportmodel.CostBasisMethodScopeLocalHybrid {
		for _, input := range group.Inputs {
			scopedInputs = append(scopedInputs, scopedActivityInput{Input: input})
		}
		return scopedInputs, nil
	}

	if shouldBroadenAssetScope(group) {
		var scope = broadenedAssetScope(group.AssetIdentityKey)
		for _, input := range group.Inputs {
			scopedInputs = append(scopedInputs, scopedActivityInput{Input: input, ApplicableScope: scope})
		}
		return scopedInputs, nil
	}

	for _, input := range group.Inputs {
		var scope, err = resolveReliableApplicableScopeFunc(group.AssetIdentityKey, input)
		if err != nil {
			return nil, newInputCalculationError(
				reportmodel.CalculationErrorKindActivityInput,
				input,
				"could not resolve the applicable scope for the scope-local method",
				err,
			)
		}
		scopedInputs = append(scopedInputs, scopedActivityInput{Input: input, ApplicableScope: scope})
	}

	return scopedInputs, nil
}

// shouldBroadenAssetScope decides whether one asset timeline can safely narrow
// to wallet or account scopes or must broaden back to asset-level scope.
// Authored by: OpenCode
func shouldBroadenAssetScope(group assetInputGroup) bool {
	var scopeKinds = make(map[string]applicableScopeKind)
	for _, input := range group.Inputs {
		if input.SourceScope == nil {
			return true
		}
		if input.SourceScope.Reliability != reportmodel.ScopeReliabilityReliable {
			return true
		}

		var scopeID = strings.TrimSpace(input.SourceScope.ID)
		if scopeID == "" {
			return true
		}

		var scopeKind, ok = supportedApplicableScopeKind(input.SourceScope.Kind)
		if !ok {
			return true
		}

		var previousKind, exists = scopeKinds[scopeID]
		if exists && previousKind != scopeKind {
			return true
		}
		scopeKinds[scopeID] = scopeKind
	}

	return false
}

// broadenedAssetScope returns the asset-level scope partition used when source
// scope data cannot be trusted for one asset timeline.
// Authored by: OpenCode
func broadenedAssetScope(assetIdentityKey string) applicableScope {
	return applicableScope{
		AssetIdentityKey: strings.TrimSpace(assetIdentityKey),
		ScopeKey:         strings.TrimSpace(assetIdentityKey),
		ScopeKind:        applicableScopeKindAsset,
		BroadenedToAsset: true,
	}
}

// resolveReliableApplicableScope narrows one activity to its reliable wallet or
// account scope.
// Authored by: OpenCode
func resolveReliableApplicableScope(assetIdentityKey string, input reportmodel.ActivityCalculationInput) (applicableScope, error) {
	if input.SourceScope == nil {
		return applicableScope{}, fmt.Errorf("source scope is required")
	}
	if input.SourceScope.Reliability != reportmodel.ScopeReliabilityReliable {
		return applicableScope{}, fmt.Errorf("source scope reliability %q does not support narrowing", input.SourceScope.Reliability)
	}

	var scopeKind, ok = supportedApplicableScopeKind(input.SourceScope.Kind)
	if !ok {
		return applicableScope{}, fmt.Errorf("source scope kind %q does not support narrowing", input.SourceScope.Kind)
	}

	var scopeID = strings.TrimSpace(input.SourceScope.ID)
	if scopeID == "" {
		return applicableScope{}, fmt.Errorf("source scope ID is required")
	}

	return applicableScope{
		AssetIdentityKey: strings.TrimSpace(assetIdentityKey),
		ScopeKey:         scopeID,
		ScopeKind:        scopeKind,
		BroadenedToAsset: false,
	}, nil
}

// supportedApplicableScopeKind maps the preserved sync scope kind into the
// calculator-local applicable-scope kind.
// Authored by: OpenCode
func supportedApplicableScopeKind(kind reportmodel.SourceScopeKind) (applicableScopeKind, bool) {
	switch kind {
	case reportmodel.SourceScopeKindAccount:
		return applicableScopeKindAccount, true
	case reportmodel.SourceScopeKindWallet:
		return applicableScopeKindWallet, true
	default:
		return "", false
	}
}
