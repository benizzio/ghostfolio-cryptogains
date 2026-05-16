// Package dto defines the Ghostfolio transport models required by the sync and
// protected-storage slices.
// Authored by: OpenCode
package dto

// AuthResponse is the successful anonymous-auth response required by the sync
// slice.
//
// Authored by: OpenCode
type AuthResponse struct {
	AuthToken string `json:"authToken"`
}
