// Package main prints a comma-separated package list for Go package patterns.
// Authored by: Copilot
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// main resolves the provided package patterns into import paths and writes them
// as a comma-separated list.
//
// Authored by: Copilot
func main() {
	var goBinaryFlag = flag.String("go", "go", "Go binary used to run go list")

	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "coverpkg: at least one package pattern is required")
		os.Exit(1)
	}

	var listArgs = append([]string{"list", "-f", "{{.ImportPath}}"}, flag.Args()...)
	var listCommand = exec.Command(*goBinaryFlag, listArgs...)
	var outputBuffer bytes.Buffer

	listCommand.Stdout = &outputBuffer
	listCommand.Stderr = os.Stderr

	if err := listCommand.Run(); err != nil {
		os.Exit(1)
	}

	var packages []string
	for _, line := range strings.Split(outputBuffer.String(), "\n") {
		var importPath = strings.TrimSpace(line)
		if importPath == "" {
			continue
		}

		packages = append(packages, importPath)
	}

	fmt.Print(strings.Join(packages, ","))
}
