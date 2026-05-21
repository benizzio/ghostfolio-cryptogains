// Package testutil provides reusable report-output fixtures for tests that need
// deterministic filesystem, clock, and opener behavior.
// Authored by: OpenCode
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// ReportIOFixture provides one controlled home-directory layout for report
// output tests that need a writable Documents directory and predictable
// environment variables.
// Authored by: OpenCode
type ReportIOFixture struct {
	BaseDir      string
	HomeDir      string
	DocumentsDir string
	XDGConfigDir string
}

// NewReportIOFixture creates one temporary home-directory fixture with a
// writable Documents directory and environment variables that later output or
// runtime tests can reuse.
//
// Example usage:
//
//	fixture := testutil.NewReportIOFixture(t)
//	path := fixture.ReportPath("report.md")
//
// Authored by: OpenCode
func NewReportIOFixture(t *testing.T) ReportIOFixture {
	t.Helper()

	var baseDir = t.TempDir()
	var homeDir = filepath.Join(baseDir, "home")
	var documentsDir = filepath.Join(homeDir, "Documents")
	var xdgConfigDir = filepath.Join(homeDir, ".config")

	mustMkdirAll(t, documentsDir)
	mustMkdirAll(t, xdgConfigDir)

	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("XDG_CONFIG_HOME", xdgConfigDir)

	return ReportIOFixture{
		BaseDir:      baseDir,
		HomeDir:      homeDir,
		DocumentsDir: documentsDir,
		XDGConfigDir: xdgConfigDir,
	}
}

// ReportPath returns one path inside the fixture Documents directory so tests
// can assert report writes without duplicating path joins.
//
// Example usage:
//
//	fixture := testutil.NewReportIOFixture(t)
//	path := fixture.ReportPath("ghostfolio-capital-gains-2024-fifo.md")
//
// Authored by: OpenCode
func (fixture ReportIOFixture) ReportPath(name string) string {
	return filepath.Join(fixture.DocumentsDir, name)
}

// SetXDGDocumentsDir writes one Linux user-dirs configuration that points the
// XDG Documents entry at the provided absolute path.
//
// Example usage:
//
//	fixture := testutil.NewReportIOFixture(t)
//	customDir := filepath.Join(fixture.BaseDir, "custom-documents")
//	fixture.SetXDGDocumentsDir(t, customDir)
//
// Authored by: OpenCode
func (fixture ReportIOFixture) SetXDGDocumentsDir(t *testing.T, documentsDir string) {
	t.Helper()

	if !filepath.IsAbs(documentsDir) {
		t.Fatalf("expected absolute XDG documents path, got %q", documentsDir)
	}

	mustMkdirAll(t, filepath.Dir(filepath.Join(fixture.XDGConfigDir, "user-dirs.dirs")))
	var configPath = filepath.Join(fixture.XDGConfigDir, "user-dirs.dirs")
	var configBody = fmt.Sprintf("XDG_DOCUMENTS_DIR=\"%s\"\n", escapeXDGDirValue(documentsDir))
	var err = os.WriteFile(configPath, []byte(configBody), 0o600)
	if err != nil {
		t.Fatalf("write XDG user-dirs config: %v", err)
	}
}

// StaticClock provides one deterministic Now function for report tests that
// need stable filenames or generated-at timestamps.
// Authored by: OpenCode
type StaticClock struct {
	now time.Time
}

// NewStaticClock returns one deterministic clock that always reports the same
// instant.
//
// Example usage:
//
//	clock := testutil.NewStaticClock(time.Date(2026, time.May, 20, 15, 4, 5, 0, time.Local))
//	timestamp := clock.Now()
//
// Authored by: OpenCode
func NewStaticClock(now time.Time) StaticClock {
	return StaticClock{now: now}
}

// Now returns the deterministic instant carried by the static clock.
//
// Example usage:
//
//	clock := testutil.NewStaticClock(time.Now())
//	_ = clock.Now()
//
// Authored by: OpenCode
func (clock StaticClock) Now() time.Time {
	return clock.now
}

// OpenPathSpy records opener requests and can be configured to return one
// deterministic error for tests that need success and failure assertions.
// Authored by: OpenCode
type OpenPathSpy struct {
	mu    sync.Mutex
	err   error
	paths []string
}

// NewOpenPathSpy constructs one reusable opener spy that records all requested
// paths and returns the provided error on each call.
//
// Example usage:
//
//	spy := testutil.NewOpenPathSpy(nil)
//	err := spy.Open("/tmp/report.md")
//
// Authored by: OpenCode
func NewOpenPathSpy(err error) *OpenPathSpy {
	return &OpenPathSpy{err: err}
}

// Open records one opener request path and returns the spy's configured error.
//
// Example usage:
//
//	spy := testutil.NewOpenPathSpy(errors.New("open failed"))
//	err := spy.Open("/tmp/report.md")
//
// Authored by: OpenCode
func (spy *OpenPathSpy) Open(path string) error {
	spy.mu.Lock()
	defer spy.mu.Unlock()

	spy.paths = append(spy.paths, path)
	return spy.err
}

// CallCount returns how many opener requests the spy has observed.
//
// Example usage:
//
//	spy := testutil.NewOpenPathSpy(nil)
//	_ = spy.CallCount()
//
// Authored by: OpenCode
func (spy *OpenPathSpy) CallCount() int {
	spy.mu.Lock()
	defer spy.mu.Unlock()

	return len(spy.paths)
}

// Paths returns one copy of all opener request paths observed by the spy.
//
// Example usage:
//
//	spy := testutil.NewOpenPathSpy(nil)
//	_ = spy.Paths()
//
// Authored by: OpenCode
func (spy *OpenPathSpy) Paths() []string {
	spy.mu.Lock()
	defer spy.mu.Unlock()

	var paths = make([]string, len(spy.paths))
	copy(paths, spy.paths)
	return paths
}

// AssertPathWithin verifies that one path stays inside the expected parent
// directory.
//
// Example usage:
//
//	testutil.AssertPathWithin(t, reportPath, fixture.DocumentsDir)
//
// Authored by: OpenCode
func AssertPathWithin(t *testing.T, path string, parentDir string) {
	t.Helper()

	var relativePath, err = filepath.Rel(parentDir, path)
	if err != nil {
		t.Fatalf("relative path from %q to %q: %v", parentDir, path, err)
	}

	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) || filepath.IsAbs(relativePath) {
		t.Fatalf("expected %q to stay within %q, relative path %q escapes the directory", path, parentDir, relativePath)
	}
}

// AssertRegularFile verifies that one path exists and resolves to a regular
// file.
//
// Example usage:
//
//	testutil.AssertRegularFile(t, reportPath)
//
// Authored by: OpenCode
func AssertRegularFile(t *testing.T, path string) {
	t.Helper()

	var info, err = os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("expected %q to be a regular file, got mode %s", path, info.Mode())
	}
}

// AssertPathMissing verifies that one path does not exist so tests can assert
// failed-write cleanup behavior.
//
// Example usage:
//
//	testutil.AssertPathMissing(t, reportPath)
//
// Authored by: OpenCode
func AssertPathMissing(t *testing.T, path string) {
	t.Helper()

	var _, err = os.Stat(path)
	if err == nil {
		t.Fatalf("expected %q to be absent", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected %q lookup to return not-exist, got %v", path, err)
	}
}

// WriteFixtureFile writes one deterministic test file so report-output tests can
// prepare existing target paths without duplicating boilerplate.
//
// Example usage:
//
//	testutil.WriteFixtureFile(t, fixture.ReportPath("report.md"), "content")
//
// Authored by: OpenCode
func WriteFixtureFile(t *testing.T, path string, content string) {
	t.Helper()

	mustMkdirAll(t, filepath.Dir(path))
	var err = os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("write fixture file %q: %v", path, err)
	}
}

// AssertFileContent verifies one file's full text content.
//
// Example usage:
//
//	testutil.AssertFileContent(t, fixture.ReportPath("report.md"), "# Report\n")
//
// Authored by: OpenCode
func AssertFileContent(t *testing.T, path string, expected string) {
	t.Helper()

	var content, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	if string(content) != expected {
		t.Fatalf("unexpected file content for %q: got %q want %q", path, string(content), expected)
	}
}

// mustMkdirAll creates one directory tree for the report IO fixture.
// Authored by: OpenCode
func mustMkdirAll(t *testing.T, path string) {
	t.Helper()

	var err = os.MkdirAll(path, 0o755)
	if err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}

// escapeXDGDirValue escapes one path for the simple quoted XDG user-dirs file
// format.
// Authored by: OpenCode
func escapeXDGDirValue(path string) string {
	var escaped = strings.ReplaceAll(path, `\`, `\\`)
	return strings.ReplaceAll(escaped, `"`, `\"`)
}
