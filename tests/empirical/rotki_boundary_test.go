package empirical

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// TestRotkiBoundaryMetadataStaysRepositoryControlled verifies BUG-002 keeps only
// pinned provenance metadata in the repository while raw oracle captures stay
// out of version control.
// Authored by: OpenCode
func TestRotkiBoundaryMetadataStaysRepositoryControlled(t *testing.T) {
	t.Parallel()

	var repositoryRoot = empiricalRepositoryRoot(t)
	var metadataPaths = []string{
		"third_party/rotki/README.md",
		"third_party/rotki/LICENSE.md",
		"testdata/empirical/rotki/README.md",
	}

	var index int
	for index = range metadataPaths {
		var filesystemPath = filepath.Join(repositoryRoot, filepath.FromSlash(metadataPaths[index]))
		var rawContent, err = os.ReadFile(filesystemPath)
		if err != nil {
			t.Fatalf("read rotki boundary metadata %s: %v", metadataPaths[index], err)
		}
		if filepath.Base(filesystemPath) != "LICENSE.md" {
			if err = fixture.ValidateSyntheticOnlyContent(metadataPaths[index], string(rawContent)); err != nil {
				t.Fatalf("validate synthetic-only rotki boundary metadata %s: %v", metadataPaths[index], err)
			}
		}
	}

	var rotkiArtifactRoot = filepath.Join(repositoryRoot, filepath.FromSlash("testdata/empirical/rotki"))
	var directoryEntries, err = os.ReadDir(rotkiArtifactRoot)
	if err != nil {
		t.Fatalf("read rotki metadata directory: %v", err)
	}
	for index = range directoryEntries {
		if directoryEntries[index].IsDir() {
			continue
		}
		if directoryEntries[index].Name() == "README.md" {
			continue
		}
		t.Fatalf("committed raw rotki artifact %s must be removed after BUG-002", filepath.ToSlash(filepath.Join("testdata/empirical/rotki", directoryEntries[index].Name())))
	}
}

// TestRotkiBoundaryRejectsVendoredSourceAndGlobalInstallAssumptions verifies
// the repository keeps the verified rotki boundary in the untracked cache path
// only and does not assume a developer-global rotki executable.
// Authored by: OpenCode
func TestRotkiBoundaryRejectsVendoredSourceAndGlobalInstallAssumptions(t *testing.T) {
	t.Parallel()

	var repositoryRoot = empiricalRepositoryRoot(t)
	var vendoredSourcePath = filepath.Join(repositoryRoot, filepath.FromSlash("third_party/rotki/source"))
	if _, err := os.Stat(vendoredSourcePath); err == nil {
		t.Fatalf("vendored rotki source must not exist at %s", filepath.ToSlash(vendoredSourcePath))
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat vendored rotki source path: %v", err)
	}

	var sourcePath = filepath.Join(repositoryRoot, filepath.FromSlash("tools/empiricaloracle/rotki_source.go"))
	var rawSource, err = os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read rotki source runtime: %v", err)
	}
	var sourceText = string(rawSource)
	if strings.Contains(sourceText, "LookPath(\"rotki\"") || strings.Contains(sourceText, "lookPath(\"rotki\"") {
		t.Fatal("rotki regeneration must not depend on a developer-global rotki executable")
	}
	if !strings.Contains(sourceText, "python3") || !strings.Contains(sourceText, "python") {
		t.Fatal("rotki regeneration must execute the project-owned Python adapter through a local Python runtime")
	}

	var gitignorePath = filepath.Join(repositoryRoot, ".gitignore")
	var rawGitignore, readErr = os.ReadFile(gitignorePath)
	if readErr != nil {
		t.Fatalf("read .gitignore: %v", readErr)
	}
	if !strings.Contains(string(rawGitignore), ".cache/empiricaloracle/rotki-source/") {
		t.Fatal("expected .gitignore to cover the untracked rotki source cache path")
	}
}
