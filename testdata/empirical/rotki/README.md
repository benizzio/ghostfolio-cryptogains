# Historical rotki Bootstrap Metadata

This directory stores historical repository-controlled rotki boundary metadata
from BUG-001. BUG-002 supersedes this bootstrap shortcut: committed raw rotki
outputs, hand-authored rotki datasets, developer-global rotki installations, and
vendored rotki source are not acceptable oracle evidence for regenerated golden
fixtures.

## Current State

- `bootstrap-boundary.json` records the historical BUG-001 bootstrap boundary.
- The manifest is not authoritative regeneration evidence after BUG-002.
- Method directories store stable normalization inputs for `fifo`, `lifo`,
  `hifo`, and `average_cost` aggregate cases plus `scope_local_hybrid`
  composite-boundary inputs.
- No file in this directory claims rotki was executed in this checkout.
- Raw rotki outputs must not be committed here as the source of regenerated
  oracle data.

## Boundary Rules

- All referenced inputs are synthetic and trace back to
  `testdata/empirical/financial-dataset.yaml`.
- Zero-priced holding reductions remain excluded from rotki-backed fixture scope.
- `scope_local_hybrid` is not represented here as a native rotki case. Its
  committed inputs remain composite-boundary inputs, not pure rotki captures.
- Default empirical test runs must continue to rely on committed golden fixtures.
- Explicit fixture regeneration must download or reuse verified pinned rotki
  source from `.cache/empiricaloracle/rotki-source/`, then execute it through the
  project-owned local adapter boundary.
- Normal fixture-backed empirical test runs must not download rotki, require the
  untracked source cache, or invoke oracle generation while committed golden
  fixtures are present.

## Intended Follow-Up

- Later adapter work must replace this bootstrap metadata with committed
  normalized golden fixtures and provenance metadata generated from verified
  untracked pinned rotki source execution.
- Any future regeneration must retain the pinned rotki identity documented under
  `third_party/rotki/README.md` or explicitly record an intentional pin change.

Authored by: OpenCode
