package testutil

import (
	"bytes"
	"encoding/hex"
	"strings"
)

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
