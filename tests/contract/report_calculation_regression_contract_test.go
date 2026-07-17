// Package contract verifies the pinned report-calculation regression boundary.
// Authored by: OpenCode
package contract

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
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

	// calculationRegressionSeparator keeps package and test names unambiguous in maps.
	// Authored by: OpenCode
	calculationRegressionSeparator = "\x00"
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

// calculationRegressionCurrentCase stores one current test identity and execution result.
// Authored by: OpenCode
type calculationRegressionCurrentCase struct {
	sourcePath  string
	fingerprint string
	status      string
}

// calculationRegressionTestEvent is the structured event shape emitted by go test -json.
// Authored by: OpenCode
type calculationRegressionTestEvent struct {
	Action  string
	Package string
	Test    string
}

// TestCalculationRegression compares the current calculation test tree and empirical artifacts with R.
// Authored by: OpenCode
func TestCalculationRegression(t *testing.T) {
	t.Parallel()

	var repositoryRoot = calculationRegressionRepositoryRoot(t)
	var baseline, err = loadCalculationRegressionBaseline(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}

	var currentCases, testRunErr = discoverCurrentCalculationCases(repositoryRoot)
	var currentArtifacts, artifactErr = hashCurrentEmpiricalArtifacts(repositoryRoot)

	var mismatches = make([]string, 0)
	var regressionNumerator = compareCalculationRegressionCases(baseline.cases, currentCases, &mismatches)
	compareEmpiricalArtifacts(baseline.artifacts, currentArtifacts, &mismatches)

	t.Logf("R=%d/%d", regressionNumerator, len(baseline.cases))
	if testRunErr != nil {
		mismatches = append(mismatches, testRunErr.Error())
	}
	if artifactErr != nil {
		mismatches = append(mismatches, artifactErr.Error())
	}
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

	return baseline, nil
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

// discoverCurrentCalculationCases runs the three pinned suites and reads structured test events.
// Authored by: OpenCode
func discoverCurrentCalculationCases(repositoryRoot string) (map[string]calculationRegressionCurrentCase, error) {
	var command = exec.CommandContext(context.Background(),
		"go",
		"test",
		"-json",
		"-count=1",
		"./internal/report/basis",
		"./internal/report/calculate",
		"./tests/empirical",
	)
	command.Dir = repositoryRoot
	command.Env = calculationRegressionTestEnvironment()

	var output bytes.Buffer
	command.Stdout = &output
	var errOutput bytes.Buffer
	command.Stderr = &errOutput
	var runErr = command.Run()

	var events, parseErr = decodeCalculationRegressionEvents(output.Bytes())
	if parseErr != nil {
		return nil, fmt.Errorf("decode structured calculation test events: %w", parseErr)
	}

	var cases = leafCalculationRegressionCases(events, repositoryRoot)
	if runErr != nil {
		return cases, fmt.Errorf("calculation regression suites failed: %w", runErr)
	}

	return cases, nil
}

// calculationRegressionTestEnvironment disables toolchain and module downloads for the local run.
// Authored by: OpenCode
func calculationRegressionTestEnvironment() []string {
	var environment = append([]string(nil), os.Environ()...)
	environment = append(environment, "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	return environment
}

// decodeCalculationRegressionEvents decodes the stable JSON event stream emitted by go test.
// Authored by: OpenCode
func decodeCalculationRegressionEvents(content []byte) ([]calculationRegressionTestEvent, error) {
	var decoder = json.NewDecoder(bytes.NewReader(content))
	var events []calculationRegressionTestEvent
	for {
		var event calculationRegressionTestEvent
		var err = decoder.Decode(&event)
		if errors.Is(err, io.EOF) {
			return events, nil
		}
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
}

// leafCalculationRegressionCases keeps top-level tests without children and leaf subtests.
// Authored by: OpenCode
func leafCalculationRegressionCases(events []calculationRegressionTestEvent, repositoryRoot string) map[string]calculationRegressionCurrentCase {
	var runs = make(map[string]calculationRegressionCurrentCase)
	var children = make(map[string]struct{})
	for _, event := range events {
		if event.Package == "" || event.Test == "" {
			continue
		}
		var key = event.Package + calculationRegressionSeparator + event.Test
		if event.Action == "run" {
			runs[key] = calculationRegressionCurrentCase{status: "running"}
			if separatorIndex := strings.LastIndexByte(event.Test, '/'); separatorIndex >= 0 {
				var parentKey = event.Package + calculationRegressionSeparator + event.Test[:separatorIndex]
				children[parentKey] = struct{}{}
			}
			continue
		}
		if event.Action == "pass" || event.Action == "fail" || event.Action == "skip" {
			var current, found = runs[key]
			if found {
				current.status = event.Action
				runs[key] = current
			}
		}
	}

	var cases = make(map[string]calculationRegressionCurrentCase)
	for key, current := range runs {
		if _, hasChildren := children[key]; hasChildren {
			continue
		}
		var separatorIndex = strings.IndexByte(key, calculationRegressionSeparator[0])
		var packagePath = key[:separatorIndex]
		var testName = key[separatorIndex+len(calculationRegressionSeparator):]
		var sourcePath, fingerprint, err = calculationRegressionSourceFingerprint(repositoryRoot, packagePath, testName)
		if err != nil {
			current.status = "source-error: " + err.Error()
		} else {
			current.sourcePath = sourcePath
			current.fingerprint = fingerprint
		}
		cases[packagePath+"/"+testName] = current
	}

	return cases
}

// calculationRegressionSourceFingerprint hashes the canonical test declaration containing its expectations.
// Authored by: OpenCode
func calculationRegressionSourceFingerprint(repositoryRoot string, packagePath string, testName string) (string, string, error) {
	if !strings.HasPrefix(packagePath, calculationRegressionModulePath+"/") {
		return "", "", fmt.Errorf("unexpected package path %q", packagePath)
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

// compareCalculationRegressionCases compares fixed identities, source locations, hashes, and statuses.
// Authored by: OpenCode
func compareCalculationRegressionCases(
	baseline map[string]calculationRegressionBaselineCase,
	current map[string]calculationRegressionCurrentCase,
	mismatches *[]string,
) int {
	var numerator int
	var identifiers = make([]string, 0, len(baseline))
	for identifier := range baseline {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)

	for _, identifier := range identifiers {
		var expected = baseline[identifier]
		var actual, found = current[identifier]
		if !found {
			*mismatches = append(*mismatches, "missing or renamed baseline case "+identifier)
			continue
		}

		var matches = true
		if actual.sourcePath != expected.sourcePath {
			matches = false
			*mismatches = append(*mismatches, fmt.Sprintf("case %s moved: got %s want %s", identifier, actual.sourcePath, expected.sourcePath))
		}
		if actual.fingerprint != expected.fingerprint {
			matches = false
			*mismatches = append(*mismatches, fmt.Sprintf("case %s calculation expectation fingerprint changed: got %s want %s", identifier, actual.fingerprint, expected.fingerprint))
		}
		if actual.status != "pass" {
			matches = false
			*mismatches = append(*mismatches, fmt.Sprintf("case %s did not pass: status=%s", identifier, actual.status))
		}
		if matches {
			numerator++
		}
	}

	return numerator
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
