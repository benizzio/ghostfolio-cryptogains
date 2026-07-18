// Package contract verifies the pinned report-calculation regression boundary.
// Authored by: OpenCode
package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	// calculationRegressionBaselinePath identifies the checked-in baseline data.
	// Authored by: OpenCode
	calculationRegressionBaselinePath = "tests/contract/testdata/report_calculation_regression_baseline.txt"

	// calculationRegressionBaselineCommit is the reviewed source of the frozen R population.
	// Authored by: OpenCode
	calculationRegressionBaselineCommit = "b7de13e597332ca8a1c36af3e05685217ab25f18"

	// calculationRegressionModulePath identifies the local module import prefix.
	// Authored by: OpenCode
	calculationRegressionModulePath = "github.com/benizzio/ghostfolio-cryptogains"
)

// calculationRegressionBaseline stores the pinned case and empirical-artifact fingerprints.
// Authored by: OpenCode
type calculationRegressionBaseline struct {
	commit    string
	cases     map[string]calculationRegressionBaselineCase
	artifacts map[string]string
}

// calculationRegressionBaselineCase stores one frozen regression identity and expectation hash.
// Authored by: OpenCode
type calculationRegressionBaselineCase struct {
	sourcePath  string
	fingerprint string
}

// TestCalculationRegression validates the pinned calculation identities and empirical artifacts.
// Authored by: OpenCode
func TestCalculationRegression(t *testing.T) {
	t.Parallel()

	var repositoryRoot = calculationRegressionRepositoryRoot(t)
	var baseline, err = loadCalculationRegressionBaseline(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}

	var regressionNumerator, mismatches = validateCalculationRegressionBaseline(repositoryRoot, baseline)

	t.Logf("R=%d/%d", regressionNumerator, len(baseline.cases))
	if len(mismatches) != 0 {
		sort.Strings(mismatches)
		t.Fatalf("calculation regression contract failed (R=%d/%d):\n- %s", regressionNumerator, len(baseline.cases), strings.Join(mismatches, "\n- "))
	}
}

// calculationRegressionRepositoryRoot resolves the repository root from this test file.
// Authored by: OpenCode
func calculationRegressionRepositoryRoot(t *testing.T) string {
	t.Helper()

	var _, currentFile, _, ok = runtime.Caller(0)
	if !ok {
		t.Fatal("resolve calculation regression repository root: runtime caller lookup failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

// loadCalculationRegressionBaseline parses and validates the frozen baseline data file.
// Authored by: OpenCode
func loadCalculationRegressionBaseline(repositoryRoot string) (calculationRegressionBaseline, error) {
	var path = filepath.Join(repositoryRoot, filepath.FromSlash(calculationRegressionBaselinePath))
	// #nosec G304 -- the path is derived from the repository root and fixed baseline path.
	var content, err = os.ReadFile(path)
	if err != nil {
		return calculationRegressionBaseline{}, fmt.Errorf("read calculation regression baseline: %w", err)
	}

	var baseline = calculationRegressionBaseline{
		cases:     make(map[string]calculationRegressionBaselineCase),
		artifacts: make(map[string]string),
	}
	var declaredCaseCount = -1
	var declaredArtifactCount = -1
	var lines = strings.Split(string(content), "\n")
	var lineNumber int
	for lineNumber = range lines {
		var line = strings.TrimSpace(lines[lineNumber])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, "=") {
			var metadata = strings.SplitN(line, "=", 2)
			if len(metadata) != 2 {
				return calculationRegressionBaseline{}, fmt.Errorf("invalid baseline metadata at line %d", lineNumber+1)
			}
			switch metadata[0] {
			case "format":
				if metadata[1] != "1" {
					return calculationRegressionBaseline{}, fmt.Errorf("unsupported calculation regression baseline format %q", metadata[1])
				}
			case "baseline_commit":
				baseline.commit = metadata[1]
			case "case_count":
				declaredCaseCount, err = strconv.Atoi(metadata[1])
				if err != nil {
					return calculationRegressionBaseline{}, fmt.Errorf("parse baseline case count: %w", err)
				}
			case "artifact_count":
				declaredArtifactCount, err = strconv.Atoi(metadata[1])
				if err != nil {
					return calculationRegressionBaseline{}, fmt.Errorf("parse baseline artifact count: %w", err)
				}
			default:
				return calculationRegressionBaseline{}, fmt.Errorf("unknown calculation regression baseline metadata %q at line %d", metadata[0], lineNumber+1)
			}
			continue
		}

		var fields = strings.Split(line, "|")
		switch fields[0] {
		case "case":
			if len(fields) != 4 {
				return calculationRegressionBaseline{}, fmt.Errorf("invalid baseline case at line %d", lineNumber+1)
			}
			if err = parseCalculationRegressionDigest(fields[3]); err != nil {
				return calculationRegressionBaseline{}, fmt.Errorf("validate case fingerprint at line %d: %w", lineNumber+1, err)
			}
			if _, found := baseline.cases[fields[1]]; found {
				return calculationRegressionBaseline{}, fmt.Errorf("duplicate baseline case %q", fields[1])
			}
			baseline.cases[fields[1]] = calculationRegressionBaselineCase{sourcePath: fields[2], fingerprint: fields[3]}
		case "artifact":
			if len(fields) != 3 {
				return calculationRegressionBaseline{}, fmt.Errorf("invalid baseline artifact at line %d", lineNumber+1)
			}
			if err = parseCalculationRegressionDigest(fields[2]); err != nil {
				return calculationRegressionBaseline{}, fmt.Errorf("validate artifact fingerprint at line %d: %w", lineNumber+1, err)
			}
			if _, found := baseline.artifacts[fields[1]]; found {
				return calculationRegressionBaseline{}, fmt.Errorf("duplicate baseline artifact %q", fields[1])
			}
			baseline.artifacts[fields[1]] = fields[2]
		default:
			return calculationRegressionBaseline{}, fmt.Errorf("unknown calculation regression baseline record %q at line %d", fields[0], lineNumber+1)
		}
	}

	if baseline.commit != calculationRegressionBaselineCommit {
		return calculationRegressionBaseline{}, fmt.Errorf("calculation regression baseline commit is %q, want %q", baseline.commit, calculationRegressionBaselineCommit)
	}
	if declaredCaseCount != len(baseline.cases) || declaredArtifactCount != len(baseline.artifacts) {
		return calculationRegressionBaseline{}, fmt.Errorf("calculation regression baseline counts are cases=%d/%d artifacts=%d/%d", len(baseline.cases), declaredCaseCount, len(baseline.artifacts), declaredArtifactCount)
	}
	if len(baseline.cases) == 0 || len(baseline.artifacts) == 0 {
		return calculationRegressionBaseline{}, fmt.Errorf("calculation regression baseline must contain non-empty cases and artifacts")
	}
	if err = rejectCalculationRegressionParentIdentities(baseline.cases); err != nil {
		return calculationRegressionBaseline{}, err
	}

	return baseline, nil
}

// rejectCalculationRegressionParentIdentities prevents a top-level test from
// being recorded alongside one of its leaf subtests.
// Authored by: OpenCode
func rejectCalculationRegressionParentIdentities(cases map[string]calculationRegressionBaselineCase) error {
	var identifiers = make([]string, 0, len(cases))
	for identifier := range cases {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)

	for _, identifier := range identifiers {
		for _, otherIdentifier := range identifiers {
			if identifier == otherIdentifier || !strings.HasPrefix(otherIdentifier, identifier+"/") {
				continue
			}
			return fmt.Errorf("calculation regression baseline records parent identity %q alongside child %q", identifier, otherIdentifier)
		}
	}

	return nil
}

// parseCalculationRegressionDigest validates one hexadecimal SHA-256 digest.
// Authored by: OpenCode
func parseCalculationRegressionDigest(value string) error {
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("digest has length %d, want %d", len(value), sha256.Size*2)
	}

	_, err := hex.DecodeString(value)
	if err != nil {
		return err
	}

	return nil
}

// validateCalculationRegressionBaseline checks the current declarations and
// empirical artifacts without executing any owner test package.
// Authored by: OpenCode
func validateCalculationRegressionBaseline(repositoryRoot string, baseline calculationRegressionBaseline) (int, []string) {
	var mismatches = make([]string, 0)
	var regressionNumerator int
	var identifiers = make([]string, 0, len(baseline.cases))
	for identifier := range baseline.cases {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)

	for _, identifier := range identifiers {
		var expected = baseline.cases[identifier]
		var actualSourcePath, actualFingerprint, err = calculationRegressionSourceFingerprint(repositoryRoot, identifier)
		if err != nil {
			mismatches = append(mismatches, fmt.Sprintf("case %s static identity validation failed: %v", identifier, err))
			continue
		}
		var matches = true
		if actualSourcePath != expected.sourcePath {
			matches = false
			mismatches = append(mismatches, fmt.Sprintf("case %s moved: got %s want %s", identifier, actualSourcePath, expected.sourcePath))
		}
		if actualFingerprint != expected.fingerprint {
			matches = false
			mismatches = append(mismatches, fmt.Sprintf("case %s calculation expectation fingerprint changed: got %s want %s", identifier, actualFingerprint, expected.fingerprint))
		}
		if matches {
			regressionNumerator++
		}
	}

	var currentArtifacts, artifactErr = hashCurrentEmpiricalArtifacts(repositoryRoot)
	if artifactErr != nil {
		mismatches = append(mismatches, artifactErr.Error())
	} else {
		compareEmpiricalArtifacts(baseline.artifacts, currentArtifacts, &mismatches)
	}

	return regressionNumerator, mismatches
}

// calculationRegressionSourceFingerprint hashes the canonical test declaration containing its expectations.
// Authored by: OpenCode
func calculationRegressionSourceFingerprint(repositoryRoot string, identifier string) (string, string, error) {
	var packagePath string
	var testName string
	for _, packageRelativePath := range []string{
		"internal/report/basis",
		"internal/report/calculate",
		"tests/empirical",
	} {
		var prefix = calculationRegressionModulePath + "/" + packageRelativePath + "/"
		if strings.HasPrefix(identifier, prefix) {
			packagePath = strings.TrimSuffix(prefix, "/")
			testName = strings.TrimPrefix(identifier, prefix)
			break
		}
	}
	if packagePath == "" || testName == "" {
		return "", "", fmt.Errorf("unexpected calculation regression identity %q", identifier)
	}
	var packageRelativePath = strings.TrimPrefix(packagePath, calculationRegressionModulePath+"/")
	var packageDirectory = filepath.Join(repositoryRoot, filepath.FromSlash(packageRelativePath))
	var testPaths, err = filepath.Glob(filepath.Join(packageDirectory, "*_test.go"))
	if err != nil {
		return "", "", err
	}
	sort.Strings(testPaths)
	var fileSet = token.NewFileSet()
	for _, testPath := range testPaths {
		var file, parseErr = parser.ParseFile(fileSet, testPath, nil, parser.ParseComments)
		if parseErr != nil {
			return "", "", parseErr
		}
		for _, declaration := range file.Decls {
			var function, ok = declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != strings.SplitN(testName, "/", 2)[0] {
				continue
			}
			var formatted bytes.Buffer
			if err = format.Node(&formatted, fileSet, function); err != nil {
				return "", "", err
			}
			var digest = sha256.Sum256(formatted.Bytes())
			var sourcePath, relativeErr = filepath.Rel(filepath.Dir(filepath.Dir(packageDirectory)), testPath)
			if relativeErr != nil {
				return "", "", relativeErr
			}
			return filepath.ToSlash(sourcePath), hex.EncodeToString(digest[:]), nil
		}
	}

	return "", "", fmt.Errorf("test function %q not found in %s", testName, packageDirectory)
}

// hashCurrentEmpiricalArtifacts hashes every repository-controlled empirical file.
// Authored by: OpenCode
func hashCurrentEmpiricalArtifacts(repositoryRoot string) (map[string]string, error) {
	var artifacts = make(map[string]string)
	var artifactRoot = filepath.Join(repositoryRoot, "testdata", "empirical")
	var err = filepath.WalkDir(artifactRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		// #nosec G122,G304 -- WalkDir restricts the path to the repository-controlled empirical tree.
		var content, readErr = os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var digest = sha256.Sum256(content)
		var relativePath, relativeErr = filepath.Rel(repositoryRoot, path)
		if relativeErr != nil {
			return relativeErr
		}
		artifacts[filepath.ToSlash(relativePath)] = hex.EncodeToString(digest[:])
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hash empirical artifacts: %w", err)
	}

	return artifacts, nil
}

// compareEmpiricalArtifacts compares the complete empirical artifact path set and raw hashes.
// Authored by: OpenCode
func compareEmpiricalArtifacts(baseline map[string]string, current map[string]string, mismatches *[]string) {
	var paths = make(map[string]struct{}, len(baseline)+len(current))
	for path := range baseline {
		paths[path] = struct{}{}
	}
	for path := range current {
		paths[path] = struct{}{}
	}

	var orderedPaths = make([]string, 0, len(paths))
	for path := range paths {
		orderedPaths = append(orderedPaths, path)
	}
	sort.Strings(orderedPaths)
	for _, path := range orderedPaths {
		var expected, expectedFound = baseline[path]
		var actual, actualFound = current[path]
		switch {
		case !expectedFound:
			*mismatches = append(*mismatches, "unexpected empirical artifact "+path)
		case !actualFound:
			*mismatches = append(*mismatches, "missing empirical artifact "+path)
		case actual != expected:
			*mismatches = append(*mismatches, fmt.Sprintf("empirical artifact fingerprint changed for %s: got %s want %s", path, actual, expected))
		}
	}
}
