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

// Setup storage constants define the persisted file names and restrictive file
// modes used by the JSON store.
// Authored by: OpenCode
const (
	appDirectoryName  = "ghostfolio-cryptogains"
	setupFileName     = "setup.json"
	directoryFileMode = 0o700
	setupFileMode     = 0o600
)

// temporaryFile defines the transient file contract used during atomic setup
// saves.
// Authored by: OpenCode
type temporaryFile interface {
	Name() string
	Chmod(os.FileMode) error
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

// temporarySetupFile wraps the transient file path used during an atomic save.
// Authored by: OpenCode
type temporarySetupFile struct {
	file temporaryFile
	path string
}

// Test seams wrap filesystem reads so unit tests can replace them safely.
// Authored by: OpenCode
var readFile = os.ReadFile

// Test seams wrap JSON encoding so unit tests can replace it safely.
// Authored by: OpenCode
var marshalIndent = json.MarshalIndent

// Test seams wrap temporary-file creation so unit tests can replace it safely.
// Authored by: OpenCode
var createTempFile = func(dir string, pattern string) (temporaryFile, error) {
	return os.CreateTemp(dir, pattern)
}

// Test seams wrap directory creation so unit tests can replace it safely.
// Authored by: OpenCode
var mkdirAll = os.MkdirAll

// Test seams wrap path chmod calls so unit tests can replace them safely.
// Authored by: OpenCode
var chmodPath = os.Chmod

// Test seams wrap atomic rename calls so unit tests can replace them safely.
// Authored by: OpenCode
var renamePath = os.Rename

// Test seams wrap file removal so unit tests can replace it safely.
// Authored by: OpenCode
var removePath = os.Remove

// Test seams wrap platform checks so unit tests can exercise platform-specific
// save behavior safely.
// Authored by: OpenCode
var isWindowsPlatform = func() bool {
	return runtime.GOOS == "windows"
}

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
	var err = s.ensureParentDirectory()
	if err != nil {
		return err
	}

	var encoded []byte
	encoded, err = encodeSetupConfig(config)
	if err != nil {
		return err
	}

	var tempFile *temporarySetupFile
	tempFile, err = s.createTemporarySetupFile()
	if err != nil {
		return err
	}
	defer cleanupTempFile(tempFile.path)

	err = writeTemporarySetupFile(tempFile.file, encoded)
	if err != nil {
		return err
	}

	err = s.replaceSetupFile(tempFile.path)
	if err != nil {
		return err
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
// Authored by: OpenCode
func (s *JSONStore) ensureParentDirectory() error {
	var parentDir = filepath.Dir(s.path)
	var err = mkdirAll(parentDir, directoryFileMode)
	if err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	err = applyPathMode(parentDir, directoryFileMode, "chmod config directory")
	if err != nil {
		return fmt.Errorf("chmod config directory: %w", err)
	}
	return nil
}

// encodeSetupConfig serializes one bootstrap setup document for atomic persistence.
// Authored by: OpenCode
func encodeSetupConfig(config configmodel.AppSetupConfig) ([]byte, error) {
	var encoded, err = marshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode setup file: %w", err)
	}
	return encoded, nil
}

// createTemporarySetupFile allocates the transient file used by atomic save.
// Authored by: OpenCode
func (s *JSONStore) createTemporarySetupFile() (*temporarySetupFile, error) {
	var parentDir = filepath.Dir(s.path)
	var file, err = createTempFile(parentDir, ".setup-*.json")
	if err != nil {
		return nil, fmt.Errorf("create temporary setup file: %w", err)
	}
	return &temporarySetupFile{file: file, path: file.Name()}, nil
}

// writeTemporarySetupFile writes and closes the transient atomic-save file.
// Authored by: OpenCode
func writeTemporarySetupFile(file temporaryFile, encoded []byte) error {
	var err = applyTemporaryFileMode(file)
	if err != nil {
		_ = file.Close()
		return err
	}

	_, err = file.Write(encoded)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("write temporary setup file: %w", err)
	}

	err = file.Sync()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("sync temporary setup file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("close temporary setup file: %w", err)
	}

	return nil
}

// applyTemporaryFileMode sets restrictive permissions on the transient atomic-save file.
// Authored by: OpenCode
func applyTemporaryFileMode(file temporaryFile) error {
	var err = file.Chmod(setupFileMode)
	if err != nil && !ignoresPermissionBits() {
		return fmt.Errorf("chmod temporary setup file: %w", err)
	}
	return nil
}

// replaceSetupFile atomically swaps the saved setup document and reapplies restrictive permissions.
// Authored by: OpenCode
func (s *JSONStore) replaceSetupFile(tempPath string) error {
	if isWindowsPlatform() {
		return s.replaceSetupFileWindows(tempPath)
	}

	var err = renamePath(tempPath, s.path)
	if err != nil {
		return fmt.Errorf("replace setup file atomically: %w", err)
	}

	err = applyPathMode(s.path, setupFileMode, "chmod setup file")
	if err != nil {
		return err
	}

	return nil
}

// replaceSetupFileWindows swaps the saved setup document using a backup path so
// repeated saves can replace an existing file on Windows.
// Authored by: OpenCode
func (s *JSONStore) replaceSetupFileWindows(tempPath string) error {
	var backupPath = s.path + ".bak"

	cleanupTempFile(backupPath)

	var existingFilePresent = false
	if _, err := os.Stat(s.path); err == nil {
		existingFilePresent = true
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect existing setup file: %w", err)
	}

	if existingFilePresent {
		if err := renamePath(s.path, backupPath); err != nil {
			return fmt.Errorf("move existing setup file aside: %w", err)
		}
	}

	var err = renamePath(tempPath, s.path)
	if err != nil {
		if existingFilePresent {
			_ = renamePath(backupPath, s.path)
		}
		return fmt.Errorf("replace setup file atomically: %w", err)
	}

	if existingFilePresent {
		cleanupTempFile(backupPath)
	}

	err = applyPathMode(s.path, setupFileMode, "chmod setup file")
	if err != nil {
		return err
	}

	return nil
}

// applyPathMode reapplies a restrictive file mode when the platform honors permission bits.
// Authored by: OpenCode
func applyPathMode(path string, mode os.FileMode, operation string) error {
	var err = chmodPath(path, mode)
	if err != nil && !ignoresPermissionBits() {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}

// cleanupTempFile removes a stale temporary file after save attempts.
// Authored by: OpenCode
func cleanupTempFile(path string) {
	_ = removePath(path)
}

// ignoresPermissionBits reports whether the current platform does not expose
// Unix-style permission bits in a meaningful way for these checks.
// Authored by: OpenCode
func ignoresPermissionBits() bool {
	return isWindowsPlatform()
}
