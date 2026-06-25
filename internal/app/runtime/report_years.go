// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"fmt"
	"strings"
)

// joinAvailableYears formats one readable available-year list.
// Authored by: OpenCode
func joinAvailableYears(years []int) string {
	var parts = make([]string, 0, len(years))
	for _, year := range years {
		parts = append(parts, fmt.Sprintf("%d", year))
	}

	return strings.Join(parts, ", ")
}

// containsReportYear reports whether the selected year exists in the unlocked
// cache metadata.
// Authored by: OpenCode
func containsReportYear(years []int, selectedYear int) bool {
	for _, year := range years {
		if year == selectedYear {
			return true
		}
	}

	return false
}
