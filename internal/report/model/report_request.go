// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import (
	"fmt"
	"time"
)

// ReportRequest stores the user-selected inputs for one report-generation run.
// Authored by: OpenCode
type ReportRequest struct {
	Year               int
	CostBasisMethod    CostBasisMethod
	ReportBaseCurrency ReportBaseCurrency
	OutputFormat       ReportOutputFormat
	RequestedAt        time.Time
}

// NewReportRequest creates one validated report-generation request.
//
// Example:
//
//	request, err := model.NewReportRequest(2024, model.CostBasisMethodFIFO, model.ReportBaseCurrencyUSD, model.ReportOutputFormatMarkdown, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = request.Year
//
// Authored by: OpenCode
func NewReportRequest(
	year int,
	method CostBasisMethod,
	reportBaseCurrency ReportBaseCurrency,
	outputFormat ReportOutputFormat,
	requestedAt time.Time,
) (ReportRequest, error) {
	var request = ReportRequest{
		Year:               year,
		CostBasisMethod:    method,
		ReportBaseCurrency: reportBaseCurrency,
		OutputFormat:       outputFormat,
		RequestedAt:        requestedAt,
	}

	if err := request.Validate(); err != nil {
		return ReportRequest{}, err
	}

	return request, nil
}

// Validate verifies that the report request is complete and reusable by later
// report-generation stages.
// Authored by: OpenCode
func (request ReportRequest) Validate() error {
	if request.Year <= 0 {
		return fmt.Errorf("report request year must be greater than zero")
	}
	if err := validateCostBasisMethod(request.CostBasisMethod); err != nil {
		return fmt.Errorf("report request cost basis method: %w", err)
	}
	if err := validateReportBaseCurrency(request.ReportBaseCurrency); err != nil {
		return fmt.Errorf("report request base currency: %w", err)
	}
	if err := validateReportOutputFormat(request.OutputFormat); err != nil {
		return fmt.Errorf("report request output format: %w", err)
	}
	if request.RequestedAt.IsZero() {
		return fmt.Errorf("report request requested-at timestamp is required")
	}

	return nil
}
