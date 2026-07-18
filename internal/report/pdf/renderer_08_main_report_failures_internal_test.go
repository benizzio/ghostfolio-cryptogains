package pdf

import (
	"testing"

	"github.com/cockroachdb/apd/v3"
)

// TestRemainingMainReportErrorBranches verifies empty-state and position
// currency failures that require the exact downstream layout operation.
// Authored by: OpenCode
func TestRemainingMainReportErrorBranches(t *testing.T) {
	var zeroReport = minimalPDFReportFixture(t)
	assertErrorContains(t, func() error {
		return renderSummarySection(&errorLayoutRecorder{failKey: "Overall Yearly Net Total"}, zeroReport, "USD")
	}, "key failed")
	assertErrorContains(t, func() error {
		return renderRateSourceSection(&errorLayoutRecorder{failParagraph: true}, zeroReport)
	}, "paragraph failed")
	assertErrorContains(t, func() error {
		return renderReferenceSection(&errorLayoutRecorder{failParagraph: true}, zeroReport)
	}, "paragraph failed")
	assertErrorContains(t, func() error {
		return renderPositionBlock(&errorLayoutRecorder{failKey: "Calculation Currency"}, "Position", *apd.New(1, 0), *apd.New(1, 0), "USD", "USD")
	}, "key failed")
}
