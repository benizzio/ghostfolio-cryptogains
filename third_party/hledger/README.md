# Vendored hledger Boundary

This directory contains the repository-vendored hledger boundary used only by
the empirical oracle tool and empirical tests.

Runtime application code under `cmd/` and `internal/` must not link, import, or
execute hledger.

## Selected Version

- Selected version: `1.99.2`
- Release status: prerelease
- Why this prerelease is selected: the feature research initially assumed
  `1.52.1`, but direct validation proved the required
  `FIFO/LIFO/HIFO/AVERAGE lot + gain behavior` exists only in prerelease
  `1.99.2`.

## Repository Path Contract

- Complete corresponding source must remain under `third_party/hledger/source/`.
- The command wrapper resolves only `third_party/hledger/bin/<goos>-<goarch>/hledger`.
- The wrapper passes explicit `-f` journal file arguments and `-n` so developer
  `LEDGER_FILE` and hledger config do not leak in.
- Missing, non-executable, or unsupported vendored artifacts must fail with an
  actionable setup error.
- Binary-only vendoring is invalid.

## Platform Support

Platform support is explicit. A platform is supported only when a matching
artifact exists under `bin/<goos>-<goarch>/hledger`.

| Repo platform path | Status | Committed artifact path | Checksum notes |
| --- | --- | --- | --- |
| `linux-amd64` | supported | `third_party/hledger/bin/linux-amd64/hledger` | committed extracted executable checksum `sha256:801f1abfae1bf3b741567a7eea9ee6a4227544b6dcdb02b71b36ffcd26cec409`; validated upstream release asset digest `sha256:4d94d701b1a9e82aa2ea1b9997ddadbd94fecba21b4bdce9f4c85e8c1a3d2b9e` |
| `darwin-arm64` | not repo-supported yet | none committed | validated upstream release asset digest `sha256:55bcb1d8341902f751d8b79b27d6f01f3c51cd7453996d7bd1eba17a2a567292` |
| `darwin-amd64` | not repo-supported yet | none committed | validated upstream release asset digest `sha256:bb5090978b84e9957fe2d7052703ec000f5a6161908ce0c3b813386450674bfe` |
| `windows-amd64` | not repo-supported yet | none committed | validated upstream release asset digest `sha256:5c5881924727e2635a9f69f88191bce8ef924c23009688f305c3df13e1198ee2` |

## Source Material

- Complete corresponding source archive:
  `third_party/hledger/source/hledger-source-1.99.2.tar.gz`
- Committed source archive checksum:
  `sha256:1ea46d762f973fed0550ae57aee38d8036a754bf1e3064a27b307e2ecbeaccdf`
- Licensing: `GPL-3.0-or-later`
- Upstream metadata, tag commit, and release provenance are recorded in
  `third_party/hledger/SOURCE.md`.

## Regeneration Notes

1. Verify the upstream tag, tag commit, and release digests recorded in
   `SOURCE.md`.
2. Replace `third_party/hledger/source/hledger-source-1.99.2.tar.gz` with the
   verified complete corresponding source for the selected tag.
3. Replace or add `third_party/hledger/bin/<goos>-<goarch>/hledger` only for
   platforms whose artifacts are intentionally supported in this repository.
4. Update `SOURCE.md`, this file, and the vendoring contract tests with the new
   committed checksums.
5. Run `go test ./tools/empiricaloracle ./tests/empirical -run 'TestHledger|TestVendored' -count=1`.
