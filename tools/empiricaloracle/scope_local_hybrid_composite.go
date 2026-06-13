package main

import (
	"context"
	"fmt"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// buildScopeLocalHybridCompositeOracleOutput converts one repository-controlled
// composite-boundary input into a normalized scope-local-hybrid oracle fixture.
// Authored by: OpenCode
func buildScopeLocalHybridCompositeOracleOutput(
	ctx context.Context,
	runtime rotkiSourceRuntime,
	repositoryRoot string,
	dataset fixture.EmpiricalDataset,
	datasetInputHash string,
	empiricalCase fixture.EmpiricalCase,
	assetIdentityKey string,
) (fixture.OracleOutput, error) {
	if !caseHasMethod(empiricalCase, reportmodel.CostBasisMethodScopeLocalHybrid) {
		return fixture.OracleOutput{}, fmt.Errorf("scope-local-hybrid case %s must declare method %s", empiricalCase.CaseID, reportmodel.CostBasisMethodScopeLocalHybrid)
	}

	return buildRotkiCompositeOracleOutputForAsset(
		ctx,
		runtime,
		repositoryRoot,
		dataset,
		datasetInputHash,
		empiricalCase,
		assetIdentityKey,
	)
}
