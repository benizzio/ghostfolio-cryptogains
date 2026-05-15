// Package model defines normalized sync data structures shared across sync,
// snapshot, and runtime packages.
// Authored by: OpenCode
package model

import "github.com/cockroachdb/apd/v3"

// SourceScopeKind identifies one preserved Ghostfolio source-scope kind.
// Authored by: OpenCode
type SourceScopeKind string

const (
	// SourceScopeKindAccount identifies an account-scoped activity source.
	SourceScopeKindAccount SourceScopeKind = "account"

	// SourceScopeKindWallet identifies a wallet-scoped activity source.
	SourceScopeKindWallet SourceScopeKind = "wallet"

	// SourceScopeKindUnknown identifies source scope data whose kind is not known.
	SourceScopeKindUnknown SourceScopeKind = "unknown"
)

// ActivityType identifies one normalized Ghostfolio activity type supported by
// this slice.
// Authored by: OpenCode
type ActivityType string

const (
	// ActivityTypeBuy identifies a normalized BUY activity.
	ActivityTypeBuy ActivityType = "BUY"

	// ActivityTypeSell identifies a normalized SELL activity.
	ActivityTypeSell ActivityType = "SELL"
)

// SourceScope stores optional source grouping information preserved from
// Ghostfolio activities.
// Authored by: OpenCode
type SourceScope struct {
	ID          string
	Name        string
	Kind        SourceScopeKind
	Reliability ScopeReliability
}

// ActivityRecord stores one normalized activity ready for later validation and
// protected persistence.
// Authored by: OpenCode
type ActivityRecord struct {
	SourceID     string
	OccurredAt   string
	ActivityType ActivityType
	AssetSymbol  string
	AssetName    string
	BaseCurrency string
	Quantity     apd.Decimal
	UnitPrice    apd.Decimal
	GrossValue   apd.Decimal
	FeeAmount    *apd.Decimal
	Comment      string
	DataSource   string
	SourceScope  *SourceScope
	RawHash      string
}
