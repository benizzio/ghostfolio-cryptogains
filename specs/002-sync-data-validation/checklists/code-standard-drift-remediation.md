# Checklist: Code Standard Drift Remediation

**Purpose**: Track correction of the coding-standard drift recorded in [`../code-standard-drift-report.md`](../code-standard-drift-report.md).
**Created**: 2026-05-10
**Feature**: [spec.md](../spec.md)

## High Priority

- [ ] DRIFT-001 Move setup save and validation-start orchestration out of `internal/tui/flow` so the TUI layer no longer builds `AppSetupConfig`, writes setup, or starts infrastructure-backed validation commands directly.
- [ ] DRIFT-002 Refactor `internal/app/runtime/sync_service.go` so the application contract no longer exposes `ghostfolioclient.FailureCategory` and no longer builds final user-facing strings.

## Medium Priority

- [ ] DRIFT-003 Replace `StartupState.InvalidSetupMessage` with structured bootstrap outcome data and move final user-facing wording to the TUI layer.
- [ ] DRIFT-004 Extract shared request and response handling from `internal/ghostfolio/client/client.go` to remove duplicated HTTP-boundary logic.
- [ ] DRIFT-005 Split `internal/config/store/json_store.go` `Save` behavior into smaller helpers with single responsibilities.
- [ ] DRIFT-006 Expand public API documentation for `Store`, `SyncService`, `NewSyncService`, `Validate`, `ParseOptions`, `ShortHelp`, and `FullHelp` so it matches the repo `CustomCodeDocs` rules.

## Low Priority

- [ ] DRIFT-007 Add missing AI documentation and author-attribution blocks to the flagged types and functions in `cmd/ghostfolio-cryptogains/main.go`, `internal/tui/flow/model.go`, `internal/app/runtime/sync_service.go`, and `internal/config/store/json_store.go`.
- [ ] DRIFT-008 Replace flagged single-name `:=` declarations with `var` declarations where the repo style requires it.

## Closure Criteria

- [ ] Re-run the coding-standards review after remediation and confirm that every drift item in `../code-standard-drift-report.md` is either resolved or intentionally reclassified.
- [ ] Confirm the remediation changes preserve project-owned automated coverage expectations.
- [ ] Confirm any updated public API comments remain accurate after the code changes.
