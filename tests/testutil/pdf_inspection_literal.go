package testutil

import (
	"regexp"
	"strings"
)

var pdfLiteralPattern = regexp.MustCompile(`\((([^\\()]|\\.)*)\)\s*Tj\b`)

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
