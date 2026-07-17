package pdf

import (
	"bytes"
	"fmt"

	"github.com/signintech/gopdf"
)

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

// AddBoldParagraph emits one sanitized wrapped paragraph using the registered
// bold font for both measurement and drawing, then advances by its measured
// height. For example, a renderer can call document.AddBoldParagraph(warning)
// to keep a multi-line warning fully bold without changing its logical block.
// Authored by: OpenCode
func (document *gopdfDocument) AddBoldParagraph(text string) error {
	if err := document.ensureSpace(0); err != nil {
		return err
	}
	var sanitized = sanitizeText(text)
	if err := document.pdf.SetFont(fontBold, "", 9); err != nil {
		return err
	}
	var rectangle = &gopdf.Rect{W: contentWide, H: pageBottom - pageMargin}
	var fits, height, err = fitMultiCellForGopdfDocument(document, rectangle, sanitized)
	if err != nil {
		return err
	}
	if !fits {
		return fmt.Errorf("bold paragraph does not fit within the printable page area")
	}
	if err := document.ensureSpace(height); err != nil {
		return err
	}
	document.pdf.SetXY(pageMargin, document.y)
	if err := writeMultiCellForGopdfDocument(document, rectangle, sanitized); err != nil {
		return err
	}
	document.y += height
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
	if err = document.prepareTableStart(table.Title, rowHeight); err != nil {
		return fmt.Errorf("table preflight failed for row 1: %w", err)
	}
	return document.addTableRows(table, columns, rowHeight)
}

// addTableRows draws preflighted rows and creates continuation pages only when
// the remaining rows require them.
// Authored by: OpenCode
func (document *gopdfDocument) addTableRows(table pdfTable, columns []pdfColumn, rowHeight float64) error {
	var rowOffset int
	var remainingRows = table.Rows
	for len(remainingRows) > 0 {
		var capacity = document.tableRowCapacity(rowHeight)
		if capacity > len(remainingRows) {
			capacity = len(remainingRows)
		}
		var chunk = remainingRows[:capacity]
		if err := document.drawTableChunk(table, columns, chunk, rowHeight, rowOffset, len(remainingRows) == len(chunk)); err != nil {
			return err
		}
		remainingRows = remainingRows[capacity:]
		rowOffset += capacity
		if len(remainingRows) > 0 {
			if document.tableRowCapacityAt(document.tableContinuationStartY(), rowHeight) == 0 {
				return fmt.Errorf("table preflight failed for row %d: row does not fit within a fresh continuation page", rowOffset+1)
			}
			if err := document.addTableContinuationPage(table.ContinuationTitle); err != nil {
				return fmt.Errorf("table continuation setup failed before row %d: %w", rowOffset+1, err)
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
		return 0, fmt.Errorf("table measurement failed before row 1: %w", err)
	}
	var rowHeight = minimum
	for rowIndex, row := range rows {
		var height, err = document.tableRowContentHeight(columns, row)
		if err != nil {
			return 0, fmt.Errorf("table measurement failed for row %d: %w", rowIndex+1, err)
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
	var fits, height, err = measureTableCellForGopdfDocument(document, &gopdf.Rect{W: columns[index].Width - 4, H: pageBottom - pageMargin}, sanitizeTableCell(cell))
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
	return document.tableRowCapacityAt(document.y, rowHeight)
}

// tableRowCapacityAt returns the row count that fits after reserving the
// table border allowance and one repeated header row at a page position.
// Authored by: OpenCode
func (document *gopdfDocument) tableRowCapacityAt(startY float64, rowHeight float64) int {
	if rowHeight <= 0 {
		return 0
	}
	var available = pageBottom - startY - 12 - rowHeight
	if available < rowHeight {
		return 0
	}
	return int(available / rowHeight)
}

// drawTableChunk draws one page-local table chunk and records its text extract.
// Authored by: OpenCode
func (document *gopdfDocument) drawTableChunk(table pdfTable, columns []pdfColumn, rows [][]string, rowHeight float64, rowOffset int, includeStyledLastRow bool) error {
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
		return fmt.Errorf("table drawing failed for row %d: %w", rowOffset+1, err)
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

// tableContinuationStartY returns the first data-table position after a
// continuation context and the table's top spacing have been emitted.
// Authored by: OpenCode
func (document *gopdfDocument) tableContinuationStartY() float64 {
	return pageMargin + 16 + tableSpacing
}

// prepareTableStart reserves enough space for a table title, header, and first
// row before emitting any part of the table block.
// Authored by: OpenCode
func (document *gopdfDocument) prepareTableStart(title string, rowHeight float64) error {
	var titleHeight float64
	if title != "" {
		titleHeight = 16
		if document.y > pageMargin {
			titleHeight += sectionSpacing
		}
	}
	var required = titleHeight + tableSpacing + rowHeight*2 + 12
	if document.y+required > pageBottom {
		document.addPage()
		titleHeight = 0
		if title != "" {
			titleHeight = 16
		}
		required = titleHeight + tableSpacing + rowHeight*2 + 12
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
	if document.tableRowCapacity(rowHeight) == 0 {
		return fmt.Errorf("table row height %.0f does not fit within the printable page area", rowHeight)
	}
	return nil
}

// Bytes returns the concrete PDF byte payload or its finalization error.
// Authored by: OpenCode
func (document *gopdfDocument) Bytes() ([]byte, error) {
	var payload, err = finalizeGopdfDocument(document)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), payload...), nil
}
