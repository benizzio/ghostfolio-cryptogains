# Checklist: Code Standard Drift Remediation

**Purpose**: Track correction of the coding-standard drift recorded in [`../code-standard-drift-report.md`](../code-standard-drift-report.md).
**Created**: 2026-05-16
**Feature**: [spec.md](../spec.md)

## High Priority

- [x] DRIFT-001 Split `internal/app/runtime/sync_service.go` so runtime orchestration delegates snapshot lifecycle, protected payload construction, diagnostic-report writing, and active-snapshot state handling to dedicated collaborators with narrower responsibilities.
- [x] DRIFT-002 Consolidate same-asset deterministic ordering and ambiguity detection into one shared domain helper used by both normalization and validation so the rule is defined in one place.
- [x] DRIFT-007 Split `defaultValidator.Validate`, `(*syncService).Run`, `(*Client).FetchActivitiesHistory`, and `deriveTimelineScopeReliability` into smaller cohesive helpers so each non-test function returns below the repository cognitive-complexity threshold of 15.

## Medium Priority

- [x] DRIFT-003 Rename or update validation-only and probe-oriented types, aliases, and AI-authored comments so runtime, Ghostfolio client, DTO, and TUI workflow names describe the full-history protected-storage flow accurately.
- [x] DRIFT-004 Move user-facing failure taxonomy and English outcome text out of `internal/ghostfolio/client/client.go` so the transport boundary returns boundary-focused failures and the application layer owns user-visible sync outcomes.
- [x] DRIFT-005 Extract shared `dto.ActivityPageEntry` normalization helpers in `internal/ghostfolio/mapper/activity_mapper.go` so `ActivityRecord` and `DiagnosticRecord` mapping reuse the same gross-value and source-scope rules.

## Low Priority

- [x] DRIFT-006 Expand the exported snapshot envelope and store API comments in `internal/snapshot/envelope/codec.go` and `internal/snapshot/store/*.go` to include the detailed AI-authored usage guidance and examples required by the repository baseline.

## Closure Criteria

- [ ] Re-run the coding-standards review after remediation and confirm that every drift item in `../code-standard-drift-report.md` is either resolved or intentionally reclassified.
- [x] Confirm the remediation changes preserve project-owned automated coverage expectations.
- [x] Confirm any updated public API comments or author-attribution notes remain accurate after the code changes.
