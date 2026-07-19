// Package output defines local filesystem save and post-save open helpers for
// generated yearly gains-and-losses report files.
// Authored by: OpenCode
package output

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// deterministicWriteFailureAfterCreateEnvName enables one package-internal
// deterministic write failure after the final report path has already been
// created on disk.
// Authored by: OpenCode
const deterministicWriteFailureAfterCreateEnvName = "GHOSTFOLIO_CRYPTOGAINS_OUTPUT_FAIL_WRITE_AFTER_CREATE"

// deterministicCleanupRemovalFailureEnvName enables one package-internal
// deterministic removal failure after a failed report output attempt.
// Authored by: OpenCode
const deterministicCleanupRemovalFailureEnvName = "GHOSTFOLIO_CRYPTOGAINS_OUTPUT_FAIL_CLEANUP_REMOVE"

// Test seams wrap platform and filesystem behavior so output tests can verify
// failure handling without mutating the host system.
// Authored by: OpenCode
var currentGOOS = func() string {
	return runtime.GOOS
}

// Test seams wrap environment lookups so output tests can control platform
// directory resolution.
// Authored by: OpenCode
var lookupEnv = os.LookupEnv

// Test seams wrap home-directory lookup so output tests can exercise fallback
// behavior safely.
// Authored by: OpenCode
var userHomeDirectory = os.UserHomeDir

// Test seams wrap config-file reads so output tests can inject XDG resolution
// failures safely.
// Authored by: OpenCode
var readFile = os.ReadFile

// Test seams wrap stat calls so output tests can verify directory failures.
// Authored by: OpenCode
var statPath = os.Stat

// Test seams wrap exclusive file creation so output tests can inject partial
// write failures safely.
// Authored by: OpenCode
var openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
	// #nosec G304 -- path is the validated, collision-safe report destination supplied to this filesystem seam.
	var file, err = os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}

	return wrapDeterministicWriteFailure(file), nil
}

// Test seams wrap file removal so output tests can verify cleanup behavior.
// Authored by: OpenCode
var removePath = func(path string) error {
	var removeErr = deterministicCleanupRemovalFailureError()
	if removeErr != nil {
		return removeErr
	}
	return os.Remove(path)
}

// Test seams wrap timestamp generation so output tests can verify fallback time
// behavior safely.
// Authored by: OpenCode
var currentTime = time.Now

// runOpenCommand starts one opener subprocess for the provided command.
// Authored by: OpenCode
var runOpenCommand = func(command OpenCommand) error {
	// #nosec G204 -- command is the platform-specific opener selected by the validated post-save opener seam.
	var process = exec.Command(command.Name, command.Args...)
	return process.Run()
}

// wrapDeterministicWriteFailure applies one package-internal deterministic
// write failure configuration to a newly created writable file.
// Authored by: OpenCode
func wrapDeterministicWriteFailure(file *os.File) writeSyncCloser {
	var writeErr = deterministicWriteFailureAfterCreateError()
	if writeErr == nil {
		return file
	}

	return failingWriteSyncCloser{writeSyncCloser: file, writeErr: writeErr}
}

// deterministicWriteFailureAfterCreateError returns the configured package-
// internal deterministic write failure, if any.
// Authored by: OpenCode
func deterministicWriteFailureAfterCreateError() error {
	var value, ok = lookupEnv(deterministicWriteFailureAfterCreateEnvName)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}

	return errors.New(strings.TrimSpace(value))
}

// deterministicCleanupRemovalFailureError returns the configured package-
// internal cleanup removal failure, if any.
// Authored by: OpenCode
func deterministicCleanupRemovalFailureError() error {
	var value, ok = lookupEnv(deterministicCleanupRemovalFailureEnvName)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}

	return errors.New(strings.TrimSpace(value))
}

// installWriteFailureAfterCreateForTesting overrides report-file reservation so
// the returned write handle fails after the final path has already been created
// on disk. Use this only in package-local tests.
// Authored by: OpenCode
func installWriteFailureAfterCreateForTesting(writeErr error) func() {
	if writeErr == nil {
		writeErr = errors.New("forced write failure")
	}

	var previousOpenWritableFile = openWritableFile
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
		// #nosec G304 -- path is the validated report destination passed through this deterministic test seam.
		var file, err = os.OpenFile(path, flag, perm)
		if err != nil {
			return nil, err
		}

		return failingWriteSyncCloser{writeSyncCloser: file, writeErr: writeErr}, nil
	}

	return func() {
		openWritableFile = previousOpenWritableFile
	}
}

// failingWriteSyncCloser forces one deterministic write error after the report
// path has already been reserved.
// Authored by: OpenCode
type failingWriteSyncCloser struct {
	writeSyncCloser
	writeErr error
}

// Write returns the configured deterministic write error.
// Authored by: OpenCode
func (file failingWriteSyncCloser) Write([]byte) (int, error) {
	return 0, file.writeErr
}
