package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

const (
	rotkiBoundaryRootRepositoryPath     = "testdata/empirical/rotki"
	rotkiBoundaryManifestRepositoryPath = "testdata/empirical/rotki/bootstrap-boundary.json"
	rotkiBoundaryReadmeRepositoryPath   = "third_party/rotki/README.md"
	rotkiBoundaryLicenseRepositoryPath  = "third_party/rotki/LICENSE.md"
	defaultCompositeRuleVersion         = "scope_local_hybrid_composite_v1"
	defaultPureOracleName               = "rotki"
	defaultHybridCompositeOracleName    = "scope_local_hybrid_composite"
	defaultRotkiSourceURL               = "https://github.com/rotki/rotki"
	defaultRotkiVersionOrCommit         = "a2e00be49a0ea36e7563a5d235cfa6a7c91edbfb"
)

// boundaryOracleInput stores one repository-controlled normalization input used
// to build one golden oracle fixture without requiring a local rotki install in
// this checkout.
// Authored by: OpenCode
type boundaryOracleInput struct {
	CaseID               string                          `json:"case_id"`
	Method               reportmodel.CostBasisMethod     `json:"method"`
	Year                 int                             `json:"year"`
	AssetIdentityKey     string                          `json:"asset_identity_key"`
	OracleName           string                          `json:"oracle_name"`
	SourceURL            string                          `json:"source_url"`
	VersionOrCommit      string                          `json:"version_or_commit"`
	AdapterArguments     []string                        `json:"adapter_arguments"`
	AdapterConstraints   []string                        `json:"adapter_constraints"`
	CompositeRuleVersion string                          `json:"composite_rule_version,omitempty"`
	Values               comparableOutputValuesInput     `json:"values"`
	Matches              []oracleMatchEvidenceInput      `json:"matches"`
	UnsupportedSegments  []unsupportedOracleSegmentInput `json:"unsupported_segments"`
	FinancialTolerances  map[string]string               `json:"financial_tolerances"`
	ToleranceNotes       map[string]string               `json:"tolerance_notes"`
}

// rotkiBoundaryManifest stores the repository-controlled bootstrap manifest
// metadata required by boundary verification.
// Authored by: OpenCode
type rotkiBoundaryManifest struct {
	Dataset struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"dataset"`
}

// isRepositoryControlledBoundaryMethod reports whether fixture regeneration for
// the method should use repository-controlled boundary inputs instead of the
// retained hledger execution path.
// Authored by: OpenCode
func isRepositoryControlledBoundaryMethod(method reportmodel.CostBasisMethod) bool {
	switch method {
	case reportmodel.CostBasisMethodFIFO,
		reportmodel.CostBasisMethodLIFO,
		reportmodel.CostBasisMethodHIFO,
		reportmodel.CostBasisMethodAverageCost,
		reportmodel.CostBasisMethodScopeLocalHybrid:
		return true
	default:
		return false
	}
}

// verifyRotkiBoundaryMaterials checks that the repository-controlled rotki
// boundary files required by BUG-001 are present before regeneration uses them.
// Authored by: OpenCode
func verifyRotkiBoundaryMaterials(repositoryRoot string, dataset fixture.EmpiricalDataset) error {
	var requiredRepositoryPaths = []string{
		rotkiBoundaryReadmeRepositoryPath,
		rotkiBoundaryLicenseRepositoryPath,
		rotkiBoundaryManifestRepositoryPath,
	}
	var index int
	for index = range requiredRepositoryPaths {
		var filesystemPath = filepath.Join(repositoryRoot, filepath.FromSlash(requiredRepositoryPaths[index]))
		if _, err := os.Stat(filesystemPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("repository-controlled rotki boundary is missing required file %s", requiredRepositoryPaths[index])
			}
			return fmt.Errorf("stat repository-controlled rotki boundary file %s: %w", requiredRepositoryPaths[index], err)
		}
	}

	var manifestPath = filepath.Join(repositoryRoot, filepath.FromSlash(rotkiBoundaryManifestRepositoryPath))
	var rawManifest, err = os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read repository-controlled rotki boundary manifest: %w", err)
	}
	var manifest rotkiBoundaryManifest
	if err = json.Unmarshal(rawManifest, &manifest); err != nil {
		return fmt.Errorf("decode repository-controlled rotki boundary manifest: %w", err)
	}
	if strings.TrimSpace(manifest.Dataset.Path) == "" || strings.TrimSpace(manifest.Dataset.SHA256) == "" {
		return fmt.Errorf("repository-controlled rotki boundary manifest must record dataset path and sha256")
	}
	var manifestDatasetPath = filepath.Join(repositoryRoot, filepath.FromSlash(strings.TrimSpace(manifest.Dataset.Path)))
	var rawDataset []byte
	rawDataset, err = os.ReadFile(manifestDatasetPath)
	if err != nil {
		return fmt.Errorf("read manifest dataset %s: %w", manifest.Dataset.Path, err)
	}
	var actualDatasetHash = strings.TrimPrefix(stablePrefixedSHA256Hash(rawDataset), "sha256:")
	if strings.TrimSpace(manifest.Dataset.SHA256) != actualDatasetHash {
		return fmt.Errorf("repository-controlled rotki boundary manifest dataset sha256 mismatch: expected %s got %s", strings.TrimSpace(manifest.Dataset.SHA256), actualDatasetHash)
	}

	var caseIndex int
	for caseIndex = range dataset.Cases {
		var empiricalCase = dataset.Cases[caseIndex]
		if empiricalCase.OracleSupport == fixture.OracleSupportUnsupported {
			continue
		}

		var methodIndex int
		for methodIndex = range empiricalCase.Methods {
			var method = empiricalCase.Methods[methodIndex]
			if !isRepositoryControlledBoundaryMethod(method) {
				continue
			}

			var assetIndex int
			for assetIndex = range empiricalCase.AssetIdentityKeys {
				var relativePath = boundaryOracleInputRelativePath(empiricalCase, method, empiricalCase.AssetIdentityKeys[assetIndex])
				var filesystemPath = filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
				if _, err := os.Stat(filesystemPath); err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("repository-controlled rotki boundary input is missing: %s", relativePath)
					}
					return fmt.Errorf("stat repository-controlled rotki boundary input %s: %w", relativePath, err)
				}
			}
		}
	}

	return nil
}

// loadBoundaryOracleInput reads one repository-controlled normalization input.
// Authored by: OpenCode
func loadBoundaryOracleInput(repositoryRoot string, empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod, assetIdentityKey string) (boundaryOracleInput, string, []byte, error) {
	var relativePath = boundaryOracleInputRelativePath(empiricalCase, method, assetIdentityKey)
	var filesystemPath = filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	var rawContent, err = os.ReadFile(filesystemPath)
	if err != nil {
		return boundaryOracleInput{}, "", nil, fmt.Errorf("read repository-controlled boundary input %s: %w", relativePath, err)
	}

	var input boundaryOracleInput
	if err = json.Unmarshal(rawContent, &input); err != nil {
		return boundaryOracleInput{}, "", nil, fmt.Errorf("decode repository-controlled boundary input %s: %w", relativePath, err)
	}

	if strings.TrimSpace(input.CaseID) == "" || input.Method == "" || input.Year <= 0 || strings.TrimSpace(input.AssetIdentityKey) == "" {
		return boundaryOracleInput{}, "", nil, fmt.Errorf("repository-controlled boundary input %s is missing required identity fields", relativePath)
	}

	return input, relativePath, rawContent, nil
}

// boundaryOracleInputRelativePath returns the repository-relative normalization
// input path for one case, method, and asset.
// Authored by: OpenCode
func boundaryOracleInputRelativePath(empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod, assetIdentityKey string) string {
	var baseName = strings.TrimSpace(empiricalCase.CaseID)
	if len(empiricalCase.AssetIdentityKeys) > 1 {
		baseName += "--" + strings.TrimSpace(assetIdentityKey)
	}

	return path.Join(rotkiBoundaryRootRepositoryPath, method.FilenameSlug(), baseName+".json")
}

// buildBoundaryOracleOutputForAsset converts one repository-controlled boundary
// input into one normalized oracle fixture.
// Authored by: OpenCode
func buildBoundaryOracleOutputForAsset(
	dataset fixture.EmpiricalDataset,
	datasetInputHash string,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	assetIdentityKey string,
	input boundaryOracleInput,
	inputRelativePath string,
	rawInput []byte,
) (fixture.OracleOutput, error) {
	var oracleName = strings.TrimSpace(input.OracleName)
	if oracleName == "" {
		oracleName = defaultPureOracleName
		if method == reportmodel.CostBasisMethodScopeLocalHybrid {
			oracleName = defaultHybridCompositeOracleName
		}
	}
	var sourceURL = strings.TrimSpace(input.SourceURL)
	if sourceURL == "" {
		sourceURL = defaultRotkiSourceURL
	}
	var versionOrCommit = strings.TrimSpace(input.VersionOrCommit)
	if versionOrCommit == "" {
		versionOrCommit = defaultRotkiVersionOrCommit
	}
	var adapterArguments = copyStringSlice(input.AdapterArguments)
	if len(adapterArguments) == 0 {
		adapterArguments = []string{"--boundary", rotkiBoundaryManifestRepositoryPath, "--input", inputRelativePath, "--method", method.FilenameSlug()}
	}
	var adapterConstraints = copyStringSlice(input.AdapterConstraints)
	if len(adapterConstraints) == 0 {
		adapterConstraints = []string{"repository-controlled normalization input"}
	}

	var compositeRuleVersion = strings.TrimSpace(input.CompositeRuleVersion)
	if method == reportmodel.CostBasisMethodScopeLocalHybrid && compositeRuleVersion == "" {
		compositeRuleVersion = defaultCompositeRuleVersion
	}

	return normalizeOracleOutput(oracleOutputNormalizationInput{
		DatasetVersion:      strings.TrimSpace(dataset.DatasetVersion),
		CaseID:              strings.TrimSpace(empiricalCase.CaseID),
		Method:              method,
		Year:                empiricalCase.Year,
		AssetIdentityKey:    strings.TrimSpace(assetIdentityKey),
		Values:              input.Values,
		Matches:             input.Matches,
		UnsupportedSegments: input.UnsupportedSegments,
		Metadata: oracleGenerationMetadataInput{
			OracleName:              oracleName,
			SourceURL:               sourceURL,
			VersionOrCommit:         versionOrCommit,
			AdapterArguments:        adapterArguments,
			AdapterConstraints:      adapterConstraints,
			DatasetInputHash:        strings.TrimSpace(datasetInputHash),
			ExternalOracleInputHash: stablePrefixedSHA256Hash(rawInput),
			DecimalPolicy:           oracleDecimalPolicy,
			CompositeRuleVersion:    compositeRuleVersion,
			FinancialTolerances:     copyStringMap(input.FinancialTolerances),
			ToleranceNotes:          copyStringMap(input.ToleranceNotes),
		},
	})
}
