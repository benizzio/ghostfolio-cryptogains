# Vendored hledger Notes

This directory reserves the repository-managed hledger boundary used only by
empirical oracle generation and empirical tests.

## Compliance Notes

- `source/` must contain the complete corresponding source for the vendored hledger version.
- `bin/<goos>-<goarch>/hledger` is the only supported executable artifact layout.
- Every vendored source payload and every supported executable artifact must have a recorded checksum in repository documentation before the tool is used.
- Platform support is explicit. A platform is supported only when a matching executable artifact exists under `bin/<goos>-<goarch>/hledger`.
- Binary-only vendoring is invalid. Executables must not be committed without the matching complete source.
- Runtime application code must not link, import, or execute hledger.
- Future vendoring work must also add the GPL-3.0-or-later license text, upstream source URL, selected version, regeneration instructions, and checksum records required by the empirical oracle contract.

## Current Phase 1 State

- `bin/` and `source/` are skeleton directories only. No hledger source or executable artifacts are vendored yet.
