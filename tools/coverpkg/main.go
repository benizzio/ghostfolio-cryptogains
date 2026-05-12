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
	var goBinary = flag.String("go", "go", "Go binary used to run go list")

	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "coverpkg: at least one package pattern is required")
		os.Exit(1)
	}

	var args = append([]string{"list", "-f", "{{.ImportPath}}"}, flag.Args()...)
	var command = exec.Command(*goBinary, args...)
	var output bytes.Buffer

	command.Stdout = &output
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		os.Exit(1)
	}

	var packages []string
	for _, line := range strings.Split(output.String(), "\n") {
		var packagePath = strings.TrimSpace(line)
		if packagePath == "" {
			continue
		}

		packages = append(packages, packagePath)
	}

	fmt.Print(strings.Join(packages, ","))
}
