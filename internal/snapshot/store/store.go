// Package store defines the protected snapshot persistence boundary.
// Authored by: OpenCode
package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
)

const (
	applicationDirectoryName = "ghostfolio-cryptogains"
	directoryFileMode        = 0o700
	snapshotFileMode         = 0o600
	backupFileSuffix         = ".bak"

	// SnapshotDirectoryName is the protected snapshot directory created beside
	// the bootstrap setup file.
	SnapshotDirectoryName = "snapshots"

	// SnapshotFileExtension is the opaque protected snapshot file suffix.
	SnapshotFileExtension = ".snapshot"
)

var (
	// ErrReadNotImplemented indicates that protected payload decrypt and load are not part of the foundational store helpers.
	ErrReadNotImplemented = errors.New("protected snapshot payload read is not implemented in the foundational store")

	// ErrWriteNotImplemented indicates that protected payload encryption and write are not part of the foundational store helpers.
	ErrWriteNotImplemented = errors.New("protected snapshot payload write is not implemented in the foundational store")
)

// temporaryFile defines the transient file contract used during atomic snapshot
// replacement.
// Authored by: OpenCode
type temporaryFile interface {
	Name() string
	Chmod(os.FileMode) error
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

// Test seams wrap filesystem reads so store tests can inject failure paths.
// Authored by: OpenCode
var readDir = os.ReadDir

// Test seams wrap snapshot file reads so store tests can inject failure paths.
// Authored by: OpenCode
var readFile = os.ReadFile

// Test seams wrap directory creation so store tests can inject failure paths.
// Authored by: OpenCode
var mkdirAll = os.MkdirAll

// Test seams wrap temporary-file creation so store tests can inject transient
// file failures.
// Authored by: OpenCode
var createTempFile = func(dir string, pattern string) (temporaryFile, error) {
	return os.CreateTemp(dir, pattern)
}

// Test seams wrap path renames so store tests can exercise atomic replacement
// failures safely.
// Authored by: OpenCode
var renamePath = os.Rename

// Test seams wrap chmod so store tests can exercise restrictive-permission
// failures safely.
// Authored by: OpenCode
var chmodPath = os.Chmod

// Test seams wrap file removal so store tests can verify cleanup behavior.
// Authored by: OpenCode
var removePath = os.Remove

// Test seams wrap stat calls so Windows replacement tests can inject inspect
// failures safely.
// Authored by: OpenCode
var statPath = os.Stat

// Test seams wrap platform checks so Linux tests can exercise Windows-specific
// replacement branches safely.
// Authored by: OpenCode
var isWindowsPlatform = func() bool {
	return runtime.GOOS == "windows"
}

// Candidate identifies one protected snapshot file discovered before decrypt.
// Authored by: OpenCode
type Candidate struct {
	SnapshotID string
	Path       string
	Header     snapshotmodel.EnvelopeHeader
}

// ReadRequest contains the token-aware inputs required to decrypt one
// protected snapshot payload.
// Authored by: OpenCode
type ReadRequest struct {
	Candidate     Candidate
	SecurityToken string
}

// WriteRequest contains the token-aware inputs required to encrypt and persist
// one protected snapshot payload.
// Authored by: OpenCode
type WriteRequest struct {
	SnapshotID    string
	SecurityToken string
	ServerOrigin  string
	Payload       snapshotmodel.Payload
}

// Store defines the protected snapshot discovery and persistence contract.
//
// Example:
//
//	var snapshots Store
//	_, _ = snapshots.Candidates(context.Background())
//
// Implementations are expected to discover cleartext headers, read decrypted
// payloads, and atomically replace protected snapshots.
// Authored by: OpenCode
type Store interface {
	Candidates(context.Context) ([]Candidate, error)
	Read(context.Context, ReadRequest) (snapshotmodel.Payload, error)
	Write(context.Context, WriteRequest) (Candidate, error)
}

// FilesystemStore resolves protected snapshot paths, enumerates snapshot
// headers, and provides atomic file-replacement helpers.
// Authored by: OpenCode
type FilesystemStore struct {
	directory string
	codec     snapshotenvelope.Codec
}

// NewFilesystemStore creates the protected snapshot filesystem helper rooted
// under the provided base config directory.
//
// Example:
//
//	codec := envelope.NewJSONCodec()
//	snapshots := store.NewFilesystemStore("/tmp/config", codec)
//	_ = snapshots.Directory()
//
// Authored by: OpenCode
func NewFilesystemStore(baseConfigDir string, codec snapshotenvelope.Codec) *FilesystemStore {
	if codec == nil {
		codec = snapshotenvelope.NewJSONCodec()
	}

	return &FilesystemStore{
		directory: filepath.Join(baseConfigDir, applicationDirectoryName, SnapshotDirectoryName),
		codec:     codec,
	}
}

// Directory returns the protected snapshot directory path.
//
// Authored by: OpenCode
func (s *FilesystemStore) Directory() string {
	return s.directory
}

// SnapshotPath resolves the full protected snapshot file path for one opaque snapshot identifier.
//
// Authored by: OpenCode
func (s *FilesystemStore) SnapshotPath(snapshotID string) string {
	return filepath.Join(s.directory, snapshotID+SnapshotFileExtension)
}

// Candidates enumerates protected snapshot files and decodes their cleartext headers.
//
// Authored by: OpenCode
func (s *FilesystemStore) Candidates(ctx context.Context) ([]Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := readDir(s.directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Candidate{}, nil
		}
		return nil, fmt.Errorf("read snapshot directory: %w", err)
	}

	var candidates []Candidate
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), SnapshotFileExtension) {
			continue
		}

		var snapshotID = strings.TrimSuffix(entry.Name(), SnapshotFileExtension)
		var path = s.SnapshotPath(snapshotID)
		var raw, readErr = readFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read snapshot file %q: %w", entry.Name(), readErr)
		}

		var envelope, decodeErr = s.codec.Decode(raw)
		if decodeErr != nil {
			return nil, fmt.Errorf("decode snapshot header %q: %w", entry.Name(), decodeErr)
		}

		candidates = append(candidates, Candidate{
			SnapshotID: snapshotID,
			Path:       path,
			Header:     envelope.Header,
		})
	}

	sort.Slice(candidates, func(left int, right int) bool {
		return candidates[left].SnapshotID < candidates[right].SnapshotID
	})

	return candidates, nil
}

// Read preserves the discovery/read/write boundary while payload decrypt and
// decode are deferred to the encrypted store implementation in the next phase.
//
// Authored by: OpenCode
func (s *FilesystemStore) Read(_ context.Context, _ ReadRequest) (snapshotmodel.Payload, error) {
	return snapshotmodel.Payload{}, ErrReadNotImplemented
}

// Write preserves the discovery/read/write boundary while payload encrypt and
// write are deferred to the encrypted store implementation in the next phase.
//
// Authored by: OpenCode
func (s *FilesystemStore) Write(_ context.Context, _ WriteRequest) (Candidate, error) {
	return Candidate{}, ErrWriteNotImplemented
}

// ReplaceFileAtomically writes one opaque protected snapshot file through a
// temp file, fsync, and atomic rename.
//
// Example:
//
//	err := store.ReplaceFileAtomically("/tmp/example.snapshot", []byte("data"))
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func ReplaceFileAtomically(path string, contents []byte) error {
	var parentDirectory = filepath.Dir(path)
	if err := mkdirAll(parentDirectory, directoryFileMode); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	if err := applyPathMode(parentDirectory, directoryFileMode); err != nil {
		return err
	}

	var tempFile, err = createTempFile(parentDirectory, ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary snapshot file: %w", err)
	}
	var tempPath = tempFile.Name()
	defer cleanupTempFile(tempPath)

	if err := tempFile.Chmod(snapshotFileMode); err != nil && !ignoresPermissionBits() {
		_ = tempFile.Close()
		return fmt.Errorf("chmod temporary snapshot file: %w", err)
	}
	if _, err := tempFile.Write(contents); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write temporary snapshot file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("sync temporary snapshot file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary snapshot file: %w", err)
	}

	if isWindowsPlatform() {
		if err := replaceFileWindows(path, tempPath); err != nil {
			return err
		}
	} else if err := renamePath(tempPath, path); err != nil {
		return fmt.Errorf("replace snapshot file atomically: %w", err)
	}

	if err := applyPathMode(path, snapshotFileMode); err != nil {
		return err
	}

	return nil
}

// replaceFileWindows swaps the snapshot file using a backup path so an existing
// file can be replaced atomically on Windows.
// Authored by: OpenCode
func replaceFileWindows(path string, tempPath string) error {
	var backupPath = path + backupFileSuffix
	cleanupTempFile(backupPath)

	var existingFilePresent = false
	if _, err := statPath(path); err == nil {
		existingFilePresent = true
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect existing snapshot file: %w", err)
	}

	if existingFilePresent {
		if err := renamePath(path, backupPath); err != nil {
			return fmt.Errorf("move existing snapshot file aside: %w", err)
		}
	}

	if err := renamePath(tempPath, path); err != nil {
		if existingFilePresent {
			_ = renamePath(backupPath, path)
		}
		return fmt.Errorf("replace snapshot file atomically: %w", err)
	}

	if existingFilePresent {
		cleanupTempFile(backupPath)
	}

	return nil
}

// applyPathMode reapplies a restrictive file mode when the platform honors permission bits.
// Authored by: OpenCode
func applyPathMode(path string, mode os.FileMode) error {
	if err := chmodPath(path, mode); err != nil && !ignoresPermissionBits() {
		return fmt.Errorf("chmod snapshot path: %w", err)
	}
	return nil
}

// cleanupTempFile removes a stale temporary or backup file after store operations.
// Authored by: OpenCode
func cleanupTempFile(path string) {
	_ = removePath(path)
}

// ignoresPermissionBits reports whether the current platform does not expose Unix-style permission bits meaningfully.
// Authored by: OpenCode
func ignoresPermissionBits() bool {
	return isWindowsPlatform()
}
