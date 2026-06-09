package empirical

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
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
	vendoredHledgerLicenseRepositoryPath        = "third_party/hledger/LICENSE"
	vendoredHledgerReadmeRepositoryPath         = "third_party/hledger/README.md"
	vendoredHledgerSourceMetadataRepositoryPath = "third_party/hledger/SOURCE.md"
	vendoredHledgerSourceArchiveRepositoryPath  = "third_party/hledger/source/hledger-source-1.99.2.tar.gz"
	vendoredHledgerLinuxBinaryRepositoryPath    = "third_party/hledger/bin/linux-amd64/hledger"

	vendoredHledgerSourceArchiveChecksum = "sha256:1ea46d762f973fed0550ae57aee38d8036a754bf1e3064a27b307e2ecbeaccdf"
	vendoredHledgerLinuxBinaryChecksum   = "sha256:801f1abfae1bf3b741567a7eea9ee6a4227544b6dcdb02b71b36ffcd26cec409"
)

// TestHledgerVendoringMaterialsAndMetadata verifies the repository contains the
// required vendored source, executable, license, checksum, version, and
// platform-support metadata for the empirical hledger boundary.
// Authored by: OpenCode
func TestHledgerVendoringMaterialsAndMetadata(t *testing.T) {
	t.Parallel()

	var repositoryRoot = empiricalRepositoryRoot(t)
	var licenseContent = mustReadRepositoryFile(t, repositoryRoot, vendoredHledgerLicenseRepositoryPath)
	var readmeContent = mustReadRepositoryFile(t, repositoryRoot, vendoredHledgerReadmeRepositoryPath)
	var sourceMetadataContent = mustReadRepositoryFile(t, repositoryRoot, vendoredHledgerSourceMetadataRepositoryPath)

	assertVendoringContentContainsAll(t, licenseContent, "GNU GENERAL PUBLIC LICENSE", "Version 3, 29 June 2007")

	assertVendoringContentContainsAll(
		t,
		readmeContent,
		"1.99.2",
		"prerelease",
		"1.52.1",
		"FIFO/LIFO/HIFO/AVERAGE lot + gain behavior",
		vendoredHledgerSourceArchiveRepositoryPath,
		vendoredHledgerLinuxBinaryRepositoryPath,
		"linux-amd64",
		"windows-amd64",
		"GPL-3.0-or-later",
		"Runtime application code under `cmd/` and `internal/` must not link, import, or",
		"execute hledger.",
		"Binary-only vendoring is invalid.",
		"actionable setup error",
	)

	assertVendoringContentContainsAll(
		t,
		sourceMetadataContent,
		"https://github.com/simonmichael/hledger",
		"https://github.com/simonmichael/hledger/releases/tag/1.99.2",
		"ad6068782cb03a0433546b80c62cd771a655ef15",
		"GPL-3.0-or-later",
		vendoredHledgerSourceArchiveRepositoryPath,
		vendoredHledgerSourceArchiveChecksum,
		vendoredHledgerLinuxBinaryRepositoryPath,
		vendoredHledgerLinuxBinaryChecksum,
		"sha256:4d94d701b1a9e82aa2ea1b9997ddadbd94fecba21b4bdce9f4c85e8c1a3d2b9e",
		"sha256:55bcb1d8341902f751d8b79b27d6f01f3c51cd7453996d7bd1eba17a2a567292",
		"sha256:bb5090978b84e9957fe2d7052703ec000f5a6161908ce0c3b813386450674bfe",
		"sha256:5c5881924727e2635a9f69f88191bce8ef924c23009688f305c3df13e1198ee2",
		"darwin-arm64",
		"darwin-amd64",
		"windows-amd64",
	)

	var sourceArchivePath = filepath.Join(repositoryRoot, filepath.FromSlash(vendoredHledgerSourceArchiveRepositoryPath))
	var sourceArchiveInfo, err = os.Stat(sourceArchivePath)
	if err != nil {
		t.Fatalf("stat vendored source archive %s: %v", vendoredHledgerSourceArchiveRepositoryPath, err)
	}
	if sourceArchiveInfo.IsDir() {
		t.Fatalf("vendored source archive %s must be a file", vendoredHledgerSourceArchiveRepositoryPath)
	}
	if sourceArchiveInfo.Size() == 0 {
		t.Fatalf("vendored source archive %s must not be empty", vendoredHledgerSourceArchiveRepositoryPath)
	}

	var sourceArchiveChecksum = mustFileSHA256(t, sourceArchivePath)
	if sourceArchiveChecksum != vendoredHledgerSourceArchiveChecksum {
		t.Fatalf("unexpected vendored source checksum: got %q want %q", sourceArchiveChecksum, vendoredHledgerSourceArchiveChecksum)
	}

	var linuxBinaryPath = filepath.Join(repositoryRoot, filepath.FromSlash(vendoredHledgerLinuxBinaryRepositoryPath))
	var linuxBinaryInfo, statErr = os.Stat(linuxBinaryPath)
	if statErr != nil {
		t.Fatalf("stat vendored executable %s: %v", vendoredHledgerLinuxBinaryRepositoryPath, statErr)
	}
	if linuxBinaryInfo.IsDir() {
		t.Fatalf("vendored executable %s must be a file", vendoredHledgerLinuxBinaryRepositoryPath)
	}
	if linuxBinaryInfo.Mode()&0o111 == 0 {
		t.Fatalf("vendored executable %s must be executable, got mode %v", vendoredHledgerLinuxBinaryRepositoryPath, linuxBinaryInfo.Mode())
	}

	var linuxBinaryChecksum = mustFileSHA256(t, linuxBinaryPath)
	if linuxBinaryChecksum != vendoredHledgerLinuxBinaryChecksum {
		t.Fatalf("unexpected vendored executable checksum: got %q want %q", linuxBinaryChecksum, vendoredHledgerLinuxBinaryChecksum)
	}
}

// TestHledgerVendoringRuntimeBoundary verifies runtime application code remains
// free of hledger imports and hledger string literals.
// Authored by: OpenCode
func TestHledgerVendoringRuntimeBoundary(t *testing.T) {
	t.Parallel()

	var repositoryRoot = empiricalRepositoryRoot(t)
	var violations []string
	var targetDirectory string
	for _, targetDirectory = range []string{"cmd", "internal"} {
		var walkRoot = filepath.Join(repositoryRoot, targetDirectory)
		var walkErr = filepath.WalkDir(walkRoot, func(currentPath string, directoryEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if directoryEntry.IsDir() {
				return nil
			}
			if filepath.Ext(currentPath) != ".go" || strings.HasSuffix(currentPath, "_test.go") {
				return nil
			}

			var fileSet = token.NewFileSet()
			var fileNode, parseErr = parser.ParseFile(fileSet, currentPath, nil, 0)
			if parseErr != nil {
				return parseErr
			}

			var relativePath, relativeErr = filepath.Rel(repositoryRoot, currentPath)
			if relativeErr != nil {
				return relativeErr
			}
			var repositoryPath = filepath.ToSlash(relativePath)

			var importSpec *ast.ImportSpec
			for _, importSpec = range fileNode.Imports {
				var importPath, unquoteErr = strconv.Unquote(importSpec.Path.Value)
				if unquoteErr != nil {
					return unquoteErr
				}
				if strings.Contains(strings.ToLower(importPath), "hledger") {
					violations = append(violations, repositoryPath+": import "+importPath)
				}
			}

			ast.Inspect(fileNode, func(node ast.Node) bool {
				var stringLiteral, ok = node.(*ast.BasicLit)
				if !ok || stringLiteral.Kind != token.STRING {
					return true
				}

				var literalValue, unquoteErr = strconv.Unquote(stringLiteral.Value)
				if unquoteErr != nil {
					violations = append(violations, repositoryPath+": invalid string literal "+stringLiteral.Value)
					return true
				}
				if strings.Contains(strings.ToLower(literalValue), "hledger") {
					violations = append(violations, repositoryPath+": string literal "+literalValue)
				}

				return true
			})

			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk runtime directory %s: %v", targetDirectory, walkErr)
		}
	}

	sort.Strings(violations)
	if len(violations) != 0 {
		t.Fatalf("runtime application code must not reference hledger: %v", violations)
	}
}

// empiricalRepositoryRoot resolves the repository root from this test file.
// Authored by: OpenCode
func empiricalRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repository root: runtime caller lookup failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

// mustReadRepositoryFile reads one repository file as UTF-8 text and fails the
// current test immediately on error.
// Authored by: OpenCode
func mustReadRepositoryFile(t *testing.T, repositoryRoot string, repositoryPath string) string {
	t.Helper()

	var filesystemPath = filepath.Join(repositoryRoot, filepath.FromSlash(repositoryPath))
	var content, err = os.ReadFile(filesystemPath)
	if err != nil {
		t.Fatalf("read %s: %v", repositoryPath, err)
	}

	return string(content)
}

// mustFileSHA256 computes the `sha256:` checksum string for one file.
// Authored by: OpenCode
func mustFileSHA256(t *testing.T, filesystemPath string) string {
	t.Helper()

	var fileHandle, err = os.Open(filesystemPath)
	if err != nil {
		t.Fatalf("open %s: %v", filesystemPath, err)
	}
	defer func() {
		_ = fileHandle.Close()
	}()

	var hasher = sha256.New()
	if _, err = io.Copy(hasher, fileHandle); err != nil {
		t.Fatalf("hash %s: %v", filesystemPath, err)
	}

	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

// assertVendoringContentContainsAll verifies one vendoring metadata file keeps
// every required contract fragment.
// Authored by: OpenCode
func assertVendoringContentContainsAll(t *testing.T, content string, wantSubstrings ...string) {
	t.Helper()

	var want string
	for _, want = range wantSubstrings {
		if !strings.Contains(content, want) {
			t.Fatalf("expected vendoring content to contain %q", want)
		}
	}
}
