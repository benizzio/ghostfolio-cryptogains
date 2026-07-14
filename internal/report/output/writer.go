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

// Output model constructor seams keep defensive finalization failures testable
// after earlier validation has guaranteed normal runtime inputs.
// Authored by: OpenCode
var (
	newReportOutputFileForWrite   = reportmodel.NewReportOutputFile
	newReportOutputBundleForWrite = reportmodel.NewReportOutputBundle
)

// writeSyncCloser defines the file contract used while reserving and writing a
// final report file.
// Authored by: OpenCode
type writeSyncCloser interface {
	Name() string
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

// WriteReportOutputBundle reserves and writes every rendered report document for
// the selected output format as one success-or-cleanup operation.
//
// Example:
//
//	bundle, err := output.WriteReportOutputBundle(model.ReportOutputFormatMarkdown, []model.ReportDocument{main, annex})
//	if err != nil {
//		panic(err)
//	}
//	_ = bundle.Files
//
// Markdown output writes a main report and Annex 1 with matched collision
// suffixes. PDF output writes one combined `.pdf` file. If any write, sync, or
// close fails, every file created by the attempt is removed before returning.
// Authored by: OpenCode
func WriteReportOutputBundle(outputFormat reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
	var savedAt = normalizeReportDocumentSavedAt(documents)
	if err := outputFormat.Validate(); err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}
	if err := reportmodel.ValidateRenderedDocuments(outputFormat, documents); err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}

	var documentsDir, err = ResolveDocumentsDirectory()
	if err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}
	if err = validateDocumentsDirectory(documentsDir); err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}

	var reservations []reservedReportFile
	reservations, err = reserveReportFiles(documentsDir, outputFormat, documents, savedAt)
	if err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}

	var cleanupPaths = true
	defer func() {
		if !cleanupPaths {
			return
		}
		cleanupReservedReportFiles(reservations)
	}()

	var files []reportmodel.ReportOutputFile
	files, err = writeReservedReportOutputFiles(documentsDir, savedAt, reservations, documents)
	if err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}

	var bundle reportmodel.ReportOutputBundle
	bundle, err = newReportOutputBundleForWrite(outputFormat, files, savedAt, false, "")
	if err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}

	cleanupPaths = false
	return bundle, nil
}

// normalizeReportDocumentSavedAt applies one generated timestamp across all
// documents when the renderer did not provide one.
// Authored by: OpenCode
func normalizeReportDocumentSavedAt(documents []reportmodel.ReportDocument) time.Time {
	var savedAt time.Time
	if len(documents) > 0 {
		savedAt = documents[0].GeneratedAt
	}
	if savedAt.IsZero() {
		savedAt = currentTime()
		for index := range documents {
			documents[index].GeneratedAt = savedAt
		}
	}

	return savedAt
}

// writeReservedReportOutputFiles writes reserved files and converts them into
// validated output metadata.
// Authored by: OpenCode
func writeReservedReportOutputFiles(
	documentsDir string,
	savedAt time.Time,
	reservations []reservedReportFile,
	documents []reportmodel.ReportDocument,
) ([]reportmodel.ReportOutputFile, error) {
	if len(reservations) != len(documents) {
		return nil, fmt.Errorf("reserved report file count does not match rendered document count")
	}

	var files = make([]reportmodel.ReportOutputFile, 0, len(reservations))
	for len(reservations) > 0 {
		var reservation = reservations[0]
		var document = documents[0]
		reservations = reservations[1:]
		documents = documents[1:]

		var outputFile, err = writeReservedReportOutputFile(documentsDir, savedAt, reservation, document)
		if err != nil {
			return nil, err
		}
		files = append(files, outputFile)
	}

	return files, nil
}

// writeReservedReportOutputFile writes one reserved file and returns its output
// metadata.
// Authored by: OpenCode
func writeReservedReportOutputFile(
	documentsDir string,
	savedAt time.Time,
	reservation reservedReportFile,
	document reportmodel.ReportDocument,
) (reportmodel.ReportOutputFile, error) {
	if err := writeReservedReportFile(reservation, document); err != nil {
		return reportmodel.ReportOutputFile{}, err
	}

	return newReportOutputFileForWrite(
		documentsDir,
		reservation.filename,
		reservation.path,
		document.Role,
		reportDocumentMediaType(document),
		savedAt,
	)
}

// reservedReportFile stores one reserved output path and handle until it is
// either committed to a successful bundle or cleaned up.
// Authored by: OpenCode
type reservedReportFile struct {
	filename string
	path     string
	file     writeSyncCloser
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

// validateDocumentsDirectory verifies the resolved Documents directory before
// reserving output files.
// Authored by: OpenCode
func validateDocumentsDirectory(documentsDir string) error {
	var info, err = statPath(documentsDir)
	if err != nil {
		return wrapFailure(
			FailureCategoryDocumentsDirectoryUnavailable,
			fmt.Errorf("inspect documents directory %q: %w", documentsDir, err),
		)
	}
	if !info.IsDir() {
		return wrapFailure(
			FailureCategoryDocumentsDirectoryUnavailable,
			fmt.Errorf("documents path %q is not a directory", documentsDir),
		)
	}

	return nil
}

// reserveReportFiles reserves the full output bundle with matched suffix policy.
// Authored by: OpenCode
func reserveReportFiles(documentsDir string, outputFormat reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument, generatedAt time.Time) ([]reservedReportFile, error) {
	var baseName = buildReportFilenameBase(documents[0].Year, documents[0].CostBasisMethod, generatedAt)

	for suffix := 1; ; suffix++ {
		var filenames = bundleFilenames(outputFormat, baseName, suffix)
		var reservations, err = reserveCandidateReportFiles(documentsDir, filenames)
		if err == nil {
			return reservations, nil
		}
		cleanupReservedReportFiles(reservations)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return nil, err
	}
}

// reserveCandidateReportFiles tries to reserve all filenames for one suffix.
// Authored by: OpenCode
func reserveCandidateReportFiles(documentsDir string, filenames []string) ([]reservedReportFile, error) {
	var reservations = make([]reservedReportFile, 0, len(filenames))
	for _, filename := range filenames {
		var path = filepath.Join(documentsDir, filename)
		var file, err = openWritableFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, reportFileMode)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return reservations, os.ErrExist
			}
			return reservations, wrapFailure(
				FailureCategoryReportFileWriteFailed,
				fmt.Errorf("reserve report file %q: %w", path, err),
			)
		}
		reservations = append(reservations, reservedReportFile{filename: filename, path: path, file: file})
	}

	return reservations, nil
}

// bundleFilenames returns the expected output filenames for one suffix.
// Authored by: OpenCode
func bundleFilenames(outputFormat reportmodel.ReportOutputFormat, baseName string, suffix int) []string {
	var mainStem = baseName
	var annexStem = buildAnnexReportFilenameBase(baseName)
	if suffix > 1 {
		mainStem = fmt.Sprintf("%s-%d", mainStem, suffix)
		annexStem = fmt.Sprintf("%s-%d", annexStem, suffix)
	}

	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		return []string{mainStem + ".md", annexStem + ".md"}
	case reportmodel.ReportOutputFormatPDF:
		return []string{mainStem + ".pdf"}
	default:
		return nil
	}
}

// buildAnnexReportFilenameBase inserts the Annex 1 marker before the timestamp
// segment of the main report filename stem.
// Authored by: OpenCode
func buildAnnexReportFilenameBase(baseName string) string {
	const timestampLength = len("2006-01-02_15-04-05")
	if len(baseName) <= timestampLength+1 {
		return baseName + "-annex-1"
	}

	var timestampStart = len(baseName) - timestampLength
	if baseName[timestampStart-1] != '-' {
		return baseName + "-annex-1"
	}

	return baseName[:timestampStart-1] + "-annex-1-" + baseName[timestampStart:]
}

// writeReservedReportFile writes, syncs, and closes one reserved output file.
// Authored by: OpenCode
func writeReservedReportFile(reservation reservedReportFile, document reportmodel.ReportDocument) error {
	if _, err := reservation.file.Write(document.Content); err != nil {
		return wrapFailure(
			FailureCategoryReportFileWriteFailed,
			fmt.Errorf("write report file %q: %w", reservation.path, err),
		)
	}
	if err := reservation.file.Sync(); err != nil {
		return wrapFailure(
			FailureCategoryReportFileWriteFailed,
			fmt.Errorf("sync report file %q: %w", reservation.path, err),
		)
	}
	if err := reservation.file.Close(); err != nil {
		return wrapFailure(
			FailureCategoryReportFileWriteFailed,
			fmt.Errorf("close report file %q: %w", reservation.path, err),
		)
	}

	return nil
}

// cleanupReservedReportFiles closes and removes every reserved path.
// Authored by: OpenCode
func cleanupReservedReportFiles(reservations []reservedReportFile) {
	for _, reservation := range reservations {
		if reservation.file != nil {
			_ = reservation.file.Close()
		}
		if reservation.path != "" {
			_ = removePath(reservation.path)
		}
	}
}

// reportDocumentMediaType returns the persisted media type for one rendered
// document.
// Authored by: OpenCode
func reportDocumentMediaType(document reportmodel.ReportDocument) string {
	if document.DocumentType == reportmodel.ReportDocumentTypePDF {
		return reportmodel.ReportMediaTypePDF
	}
	return reportmodel.ReportMediaTypeMarkdown
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
