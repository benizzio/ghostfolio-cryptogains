// Package output verifies report-output failure handling that depends on
// package-local test seams.
// Authored by: OpenCode
package output

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// TestWriteReportDocumentRemovesPartialFileOnWriteFailure verifies cleanup when
// content writing fails after exclusive file creation.
// Authored by: OpenCode
func TestWriteReportDocumentRemovesPartialFileOnWriteFailure(t *testing.T) {
	var fixtureDir = t.TempDir()
	var documentsDir = filepath.Join(fixtureDir, "Documents")
	if err := os.MkdirAll(documentsDir, 0o750); err != nil {
		t.Fatalf("mkdir documents: %v", err)
	}

	originalCurrentGOOS := currentGOOS
	originalLookupEnv := lookupEnv
	originalUserHomeDirectory := userHomeDirectory
	originalStatPath := statPath
	originalOpenWritableFile := openWritableFile
	originalRemovePath := removePath
	t.Cleanup(func() {
		currentGOOS = originalCurrentGOOS
		lookupEnv = originalLookupEnv
		userHomeDirectory = originalUserHomeDirectory
		statPath = originalStatPath
		openWritableFile = originalOpenWritableFile
		removePath = originalRemovePath
	})

	currentGOOS = func() string { return "linux" }
	lookupEnv = func(key string) (string, bool) {
		if key == "XDG_CONFIG_HOME" {
			return filepath.Join(fixtureDir, ".config"), true
		}
		return "", false
	}
	userHomeDirectory = func() (string, error) { return fixtureDir, nil }
	statPath = os.Stat

	var reservedPath string
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
		//nolint:gosec // Test seam intentionally opens the writer-provided path.
		file, err := os.OpenFile(path, flag, perm)
		if err != nil {
			return nil, err
		}
		reservedPath = path
		return failingWriteFile{File: file, writeErr: errors.New("forced write failure")}, nil
	}

	var document = reportmodel.ReportDocument{
		DocumentType:    reportmodel.ReportDocumentTypeMarkdown,
		Content:         []byte("# Report\n"),
		Year:            2024,
		CostBasisMethod: reportmodel.CostBasisMethodFIFO,
		GeneratedAt:     time.Date(2026, time.May, 21, 12, 30, 0, 0, time.Local),
	}

	_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(document))
	if err == nil {
		t.Fatalf("expected write to fail")
	}
	if reservedPath == "" {
		t.Fatalf("expected a path to be reserved before failure")
	}
	if _, statErr := os.Stat(reservedPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected partial file cleanup, stat error: %v", statErr)
	}
}

// TestOpenPathKeepsSavedFileWhenOpenFails verifies that automatic-open failure
// does not remove a report that was already saved successfully.
// Authored by: OpenCode
func TestOpenPathKeepsSavedFileWhenOpenFails(t *testing.T) {
	var reportPath = filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(reportPath, []byte("# Report\n"), 0o600); err != nil {
		t.Fatalf("write report: %v", err)
	}

	originalCurrentGOOS := currentGOOS
	originalRunOpenCommand := runOpenCommand
	t.Cleanup(func() {
		currentGOOS = originalCurrentGOOS
		runOpenCommand = originalRunOpenCommand
	})

	currentGOOS = func() string { return "linux" }
	runOpenCommand = func(OpenCommand) error {
		return errors.New("forced open failure")
	}

	err := OpenPath(reportPath)
	if err == nil {
		t.Fatalf("expected open to fail")
	}
	if _, statErr := os.Stat(reportPath); statErr != nil {
		t.Fatalf("expected saved file to remain after opener failure, got %v", statErr)
	}
}

// TestWriteReportDocumentFailureBranches verifies reservation, directory, sync,
// and close branches through package seams.
// Authored by: OpenCode
func TestWriteReportDocumentFailureBranches(t *testing.T) {
	t.Run("documents path is not a directory", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		var documentsPath = filepath.Join(fixtureDir, "Documents")
		if err := os.WriteFile(documentsPath, []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("write fake documents path: %v", err)
		}

		var originalCurrentGOOS = currentGOOS
		var originalLookupEnv = lookupEnv
		var originalUserHomeDirectory = userHomeDirectory
		var originalStatPath = statPath
		var originalOpenWritableFile = openWritableFile
		var originalRemovePath = removePath
		defer func() {
			currentGOOS = originalCurrentGOOS
			lookupEnv = originalLookupEnv
			userHomeDirectory = originalUserHomeDirectory
			statPath = originalStatPath
			openWritableFile = originalOpenWritableFile
			removePath = originalRemovePath
		}()

		currentGOOS = func() string { return "linux" }
		lookupEnv = func(key string) (string, bool) {
			if key == "XDG_CONFIG_HOME" {
				return filepath.Join(fixtureDir, ".config"), true
			}
			return "", false
		}
		userHomeDirectory = func() (string, error) { return fixtureDir, nil }
		statPath = os.Stat
		openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
			//nolint:gosec // Test seam intentionally opens the writer-provided path.
			return os.OpenFile(path, flag, perm)
		}
		removePath = os.Remove

		_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(validReportDocument(time.Now())))
		if err == nil || !strings.Contains(err.Error(), "is not a directory") {
			t.Fatalf("expected non-directory documents error, got %v", err)
		}
	})

	t.Run("documents directory stat failure", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		restoreOutputSeams := installWriterTestSeams(t, fixtureDir)
		defer restoreOutputSeams()

		statPath = func(string) (os.FileInfo, error) {
			return nil, errors.New("stat boom")
		}

		_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(validReportDocument(time.Now())))
		if err == nil || !strings.Contains(err.Error(), "inspect documents directory") {
			t.Fatalf("expected wrapped stat failure, got %v", err)
		}
	})

	t.Run("sync failure removes partial file", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		restoreOutputSeams := installWriterTestSeams(t, fixtureDir)
		defer restoreOutputSeams()

		var reservedPath string
		openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
			//nolint:gosec // Test seam intentionally opens the writer-provided path.
			var file, err = os.OpenFile(path, flag, perm)
			if err != nil {
				return nil, err
			}
			reservedPath = path
			return failingSyncFile{File: file, syncErr: errors.New("sync boom")}, nil
		}

		_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(validReportDocument(time.Now())))
		if err == nil || !strings.Contains(err.Error(), "sync report file") {
			t.Fatalf("expected wrapped sync failure, got %v", err)
		}
		assertPathRemoved(t, reservedPath)
	})

	t.Run("close failure removes partial file", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		restoreOutputSeams := installWriterTestSeams(t, fixtureDir)
		defer restoreOutputSeams()

		var reservedPath string
		openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
			//nolint:gosec // Test seam intentionally opens the writer-provided path.
			var file, err = os.OpenFile(path, flag, perm)
			if err != nil {
				return nil, err
			}
			reservedPath = path
			return failingCloseFile{File: file, closeErr: errors.New("close boom")}, nil
		}

		_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(validReportDocument(time.Now())))
		if err == nil || !strings.Contains(err.Error(), "close report file") {
			t.Fatalf("expected wrapped close failure, got %v", err)
		}
		assertPathRemoved(t, reservedPath)
	})
}

// TestWriteReportDocumentAdditionalBranches verifies timestamp fallback,
// successful writes, opener wrapping, and injected post-create failures.
// Authored by: OpenCode
func TestWriteReportDocumentAdditionalBranches(t *testing.T) {
	t.Run("writes successfully with generated-at fallback", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		var fallbackTime = time.Date(2026, time.May, 22, 12, 34, 56, 0, time.UTC)
		var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
		defer restoreOutputSeams()
		currentTime = func() time.Time { return fallbackTime }

		var bundle, err = WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(validReportDocument(time.Time{})))
		if err != nil {
			t.Fatalf("write report document: %v", err)
		}
		var outputFile = bundle.Files[0]
		if outputFile.SavedAt != fallbackTime {
			t.Fatalf("expected saved-at timestamp fallback, got %#v", outputFile)
		}
		if !strings.Contains(outputFile.Filename, fallbackTime.Format("2006-01-02_15-04-05")) {
			t.Fatalf("expected deterministic fallback timestamp in filename, got %#v", outputFile)
		}
		var body, readErr = os.ReadFile(outputFile.Path)
		if readErr != nil || string(body) != "# Report\n" {
			t.Fatalf("expected saved report content, got body=%q err=%v", string(body), readErr)
		}
	})

	t.Run("wraps invalid document before filesystem work", func(t *testing.T) {
		_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(reportmodel.ReportDocument{}))
		if err == nil || !strings.Contains(err.Error(), "report document type") {
			t.Fatalf("expected invalid document validation failure, got %v", err)
		}
	})

	t.Run("open path uses exported resolver and wraps opener failures", func(t *testing.T) {
		var previousCurrentGOOS = currentGOOS
		var previousRunOpenCommand = runOpenCommand
		defer func() {
			currentGOOS = previousCurrentGOOS
			runOpenCommand = previousRunOpenCommand
		}()
		currentGOOS = func() string { return "linux" }
		runOpenCommand = func(command OpenCommand) error {
			if command.Name != "xdg-open" || len(command.Args) != 1 || command.Args[0] != "/tmp/report.md" {
				t.Fatalf("unexpected open command: %#v", command)
			}
			return errors.New("open boom")
		}

		if err := OpenPath("/tmp/report.md"); err == nil || !strings.Contains(err.Error(), `open report path "/tmp/report.md" with xdg-open`) {
			t.Fatalf("expected wrapped open failure, got %v", err)
		}
	})

	t.Run("install write failure after create uses provided error", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		var reservedPath = filepath.Join(fixtureDir, "report.md")
		var restore = installWriteFailureAfterCreateForTesting(errors.New("custom write failure"))
		defer restore()

		var file, err = openWritableFile(reservedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			t.Fatalf("reserve file with injected write failure: %v", err)
		}
		defer func() {
			_ = file.Close()
			_ = os.Remove(reservedPath)
		}()

		if _, err = file.Write([]byte("content")); err == nil || !strings.Contains(err.Error(), "custom write failure") {
			t.Fatalf("expected injected custom write error, got %v", err)
		}
	})

	t.Run("wraps documents directory resolution failure", func(t *testing.T) {
		var previousCurrentGOOS = currentGOOS
		defer func() {
			currentGOOS = previousCurrentGOOS
		}()

		currentGOOS = func() string { return "plan9" }
		if _, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(validReportDocument(time.Now()))); err == nil || !strings.Contains(err.Error(), "documents directory resolution is unsupported") {
			t.Fatalf("expected documents resolution failure to be wrapped, got %v", err)
		}
	})

	t.Run("wraps reservation failure from write path", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
		defer restoreOutputSeams()

		openWritableFile = func(string, int, os.FileMode) (writeSyncCloser, error) {
			return nil, errors.New("reserve boom")
		}

		if _, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, markdownDocumentPair(validReportDocument(time.Now()))); err == nil || !strings.Contains(err.Error(), "reserve report file") {
			t.Fatalf("expected write path to surface reservation failure, got %v", err)
		}
	})

	t.Run("install write failure seam preserves underlying open errors", func(t *testing.T) {
		var previousOpenWritableFile = openWritableFile
		defer func() {
			openWritableFile = previousOpenWritableFile
		}()

		var restore = installWriteFailureAfterCreateForTesting(errors.New("should not matter"))
		defer restore()

		if _, err := openWritableFile(filepath.Join(t.TempDir(), "missing-parent", "report.md"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600); err == nil {
			t.Fatalf("expected wrapped failing writer seam to preserve underlying open error")
		}
	})
}

// TestWriteReportOutputBundleReservesMarkdownPair verifies that Markdown bundle
// output reserves the main report and Annex 1 files as one successful pair.
// Authored by: OpenCode
func TestWriteReportOutputBundleReservesMarkdownPair(t *testing.T) {
	var fixtureDir = t.TempDir()
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()

	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var documents = []reportmodel.ReportDocument{
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleMain, "# Main Report\n", generatedAt),
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleAnnex, "# Annex 1 - Audit\n", generatedAt),
	}

	var bundle, err = WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
	if err != nil {
		t.Fatalf("write markdown report document bundle: %v", err)
	}
	assertOutputBundle(t, bundle, reportmodel.ReportOutputFormatMarkdown, generatedAt, 2)

	var expectedMain = "ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56.md"
	var expectedAnnex = "ghostfolio-capital-gains-2024-fifo-annex-1-2026-05-21_12-34-56.md"
	assertOutputFile(t, bundle.Files[0], reportmodel.ReportDocumentRoleMain, reportmodel.ReportMediaTypeMarkdown, expectedMain, "# Main Report\n")
	assertOutputFile(t, bundle.Files[1], reportmodel.ReportDocumentRoleAnnex, reportmodel.ReportMediaTypeMarkdown, expectedAnnex, "# Annex 1 - Audit\n")
}

// TestWriteReportOutputBundleUsesMatchedMarkdownSuffixes verifies that a Markdown
// bundle collision advances both filenames to the same numeric suffix.
// Authored by: OpenCode
func TestWriteReportOutputBundleUsesMatchedMarkdownSuffixes(t *testing.T) {
	var fixtureDir = t.TempDir()
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()

	var documentsDir = filepath.Join(fixtureDir, "Documents")
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var occupiedMain = filepath.Join(documentsDir, "ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56.md")
	var occupiedAnnex = filepath.Join(documentsDir, "ghostfolio-capital-gains-2024-fifo-annex-1-2026-05-21_12-34-56-2.md")
	if err := os.WriteFile(occupiedMain, []byte("existing main"), 0o600); err != nil {
		t.Fatalf("seed existing main report: %v", err)
	}
	if err := os.WriteFile(occupiedAnnex, []byte("existing annex"), 0o600); err != nil {
		t.Fatalf("seed existing annex report: %v", err)
	}

	var documents = []reportmodel.ReportDocument{
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleMain, "# Main Report\n", generatedAt),
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleAnnex, "# Annex 1 - Audit\n", generatedAt),
	}

	var bundle, err = WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
	if err != nil {
		t.Fatalf("write markdown report document bundle after collisions: %v", err)
	}

	var expectedMain = "ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-3.md"
	var expectedAnnex = "ghostfolio-capital-gains-2024-fifo-annex-1-2026-05-21_12-34-56-3.md"
	assertOutputFile(t, bundle.Files[0], reportmodel.ReportDocumentRoleMain, reportmodel.ReportMediaTypeMarkdown, expectedMain, "# Main Report\n")
	assertOutputFile(t, bundle.Files[1], reportmodel.ReportDocumentRoleAnnex, reportmodel.ReportMediaTypeMarkdown, expectedAnnex, "# Annex 1 - Audit\n")
}

// TestWriteReportOutputBundleUsesPDFFilenameSuffixes verifies that PDF bundle
// output uses the report filename stem with a PDF extension and numeric suffixes.
// Authored by: OpenCode
func TestWriteReportOutputBundleUsesPDFFilenameSuffixes(t *testing.T) {
	var fixtureDir = t.TempDir()
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()

	var documentsDir = filepath.Join(fixtureDir, "Documents")
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var occupiedPDF = filepath.Join(documentsDir, "ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56.pdf")
	if err := os.WriteFile(occupiedPDF, []byte("existing pdf"), 0o600); err != nil {
		t.Fatalf("seed existing PDF report: %v", err)
	}

	var documents = []reportmodel.ReportDocument{
		validPDFReportDocument([]byte("%PDF-1.7\nreport\n"), generatedAt),
	}

	var bundle, err = WriteReportOutputBundle(reportmodel.ReportOutputFormatPDF, documents)
	if err != nil {
		t.Fatalf("write PDF report document bundle after collision: %v", err)
	}
	assertOutputBundle(t, bundle, reportmodel.ReportOutputFormatPDF, generatedAt, 1)

	var expectedPDF = "ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-2.pdf"
	assertOutputFile(t, bundle.Files[0], reportmodel.ReportDocumentRoleCombined, reportmodel.ReportMediaTypePDF, expectedPDF, "%PDF-1.7\nreport\n")
}

// TestWriteReportOutputBundleUsesGeneratedAtFallback verifies bundle writes apply
// the same generated-at fallback behavior as single-document writes.
// Authored by: OpenCode
func TestWriteReportOutputBundleUsesGeneratedAtFallback(t *testing.T) {
	var fixtureDir = t.TempDir()
	var fallbackTime = time.Date(2026, time.May, 22, 12, 34, 56, 0, time.UTC)
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()
	currentTime = func() time.Time { return fallbackTime }

	var documents = []reportmodel.ReportDocument{
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleMain, "# Main Report\n", time.Time{}),
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleAnnex, "# Annex 1 - Audit\n", time.Time{}),
	}

	var bundle, err = WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
	if err != nil {
		t.Fatalf("write markdown bundle with generated-at fallback: %v", err)
	}
	assertOutputBundle(t, bundle, reportmodel.ReportOutputFormatMarkdown, fallbackTime, 2)
	for _, outputFile := range bundle.Files {
		if !strings.Contains(outputFile.Filename, fallbackTime.Format("2006-01-02_15-04-05")) {
			t.Fatalf("expected fallback timestamp in filename, got %#v", outputFile)
		}
	}
}

// TestWriteReportOutputBundleCleansUpBundleOnWriteFailure verifies that a failed
// multi-file bundle write removes every file created by the attempt.
// Authored by: OpenCode
func TestWriteReportOutputBundleCleansUpBundleOnWriteFailure(t *testing.T) {
	var fixtureDir = t.TempDir()
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()

	var openedPaths []string
	var openCount int
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
		//nolint:gosec // Test seam intentionally opens the writer-provided path.
		var file, err = os.OpenFile(path, flag, perm)
		if err != nil {
			return nil, err
		}
		openedPaths = append(openedPaths, path)
		openCount++
		if openCount == 2 {
			return failingWriteFile{File: file, writeErr: errors.New("annex write boom")}, nil
		}
		return file, nil
	}

	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var documents = []reportmodel.ReportDocument{
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleMain, "# Main Report\n", generatedAt),
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleAnnex, "# Annex 1 - Audit\n", generatedAt),
	}

	_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
	if err == nil || !strings.Contains(err.Error(), "write report file") {
		t.Fatalf("expected bundle write failure, got %v", err)
	}
	if len(openedPaths) != 2 {
		t.Fatalf("expected both Markdown paths to be reserved before cleanup, got %d", len(openedPaths))
	}
	for _, path := range openedPaths {
		assertPathRemoved(t, path)
	}
}

// TestWriteReportOutputBundleCleansUpBundleOnFinalizationFailure verifies defensive
// output-model finalization failures still clean reserved files.
// Authored by: OpenCode
func TestWriteReportOutputBundleCleansUpBundleOnFinalizationFailure(t *testing.T) {
	t.Run("output file finalization", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
		defer restoreOutputSeams()
		var openedPaths []string
		openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
			//nolint:gosec // Test seam intentionally opens the writer-provided path.
			var file, err = os.OpenFile(path, flag, perm)
			if err != nil {
				return nil, err
			}
			openedPaths = append(openedPaths, path)
			return file, nil
		}

		var previousConstructor = newReportOutputFileForWrite
		defer func() { newReportOutputFileForWrite = previousConstructor }()
		newReportOutputFileForWrite = func(string, string, string, reportmodel.ReportDocumentRole, string, time.Time) (reportmodel.ReportOutputFile, error) {
			return reportmodel.ReportOutputFile{}, errors.New("output file finalization boom")
		}

		var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
		var documents = []reportmodel.ReportDocument{
			validMarkdownReportDocument(reportmodel.ReportDocumentRoleMain, "# Main Report\n", generatedAt),
			validMarkdownReportDocument(reportmodel.ReportDocumentRoleAnnex, "# Annex 1 - Audit\n", generatedAt),
		}

		_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
		if err == nil || !strings.Contains(err.Error(), "output file finalization boom") {
			t.Fatalf("expected output file finalization failure, got %v", err)
		}
		assertPathsRemoved(t, openedPaths, 2)
	})

	t.Run("output bundle finalization", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
		defer restoreOutputSeams()
		var openedPaths []string
		openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
			//nolint:gosec // Test seam intentionally opens the writer-provided path.
			var file, err = os.OpenFile(path, flag, perm)
			if err != nil {
				return nil, err
			}
			openedPaths = append(openedPaths, path)
			return file, nil
		}

		var previousConstructor = newReportOutputBundleForWrite
		defer func() { newReportOutputBundleForWrite = previousConstructor }()
		newReportOutputBundleForWrite = func(reportmodel.ReportOutputFormat, []reportmodel.ReportOutputFile, time.Time, bool, string) (reportmodel.ReportOutputBundle, error) {
			return reportmodel.ReportOutputBundle{}, errors.New("output bundle finalization boom")
		}

		var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
		var documents = []reportmodel.ReportDocument{
			validMarkdownReportDocument(reportmodel.ReportDocumentRoleMain, "# Main Report\n", generatedAt),
			validMarkdownReportDocument(reportmodel.ReportDocumentRoleAnnex, "# Annex 1 - Audit\n", generatedAt),
		}

		_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
		if err == nil || !strings.Contains(err.Error(), "output bundle finalization boom") {
			t.Fatalf("expected output bundle finalization failure, got %v", err)
		}
		assertPathsRemoved(t, openedPaths, 2)
	})
}

// TestValidateDocumentsDirectoryRejectsFile verifies non-directory Documents
// path handling before reservation.
// Authored by: OpenCode
func TestValidateDocumentsDirectoryRejectsFile(t *testing.T) {
	var filePath = filepath.Join(t.TempDir(), "Documents")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("seed file path: %v", err)
	}

	var err = validateDocumentsDirectory(filePath)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected non-directory validation failure, got %v", err)
	}
}

// TestWriteReportOutputBundleAdditionalFailureBranches verifies bundle writer
// failures that occur before and during reservation and commit.
// Authored by: OpenCode
func TestWriteReportOutputBundleAdditionalFailureBranches(t *testing.T) {
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var mainDocument = validMarkdownReportDocument(reportmodel.ReportDocumentRoleMain, "# Main Report\n", generatedAt)
	var annexDocument = validMarkdownReportDocument(reportmodel.ReportDocumentRoleAnnex, "# Annex 1\n", generatedAt)

	t.Run("rejects invalid output format before resolving documents", func(t *testing.T) {
		if _, err := WriteReportOutputBundle(reportmodel.ReportOutputFormat("html"), []reportmodel.ReportDocument{mainDocument, annexDocument}); err == nil || !strings.Contains(err.Error(), "unsupported report output format") {
			t.Fatalf("expected output format validation failure, got %v", err)
		}
	})

	t.Run("rejects invalid document before resolving documents", func(t *testing.T) {
		var invalidDocument = mainDocument
		invalidDocument.Content = []byte("   ")
		if _, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, []reportmodel.ReportDocument{invalidDocument, annexDocument}); err == nil || !strings.Contains(err.Error(), "report document 0") {
			t.Fatalf("expected document validation failure, got %v", err)
		}
	})

	t.Run("propagates documents directory resolution failure", func(t *testing.T) {
		var previousCurrentGOOS = currentGOOS
		defer func() {
			currentGOOS = previousCurrentGOOS
		}()
		currentGOOS = func() string { return "plan9" }

		if _, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, []reportmodel.ReportDocument{mainDocument, annexDocument}); err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("expected documents directory resolution failure, got %v", err)
		}
	})

	t.Run("wraps reservation failure", func(t *testing.T) {
		var fixtureDir = t.TempDir()
		var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
		defer restoreOutputSeams()

		openWritableFile = func(string, int, os.FileMode) (writeSyncCloser, error) {
			return nil, errors.New("reserve boom")
		}

		if _, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, []reportmodel.ReportDocument{mainDocument, annexDocument}); err == nil || !strings.Contains(err.Error(), "reserve report file") {
			t.Fatalf("expected reservation failure, got %v", err)
		}
	})
}

// TestWriteReportOutputBundleCleansUpOnSyncAndCloseFailures verifies bundle cleanup
// after write succeeds but file finalization fails.
// Authored by: OpenCode
func TestWriteReportOutputBundleCleansUpOnSyncAndCloseFailures(t *testing.T) {
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var documents = []reportmodel.ReportDocument{
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleMain, "# Main Report\n", generatedAt),
		validMarkdownReportDocument(reportmodel.ReportDocumentRoleAnnex, "# Annex 1\n", generatedAt),
	}

	for _, testCase := range []struct {
		name    string
		wrap    func(*os.File) writeSyncCloser
		message string
	}{
		{
			name: "sync failure",
			wrap: func(file *os.File) writeSyncCloser {
				return failingSyncFile{File: file, syncErr: errors.New("sync boom")}
			},
			message: "sync report file",
		},
		{
			name: "close failure",
			wrap: func(file *os.File) writeSyncCloser {
				return failingCloseFile{File: file, closeErr: errors.New("close boom")}
			},
			message: "close report file",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var fixtureDir = t.TempDir()
			var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
			defer restoreOutputSeams()

			var openedPaths []string
			openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
				//nolint:gosec // Test seam intentionally opens the writer-provided path.
				var file, err = os.OpenFile(path, flag, perm)
				if err != nil {
					return nil, err
				}
				openedPaths = append(openedPaths, path)
				return testCase.wrap(file), nil
			}

			_, err := WriteReportOutputBundle(reportmodel.ReportOutputFormatMarkdown, documents)
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("expected %s failure, got %v", testCase.message, err)
			}
			for _, path := range openedPaths {
				assertPathRemoved(t, path)
			}
		})
	}
}

// TestWriteReportOutputBundleRemovesMainWhenAnnexReservationFails verifies that
// a Markdown second-path reservation failure removes the already reserved main
// path and reports no partial bundle.
// Authored by: OpenCode
func TestWriteReportOutputBundleRemovesMainWhenAnnexReservationFails(t *testing.T) {
	var fixtureDir = t.TempDir()
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()

	var attemptedPaths []string
	var reservedPaths []string
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
		attemptedPaths = append(attemptedPaths, path)
		if len(attemptedPaths) == 2 {
			return nil, errors.New("synthetic annex reservation failure")
		}
		//nolint:gosec // Test seam intentionally opens the writer-provided path.
		var file, err = os.OpenFile(path, flag, perm)
		if err != nil {
			return nil, err
		}
		reservedPaths = append(reservedPaths, path)
		return file, nil
	}

	var bundle, err = WriteReportOutputBundle(
		reportmodel.ReportOutputFormatMarkdown,
		markdownDocumentPair(validReportDocument(time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC))),
	)
	if err == nil || !strings.Contains(err.Error(), "reserve report file") {
		t.Fatalf("expected second-path reservation failure, got %v", err)
	}
	assertNoSavedOutputBundle(t, bundle)
	assertPathsRemoved(t, reservedPaths, 1)
	if len(attemptedPaths) != 2 {
		t.Fatalf("expected main and Annex paths to be attempted, got %v", attemptedPaths)
	}
	assertPathDoesNotExist(t, attemptedPaths[1])
}

// TestWriteReportOutputBundleCleansUpFormatFailureMatrix verifies write, sync,
// and close failures for both formats, including failures in the Markdown Annex
// after the main document has completed.
// Authored by: OpenCode
func TestWriteReportOutputBundleCleansUpFormatFailureMatrix(t *testing.T) {
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var testCases = []struct {
		name             string
		outputFormat     reportmodel.ReportOutputFormat
		documents        []reportmodel.ReportDocument
		failureFileIndex int
		wrap             func(*os.File) writeSyncCloser
		message          string
		expectedFiles    int
	}{
		{
			name:             "Markdown Annex write failure",
			outputFormat:     reportmodel.ReportOutputFormatMarkdown,
			documents:        markdownDocumentPair(validReportDocument(generatedAt)),
			failureFileIndex: 2,
			wrap: func(file *os.File) writeSyncCloser {
				return failingWriteFile{File: file, writeErr: errors.New("synthetic Annex write failure")}
			},
			message:       "write report file",
			expectedFiles: 2,
		},
		{
			name:             "Markdown Annex sync failure",
			outputFormat:     reportmodel.ReportOutputFormatMarkdown,
			documents:        markdownDocumentPair(validReportDocument(generatedAt)),
			failureFileIndex: 2,
			wrap: func(file *os.File) writeSyncCloser {
				return failingSyncFile{File: file, syncErr: errors.New("synthetic Annex sync failure")}
			},
			message:       "sync report file",
			expectedFiles: 2,
		},
		{
			name:             "Markdown Annex close failure",
			outputFormat:     reportmodel.ReportOutputFormatMarkdown,
			documents:        markdownDocumentPair(validReportDocument(generatedAt)),
			failureFileIndex: 2,
			wrap: func(file *os.File) writeSyncCloser {
				return failingCloseFile{File: file, closeErr: errors.New("synthetic Annex close failure")}
			},
			message:       "close report file",
			expectedFiles: 2,
		},
		{
			name:             "PDF write failure",
			outputFormat:     reportmodel.ReportOutputFormatPDF,
			documents:        []reportmodel.ReportDocument{validPDFReportDocument([]byte("%PDF synthetic payload\n"), generatedAt)},
			failureFileIndex: 1,
			wrap: func(file *os.File) writeSyncCloser {
				return failingWriteFile{File: file, writeErr: errors.New("synthetic PDF write failure")}
			},
			message:       "write report file",
			expectedFiles: 1,
		},
		{
			name:             "PDF sync failure",
			outputFormat:     reportmodel.ReportOutputFormatPDF,
			documents:        []reportmodel.ReportDocument{validPDFReportDocument([]byte("%PDF synthetic payload\n"), generatedAt)},
			failureFileIndex: 1,
			wrap: func(file *os.File) writeSyncCloser {
				return failingSyncFile{File: file, syncErr: errors.New("synthetic PDF sync failure")}
			},
			message:       "sync report file",
			expectedFiles: 1,
		},
		{
			name:             "PDF close failure",
			outputFormat:     reportmodel.ReportOutputFormatPDF,
			documents:        []reportmodel.ReportDocument{validPDFReportDocument([]byte("%PDF synthetic payload\n"), generatedAt)},
			failureFileIndex: 1,
			wrap: func(file *os.File) writeSyncCloser {
				return failingCloseFile{File: file, closeErr: errors.New("synthetic PDF close failure")}
			},
			message:       "close report file",
			expectedFiles: 1,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			var fixtureDir = t.TempDir()
			var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
			defer restoreOutputSeams()

			var openedPaths []string
			openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
				//nolint:gosec // Test seam intentionally opens the writer-provided path.
				var file, err = os.OpenFile(path, flag, perm)
				if err != nil {
					return nil, err
				}
				openedPaths = append(openedPaths, path)
				if len(openedPaths) == testCase.failureFileIndex {
					return testCase.wrap(file), nil
				}
				return file, nil
			}

			var bundle, err = WriteReportOutputBundle(testCase.outputFormat, testCase.documents)
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("expected %s, got %v", testCase.message, err)
			}
			assertNoSavedOutputBundle(t, bundle)
			assertPathsRemoved(t, openedPaths, testCase.expectedFiles)
		})
	}
}

// TestWriteReportOutputBundleCleansUpOnOutputValidationAndBundleFailures
// verifies that both output metadata validation and final bundle validation
// failures remove all paths reserved by either supported format.
// Authored by: OpenCode
func TestWriteReportOutputBundleCleansUpOnOutputValidationAndBundleFailures(t *testing.T) {
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var testCases = []struct {
		name          string
		outputFormat  reportmodel.ReportOutputFormat
		documents     []reportmodel.ReportDocument
		expectedFiles int
	}{
		{
			name:          "Markdown",
			outputFormat:  reportmodel.ReportOutputFormatMarkdown,
			documents:     markdownDocumentPair(validReportDocument(generatedAt)),
			expectedFiles: 2,
		},
		{
			name:          "PDF",
			outputFormat:  reportmodel.ReportOutputFormatPDF,
			documents:     []reportmodel.ReportDocument{validPDFReportDocument([]byte("%PDF synthetic payload\n"), generatedAt)},
			expectedFiles: 1,
		},
	}

	for _, failureType := range []string{"output validation", "bundle validation"} {
		failureType := failureType
		var expectedError = "synthetic " + failureType + " failure"
		for _, testCase := range testCases {
			testCase := testCase
			t.Run(failureType+"/"+testCase.name, func(t *testing.T) {
				var fixtureDir = t.TempDir()
				var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
				defer restoreOutputSeams()

				var previousOutputFileConstructor = newReportOutputFileForWrite
				var previousBundleConstructor = newReportOutputBundleForWrite
				t.Cleanup(func() {
					newReportOutputFileForWrite = previousOutputFileConstructor
					newReportOutputBundleForWrite = previousBundleConstructor
				})

				if failureType == "output validation" {
					newReportOutputFileForWrite = func(string, string, string, reportmodel.ReportDocumentRole, string, time.Time) (reportmodel.ReportOutputFile, error) {
						return reportmodel.ReportOutputFile{}, errors.New("synthetic output validation failure")
					}
				} else {
					newReportOutputBundleForWrite = func(reportmodel.ReportOutputFormat, []reportmodel.ReportOutputFile, time.Time, bool, string) (reportmodel.ReportOutputBundle, error) {
						return reportmodel.ReportOutputBundle{}, errors.New("synthetic bundle validation failure")
					}
				}

				var openedPaths []string
				openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
					//nolint:gosec // Test seam intentionally opens the writer-provided path.
					var file, err = os.OpenFile(path, flag, perm)
					if err != nil {
						return nil, err
					}
					openedPaths = append(openedPaths, path)
					return file, nil
				}

				var bundle, err = WriteReportOutputBundle(testCase.outputFormat, testCase.documents)
				if err == nil || !strings.Contains(err.Error(), expectedError) {
					t.Fatalf("expected %s failure, got %v", failureType, err)
				}
				assertNoSavedOutputBundle(t, bundle)
				assertPathsRemoved(t, openedPaths, testCase.expectedFiles)
			})
		}
	}
}

// TestWriteReportOutputBundleRequestsOwnerOnlyPermissions verifies the exact
// mode requested for every Markdown and PDF reservation and the resulting file
// mode on the deterministic local filesystem.
// Authored by: OpenCode
func TestWriteReportOutputBundleRequestsOwnerOnlyPermissions(t *testing.T) {
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var testCases = []struct {
		name         string
		outputFormat reportmodel.ReportOutputFormat
		documents    []reportmodel.ReportDocument
	}{
		{
			name:         "Markdown",
			outputFormat: reportmodel.ReportOutputFormatMarkdown,
			documents:    markdownDocumentPair(validReportDocument(generatedAt)),
		},
		{
			name:         "PDF",
			outputFormat: reportmodel.ReportOutputFormatPDF,
			documents:    []reportmodel.ReportDocument{validPDFReportDocument([]byte("%PDF synthetic payload\n"), generatedAt)},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			var fixtureDir = t.TempDir()
			var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
			defer restoreOutputSeams()

			var requestedModes []os.FileMode
			openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
				requestedModes = append(requestedModes, perm)
				//nolint:gosec // Test seam intentionally opens the writer-provided path.
				return os.OpenFile(path, flag, perm)
			}

			var bundle, err = WriteReportOutputBundle(testCase.outputFormat, testCase.documents)
			if err != nil {
				t.Fatalf("write %s output: %v", testCase.name, err)
			}
			if len(requestedModes) != len(bundle.Files) {
				t.Fatalf("expected one mode request per saved file, got modes=%d files=%d", len(requestedModes), len(bundle.Files))
			}
			for index, requestedMode := range requestedModes {
				if requestedMode != reportFileMode {
					t.Fatalf("requested mode %d = %#o, want %#o", index, requestedMode, reportFileMode)
				}
				var info, statErr = os.Stat(bundle.Files[index].Path)
				if statErr != nil {
					t.Fatalf("stat saved file %q: %v", bundle.Files[index].Path, statErr)
				}
				if info.Mode().Perm() != reportFileMode {
					t.Fatalf("saved file mode %d = %#o, want %#o", index, info.Mode().Perm(), reportFileMode)
				}
			}
		})
	}
}

// TestWriteReportOutputBundlePreservesCollisionSentinelsOnFailure verifies that
// cleanup removes only the current suffix attempt for both output formats.
// Authored by: OpenCode
func TestWriteReportOutputBundlePreservesCollisionSentinelsOnFailure(t *testing.T) {
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var testCases = []struct {
		name             string
		outputFormat     reportmodel.ReportOutputFormat
		documents        []reportmodel.ReportDocument
		failureFileIndex int
		expectedFiles    int
	}{
		{
			name:             "Markdown",
			outputFormat:     reportmodel.ReportOutputFormatMarkdown,
			documents:        markdownDocumentPair(validReportDocument(generatedAt)),
			failureFileIndex: 2,
			expectedFiles:    2,
		},
		{
			name:             "PDF",
			outputFormat:     reportmodel.ReportOutputFormatPDF,
			documents:        []reportmodel.ReportDocument{validPDFReportDocument([]byte("%PDF synthetic payload\n"), generatedAt)},
			failureFileIndex: 1,
			expectedFiles:    1,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			var fixtureDir = t.TempDir()
			var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
			defer restoreOutputSeams()

			var sentinels = seedCollisionSentinels(t, fixtureDir, testCase.outputFormat, generatedAt)
			var openedPaths []string
			openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
				//nolint:gosec // Test seam intentionally opens the writer-provided path.
				var file, err = os.OpenFile(path, flag, perm)
				if err != nil {
					return nil, err
				}
				openedPaths = append(openedPaths, path)
				if len(openedPaths) == testCase.failureFileIndex {
					return failingWriteFile{File: file, writeErr: errors.New("synthetic collision-attempt write failure")}, nil
				}
				return file, nil
			}

			var bundle, err = WriteReportOutputBundle(testCase.outputFormat, testCase.documents)
			if err == nil {
				t.Fatalf("expected collision-attempt failure")
			}
			assertNoSavedOutputBundle(t, bundle)
			assertPathsRemoved(t, openedPaths, testCase.expectedFiles)
			for _, sentinel := range sentinels {
				assertPathContent(t, sentinel.path, sentinel.content)
			}
		})
	}
}

// TestOpenPathRetainsAllSavedBundleFilesAfterWarning verifies opener-warning
// behavior after complete Markdown and PDF saves without deleting any path.
// Authored by: OpenCode
func TestOpenPathRetainsAllSavedBundleFilesAfterWarning(t *testing.T) {
	var generatedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var testCases = []struct {
		name         string
		outputFormat reportmodel.ReportOutputFormat
		documents    []reportmodel.ReportDocument
	}{
		{
			name:         "Markdown",
			outputFormat: reportmodel.ReportOutputFormatMarkdown,
			documents:    markdownDocumentPair(validReportDocument(generatedAt)),
		},
		{
			name:         "PDF",
			outputFormat: reportmodel.ReportOutputFormatPDF,
			documents:    []reportmodel.ReportDocument{validPDFReportDocument([]byte("%PDF synthetic payload\n"), generatedAt)},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			var fixtureDir = t.TempDir()
			var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
			defer restoreOutputSeams()

			var bundle, err = WriteReportOutputBundle(testCase.outputFormat, testCase.documents)
			if err != nil {
				t.Fatalf("write %s output: %v", testCase.name, err)
			}

			var previousRunOpenCommand = runOpenCommand
			t.Cleanup(func() { runOpenCommand = previousRunOpenCommand })
			var openedPaths []string
			runOpenCommand = func(command OpenCommand) error {
				openedPaths = append(openedPaths, command.Args[0])
				return errors.New("synthetic opener warning")
			}

			for _, outputFile := range bundle.Files {
				if err := OpenPath(outputFile.Path); err == nil || !strings.Contains(err.Error(), "synthetic opener warning") {
					t.Fatalf("expected opener warning for %q, got %v", outputFile.Path, err)
				}
			}
			if len(openedPaths) != len(bundle.Files) {
				t.Fatalf("expected one opener request per saved file, got %v", openedPaths)
			}
			for _, outputFile := range bundle.Files {
				if _, statErr := os.Stat(outputFile.Path); statErr != nil {
					t.Fatalf("expected saved file to remain after opener warning, got %v", statErr)
				}
			}
		})
	}
}

// TestBundleFilenameFallbacks verifies small filename helper fallbacks not hit
// by end-to-end bundle writes.
// Authored by: OpenCode
func TestBundleFilenameFallbacks(t *testing.T) {
	if filenames := bundleFilenames(reportmodel.ReportOutputFormat("html"), "base", 1); filenames != nil {
		t.Fatalf("expected unsupported format to return nil filenames, got %#v", filenames)
	}
	if got := buildAnnexReportFilenameBase("short"); got != "short-annex-1" {
		t.Fatalf("expected short base fallback, got %q", got)
	}
	if got := buildAnnexReportFilenameBase("prefix_2026-05-21_12-34-56"); got != "prefix_2026-05-21_12-34-56-annex-1" {
		t.Fatalf("expected malformed timestamp separator fallback, got %q", got)
	}
}

// TestWriteReservedReportOutputFilesRejectsMismatchedCounts verifies the
// defensive writer guard fails before any filesystem work when reservations and
// rendered documents cannot be paired. Authored by: OpenCode
func TestWriteReservedReportOutputFilesRejectsMismatchedCounts(t *testing.T) {
	var _, err = writeReservedReportOutputFiles(t.TempDir(), time.Now(), []reservedReportFile{{}}, nil)
	if err == nil || !strings.Contains(err.Error(), "reserved report file count") {
		t.Fatalf("expected reservation/document count mismatch failure, got %v", err)
	}
}

// TestWriteReportOutputBundleReportsReservationCleanupFailure verifies a path
// reserved before a later reservation failure is surfaced when removal fails.
// Authored by: OpenCode
func TestWriteReportOutputBundleReportsReservationCleanupFailure(t *testing.T) {
	var fixtureDir = t.TempDir()
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()

	var reservationErr = errors.New("synthetic Annex reservation failure")
	var removalErr = errors.New("synthetic reservation cleanup removal failure")
	var reservedPath string
	var openCount int
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
		openCount++
		if openCount == 2 {
			return nil, reservationErr
		}
		reservedPath = path
		//nolint:gosec // Test seam intentionally opens the writer-provided path.
		return os.OpenFile(path, flag, perm)
	}
	removePath = func(path string) error {
		if path == reservedPath {
			return removalErr
		}
		return os.Remove(path)
	}

	var bundle, err = WriteReportOutputBundle(
		reportmodel.ReportOutputFormatMarkdown,
		markdownDocumentPair(validReportDocument(time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC))),
	)
	if err == nil || !errors.Is(err, reservationErr) || !errors.Is(err, removalErr) {
		t.Fatalf("expected reservation and cleanup errors to remain reachable, got %v", err)
	}
	if category, ok := FailureCategoryOf(err); !ok || category != FailureCategoryReportFileWriteFailed {
		t.Fatalf("expected write failure category, got category=%q ok=%t", category, ok)
	}
	assertNoSavedOutputBundle(t, bundle)
	if paths := ResidualPathsOf(err); len(paths) != 1 || paths[0] != reservedPath {
		t.Fatalf("expected reserved path as residual cleanup context, got %#v", paths)
	}
	if _, statErr := os.Stat(reservedPath); statErr != nil {
		t.Fatalf("expected failed-removal path to remain, got %v", statErr)
	}
}

// TestWriteReportOutputBundleReportsMixedCleanupAfterAnnexFailure verifies the
// written main report is disclosed while a successfully removed Annex is not.
// Authored by: OpenCode
func TestWriteReportOutputBundleReportsMixedCleanupAfterAnnexFailure(t *testing.T) {
	var fixtureDir = t.TempDir()
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()

	var writeErr = errors.New("synthetic Annex write failure")
	var removalErr = errors.New("synthetic main cleanup removal failure")
	var openedPaths []string
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
		//nolint:gosec // Test seam intentionally opens the writer-provided path.
		var file, err = os.OpenFile(path, flag, perm)
		if err != nil {
			return nil, err
		}
		openedPaths = append(openedPaths, path)
		if len(openedPaths) == 2 {
			return failingWriteFile{File: file, writeErr: writeErr}, nil
		}
		return file, nil
	}
	removePath = func(path string) error {
		if len(openedPaths) > 0 && path == openedPaths[0] {
			return removalErr
		}
		return os.Remove(path)
	}

	var bundle, err = WriteReportOutputBundle(
		reportmodel.ReportOutputFormatMarkdown,
		markdownDocumentPair(validReportDocument(time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC))),
	)
	if err == nil || !errors.Is(err, writeErr) || !errors.Is(err, removalErr) {
		t.Fatalf("expected Annex write and main removal failures, got %v", err)
	}
	assertNoSavedOutputBundle(t, bundle)
	if paths := CleanupPathsOf(err); len(paths) != 2 {
		t.Fatalf("expected both current-attempt paths in cleanup context, got %#v", paths)
	}
	if paths := ResidualPathsOf(err); len(paths) != 1 || paths[0] != openedPaths[0] {
		t.Fatalf("expected only failed main removal as residual, got %#v", paths)
	}
	assertPathContent(t, openedPaths[0], "# Report\n")
	assertPathDoesNotExist(t, openedPaths[1])
}

// TestWriteReportOutputBundleCollectsCleanupCloseFailure verifies cleanup close
// errors remain reachable without falsely identifying a successfully removed path.
// Authored by: OpenCode
func TestWriteReportOutputBundleCollectsCleanupCloseFailure(t *testing.T) {
	var fixtureDir = t.TempDir()
	var restoreOutputSeams = installWriterTestSeams(t, fixtureDir)
	defer restoreOutputSeams()

	var writeErr = errors.New("synthetic write failure")
	var closeErr = errors.New("synthetic cleanup close failure")
	var openedPaths []string
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
		//nolint:gosec // Test seam intentionally opens the writer-provided path.
		var file, err = os.OpenFile(path, flag, perm)
		if err != nil {
			return nil, err
		}
		openedPaths = append(openedPaths, path)
		if len(openedPaths) == 1 {
			return failingWriteAndCloseFile{File: file, writeErr: writeErr, closeErr: closeErr}, nil
		}
		return file, nil
	}

	var bundle, err = WriteReportOutputBundle(
		reportmodel.ReportOutputFormatMarkdown,
		markdownDocumentPair(validReportDocument(time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC))),
	)
	if err == nil || !errors.Is(err, writeErr) || !errors.Is(err, closeErr) {
		t.Fatalf("expected initiating write and cleanup close errors, got %v", err)
	}
	assertNoSavedOutputBundle(t, bundle)
	if paths := ResidualPathsOf(err); len(paths) != 0 {
		t.Fatalf("expected no residual path after successful removal, got %#v", paths)
	}
	assertPathsRemoved(t, openedPaths, 2)
}

// failingWriteFile injects a deterministic write error after the file has been
// reserved on disk.
// Authored by: OpenCode
type failingWriteFile struct {
	*os.File
	writeErr error
}

// failingWriteAndCloseFile injects independent initiating write and cleanup
// close failures while leaving removal available to the test.
// Authored by: OpenCode
type failingWriteAndCloseFile struct {
	*os.File
	writeErr error
	closeErr error
}

// Write returns the configured initiating write failure.
// Authored by: OpenCode
func (file failingWriteAndCloseFile) Write([]byte) (int, error) {
	return 0, file.writeErr
}

// Close returns the configured cleanup close failure.
// Authored by: OpenCode
func (file failingWriteAndCloseFile) Close() error {
	return file.closeErr
}

// Write returns the configured deterministic write error.
// Authored by: OpenCode
func (file failingWriteFile) Write([]byte) (int, error) {
	return 0, file.writeErr
}

// failingSyncFile injects a deterministic sync error after a successful write.
// Authored by: OpenCode
type failingSyncFile struct {
	*os.File
	syncErr error
}

// Sync returns the configured deterministic sync error.
// Authored by: OpenCode
func (file failingSyncFile) Sync() error {
	return file.syncErr
}

// failingCloseFile injects a deterministic close error after successful writes
// and sync.
// Authored by: OpenCode
type failingCloseFile struct {
	*os.File
	closeErr error
}

// Close closes the file handle and then returns the configured close error.
// Authored by: OpenCode
func (file failingCloseFile) Close() error {
	_ = file.File.Close()
	return file.closeErr
}

// collisionSentinel records one pre-existing synthetic output file whose path
// and content must survive a later failed output attempt.
// Authored by: OpenCode
type collisionSentinel struct {
	path    string
	content string
}

// validReportDocument returns one minimal valid report document for writer tests.
// Authored by: OpenCode
func validReportDocument(generatedAt time.Time) reportmodel.ReportDocument {
	return reportmodel.ReportDocument{
		DocumentType:    reportmodel.ReportDocumentTypeMarkdown,
		Role:            reportmodel.ReportDocumentRoleMain,
		Content:         []byte("# Report\n"),
		Year:            2024,
		CostBasisMethod: reportmodel.CostBasisMethodFIFO,
		GeneratedAt:     generatedAt,
	}
}

// markdownDocumentPair builds the required main-plus-annex bundle for legacy
// writer failure-path tests.
// Authored by: OpenCode
func markdownDocumentPair(document reportmodel.ReportDocument) []reportmodel.ReportDocument {
	var main = document
	main.Role = reportmodel.ReportDocumentRoleMain
	var annex = document
	annex.Role = reportmodel.ReportDocumentRoleAnnex
	annex.Content = []byte("# Annex 1 - Audit\n")
	return []reportmodel.ReportDocument{main, annex}
}

// validMarkdownReportDocument returns one role-specific Markdown report
// document for bundle writer tests.
// Authored by: OpenCode
func validMarkdownReportDocument(role reportmodel.ReportDocumentRole, content string, generatedAt time.Time) reportmodel.ReportDocument {
	return reportmodel.ReportDocument{
		DocumentType:    reportmodel.ReportDocumentTypeMarkdown,
		Role:            role,
		Content:         []byte(content),
		Year:            2024,
		CostBasisMethod: reportmodel.CostBasisMethodFIFO,
		GeneratedAt:     generatedAt,
	}
}

// validPDFReportDocument returns one combined PDF report document for bundle
// writer tests.
// Authored by: OpenCode
func validPDFReportDocument(content []byte, generatedAt time.Time) reportmodel.ReportDocument {
	return reportmodel.ReportDocument{
		DocumentType:    reportmodel.ReportDocumentTypePDF,
		Role:            reportmodel.ReportDocumentRoleCombined,
		Content:         append([]byte(nil), content...),
		Year:            2024,
		CostBasisMethod: reportmodel.CostBasisMethodFIFO,
		GeneratedAt:     generatedAt,
	}
}

// installWriterTestSeams installs filesystem seams that resolve the test
// fixture's Documents directory and returns a restore function.
// Authored by: OpenCode
func installWriterTestSeams(t *testing.T, homeDir string) func() {
	t.Helper()

	originalCurrentGOOS := currentGOOS
	originalLookupEnv := lookupEnv
	originalUserHomeDirectory := userHomeDirectory
	originalStatPath := statPath
	originalOpenWritableFile := openWritableFile
	originalRemovePath := removePath
	originalCurrentTime := currentTime

	var documentsDir = filepath.Join(homeDir, "Documents")
	if err := os.MkdirAll(documentsDir, 0o750); err != nil && !errors.Is(err, os.ErrExist) {
		t.Fatalf("mkdir documents: %v", err)
	}

	currentGOOS = func() string { return "linux" }
	lookupEnv = func(key string) (string, bool) {
		if key == "XDG_CONFIG_HOME" {
			return filepath.Join(homeDir, ".config"), true
		}
		return "", false
	}
	userHomeDirectory = func() (string, error) { return homeDir, nil }
	statPath = os.Stat
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
		//nolint:gosec // Test seam intentionally opens the writer-provided path.
		return os.OpenFile(path, flag, perm)
	}
	removePath = os.Remove
	currentTime = time.Now

	return func() {
		currentGOOS = originalCurrentGOOS
		lookupEnv = originalLookupEnv
		userHomeDirectory = originalUserHomeDirectory
		statPath = originalStatPath
		openWritableFile = originalOpenWritableFile
		removePath = originalRemovePath
		currentTime = originalCurrentTime
	}
}

// seedCollisionSentinels creates the base and second-suffix output files that
// force a failed test attempt to reserve the third suffix.
// Authored by: OpenCode
func seedCollisionSentinels(t *testing.T, homeDir string, outputFormat reportmodel.ReportOutputFormat, generatedAt time.Time) []collisionSentinel {
	t.Helper()

	var documentsDir = filepath.Join(homeDir, "Documents")
	var baseName = buildReportFilenameBase(2024, reportmodel.CostBasisMethodFIFO, generatedAt)
	var sentinels []collisionSentinel
	for suffix := 1; suffix <= 2; suffix++ {
		for _, filename := range bundleFilenames(outputFormat, baseName, suffix) {
			var path = filepath.Join(documentsDir, filename)
			var content = "synthetic pre-existing sentinel: " + filename + "\n"
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("seed collision sentinel %q: %v", path, err)
			}
			sentinels = append(sentinels, collisionSentinel{path: path, content: content})
		}
	}

	return sentinels
}

// assertPathRemoved verifies partial-file cleanup after a failure path.
// Authored by: OpenCode
func assertPathRemoved(t *testing.T, path string) {
	t.Helper()

	if path == "" {
		t.Fatalf("expected reserved path to be captured before failure")
	}
	assertPathDoesNotExist(t, path)
}

// assertPathDoesNotExist verifies that a path was not retained after a failure
// or reservation rejection.
// Authored by: OpenCode
func assertPathDoesNotExist(t *testing.T, path string) {
	t.Helper()

	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("expected path %q not to exist, stat error: %v", path, statErr)
	}
}

// assertPathsRemoved verifies that every reserved path was cleaned up after a
// failed bundle write.
// Authored by: OpenCode
func assertPathsRemoved(t *testing.T, paths []string, expectedCount int) {
	t.Helper()

	if len(paths) != expectedCount {
		t.Fatalf("expected %d reserved paths before failure, got %d", expectedCount, len(paths))
	}
	for _, path := range paths {
		assertPathRemoved(t, path)
	}
}

// assertNoSavedOutputBundle verifies that a failed transaction returned no
// partially populated output metadata.
// Authored by: OpenCode
func assertNoSavedOutputBundle(t *testing.T, bundle reportmodel.ReportOutputBundle) {
	t.Helper()

	if bundle.OutputFormat != "" || !bundle.SavedAt.IsZero() || len(bundle.Files) != 0 || bundle.OpenRequested || bundle.OpenError != "" {
		t.Fatalf("expected no saved output bundle after failure, got %#v", bundle)
	}
}

// assertPathContent verifies that a pre-existing synthetic sentinel was not
// changed by a later output attempt.
// Authored by: OpenCode
func assertPathContent(t *testing.T, path string, expected string) {
	t.Helper()

	// #nosec G304 -- the path is created by this test's synthetic output fixture.
	var body, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sentinel %q: %v", path, err)
	}
	if string(body) != expected {
		t.Fatalf("sentinel %q changed: got %q want %q", path, string(body), expected)
	}
}

// assertOutputBundle verifies shared output bundle metadata.
// Authored by: OpenCode
func assertOutputBundle(t *testing.T, bundle reportmodel.ReportOutputBundle, format reportmodel.ReportOutputFormat, savedAt time.Time, fileCount int) {
	t.Helper()

	if bundle.OutputFormat != format {
		t.Fatalf("expected output format %q, got %#v", format, bundle)
	}
	if !bundle.SavedAt.Equal(savedAt) {
		t.Fatalf("expected saved-at %s, got %#v", savedAt, bundle)
	}
	if len(bundle.Files) != fileCount {
		t.Fatalf("expected %d output files, got %#v", fileCount, bundle.Files)
	}
}

// assertOutputFile verifies one saved output file and its persisted content.
// Authored by: OpenCode
func assertOutputFile(t *testing.T, outputFile reportmodel.ReportOutputFile, role reportmodel.ReportDocumentRole, mediaType string, filename string, content string) {
	t.Helper()

	if outputFile.Filename != filename {
		t.Fatalf("expected filename %q, got %#v", filename, outputFile)
	}
	if outputFile.Role != role {
		t.Fatalf("expected output role %q, got %#v", role, outputFile)
	}
	if outputFile.MediaType != mediaType {
		t.Fatalf("expected media type %q, got %#v", mediaType, outputFile)
	}
	if filepath.Base(outputFile.Path) != filename {
		t.Fatalf("expected path to end with %q, got %#v", filename, outputFile)
	}
	var body, err = os.ReadFile(outputFile.Path)
	if err != nil {
		t.Fatalf("read output file %q: %v", outputFile.Path, err)
	}
	if string(body) != content {
		t.Fatalf("expected output content %q, got %q", content, string(body))
	}
}
