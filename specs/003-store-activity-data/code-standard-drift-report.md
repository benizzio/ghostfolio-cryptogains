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
- This snapshot consolidates delegated per-module reviews for `internal/app/runtime`, `internal/ghostfolio/{client,dto,mapper,validator}`, `internal/snapshot/{envelope,model,store}`, `internal/sync/{model,normalize,validate}`, `internal/tui/{flow,screen}`, and `internal/support/{decimal,redact}`.

## Standards Baseline

- `AGENTS.md`: `Coding standards > LiteratureAndIndustryReferences` requires descriptive and unambiguous names, SRP, decomposition into smaller functions tied to a single responsibility, DRY, and consistency.
- `AGENTS.md`: `Coding standards > CustomCodeDocs` requires AI-touched code documentation to describe the real purpose of modules and functions accurately, include authoring information, and provide detailed usage instructions for public APIs.
- `.specify/memory/constitution.md`: `V. Clean Architecture and Domain Clarity` requires descriptive and unambiguous names, cohesive modules, minimal duplication, SOLID boundaries, and separation of domain rules from IO and infrastructure concerns.
- No additional proprietary agent-instruction files were present in the repository or active feature scope at review time.

## Findings

### DRIFT-001: Runtime Sync Service Mixes Multiple Architectural Responsibilities

**Severity**: High
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: descriptive and unambiguous names, SRP, decomposition into smaller functions tied to a single responsibility
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: modules and functions must remain cohesive, minimize duplication, and separate domain rules from IO and infrastructure concerns

**Evidence**:

- `internal/app/runtime/sync_service.go:54-72`
- `internal/app/runtime/sync_service.go:145-259`
- `internal/app/runtime/sync_service.go:272-275`
- `internal/app/runtime/sync_service.go:324-355`
- `internal/app/runtime/sync_service.go:439-457`

**Description**:

`SyncService` still owns full-history orchestration, diagnostic-report preparation and writing entry points, active readable-snapshot queries, and server-replacement checks behind one runtime boundary. Even with helper collaborators, the service surface combines workflow execution, troubleshooting artifact policy, and protected-snapshot lifecycle concerns, which weakens cohesion and keeps storage and UI-state rules coupled to the same application service.

### DRIFT-002: Deterministic Ordering Rules Are Reimplemented In Multiple Layers

**Severity**: High
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: avoid code duplication, be consistent
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: modules and functions must remain cohesive, minimize duplication, and keep domain concepts explicit

**Evidence**:

- `internal/sync/model/activity_ordering.go:20-31`
- `internal/sync/normalize/activity_history.go:95-106`
- `internal/sync/validate/activity_history.go:94-101`

**Description**:

The codebase now has a shared `NewActivityOrderingKey` constructor, but normalization and validation each still rebuild the deterministic ordering tuple inline instead of using it. That keeps the same domain invariant encoded in multiple places and requires parallel edits whenever the same-asset ordering contract changes.

### DRIFT-003: Stale Validation Terminology Obscures The Storage Workflow

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: choose descriptive and unambiguous names, be consistent
- `AGENTS.md` `Coding standards > CustomCodeDocs`: AI-touched code documentation must describe the purpose of modules and functions accurately
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: names must be descriptive and unambiguous, and consistency is mandatory

**Evidence**:

- `internal/app/runtime/runtime.go:20-25`
- `internal/tui/flow/model.go:354-355`
- `internal/tui/flow/model.go:424-440`
- `internal/tui/flow/model.go:556-560`
- `internal/tui/flow/sync_flow.go:65-67`
- `internal/tui/flow/sync_flow.go:134-146`
- `internal/tui/flow/sync_flow.go:214-223`
- `internal/tui/screen/validation_result_screen.go:15-18`
- `internal/tui/screen/validation_result_screen.go:56-57`
- `internal/tui/screen/validation_result_screen.go:73-86`
- `internal/tui/screen/sync_validation_screen.go:15-18`
- `internal/tui/screen/main_menu_screen.go:57-60`

**Description**:

AI-authored comments, helper names, variables, and menu or status text still describe the slice as validation-only even though the implemented workflow now retrieves and securely stores full activity history. That leaves the active code and UI copy inconsistent with the feature's actual responsibility and makes the storage workflow harder to read and maintain.

### DRIFT-004: Ghostfolio Transport Client Encodes User-Facing Failure Taxonomy

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: SRP, decomposition into smaller functions tied to a single responsibility, consistency
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: modules and functions must remain cohesive and separate domain rules from IO and infrastructure concerns

**Evidence**:

- `internal/ghostfolio/client/client.go:28-60`
- `internal/ghostfolio/client/client.go:300-360`
- `internal/app/runtime/sync_service.go:358-375`

**Description**:

The Ghostfolio HTTP client defines failure categories as the slice's "single user-visible" taxonomy and builds English error messages such as rejected-token, timeout, and connectivity text directly in the transport layer. The runtime then maps those categories almost verbatim into application outcomes. This couples infrastructure code to presentation-facing wording instead of keeping the client focused on boundary semantics and exposing boundary-neutral failures upward.

### DRIFT-005: Activity Mapper Duplicates DTO Normalization Rules Across Mapping Paths

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: avoid code duplication, be consistent
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: modules and functions must remain cohesive, minimize duplication, and keep domain concepts explicit

**Evidence**:

- `internal/ghostfolio/mapper/activity_mapper.go:96-130`
- `internal/ghostfolio/mapper/activity_mapper.go:167-206`
- `internal/ghostfolio/mapper/activity_mapper.go:209-256`

**Description**:

`activity_mapper.go` converts the same `dto.ActivityPageEntry` into both `DiagnosticRecord` and `ActivityRecord` using separate field-normalization paths. The gross-value fallback rule is duplicated, and the two paths already diverge on source-scope handling because the diagnostic path emits account scope whenever `entry.Account != nil`, while the stored-record path drops scope unless `Account.ID` is non-empty. This duplicates a domain translation rule instead of keeping one local source of truth.

### DRIFT-006: Snapshot Exported APIs Miss Required AI-Authored Usage Documentation

**Severity**: Low
**Diverges from**:

- `AGENTS.md` `Coding standards > CustomCodeDocs`: AI-generated public methods and functions must include detailed purpose documentation and example usage

**Evidence**:

- `internal/snapshot/envelope/codec.go:65-85`
- `internal/snapshot/envelope/codec.go:142-183`
- `internal/snapshot/store/store.go:168-245`
- `internal/snapshot/store/discovery.go:12-33`
- `internal/snapshot/store/compatibility.go:22-39`

**Description**:

Several exported snapshot APIs are AI-authored but only carry brief summary comments and no detailed usage guidance or examples. This is below the repository's explicit documentation baseline for public AI-generated code and makes the snapshot boundary less self-describing than the local policy requires.

## Notes

- Existing `DRIFT-001` through `DRIFT-003` identifiers were preserved because the underlying findings remain substantively the same in the current implementation.
- `checklists/code-standard-drift-remediation.md` exists at review time, so the correction-tracking link resolves in this snapshot.
