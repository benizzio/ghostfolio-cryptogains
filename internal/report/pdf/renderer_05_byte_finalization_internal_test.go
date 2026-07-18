package pdf

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestGopdfDocumentBytesReturnsPayloadAndNoError verifies successful PDF byte
// finalization through the concrete adapter.
// Authored by: OpenCode
func TestGopdfDocumentBytesReturnsPayloadAndNoError(t *testing.T) {
	var document = startedTestDocument(t)

	var payload, err = document.Bytes()
	if err != nil {
		t.Fatalf("finalize PDF: %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF-")) {
		t.Fatalf("expected valid PDF payload, got %q", payload)
	}

	var previousFinalize = finalizeGopdfDocument
	t.Cleanup(func() { finalizeGopdfDocument = previousFinalize })
	finalizeGopdfDocument = func(*gopdfDocument) ([]byte, error) {
		return []byte("partial"), errors.New("synthetic byte finalization failure")
	}
	if failedPayload, finalizeErr := document.Bytes(); finalizeErr == nil || failedPayload != nil {
		t.Fatalf("failed finalization returned payload=%q error=%v", failedPayload, finalizeErr)
	}
}

// TestRendererFinalizationFailureReturnsNormallyWithoutPartialPayload verifies
// that injected finalization errors discard partial bytes and return normally.
// Authored by: OpenCode
func TestRendererFinalizationFailureReturnsNormallyWithoutPartialPayload(t *testing.T) {
	var previousDocument = newPDFDocumentForRenderer
	defer func() { newPDFDocumentForRenderer = previousDocument }()

	var finalizationErr = errors.New("synthetic PDF finalization failure")
	var document = &failingLayoutDocument{
		bytesPayload: []byte("%PDF-partial"),
		bytesErr:     finalizationErr,
	}
	newPDFDocumentForRenderer = func() pdfLayoutDocument { return document }

	var renderer, rendererErr = NewRenderer(RenderOptions{Fonts: FontData{Regular: []byte("regular"), Bold: []byte("bold")}})
	if rendererErr != nil {
		t.Fatalf("new renderer: %v", rendererErr)
	}

	var payload []byte
	payload, rendererErr = renderer.Render(minimalPDFReportFixture(t))
	if rendererErr == nil {
		t.Fatal("expected PDF finalization to return an error")
	}
	var errorText = strings.ToLower(rendererErr.Error())
	if !strings.Contains(errorText, "pdf") || !strings.Contains(errorText, "finaliz") {
		t.Fatalf("finalization error = %v, want PDF finalization context", rendererErr)
	}
	if !errors.Is(rendererErr, finalizationErr) {
		t.Fatalf("finalization error = %v, want injected cause %v", rendererErr, finalizationErr)
	}
	if payload != nil {
		t.Fatalf("payload = %q, want nil after failed finalization", payload)
	}
	if document.bytesCalls != 1 {
		t.Fatalf("finalization calls = %d, want one completed render attempt", document.bytesCalls)
	}
}
