package pdf

import "github.com/signintech/gopdf"

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

// drawTableForGopdfDocument keeps concrete gopdf table failures testable.
// Authored by: OpenCode
var drawTableForGopdfDocument = func(table gopdf.TableLayout) error {
	return table.DrawTable()
}

// measureTableCellForGopdfDocument keeps wrapped-cell measurement failures testable before table-row preflight.
// Authored by: OpenCode
var measureTableCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) (bool, float64, error) {
	return document.pdf.IsFitMultiCell(rectangle, text)
}
