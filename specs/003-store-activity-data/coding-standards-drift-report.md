# Coding Standards Drift Report: Store Activity Data

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-05-18
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.coding-standards-drift-analysis.remediation-plan`.

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.
- Reviewed implementation focused on the active feature slice under `internal/app/runtime`, `internal/snapshot`, `internal/sync`, `internal/tui`, and the directly supporting `tests/contract`, `tests/integration`, and `tests/unit` files.

## Standards Baseline

- `AGENTS.md:63-70` requires descriptive and unambiguous names, SOLID and SRP analysis, DRY, and consistency.
- `AGENTS.md:84-106` requires agent-touched code to carry minimal language-standard documentation plus authoring information at component/module and function levels.
- `AGENTS.md:110-117` states a local Go style preference for `var` over `:=`, except for the documented reuse case.
- `.specify/memory/constitution.md:128-139` requires descriptive naming, cohesive modules and functions, minimized duplication, and documented consistency.
- No additional proprietary instruction files were present in repository or feature scope at review time beyond `AGENTS.md`.

## Findings

### DRIFT-001: Feature Test Files Apply The Required Documentation Pattern Inconsistently

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:84-106`

**Evidence**:

- `tests/unit/decimal_test.go:1-39`
- `tests/unit/snapshot_store_test.go:1-49`
- `tests/unit/snapshot_envelope_test.go:1-32`
- `tests/unit/year_derivation_test.go:1-37`
- `tests/contract/ghostfolio_sync_storage_contract_test.go:1-64`
- `tests/contract/helpers_test.go:1-20`
- `tests/integration/sync_storage_flow_test.go:1-146`

**Description**:

The active feature slice applies the repository's required code-documentation and author-attribution pattern unevenly. Many neighboring feature files include package-level comments, function-level comments, and `Authored by: OpenCode` markers, but the files above do not. That leaves the feature's test surface inconsistent with the explicit documentation baseline and makes the slice harder to audit and maintain under the repository's agent-authorship rules.

### DRIFT-002: Directly Supporting Files Still Use Validation-Era Names For A Storage Slice

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:63-70`
- `.specify/memory/constitution.md:128-139`

**Evidence**:

- `tests/integration/sync_validation_flow_test.go:1-5`
- `tests/integration/sync_validation_flow_test.go:33-59`
- `tests/integration/sync_validation_flow_test.go:400-510`
- `tests/contract/ghostfolio_sync_validation_contract_test.go:10-27`
- `internal/tui/screen/sync_validation_screen.go:13-49`
- `internal/tui/screen/validation_result_screen.go:13-45`

**Description**:

The feature now implements a sync-and-store slice, and the screen types themselves describe sync entry and sync result behavior, but several directly supporting file names, comments, helper types, and test names still use `validation` terminology from the earlier slice. This leaves the active implementation surface less descriptive and less consistent than the local baseline requires, especially where storage-oriented behavior is still exercised through `syncValidation*` helpers and `*_validation_*` file names.

### DRIFT-003: Production Go Code Repeats Short Declarations Against The Local `var` Preference

**Severity**: Low
**Diverges from**:

- `AGENTS.md:110-117`

**Evidence**:

- `internal/sync/model/activity_amount_resolution.go:54-62`
- `internal/sync/model/activity_amount_resolution.go:97-107`
- `internal/sync/model/activity_amount_resolution.go:130-156`
- `internal/sync/model/activity_amount_resolution.go:171-180`

**Description**:

`internal/sync/model/activity_amount_resolution.go` repeatedly uses `:=` for first declarations such as `grossValue, grossValueCurrency, err := ...` and `unitPrice, _, err := ...` even though the repository's local Go guidance prefers explicit `var` outside the documented reuse exception. This is low-risk, but it is still a concrete consistency drift in active production code.

## Notes

- No prior `coding-standards-drift-report.md` existed in this feature directory, so this report starts a new ID sequence at `DRIFT-001`.
- Task-state review found no unchecked tasks in `specs/003-store-activity-data/tasks.md`; completed items that still include `Reopened` text were treated as historical labels inside checked tasks rather than open work.
