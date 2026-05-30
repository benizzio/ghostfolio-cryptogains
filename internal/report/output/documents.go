// Package output defines local filesystem save and post-save open helpers for
// generated yearly gains-and-losses report files.
// Authored by: OpenCode
package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveDocumentsDirectory returns the current user's report output directory
// using the active platform conventions supported by this slice.
//
// Example:
//
//	documentsDir, err := output.ResolveDocumentsDirectory()
//	if err != nil {
//		panic(err)
//	}
//	_ = documentsDir
//
// This helper supports Linux XDG user-dirs with a `$HOME/Documents` fallback,
// macOS `~/Documents`, and Windows `%USERPROFILE%\Documents`.
// Authored by: OpenCode
func ResolveDocumentsDirectory() (string, error) {
	return ResolveDocumentsDirectoryForOS(currentGOOS())
}

// ResolveDocumentsDirectoryForOS returns the current user's report output
// directory for the provided operating system identifier.
//
// Example:
//
//	documentsDir, err := output.ResolveDocumentsDirectoryForOS("linux")
//	if err != nil {
//		panic(err)
//	}
//	_ = documentsDir
//
// Use this helper from tests when the platform-specific resolution rules need
// to be verified without depending on the host operating system.
// Authored by: OpenCode
func ResolveDocumentsDirectoryForOS(goos string) (string, error) {
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return "", wrapFailure(
			FailureCategoryDocumentsDirectoryUnavailable,
			fmt.Errorf("documents directory resolution is unsupported on %q", goos),
		)
	}

	var homeDir, err = resolveHomeDirectory(goos)
	if err != nil {
		return "", wrapFailure(FailureCategoryDocumentsDirectoryUnavailable, err)
	}

	switch goos {
	case "linux":
		var documentsDir, configured, resolveErr = resolveLinuxDocumentsDirectory(homeDir)
		if resolveErr != nil {
			return "", wrapFailure(FailureCategoryDocumentsDirectoryUnavailable, resolveErr)
		}
		if configured {
			return documentsDir, nil
		}
		return filepath.Join(homeDir, "Documents"), nil
	case "darwin":
		return filepath.Join(homeDir, "Documents"), nil
	case "windows":
		return filepath.Join(homeDir, "Documents"), nil
	}

	return "", wrapFailure(
		FailureCategoryDocumentsDirectoryUnavailable,
		fmt.Errorf("documents directory resolution is unsupported on %q", goos),
	)
}

// resolveLinuxDocumentsDirectory resolves the XDG Documents directory when the
// user-dirs configuration declares one.
// Authored by: OpenCode
func resolveLinuxDocumentsDirectory(homeDir string) (string, bool, error) {
	var configHome, ok = lookupEnv("XDG_CONFIG_HOME")
	if !ok || strings.TrimSpace(configHome) == "" {
		configHome = filepath.Join(homeDir, ".config")
	}

	var configPath = filepath.Join(configHome, "user-dirs.dirs")
	var configBody, err = readFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read Linux XDG user-dirs config %q: %w", configPath, err)
	}

	var configuredDir, found, resolveErr = parseXDGDocumentsDirectory(string(configBody), homeDir)
	if resolveErr != nil {
		return "", false, resolveErr
	}
	if !found {
		return "", false, nil
	}

	return configuredDir, true, nil
}

// parseXDGDocumentsDirectory extracts the configured XDG Documents entry from a
// user-dirs file body.
// Authored by: OpenCode
func parseXDGDocumentsDirectory(configBody string, homeDir string) (string, bool, error) {
	var lines = strings.Split(configBody, "\n")
	for _, line := range lines {
		var trimmed = strings.TrimSpace(line)
		if isIgnoredXDGUserDirsLine(trimmed) {
			continue
		}

		var rawValue, found = xdgDocumentsDirectoryValue(trimmed)
		if !found {
			continue
		}

		var documentsDir, err = parseQuotedXDGDocumentsDirectoryValue(rawValue, homeDir)
		if err != nil {
			return "", false, err
		}

		return documentsDir, true, nil
	}

	return "", false, nil
}

// isIgnoredXDGUserDirsLine reports whether one user-dirs line is blank or a
// comment.
// Authored by: OpenCode
func isIgnoredXDGUserDirsLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

// xdgDocumentsDirectoryValue extracts the raw XDG documents value from one
// trimmed user-dirs line.
// Authored by: OpenCode
func xdgDocumentsDirectoryValue(line string) (string, bool) {
	if !strings.HasPrefix(line, "XDG_DOCUMENTS_DIR=") {
		return "", false
	}

	return strings.TrimSpace(strings.TrimPrefix(line, "XDG_DOCUMENTS_DIR=")), true
}

// parseQuotedXDGDocumentsDirectoryValue validates, unescapes, and resolves one
// quoted XDG documents path value.
// Authored by: OpenCode
func parseQuotedXDGDocumentsDirectoryValue(rawValue string, homeDir string) (string, error) {
	var unescaped, err = unescapeAndValidateQuotedXDGPath(rawValue)
	if err != nil {
		return "", err
	}

	return resolveXDGDocumentsDirectoryPath(unescaped, homeDir)
}

// unescapeAndValidateQuotedXDGPath validates the quoted XDG path wrapper and
// decodes the contained escape sequences.
// Authored by: OpenCode
func unescapeAndValidateQuotedXDGPath(rawValue string) (string, error) {
	if len(rawValue) < 2 || rawValue[0] != '"' || rawValue[len(rawValue)-1] != '"' {
		return "", fmt.Errorf("Linux XDG Documents entry must be a quoted path")
	}

	var unescaped, err = unescapeXDGPath(rawValue[1 : len(rawValue)-1])
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(unescaped) == "" {
		return "", fmt.Errorf("Linux XDG Documents entry must not be empty")
	}

	return unescaped, nil
}

// resolveXDGDocumentsDirectoryPath expands one validated XDG documents path to
// the final filesystem path.
// Authored by: OpenCode
func resolveXDGDocumentsDirectoryPath(pathValue string, homeDir string) (string, error) {
	if strings.HasPrefix(pathValue, "$HOME/") {
		return filepath.Join(homeDir, filepath.FromSlash(strings.TrimPrefix(pathValue, "$HOME/"))), nil
	}
	if pathValue == "$HOME" {
		return homeDir, nil
	}
	if filepath.IsAbs(pathValue) {
		return filepath.Clean(pathValue), nil
	}

	return "", fmt.Errorf("Linux XDG Documents entry %q is not absolute", pathValue)
}

// unescapeXDGPath decodes the minimal quoted escaping used by XDG user-dirs.
// Authored by: OpenCode
func unescapeXDGPath(value string) (string, error) {
	var builder strings.Builder
	builder.Grow(len(value))

	for index := 0; index < len(value); index++ {
		var current = value[index]
		if current != '\\' {
			builder.WriteByte(current)
			continue
		}
		if index+1 >= len(value) {
			return "", fmt.Errorf("Linux XDG Documents entry ends with an incomplete escape")
		}

		index++
		builder.WriteByte(value[index])
	}

	return builder.String(), nil
}

// resolveHomeDirectory resolves the current user's home directory for the
// supported report-output platforms.
// Authored by: OpenCode
func resolveHomeDirectory(goos string) (string, error) {
	if goos == "windows" {
		var userProfile, ok = lookupEnv("USERPROFILE")
		if ok && strings.TrimSpace(userProfile) != "" {
			return filepath.Clean(userProfile), nil
		}
	}

	var homeDir, err = userHomeDirectory()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	if strings.TrimSpace(homeDir) == "" {
		return "", fmt.Errorf("user home directory is empty")
	}

	return filepath.Clean(homeDir), nil
}
