package testutil

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	pdfTextOperandPattern   = regexp.MustCompile(`(?s)(\((?:\\.|[^\\()])*\)|<([0-9A-Fa-f]+)>|\[(.*?)\])\s*(Tj|TJ)\b`)
	pdfTextArrayItemPattern = regexp.MustCompile(`(\((?:\\.|[^\\()])*\)|<([0-9A-Fa-f]+)>)`)
	pdfCoordinatePattern    = regexp.MustCompile(`([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))\s+([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))\s+(Td|TD)\b|([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))\s+([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))\s+([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))\s+([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))\s+([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))\s+([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+))\s+Tm\b`)
	pdfFontOperatorPattern  = regexp.MustCompile(`/([^\s/<>()\[\]]+)\s+[-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)\s+Tf\b`)
)

// pdfTextRunEvent holds one ordered text-state or text-showing operation.
// Authored by: OpenCode
type pdfTextRunEvent struct {
	offset    int
	kind      string
	values    [6]float64
	font      string
	operand   []byte
	directHex []byte
	array     []byte
}

// extractPDFTextRuns parses text state, positioning, and text-showing operators.
// Authored by: OpenCode
func extractPDFTextRuns(stream []byte, page int, fontMaps map[string]map[string]string, unicodeMaps []map[string]string, glyphMaps []map[byte]string) ([]PDFTextRun, error) {
	var events, err = parsePDFTextRunEvents(stream)
	if err != nil {
		return nil, err
	}
	return materializePDFTextRuns(events, page, fontMaps, unicodeMaps, glyphMaps), nil
}

// parsePDFTextRunEvents collects text-state and text-showing operators in stream order.
// Authored by: OpenCode
func parsePDFTextRunEvents(stream []byte) ([]pdfTextRunEvent, error) {
	var events []pdfTextRunEvent
	var coordinateEvents, err = parsePDFCoordinateEvents(stream)
	if err != nil {
		return nil, err
	}
	events = append(events, coordinateEvents...)
	for _, match := range pdfFontOperatorPattern.FindAllSubmatchIndex(stream, -1) {
		events = append(events, pdfTextRunEvent{
			offset: match[0],
			kind:   "Tf",
			font:   string(stream[match[2]:match[3]]),
		})
	}
	for _, match := range pdfTextOperandPattern.FindAllSubmatchIndex(stream, -1) {
		var operandStart, operandEnd = match[2], match[3]
		var hexStart, hexEnd = match[4], match[5]
		var arrayStart, arrayEnd = match[6], match[7]
		var directHex, array []byte
		if hexStart >= 0 {
			directHex = stream[hexStart:hexEnd]
		}
		if arrayStart >= 0 {
			array = stream[arrayStart:arrayEnd]
		}
		events = append(events, pdfTextRunEvent{
			offset:    match[0],
			kind:      "text",
			operand:   stream[operandStart:operandEnd],
			directHex: directHex,
			array:     array,
		})
	}
	sort.SliceStable(events, func(left, right int) bool { return events[left].offset < events[right].offset })
	return events, nil
}

// parsePDFCoordinateEvents parses Td, TD, and Tm positioning operators.
// Authored by: OpenCode
func parsePDFCoordinateEvents(stream []byte) ([]pdfTextRunEvent, error) {
	var events []pdfTextRunEvent
	for _, match := range pdfCoordinatePattern.FindAllSubmatchIndex(stream, -1) {
		var event, err = parsePDFCoordinateEvent(stream, match)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// parsePDFCoordinateEvent converts one coordinate operator match into an event.
// Authored by: OpenCode
func parsePDFCoordinateEvent(stream []byte, match []int) (pdfTextRunEvent, error) {
	var event = pdfTextRunEvent{offset: match[0]}
	if match[6] >= 0 {
		event.kind = string(stream[match[6]:match[7]])
		var err error
		event.values[0], err = strconv.ParseFloat(string(stream[match[2]:match[3]]), 64)
		if err != nil {
			return pdfTextRunEvent{}, fmt.Errorf("parse PDF text coordinate: %w", err)
		}
		event.values[1], err = strconv.ParseFloat(string(stream[match[4]:match[5]]), 64)
		if err != nil {
			return pdfTextRunEvent{}, fmt.Errorf("parse PDF text coordinate: %w", err)
		}
		return event, nil
	}
	event.kind = "Tm"
	for index := 0; index < 6; index++ {
		var start = match[8+index*2]
		var end = match[9+index*2]
		var err error
		event.values[index], err = strconv.ParseFloat(string(stream[start:end]), 64)
		if err != nil {
			return pdfTextRunEvent{}, fmt.Errorf("parse PDF text matrix: %w", err)
		}
	}
	return event, nil
}

// materializePDFTextRuns applies ordered text state to decoded text operands.
// Authored by: OpenCode
func materializePDFTextRuns(events []pdfTextRunEvent, page int, fontMaps map[string]map[string]string, unicodeMaps []map[string]string, glyphMaps []map[byte]string) []PDFTextRun {
	var x, y float64
	var font string
	var runs []PDFTextRun
	for _, event := range events {
		switch event.kind {
		case "Td", "TD":
			x, y = event.values[0], event.values[1]
		case "Tm":
			x, y = event.values[4], event.values[5]
		case "Tf":
			font = event.font
		case "text":
			var decoded = decodePDFTextRunOperand(event.operand, event.directHex, event.array, fontMaps[font], unicodeMaps, glyphMaps)
			if decoded != "" {
				runs = append(runs, PDFTextRun{Page: page, Text: decoded, FontResource: font, X: x, Y: y})
			}
		}
	}
	return runs
}

// decodePDFTextRunOperand decodes one text-showing operand using its font.
// Authored by: OpenCode
func decodePDFTextRunOperand(operand, directHex, array []byte, fontMap map[string]string, unicodeMaps []map[string]string, glyphMaps []map[byte]string) string {
	if len(directHex) > 0 {
		return decodePDFTextWithMaps(directHex, fontMap, unicodeMaps, glyphMaps)
	}
	if len(array) > 0 {
		var text strings.Builder
		for _, item := range pdfTextArrayItemPattern.FindAllSubmatch(array, -1) {
			if len(item[2]) > 0 {
				text.WriteString(decodePDFTextWithMaps(item[2], fontMap, unicodeMaps, glyphMaps))
			} else if len(item[0]) >= 2 {
				text.WriteString(unescapePDFLiteral(item[0][1 : len(item[0])-1]))
			}
		}
		return text.String()
	}
	if len(operand) >= 2 && operand[0] == '(' {
		return unescapePDFLiteral(operand[1 : len(operand)-1])
	}
	return decodePDFTextWithMaps(operand, fontMap, unicodeMaps, glyphMaps)
}
