// Package dto defines the Ghostfolio transport models required by the sync and
// protected-storage slices.
// Authored by: OpenCode
package dto

// UserResponse is the authenticated Ghostfolio user response required by the
// sync slice.
//
// Authored by: OpenCode
type UserResponse struct {
	Settings *UserSettings `json:"settings"`
}

// UserSettings preserves the minimal authenticated Ghostfolio user settings
// needed by the sync slice.
//
// Authored by: OpenCode
type UserSettings struct {
	BaseCurrency string `json:"baseCurrency"`
}
