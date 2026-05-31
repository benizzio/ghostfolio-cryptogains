// Package output defines local filesystem save and post-save open helpers for
// generated yearly gains-and-losses report files.
// Authored by: OpenCode
package output

import "fmt"

// OpenCommand describes one platform-specific process invocation that asks the
// operating system to open a saved report file with the default associated
// application.
// Authored by: OpenCode
type OpenCommand struct {
	Name string
	Args []string
}

// ResolveOpenCommand returns the platform-specific process invocation required
// to ask the operating system to open the provided report path.
//
// Example:
//
//	command, err := output.ResolveOpenCommand("/tmp/report.md")
//	if err != nil {
//		panic(err)
//	}
//	_ = command.Name
//
// Use this helper when code needs to inspect the adapter decision without
// starting a subprocess.
// Authored by: OpenCode
func ResolveOpenCommand(path string) (OpenCommand, error) {
	return ResolveOpenCommandForOS(currentGOOS(), path)
}

// ResolveOpenCommandForOS returns the platform-specific process invocation for
// the provided operating system identifier.
//
// Example:
//
//	command, err := output.ResolveOpenCommandForOS("linux", "/tmp/report.md")
//	if err != nil {
//		panic(err)
//	}
//	_ = command.Args
//
// Use this helper from tests when the adapter behavior must be verified for
// platforms other than the host operating system.
// Authored by: OpenCode
func ResolveOpenCommandForOS(goos string, path string) (OpenCommand, error) {
	if path == "" {
		return OpenCommand{}, fmt.Errorf("report path is required")
	}

	switch goos {
	case "linux":
		return OpenCommand{Name: "xdg-open", Args: []string{path}}, nil
	case "darwin":
		return OpenCommand{Name: "open", Args: []string{path}}, nil
	case "windows":
		return OpenCommand{Name: "cmd", Args: []string{"/c", "start", "", path}}, nil
	default:
		return OpenCommand{}, fmt.Errorf("automatic report opening is unsupported on %q", goos)
	}
}

// OpenPath asks the operating system to open one previously saved report file
// with the default associated application.
//
// Example:
//
//	err := output.OpenPath("/tmp/report.md")
//	if err != nil {
//		panic(err)
//	}
//
// This helper performs exactly one open request and leaves any successfully
// saved file untouched when the subprocess fails.
// Authored by: OpenCode
func OpenPath(path string) error {
	var command, err = ResolveOpenCommand(path)
	if err != nil {
		return err
	}

	err = runOpenCommand(command)
	if err != nil {
		return fmt.Errorf("open report path %q with %s: %w", path, command.Name, err)
	}

	return nil
}
