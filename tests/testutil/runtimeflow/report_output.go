package runtimeflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
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

// ReportOutputPaths returns the complete saved-file set for one selected format
// and enforces its production bundle shape. For example, pass Markdown to get
// the main and Annex paths, or PDF to get the one combined-document path.
// Authored by: OpenCode
func ReportOutputPaths(t *testing.T, dir string, outputFormat reportmodel.ReportOutputFormat) []string {
	t.Helper()

	var files []string
	var expectedCount int
	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		files = AllMarkdownFiles(t, dir)
		expectedCount = 2
	case reportmodel.ReportOutputFormatPDF:
		files = PDFFiles(t, dir)
		expectedCount = 1
	default:
		t.Fatalf("unsupported report output format %q", outputFormat)
	}
	if len(files) != expectedCount {
		t.Fatalf("expected %d saved %s output file(s), got %#v", expectedCount, outputFormat, files)
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

// AssertReportResultDisclosure verifies the TUI-owned cleartext disclosure,
// deletion guidance, format-specific bundle labels, and every saved path in a
// successful report result. For example, pass normalized result content and
// the two Markdown or one PDF paths returned by ReportOutputPaths.
// Authored by: OpenCode
func AssertReportResultDisclosure(t *testing.T, content string, outputFormat reportmodel.ReportOutputFormat, paths []string) {
	t.Helper()

	if strings.Count(content, component.ReportCleartextExportDisclosureText) != 1 {
		t.Fatalf("expected cleartext export disclosure exactly once, got %q", content)
	}
	if strings.Count(content, component.ReportCleartextExportDeletionGuidanceText) != 1 {
		t.Fatalf("expected export deletion guidance exactly once, got %q", content)
	}

	var expectedLabels = reportResultPathLabels(t, outputFormat, len(paths))
	assertReportResultLabels(t, content, expectedLabels)
	assertReportResultPathsOnce(t, content, paths)
}

// reportResultPathLabels returns the required path labels for one output bundle.
// Authored by: OpenCode
func reportResultPathLabels(t *testing.T, outputFormat reportmodel.ReportOutputFormat, pathCount int) []string {
	t.Helper()
	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		if pathCount != 2 {
			t.Fatalf("expected two Markdown paths, got %d", pathCount)
		}
		return []string{"Saved Markdown Path:", "Saved Annex 1 Markdown Path:"}
	case reportmodel.ReportOutputFormatPDF:
		if pathCount != 1 {
			t.Fatalf("expected one PDF path, got %d", pathCount)
		}
		return []string{"Saved PDF Path:"}
	default:
		t.Fatalf("unsupported report output format %q", outputFormat)
		return nil
	}
}

// assertReportResultLabels checks each format-specific saved-path label once.
// Authored by: OpenCode
func assertReportResultLabels(t *testing.T, content string, labels []string) {
	t.Helper()
	for _, label := range labels {
		if strings.Count(content, label) != 1 {
			t.Fatalf("expected result label %q exactly once, got %q", label, content)
		}
	}
}

// assertReportResultPathsOnce checks unique saved paths despite TUI wrapping.
// Authored by: OpenCode
func assertReportResultPathsOnce(t *testing.T, content string, paths []string) {
	t.Helper()
	var seen = make(map[string]struct{}, len(paths))
	var compactContent = strings.Join(strings.Fields(content), "")
	for _, path := range paths {
		if path == "" {
			t.Fatal("expected saved report path to be non-empty")
		}
		if _, ok := seen[path]; ok {
			t.Fatalf("expected saved report paths to be unique, got %#v", paths)
		}
		seen[path] = struct{}{}
		if strings.Count(compactContent, path) != 1 {
			t.Fatalf("expected saved report path %q exactly once, got %q", path, content)
		}
	}
}

// AssertReportResultCleared verifies that leaving a report result clears its
// prior paths and TUI-owned export copy. For example, pass the next screen's
// normalized content and the paths from the dismissed result.
// Authored by: OpenCode
func AssertReportResultCleared(t *testing.T, content string, paths []string) {
	t.Helper()

	if strings.Contains(content, component.ReportCleartextExportDisclosureText) {
		t.Fatalf("expected cleartext export disclosure to be cleared, got %q", content)
	}
	if strings.Contains(content, component.ReportCleartextExportDeletionGuidanceText) {
		t.Fatalf("expected export deletion guidance to be cleared, got %q", content)
	}
	for _, path := range paths {
		if strings.Contains(content, path) {
			t.Fatalf("expected prior saved path %q to be cleared, got %q", path, content)
		}
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
