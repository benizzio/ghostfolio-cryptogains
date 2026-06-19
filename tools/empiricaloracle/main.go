// Package main provides the regeneration-only empirical oracle command for the
// synthetic financial dataset. It validates project-owned fixtures, executes the
// pinned rotki adapter boundary only when regeneration is required, and writes
// normalized golden fixtures for empirical tests.
//
// Authored by: OpenCode
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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

// reportGoldenWrites writes the command's user-facing completion line.
// Authored by: OpenCode
func reportGoldenWrites(stdout io.Writer, goldenWriteCount int) {
	_, _ = fmt.Fprintf(stdout, "wrote %d golden fixture(s)\n", goldenWriteCount)
}
