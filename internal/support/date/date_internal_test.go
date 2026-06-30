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

// TestLeadingCalendarDateExtractsValidatedToken verifies leading date tokens
// are trimmed, validated, and boundary checked.
// Authored by: OpenCode
func TestLeadingCalendarDateExtractsValidatedToken(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "exact", value: "2024-01-02", want: "2024-01-02"},
		{name: "trimmed with suffix", value: "  2024-01-02 conversion failed  ", want: "2024-01-02"},
		{name: "iso date time", value: "2024-01-02T10:00:00Z", want: "2024-01-02"},
		{name: "punctuation boundary", value: "2024-01-02)", want: "2024-01-02"},
		{name: "invalid calendar day", value: "2024-02-31 conversion failed", want: ""},
		{name: "too short", value: "2024-01", want: ""},
		{name: "non boundary suffix", value: "2024-01-02x", want: ""},
		{name: "invalid time delimiter suffix", value: "2024-01-02Tbad", want: ""},
		{name: "underscore suffix", value: "2024-01-02_extra", want: ""},
	} {
		var tc = tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := LeadingCalendarDate(tc.value); got != tc.want {
				t.Fatalf("unexpected leading calendar date: got %q want %q", got, tc.want)
			}
		})
	}
}
