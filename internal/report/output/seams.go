// Package output defines local filesystem save and post-save open helpers for
// generated yearly gains-and-losses report files.
// Authored by: OpenCode
package output

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"time"
)

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
	return os.OpenFile(path, flag, perm)
}

// Test seams wrap file removal so output tests can verify cleanup behavior.
// Authored by: OpenCode
var removePath = os.Remove

// Test seams wrap timestamp generation so output tests can verify fallback time
// behavior safely.
// Authored by: OpenCode
var currentTime = time.Now

// runOpenCommand starts one opener subprocess for the provided command.
// Authored by: OpenCode
var runOpenCommand = func(command OpenCommand) error {
	var process = exec.Command(command.Name, command.Args...)
	return process.Run()
}

// InstallWriteFailureAfterCreateForTesting overrides report-file reservation so
// the returned write handle fails after the final path has already been created
// on disk. Use this only in tests that need to verify partial-file cleanup
// through higher-level runtime workflows.
//
// Example:
//
//	restore := output.InstallWriteFailureAfterCreateForTesting(errors.New("forced write failure"))
//	defer restore()
//
// Authored by: OpenCode
func InstallWriteFailureAfterCreateForTesting(writeErr error) func() {
	if writeErr == nil {
		writeErr = errors.New("forced write failure")
	}

	var previousOpenWritableFile = openWritableFile
	openWritableFile = func(path string, flag int, perm os.FileMode) (writeSyncCloser, error) {
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
