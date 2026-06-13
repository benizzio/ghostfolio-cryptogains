# Deprecated rotki Bootstrap Path

This directory is intentionally README-only. BUG-002 supersedes the earlier
BUG-001 bootstrap shortcut: committed raw rotki outputs, bootstrap manifests,
hand-authored rotki datasets, developer-global rotki installations, and vendored
rotki source are not acceptable oracle evidence for regenerated golden fixtures.

## Current State

- No committed JSON payload under this path is allowed.
- No file in this directory claims rotki was executed in this checkout.
- Raw rotki outputs, bootstrap manifests, and hand-authored adapter inputs must
  not be recommitted here as the source of regenerated oracle data.

## Boundary Rules

- Zero-priced holding reductions remain excluded from rotki-backed fixture scope.
- Default empirical test runs must continue to rely on committed golden fixtures.
- Explicit fixture regeneration must download or reuse verified pinned rotki
  source from `.cache/empiricaloracle/rotki-source/`, then execute it through the
  project-owned local adapter boundary.
- Normal fixture-backed empirical test runs must not download rotki, require the
  untracked source cache, or invoke oracle generation while committed golden
  fixtures are present.
- Remove `.cache/empiricaloracle/rotki-source/` if you need to force a fresh
  verified-source acquisition during explicit regeneration.

## Intended Follow-Up

- Any future regeneration must retain the pinned rotki identity documented under
  `third_party/rotki/README.md` or explicitly record an intentional pin change.

Authored by: OpenCode
