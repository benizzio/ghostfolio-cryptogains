package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
)

type failingTempFile struct {
	name       string
	chmodErr   error
	writeErr   error
	syncErr    error
	closeErr   error
	writeBytes int
}

func (f *failingTempFile) Name() string            { return f.name }
func (f *failingTempFile) Chmod(os.FileMode) error { return f.chmodErr }
func (f *failingTempFile) Write(content []byte) (int, error) {
	if f.writeErr != nil {
		return f.writeBytes, f.writeErr
	}
	return len(content), nil
}
func (f *failingTempFile) Sync() error  { return f.syncErr }
func (f *failingTempFile) Close() error { return f.closeErr }

func TestLoadRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var store = NewJSONStore(tempDir)
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(store.Path(), []byte("{"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := store.Load(context.Background()); err == nil {
		t.Fatalf("expected invalid json error")
	}
}

func TestLoadReturnsReadErrorForDirectoryPath(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var store = NewJSONStore(tempDir)
	if err := os.MkdirAll(store.Path(), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := store.Load(context.Background()); err == nil {
		t.Fatalf("expected read error")
	}
}

func TestDeleteReturnsUnderlyingError(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var store = NewJSONStore(tempDir)
	if err := os.MkdirAll(store.Path(), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.Path(), "child"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write child: %v", err)
	}

	if err := store.Delete(context.Background()); err == nil {
		t.Fatalf("expected delete error for non-empty directory")
	}
}

func TestEnsureParentDirectoryFailsWhenBasePathIsFile(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var baseFile = filepath.Join(tempDir, "file")
	if err := os.WriteFile(baseFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var store = NewJSONStore(baseFile)
	if err := store.ensureParentDirectory(); err == nil {
		t.Fatalf("expected ensureParentDirectory error")
	}
}

func TestCleanupTempFileRemovesFile(t *testing.T) {
	t.Parallel()

	var path = filepath.Join(t.TempDir(), "temp")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	cleanupTempFile(path)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, got %v", err)
	}
}

func TestIgnoresPermissionBitsMatchesPlatform(t *testing.T) {
	t.Parallel()

	var expected = runtime.GOOS == "windows"
	if ignoresPermissionBits() != expected {
		t.Fatalf("unexpected permission-bit behavior")
	}
}

func TestSaveHandlesEnsureParentDirectoryError(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var baseFile = filepath.Join(tempDir, "file")
	if err := os.WriteFile(baseFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	var store = NewJSONStore(baseFile)
	if err := store.Save(context.Background(), configmodel.AppSetupConfig{}); err == nil {
		t.Fatalf("expected save error")
	}
}

func TestSaveHandlesMarshalIndentError(t *testing.T) {
	var previous = marshalIndent
	defer func() { marshalIndent = previous }()
	marshalIndent = func(any, string, string) ([]byte, error) {
		return nil, errors.New("encode boom")
	}

	var store = NewJSONStore(t.TempDir())
	if err := store.Save(context.Background(), configmodel.AppSetupConfig{}); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestSaveHandlesCreateTempError(t *testing.T) {
	var previous = createTempFile
	defer func() { createTempFile = previous }()
	createTempFile = func(string, string) (temporaryFile, error) {
		return nil, errors.New("temp boom")
	}

	var store = NewJSONStore(t.TempDir())
	if err := store.Save(context.Background(), configmodel.AppSetupConfig{}); err == nil {
		t.Fatalf("expected create temp error")
	}
}

func TestSaveHandlesTempFileLifecycleErrors(t *testing.T) {
	var testCases = []struct {
		name     string
		tempFile temporaryFile
	}{
		{name: "chmod", tempFile: &failingTempFile{name: filepath.Join(t.TempDir(), "temp"), chmodErr: errors.New("chmod boom")}},
		{name: "write", tempFile: &failingTempFile{name: filepath.Join(t.TempDir(), "temp"), writeErr: errors.New("write boom")}},
		{name: "sync", tempFile: &failingTempFile{name: filepath.Join(t.TempDir(), "temp"), syncErr: errors.New("sync boom")}},
		{name: "close", tempFile: &failingTempFile{name: filepath.Join(t.TempDir(), "temp"), closeErr: errors.New("close boom")}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var previousTemp = createTempFile
			defer func() { createTempFile = previousTemp }()
			createTempFile = func(string, string) (temporaryFile, error) {
				return testCase.tempFile, nil
			}

			var store = NewJSONStore(t.TempDir())
			if err := store.Save(context.Background(), configmodel.AppSetupConfig{}); err == nil {
				t.Fatalf("expected save error")
			}
		})
	}
}

func TestSaveHandlesRenameAndChmodPathErrors(t *testing.T) {
	var store = NewJSONStore(t.TempDir())

	t.Run("rename", func(t *testing.T) {
		var previousRename = renamePath
		defer func() { renamePath = previousRename }()
		renamePath = func(string, string) error { return errors.New("rename boom") }

		if err := store.Save(context.Background(), configmodel.AppSetupConfig{}); err == nil {
			t.Fatalf("expected rename error")
		}
	})

	t.Run("chmod path", func(t *testing.T) {
		if ignoresPermissionBits() {
			t.Skip("permission-bit path error is ignored on this platform")
		}

		var previousChmod = chmodPath
		defer func() { chmodPath = previousChmod }()
		chmodPath = func(path string, mode os.FileMode) error {
			if filepath.Base(path) == "setup.json" {
				return errors.New("chmod boom")
			}
			return os.Chmod(path, mode)
		}

		if err := store.Save(context.Background(), configmodel.AppSetupConfig{}); err == nil {
			t.Fatalf("expected chmod path error")
		}
	})
}

func TestReplaceSetupFileWindowsReplacesExistingDestination(t *testing.T) {
	var previousRename = renamePath
	var previousPlatform = isWindowsPlatform
	defer func() {
		renamePath = previousRename
		isWindowsPlatform = previousPlatform
	}()

	isWindowsPlatform = func() bool { return true }
	renamePath = func(source string, destination string) error {
		if _, err := os.Stat(destination); err == nil {
			return errors.New("destination exists")
		}
		return os.Rename(source, destination)
	}

	var store = NewJSONStore(t.TempDir())
	if err := store.ensureParentDirectory(); err != nil {
		t.Fatalf("ensure parent directory: %v", err)
	}

	if err := os.WriteFile(store.Path(), []byte("old"), 0o600); err != nil {
		t.Fatalf("write existing setup file: %v", err)
	}

	var tempPath = filepath.Join(filepath.Dir(store.Path()), "temp-setup.json")
	if err := os.WriteFile(tempPath, []byte("new"), 0o600); err != nil {
		t.Fatalf("write temporary setup file: %v", err)
	}

	if err := store.replaceSetupFile(tempPath); err != nil {
		t.Fatalf("replace setup file: %v", err)
	}

	var content, err = os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read replaced setup file: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("expected replaced content, got %q", string(content))
	}

	if _, err := os.Stat(store.Path() + backupFileSuffix); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected backup file cleanup, got %v", err)
	}
}

func TestReplaceSetupFileNonWindowsRenamesDirectly(t *testing.T) {
	var previousRename = renamePath
	var previousPlatform = isWindowsPlatform
	defer func() {
		renamePath = previousRename
		isWindowsPlatform = previousPlatform
	}()

	var renameCalls = 0
	isWindowsPlatform = func() bool { return false }
	renamePath = func(source string, destination string) error {
		renameCalls++
		return os.Rename(source, destination)
	}

	var store = NewJSONStore(t.TempDir())
	if err := store.ensureParentDirectory(); err != nil {
		t.Fatalf("ensure parent directory: %v", err)
	}

	if err := os.WriteFile(store.Path(), []byte("old"), 0o600); err != nil {
		t.Fatalf("write existing setup file: %v", err)
	}

	var tempPath = filepath.Join(filepath.Dir(store.Path()), "temp-setup.json")
	if err := os.WriteFile(tempPath, []byte("new"), 0o600); err != nil {
		t.Fatalf("write temporary setup file: %v", err)
	}

	if err := store.replaceSetupFile(tempPath); err != nil {
		t.Fatalf("replace setup file: %v", err)
	}

	if renameCalls != 1 {
		t.Fatalf("expected one rename call, got %d", renameCalls)
	}

	var content, err = os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read replaced setup file: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("expected replaced content, got %q", string(content))
	}
}

func TestEnsureParentDirectoryHandlesChmodFailure(t *testing.T) {
	if ignoresPermissionBits() {
		t.Skip("permission-bit path error is ignored on this platform")
	}

	var previous = chmodPath
	defer func() { chmodPath = previous }()
	chmodPath = func(path string, mode os.FileMode) error {
		if filepath.Base(path) == "ghostfolio-cryptogains" {
			return errors.New("chmod boom")
		}
		return os.Chmod(path, mode)
	}

	var store = NewJSONStore(t.TempDir())
	if err := store.ensureParentDirectory(); err == nil {
		t.Fatalf("expected ensureParentDirectory chmod error")
	}
}

func TestLoadMapsNotFoundFromInjectedRead(t *testing.T) {
	var previous = readFile
	defer func() { readFile = previous }()
	readFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	var store = NewJSONStore(t.TempDir())
	if _, err := store.Load(context.Background()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestSaveSucceedsAndReturnsNil(t *testing.T) {
	t.Parallel()

	var store = NewJSONStore(t.TempDir())
	var config = configmodel.AppSetupConfig{SchemaVersion: 1, SetupComplete: true, ServerMode: "ghostfolio_cloud", ServerOrigin: "https://ghostfol.io"}
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("expected successful save, got %v", err)
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("expected setup file to exist: %v", err)
	}
}

func TestSaveUsesCreateTempInStoreDirectory(t *testing.T) {
	var previous = createTempFile
	defer func() { createTempFile = previous }()

	var calledDir string
	createTempFile = func(dir string, pattern string) (temporaryFile, error) {
		calledDir = dir
		return &failingTempFile{name: filepath.Join(dir, "temp"), chmodErr: errors.New("stop after capture")}, nil
	}

	var store = NewJSONStore(t.TempDir())
	_ = store.Save(context.Background(), configmodel.AppSetupConfig{})
	if calledDir != filepath.Dir(store.Path()) {
		t.Fatalf("expected temp file dir %q, got %q", filepath.Dir(store.Path()), calledDir)
	}
}

func TestCleanupTempFileIgnoresRemoveErrors(t *testing.T) {
	var previous = removePath
	defer func() { removePath = previous }()
	removePath = func(string) error { return errors.New("remove boom") }

	cleanupTempFile("ignored")
}

func TestLoadReturnsDecodedConfigAndDeleteIgnoresMissingFile(t *testing.T) {
	t.Parallel()

	var store = NewJSONStore(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(store.Path(), []byte(`{"schema_version":1,"setup_complete":true,"server_mode":"ghostfolio_cloud","server_origin":"https://ghostfol.io","allow_dev_http":false,"updated_at":"2026-01-01T00:00:00Z"}`), 0o600); err != nil {
		t.Fatalf("write setup file: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.SchemaVersion != 1 || !loaded.SetupComplete || loaded.ServerOrigin != "https://ghostfol.io" {
		t.Fatalf("unexpected decoded config: %#v", loaded)
	}
	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("delete existing file: %v", err)
	}
	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("delete missing file should be ignored: %v", err)
	}
}
