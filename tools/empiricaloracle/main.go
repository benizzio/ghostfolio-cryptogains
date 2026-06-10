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
var resolveVendoredHledgerCommand = newVendoredHledgerCommand
var captureVendoredHledgerVersion = func(ctx context.Context, command vendoredHledgerCommand) (string, error) {
	return command.captureVersion(ctx)
}

const defaultEmpiricalOutputRoot = "testdata/empirical"

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

// run validates the synthetic dataset, renders case-scoped journals, invokes
// vendored hledger, and persists normalized oracle fixtures.
// Authored by: OpenCode
func run(args []string, stdout io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}

	var flagSet = flag.NewFlagSet("empiricaloracle", flag.ContinueOnError)
	flagSet.SetOutput(stdout)

	var datasetPath = flagSet.String("dataset", "testdata/empirical/financial-dataset.yaml", "Synthetic empirical dataset path")
	var outputRoot = flagSet.String("output-root", defaultEmpiricalOutputRoot, "Empirical artifact root path")
	var regenerate = flagSet.Bool("regenerate", false, "Regenerate oracle artifacts instead of reusing existing fixtures")

	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(stdout, "Usage: empiricaloracle [flags]")
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Generate empirical hledger journals and normalized golden fixtures.")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("empiricaloracle: parse flags: %w", err)
	}

	if flagSet.NArg() != 0 {
		return fmt.Errorf("empiricaloracle: unexpected positional argument(s): %s", strings.Join(flagSet.Args(), ", "))
	}

	var ctx = context.Background()
	var repositoryRoot, err = resolveEmpiricalRepositoryRoot()
	if err != nil {
		return fmt.Errorf("empiricaloracle: resolve repository root: %w", err)
	}

	var datasetAbsolutePath string
	var datasetRelativePath string
	datasetAbsolutePath, datasetRelativePath, err = resolveRepositoryPath(repositoryRoot, *datasetPath)
	if err != nil {
		return fmt.Errorf("empiricaloracle: resolve dataset path: %w", err)
	}

	var outputRootRelativePath string
	_, outputRootRelativePath, err = resolveRepositoryPath(repositoryRoot, *outputRoot)
	if err != nil {
		return fmt.Errorf("empiricaloracle: resolve output root: %w", err)
	}

	var dataset fixture.EmpiricalDataset
	var rawDatasetContent string
	dataset, rawDatasetContent, err = fixture.LoadEmpiricalDataset(datasetAbsolutePath)
	if err != nil {
		return fmt.Errorf("empiricaloracle: load dataset: %w", err)
	}
	if err = fixture.ValidateEmpiricalDataset(datasetRelativePath, rawDatasetContent, dataset); err != nil {
		return fmt.Errorf("empiricaloracle: validate dataset: %w", err)
	}
	if err = fixture.ValidateDatasetCoverage(dataset); err != nil {
		return fmt.Errorf("empiricaloracle: validate dataset coverage: %w", err)
	}

	var hledgerVersion string
	var hledger vendoredHledgerCommand
	var hledgerReady bool

	var journals []journal
	journals, err = renderJournals(dataset, rawDatasetContent)
	if err != nil {
		return fmt.Errorf("empiricaloracle: render journals: %w", err)
	}

	var journalWriteCount int
	var goldenWriteCount int
	var journalIndex int
	for journalIndex = range journals {
		var method = reportmodel.CostBasisMethod(journals[journalIndex].ledger.Method)
		var empiricalCase, findErr = findEmpiricalCase(dataset, journals[journalIndex].ledger.CaseIDs[0], method)
		if findErr != nil {
			return fmt.Errorf("empiricaloracle: %w", findErr)
		}

		var journalRelativePath string
		journalRelativePath, err = remapOutputRelativePath(outputRootRelativePath, journals[journalIndex].ledger.HledgerJournalPath)
		if err != nil {
			return fmt.Errorf("empiricaloracle: remap journal path: %w", err)
		}

		var wroteJournal bool
		wroteJournal, err = writeArtifact(repositoryRoot, journalRelativePath, []byte(journals[journalIndex].content), *regenerate)
		if err != nil {
			return fmt.Errorf("empiricaloracle: write journal %s: %w", journalRelativePath, err)
		}
		if wroteJournal {
			journalWriteCount++
		}

		var missingGoldenPaths []string
		missingGoldenPaths, err = collectMissingGoldenPaths(outputRootRelativePath, empiricalCase, method, repositoryRoot, *regenerate)
		if err != nil {
			return fmt.Errorf("empiricaloracle: collect missing golden fixtures for case %s method %s: %w", empiricalCase.CaseID, method, err)
		}
		if len(missingGoldenPaths) == 0 {
			continue
		}
		if !hledgerReady {
			hledger, err = resolveVendoredHledgerCommand()
			if err != nil {
				return fmt.Errorf("empiricaloracle: build vendored hledger command: %w", err)
			}

			hledgerVersion, err = captureVendoredHledgerVersion(ctx, hledger)
			if err != nil {
				return fmt.Errorf("empiricaloracle: capture vendored hledger version: %w", err)
			}

			hledgerReady = true
		}

		if !*regenerate {
			if err = ensureArtifactContentMatches(repositoryRoot, journalRelativePath, journals[journalIndex].content); err != nil {
				return fmt.Errorf("empiricaloracle: %w", err)
			}
		}

		var oracleData hledgerJournalOracleData
		oracleData, err = collectHledgerJournalOracleData(ctx, hledger, journalRelativePath, empiricalCase.Year)
		if err != nil {
			return fmt.Errorf("empiricaloracle: collect hledger data for %s: %w", journalRelativePath, err)
		}

		var assetIndex int
		for assetIndex = range empiricalCase.AssetIdentityKeys {
			var assetIdentityKey = strings.TrimSpace(empiricalCase.AssetIdentityKeys[assetIndex])
			var goldenRelativePath = goldenFixtureRelativePath(outputRootRelativePath, empiricalCase, method, assetIdentityKey)
			if !*regenerate {
				var exists bool
				exists, err = artifactExists(repositoryRoot, goldenRelativePath)
				if err != nil {
					return fmt.Errorf("empiricaloracle: stat golden fixture %s: %w", goldenRelativePath, err)
				}
				if exists {
					continue
				}
			}

			var output fixture.OracleOutput
			output, err = buildOracleOutputForAsset(dataset, empiricalCase, method, assetIdentityKey, hledgerVersion, journals[journalIndex], journalRelativePath, oracleData)
			if err != nil {
				return fmt.Errorf(
					"empiricaloracle: build oracle output for case %s method %s asset %s: %w",
					empiricalCase.CaseID,
					method,
					assetIdentityKey,
					err,
				)
			}

			var rawOutput []byte
			rawOutput, err = marshalValidatedOracleOutput(goldenRelativePath, output)
			if err != nil {
				return fmt.Errorf("empiricaloracle: marshal golden fixture %s: %w", goldenRelativePath, err)
			}

			var wroteGolden bool
			wroteGolden, err = writeArtifact(repositoryRoot, goldenRelativePath, rawOutput, *regenerate)
			if err != nil {
				return fmt.Errorf("empiricaloracle: write golden fixture %s: %w", goldenRelativePath, err)
			}
			if wroteGolden {
				goldenWriteCount++
			}
		}
	}

	_, _ = fmt.Fprintf(stdout, "wrote %d journal artifact(s) and %d golden fixture(s)\n", journalWriteCount, goldenWriteCount)

	return nil
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
