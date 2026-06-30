package fixture

import (
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/report/calculate"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// BuildProjectReportRequest creates one deterministic report request for the
// selected empirical method and year.
//
// Example:
//
//	request, err := fixture.BuildProjectReportRequest(2024, reportmodel.CostBasisMethodFIFO)
//	if err != nil {
//		panic(err)
//	}
//	_ = request.Year
//
// Authored by: OpenCode
func BuildProjectReportRequest(year int, method reportmodel.CostBasisMethod) (reportmodel.ReportRequest, error) {
	var requestedAt = time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	return reportmodel.NewReportRequest(year, method, reportmodel.ReportBaseCurrencyUSD, requestedAt)
}

// RunProjectCalculation executes the pure report calculator for one translated
// protected activity cache, selected method, and selected report year.
//
// Example:
//
//	report, err := fixture.RunProjectCalculation(cache, 2024, reportmodel.CostBasisMethodFIFO)
//	if err != nil {
//		panic(err)
//	}
//	_ = report.Year
//
// Authored by: OpenCode
func RunProjectCalculation(cache syncmodel.ProtectedActivityCache, year int, method reportmodel.CostBasisMethod) (reportmodel.CapitalGainsReport, error) {
	var request, err = BuildProjectReportRequest(year, method)
	if err != nil {
		return reportmodel.CapitalGainsReport{}, err
	}

	return calculate.Calculate(request, cache)
}
