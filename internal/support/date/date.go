// Package date provides reusable source-calendar date normalization helpers for
// report calculation and exchange-rate evidence boundaries.
// Authored by: OpenCode
package date

import "time"

// CalendarDate strips clock and location fields from one timestamp while
// preserving the timestamp's local calendar day. A zero time remains zero.
//
// Example:
//
//	day := date.CalendarDate(time.Now())
//	_ = day
//
// Authored by: OpenCode
func CalendarDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	var year, month, day = value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// FormatCalendarDate returns the canonical YYYY-MM-DD rendering for a
// source-calendar date.
//
// Example:
//
//	formatted := date.FormatCalendarDate(time.Now())
//	_ = formatted
//
// Authored by: OpenCode
func FormatCalendarDate(value time.Time) string {
	return CalendarDate(value).Format(time.DateOnly)
}
