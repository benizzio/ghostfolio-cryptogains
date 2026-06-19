package empirical

import (
	"path/filepath"
	"runtime"
	"testing"
)

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
