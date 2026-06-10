package main

import (
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// buildScopeLocalHybridCompositeOracleOutput converts one repository-controlled
// composite-boundary input into a normalized scope-local-hybrid oracle fixture.
// Authored by: OpenCode
func buildScopeLocalHybridCompositeOracleOutput(
	dataset fixture.EmpiricalDataset,
	datasetInputHash string,
	empiricalCase fixture.EmpiricalCase,
	assetIdentityKey string,
	input boundaryOracleInput,
	inputRelativePath string,
	rawInput []byte,
) (fixture.OracleOutput, error) {
	if reportmodel.CostBasisMethodScopeLocalHybrid != input.Method {
		return fixture.OracleOutput{}, fmt.Errorf("scope-local-hybrid composite input %s must declare method %s", inputRelativePath, reportmodel.CostBasisMethodScopeLocalHybrid)
	}
	if strings.TrimSpace(input.CompositeRuleVersion) == "" {
		input.CompositeRuleVersion = defaultCompositeRuleVersion
	}
	if strings.TrimSpace(input.OracleName) == "" {
		input.OracleName = defaultHybridCompositeOracleName
	}

	return buildBoundaryOracleOutputForAsset(
		dataset,
		datasetInputHash,
		empiricalCase,
		reportmodel.CostBasisMethodScopeLocalHybrid,
		assetIdentityKey,
		input,
		inputRelativePath,
		rawInput,
	)
}
