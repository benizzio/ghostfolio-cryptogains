// Package main provides the regeneration-only empirical oracle command for the
// synthetic financial dataset. It validates project-owned fixtures, executes the
// pinned rotki adapter boundary only when regeneration is required, and writes
// normalized golden fixtures for empirical tests.
//
// Authored by: OpenCode
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

var stderrWriter io.Writer = os.Stderr

const defaultEmpiricalOutputRoot = "testdata/empirical"

// empiricalOracleConfig stores parsed command-line options.
// Authored by: OpenCode
type empiricalOracleConfig struct {
	DatasetPath string
	OutputRoot  string
	Regenerate  bool
	Help        bool
}

// empiricalOraclePaths stores repository-resolved paths used by generation.
// Authored by: OpenCode
type empiricalOraclePaths struct {
	RepositoryRoot         string
	DatasetAbsolutePath    string
	DatasetRelativePath    string
	OutputRootRelativePath string
}

// empiricalOracleDataset stores the validated dataset plus its stable input
// hash.
// Authored by: OpenCode
type empiricalOracleDataset struct {
	Dataset   fixture.EmpiricalDataset
	InputHash string
}

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

// main parses command-line input and reports startup errors to stderr.
// Authored by: OpenCode
func main() {
	var err = run(os.Args[1:], os.Stdout)
	if err == nil {
		return
	}

	if _, writeErr := fmt.Fprintln(stderrWriter, err); writeErr != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(1)
}

// run validates the synthetic dataset, executes the active rotki-backed oracle
// boundaries when regeneration is required, and persists normalized oracle
// fixtures.
// Authored by: OpenCode
func run(args []string, stdout io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}

	var config, err = parseEmpiricalOracleConfig(args, stdout)
	if err != nil {
		return err
	}
	if config.Help {
		return nil
	}

	var paths empiricalOraclePaths
	paths, err = resolveEmpiricalOraclePaths(config)
	if err != nil {
		return err
	}

	var dataset empiricalOracleDataset
	dataset, err = loadEmpiricalOracleDataset(paths)
	if err != nil {
		return err
	}

	var generation = empiricalOracleGeneration{
		context:    context.Background(),
		paths:      paths,
		dataset:    dataset,
		regenerate: config.Regenerate,
	}
	var goldenWriteCount int
	goldenWriteCount, err = generation.generateOracleArtifacts()
	if err != nil {
		return err
	}

	reportGoldenWrites(stdout, goldenWriteCount)

	return nil
}

// parseEmpiricalOracleConfig parses CLI flags while preserving the command
// contract.
// Authored by: OpenCode
func parseEmpiricalOracleConfig(args []string, stdout io.Writer) (empiricalOracleConfig, error) {
	var flagSet = flag.NewFlagSet("empiricaloracle", flag.ContinueOnError)
	flagSet.SetOutput(stdout)

	var datasetPath = flagSet.String("dataset", "testdata/empirical/financial-dataset.yaml", "Synthetic empirical dataset path")
	var outputRoot = flagSet.String("output-root", defaultEmpiricalOutputRoot, "Empirical artifact root path")
	var regenerate = flagSet.Bool("regenerate", false, "Regenerate oracle artifacts instead of reusing existing fixtures")

	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(stdout, "Usage: empiricaloracle [flags]")
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Generate empirical oracle inputs and normalized golden fixtures.")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return empiricalOracleConfig{Help: true}, nil
		}

		return empiricalOracleConfig{}, fmt.Errorf("empiricaloracle: parse flags: %w", err)
	}

	if flagSet.NArg() != 0 {
		return empiricalOracleConfig{}, fmt.Errorf("empiricaloracle: unexpected positional argument(s): %s", strings.Join(flagSet.Args(), ", "))
	}

	return empiricalOracleConfig{
		DatasetPath: *datasetPath,
		OutputRoot:  *outputRoot,
		Regenerate:  *regenerate,
	}, nil
}

// resolveEmpiricalOraclePaths resolves CLI paths against the repository root.
// Authored by: OpenCode
func resolveEmpiricalOraclePaths(config empiricalOracleConfig) (empiricalOraclePaths, error) {
	var repositoryRoot, err = resolveEmpiricalRepositoryRoot()
	if err != nil {
		return empiricalOraclePaths{}, fmt.Errorf("empiricaloracle: resolve repository root: %w", err)
	}

	var datasetAbsolutePath string
	var datasetRelativePath string
	datasetAbsolutePath, datasetRelativePath, err = resolveRepositoryPath(repositoryRoot, config.DatasetPath)
	if err != nil {
		return empiricalOraclePaths{}, fmt.Errorf("empiricaloracle: resolve dataset path: %w", err)
	}

	var outputRootRelativePath string
	_, outputRootRelativePath, err = resolveRepositoryPath(repositoryRoot, config.OutputRoot)
	if err != nil {
		return empiricalOraclePaths{}, fmt.Errorf("empiricaloracle: resolve output root: %w", err)
	}

	return empiricalOraclePaths{
		RepositoryRoot:         repositoryRoot,
		DatasetAbsolutePath:    datasetAbsolutePath,
		DatasetRelativePath:    datasetRelativePath,
		OutputRootRelativePath: outputRootRelativePath,
	}, nil
}

// loadEmpiricalOracleDataset loads and validates the synthetic dataset.
// Authored by: OpenCode
func loadEmpiricalOracleDataset(paths empiricalOraclePaths) (empiricalOracleDataset, error) {
	var dataset fixture.EmpiricalDataset
	var rawDatasetContent string
	var err error
	dataset, rawDatasetContent, err = fixture.LoadEmpiricalDataset(paths.DatasetAbsolutePath)
	if err != nil {
		return empiricalOracleDataset{}, fmt.Errorf("empiricaloracle: load dataset: %w", err)
	}
	if err = fixture.ValidateEmpiricalDataset(paths.DatasetRelativePath, rawDatasetContent, dataset); err != nil {
		return empiricalOracleDataset{}, fmt.Errorf("empiricaloracle: validate dataset: %w", err)
	}
	if err = fixture.ValidateDatasetCoverage(dataset); err != nil {
		return empiricalOracleDataset{}, fmt.Errorf("empiricaloracle: validate dataset coverage: %w", err)
	}

	return empiricalOracleDataset{
		Dataset:   dataset,
		InputHash: stablePrefixedSHA256Hash([]byte(rawDatasetContent)),
	}, nil
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
	var goldenRelativePath = goldenFixtureRelativePath(generation.paths.OutputRootRelativePath, empiricalCase, method, assetIdentityKey)
	if !generation.regenerate {
		var exists, err = artifactExists(generation.paths.RepositoryRoot, goldenRelativePath)
		if err != nil {
			return false, fmt.Errorf("empiricaloracle: stat golden fixture %s: %w", goldenRelativePath, err)
		}
		if exists {
			return false, nil
		}
	}

	var output, err = generation.buildOracleOutput(empiricalCase, method, assetIdentityKey)
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

// reportGoldenWrites writes the command's user-facing completion line.
// Authored by: OpenCode
func reportGoldenWrites(stdout io.Writer, goldenWriteCount int) {
	_, _ = fmt.Fprintf(stdout, "wrote %d golden fixture(s)\n", goldenWriteCount)
}

// resolveRepositoryPath resolves one repository-local input path and returns its
// absolute and repository-relative forms.
// Authored by: OpenCode
func resolveRepositoryPath(repositoryRoot string, rawPath string) (string, string, error) {
	var trimmedPath = strings.TrimSpace(rawPath)
	if trimmedPath == "" {
		return "", "", fmt.Errorf("repository path is required")
	}

	var absolutePath string
	if filepath.IsAbs(trimmedPath) {
		absolutePath = filepath.Clean(trimmedPath)
	} else {
		absolutePath = filepath.Join(repositoryRoot, filepath.FromSlash(path.Clean(trimmedPath)))
	}

	var relativePath, err = filepath.Rel(repositoryRoot, absolutePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository-relative path for %s: %w", trimmedPath, err)
	}

	relativePath = filepath.ToSlash(relativePath)
	if relativePath == "." {
		return absolutePath, relativePath, nil
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, "../") {
		return "", "", fmt.Errorf("path %s escapes the repository root", trimmedPath)
	}

	return absolutePath, relativePath, nil
}

// findEmpiricalCase returns the unique empirical case for one case and method.
// Authored by: OpenCode
func findEmpiricalCase(dataset fixture.EmpiricalDataset, caseID string, method reportmodel.CostBasisMethod) (fixture.EmpiricalCase, error) {
	var caseIndex int
	for caseIndex = range dataset.Cases {
		if strings.TrimSpace(dataset.Cases[caseIndex].CaseID) != strings.TrimSpace(caseID) {
			continue
		}
		if !caseHasMethod(dataset.Cases[caseIndex], method) {
			continue
		}

		return dataset.Cases[caseIndex], nil
	}

	return fixture.EmpiricalCase{}, fmt.Errorf("empirical case %q for method %q was not found in the dataset", strings.TrimSpace(caseID), strings.TrimSpace(string(method)))
}

// remapOutputRelativePath rewrites one default empirical artifact path under the
// selected repository-relative output root.
// Authored by: OpenCode
func remapOutputRelativePath(outputRoot string, defaultRelativePath string) (string, error) {
	var cleanedOutputRoot = path.Clean(strings.TrimSpace(outputRoot))
	var cleanedDefaultPath = path.Clean(strings.TrimSpace(defaultRelativePath))
	if cleanedOutputRoot == "." || cleanedOutputRoot == "" {
		return "", fmt.Errorf("output root must be non-empty")
	}

	if cleanedDefaultPath == defaultEmpiricalOutputRoot {
		return cleanedOutputRoot, nil
	}
	if !strings.HasPrefix(cleanedDefaultPath, defaultEmpiricalOutputRoot+"/") {
		return "", fmt.Errorf("default empirical artifact path %s does not stay under %s", cleanedDefaultPath, defaultEmpiricalOutputRoot)
	}

	var suffix = strings.TrimPrefix(cleanedDefaultPath, defaultEmpiricalOutputRoot+"/")
	if suffix == "" {
		return cleanedOutputRoot, nil
	}

	return path.Join(cleanedOutputRoot, suffix), nil
}

// goldenFixtureRelativePath returns the repository-relative path for one golden
// fixture below the selected output root.
// Authored by: OpenCode
func goldenFixtureRelativePath(outputRoot string, empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod, assetIdentityKey string) string {
	var baseName = strings.TrimSpace(empiricalCase.CaseID)
	if len(empiricalCase.AssetIdentityKeys) > 1 {
		baseName += "--" + strings.TrimSpace(assetIdentityKey)
	}

	return path.Join(outputRoot, "golden", method.FilenameSlug(), baseName+".json")
}

// collectMissingGoldenPaths reports whether one case or method still needs any
// golden fixture writes under the selected output root.
// Authored by: OpenCode
func collectMissingGoldenPaths(
	outputRoot string,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	repositoryRoot string,
	regenerate bool,
) ([]string, error) {
	if regenerate {
		var allPaths = make([]string, 0, len(empiricalCase.AssetIdentityKeys))
		var assetIndex int
		for assetIndex = range empiricalCase.AssetIdentityKeys {
			allPaths = append(allPaths, goldenFixtureRelativePath(outputRoot, empiricalCase, method, empiricalCase.AssetIdentityKeys[assetIndex]))
		}

		return allPaths, nil
	}

	var missingPaths = make([]string, 0)
	var assetIndex int
	for assetIndex = range empiricalCase.AssetIdentityKeys {
		var relativePath = goldenFixtureRelativePath(outputRoot, empiricalCase, method, empiricalCase.AssetIdentityKeys[assetIndex])
		var exists, err = artifactExists(repositoryRoot, relativePath)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}

		missingPaths = append(missingPaths, relativePath)
	}

	return missingPaths, nil
}

// artifactExists reports whether one repository-relative artifact file already
// exists.
// Authored by: OpenCode
func artifactExists(repositoryRoot string, relativePath string) (bool, error) {
	var absolutePath = filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	var info, err = os.Stat(absolutePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("artifact path %s points to a directory", relativePath)
	}

	return true, nil
}

// ensureArtifactContentMatches verifies that one existing repository artifact
// already contains the expected deterministic content.
// Authored by: OpenCode
func ensureArtifactContentMatches(repositoryRoot string, relativePath string, expectedContent string) error {
	var absolutePath = filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	var actualContent, err = os.ReadFile(absolutePath)
	if err != nil {
		return fmt.Errorf("read existing artifact %s: %w", relativePath, err)
	}
	if string(actualContent) == expectedContent {
		return nil
	}

	return fmt.Errorf("existing artifact %s differs from the current deterministic render; rerun with --regenerate to refresh it", relativePath)
}

// writeArtifact persists one repository-relative artifact unless reuse without
// regeneration was requested and the file already exists.
// Authored by: OpenCode
func writeArtifact(repositoryRoot string, relativePath string, content []byte, regenerate bool) (bool, error) {
	var absolutePath = filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	var exists, err = artifactExists(repositoryRoot, relativePath)
	if err != nil {
		return false, err
	}
	if exists && !regenerate {
		return false, nil
	}

	var parentDirectory = filepath.Dir(absolutePath)
	if err = os.MkdirAll(parentDirectory, 0o755); err != nil {
		return false, fmt.Errorf("create parent directory %s: %w", filepath.ToSlash(parentDirectory), err)
	}
	if err = os.WriteFile(absolutePath, content, 0o644); err != nil {
		return false, fmt.Errorf("write artifact %s: %w", relativePath, err)
	}

	return true, nil
}

// marshalValidatedOracleOutput indents one normalized oracle fixture and
// validates the persisted JSON payload before it is written.
// Authored by: OpenCode
func marshalValidatedOracleOutput(path string, output fixture.OracleOutput) ([]byte, error) {
	var rawContent, err = json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal oracle output JSON: %w", err)
	}
	if err = fixture.ValidateOracleOutput(path, string(rawContent), output); err != nil {
		return nil, fmt.Errorf("validate oracle output JSON: %w", err)
	}

	return append(rawContent, '\n'), nil
}
