// Package output defines local filesystem save and post-save open helpers for
// generated yearly gains-and-losses report files.
// Authored by: OpenCode
package output

import "errors"

// FailureCategory identifies the report-output boundary that failed.
// Authored by: OpenCode
type FailureCategory string

const (
	// FailureCategoryDocumentsDirectoryUnavailable identifies failures while
	// resolving or validating the Documents directory.
	// Authored by: OpenCode
	FailureCategoryDocumentsDirectoryUnavailable FailureCategory = "documents_directory_unavailable"

	// FailureCategoryReportFileWriteFailed identifies failures while reserving,
	// writing, syncing, or closing the final report file.
	// Authored by: OpenCode
	FailureCategoryReportFileWriteFailed FailureCategory = "report_file_write_failed"
)

// Failure preserves one typed report-output failure category around the wrapped
// underlying error.
// Authored by: OpenCode
type Failure struct {
	category      FailureCategory
	err           error
	cleanupPaths  []string
	residualPaths []string
	cleanupFailed bool
}

// Error returns the wrapped failure text.
// Authored by: OpenCode
func (failure *Failure) Error() string {
	if failure == nil || failure.err == nil {
		return ""
	}

	return failure.err.Error()
}

// Unwrap returns the wrapped underlying error.
// Authored by: OpenCode
func (failure *Failure) Unwrap() error {
	if failure == nil {
		return nil
	}

	return failure.err
}

// Category returns the typed report-output failure category.
// Authored by: OpenCode
func (failure *Failure) Category() FailureCategory {
	if failure == nil {
		return ""
	}

	return failure.category
}

// FailureCategoryOf returns the typed report-output failure category when the
// provided error carries one.
//
// Example:
//
//	category, ok := output.FailureCategoryOf(err)
//	if ok {
//		_ = category
//	}
//
// Use this helper when callers need stable failure classification without
// depending on error-message substrings.
// Authored by: OpenCode
func FailureCategoryOf(err error) (FailureCategory, bool) {
	var failure *Failure
	if !errors.As(err, &failure) || failure == nil {
		return "", false
	}

	return failure.Category(), true
}

// CleanupPathsOf returns the current-attempt paths considered during cleanup of
// one failed report output. Callers can use these paths as redaction inputs and
// must not treat them as saved output metadata.
//
// Example:
//
//	paths := output.CleanupPathsOf(err)
//	_ = paths
//
// The returned slice is a defensive copy. It is transient failure context and
// does not indicate whether any path remains on disk.
// Authored by: OpenCode
func CleanupPathsOf(err error) []string {
	var failure *Failure
	if !errors.As(err, &failure) || failure == nil {
		return nil
	}

	return append([]string(nil), failure.cleanupPaths...)
}

// ResidualPathsOf returns current-attempt report paths whose removal failed and
// which may still contain cleartext financial data after output failure.
//
// Example:
//
//	for _, path := range output.ResidualPathsOf(err) {
//		fmt.Println(path)
//	}
//
// The returned slice is a defensive copy for immediate user-facing deletion
// guidance. It is not a valid or saved output bundle.
// Authored by: OpenCode
func ResidualPathsOf(err error) []string {
	var failure *Failure
	if !errors.As(err, &failure) || failure == nil {
		return nil
	}

	return append([]string(nil), failure.residualPaths...)
}

// CleanupFailed reports whether closing or removing any current-attempt output
// path returned an error during cleanup.
//
// Example:
//
//	if output.CleanupFailed(err) {
//		log.Print("report output cleanup was incomplete")
//	}
//
// This status does not imply that a path remains. Use ResidualPathsOf for paths
// whose removal specifically failed.
// Authored by: OpenCode
func CleanupFailed(err error) bool {
	var failure *Failure
	return errors.As(err, &failure) && failure != nil && failure.cleanupFailed
}

// NewFailure preserves one typed report-output failure category around the
// wrapped error.
//
// Example:
//
//	err := output.NewFailure(output.FailureCategoryReportFileWriteFailed, errors.New("permission denied"))
//	_ = err
//
// Use this helper when callers or tests need to construct a typed output error
// that runtime code can classify without parsing error text.
// Authored by: OpenCode
func NewFailure(category FailureCategory, err error) error {
	return wrapFailure(category, err)
}

// NewFailureWithCleanup preserves one typed output failure together with
// transient current-attempt cleanup paths and paths whose removal may have
// failed. It is intended for output adapters and deterministic boundary tests.
//
// Example:
//
//	err := output.NewFailureWithCleanup(
//		output.FailureCategoryReportFileWriteFailed,
//		errors.New("write failed"),
//		[]string{"/Documents/report.md"},
//		[]string{"/Documents/report.md"},
//	)
//
// Cleanup and residual paths are defensively copied and do not represent a
// valid saved output bundle.
// Authored by: OpenCode
func NewFailureWithCleanup(category FailureCategory, err error, cleanupPaths []string, residualPaths []string) error {
	if err == nil {
		return nil
	}

	return &Failure{
		category:      category,
		err:           err,
		cleanupPaths:  append([]string(nil), cleanupPaths...),
		residualPaths: append([]string(nil), residualPaths...),
		cleanupFailed: len(residualPaths) > 0,
	}
}

// wrapFailure preserves one typed report-output failure category around the
// wrapped error.
// Authored by: OpenCode
func wrapFailure(category FailureCategory, err error) error {
	if err == nil {
		return nil
	}

	return &Failure{category: category, err: err}
}
