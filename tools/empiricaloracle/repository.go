// Package main contains repository-root discovery and command execution helpers
// for empirical oracle tooling.
//
// Authored by: OpenCode
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// runCombinedOutputFunc executes one command and returns its combined output.
// Authored by: OpenCode
type runCombinedOutputFunc func(ctx context.Context, env []string, name string, args ...string) ([]byte, error)

// resolveEmpiricalRepositoryRoot resolves the repository root from this source
// file's location so tests and tooling do not depend on the current working
// directory.
// Authored by: OpenCode
func resolveEmpiricalRepositoryRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve repository root: runtime caller lookup failed")
	}

	var repositoryRoot = filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	var goModPath = filepath.Join(repositoryRoot, "go.mod")
	var repositoryMarker, err = os.Stat(goModPath)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: expected repository marker %s: %w", goModPath, err)
	}
	if repositoryMarker.IsDir() {
		return "", fmt.Errorf("resolve repository root: repository marker %s is a directory", goModPath)
	}

	return repositoryRoot, nil
}

// runCombinedOutput executes one command with the supplied environment.
// Authored by: OpenCode
func runCombinedOutput(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	var cmd = exec.CommandContext(ctx, name, args...)
	cmd.Env = env

	return cmd.CombinedOutput()
}
