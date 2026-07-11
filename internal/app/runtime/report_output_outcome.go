// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"fmt"
	"strings"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
)

// requestAutomaticOpenBundle performs one post-save opener request for the first
// saved file in a valid output bundle and keeps all saved files when opening
// fails.
// Authored by: OpenCode
func requestAutomaticOpenBundle(
	request reportmodel.ReportRequest,
	outputBundle reportmodel.ReportOutputBundle,
	open reportPathOpener,
) (reportmodel.ReportOutputBundle, *ReportOutcome) {
	if err := outputBundle.Validate(); err != nil {
		return reportmodel.ReportOutputBundle{}, pointerToReportOutcome(
			reportFailureOutcome(
				request,
				ReportFailureReportFileWriteFailed,
				fmt.Sprintf("Could not finalize the saved report result: %s. The saved files may still exist.", strings.TrimSpace(err.Error())),
			),
		)
	}

	var primaryPath = outputBundle.Files[0].Path
	if open == nil {
		var openErr = "automatic opening is unavailable in this runtime"
		var outcome = reportFailureOutcome(request, ReportFailureAutomaticOpenFailedAfterSave, reportOpenFailureMessage(primaryPath, openErr))
		outcome.Success = true
		outcome.OutputFormat = request.OutputFormat
		outcome.OutputBundle = reportOutputBundleForOpen(request.OutputFormat, outputBundle.Files, openErr)
		outcome.OutputFile = outputBundle.Files[0]
		return outcome.OutputBundle, pointerToReportOutcome(outcome)
	}

	var err = open(primaryPath)
	if err == nil {
		var openedBundle = reportOutputBundleForOpen(request.OutputFormat, outputBundle.Files, "")
		return openedBundle, nil
	}

	var detail = strings.TrimSpace(err.Error())
	var openedBundle = reportOutputBundleForOpen(request.OutputFormat, outputBundle.Files, detail)
	return openedBundle, pointerToReportOutcome(ReportOutcome{
		Success:       true,
		Message:       reportOpenFailureMessage(primaryPath, detail),
		FailureReason: ReportFailureAutomaticOpenFailedAfterSave,
		Request:       request,
		OutputFormat:  request.OutputFormat,
		OutputBundle:  openedBundle,
		OutputFile:    outputBundle.Files[0],
	})
}

// reportOutputBundleForOpen builds bundle-level open metadata when the saved
// files already satisfy the selected output format. Invalid partial bundles are
// left empty so legacy single-file paths keep their existing behavior until the
// bundle writer is wired in.
// Authored by: OpenCode
func reportOutputBundleForOpen(
	outputFormat reportmodel.ReportOutputFormat,
	files []reportmodel.ReportOutputFile,
	openError string,
) reportmodel.ReportOutputBundle {
	var savedAt = reportOutputBundleSavedAt(files)
	if savedAt.IsZero() {
		return reportmodel.ReportOutputBundle{}
	}

	var bundle, err = reportmodel.NewReportOutputBundle(
		outputFormat,
		files,
		savedAt,
		true,
		openError,
	)
	if err != nil {
		return reportmodel.ReportOutputBundle{}
	}

	return bundle
}

// reportOutputBundleSavedAt returns the shared saved-at timestamp for one output
// bundle candidate.
// Authored by: OpenCode
func reportOutputBundleSavedAt(files []reportmodel.ReportOutputFile) time.Time {
	if len(files) == 0 {
		return time.Time{}
	}

	return files[0].SavedAt
}

// reportWriteFailureReason classifies one save failure into the supported
// runtime taxonomy.
// Authored by: OpenCode
func reportWriteFailureReason(err error) ReportFailureReason {
	var category, ok = reportoutput.FailureCategoryOf(err)
	if ok {
		switch category {
		case reportoutput.FailureCategoryDocumentsDirectoryUnavailable:
			return ReportFailureDocumentsFolderUnavailable
		case reportoutput.FailureCategoryReportFileWriteFailed:
			return ReportFailureReportFileWriteFailed
		}
	}

	return ReportFailureReportFileWriteFailed
}

// reportWriteFailureMessage formats one actionable save failure.
// Authored by: OpenCode
func reportWriteFailureMessage(reason ReportFailureReason, err error) string {
	var detail = strings.TrimSpace(err.Error())
	if reason == ReportFailureDocumentsFolderUnavailable {
		return fmt.Sprintf(
			"Could not save the report because the Documents folder is unavailable: %s. Ensure the folder exists and is writable, then try again. No report file was saved.",
			detail,
		)
	}

	return fmt.Sprintf(
		"Could not save the report file: %s. Check write permissions and free space in the Documents folder, then try again. Any partial file created during this attempt was removed.",
		detail,
	)
}

// reportWriteDiagnosticError wraps one output-preparation failure with a stable
// report-level summary for diagnostics.
// Authored by: OpenCode
func reportWriteDiagnosticError(reason ReportFailureReason, err error) error {
	if reason == ReportFailureDocumentsFolderUnavailable {
		return fmt.Errorf("could not save the report because the Documents folder is unavailable: %w", err)
	}

	return fmt.Errorf("could not save the report file: %w", err)
}

// reportOpenFailureMessage formats one non-fatal automatic-open warning.
// Authored by: OpenCode
func reportOpenFailureMessage(path string, detail string) string {
	return fmt.Sprintf(
		"Saved the report to %q, but automatic opening failed: %s. Open the file manually. To remove this cleartext report later, delete %q.",
		path,
		detail,
		path,
	)
}

// reportBundleSuccessMessage formats one successful bundle outcome.
// Authored by: OpenCode
func reportBundleSuccessMessage(files []reportmodel.ReportOutputFile) string {
	var paths = make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, fmt.Sprintf("%q", file.Path))
	}
	return fmt.Sprintf(
		"Saved the report output to %s and requested automatic opening. To remove these cleartext reports later, delete the listed files.",
		strings.Join(paths, ", "),
	)
}

// pointerToReportOutcome returns the address of one local report outcome value.
// Authored by: OpenCode
func pointerToReportOutcome(outcome ReportOutcome) *ReportOutcome {
	return &outcome
}
