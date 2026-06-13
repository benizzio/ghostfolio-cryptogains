package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	rotkiSourceCacheRootRepositoryPath    = ".cache/empiricaloracle/rotki-source"
	rotkiSourceArchiveRepositoryPath      = ".cache/empiricaloracle/rotki-source/rotki-v1.43.1.tar.gz"
	rotkiSourceRootRepositoryPath         = ".cache/empiricaloracle/rotki-source/rotki-1.43.1"
	rotkiSourceVerificationRepositoryPath = ".cache/empiricaloracle/rotki-source/verified-source.json"
	rotkiAdapterRepositoryPath            = "tools/empiricaloracle/rotki_adapter.py"
	defaultRotkiSourceRepositoryURL       = "https://github.com/rotki/rotki"
	defaultRotkiSourceArchiveURL          = "https://github.com/rotki/rotki/archive/refs/tags/v1.43.1.tar.gz"
	defaultRotkiReleaseTag                = "v1.43.1"
	defaultRotkiVersionOrCommit           = "a2e00be49a0ea36e7563a5d235cfa6a7c91edbfb"
	defaultRotkiSignedTagObject           = "b35a3c934eedf23b1387ff564b6386fb2ce3f201"
	defaultRotkiSourceChecksum            = "sha256:8434b653104f8d5b0638e98d88a5ef256fac7720cc459eb33b729e2848900e3b"
	defaultRotkiSourceChecksumDigest      = "8434b653104f8d5b0638e98d88a5ef256fac7720cc459eb33b729e2848900e3b"
	defaultRotkiPureOracleName            = "rotki"
	defaultRotkiHybridCompositeOracleName = "scope_local_hybrid_composite"
	defaultRotkiCompositeRuleVersion      = "scope_local_hybrid_composite_v1"
)

var resolveRotkiSourceRuntime = newRotkiSourceRuntime

// rotkiSourceRuntime resolves, verifies, and executes the pinned rotki source
// boundary used only during empirical fixture regeneration.
// Authored by: OpenCode
type rotkiSourceRuntime struct {
	repositoryRoot    string
	httpClient        *http.Client
	lookPath          func(string) (string, error)
	runCombinedOutput runCombinedOutputFunc
}

// verifiedRotkiSource stores the verified local rotki source boundary metadata
// needed by adapter execution and fixture provenance.
// Authored by: OpenCode
type verifiedRotkiSource struct {
	SourceRootRelativePath    string
	SourceArchiveRelativePath string
	SourceURL                 string
	SourceChecksum            string
	VersionOrCommit           string
}

// rotkiSourceVerificationManifest stores the cached verification state for the
// untracked pinned rotki source checkout.
// Authored by: OpenCode
type rotkiSourceVerificationManifest struct {
	SourceURL          string `json:"source_url"`
	SourceChecksum     string `json:"source_checksum"`
	ReleaseTag         string `json:"release_tag"`
	ResolvedCommit     string `json:"resolved_commit"`
	SignedTagObject    string `json:"signed_tag_object"`
	SourceArchivePath  string `json:"source_archive_path"`
	SourceRootPath     string `json:"source_root_path"`
	RotkiAdapterPath   string `json:"rotki_adapter_path"`
	VerifiedAt         string `json:"verified_at,omitempty"`
	VerificationMethod string `json:"verification_method,omitempty"`
}

// newRotkiSourceRuntime builds the regeneration-only rotki source runtime for
// the current repository checkout.
// Authored by: OpenCode
func newRotkiSourceRuntime() (rotkiSourceRuntime, error) {
	var repositoryRoot, err = resolveEmpiricalRepositoryRoot()
	if err != nil {
		return rotkiSourceRuntime{}, err
	}

	return rotkiSourceRuntime{
		repositoryRoot:    repositoryRoot,
		httpClient:        &http.Client{Timeout: 2 * time.Minute},
		lookPath:          exec.LookPath,
		runCombinedOutput: runCombinedOutput,
	}, nil
}

// ensureVerifiedSource downloads or reuses the pinned rotki source archive,
// verifies archive checksum plus remote tag identity, extracts the source tree,
// and records the local verification manifest.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) ensureVerifiedSource(ctx context.Context) (verifiedRotkiSource, error) {
	var source = verifiedRotkiSource{
		SourceRootRelativePath:    rotkiSourceRootRepositoryPath,
		SourceArchiveRelativePath: rotkiSourceArchiveRepositoryPath,
		SourceURL:                 defaultRotkiSourceArchiveURL,
		SourceChecksum:            defaultRotkiSourceChecksum,
		VersionOrCommit:           defaultRotkiVersionOrCommit,
	}

	var manifest, manifestLoaded, err = runtime.loadVerificationManifest()
	if err != nil {
		return verifiedRotkiSource{}, err
	}
	if manifestLoaded {
		if err = runtime.verifyVerificationManifest(manifest); err != nil {
			return verifiedRotkiSource{}, err
		}
		return source, nil
	}

	if err = runtime.ensureSourceArchive(ctx); err != nil {
		return verifiedRotkiSource{}, err
	}
	if err = runtime.verifyRemoteTagIdentity(ctx); err != nil {
		return verifiedRotkiSource{}, err
	}
	if err = runtime.extractSourceArchive(); err != nil {
		return verifiedRotkiSource{}, err
	}
	if err = runtime.writeVerificationManifest(); err != nil {
		return verifiedRotkiSource{}, err
	}

	return source, nil
}

// captureOracleOutput executes the project-owned Python adapter against the
// verified rotki source tree and parses the adapter JSON result.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) captureOracleOutput(
	ctx context.Context,
	inputRelativePath string,
	rotkiMethod reportMethod,
) (rotkiOracleCapture, []byte, verifiedRotkiSource, error) {
	var source, err = runtime.ensureVerifiedSource(ctx)
	if err != nil {
		return rotkiOracleCapture{}, nil, verifiedRotkiSource{}, err
	}

	var pythonExecutable, pythonErr = runtime.pythonExecutablePath()
	if pythonErr != nil {
		return rotkiOracleCapture{}, nil, verifiedRotkiSource{}, pythonErr
	}

	var runCommand = runtime.runCombinedOutput
	if runCommand == nil {
		runCommand = runCombinedOutput
	}

	var inputPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(inputRelativePath))
	var sourceRootPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(source.SourceRootRelativePath))
	var adapterPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiAdapterRepositoryPath))
	var rawOutput []byte
	rawOutput, err = runCommand(
		ctx,
		os.Environ(),
		pythonExecutable,
		adapterPath,
		"--source-root",
		sourceRootPath,
		"--input",
		inputPath,
		"--rotki-method",
		string(rotkiMethod),
	)
	if err != nil {
		var detail = strings.TrimSpace(string(rawOutput))
		if detail != "" {
			return rotkiOracleCapture{}, nil, verifiedRotkiSource{}, fmt.Errorf(
				"run rotki adapter %s with %s: %w: %s",
				rotkiAdapterRepositoryPath,
				inputRelativePath,
				err,
				detail,
			)
		}

		return rotkiOracleCapture{}, nil, verifiedRotkiSource{}, fmt.Errorf(
			"run rotki adapter %s with %s: %w",
			rotkiAdapterRepositoryPath,
			inputRelativePath,
			err,
		)
	}

	var capture rotkiOracleCapture
	if err = json.Unmarshal(rawOutput, &capture); err != nil {
		return rotkiOracleCapture{}, nil, verifiedRotkiSource{}, fmt.Errorf("decode rotki adapter JSON for %s: %w", inputRelativePath, err)
	}

	return capture, rawOutput, source, nil
}

// pythonExecutablePath resolves the local Python executable used for the
// project-owned rotki adapter.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) pythonExecutablePath() (string, error) {
	var lookup = runtime.lookPath
	if lookup == nil {
		lookup = exec.LookPath
	}

	for _, candidate := range []string{"python3", "python"} {
		var executablePath, err = lookup(candidate)
		if err == nil {
			return executablePath, nil
		}
	}

	return "", fmt.Errorf(
		"empirical fixture regeneration requires python3 or python to execute %s against verified rotki source",
		rotkiAdapterRepositoryPath,
	)
}

// loadVerificationManifest reads the cached rotki source verification manifest
// when it already exists.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) loadVerificationManifest() (rotkiSourceVerificationManifest, bool, error) {
	var manifestPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceVerificationRepositoryPath))
	var rawContent, err = os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return rotkiSourceVerificationManifest{}, false, nil
		}

		return rotkiSourceVerificationManifest{}, false, fmt.Errorf("read rotki verification manifest %s: %w", rotkiSourceVerificationRepositoryPath, err)
	}

	var manifest rotkiSourceVerificationManifest
	if err = json.Unmarshal(rawContent, &manifest); err != nil {
		return rotkiSourceVerificationManifest{}, false, fmt.Errorf("decode rotki verification manifest %s: %w", rotkiSourceVerificationRepositoryPath, err)
	}

	return manifest, true, nil
}

// verifyVerificationManifest revalidates one cached verification manifest and
// the local untracked source artifacts it references.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) verifyVerificationManifest(manifest rotkiSourceVerificationManifest) error {
	if strings.TrimSpace(manifest.SourceURL) != defaultRotkiSourceArchiveURL {
		return fmt.Errorf("rotki verification manifest source_url mismatch: got %s want %s", strings.TrimSpace(manifest.SourceURL), defaultRotkiSourceArchiveURL)
	}
	if strings.TrimSpace(manifest.SourceChecksum) != defaultRotkiSourceChecksum {
		return fmt.Errorf("rotki verification manifest source_checksum mismatch: got %s want %s", strings.TrimSpace(manifest.SourceChecksum), defaultRotkiSourceChecksum)
	}
	if strings.TrimSpace(manifest.ReleaseTag) != defaultRotkiReleaseTag {
		return fmt.Errorf("rotki verification manifest release_tag mismatch: got %s want %s", strings.TrimSpace(manifest.ReleaseTag), defaultRotkiReleaseTag)
	}
	if strings.TrimSpace(manifest.ResolvedCommit) != defaultRotkiVersionOrCommit {
		return fmt.Errorf("rotki verification manifest resolved_commit mismatch: got %s want %s", strings.TrimSpace(manifest.ResolvedCommit), defaultRotkiVersionOrCommit)
	}
	if strings.TrimSpace(manifest.SignedTagObject) != defaultRotkiSignedTagObject {
		return fmt.Errorf("rotki verification manifest signed_tag_object mismatch: got %s want %s", strings.TrimSpace(manifest.SignedTagObject), defaultRotkiSignedTagObject)
	}
	if strings.TrimSpace(manifest.SourceArchivePath) != rotkiSourceArchiveRepositoryPath {
		return fmt.Errorf("rotki verification manifest source_archive_path mismatch: got %s want %s", strings.TrimSpace(manifest.SourceArchivePath), rotkiSourceArchiveRepositoryPath)
	}
	if strings.TrimSpace(manifest.SourceRootPath) != rotkiSourceRootRepositoryPath {
		return fmt.Errorf("rotki verification manifest source_root_path mismatch: got %s want %s", strings.TrimSpace(manifest.SourceRootPath), rotkiSourceRootRepositoryPath)
	}
	if strings.TrimSpace(manifest.RotkiAdapterPath) != rotkiAdapterRepositoryPath {
		return fmt.Errorf("rotki verification manifest rotki_adapter_path mismatch: got %s want %s", strings.TrimSpace(manifest.RotkiAdapterPath), rotkiAdapterRepositoryPath)
	}

	if err := runtime.verifySourceArchiveChecksum(); err != nil {
		return err
	}

	var sourceRootPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceRootRepositoryPath))
	var requiredSourcePath = filepath.Join(sourceRootPath, "rotkehlchen", "accounting", "cost_basis", "base.py")
	var sourceInfo, err = os.Stat(requiredSourcePath)
	if err != nil {
		return fmt.Errorf("verified rotki source checkout is missing required file %s: %w", filepath.ToSlash(requiredSourcePath), err)
	}
	if sourceInfo.IsDir() {
		return fmt.Errorf("verified rotki source checkout path %s points to a directory, not rotki source", filepath.ToSlash(requiredSourcePath))
	}

	return nil
}

// ensureSourceArchive reuses the cached pinned archive when present or downloads
// it into the untracked source cache when absent.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) ensureSourceArchive(ctx context.Context) error {
	if err := runtime.verifySourceArchiveChecksum(); err == nil {
		return nil
	}

	var archivePath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceArchiveRepositoryPath))
	var parentDirectory = filepath.Dir(archivePath)
	if err := os.MkdirAll(parentDirectory, 0o755); err != nil {
		return fmt.Errorf("create rotki source cache directory %s: %w", filepath.ToSlash(parentDirectory), err)
	}

	var request, err = http.NewRequestWithContext(ctx, http.MethodGet, defaultRotkiSourceArchiveURL, nil)
	if err != nil {
		return fmt.Errorf("build rotki source archive request: %w", err)
	}

	var client = runtime.httpClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}

	var response, requestErr = client.Do(request)
	if requestErr != nil {
		return fmt.Errorf("download pinned rotki source archive %s: %w", defaultRotkiSourceArchiveURL, requestErr)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download pinned rotki source archive %s: unexpected HTTP status %s", defaultRotkiSourceArchiveURL, response.Status)
	}

	var temporaryPath = archivePath + ".tmp"
	var outputFile, createErr = os.Create(temporaryPath)
	if createErr != nil {
		return fmt.Errorf("create temporary rotki source archive %s: %w", filepath.ToSlash(temporaryPath), createErr)
	}

	if _, err = io.Copy(outputFile, response.Body); err != nil {
		_ = outputFile.Close()
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("write temporary rotki source archive %s: %w", filepath.ToSlash(temporaryPath), err)
	}
	if err = outputFile.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("close temporary rotki source archive %s: %w", filepath.ToSlash(temporaryPath), err)
	}
	if err = os.Rename(temporaryPath, archivePath); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("move rotki source archive into cache %s: %w", rotkiSourceArchiveRepositoryPath, err)
	}

	if err = runtime.verifySourceArchiveChecksum(); err != nil {
		return err
	}

	return nil
}

// verifySourceArchiveChecksum rehashes the cached pinned rotki source archive.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) verifySourceArchiveChecksum() error {
	var archivePath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceArchiveRepositoryPath))
	var rawArchive, err = os.ReadFile(archivePath)
	if err != nil {
		return fmt.Errorf(
			"read pinned rotki source archive %s: %w; rerun explicit regeneration to download or refresh the verified archive",
			rotkiSourceArchiveRepositoryPath,
			err,
		)
	}

	var actualChecksum = stablePrefixedSHA256Hash(rawArchive)
	if actualChecksum != defaultRotkiSourceChecksum {
		return fmt.Errorf(
			"pinned rotki source archive checksum mismatch for %s: got %s want %s",
			rotkiSourceArchiveRepositoryPath,
			actualChecksum,
			defaultRotkiSourceChecksum,
		)
	}

	return nil
}

// verifyRemoteTagIdentity checks the configured remote tag object and peeled
// commit identity before extraction is accepted as verified.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) verifyRemoteTagIdentity(ctx context.Context) error {
	var runCommand = runtime.runCombinedOutput
	if runCommand == nil {
		runCommand = runCombinedOutput
	}

	var rawOutput, err = runCommand(
		ctx,
		os.Environ(),
		"git",
		"ls-remote",
		"--tags",
		defaultRotkiSourceRepositoryURL,
		"refs/tags/"+defaultRotkiReleaseTag,
		"refs/tags/"+defaultRotkiReleaseTag+"^{}",
	)
	if err != nil {
		return fmt.Errorf(
			"verify pinned rotki tag %s against %s with git ls-remote: %w",
			defaultRotkiReleaseTag,
			defaultRotkiSourceRepositoryURL,
			err,
		)
	}

	var signedTagObject string
	var peeledCommit string
	var lines = strings.Split(strings.TrimSpace(string(rawOutput)), "\n")
	var lineIndex int
	for lineIndex = range lines {
		var fields = strings.Fields(lines[lineIndex])
		if len(fields) != 2 {
			continue
		}

		switch fields[1] {
		case "refs/tags/" + defaultRotkiReleaseTag:
			signedTagObject = strings.TrimSpace(fields[0])
		case "refs/tags/" + defaultRotkiReleaseTag + "^{}":
			peeledCommit = strings.TrimSpace(fields[0])
		}
	}

	if signedTagObject != defaultRotkiSignedTagObject {
		return fmt.Errorf(
			"verified rotki tag object mismatch for %s: got %s want %s",
			defaultRotkiReleaseTag,
			signedTagObject,
			defaultRotkiSignedTagObject,
		)
	}
	if peeledCommit != defaultRotkiVersionOrCommit {
		return fmt.Errorf(
			"verified rotki peeled commit mismatch for %s: got %s want %s",
			defaultRotkiReleaseTag,
			peeledCommit,
			defaultRotkiVersionOrCommit,
		)
	}

	return nil
}

// extractSourceArchive expands the verified pinned source archive into the
// untracked rotki source cache when the extracted tree is absent.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) extractSourceArchive() error {
	var sourceRootPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceRootRepositoryPath))
	if _, err := os.Stat(sourceRootPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat verified rotki source root %s: %w", rotkiSourceRootRepositoryPath, err)
	}

	var cacheRootPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceCacheRootRepositoryPath))
	if err := os.MkdirAll(cacheRootPath, 0o755); err != nil {
		return fmt.Errorf("create rotki source cache root %s: %w", rotkiSourceCacheRootRepositoryPath, err)
	}

	var temporaryRootPath, err = os.MkdirTemp(cacheRootPath, "extract-")
	if err != nil {
		return fmt.Errorf("create temporary rotki extraction directory in %s: %w", rotkiSourceCacheRootRepositoryPath, err)
	}
	defer func() {
		_ = os.RemoveAll(temporaryRootPath)
	}()

	var archivePath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceArchiveRepositoryPath))
	var archiveFile, openErr = os.Open(archivePath)
	if openErr != nil {
		return fmt.Errorf("open pinned rotki source archive %s: %w", rotkiSourceArchiveRepositoryPath, openErr)
	}
	defer func() {
		_ = archiveFile.Close()
	}()

	var gzipReader, gzipErr = gzip.NewReader(archiveFile)
	if gzipErr != nil {
		return fmt.Errorf("open pinned rotki source archive gzip stream %s: %w", rotkiSourceArchiveRepositoryPath, gzipErr)
	}
	defer func() {
		_ = gzipReader.Close()
	}()

	var tarReader = tar.NewReader(gzipReader)
	for {
		var header, nextErr = tarReader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return fmt.Errorf("read pinned rotki source archive entry: %w", nextErr)
		}

		var cleanedPath = path.Clean(strings.TrimSpace(header.Name))
		if cleanedPath == "." || cleanedPath == "" {
			continue
		}
		if strings.HasPrefix(cleanedPath, "../") || cleanedPath == ".." {
			return fmt.Errorf("pinned rotki source archive contains invalid path %s", header.Name)
		}

		var targetPath = filepath.Join(temporaryRootPath, filepath.FromSlash(cleanedPath))
		var relativePath, relErr = filepath.Rel(temporaryRootPath, targetPath)
		if relErr != nil {
			return fmt.Errorf("resolve temporary rotki extraction path %s: %w", cleanedPath, relErr)
		}
		if relativePath == ".." || strings.HasPrefix(filepath.ToSlash(relativePath), "../") {
			return fmt.Errorf("pinned rotki source archive path %s escapes the extraction directory", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create extracted rotki directory %s: %w", filepath.ToSlash(targetPath), err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("create extracted rotki parent directory %s: %w", filepath.ToSlash(filepath.Dir(targetPath)), err)
			}
			var outputFile, createErr = os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if createErr != nil {
				return fmt.Errorf("create extracted rotki file %s: %w", filepath.ToSlash(targetPath), createErr)
			}
			if _, err := io.Copy(outputFile, tarReader); err != nil {
				_ = outputFile.Close()
				return fmt.Errorf("write extracted rotki file %s: %w", filepath.ToSlash(targetPath), err)
			}
			if err := outputFile.Close(); err != nil {
				return fmt.Errorf("close extracted rotki file %s: %w", filepath.ToSlash(targetPath), err)
			}
		}
	}

	var extractedRootPath = filepath.Join(temporaryRootPath, filepath.FromSlash("rotki-1.43.1"))
	if _, err := os.Stat(extractedRootPath); err != nil {
		return fmt.Errorf("pinned rotki source archive did not extract expected source root rotki-1.43.1: %w", err)
	}
	if err := os.Rename(extractedRootPath, sourceRootPath); err != nil {
		return fmt.Errorf("move extracted rotki source tree into %s: %w", rotkiSourceRootRepositoryPath, err)
	}

	return nil
}

// writeVerificationManifest records the completed local verification state so
// later explicit regenerations can reuse the verified source cache deterministically.
// Authored by: OpenCode
func (runtime rotkiSourceRuntime) writeVerificationManifest() error {
	var manifest = rotkiSourceVerificationManifest{
		SourceURL:          defaultRotkiSourceArchiveURL,
		SourceChecksum:     defaultRotkiSourceChecksum,
		ReleaseTag:         defaultRotkiReleaseTag,
		ResolvedCommit:     defaultRotkiVersionOrCommit,
		SignedTagObject:    defaultRotkiSignedTagObject,
		SourceArchivePath:  rotkiSourceArchiveRepositoryPath,
		SourceRootPath:     rotkiSourceRootRepositoryPath,
		RotkiAdapterPath:   rotkiAdapterRepositoryPath,
		VerifiedAt:         time.Now().UTC().Format(time.RFC3339),
		VerificationMethod: "archive_sha256+git_ls_remote_tag",
	}

	var rawManifest, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal rotki verification manifest: %w", err)
	}

	var manifestPath = filepath.Join(runtime.repositoryRoot, filepath.FromSlash(rotkiSourceVerificationRepositoryPath))
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("create rotki verification manifest directory %s: %w", filepath.ToSlash(filepath.Dir(manifestPath)), err)
	}
	if err := os.WriteFile(manifestPath, append(rawManifest, '\n'), 0o644); err != nil {
		return fmt.Errorf("write rotki verification manifest %s: %w", rotkiSourceVerificationRepositoryPath, err)
	}

	return nil
}

// reportMethod stores the Python-adapter cost-basis method selector.
// Authored by: OpenCode
type reportMethod string
