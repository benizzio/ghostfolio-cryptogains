// Package date provides reusable source-calendar date normalization helpers for
// report calculation and exchange-rate evidence boundaries.
// Authored by: OpenCode
package date

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

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

// LeadingCalendarDate extracts a valid leading YYYY-MM-DD calendar date token
// from arbitrary text. It trims surrounding whitespace, validates the token
// with time.DateOnly parsing, and rejects tokens continued by letters, digits,
// or underscores. ISO date-time text may continue with a T time delimiter.
//
// Example:
//
//	day := date.LeadingCalendarDate("2024-01-02 conversion failed")
//	_ = day
//
// Authored by: OpenCode
func LeadingCalendarDate(value string) string {
	var normalized = strings.TrimSpace(value)
	if len(normalized) < len(time.DateOnly) {
		return ""
	}

	var token = normalized[:len(time.DateOnly)]
	if _, err := time.Parse(time.DateOnly, token); err != nil {
		return ""
	}
	if len(normalized) == len(time.DateOnly) || calendarDateTokenBoundary(normalized[len(time.DateOnly):]) {
		return token
	}
	return ""
}

// calendarDateTokenBoundary reports whether the next rune cannot be part of a
// plain text date-like token.
// Authored by: OpenCode
func calendarDateTokenBoundary(value string) bool {
	var next, _ = utf8.DecodeRuneInString(value)
	if next == 'T' && len(value) > 1 {
		var afterTimeDelimiter, _ = utf8.DecodeRuneInString(value[1:])
		return unicode.IsDigit(afterTimeDelimiter)
	}
	return next != '_' && !unicode.IsLetter(next) && !unicode.IsDigit(next)
}
