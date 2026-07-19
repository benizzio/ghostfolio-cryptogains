package pdf

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// TestGopdfDocumentBoldParagraphGuardBranches verifies bold paragraph startup,
// font, fit, and drawing failures through the concrete adapter seams.
// Authored by: OpenCode
func TestGopdfDocumentBoldParagraphGuardBranches(t *testing.T) {
	assertErrorContains(t, func() error {
		return newGopdfDocument().AddBoldParagraph("not started")
	}, "before adding content")

	var noBoldDocument = newGopdfDocument()
	if err := noBoldDocument.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start no-bold document: %v", err)
	}
	assertErrorContains(t, func() error { return noBoldDocument.AddBoldParagraph("missing font") }, "font")

	var tooTallDocument = startedTestDocument(t)
	assertErrorContains(t, func() error {
		return tooTallDocument.AddBoldParagraph(strings.Repeat("too tall bold paragraph ", 4000))
	}, "does not fit within the printable page area")

	var previousWriter = writeMultiCellForGopdfDocument
	var previousFit = fitMultiCellForGopdfDocument
	defer func() {
		writeMultiCellForGopdfDocument = previousWriter
		fitMultiCellForGopdfDocument = previousFit
	}()
	writeMultiCellForGopdfDocument = func(*gopdfDocument, *gopdf.Rect, string) error {
		return errors.New("bold paragraph drawing failed")
	}
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddBoldParagraph("draw failure")
	}, "bold paragraph drawing failed")

	fitMultiCellForGopdfDocument = func(*gopdfDocument, *gopdf.Rect, string) (bool, float64, error) {
		return false, 0, errors.New("bold paragraph measurement failed")
	}
	assertErrorContains(t, func() error {
		return startedTestDocument(t).AddBoldParagraph("measurement failure")
	}, "bold paragraph measurement failed")
}

// TestGopdfDocumentBoldParagraphRechecksDocumentState verifies the measured
// paragraph path still returns a startup error if a font callback invalidates
// the document between measurement and drawing.
// Authored by: OpenCode
func TestGopdfDocumentBoldParagraphRechecksDocumentState(t *testing.T) {
	var document = newGopdfDocument()
	if err := document.StartPDF(PageSizeA4); err != nil {
		t.Fatalf("start callback document: %v", err)
	}
	var option = gopdf.TtfOption{
		OnGlyphNotFound:           func(rune) { document.started = false },
		OnGlyphNotFoundSubstitute: gopdf.DefaultOnGlyphNotFoundSubstitute,
	}
	if err := document.pdf.AddTTFFontByReaderWithOption(fontRegular, bytes.NewReader(goregular.TTF), option); err != nil {
		t.Fatalf("load callback regular font: %v", err)
	}
	if err := document.pdf.AddTTFFontByReaderWithOption(fontBold, bytes.NewReader(gobold.TTF), option); err != nil {
		t.Fatalf("load callback bold font: %v", err)
	}

	assertErrorContains(t, func() error {
		return document.AddBoldParagraph(string(rune(0x10ffff)))
	}, "before adding content")
}
