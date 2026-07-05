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
	return WriteReportDocuments(outputFormat, documents)
}

// WriteReportDocuments is the package-level bundle writer used by tests and the
// runtime while the output bundle API is rolled through the application.
// Authored by: OpenCode
func WriteReportDocuments(outputFormat reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) (reportmodel.ReportOutputBundle, error) {
	var savedAt = normalizeReportDocumentSavedAt(documents)
	if err := outputFormat.Validate(); err != nil {
		return reportmodel.ReportOutputBundle{}, err
	}
	if err := validateBundleDocuments(outputFormat, documents); err != nil {
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

// validateBundleDocuments verifies the selected output format's required
// rendered document roles before any filesystem writes begin.
// Authored by: OpenCode
func validateBundleDocuments(outputFormat reportmodel.ReportOutputFormat, documents []reportmodel.ReportDocument) error {
	for index, document := range documents {
		if err := document.Validate(); err != nil {
			return fmt.Errorf("report document %d: %w", index, err)
		}
	}

	switch outputFormat {
	case reportmodel.ReportOutputFormatMarkdown:
		return validateMarkdownBundleDocuments(documents)
	case reportmodel.ReportOutputFormatPDF:
		return validatePDFBundleDocuments(documents)
	}

	return validateBundleDocumentMetadata(documents)
}

// validateMarkdownBundleDocuments verifies the two Markdown output documents.
// Authored by: OpenCode
func validateMarkdownBundleDocuments(documents []reportmodel.ReportDocument) error {
	if len(documents) != 2 {
		return fmt.Errorf("markdown report output requires exactly two documents")
	}
	if documents[0].DocumentType != reportmodel.ReportDocumentTypeMarkdown || documents[0].Role != reportmodel.ReportDocumentRoleMain {
		return fmt.Errorf("markdown report output document 0 must be the main Markdown document")
	}
	if documents[1].DocumentType != reportmodel.ReportDocumentTypeMarkdown || documents[1].Role != reportmodel.ReportDocumentRoleAnnex {
		return fmt.Errorf("markdown report output document 1 must be the Annex 1 Markdown document")
	}

	return validateBundleDocumentMetadata(documents)
}

// validatePDFBundleDocuments verifies the combined PDF output document.
// Authored by: OpenCode
func validatePDFBundleDocuments(documents []reportmodel.ReportDocument) error {
	if len(documents) != 1 {
		return fmt.Errorf("pdf report output requires exactly one document")
	}
	if documents[0].DocumentType != reportmodel.ReportDocumentTypePDF || documents[0].Role != reportmodel.ReportDocumentRoleCombined {
		return fmt.Errorf("pdf report output document must be the combined PDF document")
	}

	return validateBundleDocumentMetadata(documents)
}

// validateBundleDocumentMetadata verifies that all documents in one bundle share
// the same naming metadata.
// Authored by: OpenCode
func validateBundleDocumentMetadata(documents []reportmodel.ReportDocument) error {
	if len(documents) == 0 {
		return fmt.Errorf("report output requires at least one document")
	}

	var first = documents[0]
	for index := 1; index < len(documents); index++ {
		if documents[index].Year != first.Year {
			return fmt.Errorf("report document %d year does not match the first document", index)
		}
		if documents[index].CostBasisMethod != first.CostBasisMethod {
			return fmt.Errorf("report document %d cost basis method does not match the first document", index)
		}
		if !documents[index].GeneratedAt.Equal(first.GeneratedAt) {
			return fmt.Errorf("report document %d generated-at timestamp does not match the first document", index)
		}
	}

	return nil
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
	var payload = []byte(document.Content)
	if document.DocumentType == reportmodel.ReportDocumentTypePDF {
		payload = document.PDFContent
	}

	if _, err := reservation.file.Write(payload); err != nil {
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
