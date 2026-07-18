package testutil

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
)

// PDF inspection patterns identify local PDF objects, geometry, fonts, and page content.
// Authored by: OpenCode
var (
	pdfObjectPattern          = regexp.MustCompile(`(?s)(\d+)\s+\d+\s+obj\b(.*?)\bendobj\b`)
	pdfPagePattern            = regexp.MustCompile(`/Type\s*/Page\b`)
	pdfMediaBoxPattern        = regexp.MustCompile(`/MediaBox\s*\[\s*([-+0-9.]+)\s+([-+0-9.]+)\s+([-+0-9.]+)\s+([-+0-9.]+)\s*\]`)
	pdfContentsPattern        = regexp.MustCompile(`(?s)/Contents\s+(\[[^\]]*\]|\d+\s+\d+\s+R)`)
	pdfObjectReferencePattern = regexp.MustCompile(`(\d+)\s+\d+\s+R`)
	pdfToUnicodePattern       = regexp.MustCompile(`/ToUnicode\s+(\d+)\s+\d+\s+R`)
	pdfFontReferencePattern   = regexp.MustCompile(`/([^\s/<>()\[\]]+)\s+(\d+)\s+\d+\s+R`)
)

// pdfInspectionContent accumulates the inspectable PDF object content.
// Authored by: OpenCode
type pdfInspectionContent struct {
	objects           map[int][]byte
	objectIDs         []int
	unicodeMaps       []map[string]string
	unicodeMapsByID   map[int]map[string]string
	glyphMaps         []map[byte]string
	textStreams       [][]byte
	textStreamObjects map[int][]byte
	explicitPageBoxes []PDFPageBox
	inheritedPageBox  PDFPageBox
	pageCount         int
}

// resolvedPageBoxes returns explicit boxes or the inherited page-tree box for each page.
// Authored by: OpenCode
func (content pdfInspectionContent) resolvedPageBoxes() []PDFPageBox {
	if len(content.explicitPageBoxes) > 0 || content.pageCount == 0 || content.inheritedPageBox.Width <= 0 || content.inheritedPageBox.Height <= 0 {
		return content.explicitPageBoxes
	}
	var boxes = make([]PDFPageBox, content.pageCount)
	for index := range boxes {
		boxes[index] = content.inheritedPageBox
	}
	return boxes
}

// inspectPDFObjects recovers page geometry, text streams, and font maps.
// Authored by: OpenCode
func inspectPDFObjects(payload []byte) (pdfInspectionContent, error) {
	var content = pdfInspectionContent{
		objects:           make(map[int][]byte),
		unicodeMapsByID:   make(map[int]map[string]string),
		textStreamObjects: make(map[int][]byte),
	}
	for _, match := range pdfObjectPattern.FindAllSubmatch(payload, -1) {
		var objectID, err = strconv.Atoi(string(match[1]))
		if err != nil {
			return pdfInspectionContent{}, fmt.Errorf("parse PDF object number: %w", err)
		}
		var object = match[2]
		content.objects[objectID] = object
		content.objectIDs = append(content.objectIDs, objectID)
		if err := content.inspectObject(objectID, object); err != nil {
			return pdfInspectionContent{}, err
		}
	}
	return content, nil
}

// inspectObject recovers inspection data from one indirect PDF object.
// Authored by: OpenCode
func (content *pdfInspectionContent) inspectObject(objectID int, object []byte) error {
	content.inspectPageBox(object)
	var stream, ok, err = pdfObjectStream(object)
	if err != nil || !ok {
		return err
	}
	content.inspectStream(objectID, stream)
	return nil
}

// inspectPageBox tracks explicit and inherited MediaBox page geometry.
// Authored by: OpenCode
func (content *pdfInspectionContent) inspectPageBox(object []byte) {
	var mediaBox = pdfMediaBoxPattern.FindSubmatch(object)
	if len(mediaBox) == 5 {
		content.inheritedPageBox = PDFPageBox{
			Width:  parsePDFNumber(mediaBox[3]) - parsePDFNumber(mediaBox[1]),
			Height: parsePDFNumber(mediaBox[4]) - parsePDFNumber(mediaBox[2]),
		}
	}
	if !pdfPagePattern.Match(object) {
		return
	}
	content.pageCount++
	if len(mediaBox) == 5 {
		content.explicitPageBoxes = append(content.explicitPageBoxes, content.inheritedPageBox)
	}
}

// inspectStream classifies a decoded object stream for text recovery.
// Authored by: OpenCode
func (content *pdfInspectionContent) inspectStream(objectID int, stream []byte) {
	if glyphMap := embeddedFontGlyphMap(stream); len(glyphMap) > 0 {
		content.glyphMaps = append(content.glyphMaps, glyphMap)
	}
	if unicodeMap := pdfUnicodeMap(stream); len(unicodeMap) > 0 {
		content.unicodeMaps = append(content.unicodeMaps, unicodeMap)
		content.unicodeMapsByID[objectID] = unicodeMap
		return
	}
	if bytes.Contains(stream, []byte(" Tj")) || bytes.Contains(stream, []byte(" TJ")) {
		content.textStreams = append(content.textStreams, stream)
		content.textStreamObjects[objectID] = stream
	}
}

// resolvedTextRuns returns text-showing operations in page and stream order.
// Authored by: OpenCode
func (content pdfInspectionContent) resolvedTextRuns() ([]PDFTextRun, error) {
	var runs []PDFTextRun
	var pageNumber int
	var fontMaps = content.fontUnicodeMaps()
	for _, objectID := range content.objectIDs {
		var page = content.objects[objectID]
		if !pdfPagePattern.Match(page) {
			continue
		}
		pageNumber++
		for _, streamID := range pdfPageContentReferences(page) {
			var stream, ok = content.textStreamObjects[streamID]
			if !ok {
				continue
			}
			var streamRuns, err = extractPDFTextRuns(stream, pageNumber, fontMaps, content.unicodeMaps, content.glyphMaps)
			if err != nil {
				return nil, err
			}
			runs = append(runs, streamRuns...)
		}
	}
	return runs, nil
}

// pdfPageContentReferences returns the content stream object IDs for one page.
// Authored by: OpenCode
func pdfPageContentReferences(page []byte) []int {
	var contents = pdfContentsPattern.FindSubmatch(page)
	if len(contents) != 2 {
		return nil
	}
	var references []int
	for _, reference := range pdfObjectReferencePattern.FindAllSubmatch(contents[1], -1) {
		var objectID, err = strconv.Atoi(string(reference[1]))
		if err == nil {
			references = append(references, objectID)
		}
	}
	return references
}

// fontUnicodeMaps resolves PDF font resource names to their ToUnicode maps.
// Authored by: OpenCode
func (content pdfInspectionContent) fontUnicodeMaps() map[string]map[string]string {
	var fontObjectIDs = make(map[string]int)
	for _, objectID := range content.objectIDs {
		for _, reference := range pdfFontReferencePattern.FindAllSubmatch(content.objects[objectID], -1) {
			var targetID, err = strconv.Atoi(string(reference[2]))
			if err == nil {
				fontObjectIDs[string(reference[1])] = targetID
			}
		}
	}
	var fontMaps = make(map[string]map[string]string)
	for resource, fontObjectID := range fontObjectIDs {
		var toUnicode = pdfToUnicodePattern.FindSubmatch(content.objects[fontObjectID])
		if len(toUnicode) != 2 {
			continue
		}
		var mapObjectID, err = strconv.Atoi(string(toUnicode[1]))
		if err != nil {
			continue
		}
		if unicodeMap := content.unicodeMapsByID[mapObjectID]; len(unicodeMap) > 0 {
			fontMaps[resource] = unicodeMap
		}
	}
	return fontMaps
}
