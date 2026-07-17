// Package contract verifies the report rendering confidentiality boundary.
// Authored by: OpenCode
package contract

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	runtimeapp "github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	reportmarkdown "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportpdf "github.com/benizzio/ghostfolio-cryptogains/internal/report/pdf"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	// syntheticCredentialSentinel is a non-reusable credential-shaped redaction probe.
	// Authored by: OpenCode
	syntheticCredentialSentinel = "SYNTHETIC_CREDENTIAL_NOT_REUSABLE_20260717"
	// syntheticProtectedPayloadSentinel is a non-reusable protected-payload redaction probe.
	// Authored by: OpenCode
	syntheticProtectedPayloadSentinel = "SYNTHETIC_PROTECTED_PAYLOAD_NOT_REUSABLE_20260717"
	// syntheticFinancialSentinel is a non-user financial-data redaction probe.
	// Authored by: OpenCode
	syntheticFinancialSentinel = "SYNTHETIC_FINANCIAL_FIELD_NOT_REUSABLE_20260717"
	// syntheticExportFinancialAmount is the synthetic financial value allowed in exports.
	// Authored by: OpenCode
	syntheticExportFinancialAmount = "424242.42"
	// syntheticDelimiterValue probes dynamic delimiter sanitization in both renderers.
	// Authored by: OpenCode
	syntheticDelimiterValue = "DYN-INJECT;<br>|new\nline"
)

// TestReportRenderingConfidentialityContract verifies that contracted export
// fields may contain synthetic financial values, while synthetic credential and
// protected-payload material is absent from both Markdown and PDF documents.
// Authored by: OpenCode
func TestReportRenderingConfidentialityContract(t *testing.T) {
	t.Parallel()

	var report = confidentialityReportFixture()
	var exportAmount = mustContractDecimal(syntheticExportFinancialAmount)
	report.SummaryEntries[0].NetGainOrLoss = exportAmount
	report.YearlyNetTotal = exportAmount
	report.AuditAnnex.ConversionAuditEntries[0].Amounts[0].OriginalAmount = exportAmount
	report.AuditAnnex.ConversionAuditEntries[0].Amounts[0].ConvertedAmount = exportAmount
	report.AuditAnnex.PerAssetAuditSections[0].Entries[0].Note = fmt.Sprintf("token=%s payload=%s", syntheticCredentialSentinel, syntheticProtectedPayloadSentinel)

	var documents, err = reportmarkdown.RenderDocuments(report)
	if err != nil {
		t.Fatalf("render confidential Markdown documents: %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("Markdown document count = %d, want 2", len(documents))
	}
	for index, document := range documents {
		var content = string(document.Content)
		assertConfidentialitySentinelsAbsent(t, content, "Markdown document")
		if !strings.Contains(content, syntheticExportFinancialAmount) {
			t.Fatalf("Markdown document %d omitted contracted synthetic financial value %q", index, syntheticExportFinancialAmount)
		}
	}

	var renderer reportpdf.Renderer
	renderer, err = reportpdf.NewRenderer(reportpdf.RenderOptions{Fonts: reportpdf.FontData{Regular: goregular.TTF, Bold: gobold.TTF}})
	if err != nil {
		t.Fatalf("create confidential PDF renderer: %v", err)
	}
	var payload []byte
	payload, err = renderer.Render(report)
	if err != nil {
		t.Fatalf("render confidential PDF: %v", err)
	}
	var inspection testutil.GeneratedPDF
	inspection, err = testutil.InspectGeneratedPDF(payload)
	if err != nil {
		t.Fatalf("inspect confidential PDF: %v", err)
	}
	assertConfidentialitySentinelsAbsent(t, inspection.SearchableText, "PDF document")
	if !inspection.ContainsSearchableText(syntheticExportFinancialAmount) {
		t.Fatalf("PDF omitted contracted synthetic financial value %q", syntheticExportFinancialAmount)
	}
}

// TestReportRenderingConfidentialityErrors verifies that returned and wrapped
// Markdown/PDF render errors do not disclose values supplied in an invalid
// report field and return no document payload.
// Authored by: OpenCode
func TestReportRenderingConfidentialityErrors(t *testing.T) {
	t.Parallel()

	var report = confidentialityReportFixture()
	report.ReportCalculationCurrency = fmt.Sprintf("token=%s payload=%s financial=%s", syntheticCredentialSentinel, syntheticProtectedPayloadSentinel, syntheticFinancialSentinel)

	var markdownDocument, markdownErr = reportmarkdown.Render(report)
	if markdownErr == nil {
		t.Fatal("Markdown accepted a report field containing prohibited synthetic material")
	}
	if len(markdownDocument.Content) != 0 {
		t.Fatalf("Markdown returned content after confidentiality failure: %d bytes", len(markdownDocument.Content))
	}
	assertConfidentialitySentinelsAbsent(t, markdownErr.Error(), "returned Markdown render error")
	assertConfidentialitySentinelsAbsent(t, fmt.Errorf("wrapped Markdown render error: %w", markdownErr).Error(), "wrapped Markdown render error")

	var renderer reportpdf.Renderer
	var err error
	renderer, err = reportpdf.NewRenderer(reportpdf.RenderOptions{Fonts: reportpdf.FontData{Regular: goregular.TTF, Bold: gobold.TTF}})
	if err != nil {
		t.Fatalf("create PDF renderer: %v", err)
	}
	var payload []byte
	payload, err = renderer.Render(report)
	if err == nil {
		t.Fatal("PDF accepted a report field containing prohibited synthetic material")
	}
	if payload != nil {
		t.Fatalf("PDF returned payload after confidentiality failure: %d bytes", len(payload))
	}
	assertConfidentialitySentinelsAbsent(t, err.Error(), "returned PDF render error")
	assertConfidentialitySentinelsAbsent(t, fmt.Errorf("wrapped PDF render error: %w", err).Error(), "wrapped PDF render error")
}

// TestReportRenderingConfidentialityDiagnostics verifies that a generated
// non-export diagnostic redacts credential, protected-payload, and financial
// sentinels while retaining the diagnostic's explicit redaction marker.
// Authored by: OpenCode
func TestReportRenderingConfidentialityDiagnostics(t *testing.T) {
	t.Parallel()

	var service = runtimeapp.NewSyncService(nil, 0, t.TempDir(), false, nil, nil, nil, nil)
	var path, err = service.GenerateDiagnosticReport(context.Background(), runtimeapp.DiagnosticReportRequest{
		FailureReason: runtimeapp.SyncFailureUnsupportedActivityHistory,
		ServerOrigin:  "https://synthetic.invalid",
		Attempt: runtimeapp.SyncAttempt{
			AttemptID:   "synthetic-confidentiality-attempt",
			Status:      runtimeapp.AttemptStatusFailed,
			StartedAt:   time.Unix(1, 0).UTC(),
			CompletedAt: time.Unix(2, 0).UTC(),
		},
		Context: syncmodel.DiagnosticContext{
			FailureDetail: fmt.Sprintf("render failed token=%s payload=%s", syntheticCredentialSentinel, syntheticProtectedPayloadSentinel),
			FailureCauseChain: []string{
				fmt.Sprintf("wrapped token=%s", syntheticCredentialSentinel),
				fmt.Sprintf("payload=%s", syntheticProtectedPayloadSentinel),
			},
			Records: []syncmodel.DiagnosticRecord{{
				SourceID:        "synthetic-diagnostic-record",
				Quantity:        syntheticFinancialSentinel,
				OrderUnitPrice:  syntheticFinancialSentinel,
				OrderGrossValue: syntheticFinancialSentinel,
				OrderFeeAmount:  syntheticFinancialSentinel,
				Comment:         "synthetic diagnostic context",
			}},
		},
		RedactFinancialValues: true,
	})
	if err != nil {
		t.Fatalf("generate synthetic diagnostic report: %v", err)
	}
	// #nosec G304 -- path is returned by the test-owned diagnostic writer.
	var raw, readErr = os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read synthetic diagnostic report: %v", readErr)
	}
	var content = string(raw)
	assertConfidentialitySentinelsAbsent(t, content, "generated diagnostic report")
	if !strings.Contains(content, `"financial_values_redacted": true`) {
		t.Fatalf("diagnostic report omitted its financial-redaction marker: %q", content)
	}
}

// TestReportRenderingDelimiterInjectionContract verifies that dynamic report
// text retains content without adding Markdown table pipes, Markdown conversion
// breaks, PDF conversion newlines, or arbitrary PDF newlines.
// Authored by: OpenCode
func TestReportRenderingDelimiterInjectionContract(t *testing.T) {
	t.Parallel()

	var report = confidentialityReportFixture()
	report.AuditAnnex.ConversionAuditEntries[0].SourceID = syntheticDelimiterValue
	report.AuditAnnex.ConversionAuditEntries[0].AssetLabel = syntheticDelimiterValue
	for index := range report.AuditAnnex.ConversionAuditEntries[0].Amounts {
		report.AuditAnnex.ConversionAuditEntries[0].Amounts[index].SourceID = syntheticDelimiterValue
	}

	var documents, err = reportmarkdown.RenderDocuments(report)
	if err != nil {
		t.Fatalf("render delimiter Markdown documents: %v", err)
	}
	var annex = string(documents[1].Content)
	var conversionRow = markdownConversionAuditRow(t, annex)
	var expectedMarkdownDynamic = strings.ReplaceAll(syntheticDelimiterValue, "|", "\\|")
	expectedMarkdownDynamic = strings.ReplaceAll(expectedMarkdownDynamic, "\n", " ")
	if !strings.Contains(conversionRow, expectedMarkdownDynamic) {
		t.Fatalf("Markdown lost dynamic content: row=%q want fragment=%q", conversionRow, expectedMarkdownDynamic)
	}
	if countUnescapedMarkdownPipes(conversionRow) != 10 {
		t.Fatalf("Markdown dynamic pipe changed table structure: row=%q", conversionRow)
	}
	var convertedCell = markdownConversionCell(conversionRow)
	if strings.Count(convertedCell, ";<br>") != 1 {
		t.Fatalf("Markdown dynamic value injected a conversion break: cell=%q", convertedCell)
	}
	if !strings.Contains(convertedCell, "unit_price: 27000.00 -> 25000.00") || !strings.Contains(convertedCell, "gross_value: 27000.00 -> 25000.00") {
		t.Fatalf("Markdown conversion content was not preserved: cell=%q", convertedCell)
	}

	var renderer reportpdf.Renderer
	renderer, err = reportpdf.NewRenderer(reportpdf.RenderOptions{Fonts: reportpdf.FontData{Regular: goregular.TTF, Bold: gobold.TTF}})
	if err != nil {
		t.Fatalf("create delimiter PDF renderer: %v", err)
	}
	var payload []byte
	payload, err = renderer.Render(report)
	if err != nil {
		t.Fatalf("render delimiter PDF: %v", err)
	}
	var inspection testutil.GeneratedPDF
	inspection, err = testutil.InspectGeneratedPDF(payload)
	if err != nil {
		t.Fatalf("inspect delimiter PDF: %v", err)
	}
	var pdfText = strings.Join(pdfTextRunStrings(inspection), " ")
	var expectedPDFDynamic = strings.ReplaceAll(syntheticDelimiterValue, "|", "/")
	expectedPDFDynamic = strings.ReplaceAll(expectedPDFDynamic, "\n", " ")
	if !inspection.ContainsSearchableText(expectedPDFDynamic) {
		t.Fatalf("PDF lost dynamic content: text=%q want fragment=%q", pdfText, expectedPDFDynamic)
	}
	if strings.Contains(pdfText, "|") {
		t.Fatalf("PDF dynamic value retained a structural pipe: text=%q", pdfText)
	}
	for _, run := range inspection.TextRuns {
		if strings.Contains(run.Text, "DYN-INJECT") && strings.Contains(run.Text, "\n") {
			t.Fatalf("PDF dynamic value retained an arbitrary renderer newline: run=%q", run.Text)
		}
	}
	if countPDFConversionBoundaries(inspection) != 1 {
		t.Fatalf("PDF dynamic value changed the controlled conversion boundary count: runs=%#v", inspection.TextRuns)
	}
}

// confidentialityReportFixture returns a fully synthetic report with both
// Markdown documents and a combined PDF Annex input.
// Authored by: OpenCode
func confidentialityReportFixture() reportmodel.CapitalGainsReport {
	var report = contractMarkdownReportFixture(reportmodel.ReportBaseCurrencyEUR.Label())
	report.AuditAnnex = contractDetailedAuditAnnex()
	return report
}

// assertConfidentialitySentinelsAbsent rejects synthetic material from one
// non-export or redacted output channel.
// Authored by: OpenCode
func assertConfidentialitySentinelsAbsent(t *testing.T, content string, channel string) {
	t.Helper()
	for _, sentinel := range []string{syntheticCredentialSentinel, syntheticProtectedPayloadSentinel, syntheticFinancialSentinel} {
		if strings.Contains(content, sentinel) {
			t.Errorf("%s contains prohibited synthetic sentinel %q: %q", channel, sentinel, content)
		}
	}
}

// markdownConversionAuditRow extracts the one conversion row from a synthetic
// Annex document.
// Authored by: OpenCode
func markdownConversionAuditRow(t *testing.T, annex string) string {
	t.Helper()
	for _, line := range strings.Split(annex, "\n") {
		if strings.Contains(line, "| 2024-01-01 |") {
			return line
		}
	}
	t.Fatalf("Markdown conversion audit row is missing: %q", annex)
	return ""
}

// markdownConversionCell extracts the Converted Amounts cell without treating
// escaped dynamic pipes as table separators.
// Authored by: OpenCode
func markdownConversionCell(row string) string {
	var cells []string
	var current strings.Builder
	var escaped bool
	for index := 0; index < len(row); index++ {
		var character = row[index]
		if character == '|' && !escaped {
			cells = append(cells, current.String())
			current.Reset()
			escaped = false
			continue
		}
		current.WriteByte(character)
		if character == '\\' && !escaped {
			escaped = true
			continue
		}
		escaped = false
	}
	cells = append(cells, current.String())
	if len(cells) < 8 {
		return ""
	}
	return strings.TrimSpace(cells[7])
}

// countUnescapedMarkdownPipes counts the structural pipes in one table row.
// Authored by: OpenCode
func countUnescapedMarkdownPipes(row string) int {
	var count int
	var escaped bool
	for index := 0; index < len(row); index++ {
		var character = row[index]
		if character == '|' && !escaped {
			count++
		}
		if character == '\\' && !escaped {
			escaped = true
			continue
		}
		escaped = false
	}
	return count
}

// pdfTextRunStrings returns decoded PDF text runs in document order.
// Authored by: OpenCode
func pdfTextRunStrings(inspection testutil.GeneratedPDF) []string {
	var texts = make([]string, 0, len(inspection.TextRuns))
	for _, run := range inspection.TextRuns {
		texts = append(texts, run.Text)
	}
	return texts
}

// countPDFConversionBoundaries counts the semicolon/newline boundary in the
// converted-amount text runs while ignoring ordinary PDF run separators.
// Authored by: OpenCode
func countPDFConversionBoundaries(inspection testutil.GeneratedPDF) int {
	var count int
	for index, run := range inspection.TextRuns {
		if !strings.Contains(run.Text, ";") {
			continue
		}
		for next := index + 1; next < len(inspection.TextRuns); next++ {
			if strings.TrimSpace(inspection.TextRuns[next].Text) == "" {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(inspection.TextRuns[next].Text), "unit_price:") {
				count++
			}
			break
		}
	}
	return count
}
