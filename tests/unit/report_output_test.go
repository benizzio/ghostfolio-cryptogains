// Package unit verifies focused report-output helpers that can be exercised
// without the full report runtime orchestration.
// Authored by: OpenCode
package unit

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// TestResolveDocumentsDirectoryForOSPrefersLinuxXDG verifies Linux XDG user-dir
// resolution ahead of the fallback Documents path.
// Authored by: OpenCode
func TestResolveDocumentsDirectoryForOSPrefersLinuxXDG(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var customDocumentsDir = filepath.Join(fixture.BaseDir, "xdg-documents")
	fixture.SetXDGDocumentsDir(t, customDocumentsDir)

	var documentsDir, err = reportoutput.ResolveDocumentsDirectoryForOS("linux")
	if err != nil {
		t.Fatalf("resolve Linux documents directory: %v", err)
	}
	if documentsDir != customDocumentsDir {
		t.Fatalf("unexpected Linux documents directory: got %q want %q", documentsDir, customDocumentsDir)
	}
}

// TestResolveDocumentsDirectoryForOSFallsBackToHomeDocuments verifies Linux
// fallback behavior when no XDG documents entry is configured.
// Authored by: OpenCode
func TestResolveDocumentsDirectoryForOSFallsBackToHomeDocuments(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)

	var documentsDir, err = reportoutput.ResolveDocumentsDirectoryForOS("linux")
	if err != nil {
		t.Fatalf("resolve Linux fallback documents directory: %v", err)
	}
	if documentsDir != fixture.DocumentsDir {
		t.Fatalf("unexpected Linux fallback documents directory: got %q want %q", documentsDir, fixture.DocumentsDir)
	}
}

// TestResolveDocumentsDirectoryForOSUsesMacOSHomeDocuments verifies the macOS
// home-directory convention.
// Authored by: OpenCode
func TestResolveDocumentsDirectoryForOSUsesMacOSHomeDocuments(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var expected = filepath.Join(fixture.HomeDir, "Documents")

	var documentsDir, err = reportoutput.ResolveDocumentsDirectoryForOS("darwin")
	if err != nil {
		t.Fatalf("resolve macOS documents directory: %v", err)
	}
	if documentsDir != expected {
		t.Fatalf("unexpected macOS documents directory: got %q want %q", documentsDir, expected)
	}
}

// TestResolveDocumentsDirectoryForOSUsesWindowsUserProfileDocuments verifies
// the Windows user-profile convention.
// Authored by: OpenCode
func TestResolveDocumentsDirectoryForOSUsesWindowsUserProfileDocuments(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var expected = filepath.Join(fixture.HomeDir, "Documents")

	var documentsDir, err = reportoutput.ResolveDocumentsDirectoryForOS("windows")
	if err != nil {
		t.Fatalf("resolve Windows documents directory: %v", err)
	}
	if documentsDir != expected {
		t.Fatalf("unexpected Windows documents directory: got %q want %q", documentsDir, expected)
	}
}

// TestResolveDocumentsDirectoryForOSRejectsUnsupportedPlatform verifies failure
// for platforms outside the supported report-output set.
// Authored by: OpenCode
func TestResolveDocumentsDirectoryForOSRejectsUnsupportedPlatform(t *testing.T) {
	if _, err := reportoutput.ResolveDocumentsDirectoryForOS("plan9"); err == nil {
		t.Fatalf("expected unsupported platform resolution to fail")
	}
}

// TestWriteReportDocumentUsesTimestampedFilenameAndSuffix verifies deterministic
// filename construction and same-second suffix handling.
// Authored by: OpenCode
func TestWriteReportDocumentUsesTimestampedFilenameAndSuffix(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.Local)
	var documents = outputMarkdownDocuments(reportmodel.CostBasisMethodAverageCost, "# Report\n", generatedAt)

	var firstBundle, err = reportoutput.WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
	if err != nil {
		t.Fatalf("write first report document: %v", err)
	}
	var secondBundle reportmodel.ReportOutputBundle
	secondBundle, err = reportoutput.WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
	if err != nil {
		t.Fatalf("write second report document: %v", err)
	}
	var firstOutput = firstBundle.Files[0]
	var secondOutput = secondBundle.Files[0]

	var expectedFirst = "ghostfolio-capital-gains-2024-average-cost-2026-05-21_12-34-56.md"
	var expectedSecond = "ghostfolio-capital-gains-2024-average-cost-2026-05-21_12-34-56-2.md"
	if firstOutput.Filename != expectedFirst {
		t.Fatalf("unexpected first filename: got %q want %q", firstOutput.Filename, expectedFirst)
	}
	if secondOutput.Filename != expectedSecond {
		t.Fatalf("unexpected second filename: got %q want %q", secondOutput.Filename, expectedSecond)
	}

	testutil.AssertPathWithin(t, firstOutput.Path, fixture.DocumentsDir)
	testutil.AssertPathWithin(t, secondOutput.Path, fixture.DocumentsDir)
	testutil.AssertRegularFile(t, firstOutput.Path)
	testutil.AssertRegularFile(t, secondOutput.Path)
}

// TestWriteReportDocumentUsesExclusiveCreate verifies that an existing same-name
// file is preserved and the writer reserves a later suffix.
// Authored by: OpenCode
func TestWriteReportDocumentUsesExclusiveCreate(t *testing.T) {
	var fixture = testutil.NewReportIOFixture(t)
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.Local)
	var existingPath = fixture.ReportPath("ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56.md")
	testutil.WriteFixtureFile(t, existingPath, "existing")

	var bundle, err = reportoutput.WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, outputMarkdownDocuments(reportmodel.CostBasisMethodFIFO, "new", generatedAt))
	if err != nil {
		t.Fatalf("write report document with existing base path: %v", err)
	}
	var outputFile = bundle.Files[0]

	if outputFile.Filename != "ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-2.md" {
		t.Fatalf("unexpected suffixed filename: %q", outputFile.Filename)
	}
	testutil.AssertRegularFile(t, existingPath)
	testutil.AssertRegularFile(t, outputFile.Path)
	testutil.AssertFileContent(t, existingPath, "existing")
	testutil.AssertFileContent(t, outputFile.Path, "new")
}

// outputMarkdownDocuments builds the valid main-plus-annex document bundle used
// by report-output tests.
// Authored by: OpenCode
func outputMarkdownDocuments(method reportmodel.CostBasisMethod, content string, generatedAt time.Time) []reportmodel.ReportDocument {
	return []reportmodel.ReportDocument{
		{DocumentType: reportmodel.ReportDocumentTypeMarkdown, Role: reportmodel.ReportDocumentRoleMain, Content: content, Year: 2024, CostBasisMethod: method, GeneratedAt: generatedAt},
		{DocumentType: reportmodel.ReportDocumentTypeMarkdown, Role: reportmodel.ReportDocumentRoleAnnex, Content: "# Annex 1 - Audit\n", Year: 2024, CostBasisMethod: method, GeneratedAt: generatedAt},
	}
}

// TestResolveOpenCommandForOSReturnsExpectedCommands verifies the adapter
// command details without starting a subprocess.
// Authored by: OpenCode
func TestResolveOpenCommandForOSReturnsExpectedCommands(t *testing.T) {
	var reportPath = filepath.Join("/tmp", "report.md")
	testCases := []struct {
		name    string
		goos    string
		command reportoutput.OpenCommand
	}{
		{
			name:    "linux",
			goos:    "linux",
			command: reportoutput.OpenCommand{Name: "xdg-open", Args: []string{reportPath}},
		},
		{
			name:    "darwin",
			goos:    "darwin",
			command: reportoutput.OpenCommand{Name: "open", Args: []string{reportPath}},
		},
		{
			name:    "windows",
			goos:    "windows",
			command: reportoutput.OpenCommand{Name: "cmd", Args: []string{"/c", "start", "", reportPath}},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			command, err := reportoutput.ResolveOpenCommandForOS(testCase.goos, reportPath)
			if err != nil {
				t.Fatalf("resolve open command: %v", err)
			}
			if command.Name != testCase.command.Name {
				t.Fatalf("unexpected command name: got %q want %q", command.Name, testCase.command.Name)
			}
			if !reflect.DeepEqual(command.Args, testCase.command.Args) {
				t.Fatalf("unexpected command args: got %v want %v", command.Args, testCase.command.Args)
			}
		})
	}
}

// TestResolveOpenCommandForOSRejectsUnsupportedPlatform verifies failure for
// unsupported opener adapters.
// Authored by: OpenCode
func TestResolveOpenCommandForOSRejectsUnsupportedPlatform(t *testing.T) {
	if _, err := reportoutput.ResolveOpenCommandForOS("plan9", "/tmp/report.md"); err == nil {
		t.Fatalf("expected unsupported opener platform to fail")
	}
}
