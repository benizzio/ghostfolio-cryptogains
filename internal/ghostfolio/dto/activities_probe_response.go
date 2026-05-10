// Package dto defines the minimal Ghostfolio transport models required by this
// validation-only slice.
// Authored by: OpenCode
package dto

// ActivitiesProbeResponse is the minimal successful activities probe response
// required by this slice.
//
// Example:
//
//	response := dto.ActivitiesProbeResponse{Count: 0, Activities: []dto.ActivityProbeEntry{}}
//	_ = response.Count
//
// Authored by: OpenCode
type ActivitiesProbeResponse struct {
	Activities []ActivityProbeEntry `json:"activities"`
	Count      int                  `json:"count"`
}

// ActivityProbeEntry is the minimal activity shape required by the one-page
// validation probe in this slice.
//
// Example:
//
//	entry := dto.ActivityProbeEntry{ID: "id", Date: "2026-01-31T10:00:00.000Z", Type: "BUY"}
//	_ = entry.Type
//
// Authored by: OpenCode
type ActivityProbeEntry struct {
	ID   string `json:"id"`
	Date string `json:"date"`
	Type string `json:"type"`
}
