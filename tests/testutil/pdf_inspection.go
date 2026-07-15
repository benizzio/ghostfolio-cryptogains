package testutil

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
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

var (
	pdfObjectPattern   = regexp.MustCompile(`(?s)(\d+)\s+\d+\s+obj\b(.*?)\bendobj\b`)
	pdfPagePattern     = regexp.MustCompile(`/Type\s*/Page\b`)
	pdfMediaBoxPattern = regexp.MustCompile(`/MediaBox\s*\[\s*([-+0-9.]+)\s+([-+0-9.]+)\s+([-+0-9.]+)\s+([-+0-9.]+)\s*\]`)
	pdfHexTextPattern  = regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*Tj\b|<([0-9A-Fa-f]+)>`)
	pdfLiteralPattern  = regexp.MustCompile(`\((([^\\()]|\\.)*)\)\s*Tj\b`)
	pdfCMapPattern     = regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
	pdfBFCharPattern   = regexp.MustCompile(`(?s)beginbfchar\s*(.*?)\s*endbfchar`)
	pdfBFRangePattern  = regexp.MustCompile(`(?s)beginbfrange\s*(.*?)\s*endbfrange`)
	pdfRangeEntry      = regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
)

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
	inspection.SearchableText = extractPDFText(content.textStreams, content.unicodeMaps, content.glyphMaps)
	if inspection.SearchableText == "" {
		return GeneratedPDF{}, fmt.Errorf("searchable PDF text is required")
	}

	return inspection, nil
}

// pdfInspectionContent accumulates the inspectable PDF object content.
// Authored by: OpenCode
type pdfInspectionContent struct {
	unicodeMaps       []map[string]string
	glyphMaps         []map[byte]string
	textStreams       [][]byte
	explicitPageBoxes []PDFPageBox
	inheritedPageBox  PDFPageBox
	pageCount         int
}

// resolvedPageBoxes returns explicit boxes or the inherited page-tree box for each page.
// Authored by: OpenCode
func (content pdfInspectionContent) resolvedPageBoxes() []PDFPageBox {
	if len(content.explicitPageBoxes) > 0 || content.pageCount == 0 || content.inheritedPageBox.Width <= 0 || content.inheritedPageBox.Height <= 0 {
		return content.explicitPageBoxes
	}
	var boxes = make([]PDFPageBox, content.pageCount)
	for index := range boxes {
		boxes[index] = content.inheritedPageBox
	}
	return boxes
}

// inspectPDFObjects recovers page geometry, text streams, and font maps.
// Authored by: OpenCode
func inspectPDFObjects(payload []byte) (pdfInspectionContent, error) {
	var content pdfInspectionContent
	for _, match := range pdfObjectPattern.FindAllSubmatch(payload, -1) {
		var object = match[2]
		if err := content.inspectObject(object); err != nil {
			return pdfInspectionContent{}, err
		}
	}
	return content, nil
}

// inspectObject recovers inspection data from one indirect PDF object.
// Authored by: OpenCode
func (content *pdfInspectionContent) inspectObject(object []byte) error {
	content.inspectPageBox(object)
	var stream, ok, err = pdfObjectStream(object)
	if err != nil || !ok {
		return err
	}
	content.inspectStream(stream)
	return nil
}

// inspectPageBox tracks explicit and inherited MediaBox page geometry.
// Authored by: OpenCode
func (content *pdfInspectionContent) inspectPageBox(object []byte) {
	var mediaBox = pdfMediaBoxPattern.FindSubmatch(object)
	if len(mediaBox) == 5 {
		content.inheritedPageBox = PDFPageBox{
			Width:  parsePDFNumber(mediaBox[3]) - parsePDFNumber(mediaBox[1]),
			Height: parsePDFNumber(mediaBox[4]) - parsePDFNumber(mediaBox[2]),
		}
	}
	if !pdfPagePattern.Match(object) {
		return
	}
	content.pageCount++
	if len(mediaBox) == 5 {
		content.explicitPageBoxes = append(content.explicitPageBoxes, content.inheritedPageBox)
	}
}

// inspectStream classifies a decoded object stream for text recovery.
// Authored by: OpenCode
func (content *pdfInspectionContent) inspectStream(stream []byte) {
	if glyphMap := embeddedFontGlyphMap(stream); len(glyphMap) > 0 {
		content.glyphMaps = append(content.glyphMaps, glyphMap)
	}
	if unicodeMap := pdfUnicodeMap(stream); len(unicodeMap) > 0 {
		content.unicodeMaps = append(content.unicodeMaps, unicodeMap)
		return
	}
	if bytes.Contains(stream, []byte(" Tj")) || bytes.Contains(stream, []byte(" TJ")) {
		content.textStreams = append(content.textStreams, stream)
	}
}

// pdfUnicodeMap recovers direct and ranged ToUnicode mappings from one stream.
// Authored by: OpenCode
func pdfUnicodeMap(stream []byte) map[string]string {
	if !bytes.Contains(stream, []byte("beginbfchar")) && !bytes.Contains(stream, []byte("beginbfrange")) {
		return nil
	}
	var unicodeMap = make(map[string]string)
	for _, block := range pdfBFCharPattern.FindAllSubmatch(stream, -1) {
		for _, mapping := range pdfCMapPattern.FindAllSubmatch(block[1], -1) {
			var decoded = decodePDFUnicode(mapping[2])
			if decoded != "" {
				unicodeMap[strings.ToUpper(string(mapping[1]))] = decoded
			}
		}
	}
	for _, block := range pdfBFRangePattern.FindAllSubmatch(stream, -1) {
		for _, entry := range pdfRangeEntry.FindAllSubmatch(block[1], -1) {
			addPDFUnicodeRange(unicodeMap, entry[1], entry[2], entry[3])
		}
	}
	return unicodeMap
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
		if decoded := decodePDFText(encoded, unicodeMap); decoded != "" {
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

// embeddedFontGlyphMap recovers ASCII glyph IDs from an embedded TrueType font.
// Authored by: OpenCode
func embeddedFontGlyphMap(stream []byte) map[byte]string {
	if len(stream) < 4 || !bytes.Equal(stream[:4], []byte{0x00, 0x01, 0x00, 0x00}) {
		return nil
	}
	var cmap, ok = parseTrueTypeCMapFormat4(stream)
	if !ok {
		return nil
	}
	var glyphMap = make(map[byte]string)
	for character := rune(32); character <= 126; character++ {
		var glyph, found = cmap.glyphFor(character)
		if found && glyph >= 0 && glyph <= 255 {
			glyphMap[byte(glyph)] = string(character)
		}
	}
	return glyphMap
}

// trueTypeCMapFormat4 holds the offsets required to resolve a format 4 cmap.
// Authored by: OpenCode
type trueTypeCMapFormat4 struct {
	font         []byte
	segmentCount int
	endCodes     int
	startCodes   int
	deltas       int
}

// parseTrueTypeCMapFormat4 locates and validates an embedded format 4 cmap table.
// Authored by: OpenCode
func parseTrueTypeCMapFormat4(font []byte) (trueTypeCMapFormat4, bool) {
	var cmapOffset, ok = trueTypeTableOffset(font, "cmap")
	if !ok || cmapOffset+12 > len(font) {
		return trueTypeCMapFormat4{}, false
	}
	var subtable = cmapOffset + int(pdfUint32(font[cmapOffset+8:cmapOffset+12]))
	if subtable+16 > len(font) || pdfUint16(font[subtable:subtable+2]) != 4 {
		return trueTypeCMapFormat4{}, false
	}
	var segmentCount = int(pdfUint16(font[subtable+6:subtable+8]) / 2)
	var endCodes = subtable + 14
	var startCodes = endCodes + segmentCount*2 + 2
	var deltas = startCodes + segmentCount*2
	if deltas+segmentCount*2 > len(font) {
		return trueTypeCMapFormat4{}, false
	}
	return trueTypeCMapFormat4{font: font, segmentCount: segmentCount, endCodes: endCodes, startCodes: startCodes, deltas: deltas}, true
}

// glyphFor returns the format 4 glyph ID mapped to one character.
// Authored by: OpenCode
func (cmap trueTypeCMapFormat4) glyphFor(character rune) (int, bool) {
	for segment := 0; segment < cmap.segmentCount; segment++ {
		var start = rune(pdfUint16(cmap.font[cmap.startCodes+segment*2 : cmap.startCodes+segment*2+2]))
		var end = rune(pdfUint16(cmap.font[cmap.endCodes+segment*2 : cmap.endCodes+segment*2+2]))
		if character < start || character > end {
			continue
		}
		var delta = signedTrueTypeInt16(pdfUint16(cmap.font[cmap.deltas+segment*2 : cmap.deltas+segment*2+2]))
		return int(character) + delta, true
	}
	return 0, false
}

// signedTrueTypeInt16 decodes a two's-complement TrueType delta without narrowing conversion.
// Authored by: OpenCode
func signedTrueTypeInt16(value uint16) int {
	var signed = int(value)
	if signed >= 1<<15 {
		signed -= 1 << 16
	}
	return signed
}

// trueTypeTableOffset locates one table directory record in an embedded TTF.
// Authored by: OpenCode
func trueTypeTableOffset(font []byte, tag string) (int, bool) {
	if len(font) < 12 {
		return 0, false
	}
	var tableCount = int(pdfUint16(font[4:6]))
	for index := 0; index < tableCount; index++ {
		var offset = 12 + index*16
		if offset+16 > len(font) || string(font[offset:offset+4]) != tag {
			continue
		}
		var tableOffset = int(pdfUint32(font[offset+8 : offset+12]))
		return tableOffset, tableOffset < len(font)
	}
	return 0, false
}

// pdfUint16 reads a big-endian unsigned 16-bit PDF or TrueType value.
// Authored by: OpenCode
func pdfUint16(raw []byte) uint16 {
	return uint16(raw[0])<<8 | uint16(raw[1])
}

// pdfUint32 reads a big-endian unsigned 32-bit PDF or TrueType value.
// Authored by: OpenCode
func pdfUint32(raw []byte) uint32 {
	return uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])
}

// addPDFUnicodeRange expands one sequential ToUnicode bfrange mapping.
// Authored by: OpenCode
func addPDFUnicodeRange(unicodeMap map[string]string, first []byte, last []byte, target []byte) {
	var firstValue, firstErr = strconv.ParseUint(string(first), 16, 64)
	var lastValue, lastErr = strconv.ParseUint(string(last), 16, 64)
	var targetValue, targetErr = strconv.ParseUint(string(target), 16, 64)
	if firstErr != nil || lastErr != nil || targetErr != nil || firstValue > lastValue {
		return
	}
	var width = len(first)
	for value := firstValue; value <= lastValue; value++ {
		var code = strings.ToUpper(fmt.Sprintf("%0*X", width, value))
		var character, ok = pdfUnicodeRangeCharacter(targetValue + value - firstValue)
		if !ok {
			return
		}
		unicodeMap[code] = string(character)
	}
}

// pdfUnicodeRangeCharacter validates one ToUnicode range target before conversion.
// Authored by: OpenCode
func pdfUnicodeRangeCharacter(value uint64) (rune, bool) {
	if value > unicode.MaxRune || value >= 0xD800 && value <= 0xDFFF {
		return 0, false
	}
	return rune(value), true
}

// pdfObjectStream returns one decoded object stream when present.
// Authored by: OpenCode
func pdfObjectStream(object []byte) ([]byte, bool, error) {
	var marker = []byte("stream")
	var start = bytes.Index(object, marker)
	if start < 0 {
		return nil, false, nil
	}
	start += len(marker)
	if bytes.HasPrefix(object[start:], []byte("\r\n")) {
		start += 2
	} else if bytes.HasPrefix(object[start:], []byte("\n")) || bytes.HasPrefix(object[start:], []byte("\r")) {
		start++
	}
	var end = bytes.LastIndex(object[start:], []byte("endstream"))
	if end < 0 {
		return nil, false, fmt.Errorf("unterminated PDF stream")
	}
	var raw = bytes.TrimRight(object[start:start+end], "\r\n")
	if !bytes.Contains(object[:start], []byte("/FlateDecode")) {
		return raw, true, nil
	}
	var reader, err = zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, false, fmt.Errorf("open PDF Flate stream: %w", err)
	}
	var decoded []byte
	decoded, err = io.ReadAll(reader)
	if err != nil {
		if closeErr := reader.Close(); closeErr != nil {
			return nil, false, fmt.Errorf("decode PDF Flate stream: %w; close PDF Flate stream: %w", err, closeErr)
		}
		return nil, false, fmt.Errorf("decode PDF Flate stream: %w", err)
	}
	if err = reader.Close(); err != nil {
		return nil, false, fmt.Errorf("close PDF Flate stream: %w", err)
	}
	return decoded, true, nil
}

// parsePDFNumber converts an already validated PDF numeric token.
// Authored by: OpenCode
func parsePDFNumber(raw []byte) float64 {
	var value, err = strconv.ParseFloat(string(raw), 64)
	if err != nil {
		panic(fmt.Sprintf("validated PDF number %q: %v", raw, err))
	}
	return value
}

// decodePDFText maps a hex text-showing operand through its embedded font map.
// Authored by: OpenCode
func decodePDFText(raw []byte, unicodeMap map[string]string) string {
	var encoded, err = hex.DecodeString(string(raw))
	if err != nil {
		return ""
	}
	var text strings.Builder
	for index := 0; index < len(encoded); {
		// gopdf writes the embedded Go font's digit 1 as CID 0x0014.
		if index+1 < len(encoded) && encoded[index] == 0 && encoded[index+1] == 20 {
			text.WriteString(decodeGoFontGlyph(string(encoded[index+1])))
			index += 2
			continue
		}
		var code = strings.ToUpper(hex.EncodeToString(encoded[index : index+1]))
		if value, ok := unicodeMap[code]; ok {
			text.WriteString(decodeGoFontGlyph(value))
		} else if encoded[index] >= 32 && encoded[index] <= 126 {
			text.WriteString(decodeGoFontGlyph(string(encoded[index])))
		}
		index++
	}
	return text.String()
}

// decodeGoFontGlyph translates the embedded Go font's glyph IDs after gopdf's
// identity ToUnicode mapping. The renderer uses these application-supplied
// fonts, whose ASCII glyph IDs are stable in the generated PDF subset.
// Authored by: OpenCode
func decodeGoFontGlyph(value string) string {
	if len(value) != 1 {
		return value
	}
	var glyph = value[0]
	switch {
	case glyph == 3:
		return " "
	case glyph >= 19 && glyph <= 28:
		return string(rune('0' + glyph - 19))
	case glyph >= 36 && glyph <= 61:
		return string(rune('A' + glyph - 36))
	case glyph >= 68 && glyph <= 125:
		return string(rune(glyph - 3))
	default:
		return value
	}
}

// decodePDFGlyphText converts one single-byte CID sequence through a TrueType cmap.
// Authored by: OpenCode
func decodePDFGlyphText(raw []byte, glyphMap map[byte]string) string {
	var encoded, err = hex.DecodeString(string(raw))
	if err != nil {
		return ""
	}
	var text strings.Builder
	for _, glyph := range encoded {
		var character, ok = glyphMap[glyph]
		if !ok {
			return ""
		}
		text.WriteString(decodeGoFontGlyph(character))
	}
	return text.String()
}

// decodePDFUnicode converts a UTF-16BE ToUnicode target to a Go string.
// Authored by: OpenCode
func decodePDFUnicode(raw []byte) string {
	var encoded, err = hex.DecodeString(string(raw))
	if err != nil || len(encoded)%2 != 0 {
		return ""
	}
	var runes = make([]rune, 0, len(encoded)/2)
	for index := 0; index < len(encoded); index += 2 {
		runes = append(runes, rune(encoded[index])<<8|rune(encoded[index+1]))
	}
	return string(runes)
}

// unescapePDFLiteral decodes the basic escaped characters used by PDF literals.
// Authored by: OpenCode
func unescapePDFLiteral(raw []byte) string {
	var text strings.Builder
	for index := 0; index < len(raw); index++ {
		if raw[index] == '\\' && index+1 < len(raw) {
			index++
		}
		text.WriteByte(raw[index])
	}
	return text.String()
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
