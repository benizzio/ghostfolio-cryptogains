// Package contract verifies concrete converted-amount rendering contracts.
// Authored by: OpenCode
package contract

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	testreportpdf "github.com/benizzio/ghostfolio-cryptogains/tests/testutil/reportpdf"
)

// TestReportConvertedAmountsPopulationContract verifies the concrete C and E
// denominators for the eight closed converted-amount subsequences.
// Authored by: OpenCode
func TestReportConvertedAmountsPopulationContract(t *testing.T) {
	t.Parallel()

	var manifest = testutil.DeterministicReportPresentationAcceptanceFixture()
	var cases = 0
	var entryCases = 0
	for _, acceptanceCase := range manifest.Cases {
		if acceptanceCase.Kind != testutil.ReportPresentationCaseKindConverted {
			continue
		}
		cases++
		entryCases += len(acceptanceCase.ConvertedAmountKinds)
	}
	var conversionRows = reportRenderingPopulationCount(manifest.Cases, testutil.ReportPresentationPopulationConversionRow)
	var convertedEntries = reportRenderingPopulationCount(manifest.Cases, testutil.ReportPresentationPopulationConvertedEntry)
	var conversionRowsDenominator = reportRenderingPopulationCounter(t, manifest.Counters, testutil.ReportPresentationPopulationConversionRow)
	var convertedEntriesDenominator = reportRenderingPopulationCounter(t, manifest.Counters, testutil.ReportPresentationPopulationConvertedEntry)
	if cases != 8 || conversionRows != 16 || conversionRowsDenominator != 16 {
		t.Fatalf("converted population C = cases %d, numerator/denominator %d/%d, want 8 and 16/16", cases, conversionRows, conversionRowsDenominator)
	}
	if entryCases != 12 || convertedEntries != 24 || convertedEntriesDenominator != 24 {
		t.Fatalf("converted population E = cases %d, numerator/denominator %d/%d, want 12 and 24/24", entryCases, convertedEntries, convertedEntriesDenominator)
	}
}

// TestReportConvertedAmountsConcreteMarkdownContract verifies exact Markdown
// cells, controlled boundaries, omission, and received entry order.
// Authored by: OpenCode
func TestReportConvertedAmountsConcreteMarkdownContract(t *testing.T) {
	t.Parallel()

	var report = contractConvertedAmountsReportFixture()
	var document, err = reportmarkdown.RenderAnnex(report)
	if err != nil {
		t.Fatalf("render converted-amount Annex: %v", err)
	}
	var sequences = contractConvertedAuditSequences()
	for _, sequence := range sequences {
		var cell, found = contractMarkdownConvertedAmountsCell(string(document.Content), sequence.SourceID)
		if !found {
			t.Fatalf("Markdown conversion row %q is missing", sequence.SourceID)
		}
		if cell != sequence.ExpectedMarkdownCell {
			t.Fatalf("Markdown conversion cell %q = %q, want %q", sequence.SourceID, cell, sequence.ExpectedMarkdownCell)
		}
		assertContractConvertedCellShape(t, sequence, cell)
	}
}

// TestReportConvertedAmountsConcretePDFContract verifies exact row-local PDF
// text, one logical start per entry, later vertical coordinates, order, and
// searchable/selectable content.
// Authored by: OpenCode
func TestReportConvertedAmountsConcretePDFContract(t *testing.T) {
	t.Parallel()

	var inspection, err = testreportpdf.RenderAndInspect(contractConvertedAmountsReportFixture())
	if err != nil {
		t.Fatalf("render converted-amount PDF: %v", err)
	}
	assertLandscapeA4PDF(t, inspection)
	if !inspection.ContainsSearchableText("Currency Conversion Audit") {
		t.Fatal("PDF conversion audit heading is not searchable")
	}

	var sequences = contractConvertedAuditSequences()
	var sourceIDs = make([]string, 0, len(sequences))
	for _, sequence := range sequences {
		sourceIDs = append(sourceIDs, sequence.SourceID)
	}
	for _, sequence := range sequences {
		if !inspection.ContainsSearchableText(sequence.SourceID) {
			t.Fatalf("PDF conversion source ID %q is not searchable", sequence.SourceID)
		}
		var rowRuns = contractPDFConversionRowRuns(inspection, sequence.SourceID, sourceIDs)
		if len(rowRuns) == 0 {
			t.Fatalf("PDF conversion row %q has no local text runs", sequence.SourceID)
		}
		var cellRuns = contractPDFConvertedCellRuns(rowRuns)
		var rowText = normalizeReportRenderingText(strings.ReplaceAll(strings.Join(contractPDFRunTexts(cellRuns), " "), ";", ""))
		var expectedText = normalizeReportRenderingText(strings.ReplaceAll(strings.ReplaceAll(sequence.ExpectedMarkdownCell, "<br>", " "), ";", ""))
		if expectedText != "" && !strings.Contains(rowText, expectedText) {
			t.Fatalf("PDF conversion row %q text = %q, want semantic cell %q", sequence.SourceID, rowText, expectedText)
		}
		if sequence.ExpectedMarkdownCell == "" {
			for _, kind := range contractConvertedAmountKinds() {
				if strings.Contains(rowText, string(kind)+":") {
					t.Fatalf("empty PDF conversion row %q contains omitted kind %q: %q", sequence.SourceID, kind, rowText)
				}
			}
		}
		var starts, ok = contractPDFEntryStartRuns(cellRuns, sequence.Kinds)
		if !ok {
			t.Fatalf("PDF conversion row %q does not contain one logical start per included entry: %q", sequence.SourceID, rowText)
		}
		for index := 1; index < len(starts); index++ {
			if math.Abs(starts[index].X-starts[0].X) > 0.01 {
				t.Fatalf("PDF conversion row %q entry %d X = %.2f, want cell origin %.2f", sequence.SourceID, index, starts[index].X, starts[0].X)
			}
			if starts[index].Y >= starts[index-1].Y-0.01 {
				t.Fatalf("PDF conversion row %q entry %d Y = %.2f, want lower than %.2f", sequence.SourceID, index, starts[index].Y, starts[index-1].Y)
			}
		}
		for _, kind := range sequence.Kinds {
			var expectedEntry = contractConvertedEntryText(kind)
			if !inspection.ContainsSearchableText(expectedEntry) {
				t.Fatalf("PDF conversion entry %q in row %q is not searchable", expectedEntry, sequence.SourceID)
			}
		}
	}
}

// contractPDFConvertedCellRuns isolates the Converted Amounts column by its
// recovered X coordinate and orders physical lines from top to bottom.
// Authored by: OpenCode
func contractPDFConvertedCellRuns(rowRuns []testutil.PDFTextRun) []testutil.PDFTextRun {
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
		if math.Abs(run.X-anchorX) <= 0.01 {
			result = append(result, run)
		}
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return result[left].Y > result[right].Y
	})
	return result
}

// contractConvertedSequence describes one concrete received conversion-kind
// sequence and its exact renderer-visible Markdown cell.
// Authored by: OpenCode
type contractConvertedSequence struct {
	ID                   string
	SourceID             string
	Kinds                []reportmodel.ConvertedAmountKind
	ExpectedMarkdownCell string
}

// contractConvertedAuditSequences returns the eight canonical subsequences and
// one duplicate/non-canonical received-order control.
// Authored by: OpenCode
func contractConvertedAuditSequences() []contractConvertedSequence {
	var sequences = []contractConvertedSequence{
		{ID: "empty", Kinds: nil},
		{ID: "unit-price", Kinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice}},
		{ID: "gross-value", Kinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindGrossValue}},
		{ID: "fee-amount", Kinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindFeeAmount}},
		{ID: "unit-price-gross-value", Kinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice, reportmodel.ConvertedAmountKindGrossValue}},
		{ID: "unit-price-fee-amount", Kinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice, reportmodel.ConvertedAmountKindFeeAmount}},
		{ID: "gross-value-fee-amount", Kinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindGrossValue, reportmodel.ConvertedAmountKindFeeAmount}},
		{ID: "all", Kinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice, reportmodel.ConvertedAmountKindGrossValue, reportmodel.ConvertedAmountKindFeeAmount}},
		{ID: "received-order", Kinds: []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindFeeAmount, reportmodel.ConvertedAmountKindGrossValue, reportmodel.ConvertedAmountKindFeeAmount}},
	}
	for index := range sequences {
		sequences[index].SourceID = fmt.Sprintf("cv%d", index)
		sequences[index].ExpectedMarkdownCell = contractConvertedSequenceText(sequences[index].Kinds)
	}
	return sequences
}

// contractConvertedAmountsReportFixture creates a valid synthetic report whose
// Annex conversion rows cover all concrete sequence contracts.
// Authored by: OpenCode
func contractConvertedAmountsReportFixture() reportmodel.CapitalGainsReport {
	var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
	var entries = make([]reportmodel.ConversionAuditEntry, 0, len(contractConvertedAuditSequences()))
	for _, sequence := range contractConvertedAuditSequences() {
		entries = append(entries, contractConvertedAuditEntryForSequence(sequence))
	}
	report.AuditAnnex.ConversionAuditEntries = entries
	return report
}

// contractConvertedAuditEntryForSequence creates one model-valid audit row for
// a sequence, retaining a zero-to-zero raw component for the empty case.
// Authored by: OpenCode
func contractConvertedAuditEntryForSequence(sequence contractConvertedSequence) reportmodel.ConversionAuditEntry {
	var entry = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label()).AuditAnnex.ConversionAuditEntries[0]
	entry.SourceID = sequence.SourceID
	entry.Amounts = nil
	var kinds = sequence.Kinds
	if len(kinds) == 0 {
		kinds = []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice}
	}
	for _, kind := range kinds {
		var original, converted = contractConvertedAmountValues(kind)
		if len(sequence.Kinds) == 0 {
			original = "0"
			converted = "0"
		}
		var amount = contractConvertedActivityAmount(kind, original, converted)
		amount.SourceID = sequence.SourceID
		entry.Amounts = append(entry.Amounts, amount)
	}
	return entry
}

// contractConvertedAmountValues returns the synthetic exact source and target
// values used by each supported converted amount kind.
// Authored by: OpenCode
func contractConvertedAmountValues(kind reportmodel.ConvertedAmountKind) (string, string) {
	switch kind {
	case reportmodel.ConvertedAmountKindUnitPrice:
		return "30754.70", "28673.04"
	case reportmodel.ConvertedAmountKindGrossValue:
		return "254.76", "237.52"
	case reportmodel.ConvertedAmountKindFeeAmount:
		return "1.79", "1.67"
	default:
		panic(fmt.Sprintf("unsupported contract converted amount kind %q", kind))
	}
}

// contractConvertedEntryText returns the exact single-entry syntax used in
// both output formats.
// Authored by: OpenCode
func contractConvertedEntryText(kind reportmodel.ConvertedAmountKind) string {
	var original, converted = contractConvertedAmountValues(kind)
	return fmt.Sprintf("%s: %s -> %s", kind, original, converted)
}

// contractConvertedSequenceText joins entries with the required Markdown
// renderer-controlled boundary.
// Authored by: OpenCode
func contractConvertedSequenceText(kinds []reportmodel.ConvertedAmountKind) string {
	var entries = make([]string, 0, len(kinds))
	for _, kind := range kinds {
		entries = append(entries, contractConvertedEntryText(kind))
	}
	return strings.Join(entries, ";<br>")
}

// contractMarkdownConvertedAmountsCell extracts one exact Converted Amounts
// cell from the Annex pipe table by its semantic source ID.
// Authored by: OpenCode
func contractMarkdownConvertedAmountsCell(content string, sourceID string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.Contains(line, "| "+sourceID+" |") {
			continue
		}
		var cells = strings.Split(line, "|")
		if len(cells) != 11 {
			return "", false
		}
		return strings.TrimSpace(cells[7]), true
	}
	return "", false
}

// assertContractConvertedCellShape verifies exact controlled-boundary and
// logical-start counts for one Markdown conversion cell.
// Authored by: OpenCode
func assertContractConvertedCellShape(t *testing.T, sequence contractConvertedSequence, cell string) {
	t.Helper()
	var included = len(sequence.Kinds)
	if strings.Count(cell, "<br>") != maxContractInt(included-1) {
		t.Fatalf("Markdown conversion row %q break count = %d, want %d", sequence.SourceID, strings.Count(cell, "<br>"), maxContractInt(included-1))
	}
	if strings.Count(cell, ";") != maxContractInt(included-1) {
		t.Fatalf("Markdown conversion row %q semicolon count = %d, want %d", sequence.SourceID, strings.Count(cell, ";"), maxContractInt(included-1))
	}
	if strings.HasPrefix(cell, "<br>") || strings.HasSuffix(cell, ";") {
		t.Fatalf("Markdown conversion row %q has an invalid boundary: %q", sequence.SourceID, cell)
	}
	for _, kind := range []reportmodel.ConvertedAmountKind{reportmodel.ConvertedAmountKindUnitPrice, reportmodel.ConvertedAmountKindGrossValue, reportmodel.ConvertedAmountKindFeeAmount} {
		var expectedStarts = 0
		for _, receivedKind := range sequence.Kinds {
			if receivedKind == kind {
				expectedStarts++
			}
		}
		if got := strings.Count(cell, string(kind)+":"); got != expectedStarts {
			t.Fatalf("Markdown conversion row %q logical starts for %q = %d, want %d", sequence.SourceID, kind, got, expectedStarts)
		}
	}
}

// maxContractInt returns the greater of two integer contract values.
// Authored by: OpenCode
func maxContractInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

// contractPDFConversionRowRuns returns one conversion row using the conversion
// section boundary and the nearest source-ID row baselines on that page.
// Authored by: OpenCode
func contractPDFConversionRowRuns(inspection testutil.GeneratedPDF, sourceID string, sourceIDs []string) []testutil.PDFTextRun {
	var sectionPage, sectionY, found = contractPDFConversionSectionStart(inspection)
	if !found {
		return nil
	}
	var sourceRun testutil.PDFTextRun
	sourceRun, found = contractPDFConversionSourceRun(inspection, sourceID, sectionPage, sectionY)
	if !found {
		return nil
	}
	var sourceYs = contractPDFConversionSourceYs(inspection, sourceIDs, sourceRun, sectionPage, sectionY)
	var neighborhood = contractPDFConversionRowNeighborhood(sourceRun.Y, sourceYs)
	var rowRuns []testutil.PDFTextRun
	for _, run := range inspection.TextRuns {
		if run.Page == sourceRun.Page && math.Abs(run.Y-sourceRun.Y) <= neighborhood+0.01 {
			rowRuns = append(rowRuns, run)
		}
	}
	return rowRuns
}

// contractPDFConversionSectionStart locates the first Currency Conversion Audit
// heading and returns its page-local boundary.
// Authored by: OpenCode
func contractPDFConversionSectionStart(inspection testutil.GeneratedPDF) (int, float64, bool) {
	for _, run := range inspection.TextRuns {
		if run.Text == "Currency Conversion Audit" {
			return run.Page, run.Y, true
		}
	}
	return 0, 0, false
}

// contractPDFConversionSourceRun locates a source ID after the conversion
// heading, excluding the detailed Annex table on the same or earlier pages.
// Authored by: OpenCode
func contractPDFConversionSourceRun(inspection testutil.GeneratedPDF, sourceID string, sectionPage int, sectionY float64) (testutil.PDFTextRun, bool) {
	for _, run := range inspection.TextRuns {
		if !contractPDFConversionRunInSection(run, sectionPage, sectionY) || run.Text != sourceID {
			continue
		}
		return run, true
	}
	return testutil.PDFTextRun{}, false
}

// contractPDFConversionSourceYs returns source-row baselines on the matching
// conversion page for nearest-row isolation.
// Authored by: OpenCode
func contractPDFConversionSourceYs(inspection testutil.GeneratedPDF, sourceIDs []string, sourceRun testutil.PDFTextRun, sectionPage int, sectionY float64) []float64 {
	var sourceYs []float64
	for _, run := range inspection.TextRuns {
		if run.Page != sourceRun.Page || !contractPDFConversionRunInSection(run, sectionPage, sectionY) || math.Abs(run.X-sourceRun.X) > 0.01 {
			continue
		}
		for _, sourceID := range sourceIDs {
			if run.Text == sourceID {
				sourceYs = append(sourceYs, run.Y)
				break
			}
		}
	}
	return sourceYs
}

// contractPDFConversionRunInSection reports whether a run belongs below the
// conversion heading and within the Annex conversion page range.
// Authored by: OpenCode
func contractPDFConversionRunInSection(run testutil.PDFTextRun, sectionPage int, sectionY float64) bool {
	return run.Page > sectionPage || run.Page == sectionPage && run.Y < sectionY
}

// contractPDFConversionRowNeighborhood returns the midpoint to the nearest
// source baseline, or one half of the fixed row height for a single row.
// Authored by: OpenCode
func contractPDFConversionRowNeighborhood(sourceY float64, sourceYs []float64) float64 {
	var neighborhood = 18.0
	for _, otherY := range sourceYs {
		if otherY == sourceY {
			continue
		}
		neighborhood = math.Min(neighborhood, math.Abs(sourceY-otherY)/2)
	}
	return neighborhood
}

// contractPDFRunTexts returns the decoded text fragments from ordered PDF runs.
// Authored by: OpenCode
func contractPDFRunTexts(runs []testutil.PDFTextRun) []string {
	var texts = make([]string, 0, len(runs))
	for _, run := range runs {
		texts = append(texts, run.Text)
	}
	return texts
}

// contractPDFEntryStartRuns finds each expected logical label in order and
// returns its physical run so coordinate boundaries can be asserted.
// Authored by: OpenCode
func contractPDFEntryStartRuns(runs []testutil.PDFTextRun, kinds []reportmodel.ConvertedAmountKind) ([]testutil.PDFTextRun, bool) {
	var texts = contractPDFRunTexts(runs)
	var joined strings.Builder
	var starts []int
	var ends []int
	for index, text := range texts {
		if index > 0 {
			joined.WriteByte(' ')
		}
		starts = append(starts, joined.Len())
		joined.WriteString(text)
		ends = append(ends, joined.Len())
	}
	var source = joined.String()
	var searchFrom int
	var result []testutil.PDFTextRun
	for _, kind := range kinds {
		var label = string(kind) + ":"
		var relative = strings.Index(source[searchFrom:], label)
		if relative < 0 {
			return nil, false
		}
		var position = searchFrom + relative
		var runIndex = -1
		for index := range starts {
			if position >= starts[index] && position < ends[index] {
				runIndex = index
				break
			}
		}
		if runIndex < 0 {
			return nil, false
		}
		result = append(result, runs[runIndex])
		searchFrom = position + len(label)
	}
	return result, true
}

// contractConvertedAmountKinds lists all supported kinds for empty-cell
// negative checks without adding model validation to this contract fixture.
// Authored by: OpenCode
func contractConvertedAmountKinds() []reportmodel.ConvertedAmountKind {
	return []reportmodel.ConvertedAmountKind{
		reportmodel.ConvertedAmountKindUnitPrice,
		reportmodel.ConvertedAmountKindGrossValue,
		reportmodel.ConvertedAmountKindFeeAmount,
	}
}
