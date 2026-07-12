package pdf

import (
	"bytes"
	"fmt"

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

// writeMultiCellForGopdfDocument keeps concrete gopdf wrapped-text failures
// testable.
// Authored by: OpenCode
var writeMultiCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) error {
	return document.pdf.MultiCell(rectangle, text)
}

// drawTableForGopdfDocument keeps concrete gopdf table failures testable.
// Authored by: OpenCode
var drawTableForGopdfDocument = func(table gopdf.TableLayout) error {
	return table.DrawTable()
}

// measureTableCellForGopdfDocument keeps wrapped-cell measurement failures
// testable before table-row preflight.
// Authored by: OpenCode
var measureTableCellForGopdfDocument = func(document *gopdfDocument, rectangle *gopdf.Rect, text string) (bool, float64, error) {
	return document.pdf.IsFitMultiCell(rectangle, text)
}

// gopdfDocument renders selectable text through gopdf.
// Authored by: OpenCode
type gopdfDocument struct {
	pdf        gopdf.GoPdf
	y          float64
	pageWidth  float64
	pageHeight float64
	started    bool
}

// newGopdfDocument creates one local PDF document adapter.
// Authored by: OpenCode
func newGopdfDocument() *gopdfDocument {
	return &gopdfDocument{y: pageMargin}
}

// StartPDF starts one A4 PDF document.
// Authored by: OpenCode
func (document *gopdfDocument) StartPDF(pageSize string) error {
	if pageSize != PageSizeA4 {
		return fmt.Errorf("unsupported PDF page size %q", pageSize)
	}
	document.pdf = gopdf.GoPdf{}
	document.pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4Landscape})
	document.pdf.AddPage()
	document.pageWidth = gopdf.PageSizeA4Landscape.W
	document.pageHeight = gopdf.PageSizeA4Landscape.H
	document.started = true
	return nil
}

// AddTTFFont registers one application-supplied font through gopdf.
// Authored by: OpenCode
func (document *gopdfDocument) AddTTFFont(name string, data []byte) error {
	if !document.started {
		return fmt.Errorf("PDF document must be started before loading fonts")
	}
	return document.pdf.AddTTFFontByReader(name, bytes.NewReader(data))
}

// AddTitle emits a top-level PDF heading with bold font styling.
// Authored by: OpenCode
func (document *gopdfDocument) AddTitle(text string) error {
	return document.addTextBlock(text, fontBold, 16, 24)
}

// AddSectionHeading emits a section heading with bold font styling.
// Authored by: OpenCode
func (document *gopdfDocument) AddSectionHeading(text string) error {
	return document.addSpacedTextBlock(text, fontBold, 12, 18, sectionSpacing)
}

// AddSubsectionHeading emits a subsection heading with bold font styling.
// Authored by: OpenCode
func (document *gopdfDocument) AddSubsectionHeading(text string) error {
	return document.addSpacedTextBlock(text, fontBold, 10, 16, sectionSpacing)
}

// AddKeyValue emits one styled label/value row using Cell and Text operations.
// Authored by: OpenCode
func (document *gopdfDocument) AddKeyValue(label string, value string) error {
	if err := document.ensureSpace(16); err != nil {
		return err
	}
	var labelText = sanitizeText(label) + ":"
	var valueText = sanitizeText(value)
	document.pdf.SetXY(pageMargin, document.y)
	if err := document.pdf.SetFont(fontBold, "", 9); err != nil {
		return err
	}
	if err := writeCellForGopdfDocument(document, &gopdf.Rect{W: 150, H: 12}, labelText); err != nil {
		return err
	}
	if err := document.pdf.SetFont(fontRegular, "", 9); err != nil {
		return err
	}
	document.pdf.SetXY(pageMargin+154, document.y)
	if err := writeTextForGopdfDocument(document, valueText); err != nil {
		return err
	}
	document.y += 14
	return nil
}

// AddParagraph emits wrapped paragraph text through MultiCell.
// Authored by: OpenCode
func (document *gopdfDocument) AddParagraph(text string) error {
	if err := document.ensureSpace(34); err != nil {
		return err
	}
	var sanitized = sanitizeText(text)
	if err := document.pdf.SetFont(fontRegular, "", 9); err != nil {
		return err
	}
	document.pdf.SetXY(pageMargin, document.y)
	if err := writeMultiCellForGopdfDocument(document, &gopdf.Rect{W: contentWide, H: 30}, sanitized); err != nil {
		return err
	}
	document.y += 34
	return nil
}

// AddTable emits one structured table through gopdf table layout primitives.
// Authored by: OpenCode
func (document *gopdfDocument) AddTable(table pdfTable) error {
	if len(table.Columns) == 0 {
		return fmt.Errorf("pdf table columns are required")
	}
	if len(table.Rows) == 0 {
		return nil
	}
	var rowHeight = table.RowHeight
	if rowHeight <= 0 {
		rowHeight = 24
	}
	var columns = printableWidthColumns(table.Columns)
	var err error
	rowHeight, err = document.tableRowHeight(columns, table.Rows, rowHeight)
	if err != nil {
		return err
	}
	var remainingRows = table.Rows
	if err = document.prepareTableStart(table.Title, rowHeight); err != nil {
		return err
	}
	for len(remainingRows) > 0 {
		var capacity = document.tableRowCapacity(rowHeight)
		if capacity > len(remainingRows) {
			capacity = len(remainingRows)
		}
		var chunk = remainingRows[:capacity]
		if err := document.drawTableChunk(table, columns, chunk, rowHeight, len(remainingRows) == len(chunk)); err != nil {
			return err
		}
		remainingRows = remainingRows[capacity:]
		if len(remainingRows) > 0 {
			if err := document.addTableContinuationPage(table.ContinuationTitle); err != nil {
				return err
			}
			document.y += tableSpacing
		}
	}
	return nil
}

// tableRowHeight returns a single table row height that contains every wrapped
// cell before the table preflight reserves its header, rows, and borders.
// Authored by: OpenCode
func (document *gopdfDocument) tableRowHeight(columns []pdfColumn, rows [][]string, minimum float64) (float64, error) {
	if err := document.pdf.SetFont(fontRegular, "", 6.5); err != nil {
		return 0, err
	}
	var rowHeight = minimum
	for _, row := range rows {
		var height, err = document.tableRowContentHeight(columns, row)
		if err != nil {
			return 0, err
		}
		if height > rowHeight {
			rowHeight = height
		}
	}
	return rowHeight, nil
}

// tableRowContentHeight measures the padded height required by one table row.
// Authored by: OpenCode
func (document *gopdfDocument) tableRowContentHeight(columns []pdfColumn, row []string) (float64, error) {
	var rowHeight float64
	for index, cell := range row {
		var height, measured, err = document.tableCellHeight(columns, index, cell)
		if err != nil {
			return 0, err
		}
		if measured && height > rowHeight {
			rowHeight = height
		}
	}
	return rowHeight, nil
}

// tableCellHeight measures one populated cell and includes its vertical padding.
// Authored by: OpenCode
func (document *gopdfDocument) tableCellHeight(columns []pdfColumn, index int, cell string) (float64, bool, error) {
	if cell == "" || index >= len(columns) {
		return 0, false, nil
	}
	var fits, height, err = measureTableCellForGopdfDocument(document, &gopdf.Rect{W: columns[index].Width - 4, H: pageBottom - pageMargin}, sanitizeText(cell))
	if err != nil {
		return 0, false, err
	}
	if !fits {
		return 0, false, fmt.Errorf("table cell does not fit within the printable page area")
	}
	return height + 4, true, nil
}

// AddAnnexPageBreak starts Annex 1 on a new page.
// Authored by: OpenCode
func (document *gopdfDocument) AddAnnexPageBreak() error {
	document.pdf.AddPage()
	document.y = pageMargin
	return nil
}

// addTextBlock emits a title or heading through gopdf Text.
// Authored by: OpenCode
func (document *gopdfDocument) addTextBlock(text string, font string, size float64, verticalAdvance float64) error {
	if err := document.ensureSpace(verticalAdvance); err != nil {
		return err
	}
	var sanitized = sanitizeText(text)
	if err := document.pdf.SetFont(font, "", size); err != nil {
		return err
	}
	document.pdf.SetXY(pageMargin, document.y)
	if err := writeTextForGopdfDocument(document, sanitized); err != nil {
		return err
	}
	document.y += verticalAdvance
	return nil
}

// addSpacedTextBlock emits a heading with positive top margin so adjacent PDF
// sections cannot collide vertically.
// Authored by: OpenCode
func (document *gopdfDocument) addSpacedTextBlock(text string, font string, size float64, verticalAdvance float64, topSpacing float64) error {
	if document.started && document.y > pageMargin {
		if document.y+verticalAdvance+topSpacing > pageBottom {
			document.addPage()
		} else {
			document.y += topSpacing
		}
	}
	return document.addTextBlock(text, font, size, verticalAdvance)
}

// ensureSpace adds a continuation page before content would leave the A4 area.
// Authored by: OpenCode
func (document *gopdfDocument) ensureSpace(height float64) error {
	if !document.started {
		return fmt.Errorf("PDF document must be started before adding content")
	}
	if document.y+height <= pageBottom {
		return nil
	}
	document.addPage()
	return nil
}

// tableRowCapacity returns the number of data rows whose header, cells, and
// borders fit wholly within the current page's printable area.
// Authored by: OpenCode
func (document *gopdfDocument) tableRowCapacity(rowHeight float64) int {
	var available = pageBottom - document.y - tableSpacing
	if rowHeight <= 0 || available < rowHeight*2 {
		return 0
	}
	return int(available/rowHeight) - 1
}

// drawTableChunk draws one page-local table chunk and records its text extract.
// Authored by: OpenCode
func (document *gopdfDocument) drawTableChunk(table pdfTable, columns []pdfColumn, rows [][]string, rowHeight float64, includeStyledLastRow bool) error {
	var layout = document.pdf.NewTableLayout(pageMargin, document.y, rowHeight, len(rows))
	for _, column := range columns {
		layout.AddColumn(sanitizeText(column.Header), column.Width, column.Align)
	}
	layout.SetTableStyle(tableStyle())
	layout.SetHeaderStyle(headerStyle())
	layout.SetCellStyle(cellStyle())
	for rowIndex, row := range rows {
		var sanitizedRow = sanitizeRow(row)
		if includeStyledLastRow && table.StyledLastRow && rowIndex == len(rows)-1 {
			layout.AddStyledRow(styledRowCells(sanitizedRow))
		} else {
			layout.AddRow(sanitizedRow)
		}
	}
	if err := drawTableForGopdfDocument(layout); err != nil {
		return err
	}
	document.y += rowHeight*float64(len(rows)+1) + 12
	return nil
}

// addPage starts a new page without table continuation context.
// Authored by: OpenCode
func (document *gopdfDocument) addPage() {
	document.pdf.AddPage()
	document.y = pageMargin
}

// addTableContinuationPage starts a new page with context for a table that
// actually continued from the preceding page.
// Authored by: OpenCode
func (document *gopdfDocument) addTableContinuationPage(context string) error {
	document.addPage()
	var label = sanitizeText(context)
	if err := document.pdf.SetFont(fontBold, "", 10); err != nil {
		return err
	}
	document.pdf.SetXY(pageMargin, document.y)
	if err := writeTextForGopdfDocument(document, label); err != nil {
		return err
	}
	document.y += 16
	return nil
}

// prepareTableStart reserves enough space for a table title, header, and first
// row before emitting any part of the table block.
// Authored by: OpenCode
func (document *gopdfDocument) prepareTableStart(title string, rowHeight float64) error {
	var titleHeight float64
	if title != "" {
		titleHeight = sectionSpacing + 16
	}
	var required = titleHeight + tableSpacing*2 + rowHeight*2
	if document.y+required > pageBottom {
		document.addPage()
	}
	if document.y+required > pageBottom {
		return fmt.Errorf("table row height %.0f does not fit within the printable page area", rowHeight)
	}
	if title != "" {
		if err := document.AddSubsectionHeading(title); err != nil {
			return err
		}
	}
	document.y += tableSpacing
	return nil
}

// printableWidthColumns scales source column proportions to the full printable
// width, leaving the page margins as the equal outer table margins.
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

// Bytes returns the concrete PDF byte payload.
// Authored by: OpenCode
func (document *gopdfDocument) Bytes() []byte {
	return append([]byte(nil), document.pdf.GetBytesPdf()...)
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
		sanitized = append(sanitized, sanitizeText(cell))
	}
	return sanitized
}
