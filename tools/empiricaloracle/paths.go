// Package main contains repository path resolution and empirical golden fixture
// path helpers.
//
// Authored by: OpenCode
package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

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
func goldenFixtureRelativePath(outputRoot string, empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod, assetIdentityKey string) (string, error) {
	var caseID, err = validateGoldenFixturePathComponent("case_id", empiricalCase.CaseID)
	if err != nil {
		return "", err
	}

	var assetKey string
	assetKey, err = validateGoldenFixturePathComponent("asset_identity_key", assetIdentityKey)
	if err != nil {
		return "", err
	}

	var baseName = caseID
	if len(empiricalCase.AssetIdentityKeys) > 1 {
		baseName += "--" + assetKey
	}

	var goldenRoot = path.Clean(path.Join(strings.TrimSpace(outputRoot), "golden"))
	var relativePath = path.Clean(path.Join(goldenRoot, method.FilenameSlug(), baseName+".json"))
	if !pathStaysUnder(relativePath, goldenRoot) {
		return "", fmt.Errorf("golden fixture path %s escapes %s", relativePath, goldenRoot)
	}

	return relativePath, nil
}

// validateGoldenFixturePathComponent returns one trimmed filename component only
// when it cannot alter the golden fixture path structure.
// Authored by: OpenCode
func validateGoldenFixturePathComponent(label string, rawValue string) (string, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "" {
		return "", fmt.Errorf("%s is required for a golden fixture filename component", label)
	}
	if trimmedValue == "." || trimmedValue == ".." || strings.Contains(trimmedValue, "..") {
		return "", fmt.Errorf("%s %q must not contain traversal sequences", label, trimmedValue)
	}
	if strings.Contains(trimmedValue, "/") || strings.Contains(trimmedValue, "\\") {
		return "", fmt.Errorf("%s %q must not contain path separators", label, trimmedValue)
	}

	return trimmedValue, nil
}

// pathStaysUnder reports whether a cleaned path remains at or below a cleaned
// root path.
// Authored by: OpenCode
func pathStaysUnder(cleanedPath string, cleanedRoot string) bool {
	return cleanedPath == cleanedRoot || strings.HasPrefix(cleanedPath, cleanedRoot+"/")
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
			var relativePath, err = goldenFixtureRelativePath(outputRoot, empiricalCase, method, empiricalCase.AssetIdentityKeys[assetIndex])
			if err != nil {
				return nil, fmt.Errorf("build golden fixture path for case %s method %s asset %s: %w", empiricalCase.CaseID, method, empiricalCase.AssetIdentityKeys[assetIndex], err)
			}

			allPaths = append(allPaths, relativePath)
		}

		return allPaths, nil
	}

	var missingPaths = make([]string, 0)
	var assetIndex int
	for assetIndex = range empiricalCase.AssetIdentityKeys {
		var relativePath, err = goldenFixtureRelativePath(outputRoot, empiricalCase, method, empiricalCase.AssetIdentityKeys[assetIndex])
		if err != nil {
			return nil, fmt.Errorf("build golden fixture path for case %s method %s asset %s: %w", empiricalCase.CaseID, method, empiricalCase.AssetIdentityKeys[assetIndex], err)
		}

		var exists bool
		exists, err = artifactExists(repositoryRoot, relativePath)
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
