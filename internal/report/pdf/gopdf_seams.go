package pdf

import (
	"strings"

	"github.com/signintech/gopdf"
)

// writeTextForGopdfDocument keeps concrete gopdf text failures testable.
// Authored by: OpenCode
var writeTextForGopdfDocument = func(document *gopdfDocument, text string) error {
	return document.pdf.Text(text)
}

// writeCellForGopdfDocument keeps concrete gopdf cell failures testable.
// Authored by: OpenCode
var writeCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) error {
	return document.pdf.Cell(rectangle, text)
}

// writeMultiCellForGopdfDocument keeps concrete gopdf wrapped-text failures testable.
// Authored by: OpenCode
var writeMultiCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) error {
	return document.pdf.MultiCell(rectangle, text)
}

// fitMultiCellForGopdfDocument keeps bold-paragraph measurement failures
// testable without depending on an opaque gopdf implementation detail.
// Authored by: OpenCode
var fitMultiCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) (bool, float64, error) {
	return document.pdf.IsFitMultiCell(rectangle, text)
}

// drawTableForGopdfDocument keeps concrete gopdf table failures testable.
// Authored by: OpenCode
var drawTableForGopdfDocument = func(table gopdf.TableLayout) error {
	return table.DrawTable()
}

// measureTableCellForGopdfDocument keeps wrapped-cell measurement failures testable before table-row preflight.
// Authored by: OpenCode
var measureTableCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) (bool, float64, error) {
	var splitLines, err = document.pdf.SplitTextWithOption(text, rectangle.W, tableCellBreakOption())
	if err != nil {
		return false, 0, err
	}
	return document.pdf.IsFitMultiCellWithNewline(rectangle, strings.Join(splitLines, "\n"))
}

// finalizeGopdfDocument keeps concrete PDF finalization failures testable.
// Authored by: OpenCode
var finalizeGopdfDocument = func(document *gopdfDocument) ([]byte, error) {
	return document.pdf.GetBytesPdfReturnErr()
}
