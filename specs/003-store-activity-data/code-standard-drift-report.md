# Code Standard Drift Report: Store Activity Data

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-05-16
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: [checklists/code-standard-drift-remediation.md](./checklists/code-standard-drift-remediation.md)

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.
- Reviewed feature artifacts: `spec.md`, `plan.md`, `tasks.md`, `research.md`, `data-model.md`, and `quickstart.md`.
- Reviewed implementation scope: `internal/app/runtime`, `internal/ghostfolio/{client,mapper}`, `internal/snapshot/{envelope,store}`, `internal/sync/{model,normalize,validate}`, `internal/tui/{flow,screen}`, and directly related support code under `internal/support`.
- Cognitive-complexity verification in this snapshot was limited to non-test implementation files because `AGENTS.md` now explicitly exempts test code from that rule.

## Standards Baseline

- `AGENTS.md`: `Coding standards > LiteratureAndIndustryReferences` requires descriptive and unambiguous names, SRP, decomposition into smaller functions tied to a single responsibility, DRY, and consistency.
- `AGENTS.md`: `Coding standards > LiteratureAndIndustryReferences` requires cognitive complexity under 15 for Go functions, measured with `github.com/uudashr/gocognit`, and explicitly exempts test code.
- `AGENTS.md`: `Coding standards > CustomCodeDocs` requires AI-generated public methods and functions to include detailed purpose documentation and example usage, and AI-generated structs or interfaces to include detailed purpose documentation.
- `.specify/memory/constitution.md`: `V. Clean Architecture and Domain Clarity` requires descriptive names, cohesive modules and functions, minimal duplication, and separation of domain rules from IO and infrastructure concerns.
- No additional proprietary agent-instruction files were present in the repository or active feature scope at review time.

## Findings

### DRIFT-002: Deterministic Ordering Rules Remain Reimplemented Across Normalization And Validation

**Severity**: High
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: avoid code duplication, be consistent
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: modules and functions must remain cohesive, minimize duplication, and keep domain concepts explicit

**Evidence**:

- `internal/sync/model/activity_ordering.go:20-32`
- `internal/sync/normalize/activity_history.go:95-107`
- `internal/sync/validate/activity_history.go:94-101`

**Description**:

The feature already has a shared `syncmodel.NewActivityOrderingKey` constructor, but both normalization and validation still rebuild the same ordering tuple inline. That keeps the same same-asset ordering rule in three locations. This slice already reopened ordering work for `BUG-002`, so retaining that duplication creates a concrete risk of divergence the next time the ordering rule changes.

### DRIFT-003: Validation-Only Terminology Still Obscures The Sync-And-Storage Workflow

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: choose descriptive and unambiguous names, be consistent
- `AGENTS.md` `Coding standards > CustomCodeDocs`: AI-touched code documentation must describe the real purpose accurately
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: names must be descriptive and unambiguous, and consistency is mandatory

**Evidence**:

- `internal/app/runtime/runtime.go:20-24`
- `internal/tui/flow/model.go:354-359`
- `internal/tui/flow/model.go:434-440`
- `internal/tui/flow/model.go:556-560`
- `internal/tui/flow/sync_flow.go:65-67`
- `internal/tui/flow/sync_flow.go:134-146`
- `internal/tui/screen/validation_result_screen.go:56-57`
- `internal/tui/screen/validation_result_screen.go:73-86`
- `internal/tui/screen/sync_validation_screen.go:13-18`
- `internal/tui/screen/main_menu_screen.go:57-60`

**Description**:

Comments, helper names, parameter names, and user-facing text still describe this slice as validation-only even though the implemented workflow now authenticates, retrieves, normalizes, validates, and stores protected snapshots. Examples include `cancelActiveValidation`, `releaseSyncInputToValidationMenu`, `validationSummaryText`, and main-menu copy that still says `Sync Data` is used to validate Ghostfolio communication. That leaves the active code and UI language inconsistent with the implemented responsibility.

### DRIFT-006: AI-Authored Exported APIs Still Miss The Required Detailed Usage Documentation

**Severity**: Low
**Diverges from**:

- `AGENTS.md` `Coding standards > CustomCodeDocs`: AI-generated public methods and functions must include detailed purpose documentation and example usage, and AI-generated structs or interfaces must include detailed purpose documentation

**Evidence**:

- `internal/app/runtime/sync_service.go:25-72`
- `internal/app/runtime/sync_types.go:90-140`
- `internal/tui/screen/server_replacement_screen.go:12-23`
- `internal/tui/screen/server_replacement_screen.go:25-45`

**Description**:

These exported AI-authored types and functions carry only summary-level comments, and `ServerReplacementScreenView` has no example usage block at all. The local documentation baseline is stricter than normal Go conventions for public, cross-package AI-authored code. The current comments do not meet that repository-specific requirement.

### DRIFT-007: Core Non-Test Functions Exceed The Repository Cognitive-Complexity Limit

**Severity**: High
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: cognitive complexity in functions should be kept under 15, and higher values should trigger SRP and decomposition analysis
- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: decomposition into smaller functions tied to a single responsibility
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: modules and functions must remain cohesive and respect SOLID boundaries where those boundaries improve clarity and change safety

**Evidence**:

- `internal/sync/validate/activity_history.go:74-162`
- `internal/app/runtime/sync_service.go:145-270`
- `internal/ghostfolio/client/client.go:179-218`
- `internal/sync/normalize/activity_history.go:231-265`

**Description**:

Measured with `gocognit` on non-test files only, several implementation functions exceed the explicit limit from `AGENTS.md`: `defaultValidator.Validate` scores 41, `(*syncService).Run` scores 18, `(*Client).FetchActivitiesHistory` scores 17, and `deriveTimelineScopeReliability` scores 17. Each function now carries multiple branches and rule clusters that should have been split after the threshold was exceeded. The validator is the clearest example: it combines cache-shape checks, timestamp parsing, ordering enforcement, price rules, quantity replay, and failure shaping in one function.

## Notes

- Existing `DRIFT-002`, `DRIFT-003`, and `DRIFT-006` identifiers were preserved because the underlying findings remain substantively the same in the current implementation.
- `DRIFT-007` is new in this snapshot and records the non-test cognitive-complexity violations introduced by the updated `AGENTS.md` baseline.
- Prior drift IDs not listed in this snapshot were not reissued as substantively unchanged findings in the reviewed scope.
- `checklists/code-standard-drift-remediation.md` exists at review time, so the correction-tracking link resolves in this snapshot.
