// Package testutil contains deterministic helpers for local report tests.
// Authored by: OpenCode
package testutil

import (
	"bytes"
	"fmt"
	"strings"
)

// GeneratedPDF describes the searchable text and page geometry recovered from
// a locally generated PDF. It is intentionally a small inspection model for
// tests, not a general-purpose PDF reader.
//
// Example:
//
//	inspection, err := testutil.InspectGeneratedPDF(payload)
//	if err != nil { panic(err) }
//	_ = inspection.SearchableText
//
// Authored by: OpenCode
type GeneratedPDF struct {
	PageBoxes      []PDFPageBox
	SearchableText string
	TextRuns       []PDFTextRun
}

// PDFTextRun describes one ordered text-showing operation recovered from a
// generated PDF. Coordinates use the PDF bottom-left origin and the font
// resource omits its leading slash, for example "F1".
// Authored by: OpenCode
type PDFTextRun struct {
	Page         int
	Text         string
	FontResource string
	X            float64
	Y            float64
}

// PDFPageBox describes one PDF page MediaBox in PostScript points.
// Authored by: OpenCode
type PDFPageBox struct {
	Width  float64
	Height float64
}

// ContainsSearchableText reports whether extracted PDF text contains a report
// value after case and layout whitespace normalization. This accommodates PDF
// renderers that split visible words across adjacent text-showing operators.
//
// Example:
//
//	if !inspection.ContainsSearchableText("Annex 1 - Audit") { panic("missing annex") }
//
// Authored by: OpenCode
func (inspection GeneratedPDF) ContainsSearchableText(value string) bool {
	return strings.Contains(normalizePDFSearchText(inspection.SearchableText), normalizePDFSearchText(value))
}

// InspectGeneratedPDF parses the page MediaBoxes and text-showing content of a
// generated PDF. It supports ordinary and Flate-compressed object streams plus
// embedded-font ToUnicode maps, which is sufficient for the project's local
// gopdf output without depending on a system PDF binary.
//
// Example:
//
//	inspection, err := testutil.InspectGeneratedPDF(payload)
//	if err != nil { panic(err) }
//	if inspection.PageBoxes[0].Width <= inspection.PageBoxes[0].Height { panic("portrait") }
//
// Authored by: OpenCode
func InspectGeneratedPDF(payload []byte) (GeneratedPDF, error) {
	if !bytes.HasPrefix(payload, []byte("%PDF-")) {
		return GeneratedPDF{}, fmt.Errorf("PDF header is required")
	}

	var content, err = inspectPDFObjects(payload)
	if err != nil {
		return GeneratedPDF{}, err
	}
	var inspection = GeneratedPDF{PageBoxes: content.resolvedPageBoxes()}
	if len(inspection.PageBoxes) == 0 {
		return GeneratedPDF{}, fmt.Errorf("PDF pages are required")
	}
	inspection.TextRuns, err = content.resolvedTextRuns()
	if err != nil {
		return GeneratedPDF{}, err
	}
	inspection.SearchableText = extractPDFText(content.textStreams, content.unicodeMaps, content.glyphMaps)
	if inspection.SearchableText == "" {
		return GeneratedPDF{}, fmt.Errorf("searchable PDF text is required")
	}

	return inspection, nil
}

// extractPDFText decodes text-showing operands from the collected content streams.
// Authored by: OpenCode
func extractPDFText(textStreams [][]byte, unicodeMaps []map[string]string, glyphMaps []map[byte]string) string {
	var texts []string
	for _, stream := range textStreams {
		texts = append(texts, extractPDFStreamText(stream, unicodeMaps, glyphMaps)...)
	}
	return strings.Join(texts, "\n")
}

// extractPDFStreamText decodes the text-showing operands from one content stream.
// Authored by: OpenCode
func extractPDFStreamText(stream []byte, unicodeMaps []map[string]string, glyphMaps []map[byte]string) []string {
	var texts []string
	for _, match := range pdfLiteralPattern.FindAllSubmatch(stream, -1) {
		texts = append(texts, unescapePDFLiteral(match[1]))
	}
	for _, match := range pdfHexTextPattern.FindAllSubmatch(stream, -1) {
		texts = append(texts, decodePDFTextOperand(match, unicodeMaps, glyphMaps)...)
	}
	return texts
}

// decodePDFTextOperand decodes one hexadecimal text operand through all embedded maps.
// Authored by: OpenCode
func decodePDFTextOperand(match [][]byte, unicodeMaps []map[string]string, glyphMaps []map[byte]string) []string {
	var encoded = match[1]
	if len(encoded) == 0 {
		encoded = match[2]
	}
	var texts []string
	for _, unicodeMap := range unicodeMaps {
		if decoded := decodePDFTextWithMaps(encoded, unicodeMap, nil, nil); decoded != "" {
			texts = append(texts, decoded)
		}
	}
	for _, glyphMap := range glyphMaps {
		if decoded := decodePDFGlyphText(encoded, glyphMap); decoded != "" {
			texts = append(texts, decoded)
		}
	}
	return texts
}

// normalizePDFSearchText removes PDF layout separators from searchable text.
// Authored by: OpenCode
func normalizePDFSearchText(value string) string {
	var normalized strings.Builder
	for _, character := range strings.ToUpper(value) {
		if (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') {
			normalized.WriteRune(character)
		}
	}
	return normalized.String()
}
