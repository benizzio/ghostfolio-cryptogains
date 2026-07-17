package pdf

import (
	"strings"

	"github.com/signintech/gopdf"
)

// tableCellBreakOption returns the word-break policy used by gopdf's table
// layout for cell drawing.
// Authored by: OpenCode
func tableCellBreakOption() *gopdf.BreakOption {
	return &gopdf.BreakOption{Mode: gopdf.BreakModeIndicatorSensitive, BreakIndicator: ' '}
}

// printableWidthColumns scales source column proportions to the full printable width, leaving page margins as equal outer table margins.
// Authored by: OpenCode
func printableWidthColumns(columns []pdfColumn) []pdfColumn {
	var width float64
	for _, column := range columns {
		width += column.Width
	}
	var scaled = append([]pdfColumn(nil), columns...)
	if width <= 0 {
		var equalWidth = contentWide / float64(len(scaled))
		for index := range scaled {
			scaled[index].Width = equalWidth
		}
		return scaled
	}
	var used float64
	for index := range scaled {
		if index == len(scaled)-1 {
			scaled[index].Width = contentWide - used
			break
		}
		scaled[index].Width = scaled[index].Width * contentWide / width
		used += scaled[index].Width
	}
	return scaled
}

// tableStyle returns the base table border style.
// Authored by: OpenCode
func tableStyle() gopdf.CellStyle {
	return gopdf.CellStyle{BorderStyle: gopdf.BorderStyle{Top: true, Left: true, Right: true, Bottom: true, Width: 0.4, RGBColor: gopdf.RGBColor{R: 90, G: 90, B: 90}}}
}

// headerStyle returns the table header style.
// Authored by: OpenCode
func headerStyle() gopdf.CellStyle {
	return gopdf.CellStyle{BorderStyle: gopdf.BorderStyle{Top: true, Left: true, Right: true, Bottom: true, Width: 0.4, RGBColor: gopdf.RGBColor{R: 70, G: 70, B: 70}}, FillColor: gopdf.RGBColor{R: 225, G: 230, B: 236}, TextColor: gopdf.RGBColor{R: 0, G: 0, B: 0}, Font: fontBold, FontSize: 7}
}

// cellStyle returns the default table cell style.
// Authored by: OpenCode
func cellStyle() gopdf.CellStyle {
	return gopdf.CellStyle{BorderStyle: gopdf.BorderStyle{Top: true, Left: true, Right: true, Bottom: true, Width: 0.3, RGBColor: gopdf.RGBColor{R: 120, G: 120, B: 120}}, FillColor: gopdf.RGBColor{R: 255, G: 255, B: 255}, TextColor: gopdf.RGBColor{R: 0, G: 0, B: 0}, Font: fontRegular, FontSize: 6.5}
}

// highlightedCellStyle returns the emphasized row style.
// Authored by: OpenCode
func highlightedCellStyle() gopdf.CellStyle {
	return gopdf.CellStyle{BorderStyle: gopdf.BorderStyle{Top: true, Left: true, Right: true, Bottom: true, Width: 0.4, RGBColor: gopdf.RGBColor{R: 80, G: 80, B: 80}}, FillColor: gopdf.RGBColor{R: 245, G: 247, B: 250}, TextColor: gopdf.RGBColor{R: 0, G: 0, B: 0}, Font: fontBold, FontSize: 6.5}
}

// styledRowCells converts strings into highlighted gopdf row cells.
// Authored by: OpenCode
func styledRowCells(row []string) []gopdf.RowCell {
	var cells = make([]gopdf.RowCell, 0, len(row))
	var style = highlightedCellStyle()
	for _, cell := range row {
		cells = append(cells, gopdf.NewRowCell(cell, style))
	}
	return cells
}

// sanitizeRow returns a sanitized copy of one table row.
// Authored by: OpenCode
func sanitizeRow(row []string) []string {
	var sanitized = make([]string, 0, len(row))
	for _, cell := range row {
		sanitized = append(sanitized, sanitizeTableCell(cell))
	}
	return sanitized
}

// sanitizeTableCell preserves renderer-controlled line boundaries while
// applying the existing single-line sanitization to each table-cell line.
// Authored by: OpenCode
func sanitizeTableCell(raw string) string {
	var lines = strings.Split(raw, "\n")
	for index := range lines {
		lines[index] = sanitizeText(lines[index])
		lines[index] = wrapLongNumericTableCell(lines[index])
	}
	return strings.Join(lines, "\n")
}

// wrapLongNumericTableCell adds explicit PDF line boundaries for long numeric
// values that otherwise exceed a gopdf table cell's fixed row height.
// Authored by: OpenCode
func wrapLongNumericTableCell(line string) string {
	if len(line) <= 18 || !isNumericTableCell(line) {
		return line
	}
	var parts []string
	for len(line) > 8 {
		parts = append(parts, line[:8])
		line = line[8:]
	}
	parts = append(parts, line)
	return strings.Join(parts, "\n")
}

// isNumericTableCell reports whether a table cell is one long signed decimal.
// Authored by: OpenCode
func isNumericTableCell(line string) bool {
	var digits int
	for index, character := range line {
		if character >= '0' && character <= '9' {
			digits++
			continue
		}
		if (character == '-' && index == 0) || character == '.' {
			continue
		}
		return false
	}
	return digits > 0
}
