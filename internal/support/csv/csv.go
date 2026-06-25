// Package csv provides small reusable helpers for working with CSV records.
//
// The package intentionally avoids parsing files. It only contains policy-light
// utilities that can be shared by provider mappers, tests, and tools after a CSV
// header has already been read.
// Authored by: OpenCode
package csv

import (
	"fmt"
	"strings"
)

// RequiredColumnIndexes returns the header indexes for required CSV columns in
// the same order as requiredNames.
//
// Header values are compared after trimming surrounding whitespace. Column names
// remain case-sensitive so callers can enforce provider-specific contracts when
// needed. The returned slice is independent of the input arguments and can be
// retained by the caller.
//
// Example usage:
//
//	indexes, err := csv.RequiredColumnIndexes([]string{"DATE", " VALUE "}, "DATE", "VALUE")
//	if err != nil {
//		panic(err)
//	}
//	dateColumn := indexes[0]
//	valueColumn := indexes[1]
//	_, _ = dateColumn, valueColumn
//
// Authored by: OpenCode
func RequiredColumnIndexes(header []string, requiredNames ...string) ([]int, error) {
	var indexes = make([]int, len(requiredNames))
	for index := range indexes {
		indexes[index] = -1
	}

	for columnIndex, column := range header {
		var normalized = strings.TrimSpace(column)
		for requiredIndex, requiredName := range requiredNames {
			if normalized == requiredName && indexes[requiredIndex] < 0 {
				indexes[requiredIndex] = columnIndex
			}
		}
	}

	var missing []string
	for index, columnIndex := range indexes {
		if columnIndex < 0 {
			missing = append(missing, requiredNames[index])
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("required columns %s are missing", strings.Join(missing, " and "))
	}

	return indexes, nil
}
