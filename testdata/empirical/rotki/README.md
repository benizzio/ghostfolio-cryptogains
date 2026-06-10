# Repository-Controlled rotki Bootstrap Inputs

This directory stores the repository-controlled rotki boundary inputs for BUG-001.
The committed data here is synthetic-only and is intended to bootstrap later rotki
adapter and fixture-regeneration work without relying on a developer-local rotki
installation.

## Current State

- `bootstrap-boundary.json` is the authoritative manifest for the current rotki
  bootstrap boundary.
- The manifest records the repository-controlled bootstrap policy for the pure
  rotki-backed cases.
- Method directories store stable normalization inputs for `fifo`, `lifo`,
  `hifo`, and `average_cost` aggregate cases plus `scope_local_hybrid`
  composite-boundary inputs.
- No file in this directory claims rotki was executed in this checkout.
- Exact raw rotki outputs are not committed here yet.

## Boundary Rules

- All referenced inputs are synthetic and trace back to
  `testdata/empirical/financial-dataset.yaml`.
- Zero-priced holding reductions remain excluded from rotki-backed fixture scope.
- `scope_local_hybrid` is not represented here as a native rotki case. Its
  committed inputs remain composite-boundary inputs, not pure rotki captures.
- Default empirical test runs must continue to rely on committed golden fixtures.
- Future raw rotki captures must be committed with provenance metadata instead of
  being sourced from a developer-local installation.

## Intended Follow-Up

- Later adapter work may replace `raw_output_status: not_committed` entries with
  committed captures at the documented `intended_capture_path` values.
- Any future capture must retain the pinned rotki identity documented under
  `third_party/rotki/README.md` or explicitly record an intentional pin change.

Authored by: OpenCode
