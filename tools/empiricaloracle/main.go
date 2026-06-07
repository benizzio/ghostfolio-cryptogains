package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

var stderrWriter io.Writer = os.Stderr

// main parses command-line input and reports startup errors to stderr.
// Authored by: OpenCode
func main() {
	var err = run(os.Args[1:], os.Stdout)
	if err == nil {
		return
	}

	if _, writeErr := fmt.Fprintln(stderrWriter, err); writeErr != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(1)
}

// run parses the Phase 1 placeholder flags and reserves the command boundary
// for later oracle-generation work.
// Authored by: OpenCode
func run(args []string, stdout io.Writer) error {
	var flagSet = flag.NewFlagSet("empiricaloracle", flag.ContinueOnError)
	flagSet.SetOutput(stdout)

	var datasetPath = flagSet.String("dataset", "testdata/empirical/financial-dataset.yaml", "Synthetic empirical dataset path")
	var outputRoot = flagSet.String("output-root", "testdata/empirical", "Empirical artifact root path")
	var regenerate = flagSet.Bool("regenerate", false, "Regenerate oracle artifacts instead of reusing existing fixtures")

	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(stdout, "Usage: empiricaloracle [flags]")
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Phase 1 command skeleton. Oracle generation is added in later phases.")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("empiricaloracle: parse flags: %w", err)
	}

	_ = *datasetPath
	_ = *outputRoot
	_ = *regenerate

	return errors.New("empiricaloracle: fixture generation is not implemented in Phase 1")
}
