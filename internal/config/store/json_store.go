// Package store persists and reloads the bootstrap setup file.
// Authored by: OpenCode
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
)

const (
	appDirectoryName  = "ghostfolio-cryptogains"
	setupFileName     = "setup.json"
	directoryFileMode = 0o700
	setupFileMode     = 0o600
)

type temporaryFile interface {
	Name() string
	Chmod(os.FileMode) error
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

var readFile = os.ReadFile

var marshalIndent = json.MarshalIndent

var createTempFile = func(dir string, pattern string) (temporaryFile, error) {
	return os.CreateTemp(dir, pattern)
}

var mkdirAll = os.MkdirAll

var chmodPath = os.Chmod

var renamePath = os.Rename

var removePath = os.Remove

// JSONStore persists the setup file as a JSON document under the user config
// directory.
//
// Example:
//
//	bootstrapStore := store.NewJSONStore("/tmp/config")
//	_ = bootstrapStore.Path()
//
// Authored by: OpenCode
type JSONStore struct {
	path string
}

// NewJSONStore creates a JSON-backed setup store rooted under the provided
// base config directory.
//
// Example:
//
//	bootstrapStore := store.NewJSONStore("/tmp/config")
//	_ = bootstrapStore
//
// Authored by: OpenCode
func NewJSONStore(baseConfigDir string) *JSONStore {
	return &JSONStore{
		path: filepath.Join(baseConfigDir, appDirectoryName, setupFileName),
	}
}

// Path returns the setup file location used by the store.
//
// Example:
//
//	bootstrapStore := store.NewJSONStore("/tmp/config")
//	_ = bootstrapStore.Path()
//
// Authored by: OpenCode
func (s *JSONStore) Path() string {
	return s.path
}

// Load reads the bootstrap setup file from disk.
//
// Example:
//
//	config, err := bootstrapStore.Load(context.Background())
//	if err != nil {
//		panic(err)
//	}
//	_ = config.ServerOrigin
//
// Authored by: OpenCode
func (s *JSONStore) Load(_ context.Context) (configmodel.AppSetupConfig, error) {
	var raw, err = readFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return configmodel.AppSetupConfig{}, ErrNotFound
		}
		return configmodel.AppSetupConfig{}, fmt.Errorf("read setup file: %w", err)
	}

	var config configmodel.AppSetupConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return configmodel.AppSetupConfig{}, fmt.Errorf("decode setup file: %w", err)
	}

	return config, nil
}

// Save writes the bootstrap setup file atomically with restrictive permissions
// where the current platform supports them.
//
// Example:
//
//	err := bootstrapStore.Save(context.Background(), config)
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func (s *JSONStore) Save(_ context.Context, config configmodel.AppSetupConfig) error {
	if err := s.ensureParentDirectory(); err != nil {
		return err
	}

	var encoded, err = marshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode setup file: %w", err)
	}

	var parentDir = filepath.Dir(s.path)
	var tempFile temporaryFile
	tempFile, err = createTempFile(parentDir, ".setup-*.json")
	if err != nil {
		return fmt.Errorf("create temporary setup file: %w", err)
	}
	defer cleanupTempFile(tempFile.Name())

	if err := tempFile.Chmod(setupFileMode); err != nil && !ignoresPermissionBits() {
		_ = tempFile.Close()
		return fmt.Errorf("chmod temporary setup file: %w", err)
	}
	if _, err := tempFile.Write(encoded); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write temporary setup file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("sync temporary setup file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary setup file: %w", err)
	}

	if err := renamePath(tempFile.Name(), s.path); err != nil {
		return fmt.Errorf("replace setup file atomically: %w", err)
	}
	if err := chmodPath(s.path, setupFileMode); err != nil && !ignoresPermissionBits() {
		return fmt.Errorf("chmod setup file: %w", err)
	}

	return nil
}

// Delete removes the bootstrap setup file.
//
// Example:
//
//	err := bootstrapStore.Delete(context.Background())
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func (s *JSONStore) Delete(_ context.Context) error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete setup file: %w", err)
	}
	return nil
}

// ensureParentDirectory creates the application config directory when needed.
func (s *JSONStore) ensureParentDirectory() error {
	var parentDir = filepath.Dir(s.path)
	if err := mkdirAll(parentDir, directoryFileMode); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := chmodPath(parentDir, directoryFileMode); err != nil && !ignoresPermissionBits() {
		return fmt.Errorf("chmod config directory: %w", err)
	}
	return nil
}

// cleanupTempFile removes a stale temporary file after save attempts.
func cleanupTempFile(path string) {
	_ = removePath(path)
}

// ignoresPermissionBits reports whether the current platform does not expose
// Unix-style permission bits in a meaningful way for these checks.
func ignoresPermissionBits() bool {
	return runtime.GOOS == "windows"
}
