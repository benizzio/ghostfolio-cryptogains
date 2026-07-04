// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ReportOutputFile stores the final cleartext report file details returned to
// the runtime workflow.
// Authored by: OpenCode
type ReportOutputFile struct {
	DocumentsDirectory string
	Filename           string
	Path               string
	Role               ReportDocumentRole
	MediaType          string
	SavedAt            time.Time
}

// NewReportOutputFile creates one validated report save outcome.
//
// Example:
//
//	outputFile, err := model.NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", model.ReportDocumentRoleMain, model.ReportMediaTypeMarkdown, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = outputFile.Path
//
// Authored by: OpenCode
func NewReportOutputFile(
	documentsDirectory string,
	filename string,
	path string,
	args ...any,
) (ReportOutputFile, error) {
	var role, mediaType, savedAt, argErr = parseReportOutputFileArgs(args)
	if argErr != nil {
		return ReportOutputFile{}, argErr
	}

	var outputFile = ReportOutputFile{
		DocumentsDirectory: strings.TrimSpace(documentsDirectory),
		Filename:           strings.TrimSpace(filename),
		Path:               strings.TrimSpace(path),
		Role:               role,
		MediaType:          strings.TrimSpace(mediaType),
		SavedAt:            savedAt,
	}

	if err := outputFile.Validate(); err != nil {
		return ReportOutputFile{}, err
	}

	return outputFile, nil
}

// parseReportOutputFileArgs accepts the current role/media metadata and legacy
// Markdown main output call sites that are migrated in later tasks.
// Authored by: OpenCode
func parseReportOutputFileArgs(args []any) (ReportDocumentRole, string, time.Time, error) {
	if len(args) == 3 {
		if savedAt, ok := args[0].(time.Time); ok {
			return ReportDocumentRoleMain, ReportMediaTypeMarkdown, savedAt, nil
		}
		var role, roleOK = args[0].(ReportDocumentRole)
		var mediaType, mediaTypeOK = args[1].(string)
		var savedAt, timeOK = args[2].(time.Time)
		if roleOK && mediaTypeOK && timeOK {
			return role, mediaType, savedAt, nil
		}
	}
	if len(args) == 5 {
		var savedAt, ok = args[0].(time.Time)
		if !ok {
			return "", "", time.Time{}, fmt.Errorf("report output saved-at timestamp argument is required")
		}
		return ReportDocumentRoleMain, ReportMediaTypeMarkdown, savedAt, nil
	}

	return "", "", time.Time{}, fmt.Errorf("report output requires role, media type, and saved-at timestamp")
}

// Validate verifies one persisted-output outcome returned to runtime code.
// Authored by: OpenCode
func (outputFile ReportOutputFile) Validate() error {
	if outputFile.DocumentsDirectory == "" {
		return fmt.Errorf("report output documents directory is required")
	}
	if outputFile.Filename == "" {
		return fmt.Errorf("report output filename is required")
	}
	if outputFile.Path == "" {
		return fmt.Errorf("report output path is required")
	}
	if err := validateReportDocumentRole(outputFile.Role); err != nil {
		return fmt.Errorf("report output role: %w", err)
	}
	if outputFile.MediaType != ReportMediaTypeMarkdown && outputFile.MediaType != ReportMediaTypePDF {
		return fmt.Errorf("unsupported report output media type %q", outputFile.MediaType)
	}
	if outputFile.SavedAt.IsZero() {
		return fmt.Errorf("report output saved-at timestamp is required")
	}
	if err := outputFile.validatePathInsideDocumentsDirectory(); err != nil {
		return err
	}

	return nil
}

// validatePathInsideDocumentsDirectory verifies persisted output metadata stays
// scoped to its resolved Documents directory.
// Authored by: OpenCode
func (outputFile ReportOutputFile) validatePathInsideDocumentsDirectory() error {
	var documentsDirectory = filepath.Clean(outputFile.DocumentsDirectory)
	var path = filepath.Clean(outputFile.Path)
	var relativePath, err = filepath.Rel(documentsDirectory, path)
	if err != nil {
		return fmt.Errorf("report output path must be inside documents directory: %w", err)
	}
	if relativePath == "." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || relativePath == ".." {
		return fmt.Errorf("report output path must be inside documents directory")
	}

	return nil
}
