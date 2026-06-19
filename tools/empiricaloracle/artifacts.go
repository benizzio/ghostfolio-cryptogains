// Package main contains artifact persistence helpers for empirical oracle
// fixture regeneration.
//
// Authored by: OpenCode
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// artifactExists reports whether one repository-relative artifact file already
// exists.
// Authored by: OpenCode
func artifactExists(repositoryRoot string, relativePath string) (bool, error) {
	var absolutePath = filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	var info, err = os.Stat(absolutePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("artifact path %s points to a directory", relativePath)
	}

	return true, nil
}

// ensureArtifactContentMatches verifies that one existing repository artifact
// already contains the expected deterministic content.
// Authored by: OpenCode
func ensureArtifactContentMatches(repositoryRoot string, relativePath string, expectedContent string) error {
	var absolutePath = filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	var actualContent, err = os.ReadFile(absolutePath)
	if err != nil {
		return fmt.Errorf("read existing artifact %s: %w", relativePath, err)
	}
	if string(actualContent) == expectedContent {
		return nil
	}

	return fmt.Errorf("existing artifact %s differs from the current deterministic render; rerun with --regenerate to refresh it", relativePath)
}

// writeArtifact persists one repository-relative artifact unless reuse without
// regeneration was requested and the file already exists.
// Authored by: OpenCode
func writeArtifact(repositoryRoot string, relativePath string, content []byte, regenerate bool) (bool, error) {
	var absolutePath = filepath.Join(repositoryRoot, filepath.FromSlash(relativePath))
	var exists, err = artifactExists(repositoryRoot, relativePath)
	if err != nil {
		return false, err
	}
	if exists && !regenerate {
		return false, nil
	}

	var parentDirectory = filepath.Dir(absolutePath)
	if err = os.MkdirAll(parentDirectory, 0o755); err != nil {
		return false, fmt.Errorf("create parent directory %s: %w", filepath.ToSlash(parentDirectory), err)
	}
	if err = os.WriteFile(absolutePath, content, 0o644); err != nil {
		return false, fmt.Errorf("write artifact %s: %w", relativePath, err)
	}

	return true, nil
}

// marshalValidatedOracleOutput indents one normalized oracle fixture and
// validates the persisted JSON payload before it is written.
// Authored by: OpenCode
func marshalValidatedOracleOutput(path string, output fixture.OracleOutput) ([]byte, error) {
	var rawContent, err = json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal oracle output JSON: %w", err)
	}
	if err = fixture.ValidateOracleOutput(path, string(rawContent), output); err != nil {
		return nil, fmt.Errorf("validate oracle output JSON: %w", err)
	}

	return append(rawContent, '\n'), nil
}
