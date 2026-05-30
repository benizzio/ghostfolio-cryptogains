// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"time"

	"github.com/cockroachdb/apd/v3"
)

// ActivityType identifies one normalized report activity direction used during
// capital-gains calculation and rendering.
//
// Example:
//
//	input.ActivityType = model.ActivityTypeBuy
//
// Authored by: OpenCode
type ActivityType string

const (
	// ActivityTypeBuy identifies an acquisition activity.
	ActivityTypeBuy ActivityType = "BUY"

	// ActivityTypeSell identifies a disposal or holding-reduction activity.
	ActivityTypeSell ActivityType = "SELL"
)

// SourceScopeKind identifies the source-owned grouping kind preserved for
// scope-local report calculations.
//
// Example:
//
//	input.SourceScope = &model.SourceScope{Kind: model.SourceScopeKindWallet}
//
// Authored by: OpenCode
type SourceScopeKind string

const (
	// SourceScopeKindAccount identifies an account-scoped report activity.
	SourceScopeKindAccount SourceScopeKind = "account"

	// SourceScopeKindWallet identifies a wallet-scoped report activity.
	SourceScopeKindWallet SourceScopeKind = "wallet"
)

// ScopeReliability identifies whether preserved source scope data is safe to
// use for scope-local report calculations.
//
// Example:
//
//	input.SourceScope = &model.SourceScope{Reliability: model.ScopeReliabilityReliable}
//
// Authored by: OpenCode
type ScopeReliability string

const (
	// ScopeReliabilityReliable indicates a stable non-empty source scope.
	ScopeReliabilityReliable ScopeReliability = "reliable"

	// ScopeReliabilityPartial indicates incomplete or contradictory source scope data.
	ScopeReliabilityPartial ScopeReliability = "partial"

	// ScopeReliabilityUnavailable indicates absent usable source scope data.
	ScopeReliabilityUnavailable ScopeReliability = "unavailable"
)

// SourceScope stores report-owned source grouping information for scope-local
// cost-basis calculation.
// Authored by: OpenCode
type SourceScope struct {
	ID          string
	Name        string
	Kind        SourceScopeKind
	Reliability ScopeReliability
}

// SelectedCurrencyContext identifies which complete activity monetary tier was
// selected for one priced activity input.
// Authored by: OpenCode
type SelectedCurrencyContext string

const (
	// SelectedCurrencyContextOrder identifies the order-tier monetary context.
	SelectedCurrencyContextOrder SelectedCurrencyContext = "order"

	// SelectedCurrencyContextAssetProfile identifies the asset-profile-tier
	// monetary context.
	SelectedCurrencyContextAssetProfile SelectedCurrencyContext = "asset_profile"

	// SelectedCurrencyContextBase identifies the base-tier monetary context.
	SelectedCurrencyContextBase SelectedCurrencyContext = "base"
)

// ActivityCalculationInput stores one normalized activity after one complete
// monetary context has been selected for report calculation.
// Authored by: OpenCode
type ActivityCalculationInput struct {
	SourceID                     string
	OccurredAt                   time.Time
	SourceYear                   int
	ActivityType                 ActivityType
	AssetIdentityKey             string
	DisplayLabel                 string
	Quantity                     apd.Decimal
	GrossValue                   *apd.Decimal
	FeeAmount                    *apd.Decimal
	UnitPrice                    *apd.Decimal
	SelectedCurrencyContext      SelectedCurrencyContext
	SelectedCurrencyCode         string
	SourceScope                  *SourceScope
	IsZeroPricedHoldingReduction bool
	Comment                      string
}
