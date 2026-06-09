package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const vendoredHledgerVersion = "1.99.2"

var (
	errVendoredHledgerArgumentConflict        = errors.New("vendored hledger arguments conflict with wrapper-managed flags")
	errVendoredHledgerExecutableMissing       = errors.New("vendored hledger executable is missing")
	errVendoredHledgerExecutableNotExecutable = errors.New("vendored hledger executable is not executable")
	errVendoredHledgerRepositoryRootMissing   = errors.New("vendored hledger repository root could not be resolved")
	errVendoredHledgerUnsupportedVersion      = errors.New("vendored hledger version is unsupported")
	errVendoredHledgerVersionOutputInvalid    = errors.New("vendored hledger version output is invalid")
)

// execCommandFunc builds one `exec.Cmd` for the vendored hledger wrapper.
// Authored by: OpenCode
type execCommandFunc func(ctx context.Context, name string, args ...string) *exec.Cmd

// runCombinedOutputFunc executes one command and returns its combined output.
// Authored by: OpenCode
type runCombinedOutputFunc func(ctx context.Context, env []string, name string, args ...string) ([]byte, error)

// vendoredHledgerCommand resolves, validates, and builds commands for the
// repository-vendored hledger boundary used by empirical tooling only.
// Authored by: OpenCode
type vendoredHledgerCommand struct {
	repositoryRoot    string
	goos              string
	goarch            string
	environ           func() []string
	execCommand       execCommandFunc
	runCombinedOutput runCombinedOutputFunc
}

// newVendoredHledgerCommand builds one wrapper configured for the current
// repository checkout and the current Go platform.
// Authored by: OpenCode
func newVendoredHledgerCommand() (vendoredHledgerCommand, error) {
	var repositoryRoot, err = resolveEmpiricalRepositoryRoot()
	if err != nil {
		return vendoredHledgerCommand{}, err
	}

	return vendoredHledgerCommand{
		repositoryRoot:    repositoryRoot,
		goos:              runtime.GOOS,
		goarch:            runtime.GOARCH,
		environ:           os.Environ,
		execCommand:       exec.CommandContext,
		runCombinedOutput: runVendoredHledgerCombinedOutput,
	}, nil
}

// executableRelativePath returns the repository-relative vendored executable
// path for the configured platform.
// Authored by: OpenCode
func (command vendoredHledgerCommand) executableRelativePath() string {
	return path.Join("third_party", "hledger", "bin", command.platformDirectory(), "hledger")
}

// executablePath returns the absolute vendored executable path for the
// configured platform.
// Authored by: OpenCode
func (command vendoredHledgerCommand) executablePath() string {
	return filepath.Join(command.repositoryRoot, filepath.FromSlash(command.executableRelativePath()))
}

// platformDirectory returns the `<goos>-<goarch>` directory name used by the
// vendored executable layout.
// Authored by: OpenCode
func (command vendoredHledgerCommand) platformDirectory() string {
	return fmt.Sprintf("%s-%s", command.goos, command.goarch)
}

// discoverExecutable validates that the current-platform vendored executable is
// present and runnable from the repository-managed location.
// Authored by: OpenCode
func (command vendoredHledgerCommand) discoverExecutable() (string, error) {
	var executablePath = command.executablePath()
	var executableInfo, err = os.Stat(executablePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf(
				"%w: expected %s for current platform %s; commit the vendored executable artifact or use fixture-backed tests without regeneration",
				errVendoredHledgerExecutableMissing,
				command.executableRelativePath(),
				command.platformDirectory(),
			)
		}

		return "", fmt.Errorf("stat vendored hledger executable %s: %w", command.executableRelativePath(), err)
	}

	if executableInfo.IsDir() {
		return "", fmt.Errorf("%w: %s points to a directory, not an executable file", errVendoredHledgerExecutableMissing, command.executableRelativePath())
	}

	if command.goos != "windows" && executableInfo.Mode()&0o111 == 0 {
		return "", fmt.Errorf(
			"%w: %s exists but lacks execute permissions; run `chmod +x %s` on the vendored artifact",
			errVendoredHledgerExecutableNotExecutable,
			command.executableRelativePath(),
			command.executableRelativePath(),
		)
	}

	return executablePath, nil
}

// captureVersion runs the vendored executable with `--version`, parses the
// reported version, and enforces the repository-supported hledger release.
// Authored by: OpenCode
func (command vendoredHledgerCommand) captureVersion(ctx context.Context) (string, error) {
	var executablePath, err = command.discoverExecutable()
	if err != nil {
		return "", err
	}

	var rawOutput []byte
	if command.runCombinedOutput != nil {
		rawOutput, err = command.runCombinedOutput(ctx, command.isolatedEnvironment(), executablePath, "--version")
	} else {
		rawOutput, err = runVendoredHledgerCombinedOutput(ctx, command.isolatedEnvironment(), executablePath, "--version")
	}
	if err != nil {
		return "", fmt.Errorf("run vendored hledger --version: %w", err)
	}

	var version, parseErr = parseVendoredHledgerVersion(string(rawOutput))
	if parseErr != nil {
		return "", parseErr
	}

	if version != vendoredHledgerVersion {
		return "", fmt.Errorf(
			"%w: got %s from %s; expected %s",
			errVendoredHledgerUnsupportedVersion,
			version,
			command.executableRelativePath(),
			vendoredHledgerVersion,
		)
	}

	return version, nil
}

// buildCommand constructs one explicit, config-isolated hledger invocation for
// the provided journal path and caller-managed subcommand arguments.
// Authored by: OpenCode
func (command vendoredHledgerCommand) buildCommand(ctx context.Context, journalPath string, args ...string) (*exec.Cmd, error) {
	var executablePath, err = command.discoverExecutable()
	if err != nil {
		return nil, err
	}

	var resolvedJournalPath, resolveErr = command.resolveJournalPath(journalPath)
	if resolveErr != nil {
		return nil, resolveErr
	}

	if err := validateVendoredHledgerArguments(args); err != nil {
		return nil, err
	}

	var invocationArguments = make([]string, 0, len(args)+3)
	invocationArguments = append(invocationArguments, "-n", "-f", resolvedJournalPath)
	invocationArguments = append(invocationArguments, args...)

	var buildExecCommand = command.execCommand
	if buildExecCommand == nil {
		buildExecCommand = exec.CommandContext
	}

	var cmd = buildExecCommand(ctx, executablePath, invocationArguments...)
	cmd.Env = command.isolatedEnvironment()

	return cmd, nil
}

// isolatedEnvironment removes environment variables that would cause hledger to
// pick up a developer-local default journal or config.
// Authored by: OpenCode
func (command vendoredHledgerCommand) isolatedEnvironment() []string {
	var environmentLookup = command.environ
	if environmentLookup == nil {
		environmentLookup = os.Environ
	}

	var filteredEnvironment = make([]string, 0, len(environmentLookup()))
	var rawEntry string
	for _, rawEntry = range environmentLookup() {
		var name, _, foundSeparator = strings.Cut(rawEntry, "=")
		if !foundSeparator {
			filteredEnvironment = append(filteredEnvironment, rawEntry)
			continue
		}

		var uppercaseName = strings.ToUpper(name)
		switch {
		case uppercaseName == "LEDGER_FILE":
			continue
		case uppercaseName == "LEDGER":
			continue
		case strings.HasPrefix(uppercaseName, "HLEDGER_"):
			continue
		}

		filteredEnvironment = append(filteredEnvironment, rawEntry)
	}

	return filteredEnvironment
}

// resolveJournalPath converts one journal path into the absolute path that will
// be passed explicitly with `-f`.
// Authored by: OpenCode
func (command vendoredHledgerCommand) resolveJournalPath(journalPath string) (string, error) {
	var trimmedJournalPath = strings.TrimSpace(journalPath)
	if trimmedJournalPath == "" {
		return "", fmt.Errorf("%w: journal path is required and must be supplied separately from wrapper-managed flags", errVendoredHledgerArgumentConflict)
	}

	if filepath.IsAbs(trimmedJournalPath) {
		return trimmedJournalPath, nil
	}

	return filepath.Join(command.repositoryRoot, filepath.FromSlash(trimmedJournalPath)), nil
}

// resolveEmpiricalRepositoryRoot resolves the repository root from this source
// file's location so tests and tooling do not depend on the current working
// directory.
// Authored by: OpenCode
func resolveEmpiricalRepositoryRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("%w: runtime caller lookup failed", errVendoredHledgerRepositoryRootMissing)
	}

	var repositoryRoot = filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	var goModPath = filepath.Join(repositoryRoot, "go.mod")
	var repositoryMarker, err = os.Stat(goModPath)
	if err != nil {
		return "", fmt.Errorf("%w: expected repository marker %s: %w", errVendoredHledgerRepositoryRootMissing, goModPath, err)
	}
	if repositoryMarker.IsDir() {
		return "", fmt.Errorf("%w: repository marker %s is a directory", errVendoredHledgerRepositoryRootMissing, goModPath)
	}

	return repositoryRoot, nil
}

// runVendoredHledgerCombinedOutput executes one vendored hledger command with
// the supplied isolated environment.
// Authored by: OpenCode
func runVendoredHledgerCombinedOutput(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	var cmd = exec.CommandContext(ctx, name, args...)
	cmd.Env = env

	return cmd.CombinedOutput()
}

// parseVendoredHledgerVersion extracts the semantic version token from
// `hledger --version` output.
// Authored by: OpenCode
func parseVendoredHledgerVersion(rawOutput string) (string, error) {
	var trimmedOutput = strings.TrimSpace(rawOutput)
	if trimmedOutput == "" {
		return "", fmt.Errorf("%w: empty output", errVendoredHledgerVersionOutputInvalid)
	}

	var firstLine, _, _ = strings.Cut(trimmedOutput, "\n")
	if !strings.HasPrefix(firstLine, "hledger ") {
		return "", fmt.Errorf("%w: %q", errVendoredHledgerVersionOutputInvalid, firstLine)
	}

	var versionFragment = strings.TrimPrefix(firstLine, "hledger ")
	var versionTokens = strings.FieldsFunc(versionFragment, func(character rune) bool {
		return character == ',' || character == ' ' || character == '\t'
	})
	if len(versionTokens) == 0 || strings.TrimSpace(versionTokens[0]) == "" {
		return "", fmt.Errorf("%w: %q", errVendoredHledgerVersionOutputInvalid, firstLine)
	}

	return versionTokens[0], nil
}

// validateVendoredHledgerArguments rejects caller-managed arguments that would
// bypass the wrapper's explicit file and config isolation policy.
// Authored by: OpenCode
func validateVendoredHledgerArguments(args []string) error {
	var rawArgument string
	for _, rawArgument = range args {
		switch {
		case rawArgument == "-f":
			return fmt.Errorf("%w: pass the journal path to buildCommand instead of supplying -f", errVendoredHledgerArgumentConflict)
		case rawArgument == "--file":
			return fmt.Errorf("%w: pass the journal path to buildCommand instead of supplying --file", errVendoredHledgerArgumentConflict)
		case strings.HasPrefix(rawArgument, "--file="):
			return fmt.Errorf("%w: pass the journal path to buildCommand instead of supplying --file", errVendoredHledgerArgumentConflict)
		case rawArgument == "-n":
			return fmt.Errorf("%w: buildCommand always adds -n to ignore user config", errVendoredHledgerArgumentConflict)
		case rawArgument == "--no-conf":
			return fmt.Errorf("%w: buildCommand always adds --no-conf via -n semantics", errVendoredHledgerArgumentConflict)
		case rawArgument == "--conf":
			return fmt.Errorf("%w: buildCommand forbids --conf to prevent user config leakage", errVendoredHledgerArgumentConflict)
		case strings.HasPrefix(rawArgument, "--conf="):
			return fmt.Errorf("%w: buildCommand forbids --conf to prevent user config leakage", errVendoredHledgerArgumentConflict)
		}
	}

	return nil
}
