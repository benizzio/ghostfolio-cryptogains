// Package dto defines the minimal Ghostfolio transport models required by this
// validation-only slice.
// Authored by: OpenCode
package dto

// AuthResponse is the minimal successful anonymous-auth response required by
// this slice.
//
// Example:
//
//	response := dto.AuthResponse{AuthToken: "jwt"}
//	_ = response.AuthToken
//
// Authored by: OpenCode
type AuthResponse struct {
	AuthToken string `json:"authToken"`
}
