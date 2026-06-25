// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	reportoutput "github.com/benizzio/ghostfolio-cryptogains/internal/report/output"
)

// requestAutomaticOpen performs the single post-save opener request and keeps
// the saved file when the opener fails.
// Authored by: OpenCode
func requestAutomaticOpen(
	request reportmodel.ReportRequest,
	outputFile reportmodel.ReportOutputFile,
	open reportPathOpener,
) (reportmodel.ReportOutputFile, *ReportOutcome) {
	var updatedOutputFile, updateErr = reportmodel.NewReportOutputFile(
		outputFile.DocumentsDirectory,
		outputFile.Filename,
		outputFile.Path,
		outputFile.SavedAt,
		true,
		"",
	)
	if updateErr != nil {
		return reportmodel.ReportOutputFile{}, pointerToReportOutcome(
			reportFailureOutcome(
				request,
				ReportFailureReportFileWriteFailed,
				fmt.Sprintf(
					"Could not finalize the saved report result for %q: %s. The saved file may still exist at %q.",
					outputFile.Filename,
					strings.TrimSpace(updateErr.Error()),
					outputFile.Path,
				),
			),
		)
	}
	if open == nil {
		return updatedOutputFile, pointerToReportOutcome(
			reportFailureOutcome(
				request,
				ReportFailureAutomaticOpenFailedAfterSave,
				reportOpenFailureMessage(outputFile.Path, "automatic opening is unavailable in this runtime"),
			),
		)
	}

	var err = open(outputFile.Path)
	if err == nil {
		return updatedOutputFile, nil
	}

	updatedOutputFile, _ = reportmodel.NewReportOutputFile(
		outputFile.DocumentsDirectory,
		outputFile.Filename,
		outputFile.Path,
		outputFile.SavedAt,
		true,
		strings.TrimSpace(err.Error()),
	)

	return updatedOutputFile, pointerToReportOutcome(
		ReportOutcome{
			Success:       true,
			Message:       reportOpenFailureMessage(outputFile.Path, strings.TrimSpace(err.Error())),
			FailureReason: ReportFailureAutomaticOpenFailedAfterSave,
			Request:       request,
			OutputFile:    updatedOutputFile,
		},
	)
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

// reportSuccessMessage formats one successful report outcome.
// Authored by: OpenCode
func reportSuccessMessage(path string) string {
	return fmt.Sprintf(
		"Saved the report to %q and requested automatic opening. To remove this cleartext report later, delete %q.",
		path,
		path,
	)
}

// pointerToReportOutcome returns the address of one local report outcome value.
// Authored by: OpenCode
func pointerToReportOutcome(outcome ReportOutcome) *ReportOutcome {
	return &outcome
}
