// Package calculate defines structured calculation error helpers.
// Authored by: OpenCode
package calculate

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// newGroupCalculationError creates one structured calculation error from grouped asset context.
// Authored by: OpenCode
func newGroupCalculationError(kind reportmodel.CalculationErrorKind, group assetInputGroup, message string, cause error) error {
	return reportmodel.NewCalculationError(kind, message, "", group.DisplayLabel, cause)
}

// newRecordCalculationError creates one structured calculation error from a
// normalized synced activity record.
// Authored by: OpenCode
func newRecordCalculationError(kind reportmodel.CalculationErrorKind, record syncmodel.ActivityRecord, message string, cause error) error {
	return reportmodel.NewCalculationError(kind, message, strings.TrimSpace(record.SourceID), activityDisplayLabel(record), cause).WithPersistedActivityRecord(&record)
}

// newInputCalculationError creates one structured calculation error from a
// selected activity calculation input.
// Authored by: OpenCode
func newInputCalculationError(kind reportmodel.CalculationErrorKind, input reportmodel.ActivityCalculationInput, message string, cause error) error {
	return reportmodel.NewCalculationError(kind, message, strings.TrimSpace(input.SourceID), strings.TrimSpace(input.DisplayLabel), cause).WithPersistedActivityRecord(input.PersistedActivityRecord)
}
