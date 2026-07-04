// Package output defines local filesystem save and post-save open helpers for
// generated yearly gains-and-losses report files.
// Authored by: OpenCode
package output

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

const reportFileMode = 0o600

// writeSyncCloser defines the file contract used while reserving and writing a
// final report file.
// Authored by: OpenCode
type writeSyncCloser interface {
	Name() string
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

// WriteReportDocument reserves a unique Markdown report filename inside the
// user's Documents directory, writes the rendered content, and cleans up any
// partial file when the write fails.
//
// Example:
//
//	document := reportmodel.ReportDocument{
//		DocumentType:    reportmodel.ReportDocumentTypeMarkdown,
//		Content:         "# Report\n",
//		Year:            2024,
//		CostBasisMethod: reportmodel.CostBasisMethodFIFO,
//		GeneratedAt:     time.Now(),
//	}
//	outputFile, err := output.WriteReportDocument(document)
//	if err != nil {
//		panic(err)
//	}
//	_ = outputFile.Path
//
// The resulting filename follows the deterministic report naming convention and
// uses `-2`, `-3`, and later suffixes when the same-second base name already
// exists.
// Authored by: OpenCode
func WriteReportDocument(document reportmodel.ReportDocument) (reportmodel.ReportOutputFile, error) {
	var savedAt = document.GeneratedAt
	if savedAt.IsZero() {
		savedAt = currentTime()
		document.GeneratedAt = savedAt
	}

	if err := document.Validate(); err != nil {
		return reportmodel.ReportOutputFile{}, err
	}

	var documentsDir, err = ResolveDocumentsDirectory()
	if err != nil {
		return reportmodel.ReportOutputFile{}, err
	}

	var info, statErr = statPath(documentsDir)
	if statErr != nil {
		return reportmodel.ReportOutputFile{}, wrapFailure(
			FailureCategoryDocumentsDirectoryUnavailable,
			fmt.Errorf("inspect documents directory %q: %w", documentsDir, statErr),
		)
	}
	if !info.IsDir() {
		return reportmodel.ReportOutputFile{}, wrapFailure(
			FailureCategoryDocumentsDirectoryUnavailable,
			fmt.Errorf("documents path %q is not a directory", documentsDir),
		)
	}

	var filename, path, file, reserveErr = reserveReportFile(documentsDir, document.Year, document.CostBasisMethod, savedAt)
	if reserveErr != nil {
		return reportmodel.ReportOutputFile{}, reserveErr
	}

	var cleanupPath = true
	defer func() {
		if !cleanupPath {
			return
		}
		_ = file.Close()
		_ = removePath(path)
	}()

	if _, err = file.Write([]byte(document.Content)); err != nil {
		return reportmodel.ReportOutputFile{}, wrapFailure(
			FailureCategoryReportFileWriteFailed,
			fmt.Errorf("write report file %q: %w", path, err),
		)
	}
	if err = file.Sync(); err != nil {
		return reportmodel.ReportOutputFile{}, wrapFailure(
			FailureCategoryReportFileWriteFailed,
			fmt.Errorf("sync report file %q: %w", path, err),
		)
	}
	if err = file.Close(); err != nil {
		return reportmodel.ReportOutputFile{}, wrapFailure(
			FailureCategoryReportFileWriteFailed,
			fmt.Errorf("close report file %q: %w", path, err),
		)
	}

	cleanupPath = false

	return reportmodel.NewReportOutputFile(
		documentsDir,
		filename,
		path,
		reportmodel.ReportDocumentRoleMain,
		reportmodel.ReportMediaTypeMarkdown,
		savedAt,
	)
}

// reserveReportFile reserves a unique final report path using exclusive file
// creation.
// Authored by: OpenCode
func reserveReportFile(documentsDir string, year int, method reportmodel.CostBasisMethod, generatedAt time.Time) (string, string, writeSyncCloser, error) {
	var baseName = buildReportFilenameBase(year, method, generatedAt)

	for suffix := 1; ; suffix++ {
		var filename = baseName + ".md"
		if suffix > 1 {
			filename = fmt.Sprintf("%s-%d.md", baseName, suffix)
		}

		var path = filepath.Join(documentsDir, filename)
		var file, err = openWritableFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, reportFileMode)
		if err == nil {
			return filename, path, file, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}

		return "", "", nil, wrapFailure(
			FailureCategoryReportFileWriteFailed,
			fmt.Errorf("reserve report file %q: %w", path, err),
		)
	}
}

// buildReportFilenameBase builds the deterministic report filename stem before
// suffix and extension handling.
// Authored by: OpenCode
func buildReportFilenameBase(year int, method reportmodel.CostBasisMethod, generatedAt time.Time) string {
	return fmt.Sprintf(
		"ghostfolio-capital-gains-%d-%s-%s",
		year,
		method.FilenameSlug(),
		generatedAt.Format("2006-01-02_15-04-05"),
	)
}
