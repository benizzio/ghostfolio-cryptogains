package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestVerifySourceArchiveChecksumReturnsActionableErrorWhenArchiveMissing
// verifies explicit regeneration reports the verified-source cache requirement
// clearly when the pinned archive is absent.
// Authored by: OpenCode
func TestVerifySourceArchiveChecksumReturnsActionableErrorWhenArchiveMissing(t *testing.T) {
	t.Parallel()

	var runtime = rotkiSourceRuntime{repositoryRoot: t.TempDir()}
	var err = runtime.verifySourceArchiveChecksum()
	if err == nil {
		t.Fatal("expected missing pinned archive to fail checksum verification")
	}
	if !strings.Contains(err.Error(), rotkiSourceArchiveRepositoryPath) {
		t.Fatalf("expected checksum error to mention archive path, got %v", err)
	}
	if !strings.Contains(err.Error(), "rerun explicit regeneration") {
		t.Fatalf("expected checksum error to explain explicit regeneration remediation, got %v", err)
	}
}

// TestVerifyVerificationManifestRejectsVendoredSourcePath verifies committed or
// vendored rotki source locations are rejected before regeneration can treat
// them as verified-source evidence.
// Authored by: OpenCode
func TestVerifyVerificationManifestRejectsVendoredSourcePath(t *testing.T) {
	t.Parallel()

	var runtime = rotkiSourceRuntime{repositoryRoot: t.TempDir()}
	var manifest = rotkiSourceVerificationManifest{
		SourceURL:         defaultRotkiSourceArchiveURL,
		SourceChecksum:    defaultRotkiSourceChecksum,
		ReleaseTag:        defaultRotkiReleaseTag,
		ResolvedCommit:    defaultRotkiVersionOrCommit,
		SignedTagObject:   defaultRotkiSignedTagObject,
		SourceArchivePath: rotkiSourceArchiveRepositoryPath,
		SourceRootPath:    "third_party/rotki/source",
		RotkiAdapterPath:  rotkiAdapterRepositoryPath,
	}

	var err = runtime.verifyVerificationManifest(manifest)
	if err == nil {
		t.Fatal("expected vendored source path to be rejected")
	}
	if !strings.Contains(err.Error(), "source_root_path mismatch") {
		t.Fatalf("expected vendored source path mismatch error, got %v", err)
	}
}

// TestWriteVerificationManifestRecordsVerifiedUntrackedPaths verifies explicit
// regeneration reuses only the repository-local untracked cache paths for the
// pinned rotki source archive, extracted source tree, and verification manifest.
// Authored by: OpenCode
func TestWriteVerificationManifestRecordsVerifiedUntrackedPaths(t *testing.T) {
	t.Parallel()

	var runtime = rotkiSourceRuntime{repositoryRoot: t.TempDir()}
	if err := runtime.writeVerificationManifest(); err != nil {
		t.Fatalf("write verification manifest: %v", err)
	}

	var manifest, loaded, err = runtime.loadVerificationManifest()
	if err != nil {
		t.Fatalf("load verification manifest: %v", err)
	}
	if !loaded {
		t.Fatal("expected verification manifest to be written")
	}
	if manifest.SourceArchivePath != rotkiSourceArchiveRepositoryPath {
		t.Fatalf("unexpected source archive path: got %s want %s", manifest.SourceArchivePath, rotkiSourceArchiveRepositoryPath)
	}
	if manifest.SourceRootPath != rotkiSourceRootRepositoryPath {
		t.Fatalf("unexpected source root path: got %s want %s", manifest.SourceRootPath, rotkiSourceRootRepositoryPath)
	}
	if manifest.RotkiAdapterPath != rotkiAdapterRepositoryPath {
		t.Fatalf("unexpected rotki adapter path: got %s want %s", manifest.RotkiAdapterPath, rotkiAdapterRepositoryPath)
	}

	var manifestPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceVerificationRepositoryPath))
	if !strings.Contains(filepath.ToSlash(manifestPath), "/.cache/empiricaloracle/rotki-source/") {
		t.Fatalf("expected verification manifest below untracked rotki cache path, got %s", filepath.ToSlash(manifestPath))
	}
}
