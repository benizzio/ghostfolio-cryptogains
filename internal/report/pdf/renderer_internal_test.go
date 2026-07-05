// Package pdf tests the private seams required for local A4 PDF rendering.
// Authored by: OpenCode
package pdf

import (
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/cockroachdb/apd/v3"
)

// TestStartPDFDocumentUsesA4Configuration specifies the renderer's page-size
// seam so the implementation can prove that every generated PDF starts with A4
// configuration instead of a library default.
// Authored by: OpenCode
func TestStartPDFDocumentUsesA4Configuration(t *testing.T) {
	var recorder = &pdfStartRecorder{}

	var err = startPDFDocument(recorder)
	if err != nil {
		t.Fatalf("start PDF document: %v", err)
	}

	if recorder.pageSize != PageSizeA4 {
		t.Fatalf("page size = %q, want %q", recorder.pageSize, PageSizeA4)
	}
	if recorder.startCount != 1 {
		t.Fatalf("start count = %d, want 1", recorder.startCount)
	}
}

// TestLoadApplicationFontsValidatesAndLoadsRegularAndBoldFonts specifies the
// application-supplied font seam. The renderer must reject missing font bytes
// and load both regular and bold fonts without reading platform font paths.
// Authored by: OpenCode
func TestLoadApplicationFontsValidatesAndLoadsRegularAndBoldFonts(t *testing.T) {
	var recorder = &fontLoadRecorder{}
	var fonts = FontData{
		Regular: []byte("regular-ttf-bytes"),
		Bold:    []byte("bold-ttf-bytes"),
	}

	var err = loadApplicationFonts(recorder, fonts)
	if err != nil {
		t.Fatalf("load application fonts: %v", err)
	}

	assertLoadedFont(t, recorder, "regular", fonts.Regular)
	assertLoadedFont(t, recorder, "bold", fonts.Bold)

	var missingRegularErr = loadApplicationFonts(&fontLoadRecorder{}, FontData{Bold: fonts.Bold})
	if missingRegularErr == nil || !strings.Contains(missingRegularErr.Error(), "regular font data") {
		t.Fatalf("missing regular font error = %v, want regular font data validation", missingRegularErr)
	}

	var missingBoldErr = loadApplicationFonts(&fontLoadRecorder{}, FontData{Regular: fonts.Regular})
	if missingBoldErr == nil || !strings.Contains(missingBoldErr.Error(), "bold font data") {
		t.Fatalf("missing bold font error = %v, want bold font data validation", missingBoldErr)
	}
}

// TestEmitMainAndAnnexShellWritesRequiredSelectableText specifies the first PDF
// text-emission seam for the user story. The renderer must emit the main title
// and Annex 1 shell as text calls, with Annex 1 after a page break.
// Authored by: OpenCode
func TestEmitMainAndAnnexShellWritesRequiredSelectableText(t *testing.T) {
	var recorder = &textEmissionRecorder{}
	var report = minimalPDFReportFixture(t)

	var err = emitMainAndAnnexShell(recorder, report)
	if err != nil {
		t.Fatalf("emit main and annex shell: %v", err)
	}

	assertTextEmitted(t, recorder.texts, MainReportTitle)
	assertTextEmitted(t, recorder.texts, AnnexTitle)
	if recorder.annexPageBreaks < 1 {
		t.Fatalf("annex page breaks = %d, want at least 1", recorder.annexPageBreaks)
	}
}

// pdfStartRecorder records the page-size intent passed through the renderer's
// PDF document start seam.
// Authored by: OpenCode
type pdfStartRecorder struct {
	pageSize   string
	startCount int
}

// StartPDF records a PDF start request.
// Authored by: OpenCode
func (recorder *pdfStartRecorder) StartPDF(pageSize string) error {
	recorder.pageSize = pageSize
	recorder.startCount++

	return nil
}

// fontLoadRecorder records application-supplied font loads.
// Authored by: OpenCode
type fontLoadRecorder struct {
	loaded map[string][]byte
}

// AddTTFFont records one font registration by logical font name.
// Authored by: OpenCode
func (recorder *fontLoadRecorder) AddTTFFont(name string, data []byte) error {
	if recorder.loaded == nil {
		recorder.loaded = make(map[string][]byte)
	}
	recorder.loaded[name] = append([]byte(nil), data...)

	return nil
}

// textEmissionRecorder records text and page-break calls for the initial PDF
// report shell.
// Authored by: OpenCode
type textEmissionRecorder struct {
	texts           []string
	annexPageBreaks int
}

// AddText records selectable text emission.
// Authored by: OpenCode
func (recorder *textEmissionRecorder) AddText(text string) error {
	recorder.texts = append(recorder.texts, text)

	return nil
}

// AddAnnexPageBreak records the required page break before Annex 1.
// Authored by: OpenCode
func (recorder *textEmissionRecorder) AddAnnexPageBreak() error {
	recorder.annexPageBreaks++

	return nil
}

// assertLoadedFont verifies that a named font was loaded with the expected
// application-supplied bytes.
// Authored by: OpenCode
func assertLoadedFont(t *testing.T, recorder *fontLoadRecorder, name string, want []byte) {
	t.Helper()

	var got, ok = recorder.loaded[name]
	if !ok {
		t.Fatalf("font %q was not loaded", name)
	}
	if string(got) != string(want) {
		t.Fatalf("font %q bytes = %q, want %q", name, got, want)
	}
}

// assertTextEmitted verifies that a required text fragment was emitted through
// the selectable text seam.
// Authored by: OpenCode
func assertTextEmitted(t *testing.T, texts []string, want string) {
	t.Helper()

	for _, text := range texts {
		if strings.Contains(text, want) {
			return
		}
	}

	t.Fatalf("required text %q was not emitted in %q", want, texts)
}

// minimalPDFReportFixture creates a validated report containing only the fields
// required by the initial PDF shell tests.
// Authored by: OpenCode
func minimalPDFReportFixture(t *testing.T) reportmodel.CapitalGainsReport {
	t.Helper()

	var requestedAt = time.Date(2026, time.July, 5, 9, 0, 0, 0, time.UTC)
	var request, requestErr = reportmodel.NewReportRequest(
		2024,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatPDF,
		requestedAt,
	)
	if requestErr != nil {
		t.Fatalf("new report request: %v", requestErr)
	}

	var report, reportErr = reportmodel.NewCapitalGainsReport(
		request,
		requestedAt,
		reportmodel.ReportBaseCurrencyUSD.Label(),
		nil,
		*apd.New(0, 0),
		nil,
		nil,
	)
	if reportErr != nil {
		t.Fatalf("new capital gains report: %v", reportErr)
	}

	return report
}
