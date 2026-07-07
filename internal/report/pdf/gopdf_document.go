package pdf

import (
	"bytes"
	"fmt"
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

// gopdfDocument renders selectable text through gopdf while retaining extracted
// report text comments for deterministic automated assertions.
// Authored by: OpenCode
type gopdfDocument struct {
	pdf     gopdf.GoPdf
	y       float64
	texts   []string
	started bool
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
	document.pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	document.pdf.AddPage()
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
	return document.addTextBlock(text, fontBold, 12, 18)
}

// AddSubsectionHeading emits a subsection heading with bold font styling.
// Authored by: OpenCode
func (document *gopdfDocument) AddSubsectionHeading(text string) error {
	return document.addTextBlock(text, fontBold, 10, 16)
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
	document.recordText(labelText + " " + valueText)
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
	document.recordText(sanitized)
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
	var remainingRows = table.Rows
	var firstChunk = true
	for len(remainingRows) > 0 {
		var capacity = document.tableRowCapacity(rowHeight)
		if capacity < 1 {
			document.addContinuationPage(table.ContinuationTitle)
			capacity = document.tableRowCapacity(rowHeight)
		}
		if capacity > len(remainingRows) {
			capacity = len(remainingRows)
		}
		var chunk = remainingRows[:capacity]
		if err := document.drawTableChunk(table, chunk, rowHeight, firstChunk && len(remainingRows) == len(chunk)); err != nil {
			return err
		}
		remainingRows = remainingRows[capacity:]
		firstChunk = false
		if len(remainingRows) > 0 {
			document.addContinuationPage(table.ContinuationTitle)
		}
	}
	return nil
}

// AddAnnexPageBreak starts Annex 1 on a new page.
// Authored by: OpenCode
func (document *gopdfDocument) AddAnnexPageBreak() error {
	document.pdf.AddPage()
	document.y = pageMargin
	document.recordText("PAGE BREAK: Annex 1")
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
	document.recordText(sanitized)
	document.y += verticalAdvance
	return nil
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
	document.addContinuationPage("Continued")
	return nil
}

// tableRowCapacity returns the number of data rows that fit on the current page.
// Authored by: OpenCode
func (document *gopdfDocument) tableRowCapacity(rowHeight float64) int {
	var available = pageBottom - document.y - rowHeight
	if available <= 0 {
		return 0
	}
	if available < rowHeight {
		return 1
	}
	return int(available / rowHeight)
}

// drawTableChunk draws one page-local table chunk and records its text extract.
// Authored by: OpenCode
func (document *gopdfDocument) drawTableChunk(table pdfTable, rows [][]string, rowHeight float64, includeStyledLastRow bool) error {
	if table.Title != "" {
		if err := document.AddSubsectionHeading(table.Title); err != nil {
			return err
		}
	}
	var layout = document.pdf.NewTableLayout(pageMargin, document.y, rowHeight, len(rows))
	for _, column := range table.Columns {
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
	document.recordTable(table.Columns, rows)
	document.y += rowHeight*float64(len(rows)+1) + 12
	return nil
}

// addContinuationPage starts a new page with repeated context.
// Authored by: OpenCode
func (document *gopdfDocument) addContinuationPage(context string) {
	document.pdf.AddPage()
	document.y = pageMargin
	var label = sanitizeText(context)
	if label == "" {
		label = "Continued"
	}
	document.recordText("CONTINUED: " + label)
}

// recordTable records table headers and rows for deterministic test assertions.
// Authored by: OpenCode
func (document *gopdfDocument) recordTable(columns []pdfColumn, rows [][]string) {
	var headers []string
	for _, column := range columns {
		headers = append(headers, sanitizeText(column.Header))
	}
	document.recordText(strings.Join(headers, "\t"))
	for _, row := range rows {
		document.recordText(strings.Join(sanitizeRow(row), "\t"))
	}
}

// recordText appends one sanitized extract line.
// Authored by: OpenCode
func (document *gopdfDocument) recordText(text string) {
	document.texts = append(document.texts, sanitizeText(text))
}

// Bytes returns the PDF byte payload with deterministic text comments for tests.
// Authored by: OpenCode
func (document *gopdfDocument) Bytes() []byte {
	var payload = append([]byte(nil), document.pdf.GetBytesPdf()...)
	var comments bytes.Buffer
	comments.WriteString("\n% ghostfolio-cryptogains text extract\n")
	for _, text := range document.texts {
		comments.WriteString("% ")
		comments.WriteString(strings.ReplaceAll(text, "\n", " "))
		comments.WriteByte('\n')
	}
	return append(payload, comments.Bytes()...)
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
