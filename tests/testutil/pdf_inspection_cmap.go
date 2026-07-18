package testutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	pdfHexTextPattern = regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*Tj\b|<([0-9A-Fa-f]+)>`)
	pdfCMapPattern    = regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
	pdfBFCharPattern  = regexp.MustCompile(`(?s)beginbfchar\s*(.*?)\s*endbfchar`)
	pdfBFRangePattern = regexp.MustCompile(`(?s)beginbfrange\s*(.*?)\s*endbfrange`)
	pdfRangeEntry     = regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
)

// decodePDFTextWithMaps decodes text through the available PDF font maps.
// Authored by: OpenCode
func decodePDFTextWithMaps(raw []byte, fontMap map[string]string, unicodeMaps []map[string]string, glyphMaps []map[byte]string) string {
	if decoded := decodePDFTextByUnicodeMap(raw, fontMap); decoded != "" {
		return decoded
	}
	if decoded := decodePDFText(raw, fontMap); decoded != "" {
		return decoded
	}
	for _, unicodeMap := range unicodeMaps {
		if decoded := decodePDFTextByUnicodeMap(raw, unicodeMap); decoded != "" {
			return decoded
		}
		if decoded := decodePDFText(raw, unicodeMap); decoded != "" {
			return decoded
		}
	}
	for _, glyphMap := range glyphMaps {
		if decoded := decodePDFGlyphText(raw, glyphMap); decoded != "" {
			return decoded
		}
	}
	return ""
}

// decodePDFTextByUnicodeMap decodes two-byte CID text through one ToUnicode map.
// Authored by: OpenCode
func decodePDFTextByUnicodeMap(raw []byte, unicodeMap map[string]string) string {
	if len(unicodeMap) == 0 {
		return ""
	}
	var encoded, err = hex.DecodeString(string(raw))
	if err != nil || len(encoded) == 0 || len(encoded)%2 != 0 {
		return ""
	}
	var text strings.Builder
	for index := 0; index < len(encoded); index += 2 {
		var code = strings.ToUpper(hex.EncodeToString(encoded[index : index+2]))
		var decoded, ok = unicodeMap[code]
		if !ok {
			return ""
		}
		text.WriteString(decoded)
	}
	return text.String()
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
