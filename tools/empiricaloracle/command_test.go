package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const testVendoredHledgerJournalPath = "testdata/empirical/hledger/fifo.journal"

// TestVendoredHledgerCaptureVersionFromCommittedCurrentPlatformArtifact proves
// the committed linux-amd64 vendored binary reports the repository-supported
// hledger version.
// Authored by: OpenCode
func TestVendoredHledgerCaptureVersionFromCommittedCurrentPlatformArtifact(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("current test environment does not match the committed linux-amd64 vendored artifact")
	}

	var command, err = newVendoredHledgerCommand()
	if err != nil {
		t.Fatalf("resolve vendored hledger command: %v", err)
	}

	var version string
	version, err = command.captureVersion(context.Background())
	if err != nil {
		t.Fatalf("capture vendored hledger version: %v", err)
	}

	if version != vendoredHledgerVersion {
		t.Fatalf("unexpected vendored hledger version: got %q want %q", version, vendoredHledgerVersion)
	}
	if got, want := filepath.ToSlash(command.executableRelativePath()), "third_party/hledger/bin/linux-amd64/hledger"; got != want {
		t.Fatalf("unexpected current-platform executable path: got %q want %q", got, want)
	}
	if _, err = os.Stat(command.executablePath()); err != nil {
		t.Fatalf("stat committed vendored executable: %v", err)
	}
}

// TestVendoredHledgerBuildCommandUsesExplicitFileArgumentsAndIsolatedEnvironment
// proves the wrapper forces `-n` and `-f` while stripping environment variables
// that would leak developer-local journal or config state.
// Authored by: OpenCode
func TestVendoredHledgerBuildCommandUsesExplicitFileArgumentsAndIsolatedEnvironment(t *testing.T) {
	t.Parallel()

	var repositoryRoot = t.TempDir()
	var executablePath = writeVendoredExecutableFixture(t, repositoryRoot, "linux", "amd64", 0o755)
	var command = vendoredHledgerCommand{
		repositoryRoot: repositoryRoot,
		goos:           "linux",
		goarch:         "amd64",
		environ: func() []string {
			return []string{
				"PATH=/usr/bin",
				"HOME=/home/tester",
				"TERM=xterm-256color",
				"LEDGER_FILE=/tmp/user.journal",
				"LEDGER=ignored",
				"HLEDGER_FILE=/tmp/user.conf",
				"HLEDGER_DEBUG=1",
			}
		},
	}

	var cmd, err = command.buildCommand(context.Background(), testVendoredHledgerJournalPath, "print", "--cost")
	if err != nil {
		t.Fatalf("build vendored hledger command: %v", err)
	}

	if cmd.Path != executablePath {
		t.Fatalf("unexpected executable path: got %q want %q", cmd.Path, executablePath)
	}

	var wantArgs = []string{
		executablePath,
		"-n",
		"-f",
		filepath.Join(repositoryRoot, filepath.FromSlash(testVendoredHledgerJournalPath)),
		"print",
		"--cost",
	}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Fatalf("unexpected command arguments: got %#v want %#v", cmd.Args, wantArgs)
	}

	var wantEnv = []string{"PATH=/usr/bin", "HOME=/home/tester", "TERM=xterm-256color"}
	if !reflect.DeepEqual(cmd.Env, wantEnv) {
		t.Fatalf("unexpected isolated environment: got %#v want %#v", cmd.Env, wantEnv)
	}
}

// TestVendoredHledgerDiscoverExecutableMissingReturnsActionableError proves the
// wrapper reports the exact vendored path and platform when the current-platform
// executable artifact has not been committed.
// Authored by: OpenCode
func TestVendoredHledgerDiscoverExecutableMissingReturnsActionableError(t *testing.T) {
	t.Parallel()

	var command = vendoredHledgerCommand{
		repositoryRoot: t.TempDir(),
		goos:           "linux",
		goarch:         "amd64",
	}

	_, err := command.discoverExecutable()
	if !errors.Is(err, errVendoredHledgerExecutableMissing) {
		t.Fatalf("expected missing executable error, got %v", err)
	}

	assertEmpiricalOracleErrorContainsAll(
		t,
		err,
		"third_party/hledger/bin/linux-amd64/hledger",
		"linux-amd64",
		"fixture-backed tests without regeneration",
	)
}

// TestVendoredHledgerCaptureVersionRejectsUnsupportedVersion proves the wrapper
// blocks a vendored binary that reports any version other than `1.99.2`.
// Authored by: OpenCode
func TestVendoredHledgerCaptureVersionRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()

	var repositoryRoot = t.TempDir()
	var executablePath = writeVendoredExecutableFixture(t, repositoryRoot, "linux", "amd64", 0o755)
	var capturedEnv []string
	var command = vendoredHledgerCommand{
		repositoryRoot: repositoryRoot,
		goos:           "linux",
		goarch:         "amd64",
		environ: func() []string {
			return []string{"PATH=/usr/bin", "LEDGER_FILE=/tmp/user.journal", "HLEDGER_DEBUG=1", "HOME=/home/tester"}
		},
		runCombinedOutput: func(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
			capturedEnv = append([]string(nil), env...)
			if name != executablePath {
				t.Fatalf("unexpected executable path: got %q want %q", name, executablePath)
			}
			if !reflect.DeepEqual(args, []string{"--version"}) {
				t.Fatalf("unexpected version arguments: got %#v want %#v", args, []string{"--version"})
			}

			return []byte("hledger 1.52.1, linux-x86_64\n"), nil
		},
	}

	_, err := command.captureVersion(context.Background())
	if !errors.Is(err, errVendoredHledgerUnsupportedVersion) {
		t.Fatalf("expected unsupported version error, got %v", err)
	}

	var wantEnv = []string{"PATH=/usr/bin", "HOME=/home/tester"}
	if !reflect.DeepEqual(capturedEnv, wantEnv) {
		t.Fatalf("unexpected version-check environment: got %#v want %#v", capturedEnv, wantEnv)
	}

	assertEmpiricalOracleErrorContainsAll(
		t,
		err,
		"1.52.1",
		vendoredHledgerVersion,
		"third_party/hledger/bin/linux-amd64/hledger",
	)
}

// TestVendoredHledgerBuildCommandRejectsWrapperManagedArguments proves callers
// cannot override wrapper-managed file or config-isolation flags.
// Authored by: OpenCode
func TestVendoredHledgerBuildCommandRejectsWrapperManagedArguments(t *testing.T) {
	t.Parallel()

	var repositoryRoot = t.TempDir()
	_ = writeVendoredExecutableFixture(t, repositoryRoot, "linux", "amd64", 0o755)
	var command = vendoredHledgerCommand{
		repositoryRoot: repositoryRoot,
		goos:           "linux",
		goarch:         "amd64",
	}

	_, err := command.buildCommand(context.Background(), testVendoredHledgerJournalPath, "print", "-f", "other.journal")
	if !errors.Is(err, errVendoredHledgerArgumentConflict) {
		t.Fatalf("expected wrapper-managed argument conflict, got %v", err)
	}

	assertEmpiricalOracleErrorContainsAll(t, err, "pass the journal path to buildCommand instead of supplying -f")
}

// writeVendoredExecutableFixture creates one fake vendored executable file with
// the requested platform path and permissions.
// Authored by: OpenCode
func writeVendoredExecutableFixture(t *testing.T, repositoryRoot string, goos string, goarch string, mode os.FileMode) string {
	t.Helper()

	var executablePath = filepath.Join(repositoryRoot, "third_party", "hledger", "bin", goos+"-"+goarch, "hledger")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("create vendored executable directory: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
		t.Fatalf("write vendored executable fixture: %v", err)
	}
	if err := os.Chmod(executablePath, mode); err != nil {
		t.Fatalf("chmod vendored executable fixture: %v", err)
	}

	return executablePath
}

// assertEmpiricalOracleErrorContainsAll verifies one wrapper error contains all
// required actionable fragments.
// Authored by: OpenCode
func assertEmpiricalOracleErrorContainsAll(t *testing.T, err error, wantSubstrings ...string) {
	t.Helper()

	var message = err.Error()
	var want string
	for _, want = range wantSubstrings {
		if !strings.Contains(message, want) {
			t.Fatalf("expected error %q to contain %q", message, want)
		}
	}
}
