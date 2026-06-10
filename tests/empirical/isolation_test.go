package empirical

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// empiricalForbiddenImportRule stores one static import boundary enforced over
// the empirical test package tree.
// Authored by: OpenCode
type empiricalForbiddenImportRule struct {
	ImportPath string
	Reason     string
}

// empiricalForbiddenIdentifierRule stores one static identifier or string-literal
// boundary enforced over the empirical test package tree.
// Authored by: OpenCode
type empiricalForbiddenIdentifierRule struct {
	Match  string
	Reason string
}

// TestEmpiricalIsolationRejectsForbiddenImports verifies the empirical test
// suite stays outside transport, TUI, snapshot, Markdown, and report-output
// package boundaries.
// Authored by: OpenCode
func TestEmpiricalIsolationRejectsForbiddenImports(t *testing.T) {
	t.Parallel()

	var rules = []empiricalForbiddenImportRule{
		{ImportPath: "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio", Reason: "Ghostfolio transport and DTO code is outside the empirical calculation boundary"},
		{ImportPath: "github.com/benizzio/ghostfolio-cryptogains/internal/tui", Reason: "TUI rendering is outside the empirical calculation boundary"},
		{ImportPath: "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot", Reason: "protected snapshot storage is outside the empirical calculation boundary"},
		{ImportPath: "github.com/benizzio/ghostfolio-cryptogains/internal/report/markdown", Reason: "Markdown rendering is outside the empirical calculation boundary"},
		{ImportPath: "github.com/benizzio/ghostfolio-cryptogains/internal/report/output", Reason: "report output writers and openers are outside the empirical calculation boundary"},
	}

	assertEmpiricalImportIsolation(t, rules)
}

// TestEmpiricalIsolationRejectsOutputArtifacts verifies the empirical test suite
// does not reference report-format identifiers, output filenames, or Documents-
// path behavior.
// Authored by: OpenCode
func TestEmpiricalIsolationRejectsOutputArtifacts(t *testing.T) {
	t.Parallel()

	var rules = []empiricalForbiddenIdentifierRule{
		{Match: "ReportDocument", Reason: "report document formatting is outside the empirical calculation boundary"},
		{Match: "ReportOutputFile", Reason: "report output metadata is outside the empirical calculation boundary"},
		{Match: "WriteReportDocument", Reason: "report file writing is outside the empirical calculation boundary"},
		{Match: "OpenPath", Reason: "OS opener behavior is outside the empirical calculation boundary"},
		{Match: "ResolveDocumentsDirectory", Reason: "Documents directory handling is outside the empirical calculation boundary"},
		{Match: "reserveReportFile", Reason: "generated report filename reservation is outside the empirical calculation boundary"},
		{Match: "buildReportFilenameBase", Reason: "generated report filename formatting is outside the empirical calculation boundary"},
		{Match: "ghostfolio-capital-gains", Reason: "generated report filenames are outside the empirical calculation boundary"},
		{Match: "Documents", Reason: "Documents-folder paths are outside the empirical calculation boundary"},
	}

	assertEmpiricalIdentifierIsolation(t, rules)
}

// assertEmpiricalImportIsolation scans empirical Go files for forbidden imports.
// Authored by: OpenCode
func assertEmpiricalImportIsolation(t *testing.T, rules []empiricalForbiddenImportRule) {
	t.Helper()

	var repositoryRoot = empiricalRepositoryRoot(t)
	var files = empiricalSourceFiles(t, repositoryRoot)
	var violations = make([]string, 0)
	var fileIndex int

	for fileIndex = range files {
		var fileSet = token.NewFileSet()
		var fileNode, err = parser.ParseFile(fileSet, files[fileIndex], nil, 0)
		if err != nil {
			t.Fatalf("parse empirical source file %s: %v", empiricalRelativePath(t, repositoryRoot, files[fileIndex]), err)
		}

		var importSpec *ast.ImportSpec
		for _, importSpec = range fileNode.Imports {
			var importPath string
			importPath, err = strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				t.Fatalf("unquote import path %s: %v", empiricalRelativePath(t, repositoryRoot, files[fileIndex]), err)
			}

			var ruleIndex int
			for ruleIndex = range rules {
				if !matchesForbiddenImport(importPath, rules[ruleIndex].ImportPath) {
					continue
				}

				violations = append(violations, empiricalRelativePath(t, repositoryRoot, files[fileIndex])+": import "+importPath+" ("+rules[ruleIndex].Reason+")")
			}
		}
	}

	sort.Strings(violations)
	if len(violations) != 0 {
		t.Fatalf("empirical import isolation violations:\n- %s", strings.Join(violations, "\n- "))
	}
}

// assertEmpiricalIdentifierIsolation scans empirical Go files for forbidden
// report-output identifiers and path literals.
// Authored by: OpenCode
func assertEmpiricalIdentifierIsolation(t *testing.T, rules []empiricalForbiddenIdentifierRule) {
	t.Helper()

	var repositoryRoot = empiricalRepositoryRoot(t)
	var files = empiricalSourceFiles(t, repositoryRoot)
	var violations = make([]string, 0)
	var fileIndex int

	for fileIndex = range files {
		var fileSet = token.NewFileSet()
		var fileNode, err = parser.ParseFile(fileSet, files[fileIndex], nil, 0)
		if err != nil {
			t.Fatalf("parse empirical source file %s: %v", empiricalRelativePath(t, repositoryRoot, files[fileIndex]), err)
		}

		ast.Inspect(fileNode, func(node ast.Node) bool {
			switch typedNode := node.(type) {
			case *ast.Ident:
				var ruleIndex int
				for ruleIndex = range rules {
					if typedNode.Name != rules[ruleIndex].Match {
						continue
					}

					violations = append(violations, empiricalRelativePath(t, repositoryRoot, files[fileIndex])+": identifier "+typedNode.Name+" ("+rules[ruleIndex].Reason+")")
				}
			case *ast.BasicLit:
				if typedNode.Kind != token.STRING {
					return true
				}

				var literalValue string
				literalValue, err = strconv.Unquote(typedNode.Value)
				if err != nil {
					t.Fatalf("unquote string literal %s: %v", empiricalRelativePath(t, repositoryRoot, files[fileIndex]), err)
				}

				var ruleIndex int
				for ruleIndex = range rules {
					if !strings.Contains(literalValue, rules[ruleIndex].Match) {
						continue
					}

					violations = append(violations, empiricalRelativePath(t, repositoryRoot, files[fileIndex])+": string literal "+strconv.Quote(literalValue)+" ("+rules[ruleIndex].Reason+")")
				}
			}

			return true
		})
	}

	sort.Strings(violations)
	if len(violations) != 0 {
		t.Fatalf("empirical output-boundary violations:\n- %s", strings.Join(violations, "\n- "))
	}
}

// empiricalSourceFiles returns every empirical Go source file except this
// isolation-assertion file itself.
// Authored by: OpenCode
func empiricalSourceFiles(t *testing.T, repositoryRoot string) []string {
	t.Helper()

	var sourceRoot = filepath.Join(repositoryRoot, "tests", "empirical")
	var currentFile = filepath.Join(sourceRoot, "isolation_test.go")
	var files = make([]string, 0)

	var walkErr = filepath.WalkDir(sourceRoot, func(currentPath string, directoryEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if directoryEntry.IsDir() {
			return nil
		}
		if filepath.Ext(currentPath) != ".go" {
			return nil
		}
		if filepath.Clean(currentPath) == filepath.Clean(currentFile) {
			return nil
		}

		files = append(files, currentPath)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk empirical source files: %v", walkErr)
	}

	sort.Strings(files)
	return files
}

// empiricalRelativePath converts one empirical source path into its repository-
// relative slash form for stable failure output.
// Authored by: OpenCode
func empiricalRelativePath(t *testing.T, repositoryRoot string, filesystemPath string) string {
	t.Helper()

	var relativePath, err = filepath.Rel(repositoryRoot, filesystemPath)
	if err != nil {
		t.Fatalf("resolve repository-relative path for %s: %v", filesystemPath, err)
	}

	return filepath.ToSlash(relativePath)
}

// matchesForbiddenImport reports whether one imported package is the forbidden
// package itself or one of its descendants.
// Authored by: OpenCode
func matchesForbiddenImport(importPath string, forbiddenPath string) bool {
	if importPath == forbiddenPath {
		return true
	}

	return strings.HasPrefix(importPath, forbiddenPath+"/")
}
