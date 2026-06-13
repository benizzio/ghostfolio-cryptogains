package main

import (
	"io"
	"testing"
)

// TestRunDoesNotRequireRotkiGenerationWhenGoldenFixturesExist verifies the
// default fixture-backed path does not resolve the regeneration runtime when no
// golden fixtures are missing.
//
// Authored by: OpenCode
func TestRunDoesNotRequireRotkiGenerationWhenGoldenFixturesExist(t *testing.T) {
	t.Parallel()

	var priorResolveRotkiSourceRuntime = resolveRotkiSourceRuntime
	t.Cleanup(func() {
		resolveRotkiSourceRuntime = priorResolveRotkiSourceRuntime
	})

	resolveRotkiSourceRuntime = func() (rotkiSourceRuntime, error) {
		t.Fatal("expected default fixture-backed run path to skip verified rotki source resolution")
		return rotkiSourceRuntime{}, nil
	}

	if err := run(nil, io.Discard); err != nil {
		t.Fatalf("run default empiricaloracle path: %v", err)
	}
}
