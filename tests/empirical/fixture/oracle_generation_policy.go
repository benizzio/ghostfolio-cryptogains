package fixture

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

const (
	// DefaultEmpiricalArtifactRootRepositoryPath identifies the default repository-
	// relative empirical artifact root shared by journals and golden fixtures.
	// Authored by: OpenCode
	DefaultEmpiricalArtifactRootRepositoryPath = "testdata/empirical"

	// MissingOracleFixtureGenerationEnvVar enables automatic generation for
	// absent golden fixtures during fixture-backed empirical test runs.
	// Authored by: OpenCode
	MissingOracleFixtureGenerationEnvVar = "GHOSTFOLIO_CRYPTOGAINS_GENERATE_MISSING_FIXTURES"

	// MissingOracleFixtureGenerationCommand documents the exact helper command
	// used when absent fixtures are allowed to trigger generation.
	// Authored by: OpenCode
	MissingOracleFixtureGenerationCommand = "go run ./tools/empiricaloracle"
)

// executeMissingOracleFixtureGeneration stores the command seam used by the
// missing-fixture policy.
// Authored by: OpenCode
var executeMissingOracleFixtureGeneration = runMissingOracleFixtureGeneration

// OracleFixturePolicyResult stores the deterministic fixture-path state observed
// by the empirical missing-fixture policy.
// Authored by: OpenCode
type OracleFixturePolicyResult struct {
	ExpectedPaths []string
	MissingPaths  []string
	Generated     bool
}

// missingOracleFixturesError reports absent golden fixtures together with the
// exact remediation command and opt-in environment variable.
// Authored by: OpenCode
type missingOracleFixturesError struct {
	ExpectedCount  int
	EnvVar         string
	MissingPaths   []string
	RepositoryRoot string
	Command        string
}

// Error formats one actionable missing-fixture setup error.
// Authored by: OpenCode
func (err missingOracleFixturesError) Error() string {
	var builder strings.Builder

	builder.WriteString("empirical golden fixtures are missing: ")
	builder.WriteString(strings.Join(err.MissingPaths, ", "))
	builder.WriteString("; expected fixture count ")
	builder.WriteString(fmt.Sprintf("%d", err.ExpectedCount))
	builder.WriteString("; run `")
	builder.WriteString(err.Command)
	builder.WriteString("` from ")
	builder.WriteString(err.RepositoryRoot)
	builder.WriteString(" or set ")
	builder.WriteString(err.EnvVar)
	builder.WriteString("=true to allow generation during the test run")

	return builder.String()
}

// oracleFixtureGenerationError reports one actionable generation failure or one
// incomplete post-generation fixture state.
// Authored by: OpenCode
type oracleFixtureGenerationError struct {
	Command      string
	MissingPaths []string
	Output       string
	Reason       error
}

// Error formats one missing-fixture generation failure.
// Authored by: OpenCode
func (err oracleFixtureGenerationError) Error() string {
	var builder strings.Builder

	builder.WriteString("generate missing empirical golden fixtures with `")
	builder.WriteString(err.Command)
	builder.WriteString("`")
	if err.Reason != nil {
		builder.WriteString(": ")
		builder.WriteString(err.Reason.Error())
	}
	if len(err.MissingPaths) != 0 {
		builder.WriteString("; remaining missing fixtures: ")
		builder.WriteString(strings.Join(err.MissingPaths, ", "))
	}

	var trimmedOutput = strings.TrimSpace(err.Output)
	if trimmedOutput != "" {
		builder.WriteString("; command output: ")
		builder.WriteString(trimmedOutput)
	}

	return builder.String()
}

// ExpectedGoldenFixturePaths returns the deterministic repository-relative
// golden fixture paths implied by one validated empirical dataset's oracle-
// supported cases.
//
// Example:
//
//	paths := fixture.ExpectedGoldenFixturePaths(fixture.DefaultEmpiricalArtifactRootRepositoryPath, dataset)
//	_ = paths[0]
//
// Authored by: OpenCode
func ExpectedGoldenFixturePaths(outputRoot string, dataset EmpiricalDataset) []string {
	var normalizedOutputRoot = strings.TrimSpace(outputRoot)
	if normalizedOutputRoot == "" {
		normalizedOutputRoot = DefaultEmpiricalArtifactRootRepositoryPath
	}

	var expectedPathSet = make(map[string]struct{})
	var caseIndex int

	for caseIndex = range dataset.Cases {
		var empiricalCase = dataset.Cases[caseIndex]
		if empiricalCase.OracleSupport == OracleSupportUnsupported {
			continue
		}
		var methodIndex int

		for methodIndex = range empiricalCase.Methods {
			var method = empiricalCase.Methods[methodIndex]
			var assetIndex int

			for assetIndex = range empiricalCase.AssetIdentityKeys {
				var assetIdentityKey = strings.TrimSpace(empiricalCase.AssetIdentityKeys[assetIndex])
				expectedPathSet[expectedGoldenFixturePath(normalizedOutputRoot, empiricalCase, method, assetIdentityKey)] = struct{}{}
			}
		}
	}

	var paths = make([]string, 0, len(expectedPathSet))
	for fixturePath := range expectedPathSet {
		paths = append(paths, fixturePath)
	}

	sort.Strings(paths)
	return paths
}

// EnsureGoldenFixtures enforces the empirical missing-fixture policy for one
// validated dataset and output root.
//
// Example:
//
//	result, err := fixture.EnsureGoldenFixtures(context.Background(), repositoryRoot, fixture.DefaultEmpiricalArtifactRootRepositoryPath, dataset)
//	if err != nil {
//		panic(err)
//	}
//	_ = result.ExpectedPaths
//
// The helper does nothing when every expected fixture already exists. When one
// or more fixtures are absent, it either returns an actionable setup error or,
// when `GHOSTFOLIO_CRYPTOGAINS_GENERATE_MISSING_FIXTURES=true`, runs
// `go run ./tools/empiricaloracle` and rechecks only the absent fixture paths.
// Authored by: OpenCode
func EnsureGoldenFixtures(ctx context.Context, repositoryRoot string, outputRoot string, dataset EmpiricalDataset) (OracleFixturePolicyResult, error) {
	var expectedPaths = ExpectedGoldenFixturePaths(outputRoot, dataset)
	var result = OracleFixturePolicyResult{ExpectedPaths: expectedPaths}

	var missingPaths, err = collectMissingGoldenFixturePaths(repositoryRoot, expectedPaths)
	if err != nil {
		return OracleFixturePolicyResult{}, err
	}
	if len(missingPaths) == 0 {
		return result, nil
	}

	result.MissingPaths = append([]string(nil), missingPaths...)
	if !allowMissingOracleFixtureGeneration(os.Getenv(MissingOracleFixtureGenerationEnvVar)) {
		return result, missingOracleFixturesError{
			ExpectedCount:  len(expectedPaths),
			EnvVar:         MissingOracleFixtureGenerationEnvVar,
			MissingPaths:   append([]string(nil), missingPaths...),
			RepositoryRoot: repositoryRoot,
			Command:        MissingOracleFixtureGenerationCommand,
		}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err = executeMissingOracleFixtureGeneration(ctx, repositoryRoot); err != nil {
		return result, err
	}

	missingPaths, err = collectMissingGoldenFixturePaths(repositoryRoot, expectedPaths)
	if err != nil {
		return OracleFixturePolicyResult{}, err
	}
	if len(missingPaths) != 0 {
		return result, oracleFixtureGenerationError{
			Command:      MissingOracleFixtureGenerationCommand,
			MissingPaths: append([]string(nil), missingPaths...),
		}
	}

	result.Generated = true
	result.MissingPaths = nil
	return result, nil
}

// expectedGoldenFixturePath returns the repository-relative golden fixture path
// for one case, method, and asset.
// Authored by: OpenCode
func expectedGoldenFixturePath(outputRoot string, empiricalCase EmpiricalCase, method reportmodel.CostBasisMethod, assetIdentityKey string) string {
	var baseName = strings.TrimSpace(empiricalCase.CaseID)
	if len(empiricalCase.AssetIdentityKeys) > 1 {
		baseName += "--" + strings.TrimSpace(assetIdentityKey)
	}

	return path.Join(strings.TrimSpace(outputRoot), "golden", method.FilenameSlug(), baseName+".json")
}

// collectMissingGoldenFixturePaths returns the expected fixture paths that are
// still absent below the repository root.
// Authored by: OpenCode
func collectMissingGoldenFixturePaths(repositoryRoot string, expectedPaths []string) ([]string, error) {
	var missingPaths = make([]string, 0)
	var index int

	for index = range expectedPaths {
		var filesystemPath = filepath.Join(repositoryRoot, filepath.FromSlash(expectedPaths[index]))
		var _, err = os.Stat(filesystemPath)
		if err == nil {
			continue
		}
		if os.IsNotExist(err) {
			missingPaths = append(missingPaths, expectedPaths[index])
			continue
		}

		return nil, fmt.Errorf("stat empirical golden fixture %s: %w", expectedPaths[index], err)
	}

	return missingPaths, nil
}

// allowMissingOracleFixtureGeneration parses the opt-in missing-fixture
// generation environment value.
// Authored by: OpenCode
func allowMissingOracleFixtureGeneration(rawValue string) bool {
	switch strings.ToLower(strings.TrimSpace(rawValue)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// runMissingOracleFixtureGeneration executes the repository-owned oracle helper
// command from the repository root.
// Authored by: OpenCode
func runMissingOracleFixtureGeneration(ctx context.Context, repositoryRoot string) error {
	var command = exec.CommandContext(ctx, "go", "run", "./tools/empiricaloracle")
	command.Dir = repositoryRoot
	command.Env = os.Environ()

	var output, err = command.CombinedOutput()
	if err != nil {
		return oracleFixtureGenerationError{
			Command: MissingOracleFixtureGenerationCommand,
			Output:  string(output),
			Reason:  err,
		}
	}

	return nil
}
