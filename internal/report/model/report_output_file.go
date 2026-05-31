// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
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
	SavedAt            time.Time
	OpenRequested      bool
	OpenError          string
}

// NewReportOutputFile creates one validated report save outcome.
//
// Example:
//
//	outputFile, err := model.NewReportOutputFile("/tmp/Documents", "report.md", "/tmp/Documents/report.md", time.Now(), true, "")
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
	savedAt time.Time,
	openRequested bool,
	openError string,
) (ReportOutputFile, error) {
	var outputFile = ReportOutputFile{
		DocumentsDirectory: strings.TrimSpace(documentsDirectory),
		Filename:           strings.TrimSpace(filename),
		Path:               strings.TrimSpace(path),
		SavedAt:            savedAt,
		OpenRequested:      openRequested,
		OpenError:          strings.TrimSpace(openError),
	}

	if err := outputFile.Validate(); err != nil {
		return ReportOutputFile{}, err
	}

	return outputFile, nil
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
	if outputFile.SavedAt.IsZero() {
		return fmt.Errorf("report output saved-at timestamp is required")
	}
	if !outputFile.OpenRequested && outputFile.OpenError != "" {
		return fmt.Errorf("report output open error requires an open request")
	}

	return nil
}
