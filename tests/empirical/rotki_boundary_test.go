package empirical

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// TestRepositoryControlledRotkiBoundaryArtifactsArePresent verifies BUG-001's
// repository-controlled rotki boundary files exist, parse, and remain
// synthetic-only.
// Authored by: OpenCode
func TestRepositoryControlledRotkiBoundaryArtifactsArePresent(t *testing.T) {
	t.Parallel()

	var repositoryRoot = empiricalRepositoryRoot(t)
	var artifactPaths = []string{
		"third_party/rotki/README.md",
		"third_party/rotki/LICENSE.md",
		"testdata/empirical/rotki/bootstrap-boundary.json",
		"testdata/empirical/rotki/fifo/case-fifo-alpha-2024.json",
		"testdata/empirical/rotki/lifo/case-lifo-beta-2024.json",
		"testdata/empirical/rotki/hifo/case-hifo-gamma-2024.json",
		"testdata/empirical/rotki/average-cost/case-average-cost-delta-2024.json",
		"testdata/empirical/rotki/average-cost/case-average-cost-reset-delta-2024.json",
		"testdata/empirical/rotki/average-cost/case-post-year-ignore-delta-2024.json",
		"testdata/empirical/rotki/scope-local-hybrid/case-scope-local-reliable-epsilon-2024.json",
		"testdata/empirical/rotki/scope-local-hybrid/case-scope-local-broadening-gamma-2024--asset-gamma.json",
		"testdata/empirical/rotki/scope-local-hybrid/case-scope-local-broadening-gamma-2024--asset-delta.json",
		"testdata/empirical/rotki/scope-local-hybrid/case-scope-local-reset-epsilon-2024.json",
	}

	var index int
	var manifestPayload map[string]any
	for index = range artifactPaths {
		var filesystemPath = filepath.Join(repositoryRoot, filepath.FromSlash(artifactPaths[index]))
		var rawContent, err = os.ReadFile(filesystemPath)
		if err != nil {
			t.Fatalf("read repository-controlled rotki boundary artifact %s: %v", artifactPaths[index], err)
		}
		if filepath.Base(filesystemPath) != "LICENSE.md" {
			if err = fixture.ValidateSyntheticOnlyContent(artifactPaths[index], string(rawContent)); err != nil {
				t.Fatalf("validate synthetic-only rotki boundary artifact %s: %v", artifactPaths[index], err)
			}
		}
		if filepath.Ext(filesystemPath) != ".json" {
			continue
		}

		var payload any
		if err = json.Unmarshal(rawContent, &payload); err != nil {
			t.Fatalf("decode repository-controlled rotki boundary artifact %s: %v", artifactPaths[index], err)
		}
		if artifactPaths[index] == "testdata/empirical/rotki/bootstrap-boundary.json" {
			var ok bool
			manifestPayload, ok = payload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected manifest payload type: %T", payload)
			}
		}
	}

	var datasetPath = filepath.Join(repositoryRoot, "testdata/empirical/financial-dataset.yaml")
	var rawDataset, err = os.ReadFile(datasetPath)
	if err != nil {
		t.Fatalf("read empirical dataset for manifest verification: %v", err)
	}
	var datasetSection, datasetSectionOK = manifestPayload["dataset"].(map[string]any)
	if !datasetSectionOK {
		t.Fatalf("rotki boundary manifest dataset section is missing or invalid: %#v", manifestPayload["dataset"])
	}
	var recordedHash, recordedHashOK = datasetSection["sha256"].(string)
	if !recordedHashOK || strings.TrimSpace(recordedHash) == "" {
		t.Fatalf("rotki boundary manifest dataset sha256 is missing: %#v", datasetSection)
	}
	var actualHash = strings.TrimPrefix(stableEmpiricalSHA256(rawDataset), "sha256:")
	if recordedHash != actualHash {
		t.Fatalf("rotki boundary manifest dataset sha256 mismatch: got %s want %s", recordedHash, actualHash)
	}
}

// stableEmpiricalSHA256 returns the canonical prefixed SHA-256 text used by the
// repository-controlled empirical boundary checks.
// Authored by: OpenCode
func stableEmpiricalSHA256(content []byte) string {
	var sum = sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
