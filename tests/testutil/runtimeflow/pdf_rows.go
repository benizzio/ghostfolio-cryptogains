package runtimeflow

import (
	"math"
	"sort"
	"strings"
	"unicode"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// These coordinates and tolerances mirror the production Annex geometry and
// the existing conversion-table inspection policy.
// Authored by: OpenCode
const (
	auditPDFSourceColumnTolerance    = 0.1
	auditPDFLineGap                  = 16.0
	auditPDFTableStartX              = 36.0
	auditPDFTableWidth               = 770.0
	conversionPDFCoordinateTolerance = 0.01
	conversionPDFLineGap             = 16.0
)

// AnnexPDFColumnCount is the fixed number of semantic columns in the PDF Annex
// table. For example, compare it with AnnexPDFSemanticCells output before
// reading a fixed column position.
// Authored by: OpenCode
const AnnexPDFColumnCount = 17

// auditPDFAnnexColumnWidths mirrors the source Annex proportions after scaling
// to the printable page width.
// Authored by: OpenCode
var auditPDFAnnexColumnWidths = [...]float64{42, 38, 38, 34, 34, 32, 30, 34, 34, 38, 40, 34, 34, 34, 32, 38, 38}

// FindAnnexPDFSourceRuns locates a source identifier in or after the Annex page,
// including identifiers split across same-column wrapped text runs. For
// example, pass the inspected PDF text runs, the page containing the Annex
// heading, and the source ID under test.
// Authored by: OpenCode
func FindAnnexPDFSourceRuns(runs []testutil.PDFTextRun, firstPage int, sourceID string) ([]testutil.PDFTextRun, bool) {
	return findPDFSourceRuns(runs, sourceID, func(run testutil.PDFTextRun) bool {
		return run.Page >= firstPage
	}, auditPDFSourceColumnTolerance, auditPDFLineGap, true, NormalizePDFSourceID)
}

// AnnexPDFRowRuns expands an Annex source cell's vertical span to the complete
// physical row while excluding neighboring rows and headings. For example, use
// the source runs returned by FindAnnexPDFSourceRuns before mapping cells.
// Authored by: OpenCode
func AnnexPDFRowRuns(runs []testutil.PDFTextRun, sourceRuns []testutil.PDFTextRun) []testutil.PDFTextRun {
	if len(sourceRuns) == 0 {
		return nil
	}
	var page = sourceRuns[0].Page
	var minimumY, maximumY = pdfRunYBounds(sourceRuns)
	minimumY, maximumY = expandPDFRowBounds(runs, page, minimumY, maximumY)
	return pdfRunsInBounds(runs, page, minimumY, maximumY)
}

// pdfRunYBounds returns the vertical span of inspected PDF runs.
// Authored by: OpenCode
func pdfRunYBounds(runs []testutil.PDFTextRun) (float64, float64) {
	var minimumY = runs[0].Y
	var maximumY = runs[0].Y
	for _, run := range runs[1:] {
		minimumY = math.Min(minimumY, run.Y)
		maximumY = math.Max(maximumY, run.Y)
	}
	return minimumY, maximumY
}

// expandPDFRowBounds includes every adjacent wrapped baseline in a row.
// Authored by: OpenCode
func expandPDFRowBounds(runs []testutil.PDFTextRun, page int, minimumY float64, maximumY float64) (float64, float64) {
	// Repeat because a multiline note can extend the row beyond the source span.
	for {
		var expanded bool
		minimumY, maximumY, expanded = expandPDFRowBoundsOnce(runs, page, minimumY, maximumY)
		if !expanded {
			return minimumY, maximumY
		}
	}
}

// expandPDFRowBoundsOnce extends a row span by one scan over same-page runs.
// Authored by: OpenCode
func expandPDFRowBoundsOnce(runs []testutil.PDFTextRun, page int, minimumY float64, maximumY float64) (float64, float64, bool) {
	var expanded bool
	for _, run := range runs {
		if run.Page != page {
			continue
		}
		if run.Y > maximumY && run.Y-maximumY <= auditPDFLineGap {
			maximumY = run.Y
			expanded = true
		}
		if run.Y < minimumY && minimumY-run.Y <= auditPDFLineGap {
			minimumY = run.Y
			expanded = true
		}
	}
	return minimumY, maximumY, expanded
}

// pdfRunsInBounds returns runs in the selected page and vertical span.
// Authored by: OpenCode
func pdfRunsInBounds(runs []testutil.PDFTextRun, page int, minimumY float64, maximumY float64) []testutil.PDFTextRun {
	var rowRuns []testutil.PDFTextRun
	for _, run := range runs {
		if run.Page == page && run.Y >= minimumY-0.01 && run.Y <= maximumY+0.01 {
			rowRuns = append(rowRuns, run)
		}
	}
	return rowRuns
}

// AnnexPDFSemanticCells maps physical Annex row runs to fixed columns and joins
// wrapped fragments without losing blank cells. For example, pass the output
// from AnnexPDFRowRuns to obtain all 17 semantic cells in source order.
// Authored by: OpenCode
func AnnexPDFSemanticCells(runs []testutil.PDFTextRun) []string {
	var fragments = make([][]testutil.PDFTextRun, AnnexPDFColumnCount)
	for _, run := range runs {
		var column, ok = annexPDFColumnIndex(run.X)
		if ok {
			fragments[column] = append(fragments[column], run)
		}
	}

	var cells = make([]string, AnnexPDFColumnCount)
	for column, columnRuns := range fragments {
		if column == 1 {
			cells[column] = joinPDFSourceRuns(columnRuns)
		} else {
			cells[column] = joinPDFCellRuns(columnRuns)
		}
	}
	return cells
}

// NonEmptyPDFCells removes blank semantic cells for comparisons with renderers
// that do not emit a text-showing operation for an empty cell. For example, use
// it to compare a blank classified currency cell across Markdown and PDF.
// Authored by: OpenCode
func NonEmptyPDFCells(cells []string) []string {
	var nonEmpty []string
	for _, cell := range cells {
		if cell != "" {
			nonEmpty = append(nonEmpty, cell)
		}
	}
	return nonEmpty
}

// NormalizePDFSourceID compares source IDs independently of case, whitespace,
// and non-semantic CID artifacts in extracted PDF text. For example, it makes
// a wrapped or CID-decorated source run comparable with its fixture ID.
// Authored by: OpenCode
func NormalizePDFSourceID(value string) string {
	var normalized strings.Builder
	for _, character := range strings.ToUpper(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' {
			normalized.WriteRune(character)
		}
	}
	return normalized.String()
}

// FindPDFConversionRowRuns isolates one conversion-table row using Annex
// section boundaries, source-ID coordinates, and neighboring source baselines.
// For example, pass an inspected PDF and a ConversionAuditEntry.SourceID before
// checking row-local converted entries.
// Authored by: OpenCode
func FindPDFConversionRowRuns(inspection testutil.GeneratedPDF, sourceID string) ([]testutil.PDFTextRun, bool) {
	var annexPage, conversionPage int
	var conversionY float64
	var foundAnnex, foundConversion bool
	for _, run := range inspection.TextRuns {
		if run.Text == "Annex 1 - Audit" && !foundAnnex {
			annexPage = run.Page
			foundAnnex = true
		}
		if run.Text == "Currency Conversion Audit" && foundAnnex {
			conversionPage = run.Page
			conversionY = run.Y
			foundConversion = true
			break
		}
	}
	if !foundAnnex || !foundConversion {
		return nil, false
	}

	var sourceRuns, found = findPDFConversionSourceRuns(inspection, sourceID, annexPage, conversionPage, conversionY)
	if !found {
		return nil, false
	}
	var sourceY = pdfSourceCenterY(sourceRuns)
	var sourceYs = pdfConversionSourceRowYs(inspection, sourceRuns[0].Page, sourceRuns[0].X, annexPage, conversionPage, conversionY)
	var neighborhood = pdfConversionRowNeighborhood(sourceY, sourceYs)
	var rowRuns []testutil.PDFTextRun
	for _, run := range inspection.TextRuns {
		if run.Page == sourceRuns[0].Page && math.Abs(run.Y-sourceY) <= neighborhood+0.01 {
			rowRuns = append(rowRuns, run)
		}
	}
	return rowRuns, len(rowRuns) > 0
}

// PDFConversionStartY locates one row-local converted-amount label in semantic
// coordinate order. For example, pass the source ID, label, and zero-based
// occurrence when duplicate amount kinds are present.
// Authored by: OpenCode
func PDFConversionStartY(inspection testutil.GeneratedPDF, sourceID string, label string, occurrence int) (float64, bool) {
	var rowRuns, found = FindPDFConversionRowRuns(inspection, sourceID)
	if !found {
		return 0, false
	}
	var cellRuns = pdfConversionCellRuns(rowRuns)
	var matches []testutil.PDFTextRun
	for _, run := range cellRuns {
		if strings.Contains(run.Text, label+":") {
			matches = append(matches, run)
		}
	}
	if occurrence < 0 || occurrence >= len(matches) {
		return 0, false
	}
	return matches[occurrence].Y, true
}

// PDFConversionContainsEntry verifies a complete converted entry within the
// selected conversion row instead of relying on searchable text from another
// row. For example, pass FindPDFConversionRowRuns output and the expected
// renderer entry string.
// Authored by: OpenCode
func PDFConversionContainsEntry(rowRuns []testutil.PDFTextRun, expected string) bool {
	var normalizedExpected = strings.Join(strings.Fields(expected), " ")
	for _, run := range pdfConversionCellRuns(rowRuns) {
		var normalizedRun = strings.Join(strings.Fields(strings.ReplaceAll(run.Text, ";", "")), " ")
		if strings.Contains(normalizedRun, normalizedExpected) {
			return true
		}
	}
	return false
}

// findPDFSourceRuns performs the shared wrapped-source lookup used by Annex and
// conversion-table inspection while preserving each table's coordinate policy.
// Authored by: OpenCode
func findPDFSourceRuns(runs []testutil.PDFTextRun, sourceID string, eligible func(testutil.PDFTextRun) bool, xTolerance float64, lineGap float64, samePage bool, normalize func(string) string) ([]testutil.PDFTextRun, bool) {
	var normalizedSourceID = normalize(sourceID)
	if normalizedSourceID == "" {
		return nil, false
	}
	for index, run := range runs {
		if !eligible(run) || normalize(run.Text) == "" {
			continue
		}
		var candidate, found = findPDFSourceCandidate(runs, index, run, normalizedSourceID, eligible, xTolerance, lineGap, samePage, normalize)
		if found {
			return candidate, true
		}
	}
	return nil, false
}

// findPDFSourceCandidate scans contiguous runs for one source identifier.
// Authored by: OpenCode
func findPDFSourceCandidate(runs []testutil.PDFTextRun, start int, sourceRun testutil.PDFTextRun, normalizedSourceID string, eligible func(testutil.PDFTextRun) bool, xTolerance float64, lineGap float64, samePage bool, normalize func(string) string) ([]testutil.PDFTextRun, bool) {
	var candidate []testutil.PDFTextRun
	var normalized strings.Builder
	for next := start; next < len(runs); next++ {
		var fragment = runs[next]
		if pdfSourceCandidateBreak(fragment, sourceRun, candidate, eligible, xTolerance, lineGap, samePage) {
			break
		}
		candidate = append(candidate, fragment)
		normalized.WriteString(normalize(fragment.Text))
		if strings.Contains(normalized.String(), normalizedSourceID) {
			return candidate, true
		}
	}
	return nil, false
}

// pdfSourceCandidateBreak reports whether a run cannot extend a source match.
// Authored by: OpenCode
func pdfSourceCandidateBreak(fragment testutil.PDFTextRun, sourceRun testutil.PDFTextRun, candidate []testutil.PDFTextRun, eligible func(testutil.PDFTextRun) bool, xTolerance float64, lineGap float64, samePage bool) bool {
	if !eligible(fragment) || samePage && fragment.Page != sourceRun.Page || math.Abs(fragment.X-sourceRun.X) > xTolerance {
		return true
	}
	return len(candidate) > 0 && math.Abs(fragment.Y-candidate[len(candidate)-1].Y) > lineGap
}

// annexPDFColumnIndex resolves a text run's X coordinate to the scaled Annex
// column that produced it, including right-aligned values.
// Authored by: OpenCode
func annexPDFColumnIndex(x float64) (int, bool) {
	var totalWidth float64
	for _, width := range auditPDFAnnexColumnWidths {
		totalWidth += width
	}
	var sourcePosition = (x - auditPDFTableStartX) * totalWidth / auditPDFTableWidth
	if sourcePosition < 0 || sourcePosition >= totalWidth {
		return 0, false
	}
	var columnStart float64
	for column, width := range auditPDFAnnexColumnWidths {
		if sourcePosition < columnStart+width {
			return column, true
		}
		columnStart += width
	}
	return 0, false
}

// joinPDFSourceRuns joins identifier fragments across PDF line splits.
// Authored by: OpenCode
func joinPDFSourceRuns(runs []testutil.PDFTextRun) string {
	var value strings.Builder
	for _, run := range runs {
		value.WriteString(strings.Join(strings.Fields(run.Text), ""))
	}
	return value.String()
}

// joinPDFCellRuns joins same-line fragments directly and wrapped lines with
// spaces so semantic cells match their single-line Markdown values.
// Authored by: OpenCode
func joinPDFCellRuns(runs []testutil.PDFTextRun) string {
	if len(runs) == 0 {
		return ""
	}
	var ordered = append([]testutil.PDFTextRun(nil), runs...)
	sort.SliceStable(ordered, func(left, right int) bool {
		return ordered[left].Y > ordered[right].Y
	})
	var value strings.Builder
	for index, run := range ordered {
		var fragment = strings.Join(strings.Fields(run.Text), " ")
		if fragment == "" {
			continue
		}
		if index > 0 && value.Len() > 0 && math.Abs(run.Y-ordered[index-1].Y) > 0.01 {
			value.WriteByte(' ')
		}
		value.WriteString(fragment)
	}
	return value.String()
}

// findPDFConversionSourceRuns locates a possibly wrapped source ID below the
// conversion heading and inside the Annex page range.
// Authored by: OpenCode
func findPDFConversionSourceRuns(inspection testutil.GeneratedPDF, sourceID string, annexPage int, conversionPage int, conversionY float64) ([]testutil.PDFTextRun, bool) {
	return findPDFSourceRuns(inspection.TextRuns, sourceID, func(run testutil.PDFTextRun) bool {
		return pdfConversionRunInSection(run, annexPage, conversionPage, conversionY)
	}, conversionPDFCoordinateTolerance, conversionPDFLineGap, false, pdfConversionSourceText)
}

// pdfConversionSourceRowYs groups source-column fragments into row centers.
// Authored by: OpenCode
func pdfConversionSourceRowYs(inspection testutil.GeneratedPDF, page int, sourceX float64, annexPage int, conversionPage int, conversionY float64) []float64 {
	var ys []float64
	for _, run := range inspection.TextRuns {
		if run.Page == page && math.Abs(run.X-sourceX) <= conversionPDFCoordinateTolerance && pdfConversionRunInSection(run, annexPage, conversionPage, conversionY) {
			ys = append(ys, run.Y)
		}
	}
	sort.Float64s(ys)
	var centers []float64
	for _, y := range ys {
		if len(centers) == 0 || y-centers[len(centers)-1] > conversionPDFLineGap {
			centers = append(centers, y)
			continue
		}
		centers[len(centers)-1] = (centers[len(centers)-1] + y) / 2
	}
	return centers
}

// pdfSourceCenterY returns the center of a wrapped source cell.
// Authored by: OpenCode
func pdfSourceCenterY(sourceRuns []testutil.PDFTextRun) float64 {
	var minimumY = sourceRuns[0].Y
	var maximumY = sourceRuns[0].Y
	for _, run := range sourceRuns[1:] {
		minimumY = math.Min(minimumY, run.Y)
		maximumY = math.Max(maximumY, run.Y)
	}
	return (minimumY + maximumY) / 2
}

// pdfConversionRowNeighborhood returns the midpoint to the nearest source row.
// Authored by: OpenCode
func pdfConversionRowNeighborhood(sourceY float64, sourceYs []float64) float64 {
	var neighborhood = 18.0
	for _, otherY := range sourceYs {
		if math.Abs(otherY-sourceY) <= 0.01 {
			continue
		}
		neighborhood = math.Min(neighborhood, math.Abs(otherY-sourceY)/2)
	}
	return neighborhood
}

// pdfConversionCellRuns keeps only the converted-amount column and orders its
// physical lines by semantic coordinate order.
// Authored by: OpenCode
func pdfConversionCellRuns(rowRuns []testutil.PDFTextRun) []testutil.PDFTextRun {
	var anchorX float64
	var found bool
	for _, run := range rowRuns {
		if strings.Contains(run.Text, string(reportmodel.ConvertedAmountKindUnitPrice)+":") || strings.Contains(run.Text, string(reportmodel.ConvertedAmountKindGrossValue)+":") || strings.Contains(run.Text, string(reportmodel.ConvertedAmountKindFeeAmount)+":") {
			anchorX = run.X
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	var result []testutil.PDFTextRun
	for _, run := range rowRuns {
		if math.Abs(run.X-anchorX) <= conversionPDFCoordinateTolerance {
			result = append(result, run)
		}
	}
	sort.SliceStable(result, func(left, right int) bool {
		return result[left].Y < result[right].Y
	})
	return result
}

// pdfConversionRunInSection reports whether a run is below the conversion
// heading and inside the Annex page range.
// Authored by: OpenCode
func pdfConversionRunInSection(run testutil.PDFTextRun, annexPage int, conversionPage int, conversionY float64) bool {
	return run.Page >= annexPage && (run.Page > conversionPage || run.Page == conversionPage && run.Y < conversionY)
}

// pdfConversionSourceText removes line whitespace without changing source-ID
// punctuation.
// Authored by: OpenCode
func pdfConversionSourceText(value string) string {
	return strings.Join(strings.Fields(value), "")
}
