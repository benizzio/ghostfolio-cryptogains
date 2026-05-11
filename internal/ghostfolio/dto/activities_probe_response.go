// Package dto defines the minimal Ghostfolio transport models required by this
// validation-only slice.
// Authored by: OpenCode
package dto

// ActivitiesProbeResponse is the minimal successful activities probe response
// required by this slice.
//
// Authored by: OpenCode
type ActivitiesProbeResponse struct {
	Activities []ActivityProbeEntry `json:"activities"`
	Count      int                  `json:"count"`
}

// ActivityProbeEntry is the minimal activity shape required by the one-page
// validation probe in this slice.
//
// Authored by: OpenCode
type ActivityProbeEntry struct {
	ID   string `json:"id"`
	Date string `json:"date"`
	Type string `json:"type"`
}
