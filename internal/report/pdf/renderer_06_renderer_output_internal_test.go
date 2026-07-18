package pdf

import (
	"errors"
	"strings"
	"testing"
	"time"

	reportcalculate "github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// TestRendererEmitsLandscapeA4SearchableSharedReportValues verifies real PDF
// output preserves report values rendered to Markdown from the same protected
// activity cache.
// Authored by: OpenCode
func TestRendererEmitsLandscapeA4SearchableSharedReportValues(t *testing.T) {
	var fixture = testutil.DeterministicReportLedgerFixture()
	for index := range fixture.ProtectedActivityCache.Activities {
		fixture.ProtectedActivityCache.Activities[index].OrderCurrency = "USD"
		fixture.ProtectedActivityCache.Activities[index].AssetProfileCurrency = "USD"
		fixture.ProtectedActivityCache.Activities[index].BaseCurrency = "USD"
	}
	var request, err = reportmodel.NewReportRequest(
		fixture.PrimaryReportYear,
		reportmodel.CostBasisMethodFIFO,
		reportmodel.ReportBaseCurrencyUSD,
		reportmodel.ReportOutputFormatPDF,
		time.Date(2026, time.May, 21, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("create report request: %v", err)
	}
	var report reportmodel.CapitalGainsReport
	report, err = reportcalculate.Calculate(request, fixture.ProtectedActivityCache)
	if err != nil {
		t.Fatalf("calculate report from protected cache: %v", err)
	}
	var markdownDocument reportmodel.ReportDocument
	markdownDocument, err = reportmarkdown.Render(report)
	if err != nil {
		t.Fatalf("render Markdown from protected cache: %v", err)
	}
	var renderer Renderer
	renderer, err = NewRenderer(RenderOptions{Fonts: FontData{Regular: goregular.TTF, Bold: gobold.TTF}})
	if err != nil {
		t.Fatalf("create PDF renderer: %v", err)
	}
	var payload []byte
	payload, err = renderer.Render(report)
	if err != nil {
		t.Fatalf("render PDF from protected cache: %v", err)
	}
	var inspection testutil.GeneratedPDF
	inspection, err = testutil.InspectGeneratedPDF(payload)
	if err != nil {
		t.Fatalf("inspect rendered PDF: %v", err)
	}
	for index, page := range inspection.PageBoxes {
		if page.Width != 842 || page.Height != 595 {
			t.Fatalf("page %d dimensions = %.0fx%.0f, want landscape A4 842x595", index+1, page.Width, page.Height)
		}
	}
	for _, sharedValue := range []string{"Ghostfolio Capital Gains And Losses Report", "Gains-And-Losses Summary", "Overall Yearly Net Total", "ADA", "Same currency"} {
		if !strings.Contains(string(markdownDocument.Content), sharedValue) {
			t.Fatalf("expected Markdown to contain shared value %q, got %q", sharedValue, markdownDocument.Content)
		}
		if !inspection.ContainsSearchableText(sharedValue) {
			t.Fatalf("expected searchable PDF to contain shared value %q, got %q", sharedValue, inspection.SearchableText)
		}
	}
}

// TestRendererSeamErrorBranches verifies nil seam guards and wrapped failures
// before concrete gopdf rendering is involved.
// Authored by: OpenCode
func TestRendererSeamErrorBranches(t *testing.T) {
	assertErrorContains(t, func() error { return startPDFDocument(nil) }, "starter is required")
	assertErrorContains(t, func() error { return startPDFDocument(&failingPDFStartRecorder{}) }, "start failed")
	assertErrorContains(t, func() error { return loadApplicationFonts(nil, FontData{Regular: []byte("r"), Bold: []byte("b")}) }, "font loader is required")
	assertErrorContains(t, func() error {
		return loadApplicationFonts(&failingFontLoader{failName: fontRegular}, FontData{Regular: []byte("r"), Bold: []byte("b")})
	}, "load regular font")
	assertErrorContains(t, func() error {
		return loadApplicationFonts(&failingFontLoader{failName: fontBold}, FontData{Regular: []byte("r"), Bold: []byte("b")})
	}, "load bold font")
	assertErrorContains(t, func() error { return renderMainReport(nil, minimalPDFReportFixture(t)) }, "layout document is required")
	assertErrorContains(t, func() error { return renderMainReport(&layoutRecorder{}, reportmodel.CapitalGainsReport{}) }, "capital gains report year must be greater than zero")
	assertErrorContains(t, func() error { return renderAnnex(nil, reportmodel.DefaultAuditAnnex()) }, "layout document is required")
	assertErrorContains(t, func() error { return renderAnnex(&layoutRecorder{}, reportmodel.AuditAnnex{Title: "bad"}) }, "audit annex title")

	var previousDocument = newPDFDocumentForRenderer
	defer func() { newPDFDocumentForRenderer = previousDocument }()
	var renderer, rendererErr = NewRenderer(RenderOptions{Fonts: FontData{Regular: []byte("r"), Bold: []byte("b")}})
	if rendererErr != nil {
		t.Fatalf("new renderer: %v", rendererErr)
	}

	newPDFDocumentForRenderer = func(ByteFinalizer) pdfLayoutDocument {
		return &failingLayoutDocument{startErr: errors.New("start failed")}
	}
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "start failed")
	newPDFDocumentForRenderer = func(ByteFinalizer) pdfLayoutDocument {
		return &failingLayoutDocument{fontErr: errors.New("font failed")}
	}
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "font failed")
	newPDFDocumentForRenderer = func(ByteFinalizer) pdfLayoutDocument {
		return &failingLayoutDocument{titleErr: errors.New("title failed")}
	}
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "title failed")
	newPDFDocumentForRenderer = func(ByteFinalizer) pdfLayoutDocument {
		return &failingLayoutDocument{pageBreakErr: errors.New("page break failed")}
	}
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "page break failed")
	newPDFDocumentForRenderer = func(ByteFinalizer) pdfLayoutDocument { return &secondTitleFailDocument{} }
	assertErrorContains(t, func() error { _, err := renderer.Render(minimalPDFReportFixture(t)); return err }, "annex title failed")
}
