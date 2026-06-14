package fixture

import (
	"fmt"
	"sort"
	"strings"
	"time"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	"github.com/cockroachdb/apd/v3"
)

// BuildProjectActivityCache translates the synthetic empirical dataset into the
// protected activity-cache shape consumed by pure report calculation.
//
// Example:
//
//	cache, err := fixture.BuildProjectActivityCache(dataset)
//	if err != nil {
//		panic(err)
//	}
//	_ = cache.ActivityCount
//
// Authored by: OpenCode
func BuildProjectActivityCache(dataset EmpiricalDataset) (syncmodel.ProtectedActivityCache, error) {
	var translated = make([]translatedEmpiricalActivity, 0, len(dataset.Activities))
	var index int

	for index = range dataset.Activities {
		var activityTranslation translatedEmpiricalActivity
		var err error

		activityTranslation, err = translateEmpiricalActivity(dataset.Activities[index])
		if err != nil {
			return syncmodel.ProtectedActivityCache{}, err
		}

		translated = append(translated, activityTranslation)
	}

	sort.SliceStable(translated, func(left int, right int) bool {
		return compareTranslatedEmpiricalActivities(translated[left], translated[right]) < 0
	})

	var activities = make([]syncmodel.ActivityRecord, 0, len(translated))
	for index = range translated {
		activities = append(activities, translated[index].Record)
	}

	var years = append([]int(nil), dataset.SupportedYears...)
	sort.Ints(years)

	return syncmodel.ProtectedActivityCache{
		RetrievedCount:       len(dataset.Activities),
		ActivityCount:        len(activities),
		AvailableReportYears: years,
		ScopeReliability:     normalize.DeriveScopeReliability(activities),
		Activities:           activities,
	}, nil
}

// translatedEmpiricalActivity stores one translated record with the ordering
// metadata needed to preserve deterministic empirical replay order.
// Authored by: OpenCode
type translatedEmpiricalActivity struct {
	OccurredAt         time.Time
	DeterministicOrder int
	AssetIdentityKey   string
	Record             syncmodel.ActivityRecord
}

// translateEmpiricalActivity converts one empirical dataset row into the
// protected synced-activity record shape used by report calculation.
// Authored by: OpenCode
func translateEmpiricalActivity(activity EmpiricalActivity) (translatedEmpiricalActivity, error) {
	var occurredAtText = strings.TrimSpace(activity.OccurredAt)
	var occurredAt, err = time.Parse(time.RFC3339Nano, occurredAtText)
	if err != nil {
		return translatedEmpiricalActivity{}, fmt.Errorf("translate empirical activity %q occurred_at: %w", strings.TrimSpace(activity.SourceID), err)
	}

	var quantity apd.Decimal
	quantity, _, err = ParseDecimalString(activity.Quantity)
	if err != nil {
		return translatedEmpiricalActivity{}, fmt.Errorf("translate empirical activity %q quantity: %w", strings.TrimSpace(activity.SourceID), err)
	}

	var record = syncmodel.ActivityRecord{
		SourceID:         strings.TrimSpace(activity.SourceID),
		OccurredAt:       occurredAtText,
		ActivityType:     activity.ActivityType,
		AssetIdentityKey: strings.TrimSpace(activity.AssetIdentityKey),
		AssetSymbol:      strings.TrimSpace(activity.AssetSymbol),
		AssetName:        strings.TrimSpace(activity.AssetSymbol),
		Quantity:         quantity,
		Comment:          strings.TrimSpace(activity.ZeroPricedReductionExplanation),
		DataSource:       "empirical_dataset",
		SourceScope:      translateEmpiricalScope(activity.SourceScope),
	}

	err = applyEmpiricalMoneyFields(&record, activity)
	if err != nil {
		return translatedEmpiricalActivity{}, err
	}

	return translatedEmpiricalActivity{
		OccurredAt:         occurredAt,
		DeterministicOrder: activity.DeterministicOrder,
		AssetIdentityKey:   record.AssetIdentityKey,
		Record:             record,
	}, nil
}

// applyEmpiricalMoneyFields maps empirical monetary fields into the order-tier
// synced record shape used by report calculation.
// Authored by: OpenCode
func applyEmpiricalMoneyFields(record *syncmodel.ActivityRecord, activity EmpiricalActivity) error {
	var grossValue, err = translateOptionalEmpiricalDecimal(activity.SourceID, "gross_value", activity.GrossValue)
	if err != nil {
		return err
	}
	var unitPrice *apd.Decimal
	unitPrice, err = translateOptionalEmpiricalDecimal(activity.SourceID, "unit_price", activity.UnitPrice)
	if err != nil {
		return err
	}
	var feeAmount *apd.Decimal
	feeAmount, err = translateOptionalEmpiricalDecimal(activity.SourceID, "fee_amount", activity.FeeAmount)
	if err != nil {
		return err
	}

	if strings.TrimSpace(activity.ZeroPricedReductionExplanation) != "" {
		if activity.ActivityType != syncmodel.ActivityTypeSell {
			return fmt.Errorf("translate empirical activity %q zero-priced holding reduction: activity_type must be SELL", strings.TrimSpace(activity.SourceID))
		}

		record.OrderGrossValue = firstNonNilDecimal(grossValue, zeroDecimalPointer())
		record.OrderUnitPrice = firstNonNilDecimal(unitPrice, zeroDecimalPointer())
		record.OrderFeeAmount = firstNonNilDecimal(feeAmount, zeroDecimalPointer())
		return nil
	}

	record.OrderCurrency = strings.TrimSpace(activity.Currency)
	record.OrderGrossValue = grossValue
	record.OrderUnitPrice = unitPrice
	record.OrderFeeAmount = feeAmount
	if record.OrderCurrency != "" && record.OrderFeeAmount == nil {
		record.OrderFeeAmount = zeroDecimalPointer()
	}

	return nil
}

// translateOptionalEmpiricalDecimal parses one optional empirical decimal field
// into the calculation-layer decimal pointer shape.
// Authored by: OpenCode
func translateOptionalEmpiricalDecimal(sourceID string, field string, raw string) (*apd.Decimal, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var value, _, err = ParseDecimalString(raw)
	if err != nil {
		return nil, fmt.Errorf("translate empirical activity %q %s: %w", strings.TrimSpace(sourceID), field, err)
	}

	return &value, nil
}

// translateEmpiricalScope copies one optional empirical scope into the synced
// protected-activity record shape.
// Authored by: OpenCode
func translateEmpiricalScope(scope *EmpiricalScope) *syncmodel.SourceScope {
	if scope == nil {
		return nil
	}

	return &syncmodel.SourceScope{
		ID:          strings.TrimSpace(scope.ScopeID),
		Name:        strings.TrimSpace(scope.DisplayName),
		Kind:        scope.ScopeKind,
		Reliability: scope.Reliability,
	}
}

// compareTranslatedEmpiricalActivities applies the empirical replay ordering
// rule based on timestamp, deterministic order, asset identity, and source ID.
// Authored by: OpenCode
func compareTranslatedEmpiricalActivities(left translatedEmpiricalActivity, right translatedEmpiricalActivity) int {
	if left.OccurredAt.Before(right.OccurredAt) {
		return -1
	}
	if left.OccurredAt.After(right.OccurredAt) {
		return 1
	}
	if left.DeterministicOrder != right.DeterministicOrder {
		if left.DeterministicOrder < right.DeterministicOrder {
			return -1
		}
		return 1
	}
	if comparison := strings.Compare(left.AssetIdentityKey, right.AssetIdentityKey); comparison != 0 {
		return comparison
	}

	return strings.Compare(left.Record.SourceID, right.Record.SourceID)
}

// firstNonNilDecimal returns the first available decimal pointer from the
// provided priority list.
// Authored by: OpenCode
func firstNonNilDecimal(values ...*apd.Decimal) *apd.Decimal {
	for _, value := range values {
		if value != nil {
			return value
		}
	}

	return nil
}

// zeroDecimalPointer returns a new explicit zero-value decimal pointer for
// translated zero-priced holding reductions.
// Authored by: OpenCode
func zeroDecimalPointer() *apd.Decimal {
	var zero = *apd.New(0, 0)
	return &zero
}
