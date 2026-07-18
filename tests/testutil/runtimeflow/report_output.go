package runtimeflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// MarkdownBundlePaths returns the main and Annex 1 paths from one complete
// Markdown output bundle. For example, pass AllMarkdownFiles output before
// inspecting either generated document.
// Authored by: OpenCode
func MarkdownBundlePaths(t *testing.T, files []string) (string, string) {
	t.Helper()
	if len(files) != 2 {
		t.Fatalf("expected exactly two Markdown output files, got %#v", files)
	}
	var mainPath string
	var annexPath string
	for _, file := range files {
		if strings.Contains(filepath.Base(file), "-annex-1-") {
			annexPath = file
		} else {
			mainPath = file
		}
	}
	if mainPath == "" || annexPath == "" {
		t.Fatalf("expected one main and one Annex 1 Markdown path, got %#v", files)
	}
	return mainPath, annexPath
}

// SelectedMainReportPath returns the first main output path for a selected
// report format. For example, pass Markdown files and ReportOutputFormatMarkdown
// when checking the primary report in a bundle.
// Authored by: OpenCode
func SelectedMainReportPath(t *testing.T, markdownFiles []string, pdfFiles []string, outputFormat reportmodel.ReportOutputFormat) string {
	t.Helper()
	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		if len(markdownFiles) == 0 {
			t.Fatalf("expected at least one Markdown main report")
		}
		return markdownFiles[0]
	case reportmodel.ReportOutputFormatPDF:
		if len(pdfFiles) == 0 {
			t.Fatalf("expected at least one PDF report")
		}
		return pdfFiles[0]
	default:
		t.Fatalf("unsupported report output format %q", outputFormat)
		return ""
	}
}

// PDFFiles returns all generated PDF files in one test-owned directory. For
// example, use it to assert that a failed render left no PDF artifact.
// Authored by: OpenCode
func PDFFiles(t *testing.T, dir string) []string {
	t.Helper()
	var entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pdf") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	return files
}

// AssertSavedMarkdownBundlePaths verifies that a result view reports both
// generated Markdown paths. For example, pass the normalized result-screen
// content and the paths returned by MarkdownBundlePaths.
// Authored by: OpenCode
func AssertSavedMarkdownBundlePaths(t *testing.T, content string, mainPath string, annexPath string) {
	t.Helper()
	if !strings.Contains(content, "Saved Markdown Path") || !strings.Contains(content, "Saved Annex 1 Markdown Path") {
		t.Fatalf("expected saved main and Annex 1 Markdown labels, got %q", content)
	}
	var compactContent = strings.Join(strings.Fields(content), "")
	if !strings.Contains(compactContent, mainPath) || !strings.Contains(compactContent, annexPath) {
		t.Fatalf("expected saved Markdown paths %q and %q, got %q", mainPath, annexPath, content)
	}
}

// AssertLandscapeA4PDF verifies that every generated PDF page has the
// production landscape A4 dimensions. For example, pass the result of
// testutil.InspectGeneratedPDF for the generated integration document.
// Authored by: OpenCode
func AssertLandscapeA4PDF(t *testing.T, inspection testutil.GeneratedPDF) {
	t.Helper()
	if len(inspection.PageBoxes) == 0 {
		t.Fatal("expected generated PDF to contain page boxes")
	}
	for index, page := range inspection.PageBoxes {
		if page.Width != 842 || page.Height != 595 {
			t.Fatalf("page %d dimensions = %.0fx%.0f, want landscape A4 842x595", index+1, page.Width, page.Height)
		}
	}
}
