package main

import (
	"context"
	"io"
	"testing"
)

// TestRunDoesNotRequireVendoredHledgerWhenGoldenFixturesExist verifies the default fixture-backed path does not resolve hledger when no golden fixtures are missing.
//
// Authored by: OpenCode
func TestRunDoesNotRequireVendoredHledgerWhenGoldenFixturesExist(t *testing.T) {
	t.Parallel()

	var priorResolveVendoredHledgerCommand = resolveVendoredHledgerCommand
	var priorCaptureVendoredHledgerVersion = captureVendoredHledgerVersion
	var priorResolveRotkiSourceRuntime = resolveRotkiSourceRuntime
	t.Cleanup(func() {
		resolveVendoredHledgerCommand = priorResolveVendoredHledgerCommand
		captureVendoredHledgerVersion = priorCaptureVendoredHledgerVersion
		resolveRotkiSourceRuntime = priorResolveRotkiSourceRuntime
	})

	resolveVendoredHledgerCommand = func() (vendoredHledgerCommand, error) {
		t.Fatal("expected default fixture-backed run path to skip vendored hledger resolution")
		return vendoredHledgerCommand{}, nil
	}
	captureVendoredHledgerVersion = func(ctx context.Context, command vendoredHledgerCommand) (string, error) {
		t.Fatal("expected default fixture-backed run path to skip vendored hledger version capture")
		return "", nil
	}
	resolveRotkiSourceRuntime = func() (rotkiSourceRuntime, error) {
		t.Fatal("expected default fixture-backed run path to skip verified rotki source resolution")
		return rotkiSourceRuntime{}, nil
	}

	if err := run(nil, io.Discard); err != nil {
		t.Fatalf("run default empiricaloracle path: %v", err)
	}
}
