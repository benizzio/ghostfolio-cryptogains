// Package dto defines the Ghostfolio transport models required by the sync and
// protected-storage slices.
// Authored by: OpenCode
package dto

// ActivitiesProbeResponse preserves the existing one-page validation response
// alias while the full-history page model becomes the shared transport type.
// Authored by: OpenCode
type ActivitiesProbeResponse = ActivityPageResponse

// ActivityProbeEntry preserves the existing one-page validation entry alias
// while the full-history page model becomes the shared transport type.
// Authored by: OpenCode
type ActivityProbeEntry = ActivityPageEntry
