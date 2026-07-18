package testutil

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

var pdfStreamLengthPattern = regexp.MustCompile(`/Length\s+(\d+)\b`)

// pdfObjectStream returns one decoded object stream when present.
// Authored by: OpenCode
func pdfObjectStream(object []byte) ([]byte, bool, error) {
	var start, end, found, err = pdfObjectStreamBounds(object)
	if !found {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var stream = object[start : start+end]
	var raw = bytes.TrimRight(stream, "\r\n")
	if !bytes.Contains(object[:start], []byte("/FlateDecode")) {
		return raw, true, nil
	}
	return decodePDFObjectStream(raw, stream, object[:start], end)
}

// pdfObjectStreamBounds locates the payload boundaries of one PDF stream.
// Authored by: OpenCode
func pdfObjectStreamBounds(object []byte) (int, int, bool, error) {
	var marker = []byte("stream")
	var start = bytes.Index(object, marker)
	if start < 0 {
		return 0, 0, false, nil
	}
	start += len(marker)
	if bytes.HasPrefix(object[start:], []byte("\r\n")) {
		start += 2
	} else if bytes.HasPrefix(object[start:], []byte("\n")) || bytes.HasPrefix(object[start:], []byte("\r")) {
		start++
	}
	var end = bytes.LastIndex(object[start:], []byte("endstream"))
	if end < 0 {
		return 0, 0, true, fmt.Errorf("unterminated PDF stream")
	}
	return start, end, true, nil
}

// decodePDFObjectStream decodes a Flate stream and preserves close errors.
// Authored by: OpenCode
func decodePDFObjectStream(raw []byte, stream []byte, dictionary []byte, end int) ([]byte, bool, error) {
	if lengthMatch := pdfStreamLengthPattern.FindSubmatch(dictionary); len(lengthMatch) == 2 {
		var length, lengthErr = strconv.Atoi(string(lengthMatch[1]))
		if lengthErr == nil && length >= 0 && length <= end {
			raw = stream[:length]
		}
	}
	var reader, err = zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, false, fmt.Errorf("open PDF Flate stream: %w", err)
	}
	var decoded []byte
	decoded, err = io.ReadAll(reader)
	if err != nil {
		if closeErr := reader.Close(); closeErr != nil {
			return nil, false, fmt.Errorf("decode PDF Flate stream: %w; close PDF Flate stream: %w", err, closeErr)
		}
		return nil, false, fmt.Errorf("decode PDF Flate stream: %w", err)
	}
	if err = reader.Close(); err != nil {
		return nil, false, fmt.Errorf("close PDF Flate stream: %w", err)
	}
	return decoded, true, nil
}

// parsePDFNumber converts an already validated PDF numeric token.
// Authored by: OpenCode
func parsePDFNumber(raw []byte) float64 {
	var value, err = strconv.ParseFloat(string(raw), 64)
	if err != nil {
		panic(fmt.Sprintf("validated PDF number %q: %v", raw, err))
	}
	return value
}
