// Package output verifies package-internal output seams used by deterministic
// filesystem tests.
// Authored by: OpenCode
package output

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFailureHelpersCoverNilAndWrappedBranches verifies the small typed-output
// failure helpers directly.
// Authored by: OpenCode
func TestFailureHelpersCoverNilAndWrappedBranches(t *testing.T) {
	t.Parallel()

	var failure *Failure
	if got := failure.Error(); got != "" {
		t.Fatalf("expected nil failure error text to be empty, got %q", got)
	}
	if got := failure.Unwrap(); got != nil {
		t.Fatalf("expected nil failure unwrap to be nil, got %v", got)
	}
	if got := failure.Category(); got != "" {
		t.Fatalf("expected nil failure category to be empty, got %q", got)
	}
	if got := NewFailure(FailureCategoryReportFileWriteFailed, nil); got != nil {
		t.Fatalf("expected nil wrapped error to stay nil, got %v", got)
	}

	var wrapped = NewFailure(FailureCategoryReportFileWriteFailed, errors.New("write boom"))
	var typed *Failure
	if !errors.As(wrapped, &typed) {
		t.Fatalf("expected typed output failure, got %T", wrapped)
	}
	if got := typed.Error(); got != "write boom" {
		t.Fatalf("expected wrapped failure text, got %q", got)
	}
	if got := typed.Unwrap(); got == nil || got.Error() != "write boom" {
		t.Fatalf("expected wrapped underlying error, got %v", got)
	}
	if got := typed.Category(); got != FailureCategoryReportFileWriteFailed {
		t.Fatalf("expected wrapped failure category, got %q", got)
	}

	var cleanupPaths = []string{"/tmp/report.md", "/tmp/annex.md"}
	var residualPaths = []string{"/tmp/report.md"}
	var detailed = NewFailureWithCleanup(FailureCategoryReportFileWriteFailed, errors.New("cleanup boom"), cleanupPaths, residualPaths)
	cleanupPaths[0] = "mutated"
	residualPaths[0] = "mutated"
	var gotCleanup = CleanupPathsOf(detailed)
	var gotResidual = ResidualPathsOf(detailed)
	if len(gotCleanup) != 2 || gotCleanup[0] != "/tmp/report.md" || len(gotResidual) != 1 || gotResidual[0] != "/tmp/report.md" {
		t.Fatalf("expected defensively copied cleanup details, got cleanup=%#v residual=%#v", gotCleanup, gotResidual)
	}
	gotCleanup[0] = "mutated again"
	gotResidual[0] = "mutated again"
	if CleanupPathsOf(detailed)[0] != "/tmp/report.md" || ResidualPathsOf(detailed)[0] != "/tmp/report.md" {
		t.Fatalf("expected detail accessors to return defensive copies")
	}
	if CleanupPathsOf(errors.New("plain")) != nil || ResidualPathsOf(errors.New("plain")) != nil {
		t.Fatalf("expected plain errors to expose no cleanup context")
	}
	if !CleanupFailed(detailed) || CleanupFailed(errors.New("plain")) {
		t.Fatalf("expected cleanup failure status only for detailed cleanup failure")
	}
	if NewFailureWithCleanup(FailureCategoryReportFileWriteFailed, nil, cleanupPaths, residualPaths) != nil {
		t.Fatalf("expected nil detailed failure input to remain nil")
	}
}

// TestDeterministicWriteFailureAfterCreateError verifies package-internal env
// parsing for deterministic post-create write failures.
// Authored by: OpenCode
func TestDeterministicWriteFailureAfterCreateError(t *testing.T) {
	var previousLookupEnv = lookupEnv
	t.Cleanup(func() {
		lookupEnv = previousLookupEnv
	})

	lookupEnv = func(string) (string, bool) {
		return "   ", true
	}
	if err := deterministicWriteFailureAfterCreateError(); err != nil {
		t.Fatalf("expected blank configured write failure to be ignored, got %v", err)
	}

	lookupEnv = func(string) (string, bool) {
		return " custom write failure ", true
	}
	if err := deterministicWriteFailureAfterCreateError(); err == nil || err.Error() != "custom write failure" {
		t.Fatalf("expected trimmed configured write failure, got %v", err)
	}
	if category, ok := FailureCategoryOf(wrapFailure(FailureCategoryReportFileWriteFailed, deterministicWriteFailureAfterCreateError())); !ok || category != FailureCategoryReportFileWriteFailed {
		t.Fatalf("expected configured write failure to remain wrappable as typed output failure, got category=%q ok=%t", category, ok)
	}
}

// TestDeterministicCleanupRemovalFailureError verifies package-internal env
// parsing for deterministic cleanup removal failures.
// Authored by: OpenCode
func TestDeterministicCleanupRemovalFailureError(t *testing.T) {
	var previousLookupEnv = lookupEnv
	t.Cleanup(func() {
		lookupEnv = previousLookupEnv
	})

	lookupEnv = func(string) (string, bool) {
		return "   ", true
	}
	if err := deterministicCleanupRemovalFailureError(); err != nil {
		t.Fatalf("expected blank cleanup failure to be ignored, got %v", err)
	}

	lookupEnv = func(string) (string, bool) {
		return " cleanup removal failed ", true
	}
	if err := deterministicCleanupRemovalFailureError(); err == nil || err.Error() != "cleanup removal failed" {
		t.Fatalf("expected trimmed cleanup removal failure, got %v", err)
	}
}

// TestWrapDeterministicWriteFailure verifies both the passthrough and injected
// deterministic writer branches.
// Authored by: OpenCode
func TestWrapDeterministicWriteFailure(t *testing.T) {
	var previousLookupEnv = lookupEnv
	t.Cleanup(func() {
		lookupEnv = previousLookupEnv
	})

	var fixtureDir = t.TempDir()
	var plainPath = filepath.Join(fixtureDir, "plain.md")
	var plainFile, err = os.OpenFile(plainPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatalf("open plain fixture file: %v", err)
	}
	defer func() {
		_ = plainFile.Close()
		_ = os.Remove(plainPath)
	}()

	lookupEnv = func(string) (string, bool) {
		return "", false
	}
	if wrapped := wrapDeterministicWriteFailure(plainFile); wrapped != plainFile {
		t.Fatalf("expected no configured write failure to return the original file, got %T", wrapped)
	}

	var failingPath = filepath.Join(fixtureDir, "failing.md")
	var failingFile, failingErr = os.OpenFile(failingPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if failingErr != nil {
		t.Fatalf("open failing fixture file: %v", failingErr)
	}
	defer func() {
		_ = failingFile.Close()
		_ = os.Remove(failingPath)
	}()

	lookupEnv = func(string) (string, bool) {
		return "forced env write failure", true
	}
	var wrapped = wrapDeterministicWriteFailure(failingFile)
	if _, ok := wrapped.(failingWriteSyncCloser); !ok {
		t.Fatalf("expected configured write failure to wrap the file, got %T", wrapped)
	}
	if _, failingErr = wrapped.Write([]byte("content")); failingErr == nil || !strings.Contains(failingErr.Error(), "forced env write failure") {
		t.Fatalf("expected wrapped file write to fail with configured error, got %v", failingErr)
	}
	if failingErr = wrapped.Close(); failingErr != nil {
		t.Fatalf("close wrapped fixture file: %v", failingErr)
	}
}
