package empirical

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestHledgerCleanupRejectsActiveReferences verifies BUG-003 cleanup removed
// active hledger references from empirical source, fixtures, and active
// documentation paths.
// Authored by: OpenCode
func TestHledgerCleanupRejectsActiveReferences(t *testing.T) {
	t.Parallel()

	var repositoryRoot = empiricalRepositoryRoot(t)
	var scanRoots = []string{
		filepath.Join(repositoryRoot, "tools", "empiricaloracle"),
		filepath.Join(repositoryRoot, "tests", "empirical"),
		filepath.Join(repositoryRoot, "testdata", "empirical"),
		filepath.Join(repositoryRoot, "third_party", "rotki"),
		filepath.Join(repositoryRoot, "specs", "006-empirical-financial-tests", "contracts"),
		filepath.Join(repositoryRoot, "specs", "006-empirical-financial-tests", "quickstart.md"),
	}
	var violations = make([]string, 0)
	var rootIndex int

	for rootIndex = range scanRoots {
		violations = append(violations, collectHledgerCleanupViolations(t, repositoryRoot, scanRoots[rootIndex])...)
	}

	sort.Strings(violations)
	if len(violations) != 0 {
		t.Fatalf("active hledger cleanup violations:\n- %s", strings.Join(violations, "\n- "))
	}
}

// collectHledgerCleanupViolations scans one cleanup root for forbidden hledger
// or typo references.
// Authored by: OpenCode
func collectHledgerCleanupViolations(t *testing.T, repositoryRoot string, scanRoot string) []string {
	t.Helper()

	var info, err = os.Stat(scanRoot)
	if err != nil {
		t.Fatalf("stat cleanup root %s: %v", scanRoot, err)
	}

	if !info.IsDir() {
		return scanHledgerCleanupFile(t, repositoryRoot, scanRoot)
	}

	var violations = make([]string, 0)
	var walkErr = filepath.WalkDir(scanRoot, func(currentPath string, directoryEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if directoryEntry.IsDir() {
			return nil
		}
		if filepath.Base(currentPath) == "hledger_cleanup_test.go" {
			return nil
		}
		if !shouldScanHledgerCleanupPath(currentPath) {
			return nil
		}

		violations = append(violations, scanHledgerCleanupFile(t, repositoryRoot, currentPath)...)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk cleanup root %s: %v", scanRoot, walkErr)
	}

	return violations
}

// shouldScanHledgerCleanupPath reports whether one repository file is part of
// the active cleanup surface.
// Authored by: OpenCode
func shouldScanHledgerCleanupPath(filesystemPath string) bool {
	switch filepath.Ext(filesystemPath) {
	case ".go", ".md", ".json", ".yaml", ".yml", ".txt":
		return true
	default:
		return false
	}
}

// scanHledgerCleanupFile reports repository-relative violations for one active
// cleanup file.
// Authored by: OpenCode
func scanHledgerCleanupFile(t *testing.T, repositoryRoot string, filesystemPath string) []string {
	t.Helper()

	var rawContent, err = os.ReadFile(filesystemPath)
	if err != nil {
		t.Fatalf("read cleanup file %s: %v", filesystemPath, err)
	}

	var relativePath = empiricalRelativePath(t, repositoryRoot, filesystemPath)
	var violations = make([]string, 0)
	var lines = strings.Split(string(rawContent), "\n")
	var lineIndex int

	for lineIndex = range lines {
		var lowerLine = strings.ToLower(lines[lineIndex])
		if !strings.Contains(lowerLine, "hledger") && !strings.Contains(lowerLine, "hleger") {
			continue
		}

		violations = append(violations, relativePath+":"+strconv.Itoa(lineIndex+1)+": "+strings.TrimSpace(lines[lineIndex]))
	}

	return violations
}
