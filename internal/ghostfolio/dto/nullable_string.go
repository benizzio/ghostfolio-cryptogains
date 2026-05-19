// Package dto defines the Ghostfolio transport models required by the sync and
// protected-storage slices.
// Authored by: OpenCode
package dto

import "encoding/json"

// NullableString accepts a JSON string or null while preserving the current
// slice's empty-string fallback for uninformed optional fields.
// Authored by: OpenCode
type NullableString string

// UnmarshalJSON accepts JSON strings and null for Ghostfolio fields whose
// upstream contract can be nullable.
//
// Example:
//
//	var value dto.NullableString
//	_ = value.UnmarshalJSON([]byte(`"CHF"`))
//
// Authored by: OpenCode
func (s *NullableString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = ""
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*s = NullableString(value)

	return nil
}

// String returns the underlying string value.
// Authored by: OpenCode
func (s NullableString) String() string {
	return string(s)
}
