// Package contract verifies rendered workflow and Ghostfolio-boundary contracts
// for the sync-and-storage slice.
// Authored by: OpenCode
package contract

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// TestReportOutputFormatFileCountsContract verifies the supported output formats
// and their required successful-output file counts.
// Authored by: OpenCode
func TestReportOutputFormatFileCountsContract(t *testing.T) {
	t.Parallel()

	var fixtures = testutil.DeterministicReportOutputFormatFixtures()
	if len(fixtures) != 2 {
		t.Fatalf("expected exactly two supported output formats, got %d", len(fixtures))
	}

	var expected = map[reportmodel.ReportOutputFormat]struct {
		code       string
		label      string
		fileCount  int
		extensions []string
	}{
		reportmodel.ReportOutputFormatMarkdown: {code: "markdown", label: "Markdown", fileCount: 2, extensions: []string{".md", ".md"}},
		reportmodel.ReportOutputFormatPDF:      {code: "pdf", label: "PDF", fileCount: 1, extensions: []string{".pdf"}},
	}

	for _, fixture := range fixtures {
		var expectation, ok = expected[fixture.Format]
		if !ok {
			t.Fatalf("unexpected output format fixture: %#v", fixture)
		}
		if fixture.Code != expectation.code || fixture.Label != expectation.label || fixture.FileCount != expectation.fileCount {
			t.Fatalf("unexpected output format fixture: got %#v want code=%q label=%q files=%d", fixture, expectation.code, expectation.label, expectation.fileCount)
		}
		if strings.Join(fixture.Extensions, ",") != strings.Join(expectation.extensions, ",") {
			t.Fatalf("unexpected extensions for %q: got %v want %v", fixture.Format, fixture.Extensions, expectation.extensions)
		}
	}
}

// TestReportOutputFilenamePatternContract verifies canonical unsuffixed and
// collided filename patterns for Markdown and PDF outputs.
// Authored by: OpenCode
func TestReportOutputFilenamePatternContract(t *testing.T) {
	t.Parallel()

	var filenames = testutil.DeterministicReportOutputFilenameFixture()
	assertReportFilenamePattern(t, filenames.MarkdownMainFilename, `^ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56\.md$`)
	assertReportFilenamePattern(t, filenames.MarkdownAnnexFilename, `^ghostfolio-capital-gains-2024-fifo-annex-1-2026-05-21_12-34-56\.md$`)
	assertReportFilenamePattern(t, filenames.PDFCombinedFilename, `^ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56\.pdf$`)
	assertReportFilenamePattern(t, filenames.CollidedMarkdownMain, `^ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-2\.md$`)
	assertReportFilenamePattern(t, filenames.CollidedMarkdownAnnex, `^ghostfolio-capital-gains-2024-fifo-annex-1-2026-05-21_12-34-56-2\.md$`)
	assertReportFilenamePattern(t, filenames.CollidedPDFCombined, `^ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-2\.pdf$`)
}

// TestReportOutputBundleShapeContract verifies successful output bundle metadata
// for Markdown main-plus-annex output and combined PDF output.
// Authored by: OpenCode
func TestReportOutputBundleShapeContract(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var bundles = fixture.DeterministicReportOutputBundleFixture(t)

	assertReportOutputBundleShape(t, bundles.MarkdownBundle, reportmodel.ReportOutputFormatMarkdown, []reportmodel.ReportDocumentRole{
		reportmodel.ReportDocumentRoleMain,
		reportmodel.ReportDocumentRoleAnnex,
	}, []string{
		reportmodel.ReportMediaTypeMarkdown,
		reportmodel.ReportMediaTypeMarkdown,
	})
	assertReportOutputBundleShape(t, bundles.PDFBundle, reportmodel.ReportOutputFormatPDF, []reportmodel.ReportDocumentRole{
		reportmodel.ReportDocumentRoleCombined,
	}, []string{
		reportmodel.ReportMediaTypePDF,
	})
}

// TestReportOutputWritesMarkdownPairContract verifies Markdown output creates
// exactly one main file and one Annex 1 file with matching timestamp metadata.
// Authored by: OpenCode
func TestReportOutputWritesMarkdownPairContract(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var request = testutil.DeterministicReportRequestFixture(reportmodel.ReportOutputFormatMarkdown)
	var documents = deterministicMarkdownOutputDocuments(t, request)

	var bundle, err = reportoutput.WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
	if err != nil {
		t.Fatalf("write markdown output bundle: %v", err)
	}

	assertReportOutputBundleShape(t, bundle, reportmodel.ReportOutputFormatMarkdown, []reportmodel.ReportDocumentRole{
		reportmodel.ReportDocumentRoleMain,
		reportmodel.ReportDocumentRoleAnnex,
	}, []string{
		reportmodel.ReportMediaTypeMarkdown,
		reportmodel.ReportMediaTypeMarkdown,
	})
	assertReportOutputFile(t, bundle.Files[0], fixture.DocumentsDir, `^ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56\.md$`)
	assertReportOutputFile(t, bundle.Files[1], fixture.DocumentsDir, `^ghostfolio-capital-gains-2024-fifo-annex-1-2026-05-21_12-34-56\.md$`)
	testutil.AssertFileContent(t, bundle.Files[0].Path, "# Main Report\n")
	testutil.AssertFileContent(t, bundle.Files[1].Path, "# Annex 1 - Audit\n")
}

// TestReportOutputUsesPairedMarkdownSuffixContract verifies Markdown collision
// handling reserves the same suffix for the main and Annex 1 files.
// Authored by: OpenCode
func TestReportOutputUsesPairedMarkdownSuffixContract(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	fixture.WriteDeterministicReportOutputCollisions(t)
	var request = testutil.DeterministicReportRequestFixture(reportmodel.ReportOutputFormatMarkdown)
	var documents = deterministicMarkdownOutputDocuments(t, request)

	var bundle, err = reportoutput.WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
	if err != nil {
		t.Fatalf("write collided markdown output bundle: %v", err)
	}

	assertReportOutputBundleShape(t, bundle, reportmodel.ReportOutputFormatMarkdown, []reportmodel.ReportDocumentRole{
		reportmodel.ReportDocumentRoleMain,
		reportmodel.ReportDocumentRoleAnnex,
	}, []string{
		reportmodel.ReportMediaTypeMarkdown,
		reportmodel.ReportMediaTypeMarkdown,
	})
	assertReportOutputFile(t, bundle.Files[0], fixture.DocumentsDir, `^ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-2\.md$`)
	assertReportOutputFile(t, bundle.Files[1], fixture.DocumentsDir, `^ghostfolio-capital-gains-2024-fifo-annex-1-2026-05-21_12-34-56-2\.md$`)
}

// TestReportOutputWritesPDFContract verifies PDF output creates exactly one
// combined report file with the PDF suffix rules.
// Authored by: OpenCode
func TestReportOutputWritesPDFContract(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var request = testutil.DeterministicReportRequestFixture(reportmodel.ReportOutputFormatPDF)
	var document = deterministicPDFOutputDocument(t, request)

	var bundle, err = reportoutput.WriteReportOutputBundle(reportmodel.ReportOutputFormatPDF, []reportmodel.ReportDocument{document})
	if err != nil {
		t.Fatalf("write PDF output bundle: %v", err)
	}

	assertReportOutputBundleShape(t, bundle, reportmodel.ReportOutputFormatPDF, []reportmodel.ReportDocumentRole{
		reportmodel.ReportDocumentRoleCombined,
	}, []string{
		reportmodel.ReportMediaTypePDF,
	})
	assertReportOutputFile(t, bundle.Files[0], fixture.DocumentsDir, `^ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56\.pdf$`)
	testutil.AssertFileContent(t, bundle.Files[0].Path, "%PDF-1.7\n")
}

// TestReportOutputUsesPDFSuffixContract verifies PDF collision handling appends
// the numeric suffix before the .pdf extension.
// Authored by: OpenCode
func TestReportOutputUsesPDFSuffixContract(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	fixture.WriteDeterministicReportOutputCollisions(t)
	var request = testutil.DeterministicReportRequestFixture(reportmodel.ReportOutputFormatPDF)
	var document = deterministicPDFOutputDocument(t, request)

	var bundle, err = reportoutput.WriteReportOutputBundle(reportmodel.ReportOutputFormatPDF, []reportmodel.ReportDocument{document})
	if err != nil {
		t.Fatalf("write collided PDF output bundle: %v", err)
	}

	assertReportOutputBundleShape(t, bundle, reportmodel.ReportOutputFormatPDF, []reportmodel.ReportDocumentRole{
		reportmodel.ReportDocumentRoleCombined,
	}, []string{
		reportmodel.ReportMediaTypePDF,
	})
	assertReportOutputFile(t, bundle.Files[0], fixture.DocumentsDir, `^ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-2\.pdf$`)
}

// deterministicMarkdownOutputDocuments builds the main and Annex 1 Markdown
// documents required by the output contract tests.
// Authored by: OpenCode
func deterministicMarkdownOutputDocuments(t *testing.T, request testutil.ReportRequestFixture) []reportmodel.ReportDocument {
	t.Helper()

	var mainDocument, err = reportmodel.NewReportDocument(
		reportmodel.ReportDocumentTypeMarkdown,
		reportmodel.ReportDocumentRoleMain,
		"# Main Report\n",
		request.Year,
		request.CostBasisMethod,
		request.RequestedAt,
	)
	if err != nil {
		t.Fatalf("build markdown main document: %v", err)
	}
	var annexDocument reportmodel.ReportDocument
	annexDocument, err = reportmodel.NewReportDocument(
		reportmodel.ReportDocumentTypeMarkdown,
		reportmodel.ReportDocumentRoleAnnex,
		"# Annex 1 - Audit\n",
		request.Year,
		request.CostBasisMethod,
		request.RequestedAt,
	)
	if err != nil {
		t.Fatalf("build markdown annex document: %v", err)
	}

	return []reportmodel.ReportDocument{mainDocument, annexDocument}
}

// deterministicPDFOutputDocument builds the combined PDF document required by
// the output contract tests.
// Authored by: OpenCode
func deterministicPDFOutputDocument(t *testing.T, request testutil.ReportRequestFixture) reportmodel.ReportDocument {
	t.Helper()

	var document, err = reportmodel.NewPDFReportDocument(
		reportmodel.ReportDocumentRoleCombined,
		[]byte("%PDF-1.7\n"),
		request.Year,
		request.CostBasisMethod,
		request.RequestedAt,
	)
	if err != nil {
		t.Fatalf("build PDF document: %v", err)
	}

	return document
}

// assertReportOutputBundleShape verifies the selected output format's file roles
// and media types.
// Authored by: OpenCode
func assertReportOutputBundleShape(t *testing.T, bundle reportmodel.ReportOutputBundle, outputFormat reportmodel.ReportOutputFormat, roles []reportmodel.ReportDocumentRole, mediaTypes []string) {
	t.Helper()

	if err := bundle.Validate(); err != nil {
		t.Fatalf("expected valid report output bundle: %v", err)
	}
	if bundle.OutputFormat != outputFormat {
		t.Fatalf("unexpected output format: got %q want %q", bundle.OutputFormat, outputFormat)
	}
	if len(bundle.Files) != len(roles) || len(bundle.Files) != len(mediaTypes) {
		t.Fatalf("unexpected output file count: got %d want %d", len(bundle.Files), len(roles))
	}
	for index, file := range bundle.Files {
		if file.Role != roles[index] {
			t.Fatalf("unexpected file role at index %d: got %q want %q", index, file.Role, roles[index])
		}
		if file.MediaType != mediaTypes[index] {
			t.Fatalf("unexpected media type at index %d: got %q want %q", index, file.MediaType, mediaTypes[index])
		}
	}
}

// assertReportOutputFile verifies one file's path locality, suffix, and filename
// pattern.
// Authored by: OpenCode
func assertReportOutputFile(t *testing.T, file reportmodel.ReportOutputFile, documentsDir string, pattern string) {
	t.Helper()

	assertReportFilenamePattern(t, file.Filename, pattern)
	if filepath.Ext(file.Filename) != filepath.Ext(file.Path) {
		t.Fatalf("filename and path extensions differ: filename=%q path=%q", file.Filename, file.Path)
	}
	testutil.AssertPathWithin(t, file.Path, documentsDir)
	testutil.AssertRegularFile(t, file.Path)
}

// assertReportFilenamePattern verifies one generated filename against a contract
// regular expression.
// Authored by: OpenCode
func assertReportFilenamePattern(t *testing.T, filename string, pattern string) {
	t.Helper()

	var matched, err = regexp.MatchString(pattern, filename)
	if err != nil {
		t.Fatalf("invalid filename pattern %q: %v", pattern, err)
	}
	if !matched {
		t.Fatalf("filename %q does not match pattern %q", filename, pattern)
	}
}
