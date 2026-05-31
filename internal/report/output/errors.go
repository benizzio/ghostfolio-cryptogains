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
	category FailureCategory
	err      error
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

// wrapFailure preserves one typed report-output failure category around the
// wrapped error.
// Authored by: OpenCode
func wrapFailure(category FailureCategory, err error) error {
	if err == nil {
		return nil
	}

	return &Failure{category: category, err: err}
}
