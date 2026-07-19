package pdf

import "testing"

// TestLoadApplicationFontsValidatesAndLoadsRegularAndBoldFonts specifies the
// application-supplied font seam.
// Authored by: OpenCode
func TestLoadApplicationFontsValidatesAndLoadsRegularAndBoldFonts(t *testing.T) {
	var recorder = &fontLoadRecorder{}
	var fonts = FontData{Regular: []byte("regular-ttf-bytes"), Bold: []byte("bold-ttf-bytes")}

	var err = loadApplicationFonts(recorder, fonts)
	if err != nil {
		t.Fatalf("load application fonts: %v", err)
	}

	assertLoadedFont(t, recorder, fontRegular, fonts.Regular)
	assertLoadedFont(t, recorder, fontBold, fonts.Bold)
	assertErrorContains(t, func() error { return loadApplicationFonts(&fontLoadRecorder{}, FontData{Bold: fonts.Bold}) }, "regular font data")
	assertErrorContains(t, func() error { return loadApplicationFonts(&fontLoadRecorder{}, FontData{Regular: fonts.Regular}) }, "bold font data")
}
