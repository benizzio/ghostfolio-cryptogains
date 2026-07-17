package testutil

import (
	"strings"
	"testing"
)

// TestDecodePDFTextDecodesMultiByteGoFontDigitOne verifies embedded Go font
// ToUnicode source codes preserve the Annex 1 title's digit.
// Authored by: OpenCode
func TestDecodePDFTextDecodesMultiByteGoFontDigitOne(t *testing.T) {
	var decoded = decodePDFText([]byte("0014"), nil)
	if decoded != "1" {
		t.Fatalf("decoded text = %q, want %q", decoded, "1")
	}
}

// TestDecodePDFTextWithToUnicodeMapPreservesCIDSpaces verifies real gopdf CID
// mappings retain spaces between words in ordered text runs.
// Authored by: OpenCode
func TestDecodePDFTextWithToUnicodeMapPreservesCIDSpaces(t *testing.T) {
	var decoded = decodePDFTextWithMaps([]byte("002A0003004B"), map[string]string{"002A": "G", "0003": " ", "004B": "h"}, nil, nil)
	if decoded != "G h" {
		t.Fatalf("decoded text = %q, want %q", decoded, "G h")
	}
}

// TestInspectGeneratedPDFExtractsOrderedTextRuns verifies page order, decoded
// text, font resources, and PDF-coordinate recovery from content streams.
// Authored by: OpenCode
func TestInspectGeneratedPDFExtractsOrderedTextRuns(t *testing.T) {
	var inspection, err = InspectGeneratedPDF([]byte(orderedTextRunPDF()))
	if err != nil {
		t.Fatalf("inspect synthetic PDF: %v", err)
	}
	if len(inspection.PageBoxes) != 2 {
		t.Fatalf("page boxes = %d, want 2", len(inspection.PageBoxes))
	}
	if inspection.PageBoxes[0] != (PDFPageBox{Width: 842, Height: 595}) || inspection.PageBoxes[1] != (PDFPageBox{Width: 842, Height: 595}) {
		t.Fatalf("page boxes = %#v, want two landscape A4 boxes", inspection.PageBoxes)
	}
	if !inspection.ContainsSearchableText("A") || !inspection.ContainsSearchableText("B") {
		t.Fatalf("searchable text = %q", inspection.SearchableText)
	}
	var want = []PDFTextRun{
		{Page: 1, Text: "A", FontResource: "F1", X: 72, Y: 500},
		{Page: 1, Text: "second page", FontResource: "F2", X: 72, Y: 480},
		{Page: 2, Text: "B", FontResource: "F1", X: 10, Y: 100},
	}
	if len(inspection.TextRuns) != len(want) {
		t.Fatalf("text runs = %#v, want %#v", inspection.TextRuns, want)
	}
	for index, run := range inspection.TextRuns {
		if run != want[index] {
			t.Fatalf("text run %d = %#v, want %#v", index, run, want[index])
		}
	}
}

// TestInspectGeneratedPDFReportsParsingErrors verifies the existing inspector
// errors remain explicit for invalid headers, missing pages or text, and bad streams.
// Authored by: OpenCode
func TestInspectGeneratedPDFReportsParsingErrors(t *testing.T) {
	var cases = []struct {
		name    string
		payload string
		wantErr string
	}{
		{name: "header", payload: "not a PDF", wantErr: "PDF header is required"},
		{name: "object number", payload: "%PDF-1.7\n999999999999999999999999 0 obj << >> endobj", wantErr: "parse PDF object number"},
		{name: "pages", payload: "%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj", wantErr: "PDF pages are required"},
		{name: "searchable text", payload: pageWithoutTextPDF(), wantErr: "searchable PDF text is required"},
		{name: "unterminated stream", payload: unterminatedStreamPDF(), wantErr: "unterminated PDF stream"},
		{name: "invalid Flate stream", payload: invalidFlateStreamPDF(), wantErr: "open PDF Flate stream"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := InspectGeneratedPDF([]byte(testCase.payload))
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, testCase.wantErr)
			}
		})
	}
}

// orderedTextRunPDF returns a deterministic two-page PDF with two font resources.
// Authored by: OpenCode
func orderedTextRunPDF() string {
	return strings.Join([]string{
		"%PDF-1.7",
		"1 0 obj << /Type /Page /MediaBox [0 0 842 595] /Contents 2 0 R /Resources 3 0 R >> endobj",
		"2 0 obj << /Length 1 >> stream\nBT\n/F1 10 Tf\n72 500 TD\n<01> Tj\n/F2 10 Tf\n72 480 TD\n[(second) 10 ( page)] TJ\nET\nendstream endobj",
		"3 0 obj << /Font << /F1 4 0 R /F2 5 0 R >> >> endobj",
		"4 0 obj << /Type /Font /ToUnicode 6 0 R >> endobj",
		"5 0 obj << /Type /Font >> endobj",
		"6 0 obj << /Length 1 >> stream\nbeginbfchar\n<01> <0041>\n<02> <0042>\nendbfchar\nendstream endobj",
		"7 0 obj << /Type /Page /MediaBox [0 0 842 595] /Contents 8 0 R /Resources 3 0 R >> endobj",
		"8 0 obj << /Length 1 >> stream\nBT\n/F1 10 Tf\n1 0 0 1 10 100 Tm\n<02> Tj\nET\nendstream endobj",
	}, "\n")
}

// pageWithoutTextPDF returns a valid page object without searchable content.
// Authored by: OpenCode
func pageWithoutTextPDF() string {
	return "%PDF-1.7\n1 0 obj << /Type /Page /MediaBox [0 0 842 595] >> endobj"
}

// unterminatedStreamPDF returns a page followed by a stream missing endstream.
// Authored by: OpenCode
func unterminatedStreamPDF() string {
	return strings.Join([]string{
		"%PDF-1.7",
		"1 0 obj << /Type /Page /MediaBox [0 0 842 595] >> endobj",
		"2 0 obj << >> stream\nBT\nendobj",
	}, "\n")
}

// invalidFlateStreamPDF returns a Flate-marked stream with invalid compressed data.
// Authored by: OpenCode
func invalidFlateStreamPDF() string {
	return strings.Join([]string{
		"%PDF-1.7",
		"1 0 obj << /Type /Page /MediaBox [0 0 842 595] >> endobj",
		"2 0 obj << /Filter /FlateDecode >> stream\nnot compressed\nendstream endobj",
	}, "\n")
}
