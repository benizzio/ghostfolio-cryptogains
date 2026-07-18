package pdf

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
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

	var failedDocument = startedTestDocumentWithFinalizer(t, func(func() ([]byte, error)) ([]byte, error) {
		return []byte("partial"), errors.New("synthetic byte finalization failure")
	})
	if failedPayload, finalizeErr := failedDocument.Bytes(); finalizeErr == nil || failedPayload != nil {
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
	newPDFDocumentForRenderer = func(ByteFinalizer) pdfLayoutDocument { return document }

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

// TestRendererByteFinalizerOptionPreservesCauseIdentity verifies the concrete
// renderer-scoped option redacts the displayed cause while preserving errors.Is.
// Authored by: OpenCode
func TestRendererByteFinalizerOptionPreservesCauseIdentity(t *testing.T) {
	var finalizationErr = errors.New("Bearer synthetic-renderer-finalization-secret")
	var renderer, err = NewRenderer(RenderOptions{
		Fonts: FontData{Regular: goregular.TTF, Bold: gobold.TTF},
		ByteFinalizer: func(func() ([]byte, error)) ([]byte, error) {
			return []byte("%PDF-partial"), finalizationErr
		},
	})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}

	var payload []byte
	payload, err = renderer.Render(minimalPDFReportFixture(t))
	if err == nil || payload != nil {
		t.Fatalf("expected nil payload and finalization error, payload=%q error=%v", payload, err)
	}
	if !errors.Is(err, finalizationErr) {
		t.Fatalf("finalization error = %v, want injected cause %v", err, finalizationErr)
	}
	if !strings.Contains(err.Error(), "Bearer [REDACTED]") || strings.Contains(err.Error(), finalizationErr.Error()) {
		t.Fatalf("expected redacted finalization error, got %v", err)
	}
}
