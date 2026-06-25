package date

import (
	"testing"
	"time"
)

// TestCalendarDateNormalizesToUTCMidnight verifies reusable source-calendar
// normalization preserves the local date and removes clock precision.
// Authored by: OpenCode
func TestCalendarDateNormalizesToUTCMidnight(t *testing.T) {
	t.Parallel()

	var normalized = CalendarDate(time.Date(2024, time.May, 21, 22, 3, 4, 0, time.FixedZone("offset", 2*60*60)))
	var expected = time.Date(2024, time.May, 21, 0, 0, 0, 0, time.UTC)
	if !normalized.Equal(expected) {
		t.Fatalf("unexpected calendar date: got %v want %v", normalized, expected)
	}
	var got = FormatCalendarDate(normalized)
	if got != "2024-05-21" {
		t.Fatalf("unexpected calendar date rendering: %q", got)
	}
	if !CalendarDate(time.Time{}).IsZero() {
		t.Fatalf("expected zero time to remain zero")
	}
}
