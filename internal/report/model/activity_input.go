// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"time"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

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
	ActivityType                 syncmodel.ActivityType
	AssetIdentityKey             string
	DisplayLabel                 string
	Quantity                     apd.Decimal
	GrossValue                   *apd.Decimal
	FeeAmount                    *apd.Decimal
	UnitPrice                    *apd.Decimal
	SelectedCurrencyContext      SelectedCurrencyContext
	SelectedCurrencyCode         string
	SourceScope                  *syncmodel.SourceScope
	IsZeroPricedHoldingReduction bool
	Comment                      string
}
