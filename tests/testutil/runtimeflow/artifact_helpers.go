package runtimeflow

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	stdruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var frameCharacterPattern = regexp.MustCompile(`[╭╮╰╯│─]`)

// NormalizeRenderedText removes presentation formatting from a rendered TUI view.
// Authored by: OpenCode
func NormalizeRenderedText(content string) string {
	return strings.Join(strings.Fields(frameCharacterPattern.ReplaceAllString(ansiEscapePattern.ReplaceAllString(content, ""), " ")), " ")
}

// MarkdownFiles returns generated Markdown files in dir.
// Authored by: OpenCode
func MarkdownFiles(t *testing.T, dir string) []string {
	t.Helper()
	var entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files
}

// InstallOpenCommandRecorder installs a local opener stub and returns its request log.
// Authored by: OpenCode
func InstallOpenCommandRecorder(t *testing.T, exitCode int) string {
	t.Helper()
	var commandName string
	switch stdruntime.GOOS {
	case "linux":
		commandName = "xdg-open"
	case "darwin":
		commandName = "open"
	default:
		t.Skipf("automatic-open integration is unsupported on %s", stdruntime.GOOS)
	}
	var fixtureDir = t.TempDir()
	var binDir = filepath.Join(fixtureDir, "bin")
	var err = os.MkdirAll(binDir, 0o700)
	if err != nil {
		t.Fatalf("mkdir opener bin dir: %v", err)
	}
	var logPath = filepath.Join(fixtureDir, "open.log")
	var script = "#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"" + logPath + "\"\nexit " + strconv.Itoa(exitCode) + "\n"
	// #nosec G306 -- the test fixture must be executable by the current user.
	err = os.WriteFile(filepath.Join(binDir, commandName), []byte(script), 0o700)
	if err != nil {
		t.Fatalf("write opener stub: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// ReadOpenCommandRequests returns paths received by the configured opener stub.
// Authored by: OpenCode
func ReadOpenCommandRequests(t *testing.T, logPath string) []string {
	t.Helper()
	// #nosec G304 -- logPath is created by InstallOpenCommandRecorder for this test.
	var raw, err = os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read opener log %q: %v", logPath, err)
	}
	var content = strings.TrimSpace(string(raw))
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// PersistedArtifactPaths returns sorted non-directory artifact paths under the
// app-managed storage root. For example, use it to inspect every persisted
// artifact written below a test-owned base directory.
// Authored by: OpenCode
func PersistedArtifactPaths(t *testing.T, baseDir string) []string {
	t.Helper()

	var root = filepath.Join(baseDir, "ghostfolio-cryptogains")
	var paths []string
	var err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk persisted artifacts: %v", err)
	}
	sort.Strings(paths)
	return paths
}

// AssertNoCleartextReportInAppStorage verifies app-managed artifacts do not contain a report.
// Authored by: OpenCode
func AssertNoCleartextReportInAppStorage(t *testing.T, baseDir string) {
	t.Helper()
	for _, path := range PersistedArtifactPaths(t, baseDir) {
		if strings.HasSuffix(path, ".md") {
			t.Fatalf("expected no Markdown file in app-managed storage, found %q", path)
		}
		// #nosec G304 -- paths are enumerated under the test-owned temporary app directory.
		var raw, readErr = os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read persisted artifact %q: %v", path, readErr)
		}
		if strings.Contains(string(raw), "# Ghostfolio Capital Gains And Losses Report") {
			t.Fatalf("expected %q to omit cleartext report content", path)
		}
	}
}
