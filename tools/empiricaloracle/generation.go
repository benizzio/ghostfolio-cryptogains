// Package main provides empirical oracle artifact generation orchestration for
// the regeneration-only empirical oracle command.
//
// Authored by: OpenCode
package main

import (
	"context"
	"fmt"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// empiricalOracleGeneration stores mutable regeneration state.
// Authored by: OpenCode
type empiricalOracleGeneration struct {
	context          context.Context
	paths            empiricalOraclePaths
	dataset          empiricalOracleDataset
	regenerate       bool
	rotkiSource      rotkiSourceRuntime
	rotkiSourceReady bool
}

// generateOracleArtifacts routes supported cases and methods to fixture writes.
// Authored by: OpenCode
func (generation *empiricalOracleGeneration) generateOracleArtifacts() (int, error) {
	var goldenWriteCount int
	var caseIndex int
	for caseIndex = range generation.dataset.Dataset.Cases {
		var empiricalCase = generation.dataset.Dataset.Cases[caseIndex]
		if empiricalCase.OracleSupport == fixture.OracleSupportUnsupported {
			continue
		}

		var methodIndex int
		for methodIndex = range empiricalCase.Methods {
			var method = empiricalCase.Methods[methodIndex]
			var wroteCount, err = generation.generateMethodArtifacts(empiricalCase, method)
			if err != nil {
				return 0, err
			}
			goldenWriteCount += wroteCount
		}
	}

	return goldenWriteCount, nil
}

// generateMethodArtifacts routes one case and method to the active oracle
// boundary when any golden fixture is missing or regeneration is requested.
// Authored by: OpenCode
func (generation *empiricalOracleGeneration) generateMethodArtifacts(empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod) (int, error) {
	var missingGoldenPaths, err = collectMissingGoldenPaths(generation.paths.OutputRootRelativePath, empiricalCase, method, generation.paths.RepositoryRoot, generation.regenerate)
	if err != nil {
		return 0, fmt.Errorf("empiricaloracle: collect missing golden fixtures for case %s method %s: %w", empiricalCase.CaseID, method, err)
	}
	if len(missingGoldenPaths) == 0 {
		return 0, nil
	}
	if !isRepositoryControlledBoundaryMethod(method) {
		return 0, fmt.Errorf("empiricaloracle: unsupported oracle generation method %s for case %s", method, empiricalCase.CaseID)
	}
	if err = generation.ensureRotkiSourceRuntime(); err != nil {
		return 0, err
	}

	var writeCount int
	var assetIndex int
	for assetIndex = range empiricalCase.AssetIdentityKeys {
		var wroteGolden bool
		wroteGolden, err = generation.generateAssetArtifact(empiricalCase, method, strings.TrimSpace(empiricalCase.AssetIdentityKeys[assetIndex]))
		if err != nil {
			return 0, err
		}
		if wroteGolden {
			writeCount++
		}
	}

	return writeCount, nil
}

// ensureRotkiSourceRuntime lazily resolves the verified rotki source runtime.
// Authored by: OpenCode
func (generation *empiricalOracleGeneration) ensureRotkiSourceRuntime() error {
	if generation.rotkiSourceReady {
		return nil
	}

	var rotkiSource, err = resolveRotkiSourceRuntime()
	if err != nil {
		return fmt.Errorf("empiricaloracle: resolve verified rotki source runtime: %w", err)
	}
	generation.rotkiSource = rotkiSource
	generation.rotkiSourceReady = true
	return nil
}

// generateAssetArtifact builds, validates, and writes one golden fixture.
// Authored by: OpenCode
func (generation *empiricalOracleGeneration) generateAssetArtifact(empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod, assetIdentityKey string) (bool, error) {
	var goldenRelativePath, err = goldenFixtureRelativePath(generation.paths.OutputRootRelativePath, empiricalCase, method, assetIdentityKey)
	if err != nil {
		return false, fmt.Errorf("empiricaloracle: build golden fixture path for case %s method %s asset %s: %w", empiricalCase.CaseID, method, assetIdentityKey, err)
	}

	if !generation.regenerate {
		var exists bool
		exists, err = artifactExists(generation.paths.RepositoryRoot, goldenRelativePath)
		if err != nil {
			return false, fmt.Errorf("empiricaloracle: stat golden fixture %s: %w", goldenRelativePath, err)
		}
		if exists {
			return false, nil
		}
	}

	var output fixture.OracleOutput
	output, err = generation.buildOracleOutput(empiricalCase, method, assetIdentityKey)
	if err != nil {
		return false, fmt.Errorf("empiricaloracle: build rotki-backed oracle output for case %s method %s asset %s: %w", empiricalCase.CaseID, method, assetIdentityKey, err)
	}

	var rawOutput []byte
	rawOutput, err = marshalValidatedOracleOutput(goldenRelativePath, output)
	if err != nil {
		return false, fmt.Errorf("empiricaloracle: marshal golden fixture %s: %w", goldenRelativePath, err)
	}

	var wroteGolden bool
	wroteGolden, err = writeArtifact(generation.paths.RepositoryRoot, goldenRelativePath, rawOutput, generation.regenerate)
	if err != nil {
		return false, fmt.Errorf("empiricaloracle: write golden fixture %s: %w", goldenRelativePath, err)
	}

	return wroteGolden, nil
}

// buildOracleOutput delegates one fixture to the pure rotki or composite oracle
// boundary.
// Authored by: OpenCode
func (generation *empiricalOracleGeneration) buildOracleOutput(empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod, assetIdentityKey string) (fixture.OracleOutput, error) {
	if method == reportmodel.CostBasisMethodScopeLocalHybrid {
		return buildScopeLocalHybridCompositeOracleOutput(generation.context, generation.rotkiSource, generation.paths.RepositoryRoot, generation.dataset.Dataset, generation.dataset.InputHash, empiricalCase, assetIdentityKey)
	}

	return buildRotkiOracleOutputForAsset(generation.context, generation.rotkiSource, generation.paths.RepositoryRoot, generation.dataset.Dataset, generation.dataset.InputHash, empiricalCase, method, assetIdentityKey)
}
