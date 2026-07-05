// Package testutil provides reusable report-output fixtures for tests that need
// deterministic filesystem, clock, and opener behavior.
// Authored by: OpenCode
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// ReportIOFixture provides one controlled home-directory layout for report
// output tests that need a writable Documents directory and predictable
// environment variables.
// Authored by: OpenCode
type ReportIOFixture struct {
	BaseDir      string
	HomeDir      string
	DocumentsDir string
	XDGConfigDir string
}

// ReportOutputFilenameFixture stores deterministic names from the report-output
// filename contract.
// Authored by: OpenCode
type ReportOutputFilenameFixture struct {
	Year                  int
	CostBasisMethod       reportmodel.CostBasisMethod
	GeneratedAt           time.Time
	TimestampSlug         string
	MarkdownMainFilename  string
	MarkdownAnnexFilename string
	PDFCombinedFilename   string
	CollisionSuffix       int
	CollidedMarkdownMain  string
	CollidedMarkdownAnnex string
	CollidedPDFCombined   string
}

// ReportOutputPathFixture stores deterministic paths inside one fixture
// Documents directory.
// Authored by: OpenCode
type ReportOutputPathFixture struct {
	DocumentsDirectory    string
	MarkdownMainPath      string
	MarkdownAnnexPath     string
	PDFCombinedPath       string
	CollidedMarkdownMain  string
	CollidedMarkdownAnnex string
	CollidedPDFCombined   string
}

// ReportOutputBundleFixture stores validated Markdown and PDF output bundle
// defaults backed by deterministic fixture paths.
// Authored by: OpenCode
type ReportOutputBundleFixture struct {
	SavedAt        time.Time
	MarkdownFiles  []reportmodel.ReportOutputFile
	PDFFiles       []reportmodel.ReportOutputFile
	MarkdownBundle reportmodel.ReportOutputBundle
	PDFBundle      reportmodel.ReportOutputBundle
}

// NewReportIOFixture creates one temporary home-directory fixture with a
// writable Documents directory and environment variables that later output or
// runtime tests can reuse.
//
// Example usage:
//
//	fixture := testutil.NewReportIOFixture(t)
//	path := fixture.ReportPath("report.md")
//
// Authored by: OpenCode
func NewReportIOFixture(t *testing.T) ReportIOFixture {
	t.Helper()

	var baseDir = t.TempDir()
	var homeDir = filepath.Join(baseDir, "home")
	var documentsDir = filepath.Join(homeDir, "Documents")
	var xdgConfigDir = filepath.Join(homeDir, ".config")

	mustMkdirAll(t, documentsDir)
	mustMkdirAll(t, xdgConfigDir)

	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("XDG_CONFIG_HOME", xdgConfigDir)

	return ReportIOFixture{
		BaseDir:      baseDir,
		HomeDir:      homeDir,
		DocumentsDir: documentsDir,
		XDGConfigDir: xdgConfigDir,
	}
}

// ReportPath returns one path inside the fixture Documents directory so tests
// can assert report writes without duplicating path joins.
//
// Example usage:
//
//	fixture := testutil.NewReportIOFixture(t)
//	path := fixture.ReportPath("ghostfolio-capital-gains-2024-fifo.md")
//
// Authored by: OpenCode
func (fixture ReportIOFixture) ReportPath(name string) string {
	return filepath.Join(fixture.DocumentsDir, name)
}

// DeterministicReportOutputFilenameFixture returns canonical output filenames
// for the planned Markdown pair and combined PDF output contracts.
//
// Example usage:
//
//	filenames := testutil.DeterministicReportOutputFilenameFixture()
//	_ = filenames.MarkdownAnnexFilename
//
// Authored by: OpenCode
func DeterministicReportOutputFilenameFixture() ReportOutputFilenameFixture {
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var year = 2024
	var method = reportmodel.CostBasisMethodFIFO
	var timestampSlug = "2026-05-21_12-34-56"
	var prefix = reportOutputFilenamePrefix(year, method, timestampSlug)
	var annexPrefix = reportOutputAnnexFilenamePrefix(year, method, timestampSlug)
	const collisionSuffix = 2

	return ReportOutputFilenameFixture{
		Year:                  year,
		CostBasisMethod:       method,
		GeneratedAt:           generatedAt,
		TimestampSlug:         timestampSlug,
		MarkdownMainFilename:  prefix + ".md",
		MarkdownAnnexFilename: annexPrefix + ".md",
		PDFCombinedFilename:   prefix + ".pdf",
		CollisionSuffix:       collisionSuffix,
		CollidedMarkdownMain:  fmt.Sprintf("%s-%d.md", prefix, collisionSuffix),
		CollidedMarkdownAnnex: fmt.Sprintf("%s-%d.md", annexPrefix, collisionSuffix),
		CollidedPDFCombined:   fmt.Sprintf("%s-%d.pdf", prefix, collisionSuffix),
	}
}

// DeterministicReportOutputPathFixture returns canonical output paths under the
// fixture Documents directory.
//
// Example usage:
//
//	ioFixture := testutil.NewReportIOFixture(t)
//	paths := ioFixture.DeterministicReportOutputPathFixture()
//	_ = paths.PDFCombinedPath
//
// Authored by: OpenCode
func (fixture ReportIOFixture) DeterministicReportOutputPathFixture() ReportOutputPathFixture {
	var filenames = DeterministicReportOutputFilenameFixture()

	return ReportOutputPathFixture{
		DocumentsDirectory:    fixture.DocumentsDir,
		MarkdownMainPath:      fixture.ReportPath(filenames.MarkdownMainFilename),
		MarkdownAnnexPath:     fixture.ReportPath(filenames.MarkdownAnnexFilename),
		PDFCombinedPath:       fixture.ReportPath(filenames.PDFCombinedFilename),
		CollidedMarkdownMain:  fixture.ReportPath(filenames.CollidedMarkdownMain),
		CollidedMarkdownAnnex: fixture.ReportPath(filenames.CollidedMarkdownAnnex),
		CollidedPDFCombined:   fixture.ReportPath(filenames.CollidedPDFCombined),
	}
}

// DeterministicReportOutputBundleFixture returns validated output-file and
// bundle metadata for the canonical Markdown pair and combined PDF outputs.
//
// Example usage:
//
//	ioFixture := testutil.NewReportIOFixture(t)
//	bundles := ioFixture.DeterministicReportOutputBundleFixture(t)
//	_ = bundles.MarkdownBundle.OutputFormat
//
// Authored by: OpenCode
func (fixture ReportIOFixture) DeterministicReportOutputBundleFixture(t *testing.T) ReportOutputBundleFixture {
	t.Helper()

	var filenames = DeterministicReportOutputFilenameFixture()
	var paths = fixture.DeterministicReportOutputPathFixture()
	var savedAt = filenames.GeneratedAt

	var markdownMainFile, err = reportmodel.NewReportOutputFile(
		paths.DocumentsDirectory,
		filenames.MarkdownMainFilename,
		paths.MarkdownMainPath,
		reportmodel.ReportDocumentRoleMain,
		reportmodel.ReportMediaTypeMarkdown,
		savedAt,
	)
	if err != nil {
		t.Fatalf("build markdown main output fixture: %v", err)
	}
	var markdownAnnexFile reportmodel.ReportOutputFile
	markdownAnnexFile, err = reportmodel.NewReportOutputFile(
		paths.DocumentsDirectory,
		filenames.MarkdownAnnexFilename,
		paths.MarkdownAnnexPath,
		reportmodel.ReportDocumentRoleAnnex,
		reportmodel.ReportMediaTypeMarkdown,
		savedAt,
	)
	if err != nil {
		t.Fatalf("build markdown annex output fixture: %v", err)
	}
	var pdfCombinedFile reportmodel.ReportOutputFile
	pdfCombinedFile, err = reportmodel.NewReportOutputFile(
		paths.DocumentsDirectory,
		filenames.PDFCombinedFilename,
		paths.PDFCombinedPath,
		reportmodel.ReportDocumentRoleCombined,
		reportmodel.ReportMediaTypePDF,
		savedAt,
	)
	if err != nil {
		t.Fatalf("build pdf output fixture: %v", err)
	}

	var markdownFiles = []reportmodel.ReportOutputFile{markdownMainFile, markdownAnnexFile}
	var pdfFiles = []reportmodel.ReportOutputFile{pdfCombinedFile}
	var markdownBundle reportmodel.ReportOutputBundle
	markdownBundle, err = reportmodel.NewReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownFiles, savedAt, false, "")
	if err != nil {
		t.Fatalf("build markdown output bundle fixture: %v", err)
	}
	var pdfBundle reportmodel.ReportOutputBundle
	pdfBundle, err = reportmodel.NewReportOutputBundle(reportmodel.ReportOutputFormatPDF, pdfFiles, savedAt, false, "")
	if err != nil {
		t.Fatalf("build pdf output bundle fixture: %v", err)
	}

	return ReportOutputBundleFixture{
		SavedAt:        savedAt,
		MarkdownFiles:  markdownFiles,
		PDFFiles:       pdfFiles,
		MarkdownBundle: markdownBundle,
		PDFBundle:      pdfBundle,
	}
}

// SetXDGDocumentsDir writes one Linux user-dirs configuration that points the
// XDG Documents entry at the provided absolute path.
//
// Example usage:
//
//	fixture := testutil.NewReportIOFixture(t)
//	customDir := filepath.Join(fixture.BaseDir, "custom-documents")
//	fixture.SetXDGDocumentsDir(t, customDir)
//
// Authored by: OpenCode
func (fixture ReportIOFixture) SetXDGDocumentsDir(t *testing.T, documentsDir string) {
	t.Helper()

	if !filepath.IsAbs(documentsDir) {
		t.Fatalf("expected absolute XDG documents path, got %q", documentsDir)
	}

	mustMkdirAll(t, filepath.Dir(filepath.Join(fixture.XDGConfigDir, "user-dirs.dirs")))
	var configPath = filepath.Join(fixture.XDGConfigDir, "user-dirs.dirs")
	var configBody = fmt.Sprintf("XDG_DOCUMENTS_DIR=\"%s\"\n", escapeXDGDirValue(documentsDir))
	var err = os.WriteFile(configPath, []byte(configBody), 0o600)
	if err != nil {
		t.Fatalf("write XDG user-dirs config: %v", err)
	}
}

// OpenPathSpy records opener requests and can be configured to return one
// deterministic error for tests that need success and failure assertions.
// Authored by: OpenCode
type OpenPathSpy struct {
	mu    sync.Mutex
	err   error
	paths []string
}

// NewOpenPathSpy constructs one reusable opener spy that records all requested
// paths and returns the provided error on each call.
//
// Example usage:
//
//	spy := testutil.NewOpenPathSpy(nil)
//	err := spy.Open("/tmp/report.md")
//
// Authored by: OpenCode
func NewOpenPathSpy(err error) *OpenPathSpy {
	return &OpenPathSpy{err: err}
}

// Open records one opener request path and returns the spy's configured error.
//
// Example usage:
//
//	spy := testutil.NewOpenPathSpy(errors.New("open failed"))
//	err := spy.Open("/tmp/report.md")
//
// Authored by: OpenCode
func (spy *OpenPathSpy) Open(path string) error {
	spy.mu.Lock()
	defer spy.mu.Unlock()

	spy.paths = append(spy.paths, path)
	return spy.err
}

// CallCount returns how many opener requests the spy has observed.
//
// Example usage:
//
//	spy := testutil.NewOpenPathSpy(nil)
//	_ = spy.CallCount()
//
// Authored by: OpenCode
func (spy *OpenPathSpy) CallCount() int {
	spy.mu.Lock()
	defer spy.mu.Unlock()

	return len(spy.paths)
}

// Paths returns one copy of all opener request paths observed by the spy.
//
// Example usage:
//
//	spy := testutil.NewOpenPathSpy(nil)
//	_ = spy.Paths()
//
// Authored by: OpenCode
func (spy *OpenPathSpy) Paths() []string {
	spy.mu.Lock()
	defer spy.mu.Unlock()

	var paths = make([]string, len(spy.paths))
	copy(paths, spy.paths)
	return paths
}

// AssertPathWithin verifies that one path stays inside the expected parent
// directory.
//
// Example usage:
//
//	testutil.AssertPathWithin(t, reportPath, fixture.DocumentsDir)
//
// Authored by: OpenCode
func AssertPathWithin(t *testing.T, path string, parentDir string) {
	t.Helper()

	var relativePath, err = filepath.Rel(parentDir, path)
	if err != nil {
		t.Fatalf("relative path from %q to %q: %v", parentDir, path, err)
	}

	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) || filepath.IsAbs(relativePath) {
		t.Fatalf("expected %q to stay within %q, relative path %q escapes the directory", path, parentDir, relativePath)
	}
}

// AssertRegularFile verifies that one path exists and resolves to a regular
// file.
//
// Example usage:
//
//	testutil.AssertRegularFile(t, reportPath)
//
// Authored by: OpenCode
func AssertRegularFile(t *testing.T, path string) {
	t.Helper()

	var info, err = os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("expected %q to be a regular file, got mode %s", path, info.Mode())
	}
}

// WriteFixtureFile writes one deterministic test file so report-output tests can
// prepare existing target paths without duplicating boilerplate.
//
// Example usage:
//
//	testutil.WriteFixtureFile(t, fixture.ReportPath("report.md"), "content")
//
// Authored by: OpenCode
func WriteFixtureFile(t *testing.T, path string, content string) {
	t.Helper()

	mustMkdirAll(t, filepath.Dir(path))
	var err = os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("write fixture file %q: %v", path, err)
	}
}

// WriteDeterministicReportOutputCollisions creates canonical existing output
// files so writer tests can exercise collision suffix behavior.
//
// Example usage:
//
//	fixture := testutil.NewReportIOFixture(t)
//	fixture.WriteDeterministicReportOutputCollisions(t)
//
// Authored by: OpenCode
func (fixture ReportIOFixture) WriteDeterministicReportOutputCollisions(t *testing.T) {
	t.Helper()

	var paths = fixture.DeterministicReportOutputPathFixture()
	WriteFixtureFile(t, paths.MarkdownMainPath, "existing main markdown report\n")
	WriteFixtureFile(t, paths.MarkdownAnnexPath, "existing annex markdown report\n")
	WriteFixtureFile(t, paths.PDFCombinedPath, "%PDF-existing-report\n")
}

// AssertFileContent verifies one file's full text content.
//
// Example usage:
//
//	testutil.AssertFileContent(t, fixture.ReportPath("report.md"), "# Report\n")
//
// Authored by: OpenCode
func AssertFileContent(t *testing.T, path string, expected string) {
	t.Helper()

	//nolint:gosec // Test assertion reads the caller-provided report fixture path.
	var content, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	if string(content) != expected {
		t.Fatalf("unexpected file content for %q: got %q want %q", path, string(content), expected)
	}
}

// mustMkdirAll creates one directory tree for the report IO fixture.
// Authored by: OpenCode
func mustMkdirAll(t *testing.T, path string) {
	t.Helper()

	var err = os.MkdirAll(path, 0o750)
	if err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}

// escapeXDGDirValue escapes one path for the simple quoted XDG user-dirs file
// format.
// Authored by: OpenCode
func escapeXDGDirValue(path string) string {
	var escaped = strings.ReplaceAll(path, `\`, `\\`)
	return strings.ReplaceAll(escaped, `"`, `\"`)
}

// reportOutputFilenamePrefix returns the deterministic main report filename stem
// from the report-output contract.
// Authored by: OpenCode
func reportOutputFilenamePrefix(year int, method reportmodel.CostBasisMethod, timestampSlug string) string {
	return fmt.Sprintf("ghostfolio-capital-gains-%d-%s-%s", year, method.FilenameSlug(), timestampSlug)
}

// reportOutputAnnexFilenamePrefix returns the deterministic Annex 1 filename
// stem from the report-output contract.
// Authored by: OpenCode
func reportOutputAnnexFilenamePrefix(year int, method reportmodel.CostBasisMethod, timestampSlug string) string {
	return fmt.Sprintf("ghostfolio-capital-gains-%d-%s-annex-1-%s", year, method.FilenameSlug(), timestampSlug)
}
