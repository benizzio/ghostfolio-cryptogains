// Package output verifies package-local documents-directory parsing and seam
// helpers.
// Authored by: OpenCode
package output

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveDocumentsDirectoryUsesCurrentGOOS verifies the exported resolver's
// current-platform seam.
// Authored by: OpenCode
func TestResolveDocumentsDirectoryUsesCurrentGOOS(t *testing.T) {
	var fixtureDir = t.TempDir()
	var expected = filepath.Join(fixtureDir, "Documents")
	if err := os.MkdirAll(expected, 0o755); err != nil {
		t.Fatalf("mkdir documents: %v", err)
	}

	var previousCurrentGOOS = currentGOOS
	var previousUserHomeDirectory = userHomeDirectory
	t.Cleanup(func() {
		currentGOOS = previousCurrentGOOS
		userHomeDirectory = previousUserHomeDirectory
	})

	currentGOOS = func() string { return "darwin" }
	userHomeDirectory = func() (string, error) { return fixtureDir, nil }

	var documentsDir, err = ResolveDocumentsDirectory()
	if err != nil {
		t.Fatalf("resolve documents directory through current GOOS seam: %v", err)
	}
	if documentsDir != expected {
		t.Fatalf("unexpected documents directory: got %q want %q", documentsDir, expected)
	}
}

// TestLinuxDocumentsParsingErrors verifies XDG parsing and read failures that
// the higher-level exported tests do not hit directly.
// Authored by: OpenCode
func TestLinuxDocumentsParsingErrors(t *testing.T) {
	var previousLookupEnv = lookupEnv
	t.Cleanup(func() {
		lookupEnv = previousLookupEnv
	})
	lookupEnv = func(string) (string, bool) { return "", false }

	var configDir, configured, err = resolveLinuxDocumentsDirectory(t.TempDir())
	if err != nil {
		t.Fatalf("missing XDG config should not fail: %v", err)
	}
	if configured || configDir != "" {
		t.Fatalf("expected missing XDG config to report no configured documents directory")
	}

	var previousReadFile = readFile
	t.Cleanup(func() {
		readFile = previousReadFile
	})

	readFile = func(string) ([]byte, error) { return nil, errors.New("boom") }
	_, _, err = resolveLinuxDocumentsDirectory(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "read Linux XDG user-dirs config") {
		t.Fatalf("expected XDG config read failure to be wrapped, got %v", err)
	}

	for _, testCase := range []struct {
		name       string
		configBody string
		wantErr    string
	}{
		{name: "unquoted path", configBody: `XDG_DOCUMENTS_DIR=/tmp/docs`, wantErr: `quoted path`},
		{name: "empty path", configBody: `XDG_DOCUMENTS_DIR="   "`, wantErr: `must not be empty`},
		{name: "relative path", configBody: `XDG_DOCUMENTS_DIR="Documents"`, wantErr: `is not absolute`},
		{name: "invalid escaped path", configBody: "XDG_DOCUMENTS_DIR=\"$HOME/docs\\\"", wantErr: `incomplete escape`},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			_, _, err := parseXDGDocumentsDirectory(testCase.configBody, "/home/test")
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("expected parse error containing %q, got %v", testCase.wantErr, err)
			}
		})
	}

	var homeOnly, found, homeErr = parseXDGDocumentsDirectory(`XDG_DOCUMENTS_DIR="$HOME"`, "/home/test")
	if homeErr != nil || !found || homeOnly != "/home/test" {
		t.Fatalf("expected $HOME XDG documents entry to resolve to the home directory, got path=%q found=%t err=%v", homeOnly, found, homeErr)
	}

	if _, err = unescapeXDGPath(`unfinished\`); err == nil {
		t.Fatalf("expected incomplete XDG escape to fail")
	}

	var escaped, escapedErr = unescapeXDGPath(`folder\"name\\docs`)
	if escapedErr != nil || escaped != `folder"name\docs` {
		t.Fatalf("expected XDG escape decoding to succeed, got %q err=%v", escaped, escapedErr)
	}

	var nestedHome, nestedFound, nestedErr = parseXDGDocumentsDirectory("\n# comment\nOTHER_DIR=\"/tmp/other\"\nXDG_DOCUMENTS_DIR=\"$HOME/work/docs\"\n", "/home/test")
	if nestedErr != nil || !nestedFound || nestedHome != filepath.Join("/home/test", "work", "docs") {
		t.Fatalf("expected nested $HOME documents path to resolve, got path=%q found=%t err=%v", nestedHome, nestedFound, nestedErr)
	}

	var missingPath, missingFound, missingErr = parseXDGDocumentsDirectory("\n# comment\nOTHER_DIR=\"/tmp/other\"\n", "/home/test")
	if missingErr != nil || missingFound || missingPath != "" {
		t.Fatalf("expected missing XDG documents entry to return not found, got path=%q found=%t err=%v", missingPath, missingFound, missingErr)
	}
}

// TestResolveDocumentsDirectoryAdditionalBranches verifies remaining exported
// and seam-driven documents-directory branches.
// Authored by: OpenCode
func TestResolveDocumentsDirectoryAdditionalBranches(t *testing.T) {
	t.Run("propagates home directory resolution failure", func(t *testing.T) {
		var previousUserHomeDirectory = userHomeDirectory
		defer func() {
			userHomeDirectory = previousUserHomeDirectory
		}()

		userHomeDirectory = func() (string, error) { return "", errors.New("home boom") }
		if _, err := ResolveDocumentsDirectoryForOS("darwin"); err == nil || !strings.Contains(err.Error(), "resolve user home directory") {
			t.Fatalf("expected home directory resolution failure, got %v", err)
		}
	})

	t.Run("propagates linux parser failure through exported resolver", func(t *testing.T) {
		var homeDir = t.TempDir()
		var previousLookupEnv = lookupEnv
		var previousUserHomeDirectory = userHomeDirectory
		var previousReadFile = readFile
		defer func() {
			lookupEnv = previousLookupEnv
			userHomeDirectory = previousUserHomeDirectory
			readFile = previousReadFile
		}()

		lookupEnv = func(key string) (string, bool) {
			if key == "XDG_CONFIG_HOME" {
				return filepath.Join(homeDir, ".config"), true
			}
			return "", false
		}
		userHomeDirectory = func() (string, error) { return homeDir, nil }
		readFile = func(string) ([]byte, error) {
			return []byte(`XDG_DOCUMENTS_DIR="relative-path"`), nil
		}

		if _, err := ResolveDocumentsDirectoryForOS("linux"); err == nil || !strings.Contains(err.Error(), "not absolute") {
			t.Fatalf("expected exported Linux resolver to propagate parse failure, got %v", err)
		}
	})

	t.Run("linux resolver reports present config without documents entry", func(t *testing.T) {
		var previousLookupEnv = lookupEnv
		var previousReadFile = readFile
		defer func() {
			lookupEnv = previousLookupEnv
			readFile = previousReadFile
		}()

		lookupEnv = func(string) (string, bool) { return "", false }
		readFile = func(string) ([]byte, error) {
			return []byte("# no documents entry\nXDG_DESKTOP_DIR=\"$HOME/Desktop\"\n"), nil
		}

		var path, configured, err = resolveLinuxDocumentsDirectory(t.TempDir())
		if err != nil || configured || path != "" {
			t.Fatalf("expected no configured Linux documents path, got path=%q configured=%t err=%v", path, configured, err)
		}
	})
}

// TestResolveHomeDirectoryFallbacks verifies Windows USERPROFILE preference and
// home-directory error propagation.
// Authored by: OpenCode
func TestResolveHomeDirectoryFallbacks(t *testing.T) {
	var previousLookupEnv = lookupEnv
	var previousUserHomeDirectory = userHomeDirectory
	t.Cleanup(func() {
		lookupEnv = previousLookupEnv
		userHomeDirectory = previousUserHomeDirectory
	})

	lookupEnv = func(key string) (string, bool) {
		if key == "USERPROFILE" {
			return filepath.Join("C:\\", "Users", "alice", "..", "alice"), true
		}
		return "", false
	}
	userHomeDirectory = func() (string, error) { return "/should-not-be-used", nil }

	var windowsHome, err = resolveHomeDirectory("windows")
	if err != nil {
		t.Fatalf("resolve Windows home directory through USERPROFILE: %v", err)
	}
	if windowsHome != filepath.Clean(filepath.Join("C:\\", "Users", "alice", "..", "alice")) {
		t.Fatalf("unexpected Windows home directory: %q", windowsHome)
	}

	lookupEnv = func(string) (string, bool) { return "", false }
	userHomeDirectory = func() (string, error) { return "", errors.New("boom") }
	if _, err = resolveHomeDirectory("linux"); err == nil || !strings.Contains(err.Error(), "resolve user home directory") {
		t.Fatalf("expected user-home lookup error to be wrapped, got %v", err)
	}

	userHomeDirectory = func() (string, error) { return "  ", nil }
	if _, err = resolveHomeDirectory("linux"); err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("expected empty user-home directory to fail, got %v", err)
	}
}

// TestResolveDocumentsDirectoryForOSUnsupported verifies the exported platform
// guardrail for unsupported operating systems.
// Authored by: OpenCode
func TestResolveDocumentsDirectoryForOSUnsupported(t *testing.T) {
	var previousUserHomeDirectory = userHomeDirectory
	t.Cleanup(func() {
		userHomeDirectory = previousUserHomeDirectory
	})

	userHomeDirectory = func() (string, error) { return "", errors.New("home boom") }

	_, err := ResolveDocumentsDirectoryForOS("plan9")
	if err == nil || !strings.Contains(err.Error(), `unsupported on "plan9"`) {
		t.Fatalf("expected unsupported-platform error, got %v", err)
	}
	if strings.Contains(err.Error(), "home boom") {
		t.Fatalf("expected unsupported platform to be reported before home-directory failure, got %v", err)
	}
}

// TestDocumentsDirectoryAndOpenerAdditionalBranches verifies remaining path and
// opener helper branches through direct seam control.
// Authored by: OpenCode
func TestDocumentsDirectoryAndOpenerAdditionalBranches(t *testing.T) {
	t.Run("resolves configured Linux absolute documents path", func(t *testing.T) {
		var homeDir = t.TempDir()
		var documentsDir = filepath.Join(homeDir, "custom-docs")
		if err := os.MkdirAll(filepath.Join(homeDir, ".config"), 0o755); err != nil {
			t.Fatalf("mkdir config dir: %v", err)
		}

		var previousLookupEnv = lookupEnv
		var previousReadFile = readFile
		defer func() {
			lookupEnv = previousLookupEnv
			readFile = previousReadFile
		}()

		lookupEnv = func(key string) (string, bool) {
			if key == "XDG_CONFIG_HOME" {
				return filepath.Join(homeDir, ".config"), true
			}
			return "", false
		}
		readFile = func(string) ([]byte, error) {
			return []byte(`XDG_DOCUMENTS_DIR="` + documentsDir + `"`), nil
		}

		var resolved, err = ResolveDocumentsDirectoryForOS("linux")
		if err != nil {
			t.Fatalf("resolve configured Linux documents directory: %v", err)
		}
		if resolved != documentsDir {
			t.Fatalf("unexpected configured documents directory: got %q want %q", resolved, documentsDir)
		}
	})

	t.Run("supports darwin and windows open command resolution", func(t *testing.T) {
		var command, err = ResolveOpenCommandForOS("darwin", "/tmp/report.md")
		if err != nil || command.Name != "open" || len(command.Args) != 1 || command.Args[0] != "/tmp/report.md" {
			t.Fatalf("unexpected darwin open command: %#v err=%v", command, err)
		}

		command, err = ResolveOpenCommandForOS("windows", `C:\Reports\report.md`)
		if err != nil || command.Name != "cmd" || len(command.Args) != 4 || command.Args[2] != "" {
			t.Fatalf("unexpected windows open command: %#v err=%v", command, err)
		}
	})

	t.Run("rejects empty and unsupported open path requests", func(t *testing.T) {
		if _, err := ResolveOpenCommandForOS("linux", ""); err == nil || !strings.Contains(err.Error(), "report path is required") {
			t.Fatalf("expected empty path to fail, got %v", err)
		}
		if _, err := ResolveOpenCommandForOS("plan9", "/tmp/report.md"); err == nil || !strings.Contains(err.Error(), "automatic report opening is unsupported") {
			t.Fatalf("expected unsupported open-command platform, got %v", err)
		}

		var previousCurrentGOOS = currentGOOS
		defer func() {
			currentGOOS = previousCurrentGOOS
		}()
		currentGOOS = func() string { return "plan9" }
		if err := OpenPath("/tmp/report.md"); err == nil || !strings.Contains(err.Error(), "automatic report opening is unsupported") {
			t.Fatalf("expected exported unsupported open path error, got %v", err)
		}
	})
}

// TestInstallWriteFailureAfterCreateForTestingDefaultsAndRestores verifies the
// default injected error and seam restoration.
// Authored by: OpenCode
func TestInstallWriteFailureAfterCreateForTestingDefaultsAndRestores(t *testing.T) {
	var fixtureDir = t.TempDir()
	var reservedPath = filepath.Join(fixtureDir, "report.md")
	var previousOpenWritableFile = openWritableFile
	t.Cleanup(func() {
		openWritableFile = previousOpenWritableFile
	})

	var restore = installWriteFailureAfterCreateForTesting(nil)
	var file, err = openWritableFile(reservedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatalf("reserve test file with injected write failure: %v", err)
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(reservedPath)
	}()

	if _, err = file.Write([]byte("content")); err == nil || !strings.Contains(err.Error(), "forced write failure") {
		t.Fatalf("expected injected default write failure, got %v", err)
	}
	if err = file.Close(); err != nil {
		t.Fatalf("close reserved test file: %v", err)
	}
	if err = os.Remove(reservedPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove reserved test file: %v", err)
	}

	restore()
	var restoredPath = filepath.Join(fixtureDir, "restored.md")
	var restoredFile, restoredErr = openWritableFile(restoredPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if restoredErr != nil {
		t.Fatalf("expected restored file opener to work, got %v", restoredErr)
	}
	if _, restoredErr = restoredFile.Write([]byte("content")); restoredErr != nil {
		t.Fatalf("expected restored file opener not to inject write errors, got %v", restoredErr)
	}
	if restoredErr = restoredFile.Close(); restoredErr != nil {
		t.Fatalf("close restored test file: %v", restoredErr)
	}
	if err = os.Remove(restoredPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove restored test file: %v", err)
	}
}

// TestFailureCategoryOf verifies typed output failure extraction.
// Authored by: OpenCode
func TestFailureCategoryOf(t *testing.T) {
	var wrapped = wrapFailure(FailureCategoryDocumentsDirectoryUnavailable, errors.New("boom"))
	var category, ok = FailureCategoryOf(wrapped)
	if !ok || category != FailureCategoryDocumentsDirectoryUnavailable {
		t.Fatalf("expected typed documents-directory failure, got category=%q ok=%t", category, ok)
	}

	category, ok = FailureCategoryOf(errors.New("plain"))
	if ok || category != "" {
		t.Fatalf("expected plain error not to expose an output failure category, got category=%q ok=%t", category, ok)
	}
}
