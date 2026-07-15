package testutil

import "testing"

// TestDecodePDFTextDecodesMultiByteGoFontDigitOne verifies embedded Go font
// ToUnicode source codes preserve the Annex 1 title's digit.
// Authored by: OpenCode
func TestDecodePDFTextDecodesMultiByteGoFontDigitOne(t *testing.T) {
	var decoded = decodePDFText([]byte("0014"), nil)
	if decoded != "1" {
		t.Fatalf("decoded text = %q, want %q", decoded, "1")
	}
}
