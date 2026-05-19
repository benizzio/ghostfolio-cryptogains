package store

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
)

// stubCodec is a test-only codec implementation that injects encode and decode
// behavior for filesystem-store coverage.
// Authored by: OpenCode
type stubCodec struct {
	encode func(snapshotmodel.Envelope) ([]byte, error)
	decode func([]byte) (snapshotmodel.Envelope, error)
}

// Encode implements snapshot envelope encoding for filesystem-store tests.
// Authored by: OpenCode
func (s stubCodec) Encode(envelope snapshotmodel.Envelope) ([]byte, error) {
	if s.encode != nil {
		return s.encode(envelope)
	}
	return snapshotenvelope.NewJSONCodec().Encode(envelope)
}

// Decode implements snapshot envelope decoding for filesystem-store tests.
// Authored by: OpenCode
func (s stubCodec) Decode(raw []byte) (snapshotmodel.Envelope, error) {
	if s.decode != nil {
		return s.decode(raw)
	}
	return snapshotenvelope.NewJSONCodec().Decode(raw)
}

// stubDirEntry is a test-only os.DirEntry implementation for candidate
// enumeration tests.
// Authored by: OpenCode
type stubDirEntry struct {
	name string
	dir  bool
}

// Name returns the directory entry name.
// Authored by: OpenCode
func (s stubDirEntry) Name() string { return s.name }

// IsDir reports whether the entry is a directory.
// Authored by: OpenCode
func (s stubDirEntry) IsDir() bool { return s.dir }

// Type returns a coarse file mode for the stub entry.
// Authored by: OpenCode
func (s stubDirEntry) Type() fs.FileMode {
	if s.dir {
		return fs.ModeDir
	}
	return 0
}

// Info preserves the os.DirEntry contract for store tests.
// Authored by: OpenCode
func (s stubDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// stubTemporaryFile is a test-only temporary file implementation for atomic
// replacement coverage.
// Authored by: OpenCode
type stubTemporaryFile struct {
	path     string
	chmodErr error
	writeErr error
	syncErr  error
	closeErr error
	written  []byte
}

// stubSnapshotStore is a test-only Store implementation for discovery helper
// coverage.
// Authored by: OpenCode
type stubSnapshotStore struct {
	candidates    []Candidate
	candidatesErr error
}

// Candidates returns injected discovery results for store-package tests.
// Authored by: OpenCode
func (s stubSnapshotStore) Candidates(context.Context) ([]Candidate, error) {
	if s.candidatesErr != nil {
		return nil, s.candidatesErr
	}
	return s.candidates, nil
}

// Read preserves the Store contract for discovery helper tests.
// Authored by: OpenCode
func (stubSnapshotStore) Read(context.Context, ReadRequest) (snapshotmodel.Payload, error) {
	return snapshotmodel.Payload{}, nil
}

// Write preserves the Store contract for discovery helper tests.
// Authored by: OpenCode
func (stubSnapshotStore) Write(context.Context, WriteRequest) (Candidate, error) {
	return Candidate{}, nil
}

// Name returns the stub temporary-file path.
// Authored by: OpenCode
func (s *stubTemporaryFile) Name() string { return s.path }

// Chmod records the requested mode change or returns the injected error.
// Authored by: OpenCode
func (s *stubTemporaryFile) Chmod(os.FileMode) error { return s.chmodErr }

// Write appends payload bytes or returns the injected write error.
// Authored by: OpenCode
func (s *stubTemporaryFile) Write(raw []byte) (int, error) {
	if s.writeErr != nil {
		return 0, s.writeErr
	}
	s.written = append(s.written, raw...)
	return len(raw), nil
}

// Sync flushes the stub file or returns the injected sync error.
// Authored by: OpenCode
func (s *stubTemporaryFile) Sync() error { return s.syncErr }

// Close closes the stub file or returns the injected close error.
// Authored by: OpenCode
func (s *stubTemporaryFile) Close() error { return s.closeErr }

// TestFilesystemStoreHelpersCoverBasicBranches verifies the exported helper
// methods and not-implemented boundaries on the foundational store.
// Authored by: OpenCode
func TestFilesystemStoreHelpersCoverBasicBranches(t *testing.T) {
	var baseDir = t.TempDir()
	var store = NewFilesystemStore(baseDir, nil)

	if store.Directory() == "" {
		t.Fatalf("expected snapshot directory")
	}
	if got := store.SnapshotPath("snapshot-1"); filepath.Ext(got) != SnapshotFileExtension {
		t.Fatalf("unexpected snapshot path: %q", got)
	}
	if _, err := store.Read(context.Background(), ReadRequest{}); !errors.Is(err, ErrReadNotImplemented) {
		t.Fatalf("expected not-implemented read, got %v", err)
	}
	if _, err := store.Write(context.Background(), WriteRequest{}); !errors.Is(err, ErrWriteNotImplemented) {
		t.Fatalf("expected not-implemented write, got %v", err)
	}
	if ignoresPermissionBits() {
		t.Fatalf("expected non-Windows default permission behavior")
	}

	originalWindows := isWindowsPlatform
	isWindowsPlatform = func() bool { return true }
	t.Cleanup(func() {
		isWindowsPlatform = originalWindows
	})
	if !ignoresPermissionBits() {
		t.Fatalf("expected Windows permission behavior when injected")
	}
}

// TestFilesystemStoreCandidatesCoverBranches verifies candidate enumeration
// across missing-directory, error, cancellation, and success paths.
// Authored by: OpenCode
func TestFilesystemStoreCandidatesCoverBranches(t *testing.T) {
	t.Run("context cancelled before read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := NewFilesystemStore(t.TempDir(), nil).Candidates(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled context, got %v", err)
		}
	})

	t.Run("missing directory returns empty list", func(t *testing.T) {
		candidates, err := NewFilesystemStore(filepath.Join(t.TempDir(), "missing"), nil).Candidates(context.Background())
		if err != nil {
			t.Fatalf("expected missing directory to succeed, got %v", err)
		}
		if len(candidates) != 0 {
			t.Fatalf("expected no candidates, got %#v", candidates)
		}
	})

	t.Run("read dir error", func(t *testing.T) {
		originalReadDir := readDir
		readDir = func(string) ([]os.DirEntry, error) {
			return nil, errors.New("read dir boom")
		}
		defer func() {
			readDir = originalReadDir
		}()

		_, err := NewFilesystemStore(t.TempDir(), nil).Candidates(context.Background())
		if err == nil {
			t.Fatalf("expected read-directory error")
		}
	})

	t.Run("read file error", func(t *testing.T) {
		var baseDir = t.TempDir()
		var store = NewFilesystemStore(baseDir, stubCodec{decode: func([]byte) (snapshotmodel.Envelope, error) {
			return snapshotmodel.Envelope{Header: storeHeaderFixture("https://ghostfol.io"), Ciphertext: []byte("ciphertext")}, nil
		}})
		if err := os.MkdirAll(store.Directory(), directoryFileMode); err != nil {
			t.Fatalf("mkdir snapshots: %v", err)
		}
		if err := os.WriteFile(store.SnapshotPath("snapshot-1"), []byte("raw"), snapshotFileMode); err != nil {
			t.Fatalf("write snapshot file: %v", err)
		}

		originalReadFile := readFile
		readFile = func(string) ([]byte, error) {
			return nil, errors.New("read file boom")
		}
		defer func() {
			readFile = originalReadFile
		}()

		_, err := store.Candidates(context.Background())
		if err == nil {
			t.Fatalf("expected read-file failure")
		}
	})

	t.Run("decode error", func(t *testing.T) {
		var baseDir = t.TempDir()
		var store = NewFilesystemStore(baseDir, stubCodec{decode: func([]byte) (snapshotmodel.Envelope, error) {
			return snapshotmodel.Envelope{}, errors.New("decode boom")
		}})
		if err := os.MkdirAll(store.Directory(), directoryFileMode); err != nil {
			t.Fatalf("mkdir snapshots: %v", err)
		}
		if err := os.WriteFile(store.SnapshotPath("snapshot-1"), []byte("raw"), snapshotFileMode); err != nil {
			t.Fatalf("write snapshot file: %v", err)
		}

		_, err := store.Candidates(context.Background())
		if err == nil {
			t.Fatalf("expected decode failure")
		}
	})

	t.Run("context cancelled during iteration", func(t *testing.T) {
		var baseDir = t.TempDir()
		var store = NewFilesystemStore(baseDir, stubCodec{decode: func([]byte) (snapshotmodel.Envelope, error) {
			return snapshotmodel.Envelope{Header: storeHeaderFixture("https://ghostfol.io"), Ciphertext: []byte("ciphertext")}, nil
		}})

		originalReadDir := readDir
		originalReadFile := readFile
		defer func() {
			readDir = originalReadDir
			readFile = originalReadFile
		}()

		var calls int
		ctx, cancel := context.WithCancel(context.Background())
		readDir = func(string) ([]os.DirEntry, error) {
			return []os.DirEntry{stubDirEntry{name: "a.snapshot"}, stubDirEntry{name: "b.snapshot"}}, nil
		}
		readFile = func(string) ([]byte, error) {
			calls++
			cancel()
			return []byte("raw"), nil
		}

		_, err := store.Candidates(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled context during iteration, got %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected first file read before cancellation, got %d", calls)
		}
	})

	t.Run("success sorts and filters entries", func(t *testing.T) {
		var baseDir = t.TempDir()
		var codec = snapshotenvelope.NewJSONCodec()
		var store = NewFilesystemStore(baseDir, codec)
		if err := os.MkdirAll(store.Directory(), directoryFileMode); err != nil {
			t.Fatalf("mkdir snapshots: %v", err)
		}
		if err := os.Mkdir(filepath.Join(store.Directory(), "subdir"), directoryFileMode); err != nil {
			t.Fatalf("mkdir subdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(store.Directory(), "note.txt"), []byte("ignore"), snapshotFileMode); err != nil {
			t.Fatalf("write note: %v", err)
		}
		writeRawStoreEnvelope(t, store, "b", storeHeaderFixture("https://server-b.example"))
		writeRawStoreEnvelope(t, store, "a", storeHeaderFixture("https://server-a.example"))

		candidates, err := store.Candidates(context.Background())
		if err != nil {
			t.Fatalf("enumerate candidates: %v", err)
		}
		if len(candidates) != 2 {
			t.Fatalf("expected two candidates, got %#v", candidates)
		}
		if candidates[0].SnapshotID != "a" || candidates[1].SnapshotID != "b" {
			t.Fatalf("expected sorted candidates, got %#v", candidates)
		}
	})
}

// TestDiscoverServerCandidatesCoversBranches verifies server-scoped discovery
// filtering across nil, error, and matching-candidate paths.
// Authored by: OpenCode
func TestDiscoverServerCandidatesCoversBranches(t *testing.T) {
	t.Run("nil store returns empty result", func(t *testing.T) {
		candidates, err := DiscoverServerCandidates(context.Background(), nil, "https://server-a.example")
		if err != nil {
			t.Fatalf("discover nil store candidates: %v", err)
		}
		if len(candidates) != 0 {
			t.Fatalf("expected no candidates, got %#v", candidates)
		}
	})

	t.Run("store error", func(t *testing.T) {
		_, err := DiscoverServerCandidates(context.Background(), stubSnapshotStore{candidatesErr: errors.New("discover boom")}, "https://server-a.example")
		if err == nil {
			t.Fatalf("expected discovery error")
		}
	})

	t.Run("filters by selected server", func(t *testing.T) {
		candidates, err := DiscoverServerCandidates(context.Background(), stubSnapshotStore{candidates: []Candidate{
			{SnapshotID: "match", Header: storeHeaderFixture("https://server-a.example")},
			{SnapshotID: "other", Header: storeHeaderFixture("https://server-b.example")},
		}}, "https://server-a.example")
		if err != nil {
			t.Fatalf("discover filtered candidates: %v", err)
		}
		if len(candidates) != 1 || candidates[0].SnapshotID != "match" {
			t.Fatalf("unexpected filtered candidates: %#v", candidates)
		}
	})
}

// TestReplaceFileAtomicallyCoversBranches verifies atomic replacement error and
// success paths, including Windows-style replacement.
// Authored by: OpenCode
func TestReplaceFileAtomicallyCoversBranches(t *testing.T) {
	t.Run("mkdir all error", func(t *testing.T) {
		originalMkdirAll := mkdirAll
		mkdirAll = func(string, os.FileMode) error {
			return errors.New("mkdir boom")
		}
		defer func() {
			mkdirAll = originalMkdirAll
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected mkdir error")
		}
	})

	t.Run("apply directory mode error", func(t *testing.T) {
		originalChmod := chmodPath
		chmodPath = func(string, os.FileMode) error {
			return errors.New("chmod boom")
		}
		defer func() {
			chmodPath = originalChmod
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected directory chmod error")
		}
	})

	t.Run("create temp file error", func(t *testing.T) {
		originalCreateTempFile := createTempFile
		createTempFile = func(string, string) (temporaryFile, error) {
			return nil, errors.New("temp boom")
		}
		defer func() {
			createTempFile = originalCreateTempFile
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected temporary-file creation error")
		}
	})

	t.Run("temp chmod error", func(t *testing.T) {
		var tempFile = &stubTemporaryFile{path: filepath.Join(t.TempDir(), "temp.snapshot")}
		originalCreateTempFile := createTempFile
		createTempFile = func(string, string) (temporaryFile, error) {
			tempFile.chmodErr = errors.New("chmod boom")
			return tempFile, nil
		}
		defer func() {
			createTempFile = originalCreateTempFile
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected temporary-file chmod error")
		}
	})

	t.Run("temp write error", func(t *testing.T) {
		var tempFile = &stubTemporaryFile{path: filepath.Join(t.TempDir(), "temp.snapshot"), writeErr: errors.New("write boom")}
		originalCreateTempFile := createTempFile
		createTempFile = func(string, string) (temporaryFile, error) {
			return tempFile, nil
		}
		defer func() {
			createTempFile = originalCreateTempFile
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected temporary-file write error")
		}
	})

	t.Run("temp sync error", func(t *testing.T) {
		var tempFile = &stubTemporaryFile{path: filepath.Join(t.TempDir(), "temp.snapshot"), syncErr: errors.New("sync boom")}
		originalCreateTempFile := createTempFile
		createTempFile = func(string, string) (temporaryFile, error) {
			return tempFile, nil
		}
		defer func() {
			createTempFile = originalCreateTempFile
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected temporary-file sync error")
		}
	})

	t.Run("temp close error", func(t *testing.T) {
		var tempFile = &stubTemporaryFile{path: filepath.Join(t.TempDir(), "temp.snapshot"), closeErr: errors.New("close boom")}
		originalCreateTempFile := createTempFile
		createTempFile = func(string, string) (temporaryFile, error) {
			return tempFile, nil
		}
		defer func() {
			createTempFile = originalCreateTempFile
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected temporary-file close error")
		}
	})

	t.Run("rename error", func(t *testing.T) {
		originalRename := renamePath
		renamePath = func(string, string) error {
			return errors.New("rename boom")
		}
		defer func() {
			renamePath = originalRename
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected rename error")
		}
	})

	t.Run("final path mode error", func(t *testing.T) {
		var originalChmod = chmodPath
		var calls int
		chmodPath = func(path string, mode os.FileMode) error {
			calls++
			if calls == 2 {
				return errors.New("final chmod boom")
			}
			return originalChmod(path, mode)
		}
		defer func() {
			chmodPath = originalChmod
		}()
		if err := ReplaceFileAtomically(filepath.Join(t.TempDir(), "a.snapshot"), []byte("x")); err == nil {
			t.Fatalf("expected final chmod error")
		}
	})

	t.Run("success on default platform", func(t *testing.T) {
		var path = filepath.Join(t.TempDir(), "success.snapshot")
		if err := ReplaceFileAtomically(path, []byte("payload")); err != nil {
			t.Fatalf("replace file atomically: %v", err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read written file: %v", err)
		}
		if string(raw) != "payload" {
			t.Fatalf("unexpected payload: %q", raw)
		}
	})

	t.Run("success on injected windows platform", func(t *testing.T) {
		originalWindows := isWindowsPlatform
		isWindowsPlatform = func() bool { return true }
		defer func() {
			isWindowsPlatform = originalWindows
		}()

		var path = filepath.Join(t.TempDir(), "success.snapshot")
		if err := ReplaceFileAtomically(path, []byte("payload")); err != nil {
			t.Fatalf("replace file atomically on injected windows platform: %v", err)
		}
	})

	t.Run("windows replace error", func(t *testing.T) {
		originalWindows := isWindowsPlatform
		originalRename := renamePath
		isWindowsPlatform = func() bool { return true }
		defer func() {
			isWindowsPlatform = originalWindows
			renamePath = originalRename
		}()

		var calls int
		renamePath = func(oldPath string, newPath string) error {
			calls++
			if calls == 2 {
				return errors.New("replace boom")
			}
			return originalRename(oldPath, newPath)
		}

		var dir = t.TempDir()
		var path = filepath.Join(dir, "existing.snapshot")
		if err := os.WriteFile(path, []byte("old"), snapshotFileMode); err != nil {
			t.Fatalf("write original file: %v", err)
		}
		if err := ReplaceFileAtomically(path, []byte("payload")); err == nil {
			t.Fatalf("expected windows replace-file error")
		}
	})
}

// TestReplaceFileWindowsCoversBranches verifies Windows-style replacement
// behavior across error and success paths.
// Authored by: OpenCode
func TestReplaceFileWindowsCoversBranches(t *testing.T) {
	t.Run("stat error", func(t *testing.T) {
		originalStat := statPath
		statPath = func(string) (os.FileInfo, error) {
			return nil, errors.New("stat boom")
		}
		defer func() {
			statPath = originalStat
		}()
		if err := replaceFileWindows(filepath.Join(t.TempDir(), "a.snapshot"), filepath.Join(t.TempDir(), "temp.snapshot")); err == nil {
			t.Fatalf("expected stat error")
		}
	})

	t.Run("backup rename error", func(t *testing.T) {
		var dir = t.TempDir()
		var path = filepath.Join(dir, "a.snapshot")
		var tempPath = filepath.Join(dir, "temp.snapshot")
		if err := os.WriteFile(path, []byte("old"), snapshotFileMode); err != nil {
			t.Fatalf("write original file: %v", err)
		}
		if err := os.WriteFile(tempPath, []byte("new"), snapshotFileMode); err != nil {
			t.Fatalf("write temp file: %v", err)
		}

		originalRename := renamePath
		var calls int
		renamePath = func(oldPath string, newPath string) error {
			calls++
			if calls == 1 {
				return errors.New("rename boom")
			}
			return originalRename(oldPath, newPath)
		}
		defer func() {
			renamePath = originalRename
		}()

		if err := replaceFileWindows(path, tempPath); err == nil {
			t.Fatalf("expected backup rename error")
		}
	})

	t.Run("replace rename error restores backup", func(t *testing.T) {
		var dir = t.TempDir()
		var path = filepath.Join(dir, "a.snapshot")
		var tempPath = filepath.Join(dir, "temp.snapshot")
		if err := os.WriteFile(path, []byte("old"), snapshotFileMode); err != nil {
			t.Fatalf("write original file: %v", err)
		}
		if err := os.WriteFile(tempPath, []byte("new"), snapshotFileMode); err != nil {
			t.Fatalf("write temp file: %v", err)
		}

		originalRename := renamePath
		var calls int
		renamePath = func(oldPath string, newPath string) error {
			calls++
			if calls == 2 {
				return errors.New("replace boom")
			}
			return originalRename(oldPath, newPath)
		}
		defer func() {
			renamePath = originalRename
		}()

		if err := replaceFileWindows(path, tempPath); err == nil {
			t.Fatalf("expected replace error")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read restored file: %v", err)
		}
		if string(raw) != "old" {
			t.Fatalf("expected original file to be restored, got %q", raw)
		}
	})

	t.Run("success with existing file removes backup", func(t *testing.T) {
		var dir = t.TempDir()
		var path = filepath.Join(dir, "a.snapshot")
		var tempPath = filepath.Join(dir, "temp.snapshot")
		if err := os.WriteFile(path, []byte("old"), snapshotFileMode); err != nil {
			t.Fatalf("write original file: %v", err)
		}
		if err := os.WriteFile(tempPath, []byte("new"), snapshotFileMode); err != nil {
			t.Fatalf("write temp file: %v", err)
		}

		if err := replaceFileWindows(path, tempPath); err != nil {
			t.Fatalf("replace file windows: %v", err)
		}
		if _, err := os.Stat(path + backupFileSuffix); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected backup to be removed, got %v", err)
		}
	})

	t.Run("success without existing file", func(t *testing.T) {
		var dir = t.TempDir()
		var path = filepath.Join(dir, "a.snapshot")
		var tempPath = filepath.Join(dir, "temp.snapshot")
		if err := os.WriteFile(tempPath, []byte("new"), snapshotFileMode); err != nil {
			t.Fatalf("write temp file: %v", err)
		}

		if err := replaceFileWindows(path, tempPath); err != nil {
			t.Fatalf("replace file windows without existing file: %v", err)
		}
	})
}

// TestApplyPathModeCoversBranches verifies restrictive mode application on
// Windows-like and Unix-like permission behavior.
// Authored by: OpenCode
func TestApplyPathModeCoversBranches(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "path")
	if err := os.WriteFile(path, []byte("x"), snapshotFileMode); err != nil {
		t.Fatalf("write path fixture: %v", err)
	}

	originalChmod := chmodPath
	originalWindows := isWindowsPlatform
	defer func() {
		chmodPath = originalChmod
		isWindowsPlatform = originalWindows
	}()

	chmodPath = func(string, os.FileMode) error {
		return errors.New("chmod boom")
	}
	if err := applyPathMode(path, snapshotFileMode); err == nil {
		t.Fatalf("expected chmod error on non-Windows platform")
	}

	isWindowsPlatform = func() bool { return true }
	if err := applyPathMode(path, snapshotFileMode); err != nil {
		t.Fatalf("expected chmod error to be ignored on injected Windows platform, got %v", err)
	}
}

// writeRawStoreEnvelope persists one cleartext-header fixture for candidate
// enumeration tests.
// Authored by: OpenCode
func writeRawStoreEnvelope(t *testing.T, store *FilesystemStore, snapshotID string, header snapshotmodel.EnvelopeHeader) {
	t.Helper()

	raw, err := store.codec.Encode(snapshotmodel.Envelope{Header: header, Ciphertext: []byte("ciphertext")})
	if err != nil {
		t.Fatalf("encode store envelope: %v", err)
	}
	if err := os.WriteFile(store.SnapshotPath(snapshotID), raw, snapshotFileMode); err != nil {
		t.Fatalf("write store envelope: %v", err)
	}
}

// storeHeaderFixture returns one valid snapshot header fixture for store tests.
// Authored by: OpenCode
func storeHeaderFixture(serverOrigin string) snapshotmodel.EnvelopeHeader {
	return snapshotmodel.EnvelopeHeader{
		Magic:              snapshotmodel.EnvelopeMagic,
		FormatVersion:      snapshotmodel.EnvelopeFormatVersion,
		ServerDiscoveryKey: snapshotenvelope.DeriveServerDiscoveryKey(serverOrigin),
		KDFParameters:      snapshotmodel.DefaultKDFParameters(),
		Salt:               make([]byte, snapshotmodel.DefaultSaltLength),
		Nonce:              make([]byte, snapshotmodel.DefaultNonceLength),
	}
}
